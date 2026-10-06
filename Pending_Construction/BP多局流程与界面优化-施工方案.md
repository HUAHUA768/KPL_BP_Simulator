# BP 多局流程与界面优化 — 施工方案

> 📅 制定日期：2026-10-06
> 🎯 状态：**待施工**（本文档仅为方案，尚未动代码）
> 📌 上游文档：[BP单机交互方案决策.md](BP单机交互方案决策.md)

---

## 一、背景与目标

实战测试中暴露了 3+1 个问题，本方案一次性解决：

| # | 问题 | 严重度 |
|---|------|--------|
| 1 | 英雄被 ban / pick 后，英雄池里的卡片区分度不够（只有边框变色+角标） | 体验 |
| 2 | BP 棋盘三行布局（蓝ban / 红ban / 双方pick）不符合直觉，想改成蓝方在左、红方在右的对称两列 | 体验 |
| 3 | 第一局 BP 完成后只显示"BP 完成"，**无法录入胜负、无法进入下一局** | 🔴 流程断裂 |
| 3' | 退出到首页查历史记录，**没有任何记录** | 🔴 数据丢失 |

### 已与需求方确认的设计决策

| 决策点 | 结论 |
|--------|------|
| 新布局形态 | 左右两列：左列=蓝方（上排 5 个 ban 槽 + 下排 5 个 pick 槽），右列=红方镜像对称，英雄池仍在下方 |
| 系列赛制 | 创建房间时可选 **BO5 / BO7 / 无限制**（默认 BO5）；一方先达到所需胜场自动结束比赛 |
| 胜负录入 | BP 完成后弹出面板，**手动点选**「🔵 蓝方胜 / 🔴 红方胜」，提交后才允许进入下一局 |
| 全局BP记忆 | **启用**，第 2 局起生效：上一局某方选过的英雄，下一局该方不可再选（对面可选）。后端 `GameService.applyGlobalBPMemory` 已实现，只差接线 |
| 历史页增强 | 顺带增强：显示真实队名、按场次（系列赛）分组、显示大比分 |

---

## 二、问题根因分析（施工前必读）

### 2.1 为什么无法进入下一局、也没有历史记录？

一条完整的证据链：

1. **房间只在内存里存在**。`backend/internal/handler/room_handler.go` 的 `CreateRoom` 直接
   `service.NewBPEngineWithPerms(...)` 造引擎塞进 `roomEngines map[string]*Engine`，
   **从未在数据库创建 `match_record` 行，也从未创建 `game` 行**。
2. **BP 完成后无人落库**。Ban/Pick 接口在 `engine.Advance()` 后只做广播；
   `GameService.FinishGame`（负责把 20 步 Actions 写入 `ban_pick_action` 表、
   标记 `bp_completed`）**从未被任何代码调用**。数据库验证：
   `game`、`ban_pick_action`、`match_record` 三张表全部 0 行。
3. **没有"胜负"和"下一局"的接口**。后端只有 ban/pick/status 三类房间接口，
   自然没有前端的按钮可点。
4. **历史接口本身也有 bug**。`game_handler.go` 的 `GET /api/games`：
   不带 `matchId` 参数时直接返回空数组（见源码 `if matchIDStr != ""` 分支），
   而 `HistoryPage` 恰恰是不带参数调用的 —— 即使有数据也查不出来。

结论：**`GameService`（多局编排 + 全局BP记忆 + 落库）整个是"写好了没接线"的状态**。
本次施工的核心就是把它接进房间生命周期。

### 2.2 附带发现

- 前端 `services/api.ts` 的 `getTeams()` 返回**硬编码**战队列表，且与数据库
  `team` 表不一致（如 DB id=1 是"重庆狼队"，硬编码写成"AG 超玩会"）。
  历史页/房间页要显示队名，必须新增 `GET /api/teams` 走真实数据。
- `bpStore.updateBPState` 用 `currentTurn` 做版本号防乱序。进入下一局后
  `currentTurn` 从 21 重置为 1，新快照会被当成"过期数据"丢弃 →
  **必须改成 (gameNumber, currentTurn) 双因子版本号**，否则下一局界面不刷新。
- `team` 表有 18 支队伍（含 SYG、WST 两支非 KPL 队），前端目前写死 16 支。
  改用接口后自动修复。

---

## 三、数据库变更

新增迁移 `backend/migrations/004_match_add_bo_format.sql`：

```sql
USE kpl_bp;
-- 系列赛制：5=BO5(先3胜), 7=BO7(先4胜), 0=无限制(不自动结束)
ALTER TABLE match_record
  ADD COLUMN bo_format TINYINT NOT NULL DEFAULT 5
  COMMENT '赛制: 5=BO5, 7=BO7, 0=无限制';
```

> ⚠️ 施工时对现有库手动执行一次：`mysql -uroot -p < backend/migrations/004_match_add_bo_format.sql`

`match_record` 字段复用约定（不改表结构）：
- `team_a_id` = 蓝方队伍，`team_b_id` = 红方队伍（本模拟器全程不换边）
- 蓝方赢一局 → `team_a_score + 1`；红方赢 → `team_b_score + 1`
- `current_game_num` 随"进入下一局"递增
- `status`: `ongoing`（创建时）→ `finished`（决出冠军或手动结束）

---

## 四、后端施工步骤

### 4.1 新增仓储：MatchRepo

文件：`backend/internal/repository/`（新建 `match_repo.go` 接口 + 并入 `mysql_repo.go` 实现）

```go
type MatchRepo interface {
    CreateMatch(m *model.MatchRecord) (int, error)
    GetByID(id int) (*model.MatchRecord, error)
    // 更新比分 / 当前局号 / 状态（一次性）
    UpdateProgress(id, aScore, bScore, currentGameNum int, status string) error
    ListRecent(limit int) ([]*model.MatchRecord, error) // id DESC
}
```

`GameRepo` 追加一个方法（`game_repo.go` + `mysql_repo.go`）：

```go
ListAllGames() ([]*model.Game, error) // 修复 GET /api/games 无参返回空的问题
```

### 4.2 RoomHandler 重构：房间 = 一场系列赛

`backend/internal/handler/room_handler.go`：

**① 用 Room 结构体替换裸引擎 map**（全部字段内存态，随房间生死）：

```go
type Room struct {
    mu         sync.Mutex
    MatchID    int
    GameID     int      // 当前局的 game 行 ID
    GameNumber int      // 当前第几局
    BlueTeamID int
    RedTeamID  int
    BoFormat   int      // 5 / 7 / 0
    BlueScore  int
    RedScore   int
    Winner     *string  // 当前局胜方（提交前为 nil）
    MatchOver  bool
    Engine     *service.Engine
}
// roomEngines map[string]*Room，外层再加一把 map 级锁
```

**② 注入 GameService**。`NewRoomHandler` 增加参数；`main.go` 里构造
`service.NewGameService(heroRepo, gameRepo, tournamentBanned)` 传入。

**③ CreateRoom 新流程**（请求体新增可选 `boFormat`，默认 5，仅接受 5/7/0）：

```
校验队伍 → 生成 roomId
→ matchRepo.CreateMatch(team_a=蓝, team_b=红, status=ongoing, bo_format)
→ gameRepo.CreateGame(match_id, game_number=1, 蓝/红队)
→ engine = gameService.StartGame(matchID, 1, 蓝, 红); engine.Start()
→ 组装 Room 存入 map
→ 返回 {roomId, status, matchId, gameNumber, boFormat}
```

**④ Ban/Pick 完成时落库**：`engine.Advance()` 后若 `engine.IsFinished()`：

```go
gameService.FinishGame(room.GameID, engine, nil) // 20步Actions落库 + bp_completed=1
```

（引擎完成后 `ExecuteAction` 会拒绝后续操作，天然防重；仍建议 Room 加
`persisted bool` 双保险。）

**⑤ 新接口：提交本局胜方** `POST /api/rooms/:id/result`

```
body: {"winner": "blue" | "red"}
前置: 引擎已完成 && room.Winner == nil && !MatchOver，否则 400
动作:
  gameRepo.UpdateGameBPCompleted(gameID, &winner)  // 幂等，写入 winner 列
  对应方 score+1；winsNeeded = BoFormat/2 + 1（BoFormat=0 时永不自动结束）
  若达到 winsNeeded → MatchOver=true，matchRepo.UpdateProgress(status=finished)
广播: bp_update（携带 winner/比分/matchStatus）
返回: 完整房间状态
```

**⑥ 新接口：进入下一局** `POST /api/rooms/:id/next-game`

```
前置: 引擎已完成 && Winner != nil && !MatchOver，否则 400（文案区分"还没选胜方"/"比赛已结束"）
动作:
  gameNumber++
  gameRepo.CreateGame(新一局)
  engine = gameService.StartGame(matchID, gameNumber, 蓝, 红)  // ★ 自动读取上一局
                                                                //   ban_pick_action 应用全局BP记忆
  engine.Start()
  room.Engine/GameID/GameNumber 换新；matchRepo.UpdateProgress(current_game_num=...)
广播 + 返回新状态（currentTurn=1，四个列表全空）
```

**⑦ 新接口：手动结束比赛** `POST /api/rooms/:id/finish-match`
（无限制赛制专用；设置 MatchOver、match status=finished，广播）

**⑧ buildRoomState 扩展字段**（前端契约，见 §六）：
新增 `matchId`、`gameNumber`、`boFormat`、`blueScore`、`redScore`、
`winner`（可为 null）、`matchStatus`。

### 4.3 GameHandler 扩展

`game_handler.go`：
- 修复 `ListGames`：无 `matchId` 时调 `ListAllGames()`（按 match_id DESC, game_number ASC）。
- 新增 `ListMatches` → **`GET /api/matches`**，返回（最近 50 场）：

```json
[{
  "id": 1, "teamAId": 1, "teamBId": 2,
  "teamAScore": 3, "teamBScore": 1,
  "boFormat": 5, "status": "finished", "currentGameNum": 4,
  "createdAt": "...",
  "games": [ { ...同 /api/games 元素，含 banPickActions... } ]
}]
```

新接口 `GET /api/teams`（`teamRepo.ListAll()` 已有，只差 handler 和路由）：
返回 `[{id, name, logoUrl}]`，替代前端硬编码。

### 4.4 路由汇总（main.go 新增注册）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/rooms | 创建房间（+boFormat） |
| POST | /api/rooms/:id/result | 提交本局胜方 |
| POST | /api/rooms/:id/next-game | 进入下一局（触发全局BP记忆） |
| POST | /api/rooms/:id/finish-match | 手动结束比赛（无限制赛制） |
| GET | /api/teams | 战队列表 |
| GET | /api/matches | 系列赛历史（含每局BP） |
| GET | /api/games | 修复无参返回空 |

---

## 五、前端施工步骤

### 5.1 问题1：已用英雄视觉强化

`components/HeroCard.tsx`：

- `owner` 非空时，图标容器追加 `grayscale brightness-50`（CSS filter 作用于
  整个图标渲染），形成"彩色=可选，灰色=已用"的强对比。
- 被 **ban** 的英雄额外叠加红色大 `✕` 覆盖层（`absolute inset-0 flex
  items-center justify-center`，红色粗体 + drop-shadow），保留现有「禁」角标。
- 被 **pick** 的英雄保留蓝/红边框与「蓝」/「红」角标，仅灰化图标。
- 移除旧的 `opacity-40/70` 叠加（与灰化重复且让角标发糊）；`blocked`
  （非本方回合）语义不变。

效果对照：可用=全彩 ｜ 被ban=灰图+红✕ ｜ 被pick=灰图+阵营色边框角标。

### 5.2 问题2：棋盘布局重构（左右两列）

`components/BanPickBoard.tsx`：

```
┌────────────────────────────────────────────────────────────┐
│                    状态横幅（回合方主题色）                    │
├──────────────────────────────┬─────────────────────────────┤
│  🔵 蓝方                      │                 红方 🔴      │
│  禁用  [槽][槽][槽][槽][槽]     │     [槽][槽][槽][槽][槽] 禁用 │
│  选择  [槽][槽][槽][槽][槽]     │     [槽][槽][槽][槽][槽] 选择 │
├──────────────────────────────┴─────────────────────────────┤
│                 英雄池（每行 10 个，不变）                     │
└────────────────────────────────────────────────────────────┘
```

要点：
- 删除原"🚫 已禁用 / ✅ 已选择"两个独立区块，合并为两个阵营卡片
  （`grid gap-3 md:grid-cols-2`，蓝卡片左、红卡片右，移动端纵向堆叠）。
- **槽位数量固定**：每方 ban 恒 5 槽、pick 恒 5 槽（由 20 步序列表决定：
  双方各 ban 5、各 pick 5），不再用 `banSlotCount` 动态计算 —— 布局从头到尾
  零跳动。
- 槽位闪烁逻辑（`countConsecutiveSameSlots` / `blinkFor`）原样保留，
  只是渲染位置挪进新卡片；红列做镜像排版（标签靠右、槽位靠右）。
- `HeroSlot` / `EmptySlot` 组件不改。

### 5.3 问题3：BP 完成 → 录入胜负 → 下一局

`pages/BPRoom.tsx` 用新的"局间结算面板"替换现有绿色横幅：

```
状态 finished 且 winner == null:
  ┌──────────────────────────────────────────┐
  │ 🎉 第 N 局 BP 完成！双方阵容已确定            │
  │ 请选择本局胜方：                              │
  │   [ 🔵 {蓝队名} 获胜 ]   [ 🔴 {红队名} 获胜 ]  │
  └──────────────────────────────────────────┘
状态 finished 且 winner != null 且比赛未结束:
  ┌──────────────────────────────────────────┐
  │ ✅ 第 N 局结束：{胜方队名} 获胜                │
  │ 大比分  蓝 {X} : {Y} 红    （BO{赛制}）        │
  │ 提示：下一局将启用全局BP规则——本局双方选用过的    │
  │      英雄，下一局各自不可再选                   │
  │   [ ➡️ 开始第 N+1 局 BP ]   [ 结束比赛 ]*      │
  └──────────────────────────────────────────┘
  *「结束比赛」仅在无限制赛制显示
比赛结束:
  ┌──────────────────────────────────────────┐
  │ 🏆 比赛结束！{冠军队名} 以 {X}:{Y} 赢得系列赛    │
  │   [ 返回首页 ]   [ 查看历史记录 ]              │
  └──────────────────────────────────────────┘
```

- 按钮分别调用新 API：`submitGameResult(roomId, winner)` /
  `nextGame(roomId)` / `finishMatch(roomId)`，响应直接 `updateBPState`。
- 队名来源：进房时请求 `GET /api/teams` 建立 `id→name` 映射。

### 5.4 状态层改造（关键坑位）

`types/index.ts`：`BPState` 增加
`matchId / gameNumber / boFormat / blueScore / redScore / winner('blue'|'red'|null) / matchStatus('ongoing'|'finished')`；
新增 `MatchRecord` 类型（含 `games: Game[]`）。

`store/bpStore.ts`：**版本号修复**——

```ts
// 旧：if (sameRoom && incoming.currentTurn < state.currentTurn) return state
// 新：先比 gameNumber，再比 currentTurn
const v = (g: number, t: number) => g * 100 + t
if (sameRoom && v(incoming.gameNumber, incoming.currentTurn) < v(state.gameNumber, state.currentTurn))
  return state
```

并把 7 个新字段纳入 store 与 `initialState`（`gameNumber` 初始 1）。

`services/api.ts`：
- `getTeams()` 改为真实请求 `GET /api/teams`（删掉硬编码数组）。
- 新增 `submitGameResult / nextGame / finishMatch / getMatches`。
- `createRoom(blueTeamId, redTeamId, boFormat)` 透传赛制。

### 5.5 首页与历史页

`pages/HomePage.tsx`：队伍选择器上方新增赛制选择
（BO5 / BO7 / 无限制，默认 BO5，胶囊按钮样式与分路筛选一致）。

`pages/HistoryPage.tsx` + `components/GameResultTable.tsx`：
- 数据源改为 `getMatches()` + `getTeams()`。
- 每场系列赛一张卡片：头部
  `🔵{蓝队名} {X} : {Y} {红队名}🔴 · BO{赛制} · 进行中/已结束`，
  内部复用现有表格渲染每局（蓝方/红方列可省略队名重复，保留阵容/禁用/胜方）。
- 空态文案保留「暂无对局记录」。

---

## 六、房间状态 JSON 契约（改动后全量）

```json
{
  "roomId": "a1b2c3d4",
  "status": "ongoing | finished",
  "matchId": 1,
  "gameNumber": 2,
  "boFormat": 5,
  "blueTeamId": 1, "redTeamId": 2,
  "blueScore": 1, "redScore": 0,
  "winner": null,
  "matchStatus": "ongoing",
  "currentTurn": 5, "round": 1,
  "action": "pick", "side": "blue",
  "blueBanned": [11, 23], "redBanned": [45, 67],
  "bluePicked": [8], "redPicked": [],
  "heroes": [{"id": 1, "status": "available"}]
}
```

`winner` 语义：**当前局**已提交的胜方；进入下一局后重置为 null。

---

## 七、施工顺序（建议分 4 个 commit）

| 步骤 | 内容 | 涉及文件 |
|------|------|----------|
| 1 | 数据库迁移 + 仓储层 | `migrations/004_*.sql`、`match_repo.go`(新)、`game_repo.go`、`mysql_repo.go` |
| 2 | 后端房间生命周期接线 | `room_handler.go`、`game_handler.go`(teams/matches)、`main.go` |
| 3 | 前端状态层 + API + 类型 | `types/index.ts`、`bpStore.ts`、`api.ts` |
| 4 | 前端界面（布局/灰化/结算面板/首页/历史页） | `BanPickBoard.tsx`、`HeroCard.tsx`、`BPRoom.tsx`、`HomePage.tsx`、`HistoryPage.tsx`、`GameResultTable.tsx` |

每步完成后先编译/构建验证再进入下一步；全部完成后执行 §八 验收。

---

## 八、验收清单（施工后逐项打勾）

### 构建

- [ ] `cd backend && go build ./cmd/server/` 无错误，`go test ./internal/service/` 通过
- [ ] `cd frontend && npm run build` 无 TS 错误
- [ ] 执行 004 迁移成功；重启后端

### 问题1 视觉

- [ ] ban 一个英雄 → 英雄池内该卡片灰化 + 红✕ + 「禁」角标，老远可辨
- [ ] pick 一个英雄 → 灰化 + 阵营色边框角标；可用英雄保持全彩

### 问题2 布局

- [ ] 棋盘为左右两列：左蓝（上ban下pick）、右红镜像；槽位各 5+5 固定不跳动
- [ ] 当前回合方的目标槽位仍在正确位置闪烁（含第 18→19 步蓝方双 pick 连闪）
- [ ] 窄屏（移动端宽度）下两列纵向堆叠不破版

### 问题3 多局流程（核心）

- [ ] 创建房间（选 BO5）→ 第 1 局正常走完 20 步 → 显示"第 1 局 BP 完成"结算面板
- [ ] 点击「🔴 红方胜」→ 显示比分 0:1，出现「开始第 2 局 BP」按钮
- [ ] 点「开始第 2 局」→ 棋盘重置为空，**第 1 局双方 pick 过的 10 个英雄**：
      原蓝方不可再选（点击报"受全局BP规则限制"）、原红方同理；对方仍可选
- [ ] `SELECT COUNT(*) FROM ban_pick_action;` = 20（第 1 局已落库）
- [ ] 打到某方 3 胜 → 自动显示"🏆 比赛结束"，`match_record.status='finished'`
- [ ] 比赛结束后「下一局」按钮不可用（接口返回 400 文案正确）
- [ ] 无限制赛制：不自动结束，「结束比赛」按钮可用且生效

### 历史记录

- [ ] 返回首页 → 查看历史对局：显示刚才的系列赛卡片（真实队名、大比分、每局阵容/禁用/胜方）
- [ ] 刷新页面后记录仍在（已持久化）
- [ ] `GET /api/games`（无参）返回非空

### 回归

- [ ] 英雄池分路筛选、回合横幅主题色切换、错误提示关闭等既有功能正常
- [ ] 两个浏览器窗口同时观战同一房间，WS 推送同步（含结算/下一局事件）

---

## 九、风险与边界说明

1. **房间状态纯内存**：后端重启会丢失进行中的房间（引擎状态无处恢复），
   但已完成的对局记录在库里不受影响。这是既有架构限制，本次不解决，
   如需断线续赛需另立方案（引擎快照落库/Redis）。
2. **并发安全**：Room 内部加锁保证 result/next-game/ban/pick 不会互踩；
   单人单机场景压力小，但锁必须先于任何字段读写获取。
3. **全局BP记忆的界面提示**：第 2 局起受限英雄在英雄池里**暂不预先标记**，
   点击时由后端报错文案提示（现有错误横幅机制）。若后续需要"提前灰显/锁形标记"，
   需扩展 `buildRoomState` 输出每个英雄对当前回合方的权限，列为后续优化。
4. **赛制枚举**：仅接受 5/7/0，其余值一律 400，防止前端传脏数据。
5. **前端版本号修复是隐蔽关键点**：漏改 `bpStore` 会导致"点了下一局界面不动"，
   验收时重点观察。

---

## 十、交付物

- 代码：按 §七 四个 commit 提交
- 数据库：迁移 004
- 文档：本方案已在 [README.md](README.md) 目录表登记；施工完成后把状态
  改为"已完工"，并回填验收结果
