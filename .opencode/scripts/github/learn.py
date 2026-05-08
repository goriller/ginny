#!/usr/bin/env python3
"""
github learn — 一键学习 GitHub 仓库

自动串联 fetch → extract → store 三步，使用内部函数调用传递数据。
"""
import argparse
import json
import sys
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))

from extract_patterns import (
    detect_architecture_patterns,
    detect_tech_stack,
    detect_conventions,
    extract_best_practices,
    store_to_knowledge_base,
)
from fetch_info import get_repo_info


def main():
    parser = argparse.ArgumentParser(description='一键学习 GitHub 仓库')
    parser.add_argument('url', help='GitHub 仓库 URL')
    parser.add_argument('--knowledge-dir', help='知识库目录（可选）')
    parser.add_argument('--dry-run', action='store_true', help='仅 fetch+extract，不存储')
    args = parser.parse_args()
    
    print(f"[1/3] 获取仓库信息: {args.url}", file=sys.stderr)
    repo_info = get_repo_info(args.url)
    
    print(f"[2/3] 提取知识模式...", file=sys.stderr)
    extracted = {
        'name': repo_info.get('name', 'unknown'),
        'url': repo_info.get('url', ''),
        'hash': repo_info.get('latest_hash', ''),
        'extracted_at': datetime.now().isoformat(),
        'architecture_patterns': detect_architecture_patterns(repo_info.get('file_tree', [])),
        'tech_stack': detect_tech_stack(repo_info.get('readme', '')),
        'conventions': detect_conventions(repo_info.get('readme', '')),
        'practices': extract_best_practices(repo_info.get('readme', '')),
    }
    
    if args.dry_run:
        print("[3/3] Dry-run 模式，跳过存储", file=sys.stderr)
        print(json.dumps(extracted, indent=2, ensure_ascii=False))
    else:
        print(f"[3/3] 存储到知识库...", file=sys.stderr)
        store_to_knowledge_base(extracted, args.knowledge_dir)


if __name__ == '__main__':
    main()
