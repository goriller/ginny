#!/usr/bin/env python3
"""
Knowledge Base Dashboard

Provides statistics and visualization for the knowledge base.
"""

import json
import os
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List

# Import centralized constants and path resolution
try:
    from core.config import CATEGORY_DIRS
    from core.path_resolver import get_knowledge_base_dir as get_kb_root
except ImportError:
    CATEGORY_DIRS = {
        'experience': 'experiences', 'tech-stack': 'tech-stacks',
        'scenario': 'scenarios', 'problem': 'problems',
        'testing': 'testing', 'pattern': 'patterns', 'skill': 'skills',
    }

    def get_kb_root() -> Path:
        """Fallback: Get knowledge base root directory."""
        env_path = os.environ.get('KNOWLEDGE_BASE_PATH')
        if env_path:
            kb_path = Path(env_path)
            if kb_path.exists():
                return kb_path
        opencode_kb = Path.home() / '.config' / 'opencode' / 'knowledge'
        opencode_kb.mkdir(parents=True, exist_ok=True)
        return opencode_kb


def _load_json(path: Path) -> Dict[str, Any]:
    """Safely load a JSON file."""
    if not path.exists():
        return {}
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError, UnicodeDecodeError):
        return {}


def generate_stats(kb_root: Path) -> Dict[str, Any]:
    """
    Generate statistics for the knowledge base.

    Args:
        kb_root: Path to knowledge base root directory

    Returns:
        Statistics dictionary:
        {
            "total_entries": int,
            "by_category": {"experience": N, ...},
            "top_used": [{"name": "...", "usage_count": N}, ...],  # top 10
            "recently_added": [{"name": "...", "created_at": "..."}, ...],  # top 10
            "stale_count": int,  # effectiveness < 0.2
            "avg_effectiveness": float,
        }
    """
    by_category: Dict[str, int] = {}
    all_entries: List[Dict[str, Any]] = []

    for category, cat_dir in CATEGORY_DIRS.items():
        cat_path = kb_root / cat_dir
        if not cat_path.exists():
            by_category[category] = 0
            continue

        count = 0
        for entry_file in cat_path.glob('*.json'):
            if entry_file.name == 'index.json':
                continue
            entry = _load_json(entry_file)
            if entry:
                count += 1
                all_entries.append(entry)
        by_category[category] = count

    # Top used (by usage_count)
    sorted_by_usage = sorted(
        all_entries,
        key=lambda e: e.get('usage_count', 0),
        reverse=True
    )
    top_used = [
        {"name": e.get('name', 'unknown'), "usage_count": e.get('usage_count', 0)}
        for e in sorted_by_usage[:10]
    ]

    # Recently added (by created_at)
    def parse_dt(entry: Dict) -> datetime:
        ts = entry.get('created_at', '')
        if ts:
            try:
                return datetime.fromisoformat(ts.replace('Z', '+00:00'))
            except (ValueError, TypeError):
                pass
        return datetime.min

    sorted_by_created = sorted(all_entries, key=parse_dt, reverse=True)
    recently_added = [
        {"name": e.get('name', 'unknown'), "created_at": e.get('created_at', '')}
        for e in sorted_by_created[:10]
    ]

    # Stale count (effectiveness < 0.2)
    stale_count = sum(
        1 for e in all_entries if e.get('effectiveness', 1.0) < 0.2
    )

    # Average effectiveness
    effectiveness_values = [
        e.get('effectiveness', 1.0) for e in all_entries if 'effectiveness' in e
    ]
    avg_effectiveness = (
        sum(effectiveness_values) / len(effectiveness_values)
        if effectiveness_values else 0.0
    )

    return {
        "total_entries": len(all_entries),
        "by_category": by_category,
        "top_used": top_used,
        "recently_added": recently_added,
        "stale_count": stale_count,
        "avg_effectiveness": round(avg_effectiveness, 3),
    }


def format_dashboard(stats: Dict[str, Any]) -> str:
    """
    Format statistics as a human-readable text dashboard.

    Args:
        stats: Statistics dict from generate_stats()

    Returns:
        Formatted text string
    """
    if stats["total_entries"] == 0:
        return "Knowledge base is empty"

    lines = []
    lines.append("=" * 50)
    lines.append("Knowledge Base Dashboard")
    lines.append("=" * 50)
    lines.append(f"Total entries: {stats['total_entries']}")
    lines.append(f"Avg effectiveness: {stats['avg_effectiveness']:.3f}")
    if stats['stale_count'] > 0:
        lines.append(f"⚠ Stale entries (effectiveness < 0.2): {stats['stale_count']}")
    lines.append("")

    # Category distribution (ASCII bar chart)
    lines.append("Category Distribution:")
    max_count = max(stats['by_category'].values()) if stats['by_category'] else 1
    bar_width = 20
    for category, count in stats['by_category'].items():
        if count == 0:
            continue
        filled = int((count / max(max_count, 1)) * bar_width)
        bar = "█" * filled + "░" * (bar_width - filled)
        lines.append(f"  {category:<12} [{bar}] {count}")
    lines.append("")

    # Top used
    if stats['top_used']:
        lines.append("Top Used Entries:")
        lines.append(f"  {'Name':<35} {'Uses':>5}")
        lines.append("  " + "-" * 42)
        for item in stats['top_used'][:5]:
            name = item['name'][:33]
            lines.append(f"  {name:<35} {item['usage_count']:>5}")
        lines.append("")

    # Recently added
    if stats['recently_added']:
        lines.append("Recently Added:")
        for item in stats['recently_added'][:5]:
            created = item['created_at'][:10] if item['created_at'] else 'unknown'
            lines.append(f"  [{created}] {item['name']}")
    lines.append("=" * 50)

    return "\n".join(lines)
