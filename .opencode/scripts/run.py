#!/usr/bin/env python3
"""
Evolving Agent - Unified CLI Entry Point

统一的命令行入口，具备环境检测、路径自动解析，根据参数调用不同的脚本。

用法:
    python run.py <module> <action> [options]

模块:
    mode        进化模式控制
    knowledge   知识库操作
    github      GitHub 仓库学习
    project     项目检测和经验管理
    info        显示环境信息
    task        任务管理

示例:
    python run.py mode --status
    python run.py knowledge query --trigger "react,hooks"
    python run.py github fetch https://github.com/user/repo
    python run.py project detect .
    python run.py task create --name "修复X" --priority high
    python run.py task list --status pending
    python run.py info
"""

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, List

# Ensure the scripts directory is on sys.path so sub-modules are importable
# regardless of the working directory from which run.py is invoked.
_SCRIPTS_DIR = Path(__file__).parent
if str(_SCRIPTS_DIR) not in sys.path:
    sys.path.insert(0, str(_SCRIPTS_DIR))


def _load_agent_config() -> None:
    """
    读取 $PROJECT_ROOT/.opencode/.agent_config 并注入到 os.environ。

    配置文件由 `mode --init` 生成，记录 venv python 路径和知识库路径，
    避免运行时探测主目录触发 IDE 授权弹窗。格式为 KEY=VALUE 每行一条。
    此函数在模块级别最早执行，确保后续所有路径解析都能读到正确值。
    """
    try:
        result = subprocess.run(
            ['git', 'rev-parse', '--show-toplevel'],
            capture_output=True, text=True, check=True
        )
        project_root = Path(result.stdout.strip())
    except subprocess.CalledProcessError:
        project_root = Path.cwd()

    config_file = project_root / '.opencode' / '.agent_config'
    if not config_file.exists():
        return

    for line in config_file.read_text(encoding='utf-8').splitlines():
        line = line.strip()
        if not line or line.startswith('#') or '=' not in line:
            continue
        key, _, value = line.partition('=')
        key = key.strip()
        value = value.strip()
        # 只在未被外部显式设置时才注入，保留用户环境变量的覆盖权
        if key and value and key not in os.environ:
            os.environ[key] = value


_load_agent_config()

# Import task_manager for task commands
from core.task_manager import (
    get_project_root,
    create_task,
    load_feature_list,
    transition,
    get_status_summary,
    cleanup_stale_session,
)


__version__ = "2.0.0"


# =============================================================================
# 路径解析
# =============================================================================

def get_scripts_dir() -> Path:
    """
    获取 scripts 目录路径。
    
    自动检测运行模式：
    - 开发模式: run.py 在源码的 scripts/ 目录下
    - 安装模式: run.py 在 ~/.config/opencode/skills/evolving-agent/scripts/ 下
    """
    return Path(__file__).parent


def get_skill_root() -> Path:
    """获取 evolving-agent skill 根目录"""
    return get_scripts_dir().parent


def get_script_path(module: str, script: str) -> Path:
    """
    获取目标脚本的完整路径。
    
    Args:
        module: 模块目录名 (core, knowledge, github, programming)
        script: 脚本文件名 (不含 .py 后缀)
    
    Returns:
        脚本的完整路径
    """
    return get_scripts_dir() / module / f"{script}.py"


def is_development_mode() -> bool:
    """
    检测是否在开发模式下运行。
    
    开发模式: 存在 .git 目录或 README.md 在上级目录
    安装模式: 在 ~/.config/opencode/skills/ 或 ~/.claude/skills/ 下
    """
    skill_root = get_skill_root()
    project_root = skill_root.parent
    home = Path.home()
    
    opencode_skills = home / '.config' / 'opencode' / 'skills'
    claude_skills = home / '.claude' / 'skills'
    
    # 检查是否在 skills 安装目录
    if str(skill_root).startswith(str(opencode_skills)) or \
       str(skill_root).startswith(str(claude_skills)):
        return False
    
    # 检查是否有开发标志
    if (project_root / '.git').exists() or (project_root / 'README.md').exists():
        return True
    
    return False


def detect_platform() -> str:
    """
    检测当前平台。
    
    Returns:
        'opencode' 或 'claude'
    """
    # 检查环境变量
    env_platform = os.environ.get('SKILLS_PLATFORM', '').lower()
    if env_platform in ('opencode', 'claude'):
        return env_platform
    
    # 检查哪个 skills 目录存在 evolving-agent
    home = Path.home()
    opencode_skills = home / '.config' / 'opencode' / 'skills' / 'evolving-agent'
    claude_skills = home / '.claude' / 'skills' / 'evolving-agent'
    
    if opencode_skills.exists():
        return 'opencode'
    if claude_skills.exists():
        return 'claude'
    
    # 默认 opencode
    return 'opencode'


def get_skills_dir() -> Path:
    """获取 skills 安装目录"""
    platform = detect_platform()
    home = Path.home()
    
    if platform == 'claude':
        return home / '.claude' / 'skills'
    return home / '.config' / 'opencode' / 'skills'


def get_knowledge_dir() -> Path:
    """获取知识库数据目录 — 委托给 core.path_resolver（单一权威实现）"""
    try:
        from core.path_resolver import get_knowledge_base_dir
        return get_knowledge_base_dir()
    except ImportError:
        env_path = os.environ.get('KNOWLEDGE_BASE_PATH')
        if env_path:
            return Path(env_path)
        platform = detect_platform()
        home = Path.home()
        if platform == 'claude':
            return home / '.claude' / 'knowledge'
        return home / '.config' / 'opencode' / 'knowledge'


def get_python_executable() -> str:
    """
    获取 Python 解释器路径。

    查找顺序：
    1. 环境变量 VENV_PYTHON（由 .agent_config 注入，mode --init 时探测并写入）
    2. 当前 skill_root 下的 .venv（开发模式，源码仓库）
    3. 各平台安装目录的 .venv（全量扫描，支持 Cursor/OpenClaw）
    4. sys.executable（最终 fallback）
    """
    # 1. 优先读配置文件注入的路径（已在模块加载时写入 os.environ）
    venv_python_env = os.environ.get('VENV_PYTHON', '')
    if venv_python_env and Path(venv_python_env).is_file():
        return venv_python_env

    # 2. 开发模式：skill_root/.venv
    skill_root = get_skill_root()
    local_venv_python = skill_root / '.venv' / 'bin' / 'python'
    if local_venv_python.exists() and local_venv_python.is_file():
        return str(local_venv_python)

    # 3. 全平台扫描安装目录（含 Cursor ~/.agents/ 和 OpenClaw ~/.openclaw/）
    home = Path.home()
    platform_skill_dirs = [
        home / '.agents'   / 'skills' / 'evolving-agent',   # Cursor
        home / '.config'   / 'opencode' / 'skills' / 'evolving-agent',  # OpenCode
        home / '.claude'   / 'skills' / 'evolving-agent',   # Claude Code
        home / '.openclaw' / 'skills' / 'evolving-agent',   # OpenClaw
    ]
    for skill_dir in platform_skill_dirs:
        candidate = skill_dir / '.venv' / 'bin' / 'python'
        if candidate.exists() and candidate.is_file():
            return str(candidate)

    # 4. fallback
    return sys.executable


# =============================================================================
# 环境检测
# =============================================================================

def check_python_version() -> Dict[str, Any]:
    """检查 Python 版本"""
    version_info = sys.version_info
    version_str = f"{version_info.major}.{version_info.minor}.{version_info.micro}"
    is_ok = version_info >= (3, 8)
    
    return {
        "version": version_str,
        "ok": is_ok,
        "message": None if is_ok else "需要 Python 3.8+"
    }


def check_dependencies() -> Dict[str, Any]:
    """检查依赖"""
    deps = {}
    
    # 检查 PyYAML
    try:
        import yaml
        deps["PyYAML"] = {"version": yaml.__version__, "ok": True}
    except ImportError:
        deps["PyYAML"] = {"version": None, "ok": False, "message": "pip install PyYAML"}
    
    return deps


def get_evolution_mode_status() -> str:
    """获取进化模式状态"""
    try:
        result = subprocess.run(
            ['git', 'rev-parse', '--show-toplevel'],
            capture_output=True, text=True, check=True
        )
        project_root = Path(result.stdout.strip())
    except subprocess.CalledProcessError:
        project_root = Path.cwd()
    marker = project_root / '.opencode' / '.evolution_mode_active'
    return "ACTIVE" if marker.exists() else "INACTIVE"


def get_environment_info() -> Dict[str, Any]:
    """获取完整的环境信息"""
    python_info = check_python_version()
    deps = check_dependencies()
    
    return {
        "version": __version__,
        "python": python_info,
        "platform": detect_platform(),
        "is_dev_mode": is_development_mode(),
        "paths": {
            "scripts_dir": str(get_scripts_dir()),
            "skill_root": str(get_skill_root()),
            "skills_dir": str(get_skills_dir()),
            "knowledge_dir": str(get_knowledge_dir()),
            "python_executable": get_python_executable(),
        },
        "dependencies": deps,
        "evolution_mode": get_evolution_mode_status(),
    }


def print_environment_info():
    """打印环境信息"""
    info = get_environment_info()
    
    print("=" * 60)
    print("Evolving Agent Environment")
    print("=" * 60)
    print()
    
    # 版本信息
    print(f"Version:         {info['version']}")
    py = info['python']
    py_status = "✓" if py['ok'] else "✗"
    print(f"Python:          {py['version']} {py_status}")
    if py.get('message'):
        print(f"                 {py['message']}")
    
    # 平台和模式
    print(f"Platform:        {info['platform']}")
    mode_str = "development (源码目录)" if info['is_dev_mode'] else "installed (安装目录)"
    print(f"Mode:            {mode_str}")
    print()
    
    # 路径
    print("Paths:")
    paths = info['paths']
    print(f"  Scripts:       {paths['scripts_dir']}")
    print(f"  Skill Root:    {paths['skill_root']}")
    print(f"  Skills Dir:    {paths['skills_dir']}")
    print(f"  Knowledge:     {paths['knowledge_dir']}")
    print(f"  Python:        {paths['python_executable']}")
    print()
    
    # 依赖
    print("Dependencies:")
    for name, dep in info['dependencies'].items():
        status = "✓" if dep['ok'] else "✗"
        version = dep.get('version') or 'not installed'
        print(f"  {name}:".ljust(15) + f"{version} {status}")
        if dep.get('message'):
            print(f"                 → {dep['message']}")
    print()
    
    # 进化模式
    print(f"Evolution Mode:  {info['evolution_mode']}")
    print()
    print("=" * 60)


# =============================================================================
# 脚本执行
# =============================================================================

def run_script(module: str, script: str, args: List[str]) -> int:
    """
    执行目标脚本。
    
    Args:
        module: 模块目录名
        script: 脚本文件名 (不含 .py)
        args: 传递给脚本的参数列表
    
    Returns:
        脚本的退出码
    """
    script_path = get_script_path(module, script)
    
    if not script_path.exists():
        print(f"Error: Script not found: {script_path}", file=sys.stderr)
        print("Please check your installation.", file=sys.stderr)
        return 1
    
    python_exe = get_python_executable()
    cmd = [python_exe, str(script_path)] + args
    
    # 设置环境变量，确保子进程能找到正确的路径
    env = os.environ.copy()
    env['PYTHONUNBUFFERED'] = '1'  # 确保 Python 输出不缓冲
    
    try:
        result = subprocess.run(cmd, env=env)
        return result.returncode
    except KeyboardInterrupt:
        return 130
    except Exception as e:
        print(f"Error executing script: {e}", file=sys.stderr)
        return 1


# =============================================================================
# 命令处理器
# =============================================================================

def handle_mode(args: argparse.Namespace, remaining: List[str]) -> int:
    """处理 mode 命令"""
    script_args = []
    
    if args.status:
        script_args.append("--status")
    elif args.init:
        script_args.append("--init")
    elif args.on:
        script_args.append("--on")
    elif args.off:
        script_args.append("--off")
    else:
        # 默认显示状态
        script_args.append("--status")
    
    return run_script("core", "toggle_mode", script_args)


def _handle_trigger_inprocess(args: argparse.Namespace, remaining: List[str]) -> int:
    """Handle 'knowledge trigger' in-process (no subprocess overhead).
    
    Eliminates ~300-500ms of Python startup + module reimport overhead
    by calling trigger.py functions directly. The BM25 module-level cache
    is preserved across multiple calls within the same process.
    """
    # trigger.py uses bare imports (e.g. "from query import ...") that resolve
    # when it runs as a standalone script (knowledge/ dir is on sys.path).
    # When imported as knowledge.trigger from the parent package, the knowledge/
    # directory is NOT on sys.path, causing ModuleNotFoundError.  Add it here.
    _knowledge_dir = str(_SCRIPTS_DIR / "knowledge")
    if _knowledge_dir not in sys.path:
        sys.path.insert(0, _knowledge_dir)

    from knowledge.trigger import (
        trigger_knowledge,
        format_for_context,
    )

    user_input = getattr(args, 'input', None)
    project_dir = getattr(args, 'project', None)
    mode = getattr(args, 'mode', 'hybrid') or 'hybrid'
    fmt = getattr(args, 'format', 'json') or 'json'
    limit = getattr(args, 'limit', 5) or 5

    # Parse --trigger from remaining args if present
    explicit_triggers = None
    trigger_val = getattr(args, 'trigger', None)
    if not trigger_val:
        for i, arg in enumerate(remaining):
            if arg == '--trigger' and i + 1 < len(remaining):
                trigger_val = remaining[i + 1]
                break
    if trigger_val:
        explicit_triggers = [t.strip() for t in trigger_val.split(',')]

    if not any([user_input, project_dir, explicit_triggers]):
        print("Error: --input, --project, or --trigger is required", file=sys.stderr)
        return 1

    try:
        result = trigger_knowledge(
            user_input=user_input,
            project_dir=project_dir,
            explicit_triggers=explicit_triggers,
            limit=limit,
            mode=mode,
        )
    except Exception as e:
        print(f"Error during knowledge trigger: {e}", file=sys.stderr)
        return 1

    if fmt == 'json':
        print(json.dumps(result, indent=2, ensure_ascii=False, default=str))
    elif fmt == 'context':
        print(format_for_context(result))
    elif fmt == 'triggers':
        print(','.join(result.get('triggers_used', [])))

    return 0


def handle_knowledge(args: argparse.Namespace, remaining: List[str]) -> int:
    """处理 knowledge 命令"""
    action = args.action
    
    # 'trigger' is handled in-process for performance (no subprocess overhead)
    if action == "trigger":
        return _handle_trigger_inprocess(args, remaining)

    # Scripts mapping — these actions are delegated to sub-scripts
    mapping = {
        "query": ("knowledge", "query"),
        "store": ("knowledge", "store"),
        "summarize": ("knowledge", "summarizer"),
        "migrate": ("knowledge", "migrate_to_project"),
    }
    
    # Parent-consumed args accepted by each delegated sub-script.
    # store.py accepts none of these; blindly re-injecting --format caused
    # "unrecognized arguments: --format json" errors.
    # "project" is forwarded to summarize so evolver can write to the
    # project-local KB ($PROJECT_ROOT/.opencode/knowledge/) via --project.
    _delegatable_args = {
        "query":     ("format",),
        "store":     (),
        "summarize": ("format", "project"),
        "migrate":   ("project",),   # --project is consumed by knowledge_parser; re-inject it
    }

    if action in mapping:
        mod, script = mapping[action]
        delegated_remaining = list(remaining)
        for arg_name in _delegatable_args.get(action, ()):
            flag = f'--{arg_name}'
            val = getattr(args, arg_name, None)
            if val and flag not in delegated_remaining:
                delegated_remaining.extend([flag, str(val)])
        # For migrate: also re-inject boolean flags consumed by knowledge_parser
        if action == "migrate":
            if getattr(args, 'dry_run', False) and '--dry-run' not in delegated_remaining:
                delegated_remaining.append('--dry-run')
        return run_script(mod, script, delegated_remaining)
    
    # Built-in actions
    elif action == "gc":
        from knowledge.lifecycle import gc as gc_func
        threshold = getattr(args, 'threshold', 0.1)
        dry_run = getattr(args, 'dry_run', False)
        
        removed = gc_func(threshold=threshold, dry_run=dry_run)
        
        if dry_run:
            print(f"Would remove {len(removed)} stale entries:")
        else:
            print(f"Removed {len(removed)} stale entries:")
        
        for entry in removed:
            print(f"  - {entry.get('id', 'unknown')}: {entry.get('name', 'unnamed')}")
        
        return 0
    
    elif action == "decay":
        from knowledge.lifecycle import decay_unused
        days = getattr(args, 'days', 90)
        rate = getattr(args, 'rate', 0.1)
        
        affected = decay_unused(days_threshold=days, decay_rate=rate)
        
        print(f"Decayed {len(affected)} entries:")
        for entry in affected:
            print(f"  - {entry.get('id', 'unknown')}: {entry['old_effectiveness']:.2f} → {entry['new_effectiveness']:.2f}")
        
        return 0
    
    elif action == "export":
        from knowledge.knowledge_io import export_all
        output = getattr(args, 'output', None)
        fmt = getattr(args, 'format', 'json')
        if not output:
            print("Error: --output is required for export", file=sys.stderr)
            return 1
        count = export_all(output_path=output, format=fmt)
        print(f"Exported {count} entries to {output}")
        return 0
    
    elif action == "import":
        from knowledge.knowledge_io import import_all
        input_file = getattr(args, 'input', None)
        merge = getattr(args, 'merge', 'skip')
        if not input_file:
            print("Error: --input is required for import", file=sys.stderr)
            return 1
        stats = import_all(input_path=input_file, merge_strategy=merge)
        print(json.dumps(stats, ensure_ascii=False))
        return 0
    
    elif action == "dashboard":
        from knowledge.dashboard import generate_stats, format_dashboard, get_kb_root
        kb_root = get_kb_root()
        stats = generate_stats(kb_root)
        if getattr(args, 'json', False):
            print(json.dumps(stats, indent=2, ensure_ascii=False))
        else:
            print(format_dashboard(stats))
        return 0
    
    print(f"Unknown action: {action}", file=sys.stderr)
    print("Available actions: query, store, summarize, trigger, gc, decay, export, import, dashboard", file=sys.stderr)
    return 1


def handle_github(args: argparse.Namespace, remaining: List[str]) -> int:
    """处理 github 命令"""
    action = args.action
    
    mapping = {
        "fetch": ("github", "fetch_info"),
        "extract": ("github", "extract_patterns"),
        "store": ("github", "store_to_knowledge"),
        "learn": ("github", "learn"),
    }
    
    if action in mapping:
        mod, script = mapping[action]
        return run_script(mod, script, remaining)
    
    print(f"Unknown action: {action}", file=sys.stderr)
    print("Available actions: fetch, extract, store, learn", file=sys.stderr)
    return 1


def handle_project(args: argparse.Namespace, remaining: List[str]) -> int:
    """处理 project 命令"""
    action = args.action
    
    mapping = {
        "detect": ("programming", "detect_project"),
        "store": ("programming", "store_experience"),
        "query": ("programming", "query_experience"),
    }
    
    if action in mapping:
        mod, script = mapping[action]
        return run_script(mod, script, remaining)
    
    print(f"Unknown action: {action}", file=sys.stderr)
    print("Available actions: detect, store, query", file=sys.stderr)
    return 1


def handle_info(args: argparse.Namespace, remaining: List[str]) -> int:
    """处理 info 命令"""
    if args.json:
        info = get_environment_info()
        print(json.dumps(info, indent=2, ensure_ascii=False))
    else:
        print_environment_info()
    return 0


def handle_task(args: argparse.Namespace, remaining: List[str]) -> int:
    """处理 task 命令"""
    project_root = get_project_root()
    action = args.action
    
    if action in ("create", "init"):
        # Parse depends_on if provided
        depends_on = None
        if args.depends:
            depends_on = [d.strip() for d in args.depends.split(",")]
        
        task = create_task(
            project_root=project_root,
            name=args.name,
            description=args.description or "",
            priority=args.priority or "medium",
            depends_on=depends_on
        )
        print(json.dumps(task, indent=2, ensure_ascii=False))
        return 0
    
    elif action == "transition":
        task = transition(
            project_root=project_root,
            task_id=args.task_id,
            to_status=args.status,
            actor=args.actor,
            reviewer_notes=getattr(args, 'reviewer_notes', None)
        )
        print(json.dumps(task, indent=2, ensure_ascii=False))
        return 0
    
    elif action == "list":
        data = load_feature_list(project_root)
        tasks = data.get("tasks", [])
        
        # Filter by status if provided
        if args.status:
            tasks = [t for t in tasks if t.get("status") == args.status]
        
        if args.json:
            print(json.dumps(tasks, indent=2, ensure_ascii=False))
        else:
            if not tasks:
                print("No tasks found")
            else:
                print(f"{'ID':<12} {'Name':<40} {'Status':<15} {'Priority':<8}")
                print("-" * 80)
                for t in tasks:
                    print(f"{t.get('id', 'N/A'):<12} {t.get('name', 'N/A')[:38]:<40} {t.get('status', 'N/A'):<15} {t.get('priority', 'N/A'):<8}")
        return 0
    
    elif action == "status":
        summary = get_status_summary(project_root)
        
        if summary["total"] == 0:
            print("No active task session")
            return 0
        
        if args.json:
            print(json.dumps(summary, indent=2, ensure_ascii=False))
        else:
            parts = [f"{summary['total']} tasks total"]
            if summary["completed"] > 0:
                parts.append(f"{summary['completed']} completed")
            if summary["pending"] > 0:
                parts.append(f"{summary['pending']} pending")
            if summary["in_progress"] > 0:
                parts.append(f"{summary['in_progress']} in_progress")
            if summary["review_pending"] > 0:
                parts.append(f"{summary['review_pending']} review_pending")
            if summary["rejected"] > 0:
                parts.append(f"{summary['rejected']} rejected")
            if summary["blocked"] > 0:
                parts.append(f"{summary['blocked']} blocked")
            
            print(", ".join(parts) + ".")
            if summary["current"]:
                print(f"Current: {summary['current']}")
        return 0
    
    elif action == "cleanup":
        result = cleanup_stale_session(
            project_root,
            force=getattr(args, 'force', False),
            actor=getattr(args, 'actor', None),
        )
        if args.json:
            print(json.dumps(result, ensure_ascii=False))
        else:
            if result["cleaned"]:
                print(f"Cleaned: {', '.join(result['removed'])} ({result['reason']})")
            else:
                print(f"Skip: {result['reason']}")
        return 0
    
    print(f"Unknown action: {action}", file=sys.stderr)
    return 1


# =============================================================================
# 参数解析
# =============================================================================

def create_parser() -> argparse.ArgumentParser:
    """创建参数解析器"""
    parser = argparse.ArgumentParser(
        prog="run.py",
        description="Evolving Agent - 统一命令行入口",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  python run.py mode --status              查看进化模式状态
  python run.py mode --init                初始化进化模式
  python run.py knowledge query --stats    查看知识库统计
  python run.py github fetch <url>         获取 GitHub 仓库信息
  python run.py project detect .           检测当前项目技术栈
  python run.py info                       显示环境信息
        """
    )
    
    parser.add_argument(
        "-v", "--version",
        action="version",
        version=f"%(prog)s {__version__}"
    )
    
    subparsers = parser.add_subparsers(dest="module", help="可用模块")
    
    # -------------------------------------------------------------------------
    # mode 子命令
    # -------------------------------------------------------------------------
    mode_parser = subparsers.add_parser(
        "mode",
        help="进化模式控制",
        description="控制进化模式的开启、关闭和状态查看"
    )
    mode_group = mode_parser.add_mutually_exclusive_group()
    mode_group.add_argument("--status", action="store_true", help="查看状态")
    mode_group.add_argument("--init", action="store_true", help="完整初始化")
    mode_group.add_argument("--on", action="store_true", help="开启进化模式")
    mode_group.add_argument("--off", action="store_true", help="关闭进化模式")
    
    # -------------------------------------------------------------------------
    # knowledge 子命令
    # -------------------------------------------------------------------------
    knowledge_parser = subparsers.add_parser(
        "knowledge",
        help="知识库操作",
        description="知识库的查询、存储、归纳、触发、垃圾回收和衰减"
    )
    knowledge_parser.add_argument(
        "action",
        choices=["query", "store", "summarize", "trigger", "gc", "decay", "export", "import", "dashboard", "migrate"],
        help="操作: query(查询), store(存储), summarize(归纳), trigger(触发), gc(垃圾回收), decay(衰减), export(导出), import(导入), dashboard(仪表板), migrate(迁移到项目级知识库)"
    )
    knowledge_parser.add_argument(
        "--threshold",
        type=float,
        default=0.1,
        help="gc阈值，删除effectiveness低于此值的条目 (默认: 0.1)"
    )
    knowledge_parser.add_argument(
        "--dry-run",
        action="store_true",
        help="gc预览模式，只列出不删除"
    )
    knowledge_parser.add_argument(
        "--days-threshold",
        type=int,
        default=90,
        help="decay天数阈值，未使用超过此天数的条目将被衰减 (默认: 90)"
    )
    knowledge_parser.add_argument(
        "--decay-rate",
        type=float,
        default=0.1,
        help="衰减率 (默认: 0.1)"
    )
    knowledge_parser.add_argument(
        "--output",
        help="导出文件路径 (export)"
    )
    knowledge_parser.add_argument(
        "--format",
        choices=["json", "markdown", "context", "triggers"],
        default="json",
        help="输出格式: json, markdown (export), context (trigger), triggers (trigger)"
    )
    knowledge_parser.add_argument(
        "--input",
        help="导入文件路径 (import)"
    )
    knowledge_parser.add_argument(
        "--merge",
        help="import: 合并策略 skip|overwrite|merge"
    )
    knowledge_parser.add_argument(
        "--project",
        help="项目根目录。指定后，store/query 操作将使用 $PROJECT_ROOT/.opencode/knowledge/ 项目级知识库"
    )
    knowledge_parser.add_argument(
        "--json",
        action="store_true",
        help="以 JSON 格式输出 (dashboard)"
    )
    knowledge_parser.add_argument(
        "--mode",
        choices=["keyword", "semantic", "hybrid"],
        default="hybrid",
        help="搜索模式: keyword(关键词), semantic(语义), hybrid(混合, 默认)"
    )
    
    # -------------------------------------------------------------------------
    # github 子命令
    # -------------------------------------------------------------------------
    github_parser = subparsers.add_parser(
        "github",
        help="GitHub 仓库学习",
        description="从 GitHub 仓库提取和存储知识"
    )
    github_parser.add_argument(
        "action",
        choices=["fetch", "extract", "store", "learn"],
        help="操作: fetch(获取信息), extract(提取模式), store(存储知识), learn(一键学习)"
    )
    
    # -------------------------------------------------------------------------
    # project 子命令
    # -------------------------------------------------------------------------
    project_parser = subparsers.add_parser(
        "project",
        help="项目检测和经验管理",
        description="检测项目技术栈，管理项目经验"
    )
    project_parser.add_argument(
        "action",
        choices=["detect", "store", "query"],
        help="操作: detect(检测技术栈), store(存储经验), query(查询经验)"
    )
    
    # -------------------------------------------------------------------------
    # info 子命令
    # -------------------------------------------------------------------------
    info_parser = subparsers.add_parser(
        "info",
        help="显示环境信息",
        description="显示运行环境、路径和依赖信息"
    )
    info_parser.add_argument(
        "--json",
        action="store_true",
        help="以 JSON 格式输出"
    )
    
    # -------------------------------------------------------------------------
    # task 子命令
    # -------------------------------------------------------------------------
    task_parser = subparsers.add_parser(
        "task",
        help="任务管理",
        description="创建、列出和转换任务状态"
    )
    task_parser.add_argument(
        "action",
        choices=["create", "init", "list", "transition", "status", "cleanup"],
        help="操作: create/init(创建), list(列表), transition(状态转换), status(统计), cleanup(清理已完成会话)"
    )
    task_parser.add_argument(
        "--name",
        help="任务名称 (create)"
    )
    task_parser.add_argument(
        "--description",
        help="任务描述 (create)"
    )
    task_parser.add_argument(
        "--priority",
        choices=["low", "medium", "high"],
        help="任务优先级 (create)"
    )
    task_parser.add_argument(
        "--depends",
        help="依赖的任务ID，逗号分隔 (create)"
    )
    task_parser.add_argument(
        "--task-id",
        help="任务ID (transition)"
    )
    task_parser.add_argument(
        "--status",
        help="目标状态 (transition/list)"
    )
    task_parser.add_argument(
        "--actor",
        help="执行者 (transition)"
    )
    task_parser.add_argument(
        "--reviewer-notes",
        dest="reviewer_notes",
        help="审查备注，与状态转换原子写入 (transition)"
    )
    task_parser.add_argument(
        "--json",
        action="store_true",
        help="以 JSON 格式输出 (list/status)"
    )
    task_parser.add_argument(
        "--force",
        action="store_true",
        help="强制清理，即使有未完成任务（cleanup 专用，用于废弃旧会话）"
    )
    
    return parser


# =============================================================================
# 主入口
# =============================================================================

def main() -> int:
    """主入口函数"""
    parser = create_parser()
    
    # 使用 parse_known_args 来捕获剩余参数传递给子脚本
    args, remaining = parser.parse_known_args()
    
    if args.module is None:
        parser.print_help()
        return 0
    
    # 分发到对应的处理器
    handlers = {
        "mode": handle_mode,
        "knowledge": handle_knowledge,
        "github": handle_github,
        "project": handle_project,
        "info": handle_info,
        "task": handle_task,
    }
    
    handler = handlers.get(args.module)
    if handler:
        return handler(args, remaining)
    else:
        print(f"Unknown module: {args.module}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
