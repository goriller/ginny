#!/usr/bin/env python3
"""
Lightweight Semantic Search via BM25.

Provides BM25-based search for the knowledge base.
Zero external dependencies — uses only Python stdlib + optional jieba.
"""

import json
import math
import os
import re
import tempfile
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Tuple

# 复用 query.py 已有的分词器和同义词扩展
try:
    from query import tokenize as _base_tokenize, expand_with_synonyms
except ImportError:

    def _base_tokenize(text: str) -> List[str]:
        return re.findall(r"[\u4e00-\u9fff]+|[a-zA-Z0-9]+", text.lower())

    def expand_with_synonyms(tokens: List[str], max_expansions: int = 3) -> List[str]:
        return tokens


def _bm25_tokenize(text: str) -> List[str]:
    """
    BM25-optimized tokenizer.

    When jieba is unavailable, _base_tokenize groups consecutive CJK chars
    into one long token (e.g. "修复跨域请求问题" → ["修复跨域请求问题"]).
    BM25 needs finer granularity, so we split those runs into individual
    characters, enabling character-level matching.
    With jieba, tokens are already properly segmented — no extra splitting.
    """
    tokens = _base_tokenize(text)
    result: List[str] = []
    for token in tokens:
        if len(token) > 1 and all("\u4e00" <= c <= "\u9fff" for c in token):
            result.extend(list(token))
        else:
            result.append(token)
    return result


try:
    from core.config import CATEGORY_DIRS
except ImportError:
    CATEGORY_DIRS = {
        "experience": "experiences",
        "tech-stack": "tech-stacks",
        "scenario": "scenarios",
        "problem": "problems",
        "testing": "testing",
        "pattern": "patterns",
        "skill": "skills",
    }

# BM25 参数
BM25_K1 = 1.5  # 词频饱和参数
BM25_B = 0.75  # 文档长度归一化参数

# 向后兼容标志：始终为 True（BM25 无需外部依赖）
HAS_EMBEDDING = True


def _entry_to_text(entry: Dict[str, Any]) -> str:
    """Convert a knowledge entry to a searchable text representation."""
    parts = []
    if entry.get("name"):
        parts.append(entry["name"])
    content = entry.get("content", {})
    if isinstance(content, dict):
        desc = content.get("description", "")
        if desc:
            parts.append(desc)
    triggers = entry.get("triggers", [])
    if triggers:
        parts.append(" ".join(triggers))
    return " ".join(parts)


class BM25Index:
    """
    Pure-Python BM25 index for small document collections.

    Designed for knowledge bases with <10,000 entries.
    Supports Chinese + English mixed tokenization via query.tokenize().
    """

    def __init__(self, documents: List[str], doc_ids: List[str]):
        """
        Build index from documents.

        Args:
            documents: List of document text strings
            doc_ids: Parallel list of document IDs
        """
        self.doc_ids = doc_ids
        self.doc_count = len(documents)
        self.avgdl = 0.0
        self.doc_lens: List[int] = []
        self.doc_freqs: List[Dict[str, int]] = []  # per-doc term frequencies
        self.idf: Dict[str, float] = {}  # inverse document frequency
        self._build(documents)

    def _build(self, documents: List[str]) -> None:
        df: Dict[str, int] = {}  # document frequency: how many docs contain term
        total_len = 0

        for doc_text in documents:
            tokens = _bm25_tokenize(doc_text)
            self.doc_lens.append(len(tokens))
            total_len += len(tokens)

            # Term frequency for this document
            tf: Dict[str, int] = {}
            seen_terms: set = set()
            for token in tokens:
                t = token.lower()
                tf[t] = tf.get(t, 0) + 1
                seen_terms.add(t)
            self.doc_freqs.append(tf)

            # Update document frequency
            for term in seen_terms:
                df[term] = df.get(term, 0) + 1

        self.avgdl = total_len / max(self.doc_count, 1)

        # IDF with smoothing: log((N - df + 0.5) / (df + 0.5) + 1)
        for term, freq in df.items():
            self.idf[term] = math.log(
                (self.doc_count - freq + 0.5) / (freq + 0.5) + 1.0
            )

    def score(self, query_tokens: List[str]) -> List[float]:
        """
        Compute BM25 scores for all documents against query tokens.

        Args:
            query_tokens: Tokenized and synonym-expanded query

        Returns:
            List of float scores, parallel to self.doc_ids
        """
        scores = [0.0] * self.doc_count

        for token in query_tokens:
            t = token.lower()
            if t not in self.idf:
                continue
            idf = self.idf[t]

            for i in range(self.doc_count):
                tf = self.doc_freqs[i].get(t, 0)
                if tf == 0:
                    continue
                dl = self.doc_lens[i]
                # BM25 formula
                numerator = tf * (BM25_K1 + 1)
                denominator = tf + BM25_K1 * (1 - BM25_B + BM25_B * dl / self.avgdl)
                scores[i] += idf * numerator / denominator

        return scores

    def search(
        self, query_tokens: List[str], top_k: int = 10
    ) -> List[Tuple[str, float]]:
        """
        Search for top-k documents matching query.

        Args:
            query_tokens: Tokenized and synonym-expanded query
            top_k: Number of results

        Returns:
            List of (doc_id, score) sorted by score descending
        """
        scores = self.score(query_tokens)

        # Argsort descending without numpy
        indexed = [(score, i) for i, score in enumerate(scores) if score > 0]
        indexed.sort(key=lambda x: x[0], reverse=True)

        return [(self.doc_ids[i], score) for score, i in indexed[:top_k]]


# Module-level cache (avoids rebuilding per query within same process)
_cached_index: Dict[str, Any] = {}


def _load_entries(kb_root: Path) -> Tuple[List[Dict[str, Any]], List[str], List[str]]:
    """
    Load all knowledge entries from kb_root.

    Returns:
        (entries, entry_ids, texts)
    """
    entries: List[Dict[str, Any]] = []
    entry_ids: List[str] = []
    texts: List[str] = []

    for cat_dir in CATEGORY_DIRS.values():
        cat_path = kb_root / cat_dir
        if not cat_path.exists():
            continue
        for entry_file in cat_path.glob("*.json"):
            if entry_file.name == "index.json":
                continue
            try:
                with open(entry_file, "r", encoding="utf-8") as f:
                    entry = json.load(f)
            except (json.JSONDecodeError, IOError, UnicodeDecodeError):
                continue
            if not entry:
                continue

            text = _entry_to_text(entry)
            if text.strip():
                entries.append(entry)
                entry_ids.append(entry.get("id", entry_file.stem))
                texts.append(text)

    return entries, entry_ids, texts


def _cleanup_old_cache(kb_root: Path) -> None:
    """Remove old sentence-transformers cache files if present."""
    for name in [".embedding_cache.npz", ".embedding_ids.json"]:
        old_file = kb_root / name
        if old_file.exists():
            try:
                old_file.unlink()
            except OSError:
                pass


def _get_file_stats(kb_root: Path) -> Tuple[int, float]:
    """
    Get file count and newest mtime for cache validity check.

    Returns:
        (file_count, newest_mtime)
    """
    file_count = 0
    newest_mtime = 0.0

    for cat_dir in CATEGORY_DIRS.values():
        cat_path = kb_root / cat_dir
        if not cat_path.exists():
            continue
        for entry_file in cat_path.glob("*.json"):
            if entry_file.name == "index.json":
                continue
            file_count += 1
            try:
                mtime = entry_file.stat().st_mtime
                if mtime > newest_mtime:
                    newest_mtime = mtime
            except OSError:
                pass

    return file_count, newest_mtime


def _load_cache(kb_root: Path) -> Dict[str, Any]:
    """
    Load persistent cache from disk.

    Returns:
        Cache dict or empty dict if not found/invalid
    """
    cache_file = kb_root / ".bm25_cache.json"
    if not cache_file.exists():
        return {}

    try:
        with open(cache_file, "r", encoding="utf-8") as f:
            cache = json.load(f)

        if cache.get("version") != 2:
            return {}

        return cache
    except (json.JSONDecodeError, IOError, UnicodeDecodeError):
        return {}


def _save_cache(kb_root: Path, cache: Dict[str, Any]) -> None:
    """
    Save persistent cache to disk using atomic write.

    Uses tempfile + rename for atomic write.
    """
    cache_file = kb_root / ".bm25_cache.json"

    try:
        fd, temp_path = tempfile.mkstemp(
            dir=kb_root,
            prefix=".bm25_cache.tmp.",
            suffix=".json"
        )
        try:
            with os.fdopen(fd, "w", encoding="utf-8") as f:
                json.dump(cache, f, ensure_ascii=False, indent=2)

            os.replace(temp_path, cache_file)
        except Exception:
            if os.path.exists(temp_path):
                os.unlink(temp_path)
            raise
    except OSError:
        pass


def _get_or_build_index(kb_root: Path) -> Tuple[Any, List[str], List[Dict[str, Any]]]:
    """
    Get cached index or build new one.

    Returns:
        (bm25_index, entry_ids, entries)
    """
    cache_key = str(kb_root)

    if cache_key in _cached_index:
        cached = _cached_index[cache_key]
        return cached["index"], cached["entry_ids"], cached["entries"]

    _cleanup_old_cache(kb_root)

    current_file_count, current_newest_mtime = _get_file_stats(kb_root)

    disk_cache = _load_cache(kb_root)
    cache_valid = False

    if disk_cache:
        cache_file_count = disk_cache.get("file_count", 0)
        cache_newest_mtime = disk_cache.get("newest_mtime", 0.0)

        if (
            cache_file_count == current_file_count
            and abs(cache_newest_mtime - current_newest_mtime) < 0.001
        ):
            cache_valid = True

    if cache_valid:
        doc_ids = disk_cache.get("doc_ids", [])
        doc_texts = disk_cache.get("doc_texts", [])

        if doc_ids and doc_texts:
            # Build BM25 index from cached texts (skip full file I/O)
            index = BM25Index(doc_texts, doc_ids)

            _cached_index[cache_key] = {
                "index": index,
                "entry_ids": doc_ids,
                "entries": [],  # Lazy: entries loaded on demand by caller
                "_cache_hit": True,
            }
            return index, doc_ids, []

    entries, entry_ids, texts = _load_entries(kb_root)
    if not texts:
        return None, [], []

    index = BM25Index(texts, entry_ids)

    cache_data = {
        "version": 2,
        "file_count": current_file_count,
        "newest_mtime": current_newest_mtime,
        "doc_ids": entry_ids,
        "doc_texts": texts,
        "built_at": datetime.now().isoformat(),
    }
    _save_cache(kb_root, cache_data)

    _cached_index[cache_key] = {
        "index": index,
        "entry_ids": entry_ids,
        "entries": entries,
    }
    return index, entry_ids, entries


def invalidate_cache(kb_root: Path = None) -> None:
    """
    Invalidate BM25 index cache.

    Args:
        kb_root: Specific root to invalidate, or None to clear all
    """
    if kb_root is None:
        _cached_index.clear()
    else:
        _cached_index.pop(str(kb_root), None)


def rebuild_cache(kb_root: Path) -> None:
    """
    Force rebuild BM25 cache and persist to disk.

    This function should be called after new knowledge is added
    to ensure the persistent cache is updated.

    Args:
        kb_root: Knowledge base root path
    """
    cache_key = str(kb_root)
    _cached_index.pop(cache_key, None)

    cache_file = kb_root / ".bm25_cache.json"
    if cache_file.exists():
        try:
            cache_file.unlink()
        except OSError:
            pass

    entries, entry_ids, texts = _load_entries(kb_root)
    if not texts:
        return

    index = BM25Index(texts, entry_ids)

    current_file_count, current_newest_mtime = _get_file_stats(kb_root)

    cache_data = {
        "version": 2,
        "file_count": current_file_count,
        "newest_mtime": current_newest_mtime,
        "doc_ids": entry_ids,
        "doc_texts": texts,
        "built_at": datetime.now().isoformat(),
    }
    _save_cache(kb_root, cache_data)

    _cached_index[cache_key] = {
        "index": index,
        "entry_ids": entry_ids,
        "entries": entries,
    }


def build_index(kb_root: Path) -> Tuple[Any, List[str], List[Dict[str, Any]]]:
    """
    Build a BM25 index from all knowledge entries.

    Signature matches the old embedding.py for backward compatibility.

    Args:
        kb_root: Knowledge base root path

    Returns:
        (bm25_index, entry_ids, entries)
    """
    return _get_or_build_index(kb_root)


def search(
    query: str,
    kb_root: Path,
    top_k: int = 10,
) -> List[Tuple[str, float]]:
    """
    BM25 search over the knowledge base.

    API-compatible with the old embedding-based search().

    Args:
        query: Search query text
        kb_root: Knowledge base root
        top_k: Number of top results

    Returns:
        List of (entry_id, score) tuples, sorted by score descending.
        Scores are normalized to 0-1 range for backward compatibility.
    """
    index, entry_ids, entries = _get_or_build_index(kb_root)
    if index is None:
        return []

    query_tokens = _bm25_tokenize(query)
    expanded_tokens = expand_with_synonyms(query_tokens, max_expansions=3)

    raw_results = index.search(expanded_tokens, top_k=top_k)
    if not raw_results:
        return []

    # Normalize scores to 0-1 range for backward compatibility with embedding search
    max_score = raw_results[0][1]  # Already sorted descending
    if max_score > 0:
        return [(doc_id, score / max_score) for doc_id, score in raw_results]
    return raw_results
