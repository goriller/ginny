# 平台差异 — 唯一定义

其他文件通过"见 `$SKILLS_DIR/evolving-agent/references/platform.md`"引用本文件，不各自重复。

---

## 安装路径

| 平台 | Skills 目录 | 安装命令 |
|------|------------|---------|
| **OpenCode** | `~/.config/opencode/skills/` | `./install.sh --opencode` |
| **Claude Code** | `~/.claude/skills/` | `./install.sh --claude-code` |
| **Cursor** | `~/.agents/skills/` | `./install.sh --cursor` |
| **OpenClaw** | `~/.openclaw/skills/` | `./install.sh --openclaw` |

> Cursor 使用独立的 `~/.agents/skills/` 目录（agent skills 系统），与 Claude Code 的 `~/.claude/skills/` 相互独立，需单独安装。
> OpenClaw 使用 `~/.openclaw/skills/` 作为共享技能目录，所有 agent 实例共享同一个技能集。

---

## Agent 调度方式

| 平台 | 调度语法 | 说明 |
|------|---------|------|
| **OpenCode** | `@agent_name` | 原生多 agent，可指定模型，独立上下文 |
| **Claude Code** | `Task(subagent_type, prompt)` | Task tool spawn 匿名 subagent，继承 parent 模型 |
| **Cursor** | `Task(subagent_type, prompt)` | 同 Claude Code，通过 Task tool 调度子 agent |
| **OpenClaw** | `sessions_spawn(agent_id, task)` | 通过 skill 调用 spawn subagent，独立会话上下文 |

四者语义一致：subagent 有独立上下文窗口，完成后返回结果给 parent。
SKILL.md 中标注 `[Claude Code/Cursor]` 的语法适用于 Claude Code 和 Cursor 两个平台。
OpenClaw 通过 skill 文件中的 `sessions_spawn()` 函数或 chat 命令 `/subagents spawn <agent-id> <task>` 触发 subagent。

---

## 审查门控

```
编码完成 → status: review_pending

[OpenCode]    调用 @reviewer 执行代码审查
[Claude Code] Task tool spawn reviewer subagent
[Cursor]      Task tool spawn reviewer subagent
[OpenClaw]    sessions_spawn('reviewer', '审查任务...')

pass   → python run.py task transition --task-id $TASK_ID --status completed --actor reviewer
reject → python run.py task transition --task-id $TASK_ID --status rejected
          读取 reviewer_notes → 针对性修复 → 重新提交
```
编码完成 → status: review_pending

[OpenCode]    调用 @reviewer 执行代码审查
[Claude Code] Task tool spawn reviewer subagent
              （加载 $SKILLS_DIR/evolving-agent/agents/reviewer.md 作为 prompt）

pass   → python run.py task transition --task-id $TASK_ID --status completed --actor reviewer
reject → python run.py task transition --task-id $TASK_ID --status rejected
         读取 reviewer_notes → 针对性修复 → 重新提交
```

---

## 知识归纳

```
所有任务 completed 后：

[OpenCode]    调用 @evolver 提取经验
[Claude Code] Task tool spawn evolver subagent
[Cursor]      Task tool spawn evolver subagent
[OpenClaw]    sessions_spawn('evolver', '知识归纳...')
```

---

## 多 Agent 编排

主进程（SKILL.md）即 orchestrator，直接调度子 agent：

### OpenCode

```
主进程 = orchestrator（SKILL.md）
    ├─ python $RUN_PY knowledge trigger ...  ← 知识检索（直接脚本，<1s）
    ├─ @coder      ← 代码执行，可并行多个
    ├─ @reviewer   ← 代码审查，独立上下文
    └─ @evolver    ← 知识归纳
```

### Claude Code / Cursor

```
主进程 = orchestrator（SKILL.md）
    ├─ Bash("python $RUN_PY knowledge trigger ...")  ← 知识检索（直接脚本，<1s）
    ├─ Task(coder, "...")       ← 代码执行，可并行多个
    ├─ Task(reviewer, "...")    ← 代码审查，独立上下文
    └─ Task(evolver, "...")     ← 知识归纳
```

### OpenClaw

```
主进程 = orchestrator（SKILL.md）
    ├─ run("python $RUN_PY knowledge trigger ...")   ← 知识检索（直接脚本，<1s）
    ├─ sessions_spawn('coder', '...')       ← 代码执行，可并行多个
    ├─ sessions_spawn('reviewer', '...')    ← 代码审查，独立上下文
    └─ sessions_spawn('evolver', '...')     ← 知识归纳
```

---

## Agent 文件位置

| Agent | 模型 | OpenCode | Claude Code/Cursor | OpenClaw |
|-------|------|----------|-------------------|-----------|
| coder | `zai-coding-plan/glm-5.1` | `~/.config/opencode/agents/coder.md` | `$SKILLS_DIR/evolving-agent/agents/coder.md` | `$SKILLS_DIR/evolving-agent/agents/coder.md` |
| reviewer | `opencode/claude-sonnet-4-6` | `~/.config/opencode/agents/reviewer.md` | `$SKILLS_DIR/evolving-agent/agents/reviewer.md` | `$SKILLS_DIR/evolving-agent/agents/reviewer.md` |
| evolver | `zai-coding-plan/glm-5` | `~/.config/opencode/agents/evolver.md` | `$SKILLS_DIR/evolving-agent/agents/evolver.md` | `$SKILLS_DIR/evolving-agent/agents/evolver.md` |

> OpenCode 使用原生 agent 目录 (`~/.config/opencode/agents/`)，其他平台将 agent 文件放在 skill 目录中作为 subagent prompt。
> OpenClaw 将 agent 文件放在 `~/.openclaw/skills/` 作为 subagent prompt。
> orchestrator 由主进程（SKILL.md）承担，不需要单独的 agent 文件。
