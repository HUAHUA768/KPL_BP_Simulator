# 下一版本实施清单 — 单机 BP 交互改造

> **给执行者**：本清单是唯一任务来源，按步骤顺序做，每步做完先自测再进入下一步。所有设计决策已定稿，**不要重新讨论方案**；如遇清单未覆盖的细节，以 `learn-as-you-go/BP单机交互方案决策.md` 为准。
> **状态**：纯计划文档，尚未实施。行号基于当前代码快照，动手前以实际代码为准。

---

## 〇、开工前必读（按顺序）

| 顺序 | 文档 | 读什么 |
|------|------|--------|
| 1 | `learn-as-you-go/BP单机交互方案决策.md` | **本次任务的完整规格**：四个决策点、改动清单、验收标准（§五必须逐条对照） |
| 2 | `实现思路.md` §四「BP 规则详解」 | 20 步顺序表与先手规则——槽位闪烁算法的依据 |
| 3 | `docs/API.md`「WebSocket 协议」章节 | 目标通信模型：命令走 REST、WS 纯下行、幂等合并 |
| 4 | `backend/internal/service/bp_engine.go` | 只需通读 `getTurnInfo` / `ExecuteAction` / `Advance`，确认引擎无请求者校验（后端零改动的依据） |
| 5 | 前端四件套 | `frontend/src/pages/BPRoom.tsx`、`components/BanPickBoard.tsx`、`hooks/useWebSocket.ts`、`store/bpStore.ts` |

选读（不阻塞开工）：`learn-as-you-go/BP交互设计-面试版.md`。

**环境启动**：`source scripts/env.sh` → 后端 `cd backend && go build ./cmd/server/ && ./server` → 前端 `cd frontend && npm run dev`。MySQL 密码见 `scripts/env.sh`（不存在则从 `.example` 复制）。

---

## 一、现状速览（省得你满仓库找）

- 卡点源头：`BPRoom.tsx:18` `mySide = searchParams.get('side') || 'blue'` → 传给 `BanPickBoard` → `BanPickBoard.tsx:119` `isMyTurnNow = side === mySide && ...` 把红方回合的点击全部吞掉。
- 双写通道：`BPRoom.tsx` 的 `handleBanHero/handlePickHero` 先 `await banHero/pickHero(...)`（REST，返回**完整 BPState**），又调 `wsBan/wsPick(hero.id)` 上行一遍；后端 Ban/Pick handler 既 `broadcastUpdate`（WS 广播全量快照）又在 HTTP 响应里返回同一份快照——一次点击两次状态写入 + 一次冗余上行。
- 好消息：HTTP 响应和 WS 广播用的是同一个 `buildRoomState()`，形状一致（含 `currentTurn`），幂等合并有现成版本号可用。
- 引擎侧：`ExecuteAction` 只校验动作类型和英雄合法性，回合归属由 `getTurnInfo(CurrentTurn)` 决定，**天然兼容单机，后端一行不用改**。
- 槽位现状：ban 区两行（蓝/红各一行，行数取双方较长者，`banSlotCount`）；pick 区固定每方 5 个空位（`PICKS_PER_SIDE`）。闪烁要作用在"接下来将被填充的空槽位"上。

---

## 二、改动步骤

### Step 1 — 新建 `frontend/src/utils/actor.ts`（双人扩展接缝）

```ts
export type Mode = 'solo' | 'dual'
/** 谁是有效操作方：solo = 引擎当前回合方；dual = URL 声明方 */
export function resolveActor(stateSide: string, mode: Mode, urlSide: string): string
export function parseMode(search: string): Mode   // 解析 ?mode=，非法值回退 solo
```

- solo：返回 `stateSide`（门控恒通过）；dual：返回 `urlSide`（保留旧等待语义）。
- 纯函数、零依赖，这是全项目唯一允许出现"操作方判定"逻辑的地方。

### Step 2 — `bpStore.ts` 增加模式字段

- state 加 `mode: Mode`（默认 `'solo'`），action 加 `setMode`。
- `updateBPState` 改为**幂等入口**：签名不变，但内部用 zustand 的 `(state) => ...` 形式，比较 `incoming.currentTurn >= 本地 currentTurn` 才应用（相等也应用，保证 REST 自己写的权威数据不被丢弃；更旧直接 return partial=false）。
- 不动其他字段结构。

### Step 3 — `useWebSocket.ts` 降级为纯下行

- 删除 `ban` / `pick` 两个 useCallback 及返回值（`services/ws.ts` 里的 `sendBan/sendPick` 一并删除或注释标记 deprecated——协议保留靠 API.md 说明，不靠死代码）。
- `connectWS(roomId, side)` 的 `side` 参数传 `"spectator"`（后端 `WebSocketHandler` 对空/未知 side 已兜底 spectator，无需改后端）。
- 消息处理不变（`bp_update`/`bp_finished` → `updateBPState`），幂等已由 Step 2 的 store 兜住。
- 顺手修一个现存隐患：`useEffect` 依赖里有 `updateBPState`，zustand selector 返回的函数引用稳定，保留即可，但要确认断线后没有自动重连逻辑——**本期不加**（范围外）。

### Step 4 — `BPRoom.tsx` 主接线

1. 解析 `mode`（`parseMode(searchParams.toString())`），写入 store。
2. 计算 `actorSide = resolveActor(bpState.side, mode, mySide)`，传给 Board 的 `mySide` prop 改名或直接传 `actorSide`。
3. `handleBanHero/handlePickHero`：**删掉 `wsBan(hero.id)` / `wsPick(hero.id)` 调用**，只保留 REST + `updateBPState(state)`。
4. 顶栏文案：solo 模式下显示 `当前扮演: 🔴 红方`（跟随 actorSide），dual 模式保持原文案"你: X方"。
5. `HomePage.tsx:29` 跳转 `/room/${id}?side=blue`：改为不带 `side`（solo 不需要；dual 用户手动加参数即可）。

### Step 5 — `BanPickBoard.tsx` 门控 + 换边视觉

**门控**：
- `isMyTurnNow = actorSide === side && (action === 'ban' || action === 'pick')`——solo 下恒真，dual 下即旧行为。组件不再自行判断身份，只接收算好的 `actorSide`。

**主题色切换**（定义一个 tone 对象，随 `side` 整页切换强调色）：
- 状态横幅：蓝回合蓝色系渐变背景/边框，红回合红色系；文字升级为"🎭 当前扮演：🔵 蓝方 · 禁用第 N 手"。
- 棋盘外围容器加一层随回合变化的 glow/边框色（Tailwind 条件 class 即可，别引新库）。

**槽位闪烁（核心动画）**：
- 闪烁目标 = **当前方的下一个空槽位**：
  - ban 回合：该方 ban 行的下一个空位。注意连续同方多步的场景（如 17→20 的红 pick4/蓝 pick4/蓝 pick5/红 pick5）——当某方在本步之后**紧接着还有一步同方同类型**时，对应的**两个空位同时闪**（用户明确要求过这一点）。通用化：向后看 `getTurnInfo` 序列，连续的"同 side 同 action"步数 k → 闪该方接下来 k 个空槽位。k 的计算写成 Board 内一个小纯函数 `countConsecutiveSameSlots(side, action, currentTurn)`，内置一张 20 步 `[side, action]` 序列表（与后端 `bp_engine.go` 的表**逐行核对**抄过来，这是唯一事实来源风险点，做错测试会暴露）。
  - pick 回合：同理闪当前方阵容区的下 k 个空位。
- 动画实现：`index.css` 里加一组 `@keyframes slot-blink`（border-color + box-shadow 呼吸，周期 ~1.2s），Tailwind 任意值或自定义 class `slot-blinking` 挂到命中的 `EmptySlot` 上；`EmptySlot` 加可选 prop `blinking?: boolean` 与 `tone?: 'blue' | 'red'`。**动作完成后快照推进、槽位被 HeroSlot 取代，动画自然消失**——不需要任何"停止动画"的逻辑，这是这个设计的自洽性所在。
- 尊重系统减弱动画偏好：`@media (prefers-reduced-motion: reduce)` 下关闭 keyframes。

### Step 6 — 验证（对照决策文档 §五逐条勾）

1. `cd frontend && npx tsc --noEmit` 通过。
2. 起前后端，浏览器跑完整一局 20 步：全程无卡顿、无需改 URL；重点盯第 17–20 步（红 pick4 → 蓝 pick4+pick5 两位同闪 → 红 pick5）。
3. DevTools Network/WS 面板：每次点击恰好 1 个 REST 请求、**0 个 WS 上行帧**；WS 下行快照到达后 UI 不抖动、不回退（幂等生效）。
4. 换边观感：蓝↔红回合主题色肉眼可辨；闪烁只出现在当前方空槽位。
5. 回归 dual：访问 `?mode=dual&side=blue`，红方回合应不可点击并显示"等待对方操作"——证明接缝存在且没污染 solo。
6. 回归历史：HistoryPage 正常（本次不应触碰它）。

**禁止事项**：不改后端；不加 AI/防误触/跨窗口同步；不重构英雄池渲染；不引入新依赖。

---

## 三、给执行者的开场提示词（复制到新对话直接用）

> 你是本仓库的实施工程师。任务：按 `learn-as-you-go/BP单机交互改造-实施清单.md` 完成单机 BP 交互改造（一人分饰两角 + 通信通道收敛）。
>
> 开工前先完整阅读该清单，并按其中"〇、开工前必读"的顺序读完四份文档和五个前端文件；然后严格按 Step 1→6 顺序执行，不做清单外的任何事（尤其：不改后端、不加 AI 代打、不引新依赖）。
>
> 三条铁律：
> 1. 决策已全部定稿，遇到疑问以 `learn-as-you-go/BP单机交互方案决策.md` 为准，不要重新设计方案；
> 2. Step 5 里抄 20 步 `[side, action]` 序列表时，必须打开 `backend/internal/service/bp_engine.go` 的 `getTurnInfo` 逐行核对，不许凭记忆默写；
> 3. 每个 Step 完成后先跑 `npx tsc --noEmit` 再继续；Step 6 的 6 项验证逐项给出结果，任何一项不过就回头修，修完重跑全表。
>
> 全部通过后：把实施清单中每一步标注"✅ 已完成"，更新 `README.md` 的"单机操作方式"一节（若与实际行为有出入），并汇报改动文件清单。
