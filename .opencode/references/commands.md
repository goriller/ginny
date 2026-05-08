# 命令速查 & 运维清单

## 命令速查

```bash
# 初始化（首次使用，$SKILLS_DIR 路径只需用这一次）
if [ -d ~/.config/opencode/skills/evolving-agent ]; then SKILLS_DIR=~/.config/opencode/skills; elif [ -d ~/.openclaw/skills/evolving-agent ]; then SKILLS_DIR=~/.openclaw/skills; elif [ -d ~/.agents/skills/evolving-agent ]; then SKILLS_DIR=~/.agents/skills; else SKILLS_DIR=~/.claude/skills; fi
python $SKILLS_DIR/evolving-agent/scripts/run.py mode --init
# → init 完成后，脚本拷贝到 $PROJECT_ROOT/.opencode/scripts/run.py

# 后续所有命令使用本地路径（避免 IDE 授权提示）
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
RUN_PY="$PROJECT_ROOT/.opencode/scripts/run.py"

# 进化模式
python $RUN_PY mode --status|--off

# 知识库（全局）
python $RUN_PY knowledge query --stats
python $RUN_PY knowledge trigger --input "..." --project $PROJECT_ROOT
python $RUN_PY knowledge dashboard

# 知识库（项目级，隔离跨项目噪音）
echo "问题：xxx → 解决：yyy" | python $RUN_PY knowledge summarize --auto-store --project $PROJECT_ROOT

# 迁移：将全局KB中的项目特有条目移到项目级KB
python $RUN_PY knowledge migrate --list                          # 列出全局KB所有条目
python $RUN_PY knowledge migrate \
  --project /path/to/project \
  --keywords "kw1,kw2,kw3" \
  --dry-run                                                       # 预览哪些条目将被迁移
python $RUN_PY knowledge migrate \
  --project /path/to/project \
  --keywords "kw1,kw2,kw3"                                        # 执行迁移（自动备份）
python $RUN_PY knowledge migrate \
  --batch rules.json                                              # 批量多项目迁移

# rules.json 格式:
# [{"project": "/path/to/project", "keywords": ["kw1", "kw2"]}]

# GitHub
python $RUN_PY github fetch <url>

# 项目
python $RUN_PY project detect .

# 任务管理
python $RUN_PY task status --json            # 查看所有任务状态（JSON）
python $RUN_PY task list                     # 列出所有任务
python $RUN_PY task create --name "..." --description "..."  # 创建任务
python $RUN_PY task transition \
  --task-id task-001 --status in_progress    # 状态转换（orchestrator/coder 用）
python $RUN_PY task transition \
  --task-id task-001 --status completed \
  --actor reviewer \
  --reviewer-notes "LGTM: no issues"        # 标记完成（仅 reviewer 可用）
python $RUN_PY task cleanup                  # 清理已完成会话
python $RUN_PY task cleanup --force          # 强制清理旧会话（废弃 review_pending 残留）
```

---

## 健康检查清单

| 检查项 | 检查方式 | 异常处理 |
|--------|----------|----------|
| 任务进度 | 读取 `.opencode/progress.txt` | 如长时间无更新，检查是否阻塞 |
| 任务状态 | 读取 `.opencode/feature_list.json` | 如有 blocked 状态，分析依赖并调整 |
| 代码规范 | 运行 linter/formatter | 如有错误，中断并修复 |
| 测试通过 | 运行测试命令 | 如失败，中断并修复 |

---

## 结果验证清单

| 验证项 | 验证方式 | 通过条件 |
|--------|----------|----------|
| 任务完成 | 检查 `feature_list.json` | 所有任务状态为 `completed`（无 review_pending/rejected） |
| 审查通过 | 检查 `review_status` 字段 | 所有任务 `review_status` 为 `pass` |
| 经验提取 | 检查 `.evolution_mode_active` 存在时 evolver 是否调用 | 进化模式激活时，evolver 已调用，知识库已更新 |
| 产出质量 | reviewer 审查结论 | reviewer 全部 pass |
