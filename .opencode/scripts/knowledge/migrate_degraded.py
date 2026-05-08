#!/usr/bin/env python3
"""
Batch migration for degraded knowledge entries.

Fixes entries created before the summarizer extraction regex fix:
- name had redundant "经验: " prefix with the full original text
- content.description/solution were unseparated flat text
- content.context was always "用户记录的经验"

This script re-parses the original text through the corrected extraction
patterns to produce proper structured entries.

Usage:
    python migrate_degraded.py --dry-run    # Preview changes
    python migrate_degraded.py              # Apply changes
    python migrate_degraded.py --rollback   # Restore from backup
"""

import argparse
import json
import os
import re
import shutil
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

_scripts_root = Path(__file__).parent.parent
if str(_scripts_root) not in sys.path:
    sys.path.insert(0, str(_scripts_root))

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

DIR_TO_CATEGORY = {v: k for k, v in CATEGORY_DIRS.items()}

# ─── Re-parsing patterns (mirrors the fixed summarizer.py) ──────────────

REPARSE_PATTERNS: List[Tuple[str, str, List[str]]] = [
    # (regex, entry_type, group_names)
    # problem → solution (arrow format)
    (r'问题[：:]\s*(.+?)\s*(?:→|->)\s*解决[：:]\s*(.+)', 'problem', ['problem', 'solution']),
    # lesson → avoidance
    (r'教训[：:]\s*(.+?)\s*(?:→|->)\s*避免[：:]\s*(.+)', 'lesson', ['lesson', 'avoidance']),
    # decision → reason
    (r'决策[：:]\s*(.+?)\s*(?:→|->)\s*原因[：:]\s*(.+)', 'decision', ['decision', 'reason']),
    # pattern/design → description (arrow with continuation)
    (r'(?:设计)?模式[：:]\s*(.+?)\s*(?:→|->)\s*(.+)', 'pattern_desc', ['pattern', 'detail']),
    # optimization → result
    (r'(?:性能)?优化[：:]\s*(.+?)\s*(?:→|->)\s*(.+)', 'optimization', ['what', 'result']),
    # experience → (arrow with continuation, generic)
    (r'经验[：:]\s*(.+?)\s*(?:→|->)\s*(.+)', 'generic_arrow', ['subject', 'detail']),
    # problem (sentence, no arrow)
    (r'问题[：:]\s*(.+?)(?:(?:，|,)\s*(.+))?$', 'problem_no_arrow', ['problem', 'detail']),
    # lesson (sentence, no arrow)
    (r'教训[：:]\s*(.+?)(?:(?:，|,)\s*(.+))?$', 'lesson_no_arrow', ['lesson', 'detail']),
    # decision (sentence, no arrow)
    (r'决策[：:]\s*(.+?)(?:(?:，|,)\s*(.+))?$', 'decision_no_arrow', ['decision', 'detail']),
]


def strip_prefix(name: str) -> str:
    """Remove known degraded prefixes."""
    for prefix in ('经验: ', '经验：', '经验:', '经验: 经验：'):
        if name.startswith(prefix):
            name = name[len(prefix):]
    return name.strip()


def reparse_text(text: str) -> Optional[Dict[str, Any]]:
    """Try to parse degraded text into structured entry data."""
    text = text.strip()
    if not text:
        return None

    for regex, entry_type, group_names in REPARSE_PATTERNS:
        m = re.match(regex, text, re.IGNORECASE | re.DOTALL)
        if not m:
            continue

        groups = list(m.groups())
        groups = [g.strip() if g else '' for g in groups]

        if entry_type == 'problem':
            problem, solution = groups[0], groups[1]
            return {
                'type': 'problem',
                'name': problem[:60],
                'inferred_category': 'problem',
                'content': {
                    'problem_name': problem,
                    'symptoms': [problem],
                    'root_causes': [],
                    'solutions': [{'description': solution}],
                    'prevention': []
                }
            }

        elif entry_type == 'lesson':
            lesson, avoidance = groups[0], groups[1]
            return {
                'type': 'experience',
                'name': lesson[:60],
                'inferred_category': 'experience',
                'content': {
                    'description': lesson,
                    'context': '教训总结',
                    'solution': avoidance,
                    'pitfalls': [lesson],
                    'related_tech': []
                }
            }

        elif entry_type == 'decision':
            decision, reason = groups[0], groups[1]
            return {
                'type': 'experience',
                'name': decision[:60],
                'inferred_category': 'pattern',
                'content': {
                    'description': f"{decision}（{reason}）",
                    'context': '架构/技术决策',
                    'solution': decision,
                    'pitfalls': [],
                    'related_tech': []
                }
            }

        elif entry_type == 'pattern_desc':
            pattern_name, detail = groups[0], groups[1]
            return {
                'type': 'pattern',
                'name': pattern_name[:60],
                'inferred_category': 'pattern',
                'content': {
                    'pattern_name': pattern_name,
                    'category': 'design',
                    'description': f"{pattern_name} → {detail}",
                    'when_to_use': '',
                    'structure': '',
                    'example': '',
                    'pros': [],
                    'cons': []
                }
            }

        elif entry_type == 'optimization':
            what, result = groups[0], groups[1]
            return {
                'type': 'experience',
                'name': what[:60],
                'inferred_category': 'experience',
                'content': {
                    'description': f"{what} → {result}",
                    'context': '性能优化',
                    'solution': result,
                    'pitfalls': [],
                    'related_tech': []
                }
            }

        elif entry_type == 'generic_arrow':
            subject, detail = groups[0], groups[1]
            return {
                'type': 'experience',
                'name': subject[:60],
                'inferred_category': 'experience',
                'content': {
                    'description': f"{subject} → {detail}",
                    'context': '',
                    'solution': detail,
                    'pitfalls': [],
                    'related_tech': []
                }
            }

        elif entry_type == 'problem_no_arrow':
            problem = groups[0]
            detail = groups[1] if len(groups) > 1 and groups[1] else ''
            return {
                'type': 'problem',
                'name': problem[:60],
                'inferred_category': 'problem',
                'content': {
                    'problem_name': problem,
                    'symptoms': [problem],
                    'root_causes': [],
                    'solutions': [{'description': detail}] if detail else [],
                    'prevention': []
                }
            }

        elif entry_type in ('lesson_no_arrow', 'decision_no_arrow'):
            subject = groups[0]
            detail = groups[1] if len(groups) > 1 and groups[1] else ''
            cat = 'experience'
            return {
                'type': 'experience',
                'name': subject[:60],
                'inferred_category': cat,
                'content': {
                    'description': f"{subject}. {detail}" if detail else subject,
                    'context': '教训总结' if 'lesson' in entry_type else '架构/技术决策',
                    'solution': detail or subject,
                    'pitfalls': [subject] if 'lesson' in entry_type else [],
                    'related_tech': []
                }
            }

    return None


def is_degraded(entry: Dict[str, Any]) -> bool:
    """Check if an entry is a degraded flat-text entry."""
    name = entry.get('name', '')
    content = entry.get('content', {})
    ctx = content.get('context', '')

    if name.startswith('经验: ') or name.startswith('经验：'):
        return True

    if ctx == '用户记录的经验' and content.get('description') == content.get('solution'):
        return True

    return False


def migrate_entry(
    entry: Dict[str, Any],
    old_path: Path,
    kb_root: Path,
    dry_run: bool = False
) -> Dict[str, Any]:
    """Migrate a single degraded entry. Returns migration report."""
    report = {
        'id': entry.get('id', '?'),
        'old_name': entry.get('name', '?'),
        'old_category': entry.get('category', '?'),
        'action': 'skip',
        'new_name': None,
        'new_category': None,
    }

    raw_name = entry.get('name', '')
    original_text = strip_prefix(raw_name)

    content = entry.get('content', {})
    desc = content.get('description', '')
    if desc and len(desc) > len(original_text):
        original_text = desc

    parsed = reparse_text(original_text)

    if parsed is None:
        if raw_name.startswith('经验: ') or raw_name.startswith('经验：'):
            new_name = strip_prefix(raw_name)[:60]
            if not dry_run:
                entry['name'] = new_name
                entry['updated_at'] = datetime.now().isoformat()
                with open(old_path, 'w', encoding='utf-8') as f:
                    json.dump(entry, f, indent=2, ensure_ascii=False)
            report['action'] = 'strip_prefix'
            report['new_name'] = new_name
            report['new_category'] = entry.get('category')
        else:
            report['action'] = 'skip'
        return report

    new_name = parsed['name']
    new_category = parsed.get('inferred_category', entry.get('category', 'experience'))
    new_content = parsed['content']

    old_related = content.get('related_tech', [])
    if old_related and 'related_tech' in new_content:
        new_content['related_tech'] = old_related

    report['action'] = 'migrate'
    report['new_name'] = new_name
    report['new_category'] = new_category

    if dry_run:
        return report

    entry['name'] = new_name
    entry['content'] = new_content
    entry['updated_at'] = datetime.now().isoformat()

    old_category = entry.get('category', 'experience')
    if new_category != old_category:
        entry['category'] = new_category
        new_dir = kb_root / CATEGORY_DIRS.get(new_category, f"{new_category}s")
        new_dir.mkdir(parents=True, exist_ok=True)
        new_path = new_dir / old_path.name
        if not new_path.exists():
            with open(new_path, 'w', encoding='utf-8') as f:
                json.dump(entry, f, indent=2, ensure_ascii=False)
            old_path.unlink()
            report['moved'] = str(new_path.relative_to(kb_root))
        else:
            with open(old_path, 'w', encoding='utf-8') as f:
                json.dump(entry, f, indent=2, ensure_ascii=False)
    else:
        with open(old_path, 'w', encoding='utf-8') as f:
            json.dump(entry, f, indent=2, ensure_ascii=False)

    return report


def rebuild_indexes(kb_root: Path) -> None:
    """Rebuild all indexes from entry files."""
    global_index = {
        'trigger_index': {},
        'category_index': {},
        'stats': {'total_entries': 0, 'by_category': {}},
        'recent_entries': [],
        'last_updated': datetime.now().isoformat()
    }

    for category, cat_dir_name in CATEGORY_DIRS.items():
        cat_dir = kb_root / cat_dir_name
        if not cat_dir.exists():
            continue

        cat_entries = []
        for entry_file in sorted(cat_dir.glob('*.json')):
            if entry_file.name == 'index.json':
                continue
            try:
                with open(entry_file, 'r', encoding='utf-8') as f:
                    entry = json.load(f)
            except (json.JSONDecodeError, IOError, UnicodeDecodeError):
                continue

            eid = entry.get('id', entry_file.stem)
            cat_entries.append({
                'id': eid,
                'name': entry.get('name', 'Unknown'),
                'created_at': entry.get('created_at', ''),
                'updated_at': entry.get('updated_at', '')
            })

            for trigger in entry.get('triggers', []):
                tl = trigger.lower()
                global_index['trigger_index'].setdefault(tl, [])
                if eid not in global_index['trigger_index'][tl]:
                    global_index['trigger_index'][tl].append(eid)

            global_index['category_index'].setdefault(cat_dir_name, [])
            if eid not in global_index['category_index'][cat_dir_name]:
                global_index['category_index'][cat_dir_name].append(eid)

        cat_index = {
            'entries': cat_entries,
            'last_updated': datetime.now().isoformat()
        }
        cat_index_path = cat_dir / 'index.json'
        atomic_write_json(cat_index_path, cat_index)

    global_index['stats']['total_entries'] = sum(
        len(entries) for entries in global_index['category_index'].values()
    )
    global_index['stats']['by_category'] = {
        cat: len(entries) for cat, entries in global_index['category_index'].items()
    }

    all_entries = []
    for entries in global_index['category_index'].values():
        all_entries.extend(entries)
    global_index['recent_entries'] = all_entries[-20:]

    atomic_write_json(kb_root / 'index.json', global_index)


def retrigger_all(kb_root: Path, dry_run: bool = False) -> Dict[str, int]:
    """
    Re-extract triggers for ALL entries using the improved extract_triggers().
    
    This fixes noisy triggers left from earlier summarizer versions by
    re-running the (now cleaned) extraction logic on every entry.
    """
    try:
        from store import extract_triggers as _extract
    except ImportError:
        _scripts_root_local = Path(__file__).parent.parent
        if str(_scripts_root_local) not in sys.path:
            sys.path.insert(0, str(_scripts_root_local))
        sys.path.insert(0, str(Path(__file__).parent))
        from store import extract_triggers as _extract

    stats = {'total': 0, 'updated': 0, 'unchanged': 0}

    for cat_dir_name in CATEGORY_DIRS.values():
        cat_dir = kb_root / cat_dir_name
        if not cat_dir.exists():
            continue
        for entry_file in sorted(cat_dir.glob('*.json')):
            if entry_file.name == 'index.json':
                continue
            stats['total'] += 1
            try:
                with open(entry_file, 'r', encoding='utf-8') as f:
                    entry = json.load(f)
            except (json.JSONDecodeError, IOError, UnicodeDecodeError):
                continue

            old_triggers = set(entry.get('triggers', []))
            new_triggers = _extract(
                entry.get('name', ''),
                entry.get('content', {}),
                entry.get('tags'),
            )

            if set(new_triggers) != old_triggers:
                stats['updated'] += 1
                if not dry_run:
                    entry['triggers'] = new_triggers
                    entry['updated_at'] = datetime.now().isoformat()
                    with open(entry_file, 'w', encoding='utf-8') as f:
                        json.dump(entry, f, indent=2, ensure_ascii=False)
            else:
                stats['unchanged'] += 1

    return stats


def main():
    parser = argparse.ArgumentParser(description='Migrate degraded knowledge entries')
    parser.add_argument('--dry-run', action='store_true', help='Preview without changes')
    parser.add_argument('--rollback', action='store_true', help='Restore from backup')
    parser.add_argument('--retrigger', action='store_true',
                        help='Re-extract triggers for ALL entries using the improved logic')
    parser.add_argument('--kb-dir', help='Knowledge base directory')
    args = parser.parse_args()

    kb_root = Path(args.kb_dir) if args.kb_dir else get_knowledge_base_dir()

    if not kb_root.exists():
        print(f"Error: KB directory not found: {kb_root}", file=sys.stderr)
        sys.exit(1)

    backup_dir = kb_root.parent / 'knowledge-backup-pre-migration'

    if args.rollback:
        if not backup_dir.exists():
            print("No backup found to restore.", file=sys.stderr)
            sys.exit(1)
        shutil.rmtree(kb_root)
        shutil.copytree(backup_dir, kb_root)
        print(f"Restored from {backup_dir}")
        return

    # ── Retrigger mode ──
    if args.retrigger:
        if not args.dry_run and not backup_dir.exists():
            shutil.copytree(kb_root, backup_dir)
            print(f"Backup created: {backup_dir}")

        stats = retrigger_all(kb_root, dry_run=args.dry_run)

        if not args.dry_run:
            print("Rebuilding indexes...")
            rebuild_indexes(kb_root)

        mode = "DRY RUN" if args.dry_run else "RETRIGGER"
        print(f"\n{'='*60}")
        print(f"  {mode} RESULTS")
        print(f"{'='*60}")
        print(f"  Total entries scanned:  {stats['total']}")
        print(f"  Triggers updated:       {stats['updated']}")
        print(f"  Unchanged:              {stats['unchanged']}")
        print(f"{'='*60}\n")
        if args.dry_run:
            print("This was a dry run. Run without --dry-run to apply changes.")
        return

    # ── Original degraded migration ──
    if not args.dry_run and not backup_dir.exists():
        shutil.copytree(kb_root, backup_dir)
        print(f"Backup created: {backup_dir}")

    reports = []
    entry_files = []
    for cat_dir_name in CATEGORY_DIRS.values():
        cat_dir = kb_root / cat_dir_name
        if cat_dir.exists():
            for f in cat_dir.glob('*.json'):
                if f.name != 'index.json':
                    entry_files.append(f)

    stats = {'total': 0, 'degraded': 0, 'migrated': 0, 'stripped': 0, 'skipped': 0, 'moved': 0}

    for entry_file in sorted(entry_files):
        stats['total'] += 1
        try:
            with open(entry_file, 'r', encoding='utf-8') as f:
                entry = json.load(f)
        except (json.JSONDecodeError, IOError, UnicodeDecodeError):
            continue

        if not is_degraded(entry):
            continue

        stats['degraded'] += 1
        report = migrate_entry(entry, entry_file, kb_root, dry_run=args.dry_run)
        reports.append(report)

        if report['action'] == 'migrate':
            stats['migrated'] += 1
            if 'moved' in report:
                stats['moved'] += 1
        elif report['action'] == 'strip_prefix':
            stats['stripped'] += 1
        else:
            stats['skipped'] += 1

    if not args.dry_run:
        print("Rebuilding indexes...")
        rebuild_indexes(kb_root)

    mode = "DRY RUN" if args.dry_run else "MIGRATION"
    print(f"\n{'='*60}")
    print(f"  {mode} RESULTS")
    print(f"{'='*60}")
    print(f"  Total entries scanned:  {stats['total']}")
    print(f"  Degraded entries found: {stats['degraded']}")
    print(f"  Migrated (structured):  {stats['migrated']}")
    print(f"    - Moved to new dir:   {stats['moved']}")
    print(f"  Prefix stripped only:   {stats['stripped']}")
    print(f"  Skipped (no match):     {stats['skipped']}")
    print(f"{'='*60}\n")

    if reports:
        print(f"{'Action':<16} {'Old Category':<12} {'New Category':<12} {'Name'}")
        print('-' * 90)
        for r in reports:
            name_display = r.get('new_name') or r.get('old_name', '?')
            print(f"{r['action']:<16} {r['old_category']:<12} {(r.get('new_category') or ''):<12} {name_display[:55]}")

    if args.dry_run:
        print(f"\nThis was a dry run. Run without --dry-run to apply changes.")
    else:
        print(f"\nBackup at: {backup_dir}")
        print(f"To rollback: python {__file__} --rollback")


if __name__ == '__main__':
    main()
