#!/usr/bin/env python3
"""
Evolution Mode Toggle Script

Controls the persistent evolution mode state for a session.
The marker file is created in the CURRENT WORKING DIRECTORY,
allowing multiple projects to have independent evolution mode states.
"""

import argparse
import os
import shutil
import subprocess
import sys
from pathlib import Path


def get_workspace_root() -> Path:
    """
    Get the current working directory as workspace root.
    
    This ensures each project directory has its own evolution mode state,
    supporting parallel development across multiple projects.
    
    Returns:
        Path: The current working directory
    """
    try:
        result = subprocess.run(
            ['git', 'rev-parse', '--show-toplevel'],
            capture_output=True, text=True, check=True
        )
        return Path(result.stdout.strip())
    except subprocess.CalledProcessError:
        return Path.cwd()


def get_mode_marker_path() -> Path:
    """
    Get the path to the evolution mode marker file.
    
    The marker file is always created in the current working directory's
    .opencode subdirectory, ensuring project-level isolation.
    
    Returns:
        Path: Path to .evolution_mode_active file
    """
    root = get_workspace_root()
    return root / '.opencode' / '.evolution_mode_active'


def check_write_permission(path: Path) -> bool:
    """
    Check if we have write permission for the given path.
    
    Args:
        path: The path to check
        
    Returns:
        bool: True if writable, False otherwise
    """
    # Check the path itself or its parent
    check_path = path if path.exists() else path.parent
    if not check_path.exists():
        # Check parent of parent
        check_path = path.parent.parent
        if not check_path.exists():
            check_path = Path.cwd()
    
    return os.access(check_path, os.W_OK)


def run_with_sudo(command: list[str]) -> tuple[bool, str]:
    """
    Run a command with sudo after user confirmation.
    
    Args:
        command: The command to run
        
    Returns:
        tuple: (success, message)
    """
    try:
        # Ask for user confirmation
        print(f"需要管理员权限来写入文件")
        response = input("是否使用 sudo 继续? [y/N]: ").strip().lower()
        
        if response not in ('y', 'yes'):
            return False, "用户取消操作"
        
        # Run with sudo
        result = subprocess.run(
            ['sudo'] + command,
            capture_output=True,
            text=True
        )
        
        if result.returncode == 0:
            return True, "操作成功"
        else:
            return False, f"命令执行失败: {result.stderr}"
            
    except KeyboardInterrupt:
        return False, "用户取消操作"
    except Exception as e:
        return False, f"执行出错: {e}"


def get_local_scripts_dir() -> Path:
    """返回项目本地脚本目录 $PROJECT_ROOT/.opencode/scripts/"""
    return get_workspace_root() / '.opencode' / 'scripts'


def get_local_run_py() -> Path:
    """返回项目本地 run.py 路径"""
    return get_local_scripts_dir() / 'run.py'


def get_source_scripts_dir() -> Path:
    """返回 skill 安装目录中的 scripts/ 目录"""
    return Path(__file__).parent.parent


def get_source_skill_root() -> Path:
    """返回 skill 根目录（evolving-agent/），即 scripts/ 的上级目录"""
    return get_source_scripts_dir().parent


def _find_venv_python() -> str:
    """
    探测已安装的 venv python 路径，按平台优先级依次查找。
    找不到时返回空字符串（运行时 fallback 到 sys.executable）。
    """
    home = Path.home()
    candidates = [
        home / '.agents'   / 'skills' / 'evolving-agent' / '.venv' / 'bin' / 'python',  # Cursor
        home / '.config'   / 'opencode' / 'skills' / 'evolving-agent' / '.venv' / 'bin' / 'python',  # OpenCode
        home / '.claude'   / 'skills' / 'evolving-agent' / '.venv' / 'bin' / 'python',  # Claude Code
        home / '.openclaw' / 'skills' / 'evolving-agent' / '.venv' / 'bin' / 'python',  # OpenClaw
    ]
    for p in candidates:
        if p.exists() and p.is_file():
            return str(p)
    return ''


def _get_skill_version() -> str:
    """
    读取 scripts/VERSION 文件获取 skill 版本（git commit hash 短格式）。

    VERSION 由源码仓库的 post-commit hook 在每次提交后自动写入，
    并随安装脚本一起拷贝到目标目录，安装后无需 git 环境即可读取。
    获取失败时返回空字符串（触发强制拷贝）。
    """
    version_file = get_source_scripts_dir() / 'VERSION'
    if version_file.exists():
        return version_file.read_text(encoding='utf-8').strip()
    return ''


def _read_local_version(workspace_root: Path) -> str:
    """读取项目本地已拷贝的版本号，不存在时返回空字符串。"""
    version_file = workspace_root / '.opencode' / '.scripts_version'
    if version_file.exists():
        return version_file.read_text(encoding='utf-8').strip()
    return ''


def copy_scripts_to_project() -> str:
    """
    将 scripts/ 目录拷贝到 $PROJECT_ROOT/.opencode/scripts/。

    使用 git commit hash 做版本检测：
    - 本地 .scripts_version 不存在或版本不一致 → 覆盖拷贝并更新版本文件
    - 版本一致 → 跳过

    Returns:
        str: 操作结果消息
    """
    src = get_source_scripts_dir()
    dst = get_local_scripts_dir()
    workspace_root = get_workspace_root()

    if not src.exists():
        return f"✗ 源脚本目录不存在: {src}"

    try:
        src_version = _get_skill_version()
        local_version = _read_local_version(workspace_root)

        if dst.exists() and src_version and src_version == local_version:
            # 版本一致跳过 scripts 拷贝，但确保运行时配置文件存在
            config_file = workspace_root / '.opencode' / '.agent_config'
            if not config_file.exists():
                venv_python = _find_venv_python()
                knowledge_dir = str(Path.home() / '.config' / 'opencode' / 'knowledge')
                config_lines = []
                if venv_python:
                    config_lines.append(f'VENV_PYTHON={venv_python}')
                config_lines.append(f'KNOWLEDGE_BASE_PATH={knowledge_dir}')
                config_file.write_text('\n'.join(config_lines) + '\n', encoding='utf-8')
            # 补齐 agents/ workflows/ references/（可能因升级或首次拷贝而缺失）
            skill_root = get_source_skill_root()
            for folder in ('agents', 'workflows', 'references'):
                src_dir = skill_root / folder
                dst_dir = workspace_root / '.opencode' / folder
                if not src_dir.exists() or dst_dir.exists():
                    continue
                dst_dir.mkdir(parents=True, exist_ok=True)
                for item in src_dir.rglob('*'):
                    if not item.is_file():
                        continue
                    rel = item.relative_to(src_dir)
                    target = dst_dir / rel
                    target.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(item, target)
            return (
                f"✓ 本地脚本已是最新版本，无需更新 ({src_version})\n"
                f"  本地路径: {dst / 'run.py'}"
            )

        # 版本不一致或首次拷贝，清空旧版本后重新拷贝
        if dst.exists():
            shutil.rmtree(dst)
        dst.mkdir(parents=True, exist_ok=True)

        _skip_patterns = {'__pycache__', '.pyc', '.pyo', '.venv'}

        copied = 0
        for item in src.rglob('*'):
            if not item.is_file():
                continue
            if any(p in item.parts for p in _skip_patterns) or item.suffix in ('.pyc', '.pyo'):
                continue
            rel = item.relative_to(src)
            target = dst / rel
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(item, target)
            if target.suffix == '.py':
                target.chmod(target.stat().st_mode | 0o111)
            copied += 1

        # 写入版本文件
        version_file = workspace_root / '.opencode' / '.scripts_version'
        version_file.write_text(src_version + '\n', encoding='utf-8')

        # 写入路径标记，供 LLM 读取
        marker = workspace_root / '.opencode' / '.run_py_path'
        marker.write_text(str(dst / 'run.py') + '\n', encoding='utf-8')

        # 写入运行时配置：venv python 路径、知识库路径
        # 本地脚本启动时读取，避免运行时再去探测主目录触发 IDE 授权
        config_file = workspace_root / '.opencode' / '.agent_config'
        venv_python = _find_venv_python()
        knowledge_dir = str(Path.home() / '.config' / 'opencode' / 'knowledge')
        config_lines = []
        if venv_python:
            config_lines.append(f'VENV_PYTHON={venv_python}')
        config_lines.append(f'KNOWLEDGE_BASE_PATH={knowledge_dir}')
        config_file.write_text('\n'.join(config_lines) + '\n', encoding='utf-8')

        # 同步 agents/ workflows/ references/ 到 .opencode/
        skill_root = get_source_skill_root()
        for folder in ('agents', 'workflows', 'references'):
            src_dir = skill_root / folder
            dst_dir = workspace_root / '.opencode' / folder
            if not src_dir.exists():
                continue
            if dst_dir.exists():
                shutil.rmtree(dst_dir)
            dst_dir.mkdir(parents=True, exist_ok=True)
            for item in src_dir.rglob('*'):
                if not item.is_file():
                    continue
                rel = item.relative_to(src_dir)
                target = dst_dir / rel
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(item, target)
                copied += 1

        action = "已更新" if local_version else "已拷贝"
        version_info = f" ({local_version} → {src_version})" if local_version and local_version != src_version else f" ({src_version})"
        return (
            f"✓ 脚本{action}到项目本地{version_info}，共 {copied} 个文件\n"
            f"  本地路径: {dst / 'run.py'}"
        )
    except Exception as e:
        return f"✗ 拷贝脚本失败: {e}"


def enable_mode() -> str:
    """
    Enable evolution mode by creating the marker file.
    Handles permission issues with sudo if needed.
    
    Returns:
        str: Success message
    """
    marker_path = get_mode_marker_path()
    parent_dir = marker_path.parent
    
    # Try to create directory and file normally first
    try:
        parent_dir.mkdir(parents=True, exist_ok=True)
        marker_path.touch()
        return f"✓ Evolution Mode ENABLED for this session\n  Marker: {marker_path}"
    except PermissionError:
        pass
    
    # Permission denied - try with sudo
    print(f"无法写入 {marker_path}")
    
    # Create directory with sudo if needed
    if not parent_dir.exists():
        success, msg = run_with_sudo(['mkdir', '-p', str(parent_dir)])
        if not success:
            return f"✗ 无法创建目录: {msg}"
    
    # Create marker file with sudo
    success, msg = run_with_sudo(['touch', str(marker_path)])
    if success:
        return f"✓ Evolution Mode ENABLED for this session (with sudo)\n  Marker: {marker_path}"
    else:
        return f"✗ 无法启用进化模式: {msg}"


def disable_mode() -> str:
    """
    Disable evolution mode by removing the marker file.
    Handles permission issues with sudo if needed.
    
    Returns:
        str: Success message
    """
    marker_path = get_mode_marker_path()
    
    if not marker_path.exists():
        return "ℹ Evolution Mode was not active"
    
    # Try to remove normally first
    try:
        marker_path.unlink()
        return f"✓ Evolution Mode DISABLED for this session"
    except PermissionError:
        pass
    
    # Permission denied - try with sudo
    print(f"无法删除 {marker_path}")
    success, msg = run_with_sudo(['rm', '-f', str(marker_path)])
    
    if success:
        return f"✓ Evolution Mode DISABLED for this session (with sudo)"
    else:
        return f"✗ 无法禁用进化模式: {msg}"


def is_mode_active() -> bool:
    """
    Check if evolution mode is currently active.
    
    Returns:
        bool: True if mode is active
    """
    return get_mode_marker_path().exists()


def get_status_message() -> str:
    """
    Get the current status message.
    
    Returns:
        str: Status message
    """
    status = "ACTIVE" if is_mode_active() else "INACTIVE"
    marker_path = get_mode_marker_path()
    return f"Evolution Mode Status: {status}\nMarker file: {marker_path}"


def get_context_prompt() -> str:
    """
    Get the context prompt that should be injected into the AI's context.
    
    Returns:
        str: The reinforcement prompt
    """
    if not is_mode_active():
        return ""
    
    return """
## Evolution Mode Active
This session is in EVOLUTION MODE.
- After completing tasks, automatically check for extractable knowledge
- Run trigger detection even without explicit user commands
- Store valuable experiences to the knowledge base
- Only report to user when new knowledge is extracted
""".strip()


def main():
    parser = argparse.ArgumentParser(
        description='Toggle or check evolution mode status',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python toggle_mode.py                    # Show current status
  python toggle_mode.py --init            # Full init: start coordinator + enable mode
  python toggle_mode.py --on              # Enable evolution mode
  python toggle_mode.py --off              # Disable evolution mode
  python toggle_mode.py --on --inject      # Enable and print context prompt
  python toggle_mode.py --status           # Show detailed status
        """
    )

    parser.add_argument('--init', '-i', action='store_true',
                        help='Full initialization: start coordinator and enable evolution mode')
    parser.add_argument('--on', '-e', action='store_true',
                        help='Enable evolution mode')
    parser.add_argument('--off', '-d', action='store_true',
                        help='Disable evolution mode')
    parser.add_argument('--toggle', '-t', action='store_true',
                        help='Toggle current state')
    parser.add_argument('--inject', action='store_true',
                        help='Print context prompt for injection')
    parser.add_argument('--status', '-s', action='store_true',
                        help='Show detailed status')
    
    args = parser.parse_args()

    # Full initialization (manual trigger /evolve)
    if args.init:
        was_active = is_mode_active()

        # 拷贝脚本到项目本地（每次 init 都执行，确保版本同步）
        copy_result = copy_scripts_to_project()

        result = enable_mode()

        # Only show message if this is a fresh activation
        if not was_active:
            print(result)  # Print the enable message
            print(copy_result)
            print("\n" + "="*60)
            print("协调器已启动")
            print("="*60)
            print("\n下一步建议：")
            print("   - 输入编程任务（如：帮我实现一个登录功能）")
            print("   - 或直接开始描述您的需求")
            print("\n提示：")
            print("   - 编程工作流将自动加载")
            print("   - 进化模式已激活，会自动提取有价值经验")
            print("   - 使用 'python toggle_mode.py --off' 可关闭进化模式")
            print("   - 后续命令使用本地脚本路径，无需重复授权")
            print("="*60 + "\n")
        else:
            # 已激活时静默同步脚本
            print(copy_result)
        return 0

    # Inject context prompt (can be combined with other operations)
    if args.inject:
        if args.on or args.off or args.toggle:
            # Combine with state change
            if args.on:
                print(enable_mode())
            elif args.off:
                print(disable_mode())
            elif args.toggle:
                if is_mode_active():
                    print(disable_mode())
                else:
                    print(enable_mode())
            # Then print context if enabled
            if is_mode_active():
                print("\n--- Context Prompt ---")
                print(get_context_prompt())
            else:
                print("\n--- Context Prompt ---")
                print("(No context prompt - evolution mode is inactive)")
        else:
            # Just print context
            if is_mode_active():
                print("--- Context Prompt ---")
                print(get_context_prompt())
            else:
                print("Evolution mode is not active. No context prompt to inject.")
        return 0

    # Enable
    if args.on:
        print(enable_mode())
        return 0
    
    # Disable
    if args.off:
        print(disable_mode())
        return 0
    
    # Toggle
    if args.toggle:
        if is_mode_active():
            print(disable_mode())
        else:
            print(enable_mode())
        return 0
    
    # Status query
    if args.status:
        print(get_status_message())
        return 0
    
    # Default: show status
    print(get_status_message())
    return 0


if __name__ == '__main__':
    sys.exit(main())
