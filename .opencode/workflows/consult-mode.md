# Consult Mode — 咨询工作流

轻量问答流程。不需要 @reviewer，orchestrator 直接执行本工作流。

## 环境变量

```bash
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
if [ -f "$PROJECT_ROOT/.opencode/scripts/run.py" ]; then RUN_PY="$PROJECT_ROOT/.opencode/scripts/run.py"; elif [ -d ~/.config/opencode/skills/evolving-agent ]; then RUN_PY=~/.config/opencode/skills/evolving-agent/scripts/run.py; elif [ -d ~/.agents/skills/evolving-agent ]; then RUN_PY=~/.agents/skills/evolving-agent/scripts/run.py; else RUN_PY=~/.claude/skills/evolving-agent/scripts/run.py; fi
```

---

## Checklist

```
TodoWrite:
- [ ] 知识检索
- [ ] 分析回答
- [ ] 知识归纳判断
```

---

## 流程

```
步骤 1: 知识检索
  python $RUN_PY knowledge trigger \
    --input "用户问题" --format context
  ├─ 有匹配 → 优先基于历史经验回答，附上经验来源
  └─ 无匹配 → 结合代码分析回答

步骤 2: 分析回答
  ├─ 阅读相关代码（如问题涉及项目代码）
  ├─ 判断是否需要修改代码：
  │   ├─ 需要改代码 → 停止本流程，更新 TodoWrite 为编程意图 checklist，
  │   │               切换到 SKILL.md 步骤 3 编程调度闭环（simple-mode）
  │   └─ 仅需解释 → 继续
  └─ 综合知识库经验 + 代码分析，给出回答

步骤 3: 知识归纳判断
  回答完成后，评估是否值得保存：
  ├─ 保存：排查出的隐蔽根因、环境/配置方案、可复用模式、用户说"记住"
  └─ 不保存：通用常识、一次性查询

  如需保存（每条经验单独存储）：
  echo "问题：{描述} → 解决：{方案}" | \
    python $RUN_PY knowledge summarize --auto-store
```
