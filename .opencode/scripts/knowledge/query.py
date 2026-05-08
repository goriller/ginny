#!/usr/bin/env python3
"""
Unified Knowledge Query

统一知识库查询接口：
- 按关键字触发查询
- 按分类查询
- 按标签查询
- 全文搜索
"""

import argparse
import json
import os
import re
import sys
from datetime import datetime
from difflib import SequenceMatcher
from pathlib import Path
from typing import Any, Dict, List, Optional, Set

# Optional jieba import for Chinese tokenization
try:
    import jieba

    HAS_JIEBA = True
except ImportError:
    HAS_JIEBA = False

# Import config constants
try:
    from core.config import (
        FUZZY_MATCH_THRESHOLD,
        RELEVANCE_WEIGHTS,
        RECENCY_DECAY_DAYS,
        USAGE_NORMALIZATION,
        TOP_K_RESULTS,
        CATEGORY_DIRS,
        FUZZY_MATCH_EFF_SCALE,
        FUZZY_MATCH_REC_SCALE,
    )
except ImportError:
    FUZZY_MATCH_THRESHOLD = 0.72
    RELEVANCE_WEIGHTS = {
        "trigger_match": 0.4,
        "effectiveness": 0.3,
        "recency": 0.2,
        "usage": 0.1,
    }
    RECENCY_DECAY_DAYS = 365.0
    USAGE_NORMALIZATION = 100.0
    TOP_K_RESULTS = 10
    CATEGORY_DIRS = {
        "experience": "experiences",
        "tech-stack": "tech-stacks",
        "scenario": "scenarios",
        "problem": "problems",
        "testing": "testing",
        "pattern": "patterns",
        "skill": "skills",
    }
    FUZZY_MATCH_EFF_SCALE = 0.35
    FUZZY_MATCH_REC_SCALE = 0.50

SYNONYM_MAP = {
    # performance / optimization
    "优化": ["optimize", "optimization", "performance", "性能", "perf"],
    "optimize": ["优化", "optimization", "performance", "性能"],
    "performance": ["性能", "优化", "optimize", "perf"],
    "性能": ["performance", "优化", "optimize", "perf"],
    # rendering
    "渲染": ["render", "rendering", "paint", "repaint"],
    "render": ["渲染", "rendering", "paint"],
    "rendering": ["渲染", "render", "paint"],
    # list / virtualization
    "列表": ["list", "virtual", "virtualization", "scroll", "大列表", "windowing"],
    "list": ["列表", "virtual", "virtualization", "大列表"],
    "virtualization": ["列表", "virtual", "大列表", "windowing", "list"],
    "大列表": ["list", "virtualization", "列表", "virtual", "windowing"],
    # error / bug
    "错误": ["error", "bug", "issue", "报错", "exception"],
    "error": ["错误", "bug", "issue", "报错", "exception"],
    "bug": ["错误", "error", "issue", "报错"],
    "报错": ["error", "bug", "错误", "exception"],
    # testing
    "测试": ["test", "testing", "spec", "unittest"],
    "test": ["测试", "testing", "spec", "unittest"],
    "testing": ["测试", "test", "spec"],
    # deploy
    "部署": ["deploy", "deployment", "ci/cd", "release"],
    "deploy": ["部署", "deployment", "release"],
    # database
    "数据库": ["database", "db", "sql", "query"],
    "database": ["数据库", "db", "sql"],
    # auth
    "认证": ["auth", "authentication", "登录", "login"],
    "auth": ["认证", "authentication", "登录", "login"],
    "登录": ["login", "auth", "认证", "authentication"],
    "login": ["登录", "auth", "认证"],
    # cache
    "缓存": ["cache", "caching", "memoize", "memo"],
    "cache": ["缓存", "caching", "memoize"],
    # component / 组件
    "组件": ["component", "widget", "element"],
    "component": ["组件", "widget"],
    # state
    "状态": ["state", "store", "reducer"],
    "state": ["状态", "store", "reducer"],
    # refactor
    "重构": ["refactor", "restructure", "cleanup"],
    "refactor": ["重构", "restructure", "cleanup"],
    # async
    "异步": ["async", "asynchronous", "promise", "concurrent"],
    "async": ["异步", "asynchronous", "promise"],
    # memory
    "内存": ["memory", "leak", "gc", "heap"],
    "memory": ["内存", "leak", "gc", "heap"],
    # cross-origin
    "跨域": ["cors", "cross-origin", "proxy"],
    "cors": ["跨域", "cross-origin", "proxy"],
    # type / typing
    "类型": ["type", "typescript", "typing", "interface"],
    "type": ["类型", "typescript", "typing"],
    "typescript": ["type", "类型", "typing", "interface"],
    # --- 以下为新增同义词组 ---
    # request / response / HTTP
    "请求": ["request", "http", "fetch", "ajax", "xhr"],
    "request": ["请求", "http", "fetch", "ajax"],
    "响应": ["response", "reply", "返回"],
    "response": ["响应", "reply", "返回"],
    "接口": ["api", "endpoint", "interface", "rest"],
    "api": ["接口", "endpoint", "rest", "restful"],
    # proxy / middleware
    "代理": ["proxy", "middleware", "gateway"],
    "proxy": ["代理", "middleware", "gateway"],
    # container / docker
    "容器": ["container", "docker", "k8s", "kubernetes", "pod"],
    "container": ["容器", "docker", "pod"],
    "docker": ["容器", "container", "image", "dockerfile"],
    # CI/CD / pipeline
    "pipeline": ["ci", "cd", "workflow", "流水线", "github-actions"],
    "流水线": ["pipeline", "ci", "cd", "workflow"],
    # data structure
    "数组": ["array", "list", "slice"],
    "array": ["数组", "list", "slice"],
    "字典": ["dict", "map", "object", "hashmap"],
    "dict": ["字典", "map", "object", "hashmap"],
    "map": ["字典", "dict", "object", "hashmap"],
    # config / settings
    "配置": ["config", "configuration", "settings", "env"],
    "config": ["配置", "configuration", "settings"],
    # route / navigation
    "路由": ["route", "router", "navigation", "routing"],
    "route": ["路由", "router", "navigation", "routing"],
    # dependency / package
    "依赖": ["dependency", "package", "module", "npm", "pip"],
    "dependency": ["依赖", "package", "module"],
    "package": ["依赖", "dependency", "module", "library"],
    # log / monitoring
    "日志": ["log", "logging", "logger", "monitor"],
    "log": ["日志", "logging", "logger"],
    # concurrency
    "并发": ["concurrent", "parallel", "thread", "multiprocess", "多线程"],
    "concurrent": ["并发", "parallel", "thread", "多线程"],
    "parallel": ["并发", "concurrent", "多线程"],
    # validation / form
    "校验": ["validate", "validation", "check", "verify", "验证"],
    "validate": ["校验", "validation", "verify", "验证"],
    "表单": ["form", "input", "field"],
    "form": ["表单", "input", "field"],
    # pagination / scroll
    "分页": ["pagination", "paging", "page", "infinite-scroll"],
    "pagination": ["分页", "paging", "infinite-scroll"],
    # WebSocket / real-time
    "websocket": ["ws", "socket", "realtime", "实时"],
    "实时": ["realtime", "websocket", "socket", "sse"],
    # file / upload
    "上传": ["upload", "file", "multipart"],
    "upload": ["上传", "file", "multipart"],
    # permission / role
    "权限": ["permission", "role", "rbac", "acl", "授权"],
    "permission": ["权限", "role", "rbac", "acl"],
}


def expand_with_synonyms(tokens: List[str], max_expansions: int = 3, max_total: int = 30) -> List[str]:
    """Expand a token list with synonyms to improve recall.
    
    Args:
        tokens: List of tokens to expand
        max_expansions: Max synonyms per token (default 3)
        max_total: Max total expanded tokens (default 30)
    
    Returns:
        Expanded token list (capped at max_total)
    """
    expanded = list(tokens)
    for token in tokens:
        if len(expanded) >= max_total:
            break
        key = token.lower()
        synonyms = SYNONYM_MAP.get(key, [])
        for syn in synonyms[:max_expansions]:
            if len(expanded) >= max_total:
                break
            if syn.lower() not in {t.lower() for t in expanded}:
                expanded.append(syn)
    return expanded


def tokenize(text: str) -> List[str]:
    """
    Tokenize text into a list of tokens.

    Uses jieba for Chinese tokenization if available; otherwise falls back
    to whitespace splitting + regex extraction of Chinese characters and
    ASCII alphanumeric sequences.

    Args:
        text: Input text (may contain Chinese, English, or mixed content)

    Returns:
        List of non-empty token strings
    """
    if HAS_JIEBA:
        tokens = jieba.lcut(text)
        return [t for t in tokens if t.strip() and re.search(r"[\w\u4e00-\u9fff]", t)]
    else:
        return re.findall(r"[\u4e00-\u9fff]+|[a-zA-Z0-9]+", text.lower())


# Import atomic_write_json from file_utils
try:
    from core.file_utils import atomic_write_json
except ImportError:
    # Fallback if file_utils is not available
    def atomic_write_json(filepath, data):
        """Fallback atomic write"""
        filepath = Path(filepath)
        filepath.parent.mkdir(parents=True, exist_ok=True)
        with open(filepath, "w", encoding="utf-8") as f:
            json.dump(data, f, indent=2, ensure_ascii=False)


# 路径解析 — 委托给 core.path_resolver（单一权威实现）
try:
    from core.path_resolver import get_knowledge_base_dir as get_kb_root
except ImportError:

    def get_kb_root() -> Path:  # type: ignore[misc]
        """Fallback: 仅在 core.path_resolver 不可用时使用"""
        env_path = os.environ.get("KNOWLEDGE_BASE_PATH")
        if env_path:
            kb_path = Path(env_path)
            if kb_path.exists():
                return kb_path
        opencode_kb = Path.home() / ".config" / "opencode" / "knowledge"
        opencode_kb.mkdir(parents=True, exist_ok=True)
        return opencode_kb


def load_json(path: Path) -> Dict[str, Any]:
    """Safely load JSON file."""
    if not path.exists():
        return {}
    try:
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError, UnicodeDecodeError):
        return {}


def update_usage(entry_path: Path, entry_data: Dict[str, Any]) -> None:
    """
    Update usage statistics for a knowledge entry.

    Increments usage_count and updates last_used_at timestamp.
    Uses atomic write to prevent corruption.

    Args:
        entry_path: Path to the entry JSON file
        entry_data: Entry data dict (will be modified in place)
    """
    if not entry_path.exists():
        return

    # Update usage statistics
    entry_data["usage_count"] = entry_data.get("usage_count", 0) + 1
    entry_data["last_used_at"] = datetime.now().isoformat()

    # Atomic write to prevent corruption
    try:
        atomic_write_json(entry_path, entry_data)
    except Exception:
        # Silently fail if write fails (don't break query)
        pass


def batch_update_usage(entries: List[tuple]) -> None:
    """
    Batch update usage statistics for multiple knowledge entries.
    
    More efficient than individual update_usage() calls when updating
    multiple entries at once (e.g., final query results).
    
    Args:
        entries: List of (entry_path, entry_data) tuples
    """
    for entry_path, entry_data in entries:
        update_usage(entry_path, entry_data)


def fuzzy_match(
    query_tokens: List[str],
    trigger_tokens: List[str],
    threshold: float = FUZZY_MATCH_THRESHOLD,
) -> float:
    """
    计算查询词和触发词之间的模糊匹配分数。

    使用 difflib.SequenceMatcher 进行模糊匹配。

    Args:
        query_tokens: 查询词列表
        trigger_tokens: 触发词列表
        threshold: 匹配阈值（默认 0.6）

    Returns:
        最高相似度分数（0.0 到 1.0）
    """
    max_score = 0.0

    for query in query_tokens:
        query_lower = query.lower()
        for trigger in trigger_tokens:
            trigger_lower = trigger.lower()

            # Calculate similarity
            matcher = SequenceMatcher(None, query_lower, trigger_lower)
            score = matcher.ratio()

            if score > max_score:
                max_score = score

            # Early exit if perfect match
            if max_score == 1.0:
                return 1.0

    return max_score if max_score >= threshold else 0.0


def compute_relevance(entry: Dict[str, Any], query_tokens: List[str]) -> float:
    """
    计算知识条目的相关性分数。

    综合评分 = 触发词匹配分 × 0.4 + effectiveness × 0.3 + recency × 0.2 + usage_count 归一化 × 0.1

    关键约束：
    - 关键词条目（非语义）若匹配分为 0，直接返回 0.0，防止高 effectiveness 的跨项目
      条目仅凭时效性/使用次数进入结果集（根本 bug 修复）。
    - 仅模糊匹配（match_score < 2）时，降低 effectiveness/recency 权重，减少跨项目噪音。

    Args:
        entry: 知识条目
        query_tokens: 查询词列表

    Returns:
        相关性分数（0.0 到 1.0）
    """
    raw_match = entry.get("_match_score", 0)
    is_semantic = entry.get("_match_type") == "semantic"

    # Critical fix: keyword entry with zero match score = irrelevant.
    # Without this gate, entries with effectiveness=1.0 + recency=1.0 score 0.5,
    # exceeding the old high_threshold=0.45 despite having NO keyword overlap.
    if raw_match == 0 and not is_semantic:
        return 0.0

    # 1. Trigger match score (normalize; accumulates as multiples of 3 for exact+partial combos)
    match_score = min(1.0, raw_match / 3.0)

    # 2. Effectiveness (0.3 weight base)
    effectiveness = entry.get("effectiveness", 0.5)

    # 3. Recency
    last_used_str = entry.get("last_used_at") or entry.get("created_at")
    if last_used_str:
        try:
            last_used = datetime.fromisoformat(last_used_str.replace("Z", "+00:00"))
            days_since_use = (datetime.now(last_used.tzinfo) - last_used).days
            recency = max(0.0, 1.0 - (days_since_use / RECENCY_DECAY_DAYS))
        except (ValueError, TypeError, OSError):
            recency = 0.5
    else:
        recency = 0.5

    # 4. Usage count (normalized)
    usage_count = entry.get("usage_count", 0)
    usage_normalized = min(1.0, usage_count / USAGE_NORMALIZATION)

    # For fuzzy-only matches (raw_match < 2), scale down effectiveness/recency
    # so high-effectiveness cross-project entries don't rank above weak-matching ones.
    if not is_semantic and raw_match < 2:
        eff_weight = RELEVANCE_WEIGHTS["effectiveness"] * FUZZY_MATCH_EFF_SCALE
        rec_weight = RELEVANCE_WEIGHTS["recency"] * FUZZY_MATCH_REC_SCALE
    else:
        eff_weight = RELEVANCE_WEIGHTS["effectiveness"]
        rec_weight = RELEVANCE_WEIGHTS["recency"]

    relevance = (
        match_score * RELEVANCE_WEIGHTS["trigger_match"]
        + effectiveness * eff_weight
        + recency * rec_weight
        + usage_normalized * RELEVANCE_WEIGHTS["usage"]
    )

    return relevance


def get_global_index() -> Dict[str, Any]:
    """Load global index."""
    return load_json(get_kb_root() / "index.json")


def query_by_triggers(
    triggers: List[str],
    limit: int = TOP_K_RESULTS,
    use_synonyms: bool = True,
) -> List[Dict[str, Any]]:
    """
    根据触发关键字查询知识。

    支持精确匹配、部分匹配、模糊匹配和同义词扩展。
    查询全局知识库 (~/.config/opencode/knowledge/)。

    Args:
        triggers: 触发关键字列表
        limit: 返回结果数量限制
        use_synonyms: 是否启用同义词扩展（默认 True）

    Returns:
        匹配的知识条目列表，按匹配度排序
    """
    if use_synonyms:
        triggers = expand_with_synonyms(triggers, max_expansions=2)

    return _query_by_triggers_single_root(triggers, limit, get_kb_root())


def query_by_triggers_in(
    triggers: List[str],
    kb_root: Path,
    limit: int = TOP_K_RESULTS,
    use_synonyms: bool = True,
) -> List[Dict[str, Any]]:
    """
    根据触发关键字查询指定知识库（支持项目级知识库）。

    与 query_by_triggers 相同逻辑，但允许指定自定义 kb_root，
    用于搜索项目级知识库 ($PROJECT_ROOT/.opencode/knowledge/)。

    Args:
        triggers: 触发关键字列表
        kb_root: 知识库根目录路径
        limit: 返回结果数量限制
        use_synonyms: 是否启用同义词扩展（默认 True）

    Returns:
        匹配的知识条目列表，按匹配度排序
    """
    if use_synonyms:
        triggers = expand_with_synonyms(triggers, max_expansions=2)

    return _query_by_triggers_single_root(triggers, limit, kb_root)


def _query_by_triggers_single_root(
    triggers: List[str], limit: int, kb_root
) -> List[Dict[str, Any]]:
    """
    Internal: query a single knowledge base root by triggers.
    
    Optimizations:
    - Trigger cap: Limits triggers to MAX_TRIGGERS (20) to prevent performance issues
    - Early termination: Skips fuzzy matching if exact+partial matches are sufficient
    - Deferred usage update: No longer writes usage stats during query (caller handles)
    """
    MAX_TRIGGERS = 20
    
    index = load_json(kb_root / "index.json")
    trigger_index = index.get("trigger_index", {})

    # Trigger cap: Limit number of triggers to prevent performance degradation
    if len(triggers) > MAX_TRIGGERS:
        # Sort by length (descending) to keep more specific triggers
        triggers = sorted(triggers, key=len, reverse=True)[:MAX_TRIGGERS]

    # Track matches with type information
    entry_info: Dict[str, Dict[str, Any]] = {}  # entry_id -> {score, match_type}

    for trigger in triggers:
        trigger_lower = trigger.lower()

        # 1. Exact match (highest priority)
        if trigger_lower in trigger_index:
            for entry_id in trigger_index[trigger_lower]:
                if entry_id not in entry_info:
                    entry_info[entry_id] = {"score": 0, "match_type": "exact"}
                entry_info[entry_id]["score"] += 3  # Highest weight

        # 2. Partial match (medium priority)
        for indexed_trigger, entry_ids in trigger_index.items():
            if trigger_lower in indexed_trigger or indexed_trigger in trigger_lower:
                for entry_id in entry_ids:
                    if entry_id not in entry_info:
                        entry_info[entry_id] = {"score": 0, "match_type": "partial"}
                    elif entry_info[entry_id]["match_type"] == "exact":
                        continue  # Don't downgrade exact match
                    entry_info[entry_id]["score"] += 2

    # Early termination: Skip fuzzy matching if we have enough high-quality results
    skip_fuzzy = False
    if len(entry_info) >= limit:
        min_score = min(info["score"] for info in entry_info.values())
        if min_score >= 2:  # At least partial match
            skip_fuzzy = True

    # 3. Fuzzy match (lowest priority) — only if not skipped
    if not skip_fuzzy:
        for trigger in triggers:
            trigger_tokens = tokenize(trigger)
            for indexed_trigger, entry_ids in trigger_index.items():
                indexed_tokens = tokenize(indexed_trigger)
                fuzzy_score = fuzzy_match(
                    trigger_tokens if trigger_tokens else [trigger],
                    indexed_tokens if indexed_tokens else [indexed_trigger],
                    threshold=FUZZY_MATCH_THRESHOLD,
                )
                if fuzzy_score > 0:
                    for entry_id in entry_ids:
                        if entry_id not in entry_info:
                            entry_info[entry_id] = {
                                "score": fuzzy_score,  # Use fuzzy score directly
                                "match_type": "fuzzy",
                            }

    if not entry_info:
        return []

    # Sort by score
    sorted_entries = sorted(
        entry_info.items(), key=lambda x: x[1]["score"], reverse=True
    )

    # Load entry details
    results: List[Dict[str, Any]] = []
    for entry_id, info in sorted_entries[:limit]:
        # Determine category from entry_id
        category = entry_id.split("-")[0] if "-" in entry_id else "experience"
        cat_dir = CATEGORY_DIRS.get(category, "experiences")

        entry_path = kb_root / cat_dir / f"{entry_id}.json"
        if entry_path.exists():
            entry = load_json(entry_path)
            entry["_match_score"] = info["score"]
            entry["_match_type"] = info["match_type"]
            entry["_entry_path"] = entry_path  # Store path for deferred usage update

            # Compute relevance score
            entry["_relevance_score"] = compute_relevance(entry, triggers)

            results.append(entry)

    # Sort by relevance score
    results.sort(key=lambda x: x.get("_relevance_score", 0), reverse=True)

    return results


def query_by_category(category: str, limit: int = 20) -> List[Dict[str, Any]]:
    """
    按分类查询所有知识条目。

    Args:
        category: 知识分类
        limit: 返回数量限制

    Returns:
        该分类下的知识条目列表
    """
    kb_root = get_kb_root()
    cat_dir = CATEGORY_DIRS.get(category)
    if not cat_dir:
        return []

    cat_path = kb_root / cat_dir
    if not cat_path.exists():
        return []

    results: List[Dict[str, Any]] = []
    for entry_file in cat_path.glob("*.json"):
        if entry_file.name == "index.json":
            continue
        entry = load_json(entry_file)
        if entry:
            results.append(entry)
        if len(results) >= limit:
            break

    # Sort by effectiveness and usage
    results.sort(
        key=lambda x: (x.get("effectiveness", 0), x.get("usage_count", 0)), reverse=True
    )
    return results


def query_by_tags(tags: List[str], limit: int = 10) -> List[Dict[str, Any]]:
    """
    按标签查询知识。

    Args:
        tags: 标签列表
        limit: 返回数量限制

    Returns:
        匹配标签的知识条目列表
    """
    kb_root = get_kb_root()
    tags_lower = [t.lower() for t in tags]
    results: List[Dict[str, Any]] = []

    # Search all categories
    for cat_dir in CATEGORY_DIRS.values():
        cat_path = kb_root / cat_dir
        if not cat_path.exists():
            continue

        for entry_file in cat_path.glob("*.json"):
            if entry_file.name == "index.json":
                continue
            entry = load_json(entry_file)
            if not entry:
                continue

            entry_tags = [t.lower() for t in entry.get("tags", [])]
            if any(tag in entry_tags for tag in tags_lower):
                results.append(entry)
                if len(results) >= limit:
                    break

        if len(results) >= limit:
            break

    return results


def search_content(keyword: str, limit: int = 10) -> List[Dict[str, Any]]:
    """
    全文搜索知识内容。

    Args:
        keyword: 搜索关键字
        limit: 返回数量限制

    Returns:
        包含关键字的知识条目列表
    """
    kb_root = get_kb_root()
    keyword_lower = keyword.lower()
    results: List[Dict[str, Any]] = []

    # Search all categories
    for cat_dir in CATEGORY_DIRS.values():
        cat_path = kb_root / cat_dir
        if not cat_path.exists():
            continue

        for entry_file in cat_path.glob("*.json"):
            if entry_file.name == "index.json":
                continue

            # Read raw content for search
            try:
                content_str = entry_file.read_text(encoding="utf-8").lower()
                if keyword_lower in content_str:
                    entry = load_json(entry_file)
                    if entry:
                        # Update usage statistics
                        update_usage(entry_file, entry)
                        results.append(entry)
                        if len(results) >= limit:
                            break
            except IOError:
                continue

        if len(results) >= limit:
            break

    return results


def get_entry(entry_id: str) -> Optional[Dict[str, Any]]:
    """
    获取单个知识条目。

    Args:
        entry_id: 知识条目ID

    Returns:
        知识条目，如不存在则返回 None
    """
    kb_root = get_kb_root()

    # Try to determine category from ID
    parts = entry_id.split("-")
    if parts:
        category = parts[0]
        cat_dir = CATEGORY_DIRS.get(category)
        if cat_dir:
            entry_path = kb_root / cat_dir / f"{entry_id}.json"
            if entry_path.exists():
                return load_json(entry_path)

    # Fallback: search all categories
    for cat_dir in CATEGORY_DIRS.values():
        entry_path = kb_root / cat_dir / f"{entry_id}.json"
        if entry_path.exists():
            return load_json(entry_path)

    return None


def query_semantic(query_text: str, limit: int = TOP_K_RESULTS) -> List[Dict[str, Any]]:
    """
    Semantic search using BM25.

    Args:
        query_text: Natural language query
        limit: Max results

    Returns:
        Matched knowledge entries with _relevance_score
    """
    try:
        from embedding import search as bm25_search
    except ImportError:
        tokens = query_text.replace(",", " ").split()
        return query_by_triggers(tokens, limit=limit)

    kb_root = get_kb_root()
    hits = bm25_search(query_text, kb_root, top_k=limit)

    results: List[Dict[str, Any]] = []
    for entry_id, score in hits:
        entry = get_entry(entry_id)
        if entry:
            entry["_relevance_score"] = score
            entry["_match_type"] = "semantic"
            results.append(entry)

    return results


def query_hybrid(query_text: str, limit: int = TOP_K_RESULTS) -> List[Dict[str, Any]]:
    """
    Hybrid search: combine keyword + semantic results.

    Args:
        query_text: Query string
        limit: Max results

    Returns:
        Merged and deduplicated results
    """
    tokens = query_text.replace(",", " ").split()
    keyword_results = query_by_triggers(tokens, limit=limit)
    semantic_results = query_semantic(query_text, limit=limit)

    seen_ids: Set[str] = set()
    merged: List[Dict[str, Any]] = []

    for entry in keyword_results:
        eid = entry.get("id", "")
        if eid not in seen_ids:
            seen_ids.add(eid)
            merged.append(entry)

    for entry in semantic_results:
        eid = entry.get("id", "")
        if eid not in seen_ids:
            seen_ids.add(eid)
            merged.append(entry)

    merged.sort(key=lambda x: x.get("_relevance_score", 0), reverse=True)
    final_results = merged[:limit]
    
    # Batch update usage statistics for final results only
    entries_to_update = []
    for entry in final_results:
        if "_entry_path" in entry:
            entries_to_update.append((entry["_entry_path"], entry))
    
    if entries_to_update:
        batch_update_usage(entries_to_update)
    
    return final_results


def get_stats() -> Dict[str, Any]:
    """获取知识库统计信息。"""
    index = get_global_index()
    return {
        "version": index.get("version", "unknown"),
        "last_updated": index.get("last_updated"),
        "stats": index.get("stats", {}),
        "trigger_count": len(index.get("trigger_index", {})),
        "recent_entries": index.get("recent_entries", [])[:5],
    }


def format_output(data: Any, fmt: str = "json") -> str:
    """Format output based on type."""
    if fmt == "json":
        return json.dumps(data, indent=2, ensure_ascii=False)
    elif fmt == "markdown":
        if isinstance(data, list):
            lines = []
            for entry in data:
                lines.append(f"### {entry.get('name', 'Unknown')}")
                lines.append(f"- **Category**: {entry.get('category', 'N/A')}")
                lines.append(
                    f"- **Triggers**: {', '.join(entry.get('triggers', [])[:5])}"
                )
                content = entry.get("content", {})
                if "description" in content:
                    lines.append(
                        f"- **Description**: {content['description'][:100]}..."
                    )
                lines.append("")
            return "\n".join(lines)
        elif isinstance(data, dict):
            lines = []
            for key, value in data.items():
                if isinstance(value, (list, dict)):
                    lines.append(f"**{key}**: {json.dumps(value, ensure_ascii=False)}")
                else:
                    lines.append(f"**{key}**: {value}")
            return "\n".join(lines)
    return str(data)


def main():
    parser = argparse.ArgumentParser(
        description="Query unified knowledge base",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Query by triggers (most common)
  python knowledge_query.py --trigger react,hooks,state
  
  # Query by category
  python knowledge_query.py --category problem
  
  # Search content
  python knowledge_query.py --search "跨域"
  
  # Get single entry
  python knowledge_query.py --id problem-cors-abc123
  
  # Get stats
  python knowledge_query.py --stats
        """,
    )

    parser.add_argument("--trigger", "-t", help="Comma-separated trigger keywords")
    parser.add_argument(
        "--category", "-c", choices=list(CATEGORY_DIRS.keys()), help="Query by category"
    )
    parser.add_argument("--tags", help="Comma-separated tags")
    parser.add_argument("--search", "-s", help="Full-text search keyword")
    parser.add_argument("--id", help="Get entry by ID")
    parser.add_argument(
        "--stats", action="store_true", help="Show knowledge base stats"
    )
    parser.add_argument("--limit", "-l", type=int, default=10, help="Result limit")
    parser.add_argument(
        "--format",
        "-f",
        choices=["json", "markdown"],
        default="json",
        help="Output format",
    )
    parser.add_argument(
        "--mode",
        "-m",
        choices=["keyword", "semantic", "hybrid"],
        default="keyword",
        help="Search mode: keyword (default), semantic, hybrid",
    )

    args = parser.parse_args()

    if args.stats:
        result = get_stats()
    elif args.id:
        result = get_entry(args.id)
        if not result:
            print(f"Entry not found: {args.id}", file=sys.stderr)
            sys.exit(1)
    elif args.trigger:
        triggers = [t.strip() for t in args.trigger.split(",")]
        query_text = " ".join(triggers)
        if args.mode == "semantic":
            result = query_semantic(query_text, args.limit)
        elif args.mode == "hybrid":
            result = query_hybrid(query_text, args.limit)
        else:
            result = query_by_triggers(triggers, args.limit)
    elif args.category:
        result = query_by_category(args.category, args.limit)
    elif args.tags:
        tags = [t.strip() for t in args.tags.split(",")]
        result = query_by_tags(tags, args.limit)
    elif args.search:
        result = search_content(args.search, args.limit)
    else:
        # Default: show stats
        result = get_stats()

    print(format_output(result, args.format))


if __name__ == "__main__":
    main()
