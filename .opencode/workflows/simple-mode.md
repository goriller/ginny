# Simple Mode — @coder 修复指南

你是 @coder，被 orchestrator 调度来修复问题。按此指南完成任务后将状态更新为 `review_pending`，等待 @reviewer 审查。**不要自我审查，不要标记 completed。**

## 环境变量

```bash
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
if [ -f "$PROJECT_ROOT/.opencode/scripts/run.py" ]; then RUN_PY="$PROJECT_ROOT/.opencode/scripts/run.py"; elif [ -d ~/.config/opencode/skills/evolving-agent ]; then RUN_PY=~/.config/opencode/skills/evolving-agent/scripts/run.py; elif [ -d ~/.agents/skills/evolving-agent ]; then RUN_PY=~/.agents/skills/evolving-agent/scripts/run.py; else RUN_PY=~/.claude/skills/evolving-agent/scripts/run.py; fi
if [ ! -f "$RUN_PY" ]; then echo "run.py not found: $RUN_PY"; exit 1; fi
```

---

## 流程

```
1. 加载上下文
   读取 $PROJECT_ROOT/.opencode/.knowledge-context.md（如存在）
   读取 reviewer_notes（如上次被 reject）
   更新任务状态为 in_progress：
     python "$RUN_PY" task transition --task-id "$TASK_ID" --status in_progress

2. 问题分析
   ├─ 阅读相关代码，理解当前实现
   ├─ 分析根因，定位问题源头
   └─ 制定方案，选择最小化修改

3. 执行修复
   ├─ 如有 reviewer_notes，优先针对性修复
   ├─ 最小化修改代码
   ├─ 编写或更新测试
   └─ 运行测试验证通过

4. 完成，更新状态为 review_pending
   python "$RUN_PY" task transition --task-id "$TASK_ID" --status review_pending

   更新 $PROJECT_ROOT/.opencode/progress.txt：
   - 记录"问题根因"
   - 记录"关键发现"
```

---

## 错误处理

```
执行失败 → 分析原因 → 尝试方案（最多 3 次）
├─ 成功 → 继续
└─ 连续失败 3 次 → 标记 blocked，记录详情
    python "$RUN_PY" task transition --task-id "$TASK_ID" --status blocked
```

---

## 状态文件

| 文件 | 用途 |
|------|------|
| `$PROJECT_ROOT/.opencode/feature_list.json` | 任务清单和状态 |
| `$PROJECT_ROOT/.opencode/progress.txt` | 当前任务进度 |
| `$PROJECT_ROOT/.opencode/.knowledge-context.md` | 知识检索结果 |

> `.opencode` 在项目根目录（git 仓库根）。
