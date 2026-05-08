# GitHub to Knowledge

从 GitHub 仓库提取结构化知识。

## 核心命令

```bash
# 设置路径变量（优先使用项目本地脚本）
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
if [ -f "$PROJECT_ROOT/.opencode/scripts/run.py" ]; then RUN_PY="$PROJECT_ROOT/.opencode/scripts/run.py"; elif [ -d ~/.config/opencode/skills/evolving-agent ]; then RUN_PY=~/.config/opencode/skills/evolving-agent/scripts/run.py; elif [ -d ~/.agents/skills/evolving-agent ]; then RUN_PY=~/.agents/skills/evolving-agent/scripts/run.py; else RUN_PY=~/.claude/skills/evolving-agent/scripts/run.py; fi

# 获取仓库信息
python $RUN_PY github fetch <github_url>

# 提取知识
python $RUN_PY github extract --input repo_info.json

# 存储到知识库
python $RUN_PY github store --input extracted.json
```

## 工作流程

```
fetch → extract → store → 进化检查
```

### 1. 获取信息

```bash
python $RUN_PY github fetch https://github.com/user/repo
```

输出：仓库元数据、文件树、README、配置文件。

### 2. 提取知识

```bash
python $RUN_PY github extract --input repo_info.json
```

| 提取内容 | 目标分类 |
|---------|---------|
| 框架配置 | tech-stacks/ |
| 目录结构、分层设计 | patterns/ |
| 代码规范 | skills/ |
| 常见报错修复 | problems/ |

### 3. 存储知识

```bash
python $RUN_PY github store --input extracted.json
```

更新知识库和索引。

## 使用场景

### 主动学习

```
用户: "学习这个仓库 https://github.com/shadcn/ui"

执行:
1. fetch → 获取仓库信息
2. extract → 提取组件模式、Tailwind配置
3. store → 存储到 patterns/, tech-stacks/
4. 进化检查 → 如有额外经验则归纳
```

### 知识复用

```
用户: "用 shadcn/ui 风格写卡片组件"

执行:
1. 检索 "shadcn", "ui"
2. 命中 patterns/shadcn-component.json
3. 加载到上下文
4. 生成符合规范的代码
```

## 约束

1. **Schema 兼容**: 必须符合 `$SKILLS_DIR/evolving-agent/references/schema.json`
2. **不存储源码**: 仅提取知识，不克隆完整仓库
3. **原子化**: 拆分存储，不打包成大文件
4. **幂等性**: 重复学习更新现有条目
