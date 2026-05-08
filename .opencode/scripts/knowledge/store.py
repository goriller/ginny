#!/usr/bin/env python3
"""
Unified Knowledge Store

统一知识库存储接口，支持所有分类：
- experience: 经验积累
- tech-stack: 技术栈积累
- scenario: 场景积累
- problem: 问题积累
- testing: 测试积累
- pattern: 编程范式
- skill: 编程技能
"""

import argparse
import hashlib
import json
import os
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional

# Atomic write support
try:
    _scripts_dir = Path(__file__).parent.parent
    if str(_scripts_dir) not in sys.path:
        sys.path.insert(0, str(_scripts_dir))
    from core.file_utils import atomic_write_json as _atomic_write_json
except ImportError:
    def _atomic_write_json(filepath, data):
        """Fallback non-atomic write"""
        filepath = Path(filepath)
        filepath.parent.mkdir(parents=True, exist_ok=True)
        with open(filepath, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)

# Import centralized constants and path resolution
try:
    from core.config import CATEGORY_DIRS, VALID_CATEGORIES
    from core.path_resolver import get_knowledge_base_dir as get_kb_root
except ImportError:
    CATEGORY_DIRS = {
        'experience': 'experiences', 'tech-stack': 'tech-stacks',
        'scenario': 'scenarios', 'problem': 'problems',
        'testing': 'testing', 'pattern': 'patterns', 'skill': 'skills',
    }
    VALID_CATEGORIES = list(CATEGORY_DIRS.keys())

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


def generate_id(category: str, name: str) -> str:
    """Generate unique ID for knowledge entry."""
    hash_input = f"{category}:{name}:{datetime.now().isoformat()}"
    hash_suffix = hashlib.md5(hash_input.encode()).hexdigest()[:8]
    name_slug = name.lower().replace(' ', '-').replace('/', '-')[:30]
    return f"{category}-{name_slug}-{hash_suffix}"


def load_json(path: Path) -> Dict[str, Any]:
    """Safely load JSON file."""
    if not path.exists():
        return {}
    try:
        with open(path, 'r', encoding='utf-8') as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError, UnicodeDecodeError):
        return {}


def save_json(path: Path, data: Dict[str, Any]) -> None:
    """Save JSON file with atomic write to prevent data corruption."""
    _atomic_write_json(path, data)


NOISE_PREFIXES = ('经验:', '经验：', '经验: ', '经验： ',
                   '最佳实践:', '最佳实践：', '最佳实践: ', '最佳实践： ',
                   '注意:', '注意：', '注意: ', '注意： ',
                   '偏好:', '偏好：', '偏好: ', '偏好： ',
                   '问题:', '问题：', '问题: ', '问题： ',
                   '解决:', '解决：', '解决: ', '解决： ')

STOP_WORDS = {
    # English
    'a', 'an', 'the', 'is', 'are', 'was', 'were', 'be', 'been',
    'to', 'of', 'in', 'for', 'on', 'with', 'at', 'by', 'from',
    'and', 'or', 'but', 'not', 'as', 'if', 'when', 'than',
    'this', 'that', 'these', 'those', 'has', 'have', 'had',
    'will', 'would', 'can', 'could', 'should', 'may', 'might',
    # Chinese noise tokens (common in degraded entries)
    '经验', '解决', '问题', '注意', '偏好', '最佳实践',
    '需要', '使用', '通过', '进行', '可以', '应该',
    '的', '了', '在', '是', '和', '与', '或', '但',
}

import re as _re


def _clean_name(name: str) -> str:
    """Strip known noise prefixes from entry names."""
    for prefix in NOISE_PREFIXES:
        if name.startswith(prefix):
            name = name[len(prefix):]
    return name.strip()


def extract_triggers(name: str, content: Dict[str, Any], tags: Optional[List[str]] = None) -> List[str]:
    """
    自动从知识内容中提取触发关键字。
    
    提取规则：
    1. 名称分词（去除噪音前缀）
    2. 相关技术栈
    3. 显式标签
    4. 内容中的关键术语
    5. description / solution 中的技术关键词
    """
    triggers: set = set()
    
    cleaned_name = _clean_name(name)
    
    # English words (3+ chars) from name
    en_words = _re.findall(r'\b[a-zA-Z][a-zA-Z0-9\-\.]+\b', cleaned_name.lower())
    triggers.update(w for w in en_words if len(w) >= 3)
    # Chinese phrases (2-4 chars) from name
    zh_words = _re.findall(r'[\u4e00-\u9fa5]{2,4}', cleaned_name)
    triggers.update(zh_words)
    # Tech terms with hyphens/dots (e.g. react-query, vue.js)
    tech_terms = _re.findall(r'[a-zA-Z]+[\-\.][a-zA-Z]+', cleaned_name.lower())
    triggers.update(tech_terms)
    
    if tags:
        triggers.update(t.lower() for t in tags)
    
    related_tech = content.get('related_tech', [])
    triggers.update(t.lower() for t in related_tech)
    
    if 'tech_name' in content:
        triggers.add(content['tech_name'].lower())
    if 'framework' in content:
        triggers.add(content['framework'].lower())
    
    if 'symptoms' in content:
        for symptom in content['symptoms']:
            en_sym = _re.findall(r'\b[a-zA-Z][a-zA-Z0-9\-\.]+\b', symptom.lower())
            triggers.update(w for w in en_sym if len(w) >= 3)
            zh_sym = _re.findall(r'[\u4e00-\u9fa5]{2,4}', symptom)
            triggers.update(zh_sym)
    
    # Extract keywords from description and solution fields
    for field in ('description', 'solution', 'problem_name', 'scenario_name'):
        text = content.get(field, '')
        if text and isinstance(text, str):
            en_f = _re.findall(r'\b[a-zA-Z][a-zA-Z0-9\-\.]+\b', text.lower())
            triggers.update(w for w in en_f if len(w) >= 3)
            zh_f = _re.findall(r'[\u4e00-\u9fa5]{2,4}', text)
            triggers.update(zh_f)
    
    triggers = {t for t in triggers if len(t) > 1 and t.lower() not in STOP_WORDS}
    
    return sorted(list(triggers))


def update_global_index(kb_root: Path, entry_id: str, category: str, triggers: List[str]) -> None:
    """Update the global index with new entry and trigger mappings."""
    index_path = kb_root / 'index.json'
    index = load_json(index_path)
    
    # Ensure structure
    if 'trigger_index' not in index:
        index['trigger_index'] = {}
    if 'category_index' not in index:
        index['category_index'] = {d: [] for d in CATEGORY_DIRS.values()}
    if 'stats' not in index:
        index['stats'] = {'total_entries': 0, 'by_category': {}}
    
    # Update trigger index
    for trigger in triggers:
        if trigger not in index['trigger_index']:
            index['trigger_index'][trigger] = []
        if entry_id not in index['trigger_index'][trigger]:
            index['trigger_index'][trigger].append(entry_id)
    
    # Update category index
    cat_dir = CATEGORY_DIRS.get(category, category)
    if cat_dir not in index['category_index']:
        index['category_index'][cat_dir] = []
    if entry_id not in index['category_index'][cat_dir]:
        index['category_index'][cat_dir].append(entry_id)
    
    # Update stats
    index['stats']['total_entries'] = sum(
        len(entries) for entries in index['category_index'].values()
    )
    index['stats']['by_category'] = {
        cat: len(entries) for cat, entries in index['category_index'].items()
    }
    
    # Update recent entries (keep last 20)
    if 'recent_entries' not in index:
        index['recent_entries'] = []
    index['recent_entries'].insert(0, entry_id)
    index['recent_entries'] = index['recent_entries'][:20]
    
    index['last_updated'] = datetime.now().isoformat()
    save_json(index_path, index)


def update_category_index(kb_root: Path, category: str, entry_id: str, name: str) -> None:
    """Update the category-specific index."""
    cat_dir = CATEGORY_DIRS.get(category, category)
    index_path = kb_root / cat_dir / 'index.json'
    index = load_json(index_path)
    
    if 'entries' not in index:
        index['entries'] = []
    
    # Check if entry already exists
    existing = next((e for e in index['entries'] if e.get('id') == entry_id), None)
    if existing:
        existing['name'] = name
        existing['updated_at'] = datetime.now().isoformat()
    else:
        index['entries'].append({
            'id': entry_id,
            'name': name,
            'created_at': datetime.now().isoformat()
        })
    
    index['last_updated'] = datetime.now().isoformat()
    save_json(index_path, index)


def store_knowledge(
    category: str,
    name: str,
    content: Dict[str, Any],
    sources: Optional[List[str]] = None,
    tags: Optional[List[str]] = None,
    triggers: Optional[List[str]] = None,
    entry_id: Optional[str] = None,
    kb_root: Optional[Path] = None,
    project_path: Optional[str] = None,
) -> Dict[str, Any]:
    """
    存储知识条目到统一知识库。
    
    Args:
        category: 知识分类 (experience, tech-stack, scenario, problem, testing, pattern, skill)
        name: 知识条目名称
        content: 知识内容 (符合对应分类的 schema)
        sources: 来源列表 (GitHub URL, 会话 ID 等)
        tags: 额外标签
        triggers: 显式指定的触发关键字 (可选，会自动提取)
        entry_id: 已有条目ID (用于更新)
        kb_root: 知识库根目录 (可选，主要用于测试注入；默认通过 get_kb_root() 自动解析)
    
    Returns:
        创建/更新的知识条目
    """
    if category not in VALID_CATEGORIES:
        raise ValueError(f"Invalid category: {category}. Must be one of: {VALID_CATEGORIES}")
    
    kb_root = kb_root or get_kb_root()
    cat_dir = CATEGORY_DIRS[category]
    
    # Generate or use existing ID
    if not entry_id:
        entry_id = generate_id(category, name)
    
    # Auto-extract triggers if not provided
    if triggers is None:
        triggers = extract_triggers(name, content, tags)
    else:
        # Merge auto-extracted with explicit triggers
        auto_triggers = extract_triggers(name, content, tags)
        triggers = list(set(triggers + auto_triggers))
    
    # Build entry
    now = datetime.now().isoformat()
    entry_path = kb_root / cat_dir / f"{entry_id}.json"
    
    # Load existing entry if updating
    existing = load_json(entry_path) if entry_path.exists() else {}
    
    entry: Dict[str, Any] = {
        'id': entry_id,
        'category': category,
        'name': name,
        'triggers': triggers,
        'content': content,
        'sources': sources or existing.get('sources', []),
        'tags': tags or existing.get('tags', []),
        'created_at': existing.get('created_at', now),
        'updated_at': now,
        'usage_count': existing.get('usage_count', 0),
        'effectiveness': existing.get('effectiveness', 0.5),
    }

    # Stamp project_path when storing to a project-local KB.
    # This field enables retrieval scoring to apply a cross-project penalty
    # when the querying project differs from the entry's origin.
    resolved_project_path = project_path
    if resolved_project_path is None and kb_root is not None:
        # Heuristic: if kb_root is inside a project's .opencode/, infer project_path
        _opencode = kb_root.parent
        if _opencode.name == '.opencode' and (_opencode.parent / '.git').exists():
            resolved_project_path = str(_opencode.parent)
    if resolved_project_path:
        entry['project_path'] = resolved_project_path
    elif 'project_path' in existing:
        entry['project_path'] = existing['project_path']
    
    # Merge sources if updating
    if existing.get('sources'):
        all_sources = list(set(existing['sources'] + (sources or [])))
        entry['sources'] = all_sources
    
    # Save entry
    save_json(entry_path, entry)
    
    # Update indexes
    update_category_index(kb_root, category, entry_id, name)
    update_global_index(kb_root, entry_id, category, triggers)
    
    return entry


def store_experience(
    name: str,
    description: str,
    solution: str,
    context: Optional[str] = None,
    pitfalls: Optional[List[str]] = None,
    related_tech: Optional[List[str]] = None,
    sources: Optional[List[str]] = None,
    tags: Optional[List[str]] = None,
    triggers: Optional[List[str]] = None,
    kb_root: Optional[Path] = None,
) -> Dict[str, Any]:
    """便捷方法：存储经验类知识。"""
    content = {
        'description': description,
        'context': context or '',
        'solution': solution,
        'pitfalls': pitfalls or [],
        'related_tech': related_tech or []
    }
    return store_knowledge('experience', name, content, sources, tags,
                            triggers=triggers, kb_root=kb_root)


def store_tech_stack(
    tech_name: str,
    best_practices: Optional[List[str]] = None,
    conventions: Optional[List[str]] = None,
    common_patterns: Optional[List[str]] = None,
    gotchas: Optional[List[str]] = None,
    version: Optional[str] = None,
    sources: Optional[List[str]] = None,
    triggers: Optional[List[str]] = None,
    kb_root: Optional[Path] = None,
    # Allow extra kwargs for flexibility (e.g. name, description passed from tests)
    **kwargs
) -> Dict[str, Any]:
    """便捷方法：存储技术栈知识。"""
    # Support 'name' as alias for tech_name (test compatibility)
    if 'name' in kwargs and not tech_name:
        tech_name = kwargs.pop('name')
    content = {
        'tech_name': tech_name,
        'version': kwargs.get('version', version or ''),
        'best_practices': best_practices or [],
        'conventions': conventions or [],
        'common_patterns': common_patterns or [],
        'gotchas': gotchas or []
    }
    return store_knowledge('tech-stack', tech_name, content, sources,
                           [tech_name.lower()], triggers=triggers, kb_root=kb_root)


def store_scenario(
    scenario_name: str,
    description: str,
    typical_approach: str,
    steps: Optional[List[str]] = None,
    considerations: Optional[List[str]] = None,
    related_tech: Optional[List[str]] = None,
    sources: Optional[List[str]] = None,
    triggers: Optional[List[str]] = None,
    kb_root: Optional[Path] = None
) -> Dict[str, Any]:
    """便捷方法：存储场景知识。"""
    content = {
        'scenario_name': scenario_name,
        'description': description,
        'typical_approach': typical_approach,
        'steps': steps or [],
        'considerations': considerations or [],
        'related_tech': related_tech or []
    }
    return store_knowledge('scenario', scenario_name, content, sources,
                           triggers=triggers, kb_root=kb_root)


def store_problem(
    problem_name: str,
    symptoms: List[str],
    root_causes: List[str],
    solutions: List[Dict[str, str]],
    prevention: Optional[List[str]] = None,
    sources: Optional[List[str]] = None,
    tags: Optional[List[str]] = None,
    triggers: Optional[List[str]] = None,
    kb_root: Optional[Path] = None
) -> Dict[str, Any]:
    """便捷方法：存储问题知识。"""
    content = {
        'problem_name': problem_name,
        'symptoms': symptoms,
        'root_causes': root_causes,
        'solutions': solutions,
        'prevention': prevention or []
    }
    return store_knowledge('problem', problem_name, content, sources, tags,
                           triggers=triggers, kb_root=kb_root)


def store_testing(
    name: str,
    testing_type: str,
    framework: Optional[str] = None,
    best_practices: Optional[List[str]] = None,
    patterns: Optional[List[str]] = None,
    anti_patterns: Optional[List[str]] = None,
    example_structure: Optional[str] = None,
    sources: Optional[List[str]] = None,
    triggers: Optional[List[str]] = None,
    kb_root: Optional[Path] = None
) -> Dict[str, Any]:
    """便捷方法：存储测试知识。"""
    content = {
        'testing_type': testing_type,
        'framework': framework or '',
        'best_practices': best_practices or [],
        'patterns': patterns or [],
        'anti_patterns': anti_patterns or [],
        'example_structure': example_structure or ''
    }
    tags = [testing_type, 'testing']
    if framework:
        tags.append(framework.lower())
    return store_knowledge('testing', name, content, sources, tags,
                           triggers=triggers, kb_root=kb_root)


def store_pattern(
    pattern_name: str,
    pattern_category: str,
    description: str,
    when_to_use: str,
    structure: Optional[str] = None,
    example: Optional[str] = None,
    pros: Optional[List[str]] = None,
    cons: Optional[List[str]] = None,
    sources: Optional[List[str]] = None,
    triggers: Optional[List[str]] = None,
    kb_root: Optional[Path] = None
) -> Dict[str, Any]:
    """便捷方法：存储编程范式。"""
    content = {
        'pattern_name': pattern_name,
        'category': pattern_category,
        'description': description,
        'when_to_use': when_to_use,
        'structure': structure or '',
        'example': example or '',
        'pros': pros or [],
        'cons': cons or []
    }
    return store_knowledge('pattern', pattern_name, content, sources, [pattern_category],
                           triggers=triggers, kb_root=kb_root)


def store_skill(
    skill_name: str,
    level: str,
    description: str,
    key_concepts: Optional[List[str]] = None,
    practical_tips: Optional[List[str]] = None,
    common_mistakes: Optional[List[str]] = None,
    sources: Optional[List[str]] = None,
    triggers: Optional[List[str]] = None,
    kb_root: Optional[Path] = None
) -> Dict[str, Any]:
    """便捷方法：存储编程技能。"""
    content = {
        'skill_name': skill_name,
        'level': level,
        'description': description,
        'key_concepts': key_concepts or [],
        'practical_tips': practical_tips or [],
        'common_mistakes': common_mistakes or []
    }
    return store_knowledge('skill', skill_name, content, sources, [level],
                           triggers=triggers, kb_root=kb_root)


def main():
    parser = argparse.ArgumentParser(
        description='Store knowledge to unified knowledge base',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Store from JSON input
  echo '{"name": "CORS Issue", "content": {...}}' | python knowledge_store.py --category problem
  
  # Store with explicit parameters
  python knowledge_store.py --category tech-stack --name "React" --content '{"best_practices": [...]}'
  
  # Import from stdin
  python knowledge_store.py --from-json --source "https://github.com/..."
        """
    )
    
    parser.add_argument('--category', '-c', choices=VALID_CATEGORIES,
                        help='Knowledge category')
    parser.add_argument('--name', '-n', help='Knowledge entry name')
    parser.add_argument('--content', help='JSON content string')
    parser.add_argument('--source', '-s', help='Source URL or identifier')
    parser.add_argument('--tags', '-t', help='Comma-separated tags')
    parser.add_argument('--from-json', action='store_true',
                        help='Read full entry from stdin as JSON')
    
    args = parser.parse_args()
    
    if args.from_json:
        # Read from stdin
        try:
            data = json.load(sys.stdin)
            entry = store_knowledge(
                category=data.get('category', 'experience'),
                name=data.get('name', 'Unnamed'),
                content=data.get('content', {}),
                sources=[args.source] if args.source else data.get('sources'),
                tags=data.get('tags'),
                triggers=data.get('triggers')
            )
            print(json.dumps(entry, indent=2, ensure_ascii=False))
        except json.JSONDecodeError as e:
            print(f"Error parsing JSON: {e}", file=sys.stderr)
            sys.exit(1)
    elif args.category and args.name:
        content = json.loads(args.content) if args.content else {}
        sources = [args.source] if args.source else None
        tags = args.tags.split(',') if args.tags else None
        
        entry = store_knowledge(
            category=args.category,
            name=args.name,
            content=content,
            sources=sources,
            tags=tags
        )
        print(json.dumps(entry, indent=2, ensure_ascii=False))
    else:
        parser.print_help()


if __name__ == '__main__':
    main()
