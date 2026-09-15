# API 接口文档

## 基础信息

- Base URL: `http://localhost:8080/api`
- WebSocket: `ws://localhost:8080/ws`
- 数据格式: JSON (UTF-8)

---

## 通用错误契约

所有接口错误返回统一结构：

```json
{
  "code": 40001,
  "message": "英雄99为赛事禁用英雄，不可ban"
}
```

| 错误码 | message 示例 | 含义 |
|--------|-------------|------|
| `40001` | `英雄X为赛事禁用英雄，不可ban/pick` | 赛事禁用（如雅典娜），情况① |
| `40002` | `英雄X已被禁用，不可ban/pick` | 已被任一方 ban 过 |
| `40003` | `英雄X已被蓝方/红方选择，不可ban/pick` | 已被对方 pick，情况② |
| `40004` | `英雄X不存在，不可ban/pick` | 英雄 ID 不合法 |
| `40005` | `英雄X受全局BP规则限制，此方不可ban/pick` | 全局BP记忆的不对称权限限制 |
| `40006` | `期望的动作是 X，实际传入 Y` | 动作类型与当前步骤不匹配 |
| `40007` | `BP流程未运行，当前状态=X` | 引擎未处于 Running |
| `40008` | `BP 尚未完成，不能收尾` | FinishGame 前校验失败 |

---

## 英雄状态契约（HeroStatus）

> 后端每次推送 BP 状态时，携带每个英雄的 `status` 字段（或前端根据 ban/pick 列表自行推导）。
> 前端按下表渲染：

| status | 含义 | 可ban | 可pick | 前端表现 |
|--------|------|:-----:|:------:|----------|
| `available` | 可用 | ✅ | ✅ | 正常显示，可点击 |
| `tournament_banned` | 赛事禁用（如雅典娜） | ❌ | ❌ | 界面完全不显示 |
| `banned_by_blue` | 已被蓝方禁用 | ❌ | ❌ | 禁用区，灰色 |
| `banned_by_red` | 已被红方禁用 | ❌ | ❌ | 禁用区，灰色 |
| `picked_by_blue` | 已被蓝方选择 | ❌ | ❌ | 蓝方阵容，灰色 |
| `picked_by_red` | 已被红方选择 | ❌ | ❌ | 红方阵容，灰色 |
| `restricted_by_rule` | 全局BP记忆限制（不对称权限） | 视规则 | 视规则 | 按 `permissions` 字段分别渲染 |
| `unknown` | 英雄ID不存在 | ❌ | ❌ | 忽略 |

**判定顺序（后端 `heroCan()`）**：赛事禁用 → 已被禁用 → 已被选择 → 权限表（全局BP记忆） → 是否在池 → 未知

**全局BP记忆规则（跨局）**：每局开始由编排器依据上一局结果派生 `permissions`——
- 上局红方选了 x → 本局 x：`{blueCanBan:false, blueCanPick:true, redCanBan:true, redCanPick:false}`
- 上局蓝方选了 y → 本局 y：`{blueCanBan:true, blueCanPick:false, redCanBan:false, redCanPick:true}`
- 赛事禁用英雄：`{全部为 false}`
- 其余英雄：`{全部为 true}`

> 接口响应中，`heroes[].status` 为该英雄的全局状态；若有全局BP记忆，额外携带 `heroes[].permissions` 字段（4 个布尔）供前端按双方视角渲染。

---

## 英雄相关

### `GET /api/heroes` 获取英雄列表

- 查询参数：
  - `status`（可选）：按状态过滤（`available` / `banned` / `picked`，默认全部活跃英雄）
- 响应 200：

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "name": "关羽",
      "iconPath": "/images/heroes/guan_yu.png",
      "lanes": ["对抗路"],
      "status": "available"
    }
  ]
}
```

> 说明：赛事禁用英雄（如雅典娜）**不会出现在此接口的返回中**，前端天然不显示。

---

## 房间相关

### `POST /api/rooms` 创建房间

- 请求体：

```json
{
  "blueTeamId": 1,
  "redTeamId": 2
}
```

- 响应 201：

```json
{
  "code": 0,
  "data": {
    "roomId": "7f4a2c9e",
    "status": "waiting"
  }
}
```

### `POST /api/rooms/:id/join` 加入房间

- 请求体：

```json
{
  "teamId": 1,
  "side": "blue"
}
```

- 响应 200：`{ "code": 0, "data": { "roomId": "...", "side": "blue" } }`

### `GET /api/rooms/:id/status` 获取当前 BP 状态

- 响应 200：

```json
{
  "code": 0,
  "data": {
    "roomId": "7f4a2c9e",
    "status": "ongoing",
    "blueTeamId": 1,
    "redTeamId": 2,
    "currentTurn": 5,
    "round": 1,
    "action": "pick",
    "side": "blue",
    "blueBanned": [],
    "redBanned": [],
    "bluePicked": [],
    "redPicked": [],
    "heroes": [
      { "id": 1, "status": "available" }
    ]
  }
}
```

### `POST /api/rooms/:id/ban` 执行 ban 操作

- 请求体：`{ "heroId": 12 }`
- 响应 200：`{ "code": 0, "data": { "currentTurn": 6 } }`
- 错误：`40001`~`40006` 之一

### `POST /api/rooms/:id/pick` 执行 pick 操作

- 请求体：`{ "heroId": 5 }`
- 响应 200：`{ "code": 0, "data": { "currentTurn": 7 } }`
- 错误：`40001`~`40006` 之一

---

## 对局相关

### `GET /api/games` 查询历史对局

- 查询参数：`matchId`（可选）
- 响应 200：

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "matchId": 1,
      "gameNumber": 1,
      "blueTeamId": 1,
      "redTeamId": 2,
      "winner": "blue",
      "bpCompleted": true,
      "banPickActions": [
        {
          "stepOrder": 1,
          "actionType": "ban",
          "side": "blue",
          "heroId": 12
        }
      ]
    }
  ]
}
```

---

## WebSocket 协议

连接地址：`ws://localhost:8080/ws?roomId=7f4a2c9e`

### 客户端 → 服务端消息

```json
{ "type": "ban",  "heroId": 12 }
{ "type": "pick", "heroId": 5 }
```

### 服务端 → 客户端消息

```json
{
  "type": "bp_update",
  "data": {
    "currentTurn": 6,
    "action": "pick",
    "side": "blue",
    "blueBanned": [12],
    "redBanned": [],
    "bluePicked": [],
    "redPicked": [],
    "heroes": [ { "id": 1, "status": "available" } ]
  }
}

{ "type": "error", "data": { "code": 40003, "message": "英雄5已被蓝方选择，不可pick" } }
```

### 消息类型

| type | 方向 | 说明 |
|------|------|------|
| `ban` / `pick` | 客户端→服务端 | 执行 BP 操作 |
| `bp_update` | 服务端→客户端 | 广播最新 BP 状态 |
| `error` | 服务端→客户端 | 操作失败，携带错误码和原因 |
| `bp_finished` | 服务端→客户端 | BP 全部完成 |