#!/usr/bin/env python3
"""
Knowledge Import/Export Utilities.

Provides backup, restore, and sharing capabilities for knowledge base.
"""

import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List

# Import centralized constants and path resolution
try:
    from core.config import CATEGORY_DIRS
    from core.path_resolver import get_knowledge_base_dir as get_kb_root
    from core.file_utils import atomic_write_json
except ImportError:
    CATEGORY_DIRS = {
        'experience': 'experiences', 'tech-stack': 'tech-stacks',
        'scenario': 'scenarios', 'problem': 'problems',
        'testing': 'testing', 'pattern': 'patterns', 'skill': 'skills',
    }

    def atomic_write_json(filepath, data):
        """Fallback atomic write"""
        filepath = Path(filepath)
        filepath.parent.mkdir(parents=True, exist_ok=True)
        with open(filepath, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)

    def get_kb_root() -> Path:
        """Fallback: Get knowledge base root directory."""
        import os
        env_path = os.environ.get('KNOWLEDGE_BASE_PATH')
        if env_path:
            kb_path = Path(env_path)
            if kb_path.exists():
                return kb_path
        opencode_kb = Path.home() / '.config' / 'opencode' / 'knowledge'
        opencode_kb.mkdir(parents=True, exist_ok=True)
        return opencode_kb


def load_json(path: Path) -> Dict[str, Any]:
    """Safely load JSON file."""
    if not path.exists():
        return {}
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError, UnicodeDecodeError):
        return {}


def export_all(output_path: str, format: str = "json") -> int:
    """
    Export all knowledge entries to a single file.
    
    Args:
        output_path: Output file path
        format: Export format ("json" or "markdown")
        
    Returns:
        Number of exported entries
    """
    kb_root = get_kb_root()
    all_entries: List[Dict[str, Any]] = []
    
    # Collect all entries
    for cat_dir in CATEGORY_DIRS.values():
        cat_path = kb_root / cat_dir
        if not cat_path.exists():
            continue
        
        for entry_file in cat_path.glob('*.json'):
            if entry_file.name == 'index.json':
                continue
            
            entry = load_json(entry_file)
            if entry:
                entry['_category'] = cat_dir
                entry['_source_file'] = entry_file.name
                all_entries.append(entry)
    
    # Export
    output_path = Path(output_path)
    
    if format == "json":
        export_data = {
            "exported_at": datetime.now().isoformat(),
            "total_entries": len(all_entries),
            "entries": all_entries
        }
        
        atomic_write_json(output_path, export_data)
        
    elif format == "markdown":
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(f"# Knowledge Base Export\n\n")
            f.write(f"**Exported at**: {datetime.now().isoformat()}\n")
            f.write(f"**Total entries**: {len(all_entries)}\n\n")
            f.write("---\n\n")
            
            for entry in all_entries:
                f.write(f"## {entry.get('name', 'Unnamed')}\n\n")
                f.write(f"- **ID**: {entry.get('id', 'unknown')}\n")
                f.write(f"- **Category**: {entry.get('_category', 'unknown')}\n")
                
                if 'content' in entry:
                    content = entry['content']
                    if isinstance(content, dict):
                        f.write(f"- **Description**: {content.get('description', 'N/A')}\n")
                
                f.write("\n---\n\n")
    
    return len(all_entries)


def import_all(input_path: str, merge_strategy: str = "skip") -> Dict[str, int]:
    """
    Import knowledge entries from a file.
    
    Args:
        input_path: Input file path
        merge_strategy: Merge strategy ("skip", "overwrite", or "merge")
        
    Returns:
        Dict with import statistics: {"imported": N, "skipped": N, "overwritten": N}
    """
    input_path = Path(input_path)
    
    if not input_path.exists():
        return {"imported": 0, "skipped": 0, "overwritten": 0, "error": "File not found"}
    
    # Load export data
    with open(input_path, 'r', encoding='utf-8') as f:
        export_data = json.load(f)
    
    entries = export_data.get('entries', [])
    
    stats = {
        "imported": 0,
        "skipped": 0,
        "overwritten": 0
    }
    
    kb_root = get_kb_root()
    
    for entry in entries:
        entry_id = entry.get('id')
        if not entry_id:
            continue
        
        category = entry.get('_category', 'experiences')
        cat_dir = CATEGORY_DIRS.get(category, 'experiences')
        cat_path = kb_root / cat_dir
        cat_path.mkdir(parents=True, exist_ok=True)
        
        entry_path = cat_path / f"{entry_id}.json"
        
        # Clean internal fields
        entry_clean = {k: v for k, v in entry.items() if not k.startswith('_')}
        
        if entry_path.exists():
            if merge_strategy == "skip":
                stats["skipped"] += 1
            elif merge_strategy == "overwrite":
                atomic_write_json(entry_path, entry_clean)
                stats["overwritten"] += 1
            elif merge_strategy == "merge":
                existing = load_json(entry_path)
                # Merge arrays (reviewer_notes, tags, etc.)
                for key in ['reviewer_notes', 'tags', 'triggers']:
                    if key in entry_clean and key in existing:
                        existing[key] = list(set(existing.get(key, []) + entry_clean.get(key, [])))
                atomic_write_json(entry_path, existing)
                stats["overwritten"] += 1
        else:
            atomic_write_json(entry_path, entry_clean)
            stats["imported"] += 1
    
    return stats
