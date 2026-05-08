# Knowledge Base

知识存储、查询、归纳系统。

## 知识分类

| 分类 | 目录 | 触发场景 |
|------|------|----------|
| experience | `experiences/` | 优化、重构、最佳实践 |
| tech-stack | `tech-stacks/` | 框架相关 |
| scenario | `scenarios/` | 创建、实现功能 |
| problem | `problems/` | 修复、调试、报错 |
| testing | `testing/` | 测试相关 |
| pattern | `patterns/` | 架构、设计模式 |
| skill | `skills/` | 通用技巧 |

## 核心命令

```bash
# 设置路径变量（优先使用项目本地脚本）
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
if [ -f "$PROJECT_ROOT/.opencode/scripts/run.py" ]; then RUN_PY="$PROJECT_ROOT/.opencode/scripts/run.py"; elif [ -d ~/.config/opencode/skills/evolving-agent ]; then RUN_PY=~/.config/opencode/skills/evolving-agent/scripts/run.py; elif [ -d ~/.agents/skills/evolving-agent ]; then RUN_PY=~/.agents/skills/evolving-agent/scripts/run.py; else RUN_PY=~/.claude/skills/evolving-agent/scripts/run.py; fi

# 查询
python $RUN_PY knowledge query --stats           # 统计
python $RUN_PY knowledge query --trigger "react,hooks"  # 按触发词
python $RUN_PY knowledge query --category problem       # 按分类
python $RUN_PY knowledge query --search "跨域"          # 全文搜索

# 触发检测
python $RUN_PY knowledge trigger --input "修复CORS问题"
python $RUN_PY knowledge trigger --input "..." --project .

# 归纳存储
echo "内容" | python $RUN_PY knowledge summarize --auto-store

# 存储
python $RUN_PY knowledge store --category experience --name "xxx"
```

## 工作流程

### 检索流程（任务开始时）

Orchestrator 直接执行脚本检索全局知识库并合并已有项目经验（<1s，无需 sub-agent）：

```bash
python $RUN_PY knowledge trigger \
  --input "..." --format context --mode hybrid \
  --merge "$PROJECT_ROOT/.opencode/.knowledge-context.md" \
  > "$PROJECT_ROOT/.opencode/.knowledge-context.md"
```

> `--merge` 保留文件中的"项目经验"部分，全局知识库检索结果每次刷新。

### 归纳流程（任务结束后）

检查 `.opencode/.evolution_mode_active`，满足条件则由 @evolver 执行：

1. **全局知识库**（跨项目复用）：
```bash
echo "问题：xxx → 解决：yyy" | python $RUN_PY knowledge summarize --auto-store
```

2. **项目知识上下文**（项目专属，追加）：
```bash
echo -e "\n### $(date +%Y-%m-%d) 问题：xxx → 解决：yyy" >> "$PROJECT_ROOT/.opencode/.knowledge-context.md"
```

## 知识条目 Schema（全局知识库）

```json
{
  "id": "category-name-hash",
  "category": "experience|tech-stack|scenario|problem|testing|pattern|skill",
  "name": "名称",
  "triggers": ["触发词"],
  "content": {},
  "sources": ["来源"],
  "created_at": "ISO-8601",
  "effectiveness": 0.5
}
```

## 数据位置

| 类型 | 路径 | 说明 |
|------|------|------|
| 全局知识库 | `~/.config/opencode/knowledge/` | 结构化JSON，跨平台跨项目共享 |
| 项目知识上下文 | `$PROJECT_ROOT/.opencode/.knowledge-context.md` | Markdown，项目专属，跨会话持久化 |

> 全局知识库由 OpenCode、Claude Code、Cursor 共享。项目知识上下文天然隔离。

### .knowledge-context.md 文件格式

```markdown
## 相关知识（每次检索刷新）
### [problem] CORS跨域问题
**解决方案**: ...

## 可能相关（每次检索刷新）
### [experience] React性能优化
**最佳实践**: ...

## 项目经验（跨会话持久化）
### 2026-03-13 问题：MinIO签名过期 → 解决：设置 presigned URL 有效期为 7 天
### 2026-03-12 决策：选择 Gin 而非 Echo → 原因：团队更熟悉 Gin 中间件体系
```

## 子代理

| 代理 | 文件 | 用途 |
|------|------|------|
| evolver | `$SKILLS_DIR/evolving-agent/agents/evolver.md` | 经验归纳 → 全局知识库 + 项目知识上下文 |

> 注：知识检索已改为 orchestrator 直接执行脚本（`run.py knowledge trigger`），不再使用 sub-agent。经验归纳仍由 @evolver 子代理执行。
