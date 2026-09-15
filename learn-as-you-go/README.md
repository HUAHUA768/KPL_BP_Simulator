# KPL BP Simulator — 边学边做指南

> 📚 **本文档记录整个项目的提交过程和技术要点**，适合初学者学习全栈项目开发。

---

## 📋 任务概览

| 步骤 | 操作 | 说明 |
|------|------|------|
| 1️⃣ | 初始化 Git 仓库 | 本地版本控制管理 |
| 2️⃣ | 配置 Git 用户信息 | 提交时显示作者信息 |
| 3️⃣ | 添加文件到暂存区 | `git add .` 准备提交 |
| 4️⃣ | 创建首次提交 | `git commit -m "..."` |
| 5️⃣ | 关联远程仓库 | 连接到 GitHub |
| 6️⃣ | 推送到 GitHub | `git push` 发布代码 |

---

## 🔧 具体命令详解

### 1. 初始化 Git 仓库

```bash
cd /home/xiaohua/KPL_BP_Simulator
git init -b main
```

- `git init`: 在当前目录初始化 Git 仓库（创建 `.git` 目录）
- `-b main`: 指定默认分支名称为 `main`（Git 新版本默认）

> 💡 **提示**: 之前 Git 默认使用 `master` 分支，现在 GitHub 默认使用 `main`。

---

### 2. 配置 Git 用户信息

```bash
git config user.email "your-email@example.com"
git config user.name "Your Name"
```

或一次性设置：

```bash
git config user.email "youremail@example.com" && git config user.name "你的昵称"
```

> 💡 **说明**: 这些信息会出现在每次提交的元数据中，GitHub 会根据邮箱显示头像。

---

### 3. 查看仓库状态

```bash
git status
```

- 显示未跟踪的文件（未被 Git 管理的文件）
- 显示已修改但未暂存的文件

---

### 4. 优化 .gitignore

`.gitignore` 文件用于排除不需要版本控制的文件：

```gitignore
# KPL BP Simulator - .gitignore

# External dependencies
node_modules/
vendor/

# Go standard library and test files (exclude the entire go directory)
go/

# Build output
dist/
build/

# Go build cache
.gocache/
.gomodcache/

# IDE settings
.vscode/
.idea/

# Environment variables
.env
.env.local
```

> 💡 **重要**: 
> - `node_modules/` → npm 依赖（体积大，可重新安装）
> - `go/` → Go 标准库（不是项目代码，无需提交）
> - `.gocache/` 和 `.gomodcache/` → Go 编译缓存

---

### 5. 添加所有文件到暂存区

```bash
git add .
```

- 将所有新文件和修改添加到暂存区（staging area）
- `git add <file>` 可以添加单个文件
- `git add -A` 添加所有变更（包括删除）

---

### 6. 检查待提交内容

```bash
git status --short
```

- 列出所有将要被提交的文件
- `??` 表示未跟踪文件
- `M` 表示已修改文件
- `A` 表示已添加文件

---

### 7. 创建提交

```bash
git commit -m "Initial commit: KPL BP Simulator project (React + Go + MySQL + Redis)"
```

- `-m`: 附带提交信息（commit message）
- **好的提交信息格式**：简洁描述 + 关键特性
  ✅ `Initial commit: KPL BP Simulator project (React + Go + MySQL + Redis)`
  ❌ `update` 或 `fix`（太模糊）

---

### 8. 推送前的清理（如果包含无用文件）

如果发现提交了太多不必要的文件：

```bash
# 移除特定目录的缓存引用
git rm -r --cached go/ .gocache/ .gomodcache/

# 重置提交并重新添加
git commit --amend -m "新的提交信息" --quiet
```

- `--cached`: 只从 Git 索引中移除，不影响工作目录
- `--amend`: 修改最后一次提交

---

### 9. 关联远程仓库

```bash
git remote add origin https://github.com/HUAHUA768/KPL_BP_Simulator.git
```

- `origin`: 远程仓库的别名（约定俗成）
- `https://...`: 远程仓库地址

> 💡 **验证**: `git remote -v` 可查看关联的远程仓库

---

### 10. 推送到 GitHub

```bash
git push -u origin main
```

- `-u`: 设置上游分支（upstream），之后只需 `git push`
- `origin main`: 推送到 `origin` 的 `main` 分支

#### 🔐 认证方式

**方式一：HTTPS + Token（推荐新手）**

1. 访问 GitHub → Settings → Developer settings → Personal access tokens
2. 生成 Token（Classic → `repo` 权限）
3. 推送时输入：
   - Username: GitHub 用户名
   - Password: 刚才生成的 Token（不是账号密码！）

**方式二：SSH Key（更便捷）**

```bash
# 生成 SSH Key
ssh-keygen -t ed25519 -C "your-email@example.com"

# 复制到剪贴板
cat ~/.ssh/id_ed25519.pub | pbcopy  # Mac
# 或 xclip -sel clip < ~/.ssh/id_ed25519.pub  # Linux

# 在 GitHub 添加 SSH Key
Settings → SSH and GPG keys → New SSH key

# 修改远程仓库 URL
git remote set-url origin git@github.com:HUAHUA768/KPL_BP_Simulator.git

# 推送
git push
```

---

## 🏗️ 项目结构说明

```
KPL_BP_Simulator/
├── backend/              # Go 后端
│   ├── cmd/server/       # 服务端入口
│   ├── internal/         # 业务逻辑（handler/service/repository）
│   ├── config/           # 配置文件
│   ├── migrations/       # 数据库迁移脚本
│   └── go.mod            # Go 依赖声明
├── frontend/             # React 前端
│   ├── src/              # 源代码
│   ├── public/           # 静态资源
│   └── package.json      # npm 依赖声明
├── deploy/               # Docker 部署配置
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend
│   └── docker-compose.yml
├── docs/                 # 项目文档
├── scripts/              # 工具脚本
├── README.md             # 项目说明
├── 实现思路.md            # BP 规则实现细节
├── 项目架构.md            # 系统架构设计
└── 边学边做/             # 本学习文档
    └── README.md
```

---

## 🎯 核心技术点

### 1. 前后端分离架构

| 层级 | 技术 | 职责 |
|------|------|------|
| 前端 | React + TypeScript | 界面展示、用户交互 |
| 后端 | Go + Gin | RESTful API + WebSocket |
| 数据库 | MySQL | 持久化存储 |
| 缓存 | Redis | 临时数据、会话管理 |

### 2. BP 状态机设计

使用 **枚举 + switch** 模式实现严格的步骤控制：

```go
const (
    BPTurn1BlueBan = iota + 1 // 1: 蓝方 Ban
    BPTurn2RedBan             // 2: 红方 Ban
    // ... 共 20 步
    BPTurn20RedPick           // 20: 红方 Pick
)

func (e *Engine) ExecuteAction(action ActionType, heroID int) error {
    switch e.currentTurn {
    case BPTurn1BlueBan, BPTurn3BlueBan, BPTurn4RedBan:
        return e.handleBan(side, heroID)
    // ... 其他情况
    default:
        return ErrInvalidTurn
    }
}
```

> 💡 **优势**: 
> - exhaustive check → 漏掉 case 会编译报错
> - 清晰的状态流转逻辑
> - 易于测试和维护

### 3. 英雄可用性规则

两队共享一个 HeroPool：
- 任意一方 ban/pick → 英雄从池中移除
- 双方都无法再使用该英雄
- 区分具体禁用原因（赛事禁用 vs 已被 ban vs 已被 pick）

### 4. WebSocket 实时通信

```
前端 ←→ WebSocket Hub ←→ Room Manager ←→ BP Engine
          ↑
      Client(连接管理)
```

- Hub: 管理所有房间的连接
- Client: 处理单个 WS 连接的收发消息
- 实时推送 BP 状态更新

---

## 📚 Git 常用命令速查表

| 命令 | 说明 |
|------|------|
| `git init` | 初始化 Git 仓库 |
| `git add <file>` | 添加文件到暂存区 |
| `git add .` | 添加所有变化到暂存区 |
| `git status` | 查看仓库状态 |
| `git commit -m "msg"` | 创建提交 |
| `git log --oneline` | 查看提交历史 |
| `git remote add origin url` | 添加远程仓库 |
| `git push -u origin main` | 推送到远程 |
| `git pull` | 拉取远程更新 |
| `git checkout -b feature` | 新建分支 |
| `git merge branch` | 合并分支 |

---

## 🚀 下一步建议

1. **本地测试运行项目**
   ```bash
   source scripts/env.sh
   cd backend && go build ./cmd/server/ && ./server
   cd frontend && npm run dev
   ```

2. **完善 CI/CD 流程**
   - 配置 GitHub Actions 自动测试
   - Docker 镜像自动化构建

3. **编写单元测试**
   - 后端 BP 引擎测试 (`bp_engine_test.go`)
   - 前端组件测试 (Jest + React Testing Library)

4. **添加更多功能**
   - 用户登录系统
   - 历史记录查询
   - 观战模式

---

## 📖 参考资源

- [Go 官方文档](https://go.dev/doc/)
- [Gin 框架教程](https://github.com/gin-gonic/gin)
- [React 官方文档](https://react.dev/)
- [TypeScript 手册](https://www.typescriptlang.org/docs/)
- [Git 完整教程](https://git-scm.com/book/zh/v2)

---

**祝你学习愉快！** 🎉
