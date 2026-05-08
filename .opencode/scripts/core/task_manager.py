#!/usr/bin/env python3
"""
Task Manager - State machine for task lifecycle management.

Enforces valid state transitions and prevents invalid operations.
Solves P0 issue: "文档即代码的脆弱性" by making state transitions enforceable in code.
"""

import re
import subprocess
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional

from .file_utils import atomic_read_json, atomic_write_json


# Valid state transitions (current_state -> [allowed_next_states])
VALID_TRANSITIONS = {
    "pending":        ["in_progress"],
    "in_progress":    ["review_pending", "blocked"],
    "review_pending": ["completed", "rejected"],
    "rejected":       ["in_progress"],
    "blocked":        ["pending"],
}


def get_project_root() -> Path:
    """
    Get project root directory using git.
    
    Returns:
        Path to project root directory
        
    Raises:
        RuntimeError: If not in a git repository
    """
    try:
        result = subprocess.run(
            ['git', 'rev-parse', '--show-toplevel'],
            capture_output=True,
            text=True,
            check=True
        )
        return Path(result.stdout.strip())
    except subprocess.CalledProcessError:
        raise RuntimeError("Not in a git repository")


def load_feature_list(project_root: Path) -> Dict[str, Any]:
    """
    Load feature_list.json from project root.
    
    Args:
        project_root: Path to project root directory
        
    Returns:
        Feature list data (empty dict if file doesn't exist)
    """
    filepath = project_root / ".opencode" / "feature_list.json"
    data = atomic_read_json(filepath)
    return data if data is not None else {"tasks": []}


def save_feature_list(project_root: Path, data: Dict[str, Any]) -> None:
    """
    Save feature_list.json to project root using atomic write.
    
    Args:
        project_root: Path to project root directory
        data: Feature list data to save
    """
    filepath = project_root / ".opencode" / "feature_list.json"
    atomic_write_json(filepath, data)


def find_task(data: Dict[str, Any], task_id: str) -> Optional[Dict[str, Any]]:
    """
    Find a task by ID in feature list.
    
    Args:
        data: Feature list data
        task_id: Task ID to find
        
    Returns:
        Task dict if found, None otherwise
        
    Raises:
        ValueError: If tasks contains non-dict entries (malformed feature_list.json)
    """
    tasks = data.get("tasks", [])
    for task in tasks:
        if not isinstance(task, dict):
            raise ValueError(
                f"Malformed feature_list.json: tasks[] contains {type(task).__name__} "
                f"instead of dict (value: {task!r}). "
                f"Each task must be a JSON object with 'id', 'name', 'status' fields."
            )
        if task.get("id") == task_id:
            return task
    return None


_REVIEWER_NOTES_MIN_LENGTH = 10
_REVIEWER_NOTES_PATTERN = re.compile(
    r"\[P[0-3]\]|LGTM",
    re.IGNORECASE,
)

_REVIEW_HINT = (
    "You must dispatch @reviewer (Task subagent_type='reviewer') to perform "
    "the review and let it call this transition. "
    "Do NOT call this transition directly from the orchestrator."
)


def _validate_reviewer_notes(notes: Optional[str], task_id: str) -> None:
    """Enforce that reviewer_notes are present and substantive for completed transitions."""
    if not notes or not notes.strip():
        raise ValueError(
            f"Task {task_id}: reviewer_notes is required for review_pending → completed. "
            f"{_REVIEW_HINT}"
        )
    stripped = notes.strip()
    if len(stripped) < _REVIEWER_NOTES_MIN_LENGTH:
        raise ValueError(
            f"Task {task_id}: reviewer_notes too short ({len(stripped)} chars, "
            f"minimum {_REVIEWER_NOTES_MIN_LENGTH}). {_REVIEW_HINT}"
        )
    if not _REVIEWER_NOTES_PATTERN.search(stripped):
        raise ValueError(
            f"Task {task_id}: reviewer_notes must contain a severity marker "
            f"([P0]-[P3]) or 'LGTM'. {_REVIEW_HINT}"
        )


def transition(
    project_root: Path,
    task_id: str,
    to_status: str,
    actor: Optional[str] = None,
    reviewer_notes: Optional[str] = None
) -> Dict[str, Any]:
    """
    Transition a task to a new status with validation.
    
    Args:
        project_root: Path to project root directory
        task_id: Task ID to transition
        to_status: Target status
        actor: Actor performing the transition (required for "completed")
        reviewer_notes: Review comments from reviewer (written atomically with status change)
        
    Returns:
        Updated task dict
        
    Raises:
        ValueError: If transition is invalid or task not found
    """
    # Load current data
    data = load_feature_list(project_root)
    
    # Find task
    task = find_task(data, task_id)
    if task is None:
        raise ValueError(f"Task not found: {task_id}")
    
    # Get current status
    current_status = task.get("status", "pending")
    
    # Idempotent: same-state transition is a no-op
    if current_status == to_status:
        return task
    
    # Validate transition
    if current_status not in VALID_TRANSITIONS:
        raise ValueError(f"Invalid current status: {current_status}")
    
    allowed_transitions = VALID_TRANSITIONS[current_status]
    if to_status not in allowed_transitions:
        raise ValueError(
            f"Invalid transition: {current_status} → {to_status}. "
            f"Allowed: {allowed_transitions}"
        )
    
    # Special rule: completed only allowed by reviewer
    if to_status == "completed" and actor != "reviewer":
        raise ValueError(
            "Only reviewer can mark task as completed. "
            "Use 'review_pending' status after completing work."
        )
    
    # Gate: review_pending → completed requires substantive reviewer_notes.
    # This prevents orchestrators from bulk-completing tasks without an
    # actual reviewer subagent pass (e.g. under context-window pressure).
    if current_status == "review_pending" and to_status == "completed":
        _validate_reviewer_notes(reviewer_notes, task_id)
    
    # Update task
    task["status"] = to_status
    task["updated_at"] = datetime.now().isoformat()
    
    if to_status == "completed":
        task["completed_at"] = datetime.now().isoformat()
    
    # Write reviewer_notes atomically with status change
    if reviewer_notes is not None:
        task["reviewer_notes"] = reviewer_notes
    
    # Append audit log entry
    if "audit_log" not in task:
        task["audit_log"] = []
    task["audit_log"].append({
        "from": current_status,
        "to": to_status,
        "actor": actor,
        "timestamp": datetime.now().isoformat(),
    })
    
    # Save updated data
    save_feature_list(project_root, data)
    
    return task


def init_feature_list(project_root: Path, project_name: str) -> Dict[str, Any]:
    """
    Initialize an empty feature_list.json.
    
    Args:
        project_root: Path to project root directory
        project_name: Name of the project
        
    Returns:
        Initialized feature list data
    """
    now = datetime.now().isoformat()
    data = {
        "project": project_name,
        "created_at": now,
        "tasks": []
    }
    save_feature_list(project_root, data)
    return data


def create_task(
    project_root: Path,
    name: str,
    description: str = "",
    priority: str = "medium",
    depends_on: Optional[List[str]] = None
) -> Dict[str, Any]:
    """
    Create a new task in the feature list.
    
    Args:
        project_root: Path to project root directory
        name: Task name
        description: Task description (optional)
        priority: Task priority (low/medium/high)
        depends_on: List of task IDs this task depends on
        
    Returns:
        Created task dict
    """
    # Load or initialize feature list
    data = load_feature_list(project_root)
    if not data.get("tasks"):
        data["tasks"] = []
    
    # Generate task ID (task-NNN format)
    existing_ids = [
        t.get("id", "") for t in data.get("tasks", [])
        if t.get("id", "").startswith("task-")
    ]
    max_num = 0
    for task_id in existing_ids:
        try:
            num = int(task_id.split("-")[1])
            max_num = max(max_num, num)
        except (IndexError, ValueError):
            pass
    
    new_id = f"task-{max_num + 1:03d}"
    
    # Create task
    now = datetime.now().isoformat()
    task = {
        "id": new_id,
        "name": name,
        "description": description,
        "status": "pending",
        "priority": priority,
        "depends_on": depends_on or [],
        "audit_log": [],
        "created_at": now,
        "updated_at": now
    }
    
    # Add to list and save
    data["tasks"].append(task)
    save_feature_list(project_root, data)
    
    return task


def get_status_summary(project_root: Path) -> Dict[str, Any]:
    """
    Get summary statistics of task statuses.
    
    Args:
        project_root: Path to project root directory
        
    Returns:
        Dict with status counts and current task info:
        {
            "total": N,
            "pending": N,
            "in_progress": N,
            "review_pending": N,
            "completed": N,
            "rejected": N,
            "blocked": N,
            "has_active_session": bool,
            "has_pending": bool,
            "has_rejected": bool,
            "current_task": "task-002" or None
        }
    """
    feature_list_path = project_root / ".opencode" / "feature_list.json"
    has_active_session = feature_list_path.exists()
    
    data = load_feature_list(project_root) if has_active_session else {"tasks": []}
    tasks = data.get("tasks", [])
    
    # Initialize counts
    summary = {
        "total": len(tasks),
        "pending": 0,
        "in_progress": 0,
        "review_pending": 0,
        "completed": 0,
        "rejected": 0,
        "blocked": 0,
        "has_active_session": has_active_session,
        "has_pending": False,
        "has_rejected": False,
        "current": None,
        "current_task": None
    }
    
    # Count by status (skip malformed entries gracefully)
    current_task = None
    for task in tasks:
        if not isinstance(task, dict):
            continue
        status = task.get("status", "pending")
        if status in ["pending", "in_progress", "review_pending", "completed", "rejected", "blocked"]:
            summary[status] += 1
        
        # Track first in_progress task as "current"
        if status == "in_progress" and current_task is None:
            current_task = task
    
    # Set boolean flags
    summary["has_pending"] = summary["pending"] > 0 or summary["in_progress"] > 0 or summary["review_pending"] > 0
    summary["has_rejected"] = summary["rejected"] > 0
    
    if current_task:
        summary["current"] = f"{current_task.get('id', 'unknown')} (in_progress)"
        summary["current_task"] = current_task.get('id')
    
    return summary


_STALE_SESSION_FILES = ["feature_list.json", "progress.txt"]


def cleanup_stale_session(
    project_root: Path,
    force: bool = False,
    actor: Optional[str] = None,
) -> Dict[str, Any]:
    """
    Reset session files when no incomplete tasks remain.

    Removes feature_list.json and progress.txt so that a fresh task session
    starts with a clean context.  .knowledge-context.md is preserved across
    sessions as the persistent project knowledge file.  Valuable experiences
    should already have been extracted by @evolver before this is called.

    Args:
        project_root: Path to project root directory
        force: If True, remove session files even when incomplete tasks remain
               (treats session as abandoned). Use when reviewer passed verbally
               but did not update task status via CLI.
               Requires actor="orchestrator".
        actor: Caller identity. "orchestrator" is required when force=True.

    Returns:
        {"cleaned": True/False, "removed": [...], "reason": "..."}
    """
    # Guard: --force is an orchestrator-only privilege.
    # Coders must not use it to skip the reviewer gate.
    if force and actor != "orchestrator":
        return {
            "cleaned": False,
            "removed": [],
            "reason": (
                "--force cleanup requires --actor orchestrator. "
                "Only the orchestrator may abandon an active session. "
                "Coders must transition tasks to review_pending and wait for @reviewer."
            ),
        }

    opencode_dir = project_root / ".opencode"
    feature_list_path = opencode_dir / "feature_list.json"

    if not feature_list_path.exists():
        return {"cleaned": False, "removed": [], "reason": "no session file"}

    data = load_feature_list(project_root)
    tasks = data.get("tasks", [])

    incomplete = [
        t for t in tasks
        if isinstance(t, dict) and t.get("status") in ("pending", "in_progress", "review_pending", "rejected", "blocked")
    ]
    if incomplete and not force:
        return {
            "cleaned": False,
            "removed": [],
            "reason": (
                f"{len(incomplete)} incomplete tasks remain "
                f"(statuses: {', '.join(t.get('status','?') for t in incomplete)}). "
                f"Use --force --actor orchestrator to abandon this session."
            ),
        }

    removed: List[str] = []
    for name in _STALE_SESSION_FILES:
        p = opencode_dir / name
        if p.exists():
            p.unlink()
            removed.append(name)

    reason = "all tasks completed" if not incomplete else f"forced cleanup ({len(incomplete)} tasks abandoned)"
    return {"cleaned": True, "removed": removed, "reason": reason}
