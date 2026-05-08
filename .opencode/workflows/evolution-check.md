# Evolution Check — 知识归纳

在编码/修复循环结束后执行。由 SKILL.md 步骤 3.5 调度 @evolver 执行。

## 环境变量

```bash
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
if [ -f "$PROJECT_ROOT/.opencode/scripts/run.py" ]; then RUN_PY="$PROJECT_ROOT/.opencode/scripts/run.py"; elif [ -d ~/.config/opencode/skills/evolving-agent ]; then RUN_PY=~/.config/opencode/skills/evolving-agent/scripts/run.py; elif [ -d ~/.agents/skills/evolving-agent ]; then RUN_PY=~/.agents/skills/evolving-agent/scripts/run.py; else RUN_PY=~/.claude/skills/evolving-agent/scripts/run.py; fi
```

---

## 执行条件

检查 `.opencode/.evolution_mode_active`：
- 存在 → 执行下方流程
- 不存在 → 跳过

即使进化模式激活，以下情况可跳过：
- 简单修改一行代码，reviewer 无发现
- 用户说"很好"/"ok"，无新知识产出

以下情况必须执行：
- reviewer reject 后修复成功（高价值）
- 发现隐蔽的 bug 根因
- 环境特定的 workaround
- 用户说"记住这个"

---

## 流程

```
步骤 1: 提取经验来源
  - 读取 $PROJECT_ROOT/.opencode/progress.txt（"遇到的问题"、"关键决策"）
  - 读取 $PROJECT_ROOT/.opencode/feature_list.json（所有 reviewer_notes）
  - 回顾本次会话中的架构决策

步骤 2: 知识归纳
  调用 evolver 执行（平台差异见 $SKILLS_DIR/evolving-agent/references/platform.md）

  存储格式——每条经验单独一个命令：
  echo "问题：xxx → 解决：yyy" | \
    python $RUN_PY knowledge summarize --auto-store
```

> 不要用 heredoc 批量存储，会返回空结果。一条命令存一条经验。

---

## 格式规范

```
问题：<问题描述> → 解决：<解决方案>
决策：<选择了什么> → 原因：<为什么>
教训：<什么情况下> → 避免：<不要做什么>
```

---

## 示例

```bash
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
RUN_PY="$PROJECT_ROOT/.opencode/scripts/run.py"

echo "问题：Vite项目跨域报错 → 解决：配置 server.proxy" | \
  python $RUN_PY knowledge summarize --auto-store

echo "教训：SQL 拼接字符串存在注入风险 → 避免：始终使用参数化查询" | \
  python $RUN_PY knowledge summarize --auto-store
```
