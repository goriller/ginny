#!/usr/bin/env python3
"""
Global-to-Project KB Migration Tool

将全局知识库中的项目特有条目迁移到对应的项目级知识库（$PROJECT_ROOT/.opencode/knowledge/）。

用法:
  # 1. 预览：扫描全局 KB，按关键词识别属于某项目的条目
  python migrate_to_project.py --project /path/to/koatty-monorepo \\
      --keywords "koatty,monorepo,pnpm,turborepo" --dry-run

  # 2. 执行：将匹配条目复制到项目 KB，并从全局 KB 删除
  python migrate_to_project.py --project /path/to/koatty-monorepo \\
      --keywords "koatty,monorepo,pnpm,turborepo"

  # 3. 列出全局 KB 所有条目（用于人工审查）
  python migrate_to_project.py --list

  # 4. 批量模式：从 YAML/JSON 配置文件读取多项目迁移规则
  python migrate_to_project.py --batch rules.json --dry-run

迁移后需运行 mode --init 或手动同步 .opencode/scripts/ 到目标项目。
"""

import argparse
import json
import re
import shutil
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional, Set, Tuple

# Ensure scripts/ is on sys.path so core.* and knowledge.* imports work
_SCRIPTS_ROOT = Path(__file__).parent.parent
if str(_SCRIPTS_ROOT) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS_ROOT))
_KNOWLEDGE_DIR = Path(__file__).parent
if str(_KNOWLEDGE_DIR) not in sys.path:
    sys.path.insert(0, str(_KNOWLEDGE_DIR))

try:
    from core.path_resolver import get_knowledge_base_dir
    from core.config import CATEGORY_DIRS
    from core.file_utils import atomic_write_json
except ImportError:
    CATEGORY_DIRS = {
        'experience': 'experiences', 'tech-stack': 'tech-stacks',
        'scenario': 'scenarios', 'problem': 'problems',
        'testing': 'testing', 'pattern': 'patterns', 'skill': 'skills',
    }
    def get_knowledge_base_dir() -> Path:
        return Path.home() / '.config' / 'opencode' / 'knowledge'
    def atomic_write_json(filepath, data):
        filepath = Path(filepath)
        filepath.parent.mkdir(parents=True, exist_ok=True)
        with open(filepath, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)


# ─────────────────────────────────────────────────────────────────────────────
# Index rebuild (identical to migrate_degraded.py to keep logic DRY-ish)
# ─────────────────────────────────────────────────────────────────────────────

def rebuild_indexes(kb_root: Path) -> None:
    """Rebuild index.json and per-category indexes from entry files."""
    global_index: Dict[str, Any] = {
        'trigger_index': {},
        'category_index': {},
        'stats': {'total_entries': 0, 'by_category': {}},
        'recent_entries': [],
        'last_updated': datetime.now().isoformat(),
    }

    for category, cat_dir_name in CATEGORY_DIRS.items():
        cat_dir = kb_root / cat_dir_name
        if not cat_dir.exists():
            continue

        cat_entries: List[Dict[str, Any]] = []
        for entry_file in sorted(cat_dir.glob('*.json')):
            if entry_file.name == 'index.json':
                continue
            try:
                entry = json.loads(entry_file.read_text(encoding='utf-8'))
            except (json.JSONDecodeError, IOError, UnicodeDecodeError):
                continue

            eid = entry.get('id', entry_file.stem)
            cat_entries.append({
                'id': eid,
                'name': entry.get('name', 'Unknown'),
                'created_at': entry.get('created_at', ''),
                'updated_at': entry.get('updated_at', ''),
            })

            for trigger in entry.get('triggers', []):
                tl = trigger.lower()
                global_index['trigger_index'].setdefault(tl, [])
                if eid not in global_index['trigger_index'][tl]:
                    global_index['trigger_index'][tl].append(eid)

            global_index['category_index'].setdefault(cat_dir_name, [])
            if eid not in global_index['category_index'][cat_dir_name]:
                global_index['category_index'][cat_dir_name].append(eid)

        cat_index_path = cat_dir / 'index.json'
        atomic_write_json(cat_index_path, {
            'entries': cat_entries,
            'last_updated': datetime.now().isoformat(),
        })

    global_index['stats']['total_entries'] = sum(
        len(v) for v in global_index['category_index'].values()
    )
    global_index['stats']['by_category'] = {
        cat: len(entries) for cat, entries in global_index['category_index'].items()
    }
    all_ids: List[str] = []
    for v in global_index['category_index'].values():
        all_ids.extend(v)
    global_index['recent_entries'] = all_ids[-20:]

    atomic_write_json(kb_root / 'index.json', global_index)


# ─────────────────────────────────────────────────────────────────────────────
# Entry matching
# ─────────────────────────────────────────────────────────────────────────────

def _entry_text(entry: Dict[str, Any]) -> str:
    """Concatenate all searchable text from an entry for keyword matching."""
    parts = [
        entry.get('name', ''),
        ' '.join(entry.get('triggers', [])),
        ' '.join(entry.get('tags', [])),
    ]
    content = entry.get('content', {})
    if isinstance(content, dict):
        for field in ('description', 'solution', 'problem_name', 'summary',
                      'scenario_name', 'tech_name', 'typical_approach'):
            val = content.get(field, '')
            if isinstance(val, str):
                parts.append(val)
    return ' '.join(parts)


def matches_keywords(entry: Dict[str, Any], keywords: List[str]) -> bool:
    """Return True if any keyword is found (case-insensitive) in entry text."""
    text = _entry_text(entry).lower()
    return any(kw.lower() in text for kw in keywords)


def scan_global_kb(
    kb_root: Path,
    keywords: Optional[List[str]] = None,
) -> List[Dict[str, Any]]:
    """
    Scan the global KB and return entries matching keywords.
    If keywords is None, returns ALL entries.
    Each returned dict includes _file_path and _cat_dir_name.
    """
    results: List[Dict[str, Any]] = []

    for cat_dir_name in CATEGORY_DIRS.values():
        cat_dir = kb_root / cat_dir_name
        if not cat_dir.exists():
            continue
        for entry_file in sorted(cat_dir.glob('*.json')):
            if entry_file.name == 'index.json':
                continue
            try:
                entry = json.loads(entry_file.read_text(encoding='utf-8'))
            except (json.JSONDecodeError, IOError, UnicodeDecodeError):
                continue

            if keywords is None or matches_keywords(entry, keywords):
                entry['_file_path'] = str(entry_file)
                entry['_cat_dir_name'] = cat_dir_name
                results.append(entry)

    return results


# ─────────────────────────────────────────────────────────────────────────────
# Migration: copy entry to project-local KB, delete from global
# ─────────────────────────────────────────────────────────────────────────────

def migrate_entry_to_project(
    entry: Dict[str, Any],
    project_kb: Path,
    delete_from_global: bool = True,
    dry_run: bool = False,
) -> Dict[str, str]:
    """
    Copy one entry to the project-local KB directory.
    Stamps project_path field. Deletes original if delete_from_global=True.

    Returns a report dict.
    """
    file_path = Path(entry.pop('_file_path', ''))
    cat_dir_name = entry.pop('_cat_dir_name', 'experiences')

    entry_id = entry.get('id', file_path.stem)
    report = {
        'id': entry_id,
        'name': entry.get('name', '')[:60],
        'action': 'dry-run' if dry_run else 'migrated',
        'from': str(file_path),
        'to': str(project_kb / cat_dir_name / file_path.name),
    }

    if dry_run:
        return report

    # Stamp project_path
    entry['project_path'] = str(project_kb.parent.parent)  # $PROJECT_ROOT
    entry['updated_at'] = datetime.now().isoformat()

    dest_dir = project_kb / cat_dir_name
    dest_dir.mkdir(parents=True, exist_ok=True)
    dest_file = dest_dir / file_path.name

    atomic_write_json(dest_file, entry)

    if delete_from_global and file_path.exists():
        file_path.unlink()
        report['action'] = 'moved'
    else:
        report['action'] = 'copied'

    return report


# ─────────────────────────────────────────────────────────────────────────────
# Tag-only mode: add project_path to global entries without moving
# ─────────────────────────────────────────────────────────────────────────────

def tag_entries(
    entries: List[Dict[str, Any]],
    project_path: str,
    dry_run: bool = False,
) -> int:
    """Stamp project_path field onto matching entries in-place."""
    count = 0
    for entry in entries:
        file_path_str = entry.get('_file_path', '')
        if not file_path_str:
            continue
        file_path = Path(file_path_str)

        entry_copy = {k: v for k, v in entry.items()
                      if not k.startswith('_')}
        entry_copy['project_path'] = project_path
        entry_copy['updated_at'] = datetime.now().isoformat()

        if not dry_run:
            atomic_write_json(file_path, entry_copy)
        count += 1

    return count


# ─────────────────────────────────────────────────────────────────────────────
# Batch migration from JSON rules file
# ─────────────────────────────────────────────────────────────────────────────

def run_batch(
    rules_file: Path,
    kb_root: Path,
    dry_run: bool = False,
    delete_after_copy: bool = True,
) -> None:
    """
    Run batch migration from a JSON rules file.

    rules.json format:
    [
      {
        "project": "/path/to/project-root",
        "keywords": ["koatty", "monorepo", "pnpm"]
      },
      ...
    ]
    """
    try:
        rules = json.loads(rules_file.read_text(encoding='utf-8'))
    except (json.JSONDecodeError, IOError) as e:
        print(f"Error reading rules file: {e}", file=sys.stderr)
        sys.exit(1)

    total_migrated = 0
    for rule in rules:
        project_dir = Path(rule.get('project', ''))
        keywords = rule.get('keywords', [])

        if not project_dir or not keywords:
            print(f"Skipping invalid rule: {rule}", file=sys.stderr)
            continue

        project_kb = project_dir / '.opencode' / 'knowledge'
        entries = scan_global_kb(kb_root, keywords)

        if not entries:
            print(f"[{project_dir.name}] No matching entries for keywords: {keywords}")
            continue

        print(f"\n[{project_dir.name}] {len(entries)} entries matched → {project_kb}")
        reports = []
        for entry in entries:
            report = migrate_entry_to_project(
                entry, project_kb, delete_from_global=delete_after_copy, dry_run=dry_run
            )
            reports.append(report)
            print(f"  {report['action']}: {report['name']}")

        total_migrated += len(reports)

        if not dry_run:
            rebuild_indexes(kb_root)
            rebuild_indexes(project_kb)

    print(f"\nTotal: {total_migrated} entries {'(dry run)' if dry_run else 'processed'}")


# ─────────────────────────────────────────────────────────────────────────────
# CLI
# ─────────────────────────────────────────────────────────────────────────────

def main() -> None:
    parser = argparse.ArgumentParser(
        description='Migrate global KB entries to project-local KB',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument('--project', '-p',
        help='Target project root directory (creates .opencode/knowledge/ inside it)')
    parser.add_argument('--keywords', '-k',
        help='Comma-separated keywords to match entries (case-insensitive)')
    parser.add_argument('--dry-run', action='store_true',
        help='Preview: show what would be migrated without writing anything')
    parser.add_argument('--list', action='store_true',
        help='List ALL entries in the global KB (omit --keywords to see all)')
    parser.add_argument('--tag-only', action='store_true',
        help='Only stamp project_path field; do NOT move entries to project KB')
    parser.add_argument('--keep-in-global', action='store_true',
        help='Copy (not move) to project KB; keep original in global KB')
    parser.add_argument('--batch',
        help='Path to JSON batch-rules file for multi-project migration')
    parser.add_argument('--source-kb',
        help='Override global KB path (default: auto-detect)')
    args = parser.parse_args()

    kb_root = Path(args.source_kb) if args.source_kb else get_knowledge_base_dir()
    if not kb_root.exists():
        print(f"Error: Global KB not found at {kb_root}", file=sys.stderr)
        sys.exit(1)

    # ── Batch mode ──────────────────────────────────────────────────────────
    if args.batch:
        run_batch(
            rules_file=Path(args.batch),
            kb_root=kb_root,
            dry_run=args.dry_run,
            delete_after_copy=not args.keep_in_global,
        )
        return

    # ── List mode ───────────────────────────────────────────────────────────
    keywords: Optional[List[str]] = None
    if args.keywords:
        keywords = [k.strip() for k in args.keywords.split(',') if k.strip()]

    if args.list:
        entries = scan_global_kb(kb_root, keywords)
        print(f"{'Cat':<12} {'Score':>5} {'Name'}")
        print('─' * 80)
        for e in sorted(entries, key=lambda x: x.get('category', '')):
            cat = e.get('category', '')[:10]
            eff = e.get('effectiveness', 0.5)
            name = e.get('name', '')[:60]
            proj = e.get('project_path', '')
            tag = f' [{Path(proj).name}]' if proj else ''
            print(f"{cat:<12} {eff:>5.2f} {name}{tag}")
        print(f"\nTotal: {len(entries)} entries")
        return

    # ── Single-project migration ─────────────────────────────────────────────
    if not args.project:
        parser.error('--project is required (or use --list / --batch)')

    if not keywords:
        parser.error('--keywords is required for single-project migration')

    project_dir = Path(args.project)
    project_kb = project_dir / '.opencode' / 'knowledge'

    entries = scan_global_kb(kb_root, keywords)
    if not entries:
        print(f"No entries matched keywords: {keywords}")
        return

    print(f"Found {len(entries)} entries matching {keywords}")
    print(f"Target: {project_kb}\n")

    # ── Tag-only mode ────────────────────────────────────────────────────────
    if args.tag_only:
        count = tag_entries(entries, str(project_dir), dry_run=args.dry_run)
        mode = 'Would tag' if args.dry_run else 'Tagged'
        print(f"{mode} {count} entries with project_path={project_dir}")
        return

    # ── Copy / Move mode ─────────────────────────────────────────────────────
    if not args.dry_run:
        # Backup global KB before destructive operation
        backup_dir = kb_root.parent / f'knowledge-backup-{datetime.now().strftime("%Y%m%d%H%M%S")}'
        if not args.keep_in_global:
            shutil.copytree(kb_root, backup_dir)
            print(f"Backup: {backup_dir}\n")

        project_kb.mkdir(parents=True, exist_ok=True)

    reports = []
    for entry in entries:
        report = migrate_entry_to_project(
            entry,
            project_kb=project_kb,
            delete_from_global=not args.keep_in_global,
            dry_run=args.dry_run,
        )
        reports.append(report)

    # Print report table
    print(f"{'Action':<10} {'Category':<12} {'Name'}")
    print('─' * 80)
    for r in reports:
        name = r.get('name', '')[:55]
        print(f"{r['action']:<10} {'':<12} {name}")

    if not args.dry_run:
        print('\nRebuilding indexes...')
        rebuild_indexes(kb_root)
        rebuild_indexes(project_kb)
        print(f"Global KB index rebuilt: {kb_root / 'index.json'}")
        print(f"Project KB index rebuilt: {project_kb / 'index.json'}")
        print(f"\nBackup at: {backup_dir if not args.keep_in_global else 'N/A (keep-in-global mode)'}")
        print(f"Rollback: copy {backup_dir} back to {kb_root}")
    else:
        print(f"\nDry run — no files modified. Remove --dry-run to apply.")


if __name__ == '__main__':
    main()
