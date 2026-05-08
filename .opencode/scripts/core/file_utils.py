#!/usr/bin/env python3
"""
File Utilities - Atomic file operations for JSON data.

Provides atomic read/write operations to prevent data corruption
during concurrent access or system crashes.
"""

import json
import os
import tempfile
from pathlib import Path
from typing import Any, Dict, Optional


def atomic_write_json(filepath: Path | str, data: Dict[str, Any]) -> None:
    """
    Atomically write JSON data to a file.
    
    Uses tempfile + os.replace() for atomic operation (POSIX guarantee).
    Prevents partial writes and data corruption.
    
    Args:
        filepath: Target file path (Path or str)
        data: Dictionary to write as JSON
        
    Raises:
        OSError: If file operations fail
        TypeError: If data is not JSON-serializable
    """
    filepath = Path(filepath)
    
    # Create parent directories if needed
    filepath.parent.mkdir(parents=True, exist_ok=True)
    
    # Write to temp file in same directory (same filesystem for atomic rename)
    fd, temp_path = tempfile.mkstemp(
        dir=filepath.parent,
        prefix=f".{filepath.name}.",
        suffix=".tmp"
    )
    
    try:
        # Write data to temp file
        with os.fdopen(fd, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
            f.flush()
            os.fsync(f.fileno())
        
        # Atomic replace (POSIX guarantee)
        os.replace(temp_path, filepath)
        
    except Exception:
        # Clean up temp file on error
        try:
            os.unlink(temp_path)
        except OSError:
            pass
        raise


def atomic_read_json(filepath: Path | str) -> Optional[Dict[str, Any]]:
    """
    Atomically read JSON data from a file.
    
    Args:
        filepath: Target file path (Path or str)
        
    Returns:
        Dictionary with JSON data, or None if file doesn't exist
        
    Raises:
        json.JSONDecodeError: If file contains invalid JSON
        OSError: If file read fails (other than file not found)
    """
    filepath = Path(filepath)
    
    if not filepath.exists():
        return None
    
    with open(filepath, 'r', encoding='utf-8') as f:
        return json.load(f)
