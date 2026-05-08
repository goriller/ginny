#!/usr/bin/env python3
"""
Knowledge Lifecycle Management

管理知识库的生命周期：
- 衰减长期未使用的知识条目
- 清理低效条目
- 知识库健康度维护
"""

import json
import sys
from datetime import datetime, timedelta
from pathlib import Path
from typing import Any, Dict, List

# Import config constants, path resolution, and file utils
try:
    from core.config import DECAY_DAYS_THRESHOLD, DECAY_RATE, GC_EFFECTIVENESS_THRESHOLD, CATEGORY_DIRS
    from core.path_resolver import get_knowledge_base_dir as get_kb_root
    from core.file_utils import atomic_write_json
except ImportError:
    DECAY_DAYS_THRESHOLD = 90
    DECAY_RATE = 0.1
    GC_EFFECTIVENESS_THRESHOLD = 0.1
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


def decay_unused(days_threshold: int = DECAY_DAYS_THRESHOLD, decay_rate: float = DECAY_RATE) -> List[Dict[str, Any]]:
    """
    衰减长期未使用的知识条目。
    
    遍历所有知识条目，超过阈值天数未使用的条目 effectiveness 减少 decay_rate（最低 0）。
    
    Args:
        days_threshold: 未使用天数阈值（默认 90 天）
        decay_rate: 衰减率（默认 0.1，即 10%）
        
    Returns:
        受影响的条目列表
    """
    kb_root = get_kb_root()
    threshold_date = datetime.now() - timedelta(days=days_threshold)
    affected_entries: List[Dict[str, Any]] = []
    
    for cat_dir in CATEGORY_DIRS.values():
        cat_path = kb_root / cat_dir
        if not cat_path.exists():
            continue
        
        for entry_file in cat_path.glob('*.json'):
            if entry_file.name == 'index.json':
                continue
            
            entry = load_json(entry_file)
            if not entry:
                continue
            
            # Check last_used_at
            last_used_str = entry.get('last_used_at')
            if not last_used_str:
                # Never used, use created_at or skip
                last_used_str = entry.get('created_at')
                if not last_used_str:
                    continue
            
            try:
                last_used = datetime.fromisoformat(last_used_str.replace('Z', '+00:00'))
            except (ValueError, AttributeError):
                continue
            
            # Skip recently used entries
            if last_used > threshold_date:
                continue
            
            # Decay effectiveness
            current_effectiveness = entry.get('effectiveness', 0.5)
            new_effectiveness = max(0.0, current_effectiveness - decay_rate)
            
            if new_effectiveness != current_effectiveness:
                entry['effectiveness'] = new_effectiveness
                entry['last_decayed_at'] = datetime.now().isoformat()
                
                # Save using atomic write
                try:
                    atomic_write_json(entry_file, entry)
                    affected_entries.append({
                        'id': entry.get('id'),
                        'name': entry.get('name'),
                        'old_effectiveness': current_effectiveness,
                        'new_effectiveness': new_effectiveness
                    })
                except Exception as e:
                    print(f"Error updating {entry_file}: {e}", file=sys.stderr)
    
    return affected_entries


def get_stale_entries(effectiveness_threshold: float = GC_EFFECTIVENESS_THRESHOLD) -> List[Dict[str, Any]]:
    """
    获取低于有效性阈值的条目。
    
    Args:
        effectiveness_threshold: 有效性阈值（默认 0.1）
        
    Returns:
        低效条目列表
    """
    kb_root = get_kb_root()
    stale_entries: List[Dict[str, Any]] = []
    
    for cat_dir in CATEGORY_DIRS.values():
        cat_path = kb_root / cat_dir
        if not cat_path.exists():
            continue
        
        for entry_file in cat_path.glob('*.json'):
            if entry_file.name == 'index.json':
                continue
            
            entry = load_json(entry_file)
            if not entry:
                continue
            
            effectiveness = entry.get('effectiveness', 0.5)
            if effectiveness < effectiveness_threshold:
                stale_entries.append(entry)
    
    return stale_entries


def gc(threshold: float = GC_EFFECTIVENESS_THRESHOLD, dry_run: bool = False) -> List[Dict[str, Any]]:
    """
    清理低效条目（垃圾回收）。
    
    Args:
        threshold: 有效性阈值（默认 0.1）
        dry_run: 预览模式，只列出不删除
        
    Returns:
        被清理（或将被清理）的条目列表
    """
    stale_entries = get_stale_entries(threshold)
    
    if not dry_run:
        kb_root = get_kb_root()
        for entry in stale_entries:
            entry_id = entry.get('id', '')
            if not entry_id:
                continue

            # Prefer the stored 'category' field; fall back to ID prefix heuristic
            stored_category = entry.get('category', '')
            cat_dir = CATEGORY_DIRS.get(stored_category)
            if not cat_dir:
                # Fallback: try ID prefix (less reliable for multi-part names like 'tech-stack')
                id_prefix = entry_id.split('-')[0] if '-' in entry_id else ''
                cat_dir = CATEGORY_DIRS.get(id_prefix, 'experiences')

            entry_path = kb_root / cat_dir / f"{entry_id}.json"
            if entry_path.exists():
                try:
                    entry_path.unlink()
                except Exception as e:
                    print(f"Error deleting {entry_path}: {e}", file=sys.stderr)
    
    return stale_entries
