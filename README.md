# KPL BP Simulator

KPL 王者荣耀 BP（Ban/Pick）模拟器 — 前后端分离的实时对战项目。

> **技术栈**：React + TypeScript + Vite（前端） / Go + Gin（后端） / MySQL + Redis（存储） / Docker（部署）

---

## 快速开始

```bash
# 加载环境变量（Go 路径等）
source scripts/env.sh

# 启动后端
cd backend && go build ./cmd/server/ && ./server

# 启动前端
cd frontend && npm run dev
```

---

## 依赖配置

### 环境要求

| 依赖 | 版本 | 用途 |
|------|------|------|
| Go | ≥ 1.22 | 后端运行时 |
| Node.js | ≥ 18 | 前端运行时 |
| npm | ≥ 10 | 前端包管理 |
| Docker | ≥ 20 | 容器化部署（可选） |
| Docker Compose | ≥ 2.0 | 多服务编排（可选） |

### 后端依赖（Go Modules）

配置在 [`backend/go.mod`](backend/go.mod)：

```bash
cd backend
go mod tidy          # 下载/更新依赖
go mod download      # 仅下载
go build ./...       # 编译
go test ./...        # 运行测试
```

| 包 | 版本 | 用途 |
|----|------|------|
| `github.com/gin-gonic/gin` | v1.9.1 | HTTP Web 框架 |
| `github.com/redis/go-redis/v9` | v9.4.0 | Redis 客户端 |
| `github.com/gorilla/websocket` | v1.5.1 | WebSocket 通信 |
| `gorm.io/gorm` | v1.25.5 | ORM 框架 |
| `gorm.io/driver/mysql` | v1.5.2 | MySQL 数据库驱动 |

### 前端依赖（npm）

配置在 [`frontend/package.json`](frontend/package.json)：

```bash
cd frontend
npm install          # 安装依赖
npm run dev          # 开发模式
npm run build        # 生产构建
npm run preview      # 预览构建产物
```

| 包 | 类型 | 用途 |
|----|------|------|
| `react` | 生产 | UI 框架 |
| `react-dom` | 生产 | React DOM 渲染 |
| `react-router-dom` | 生产 | 路由管理 |
| `axios` | 生产 | HTTP 请求 |
| `zustand` | 生产 | 轻量状态管理 |
| `typescript` | 开发 | 类型系统 |
| `vite` | 开发 | 构建工具 |
| `@vitejs/plugin-react` | 开发 | React 插件 |

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_PORT` | `8080` | 后端 HTTP 端口 |
| `DB_HOST` | `localhost` | MySQL 地址 |
| `DB_PORT` | `3306` | MySQL 端口 |
| `DB_USER` | `root` | MySQL 用户名 |
| `DB_PASSWORD` | `root` | MySQL 密码 |
| `DB_NAME` | `kpl_bp` | 数据库名 |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址 |

---

## Docker 一键启动

```bash
cd deploy
docker compose up -d
```

服务启动后访问：
- 前端：<http://localhost:3000>
- 后端 API：<http://localhost:8080/api>
- WebSocket：`ws://localhost:8080/ws`

```bash
docker compose down    # 停止服务
docker compose logs -f # 查看日志
```

---

## 项目结构

```
kpl-bp-simulator/
├── backend/          # 后端（Go + Gin）
│   ├── cmd/server/   # 服务入口
│   ├── internal/     # 业务代码（handler/service/repository/model）
│   ├── config/       # 配置
│   ├── migrations/   # 数据库迁移 SQL
│   └── go.mod        # Go 依赖声明
├── frontend/         # 前端（React + TypeScript + Vite）
│   ├── src/          # 源代码
│   ├── index.html    # Vite 入口
│   └── package.json  # npm 依赖声明
├── deploy/           # Docker 部署配置
│   ├── Dockerfile.backend
│   ├── Dockerfile.frontend
│   └── docker-compose.yml
├── docs/             # 文档
├── scripts/          # 工具脚本
│   └── env.sh        # 环境变量配置
└── README.md
```

---

## 开发指南

### BP 规则速览

共 **20 步**，分两轮：

| 轮次 | 先手方 | Ban 顺序 | Pick 顺序 |
|------|--------|----------|-----------|
| 第一轮 | 🔵 蓝队 | 蓝→红→蓝→红 (4ban) | 蓝→红→红→蓝→蓝→红 (6pick) |
| 第二轮 | 🔴 红队 | 红→蓝→红→蓝→红→蓝 (6ban) | 红→蓝→蓝→红 (4pick) |

### 英雄可用状态（HeroStatus）

> 前后端联调的核心契约。后端通过 `heroStatus()` / `heroCan()` 计算，前端据此渲染 BP 界面。

| 状态 | 含义 | 可ban | 可pick | 前端表现 |
|------|------|:-----:|:------:|----------|
| `HeroAvailable` | 可用 | ✅ | ✅ | 正常显示，可点击 |
| `HeroTournamentBanned` | 赛事禁用（如雅典娜） | ❌ | ❌ | 界面完全不显示 |
| `HeroBannedByBlue` | 已被蓝方禁用 | ❌ | ❌ | 禁用区，灰色 |
| `HeroBannedByRed` | 已被红方禁用 | ❌ | ❌ | 禁用区，灰色 |
| `HeroPickedByBlue` | 已被蓝方选择 | ❌ | ❌ | 蓝方阵容，灰色 |
| `HeroPickedByRed` | 已被红方选择 | ❌ | ❌ | 红方阵容，灰色 |
| `HeroRestrictedByRule` | 全局BP记忆限制（某方仅部分权限） | 视规则 | 视规则 | 按双方权限分别渲染 |
| `HeroUnknown` | 英雄ID不存在 | ❌ | ❌ | 忽略 |

**英雄池规则（标准模式）**：两队共享一个 HeroPool，任何被 ban/pick 的英雄都会从池中移除，双方都不能再用。

**全局BP记忆模式（跨局）**：上局某方 pick 过的英雄，本局产生不对称权限——
- 上局**红方**选了 x → 本局 x：**蓝方可选但不可ban**，**红方可ban但不可选**
- 上局**蓝方**选了 y → 本局 y：**红方可选但不可ban**，**蓝方可ban但不可选**
- 其余英雄（含上局被 ban 的）恢复为双方全可用

详细规则见 [实现思路.md](实现思路.md)，完整架构见 [项目架构.md](项目架构.md)。

---

## License

MIT
