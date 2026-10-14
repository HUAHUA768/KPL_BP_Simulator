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
- 上局某队选了 x → 本局 x：**该队** `可 ban 不可 pick`，**对手** `可选不可 ban`
- 赛事禁用英雄：`{全部为 false}`
- 其余英雄：`{全部为 true}`

> ⚠️ **限制跟队伍走，不跟颜色走**（选边后尤其重要）：第 2 局交换红蓝后，
> 某队上局选过的英雄依然对该队"可 ban 不可选"。后端会用**上一局**的
> `blue_team_id`/`red_team_id` 把记录的 `side`（颜色）翻译成队伍，再用**本局**归属翻译回颜色。
> 下表按"该队在本局是蓝方/红方"给出：
> - 该队本局是**蓝方** → `{blueCanBan:true, blueCanPick:false, redCanBan:false, redCanPick:true}`
> - 该队本局是**红方** → `{blueCanBan:false, blueCanPick:true, redCanBan:true, redCanPick:false}`

> 💡 **前端应据此提前置灰**：`heroes[].permissions` 是给前端做"预先禁用"用的 ——
> 当前操作方不可操作的英雄应直接变灰、鼠标禁用，而不是等用户点击后才收到
> 「受全局BP规则限制」的错误提示。

> 接口响应中，`heroes[].status` 为该英雄的全局状态；若有全局BP记忆，额外携带 `heroes[].permissions` 字段（4 个布尔）供前端按双方视角渲染。

---

## 元流之子「五选一」规则

> 五个分路的元流之子（坦克/法师/射手/刺客/辅助），**每支队伍本场比赛只能 pick 一个**。

**规则定义**（与全局BP记忆叠加生效）：

| 情形 | 该队对元流组的效果 |
|------|-------------------|
| 本场未 pick 过任何元流 | 五个元流**全部可 pick**（完整的"五选一"权力） |
| 本场已 pick 过某一个 | 该英雄自身：不可再 pick（全局BP记忆）；**其余四个：不可 pick**（本规则） |

> ⚠️ **ban 权是「不对称」的**（需求 2，易混，务必看清）：
>
> | | 用过元流的一方（A） | 对手（B） |
> |---|---|---|
> | pick 其余四个 | ❌ 不可以 | ✅ 可以（B 依然五选一） |
> | **ban 其余四个** | ✅ **可以**（"A 队依然有 ban 五个元流的权力"） | ❌ **不可以**（"B 队不用再 ban 其它四个元流"） |
>
> 一句话：**A 用过之后，其余四个元流对 B 已无 pick 威胁，因此 B 不必（也不得）再 ban 它们；
> 而 A 自己仍可 ban —— ban 权是防守手段，不因己方已使用而剥夺。**
>
> 前端据此渲染出两种不同表现：对 A 而言不可 pick 但**可 ban**（ban 轮次仍彩色可点）；
> 对 B 而言不可 pick 也**不可 ban**（始终灰化 + 「限」角标）。

> ⚠️ **锁是按队伍的，不是全局资源**：A 队被锁**不影响** B 队的五选一权力。

> ⚠️ **ban 不计入**：ban 掉元射不会锁死其它四个，也不会剥夺对手的 ban 权；
> 被 ban 也不触发锁定（仅该英雄本局不可用）。

**生效时机（两条路径，结果一致）**：

1. **局内动态**：本局某方 pick 组内英雄后，引擎**立刻**收紧权限
   （`Engine.ExclusiveGroups` + `lockExclusiveGroupFor`）；
2. **跨局累计**：每局开始时（`GameService.StartGame`）遍历**全部历史局**的 pick 记录，
   把累计使用情况写入本局初始权限表。

> 💡 限制**跟队伍走，不跟颜色走**：换边后，某队用过的元流依然锁它自己，
> 不会把限制加到对手身上（与全局BP记忆同一套队伍翻译逻辑）。

**权限位示例**（第 1 局蓝方 pick 元射，第 2 局未换边）：

```
坦克/法师/刺客/辅助  blue(ban=true,  pick=false)  red(ban=false, pick=true)
射手(元射)           blue(ban=true,  pick=false)  red(ban=false, pick=true)
                     ↑ A队可ban不可选        ↑ B队可五选一但不可ban
```

**错误码**：非法 pick 被后端独立拒绝，返回
`{"code":40001,"message":"英雄X受全局BP规则限制，此方不可pick"}`。
前端据此提前置灰（`restricted=true` → 灰化 + 「限」角标），因此正常操作路径下不会触发该错误。

**英雄识别**：按**名字前缀** `元流之子` 判定，**不硬编码 ID**
（ID 由 auto_increment 决定，换环境可能变化）。当前库中五个元流 ID 为 33/60/83/105/132。

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

> 一个房间 = 一场系列赛（BO5 / BO7 / 无限制）。创建时会同时写入 `match_record`
> 与第 1 局的 `game` 行，因此**无需**再单独调用开始对局接口。

- 请求体：

```json
{
  "blueTeamId": 1,
  "redTeamId": 2,
  "boFormat": 5
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `boFormat` | 否 | `5`=BO5(先 3 胜)、`7`=BO7(先 4 胜)、`0`=无限制(不自动结束)；缺省 `5`。其他值一律 400 |

- 响应 201：完整房间状态快照（结构同下方 `GET /api/rooms/:id/status`）

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
    "matchId": 1,
    "gameNumber": 2,
    "boFormat": 5,
    "blueTeamId": 1,
    "redTeamId": 2,
    "blueTeam": { "id": 1, "name": "重庆狼队", "logoUrl": "/images/teams/chongqin_wolves.png" },
    "redTeam": { "id": 2, "name": "成都AG超玩会", "logoUrl": "/images/teams/chengdu_ag.png" },
    "teamAId": 1,
    "teamBId": 2,
    "sidePickerTeamId": 2,
    "nextBlueTeamId": null,
    "nextRedTeamId": null,
    "blueScore": 1,
    "redScore": 0,
    "winner": null,
    "matchStatus": "ongoing",
    "currentTurn": 5,
    "round": 1,
    "action": "pick",
    "side": "blue",
    "blueBanned": [12, 34],
    "redBanned": [45, 67],
    "bluePicked": [],
    "redPicked": [],
    "heroes": [
      {
        "id": 1,
        "status": "available",
        "permissions": {
          "blueCanBan": true,
          "blueCanPick": true,
          "redCanBan": true,
          "redCanPick": true
        }
      }
    ]
  }
}
```

**系列赛字段语义**：

| 字段 | 说明 |
|------|------|
| `status` | **本局** BP 状态：`ongoing` / `finished` |
| `gameNumber` | 当前第几局（1 起） |
| `boFormat` | 赛制：5 / 7 / 0 |
| `blueTeam` / `redTeam` | 双方队名 + 队标（`{id, name, logoUrl}`）；查不到时为 `null`，前端降级显示"🔵 蓝方 / 红方 🔴" |
| `teamAId` / `teamBId` | 系列赛**固定队伍身份**，创建后永不改变（选边不影响） |
| `blueScore` / `redScore` | 大比分的**红蓝视图**（按当前局红蓝归属映射，会随选边而"翻转视图"，但队伍实际得分不变） |
| `sidePickerTeamId` | 当前拥有选边权的队伍（联机模式 = 败方；单机模式仅落库预留） |
| `nextBlueTeamId` / `nextRedTeamId` | 已指定的下一局红蓝归属；未指定为 `null` |
| `winner` | **本局**已提交的胜方 `"blue"` / `"red"`；未提交或进入下一局后为 `null` |
| `matchStatus` | **整个系列赛**状态：`ongoing` / `finished` |

> ⚠️ **比分语义（重要）**：`team_a_score` / `team_b_score` 落库时**跟队伍身份走**，
> 而 `blueScore` / `redScore` 是同一组比分在当前红蓝归属下的**视图**。
> 第 2 局换边后，某队的分不会因为"它从蓝方变成红方"而记到对手名下。

### `POST /api/rooms/:id/ban` 执行 ban 操作

- 请求体：`{ "heroId": 12 }`
- 响应 200：完整房间状态快照（结构同 `GET /api/rooms/:id/status` 的 `data`，含执行后的 `currentTurn`）
- 错误：`40001`~`40007` 之一
- 第 20 步完成后本局 BP 结束，20 步操作会自动落库到 `ban_pick_action`

### `POST /api/rooms/:id/pick` 执行 pick 操作

- 请求体：`{ "heroId": 5 }`
- 响应 200：完整房间状态快照（同上）
- 错误：`40001`~`40007` 之一

> **回合归属说明**：ban/pick 接口都**不接收"操作方"参数**。当前该谁操作完全由服务端 BP 引擎的 20 步状态机决定（快照中的 `side` 即下一步的归属方），客户端无需也不应声明身份。前端单机模式据此实现"一人分饰两角"，详见 [BP单机交互方案决策](../learn-as-you-go/BP单机交互方案决策.md)。

### `POST /api/rooms/:id/result` 提交本局胜方

> BP 完成（`status=finished`）后必须提交胜方，才允许进入下一局。

- 请求体：`{ "winner": "blue" }`（或 `"red"`）
- 响应 200：完整房间状态快照，其中大比分已更新、`winner` 已写入
- 达到赛制所需胜场（BO5=3、BO7=4）时自动置 `matchStatus=finished`
- 副作用：把**败方**写入 `match_record.side_picker_team_id`（联机模式选边权，单机为预留）；该写入失败只记日志，**不阻断**胜负录入
- 错误：
  - `400` 本局 BP 尚未完成，不能提交胜负
  - `400` 本局胜方已提交
  - `400` 比赛已结束

### `POST /api/rooms/:id/next-game` 进入下一局（可同时选边）

> 触发**全局BP记忆规则**：后端读取上一局 `ban_pick_action`，按规则派生新一局的
> 英雄权限后创建新引擎。
>
> 全局BP限制**跟队伍走、不跟颜色走**：第 2 局选边交换红蓝后，某队上局选过的英雄
> 依然对该队"可 ban 不可选"，对对手"可选不可 ban"。

- 请求体（**全部可选**）：

```json
{
  "blueTeamId": 2,
  "redTeamId": 1
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `blueTeamId` / `redTeamId` | 否 | 指定下一局红蓝归属。**二者必须同时提供**，且必须是本场比赛的两支参赛队伍 |

- **不传 body**（`{}`）时沿用当前局红蓝归属，行为与旧版本完全一致（向后兼容）
- 响应 200：完整房间状态快照，`gameNumber` +1、`currentTurn=1`、四个 ban/pick 列表清空、`winner=null`；若指定了选边，`blueTeamId` / `redTeamId` 已切换，且 `blueScore` / `redScore` 视图已按新归属重新映射（队伍实际得分不变）
- 错误：
  - `400` 本局 BP 尚未完成 / 请先提交本局胜方 / 比赛已结束
  - `400` 蓝方与红方不能是同一支队伍
  - `400` 选边队伍必须是本场比赛的两支参赛队伍
  - `400` 选边需同时提供 blueTeamId 与 redTeamId
  - `500` 上一局 BP 记录尚未保存完成（防御性检查，稍后重试即可）

### `POST /api/rooms/:id/finish-match` 手动结束比赛

> 前端仅在无限制赛制（`boFormat=0`）显示入口；接口本身不校验赛制，
> 任何赛制都可提前结束。

- 请求体：无（`{}`）
- 响应 200：`matchStatus=finished` 的快照
- 错误：`400` 比赛已结束

### `POST /api/matches/:id/resume` 恢复对局，继续模拟 BP

> **用途**：从「历史对局」里选中一场**未结束**的比赛，回到 BP 房间继续打。
>
> 房间（Room）原本只存在于后端内存中，后端重启后 DB 里 `status='ongoing'` 的比赛
> 就回不到房间。本接口依据已落库数据把房间**重建**出来：
>
> | 数据源 | 提供的信息 |
> |--------|-----------|
> | `match_record` | 固定身份 team_a/team_b、赛制、大比分、`room_id` |
> | `game` | 每局红蓝归属、胜方、`bp_completed` |
> | `ban_pick_action` | 当前局已走的 N 步 → 逐字节回放到第 N 步 |
>
> 恢复目标是**当前进度**：
> - 当前局 BP 未走完 → 停在第 N+1 步继续
> - 当前局 BP 已走完但未录胜负 → 引擎置为 `finished`，前端直接进结算面板

- 请求体：无（`{}`）
- 响应 200：

```json
{
  "code": 0,
  "data": {
    "roomId": "bdf856fe",
    "state": { "...与 GET /api/rooms/:id/status 完全一致..." }
  }
}
```

- 前端拿到 `roomId` 后跳转 `/room/{roomId}` 即从当前进度继续
- **幂等**：房间已在内存中时直接复用，不会重复回放
- 错误：
  - `400` matchId 格式错误
  - `404` 比赛记录不存在
  - `400` 该比赛已结束，无法继续
  - `500` 恢复对局失败（如历史步骤与 20 步状态机序列不自洽）

> ⚠️ **前提：BP 步骤必须已落库**。房间层现在会**每步增量写入** `ban_pick_action`
> （见 `RoomHandler.persistAction`），因此中途退出/重启都能恢复；
> 若某步落库失败，恢复时会缺步，日志中会有 `[room] 第N步落库失败` 记录。

---

## 战队相关

### `GET /api/teams` 获取战队列表

- 响应 200：

```json
{
  "code": 0,
  "data": [
    { "id": 1, "name": "重庆狼队", "logoUrl": "images/teams/chongqin_wolves.png" }
  ]
}
```

> 数据来自 `team` 表（共 18 支，含 SYG / WST）。该接口取代了前端此前的硬编码战队数组。

---

## 对局相关

### `GET /api/games` 查询历史对局

- 查询参数：`matchId`（可选）。**不传时返回全部对局**（按 `match_id DESC, game_number ASC`）
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

### `GET /api/matches` 查询系列赛历史

- 查询参数：无（返回最近 50 场）
- 响应 200：

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "teamAId": 1,
      "teamBId": 2,
      "teamAScore": 3,
      "teamBScore": 1,
      "boFormat": 5,
      "status": "finished",
      "currentGameNum": 4,
      "createdAt": "2026-10-06T19:17:36+08:00",
      "games": [
        { "id": 1, "gameNumber": 1, "winner": "blue", "bpCompleted": true, "banPickActions": [] }
      ]
    }
  ]
}
```

| 字段 | 说明 |
|------|------|
| `id` | 系列赛 ID，可用于 `POST /api/matches/:id/resume` 恢复对局 |
| `teamAId` / `teamBId` | **固定队伍身份**，与红蓝归属无关（换边不变） |
| `teamAScore` / `teamBScore` | 大比分（按队伍身份记账；红蓝视图见房间快照） |
| `boFormat` | 赛制：5 / 7 / 0 |
| `status` | 系列赛状态：`ongoing` / `finished`；**`ongoing` 才可恢复** |
| `currentGameNum` | 当前进行到第几局 |
| `games[]` | 该系列赛每局明细；**每局的 `blueTeamId` / `redTeamId` 才是该局真实红蓝归属** |

> ⚠️ **红蓝归属是"每局"概念**：第 2 局起可能换边（单机由用户选、联机由败方选），
> 因此展示每局阵容/禁用/胜方时，必须用**该局**的 `game.blueTeamId` / `game.redTeamId`
> 解析真实队名，不能沿用头部的 `teamAId` / `teamBId`。

> 历史页即以此接口一次取回"系列赛 + 每局 BP"，避免前端二次拼接。

---

## WebSocket 协议

连接地址：`ws://localhost:8080/ws?roomId=7f4a2c9e`

### 通道定位（命令/事件分离）

自单机交互方案改造后，WS 定位为**纯下行状态推送通道**：

- **写路径唯一走 REST**：BP 动作只通过 `POST /api/rooms/:id/ban|pick` 提交，HTTP 响应即权威状态快照；
- **WS 只负责扩散**：服务端在每次动作执行后向房间内所有客户端广播完整状态快照（含发起者），供观察者与断线重连场景使用；
- **幂等合并**：快照以 `currentTurn` 为版本号，客户端本地比较后丢弃不领先的快照，消息乱序或丢失不会污染状态。

### 客户端 → 服务端消息

> 协议保留、当前前端单机模式不再上行发送（能力预留给未来双人模式）。

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
    "side": "red",
    "blueBanned": [12, 34],
    "redBanned": [45, 67],
    "bluePicked": [5],
    "redPicked": [],
    "heroes": [ { "id": 1, "status": "available" } ]
  }
}

{ "type": "error", "data": { "code": 40003, "message": "英雄5已被蓝方选择，不可pick" } }
```

### 消息类型

| type | 方向 | 说明 |
|------|------|------|
| `ban` / `pick` | 客户端→服务端 | 执行 BP 操作（协议保留；单机模式前端不使用） |
| `bp_update` | 服务端→客户端 | 广播最新 BP 状态快照（完整推送，客户端按 `currentTurn` 幂等合并） |
| `error` | 服务端→客户端 | 操作失败，携带错误码和原因 |
| `bp_finished` | 服务端→客户端 | BP 全部完成 |