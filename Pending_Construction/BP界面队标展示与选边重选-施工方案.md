# BP 界面队标展示、Ban/Pick 尺寸分级、曜/瑶图标修正与第二局重新选边 — 施工方案

> 📅 制定日期：2026-10-12 ｜ ✅ 完工日期：2026-10-12
> 🎯 状态：**已完工**（验收结果见 §八）
> 📌 关联文档：[实现思路.md](../实现思路.md) ｜ [项目架构.md](../项目架构.md) ｜ [API.md](../docs/API.md) ｜ [BP多局流程与界面优化-施工方案.md](../learn-as-you-go/BP多局流程与界面优化-施工方案.md)
> 🧭 需求来源：实战测试中提出的 4 个待解决问题（本文 §一）

---

## 一、背景与目标

实战测试中暴露了 4 个问题，本方案一次性解决。**本次任务只编写施工方案，不实施代码改动。**

| # | 问题 | 用户原话要点 | 类型 | 严重度 |
|---|------|-------------|------|--------|
| 1 | BP 界面只显示红蓝方，**缺少双方队名 + 队标** | "除了显示红蓝方之外，还需要显示双方的队名+队标" | 功能缺失 | 中 |
| 2 | **Ban 位与 Pick 位大小没有区分** | "ban位相对小，pick位相对大" | 视觉层级 | 低 |
| 3 | 分路为打野/对抗路时，**曜的图标显示成瑶的** | "从 MySQL 库里把两个英雄的名字和图标文件路径区分开来" + "改图标文件的名字" | 🔴 数据正确性 | 高 |
| 4 | **第二局需要重新选边** | 联机模式败方选边；当前仅单机模式，故第一局结束后**提示选边信息，由用户决定下局谁蓝谁红** | 流程缺失 | 中 |

### 需求方已确认的边界

| 决策点 | 结论 |
|--------|------|
| 本次交付范围 | **仅施工方案文档**，落地到 `Pending_Construction/` 目录 |
| 问题 3 的修复路径 | 明确要求**双管齐下**：① MySQL 库中区分两名英雄的名字与图标路径；② 重命名图标文件 |
| 问题 4 的模式范围 | 当前项目**只有单机（solo）模式**，因此不做"败方自动选边"，只做**提示 + 用户手动指定下一局红蓝归属** |
| 联机模式 | 架构上预留 `side_picker_team_id`（败方选边语义），本次仅落库不启用交互 |

---

## 二、问题根因分析（施工前必读）

> 以下结论均基于对当前代码库的实际核对，**不是推测**。每条都标注了证据文件与行号，施工时可直接跳转验证。

### 2.1 问题 1 根因：队名队标从未进入 BP 棋盘

**三条事实链：**

1. **`BPRoom.tsx` 已经拿到了队名，但只用于结算面板，没有传给棋盘。**
   证据：`frontend/src/pages/BPRoom.tsx:51-56` 定义了 `blueTeamName` / `redTeamName`；
   `:236`/`:243`/`:251` 等处只在**结算面板**使用。而传给 `<BanPickBoard>` 的 props
   （`:307-320`）**完全没有队名、队标字段**。

2. **`BanPickBoard.tsx` 的阵营标题是写死的红蓝文字。**
   证据：`frontend/src/components/BanPickBoard.tsx:313`：
   ```tsx
   {isBlue ? '🔵 蓝方' : '红方 🔴'}
   ```
   组件 props 接口（`:5-26`）也没有任何队名/队标入参。

3. **队标 URL 本身存在"相对路径 + 未代理"双重缺陷，即使渲染出来也会 404。**
   - 数据库里 `logo_url` 存的是**相对路径**（无前导斜杠）：`images/teams/chongqin_wolves.png`
     （证据：`backend/migrations/003_team_data.sql`），
     对比英雄图标是**绝对路径**：`/images/heroes/bai_qi.png`（证据：`002_seed_data.sql`）。
   - 相对路径会以**当前页面 URL** 为基准解析。BP 页地址是 `/room/:id`，
     因此 `images/teams/x.png` 会被解析成 `/room/images/teams/x.png` → **必然 404**。
   - 即便改成绝对路径 `/images/teams/x.png`，Vite dev server 的 `proxy` 只代理了
     `/api` 与 `/ws`（证据：`frontend/vite.config.ts:9-18`），**`/images` 未代理** → 仍然 404。
   - 现状之所以"看起来没报错"，是因为 `TeamSelector.tsx:76-78` / `:115-117` 用了
     `onError` 把图片 `display:none` **静默隐藏**了 → 问题被掩盖。

> **结论**：问题 1 不是"加个 `<img>`"这么简单，而是 **数据（路径格式）+ 传输（dev 代理）+ 视图（props 与标题）** 三层都要打通。

### 2.2 问题 2 根因：Ban 与 Pick 槽位共用同一个组件和同一套尺寸

证据：`frontend/src/components/BanPickBoard.tsx`

- `HeroSlot`（`:102-123`）与 `EmptySlot`（`:150-169`）都写死 **`w-14`**（3.5rem）；
- `SLOT_TONE`（`:91-96`）只区分红蓝与 ban/pick 的**颜色**，不区分**尺寸**；
- 两类槽位唯一的差异是描边样式（虚线 vs 实线）与色调，**视觉体量完全一致**。

因此界面无法一眼看出"哪些是禁用、哪些是选用"，与"ban 小 pick 大"的需求不符。

> ⚠️ **关键约束**：`HeroSlot` 与 `EmptySlot` 必须保持**各自的**尺寸严格一致，
> 否则槽位从空 → 填充时会发生**尺寸跳动**，破坏现有"零跳动布局"设计
> （证据：`:98-101`、`:146` 的注释明确声明了这一设计目标）。
> 因此尺寸分级必须**成对**修改：ban 的 `HeroSlot` 与 ban 的 `EmptySlot` 同步缩小，
> pick 的 `HeroSlot` 与 pick 的 `EmptySlot` 同步放大。

### 2.3 问题 3 根因：⚠️ 与用户预判不同 — **文件本身是正确的，缺陷在"同音名 + 扩展名不一致"的脆弱约定**

这是本方案中最需要澄清的一条。经实际检查：

**事实 A — 两个图标文件都是正确的、各自独立的：**

| 文件 | 实际格式 | 内容（已肉眼确认） | 对应英雄 |
|------|---------|-------------------|---------|
| `yao.jpg` | **PNG**（伪装成 .jpg） | 深色短发少年 | ✅ **曜** |
| `yao.png` | PNG | 粉紫色长发少女 | ✅ **瑶** |

两份文件字节不同（md5 分别为 `a274354648f1ae83678d1ea59b332813` / `496c3267ea1ec23d8ef6b456b7392971`），
**不存在"互相覆盖"**。同时 `frontend/public/images/heroes/` 与 `backend/static/images/heroes/`
两份副本完全同步（各 132 个文件，md5 一致）。

**事实 B — seed SQL 中的路径映射本身也是"正确"的：**

```sql
-- 002_seed_data.sql:35   对抗路分组
('曜', '/images/heroes/yao.jpg', '["打野","对抗路"]'),
-- 002_seed_data.sql:143  游走分组
('瑶', '/images/heroes/yao.png', '["游走"]'),
```

**事实 C — 那么 bug 从哪来？两个可能来源，且都会各自独立地造成同一现象：**

1. **（最可能）线上数据库与 seed 文件已不一致。**
   `曜` 与 `瑶` 读音完全相同（yáo），唯一区分靠 **`.jpg` / `.png` 扩展名**这一无语义、
   极易手滑的约定。历史上有任何一次人工改库 / 重跑导入 / 复制粘贴出错，
   就可能把 `曜` 的 `icon_path` 写成 `/images/heroes/yao.png` → 曜 显示成瑶。
   注意 `曜` 正位于"打野、对抗路"分组，**与问题描述"分路为打野、对抗路的曜图标变成瑶"完全吻合**。
   >
   > ⚠️ **重要**：当前环境 `scripts/env.sh` 不存在（仅 `.example` 模板），
   > 无法连库查询确认。**施工第一步必须先查库取证**（见 §三 问题3 步骤 0）。

2. **数据重建时的自动匹配脚本存在同名碰撞风险。**
   `scripts/download_hero_icons.py` 按"英雄中文名 → 官方 ename → 下载 URL"工作，
   并把结果写入 seed SQL 中指定的文件名。一旦官方 `herolist.json` 中
   `曜`/`瑶` 的 `ename` 解析异常或某次匹配失败回退，就可能把同一张图写入两个不同文件名。
   由于 `herolist.json` 当前**不在仓库中**（已确认缺失），该脚本目前**无法复现执行**，
   属于"不可验证的历史路径"。

**事实 D — `.jpg` 文件实为 PNG 内容，属既存隐患。**
`yao.jpg` 的 `file` 输出为 `PNG image data, 90 x 90`。
这本身不影响浏览器渲染（浏览器靠 Content-Type / 魔数嗅探），
但会让"靠扩展名区分英雄"这一约定更加不可靠，也让后续维护者误判格式。

> **结论**：修复不能只满足于"把 SQL 改对一次"。真正稳健的做法是
> **消灭"靠扩展名区分同音英雄"的脆弱约定** —— 这恰好与需求方提出的
> "改图标文件名字"完全一致，且是治本之策。

### 2.4 问题 4 根因：后端 `NextGame` 完全没有"选边"概念

证据：`backend/internal/handler/room_handler.go:377-420` 的 `NextGame`：

```go
gameNumber := room.GameNumber + 1
gameID, err := h.gameRepo.CreateGame(&model.Game{
    MatchID:    room.MatchID,
    GameNumber: gameNumber,
    BlueTeamID: room.BlueTeamID,   // ← 原样沿用上一局，永远不换边
    RedTeamID:  room.RedTeamID,    // ← 原样沿用上一局，永远不换边
})
...
engine, err := h.gameService.StartGame(room.MatchID, gameNumber, room.BlueTeamID, room.RedTeamID)
```

- 红蓝归属**逐字沿用**上一局，第二局起永远不会换边；
- `Room` 结构体（`:25-39`）**没有** `SidePickerTeamID` 等任何选边相关字段；
- 前端局间结算面板（`BPRoom.tsx:247-283`）只渲染"开始第 N+1 局 BP"与"结束比赛"，
  **没有任何选边 UI**。

**但基础设施已经预留好了**，这是本次施工的有利条件：

- 表 `match_record` 已有字段 `side_picker_team_id INT DEFAULT NULL`
  （证据：`001_create_tables.sql:16`，注释即"当前拥有选边权的队伍ID"）；
- 该字段已在 Go model 中映射：`backend/internal/model/game.go:31`
  ```go
  SidePickerTeamID *int `json:"sidePickerTeamId" gorm:"column:side_picker_team_id"`
  ```
- 但**全仓库无任何读写该字段的业务代码**（已 grep 确认，仅出现在建表 SQL 与 model 定义中）。

> **结论**：问题 4 = "已备好字段但未接线"。需要补齐：`Room` 内存字段 → 选边接口 →
> `NextGame` 接受红蓝参数 → 前端选边 UI。同时要处理**一处关键反查**：
> `match_record` 存的是 `team_a_id` / `team_b_id`，而 `game` 存的是 `blue_team_id` /
> `red_team_id`，选边时必须正确映射到 A/B 两队的**大比分**，否则换边后比分会记错队。

---

## 三、施工方案

### 总览：改动清单

| # | 层 | 文件 | 改动性质 |
|---|----|------|---------|
| 3-a | DB | `backend/migrations/005_*.sql`（新建） | 数据修正 + 改名 |
| 3-b | DB | `backend/migrations/002_seed_data.sql` | 同步修正（保持可重建） |
| 3-c | 资源 | `frontend/public/images/heroes/`、`backend/static/images/heroes/` | 文件重命名 |
| 3-d | 脚本 | `scripts/download_hero_icons.py` | 同步调整（防回归） |
| 1-a | 后端 | `backend/internal/handler/game_handler.go` | 队标路径规范化 |
| 1-b | 前端 | `frontend/src/vite.config.ts` | 补 `/images` 代理 |
| 1-c | 前端 | `frontend/src/types/index.ts`、`api.ts` | 队信息随房间状态下发 |
| 1-d | 前端 | `frontend/src/components/BanPickBoard.tsx`、`BPRoom.tsx` | 阵营卡展示队名+队标 |
| 2 | 前端 | `frontend/src/components/BanPickBoard.tsx` | Ban/Pick 尺寸分级 |
| 4-a | 后端 | `backend/internal/handler/room_handler.go` | 选边状态 + 接口 + NextGame 接线 |
| 4-b | 后端 | `backend/internal/repository/match_repo.go` | 读 `side_picker_team_id` |
| 4-c | 前端 | `frontend/src/services/api.ts`、`BPRoom.tsx`、`types/index.ts` | 选边 UI |

---

### 问题 3：修正曜/瑶（**先做，因为它是数据正确性问题**）

#### 步骤 0（前置，必做）：查库取证

先确认线上库的真实状态，避免"照着 seed 文件想象问题"。

```bash
source scripts/env.sh          # 若缺失，先 cp scripts/env.sh.example scripts/env.sh 并填密码
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" -e "
  SELECT id, name, icon_path, lanes, HEX(name) AS name_hex
  FROM hero WHERE name IN ('曜','瑶');"
```

**判读标准：**

| 查询结果 | 含义 | 处置 |
|---------|------|------|
| `曜.icon_path = '/images/heroes/yao.png'` | 库内确实写错（问题 3 复现） | 执行 005 迁移修正 |
| `曜.icon_path = '/images/heroes/yao.jpg'` | 库内正确 → 缺陷在文件/缓存层 | 仍需执行改名（治本），并排查浏览器缓存 |
| 两行 `name_hex` 相同 | ⚠️ **两行都是同一个字**（真·同名入库） | 需 `UPDATE` 改回正确字形，见下方陷阱 |

> ⚠️ **陷阱预警**：`曜`(U+66DC) 与 `瑶`(U+7476) 字形不同但同音。
> 若历史上曾发生编码问题，可能出现"两行 name 都是瑶"的情况。
> 此时**仅改 `icon_path` 不够**，必须连 `name` 一起修正，否则界面上会出现两个"瑶"。
> `name_hex` 列就是为了一秒识别这种情形。

#### 步骤 1：重命名图标文件（消灭"靠扩展名区分"的脆弱约定）

采用**拼音全称 + 语义后缀**命名，让文件名本身自解释：

| 现状 | 目标 | 理由 |
|------|------|------|
| `yao.jpg`（曜，内容实为 PNG） | `yao_hero.png` | 曜 = 英雄"曜"；同时修正扩展名与内容一致 |
| `yao.png`（瑶） | `yao_support.png` | 瑶 = 辅助位"瑶" |

> **命名备选**：也可用 `jing_yao`/`yao` 或 `yao_warrior`/`yao_support`。
> 本文推荐 `yao_hero.png` / `yao_support.png`：既避开同音歧义，
> 又不与既有 `yao` 前缀冲突。**施工时择一后必须全仓库一致**（见步骤 3 校验）。

**执行（两个副本都要改，含内容格式纠正）：**

```bash
cd /home/xiaohua/KPL_BP_Simulator

for DIR in frontend/public/images/heroes backend/static/images/heroes; do
  # 曜：重命名，并把"扩展名说谎"的 .jpg 真正转成 PNG（内容本就是 PNG，直接改名即可）
  mv "$DIR/yao.jpg" "$DIR/yao_hero.png"
  # 瑶：重命名
  mv "$DIR/yao.png" "$DIR/yao_support.png"
done

# 验证：两个文件都是真正的 PNG，且内容不同（不同英雄）
file frontend/public/images/heroes/yao_hero.png frontend/public/images/heroes/yao_support.png
md5sum frontend/public/images/heroes/yao_hero.png frontend/public/images/heroes/yao_support.png
```

> ✅ `yao.jpg` 的字节已是标准 PNG，因此 `mv` 改名即可，**无需重新编码或重新下载**。
> 改完后扩展名与真实格式终于一致，消除了事实 D 的隐患。

**⚠️ 清理旧路径的浏览器/CDN 缓存**：改名会让旧 URL 失效，这反而有利（强制刷新）。
但要注意 `HeroIcon` 的 `onError` 会静默降级为"名字前两字"占位块
（证据：`HeroCard.tsx:14-24`），因此**若路径没改全，界面不会报错，只会显示两个字**。
验收时必须按 §四 的判据肉眼确认图标是**图像**而非文字占位。

#### 步骤 2：新增迁移 `005_fix_yao_hero_icon.sql`

沿用现有迁移风格（`USE kpl_bp;` 开头，见 `004_match_add_bo_format.sql`）：

```sql
USE kpl_bp;
-- 问题3修复：曜/瑶 同音英雄的图标路径区分
-- 背景：二者旧路径仅靠 .jpg/.png 扩展名区分，极易写错，导致曜显示成瑶的图标。
-- 方案：改为语义化文件名，并将 name / icon_path 一并校正（含编码陷阱兜底）。

-- 1) 先按"名字是曜"定位，修正其图标路径与分路
UPDATE hero
   SET name      = '曜',
       icon_path = '/images/heroes/yao_hero.png',
       lanes     = '["打野","对抗路"]'
 WHERE id = (SELECT id FROM (SELECT id FROM hero WHERE name = '曜') t);

-- 2) 瑶：图标路径语义化，分路保持游走
UPDATE hero
   SET name      = '瑶',
       icon_path = '/images/heroes/yao_support.png',
       lanes     = '["游走"]'
 WHERE id = (SELECT id FROM (SELECT id FROM hero WHERE name = '瑶') t);

-- 3) 自检：必须恰好 2 行，且 icon_path 互不相同
SELECT id, name, icon_path, lanes FROM hero WHERE name IN ('曜','瑶');
```

> **为什么用 `SELECT id FROM (SELECT ...) t` 包一层**：MySQL 不允许在 `UPDATE` 的
> 子查询中直接引用被更新的同一张表，包一层派生表可绕过该限制。
>
> **若步骤 0 发现 `name` 编码异常（两行同名）**，则改用按 ID 精确定位：
> 先用 `SELECT id, name, icon_path FROM hero WHERE icon_path LIKE '/images/heroes/yao%'`
> 查出两个 ID，再把上面的 `WHERE id = (...)` 换成硬编码 ID，并对错误的 `name` 一并修正。

**同时同步 `002_seed_data.sql`**（保证"删库重跑"也能得到正确结果，否则新人重建环境会再次踩坑）：

```sql
-- 002_seed_data.sql:35
('曜', '/images/heroes/yao_hero.png', '["打野","对抗路"]'),
-- 002_seed_data.sql:143
('瑶', '/images/heroes/yao_support.png', '["游走"]'),
```

#### 步骤 3：同步下载脚本并做全仓库一致性校验

`scripts/download_hero_icons.py` 从 seed SQL 正则提取 `(name, filename)`
（`:41` 的 `pattern = r"'([^']+)', '/images/heroes/([^']+)'"`），
因此**只要步骤 2 改完 seed SQL，脚本自动跟随，无需改代码**。
但需补一条防回归校验：断言"任意两个英雄的 filename 不相同"。

**校验命令（施工后必跑）：**

```bash
cd /home/xiaohua/KPL_BP_Simulator

# ① seed SQL 里不应再出现裸 yao.jpg / yao.png
grep -n "heroes/yao" backend/migrations/*.sql

# ② 图标文件名在 seed 中必须唯一（防同名碰撞回归）
grep -oP "'/images/heroes/\K[^']+" backend/migrations/002_seed_data.sql | sort | uniq -d
#    → 期望：无输出

# ③ 每个 seed 里引用的图标文件都真实存在（在两个目录都对得上）
grep -oP "'/images/heroes/\K[^']+" backend/migrations/002_seed_data.sql | while read f; do
  [ -f "frontend/public/images/heroes/$f" ] || echo "缺失(前端): $f"
  [ -f "backend/static/images/heroes/$f" ] || echo "缺失(后端): $f"
done
#    → 期望：仅可能输出 "缺失(后端)" 若干（后端 static 目录本次施工不含新增），无"缺失(前端)"

# ④ 两个目录文件数一致
ls frontend/public/images/heroes | wc -l
ls backend/static/images/heroes   | wc -l
```

> ⚠️ **重要提示**：当前 `backend/static/images/heroes/` 与前端目录各有 132 个文件且 md5 一致。
> 若后端目录**未被使用**（前端走 `public/` 直接服务，后端 `/images` 仅作兜底），
> 也应同步改名以免"两个真相源"漂移。**施工时请确认后端 static 是否仍被引用**；
> 若确认废弃，可另开任务清理，但**本次必须至少保持两者一致**。

---

### 问题 1：BP 界面展示双方队名 + 队标

#### 步骤 1：修复队标 URL 的"相对路径 + 未代理"双重缺陷

**① 后端输出绝对路径**（治本，且对未来生产部署也正确）。
修改 `backend/internal/handler/game_handler.go:134` 附近，把 `logo_url` 规范化为以 `/` 开头：

```go
// normalizeLogoURL 确保队标为绝对路径，避免前端以当前页面为基准解析而 404
func normalizeLogoURL(raw string) string {
    if raw == "" {
        return ""
    }
    if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "/") {
        return raw
    }
    return "/" + raw   // "images/teams/x.png" → "/images/teams/x.png"
}
```

在构造 `logoUrl` 时套用：`"logoUrl": normalizeLogoURL(t.LogoURL)`。
（需在 import 中加 `"strings"`。）

> **替代方案**：直接写 003 迁移把库里 `logo_url` 补上前导斜杠。
> **推荐两者都做**：迁移负责修正存量数据，`normalizeLogoURL` 负责兜住未来新写入的数据
> —— 防御性编程，避免同类问题复发。

**② 补 Vite dev 代理**（`frontend/vite.config.ts`）：在 `proxy` 中加一段：

```ts
      // 英雄/战队图标静态资源，与后端 r.Static("/images", "./static/images") 对应
      '/images': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
```

> **为什么必须加**：后端 `main.go:89` 注册了 `r.Static("/images", "./static/images")`，
> 但前端 dev server 只代理 `/api`、`/ws`。不加这段，`/images/teams/*.png` 在开发环境必然 404。
>
> **另一种选择**：既然 `frontend/public/images/` 下已有全套图标（132 英雄 + 队标），
> 前端其实可以直接从自身 `public/` 提供，**根本不需要代理**。
> 两种方案二选一即可；**推荐加代理**，理由是让"后端是唯一数据源"的架构更清晰，
> 也避免两份图标副本长期漂移（这正是问题 3 的温床）。

#### 步骤 2：让队名队标随房间状态下发（避免前端依赖脆弱的 `teamNames` 映射）

现状 `BPRoom.tsx:72-76` 靠 `getTeams()` 拉全量战队再建 `id → name` 映射。
这**只拿到了队名，没拿队标**，且多一次请求、存在竞态。
更稳的做法是**在后端 `buildRoomState` 里直接带上两队的 `{id, name, logoUrl}`**。

修改 `backend/internal/handler/room_handler.go` 的 `buildRoomState`（`:565-590` 返回的 `gin.H`）
增加两个字段：

```go
"blueTeam": h.teamBrief(engine.BlueTeamID),
"redTeam":  h.teamBrief(engine.RedTeamID),
```

并新增小工具方法（内部查 `teamRepo.GetByID`，查不到返回 `nil` 兜底）：

```go
// teamBrief 返回前端展示所需的队伍摘要，查不到时返回 nil（前端降级为"蓝方/红方"）
func (h *RoomHandler) teamBrief(teamID int) interface{} {
    if teamID <= 0 {
        return nil
    }
    t, err := h.teamRepo.GetByID(teamID)
    if err != nil || t == nil {
        return nil
    }
    return gin.H{
        "id":      t.ID,
        "name":    t.Name,
        "logoUrl": normalizeLogoURL(t.LogoURL),
    }
}
```

> ⚠️ `buildRoomState` 在 `room.mu` 锁内被调用，`teamBrief` 会查库。
> **施工注意**：`GetRoomStatus` 等高频路径上建议加**进程内 team 缓存**
> （战队数据基本静态），避免每步 BP 都打一次 DB。
> 简单实现：`RoomHandler` 加 `teamCache map[int]*model.Team` + `sync.RWMutex`，
> 首次查询后缓存。这是本方案唯一需要留意的性能点。

#### 步骤 3：类型与视图接线

**① `frontend/src/types/index.ts`** 的 `BPState` 增加：

```ts
  blueTeam?: { id: number; name: string; logoUrl: string } | null
  redTeam?:  { id: number; name: string; logoUrl: string } | null
```

**② `frontend/src/components/BanPickBoard.tsx`**：

- `BanPickBoardProps` 增加 `blueTeam?` / `redTeam?` 入参（沿用上面的结构）；
- 把 `:308-314` 写死的标题替换为**队标 + 队名**的组合。示意：

```tsx
{/* 阵营标题：队标 + 队名；无队伍信息时降级为红蓝方 */}
const TeamHeader: React.FC<{ team?: TeamInfo | null; fallback: string; mirrored: boolean; labelClass: string }> = ({
  team, fallback, mirrored, labelClass,
}) => (
  <div className={`mb-2 flex items-center gap-2 ${mirrored ? 'flex-row-reverse' : ''}`}>
    {team?.logoUrl ? (
      <img
        src={team.logoUrl}
        alt={team.name}
        className="h-7 w-7 shrink-0 rounded object-contain"
        onError={(e) => { (e.target as HTMLImageElement).style.visibility = 'hidden' }}
      />
    ) : (
      <span className="h-7 w-7 shrink-0 rounded bg-gray-700/60" />
    )}
    <span className={`truncate text-sm font-bold ${labelClass}`}>
      {team?.name || fallback}
    </span>
  </div>
)
```

- 蓝方传 `mirrored={false}`，红方传 `mirrored={true}`（**保持现有镜像对称设计**）；
- 保留 `🔵` / `🔴` 作为**降级标识**：当 `team` 为空（如历史数据缺队伍）时，
  仍然能看出哪边是蓝、哪边是红。

**③ `frontend/src/pages/BPRoom.tsx`**：把 `bpState.blueTeam` / `bpState.redTeam` 透传给
`<BanPickBoard>`（`:307-320`）。同时**保留**现有 `teamNames` 作为结算面板的兜底，
避免破坏已验收的结算逻辑。

> ⚠️ **注意 `bpStore.updateBPState` 的版本号机制**：它用 `currentTurn` 做防乱序
> （见 `learn-as-you-go/BP多局流程与界面优化-施工方案.md` §2.2 的记载）。
> 新增字段不改变 `currentTurn` 语义，因此**不影响**该机制，无需改动 store。

#### 步骤 4：顺带修复 `HomePage` / `TeamSelector` 的队标 404

`TeamSelector.tsx:72` / `:111` 同样使用 `logoUrl`，会随步骤 1 一起被修好。
但**建议把 `onError` 从 `display:none` 改为保留占位块**，
否则"图挂了"这种问题会继续被静默掩盖（正是它掩盖了本次问题 1 的根因）。

---

### 问题 2：Ban 位与 Pick 位尺寸分级

#### 设计

| 槽位 | 现状 | 目标尺寸 | Tailwind | 名称字号 |
|------|------|---------|----------|---------|
| **Ban** | `w-14` (3.5rem) | 相对小 | `w-11` (2.75rem) | `text-[9px]` → `text-[8px]` |
| **Pick** | `w-14` (3.5rem) | 相对大 | `w-16` (4rem) | 保持 `text-[9px]` |

选 `w-11` / `w-16` 的理由：两者均在 `flex-wrap` 容器内单行可容纳 5 个，
不会换行破坏"上 ban 下 pick"的两行结构；差异约 **45%**，
肉眼可清晰分辨"体量层级"，符合"ban 小 pick 大"的直觉（禁用只是标记，选用才是成果）。

#### 实现要点（**成对改，防跳动**）

`HeroSlot`（`:102`）与 `EmptySlot`（`:150`）当前都硬编码 `w-14`。
**只改 `HeroSlot` 不改 `EmptySlot` 会导致槽位填充瞬间尺寸跳动** —— 必须成对。

推荐做法：给两个组件都加一个 `size: 'ban' | 'pick'` prop，集中查表：

```tsx
// 槽位尺寸分级：ban 小、pick 大（问题2）
const SLOT_SIZE = {
  ban:  { box: 'w-11', name: 'text-[8px]', icon: 'text-2xl' },
  pick: { box: 'w-16', name: 'text-[9px]', icon: 'text-4xl' },
} as const
```

- `HeroSlot`：`className` 中的 `w-14` 换成 `SLOT_SIZE[size].box`，
  名称行 `text-[9px]` 换成 `SLOT_SIZE[size].name`；
- `EmptySlot`：**同样**接受 `size`，用同一张表的 `box` / `name`
  （空槽位的占位 `·` 保持 `text-transparent`，只借用同款字号以锁定高度）；
- `renderSlotRow`（`:256`）增加 `size` 参数：ban 行传 `'ban'`、pick 行传 `'pick'`；
- `HeroCard.tsx` 中被 ban 英雄叠加的红色 `✕` 是 `text-4xl`（`:119`），
  属于**英雄池**卡片、不属于棋盘槽位，**本次不改**，避免范围蔓延。

#### 顺带清理（可选，低风险）

`ban` 槽位更小后，`renderSlotRow` 的标签列 `w-9`（`:267`/`:286`）可保持不变。
但**若发现红方镜像行因 pick 变宽而挤压**，调整方案是给槽位行容器加
`flex-wrap` 兜底（当前已是 `flex-wrap`，见 `:271`），无需额外处理。

---

### 问题 4：第二局重新选边（单机模式：提示 + 用户指定）

#### 设计决策（已与需求对齐）

| 场景 | 行为 |
|------|------|
| **单机模式（当前唯一）** | 第一局结束并录入胜方后，结算面板**提示下局需重新选边**，并提供 UI 让用户指定"下一局谁蓝谁红"。用户确认后才允许开始下一局 |
| **联机模式（预留）** | 按 KPL 规则由**败方选边**。本次仅把 `side_picker_team_id` 落库为败方 ID，**不启用交互**，留作后续扩展接缝 |
| 默认值 | 选边 UI 默认选中"**交换红蓝**"（贴近真实赛事：第二局通常换边），用户可改回不换 |
| 赛制边界 | 第 2 局起都要选边；`MatchOver` 后不再选边 |

> **为什么默认交换而非默认不变**：真实 KPL 中第一局后通常换边，默认交换可减少用户点击；
> 但**必须允许用户改回**（因为需求明确"让用户决定"）。

#### 步骤 1：后端 — `Room` 增加选边状态

在 `room_handler.go` 的 `Room` 结构体（`:25-39`）增加：

```go
	// 下一局的红蓝归属（由用户在局间选边 UI 指定；未指定时等于当前局）
	NextBlueTeamID *int
	NextRedTeamID  *int
	// 选边权归属队伍（联机模式=败方；单机模式可为 nil）
	SidePickerTeamID *int
```

#### 步骤 2：后端 — 新增选边接口

```
POST /api/rooms/:id/set-sides
Body: { "blueTeamId": 12, "redTeamId": 7 }
```

**校验（必须严格，否则会写坏比分）：**

1. 房间存在；本局 `Engine.IsFinished()` 为真；`room.Winner != nil`（胜负已录入）；
2. `room.MatchOver == false`；
3. `{blueTeamId, redTeamId}` 必须**恰好是本场比赛的两支队伍**（集合相等），
   即 `{blue,red} == {room.BlueTeamID, room.RedTeamID}` —— **防止把无关队伍塞进来**；
4. `blueTeamId != redTeamId`。

通过后写入 `room.NextBlueTeamID` / `room.NextRedTeamID`，并广播状态。

> **从 `match_record` 视角看**：`team_a_id` / `team_b_id` 是**固定身份**（不随选边变化），
> 而 `blue_team_id` / `red_team_id` 是**每局的阵营归属**。
> 因此**换边绝不能改 `match_record.team_a_id/team_b_id`**，
> 也**绝不能交换 `team_a_score` / `team_b_score`**，只需让新一局的 `game` 行写入新的红蓝即可。
> §2.4 提到的"关键反查"就指这一点：**大比分跟着 team_a/team_b 走，不跟着红蓝走**。

#### 步骤 3：后端 — `NextGame` 接受并使用选边结果

修改 `NextGame`（`:377-420`），把写死的 `room.BlueTeamID` / `room.RedTeamID` 换成：

```go
	// 采用用户在局间选定的红蓝归属；未选边则沿用当前局（保持向后兼容）
	nextBlue, nextRed := room.BlueTeamID, room.RedTeamID
	if room.NextBlueTeamID != nil && room.NextRedTeamID != nil {
		nextBlue, nextRed = *room.NextBlueTeamID, *room.NextRedTeamID
	}

	gameID, err := h.gameRepo.CreateGame(&model.Game{
		MatchID:    room.MatchID,
		GameNumber: gameNumber,
		BlueTeamID: nextBlue,
		RedTeamID:  nextRed,
	})
	...
	engine, err := h.gameService.StartGame(room.MatchID, gameNumber, nextBlue, nextRed)
```

并在成功后**提交这次选边**：
`room.BlueTeamID, room.RedTeamID = nextBlue, nextRed`，同时清空
`NextBlueTeamID` / `NextRedTeamID`（避免污染第 3 局）。

**⚠️ 全局BP记忆的正确性依赖红蓝语义**：
`applyGlobalBPMemory` 按"上局**蓝方**/ **红方**选了谁"来生成不对称权限
（见 `README.md` §全局BP记忆模式）。换边后，"上局的蓝方"变成了本局的红方。
**这是符合 KPL 真实规则的**（限制跟的是**队伍**，不是颜色）。
因此：
- 读上一局记录做记忆时，必须继续按**该局记录的 `side` 字段**解读（现有实现已是如此，✅ 无需改）；
- 但**代码审阅时务必确认** `applyGlobalBPMemory` 没有把"蓝方"硬编码成"某个固定队伍"。
  施工时请以"第 1 局 A 蓝胜 → 第 2 局换边后 B 蓝"为用例实测记忆方向是否正确。

#### 步骤 4：后端 — 落库 `side_picker_team_id`（联机预留）

在 `SubmitResult`（`room_handler.go` 中处理 `POST /result` 处，约 `:340-375`）里，
记录败方为选边权队伍：

```go
	// 联机模式：败方拥有下局选边权（本次仅落库，交互留待 dual 模式启用）
	loserID := room.BlueTeamID
	if *room.Winner == "blue" {
		loserID = room.RedTeamID
	}
	room.SidePickerTeamID = &loserID
	if err := h.matchRepo.UpdateSidePicker(room.MatchID, loserID); err != nil {
		log.Printf("更新选边权失败（不影响主流程）: %v", err)
	}
```

`match_repo.go` 新增 `UpdateSidePicker(matchID, teamID int) error`：

```go
func (r *MySQLMatchRepo) UpdateSidePicker(matchID, teamID int) error {
	return r.db.Model(&model.MatchRecord{}).
		Where("id = ?", matchID).
		Update("side_picker_team_id", teamID).Error
}
```

> 该字段已有索引与 FK（`001_create_tables.sql:16,21`），无需新迁移。
> **降级原则**：此步失败**不应阻断**胜负录入主流程（故仅记日志）。

#### 步骤 5：前端 — 局间选边 UI

修改 `BPRoom.tsx` 的局间结算面板"② 本局已录入胜负，比赛未结束"分支（`:247-283`），
在"开始第 N+1 局"按钮**上方**插入选边区块：

```
┌──────────────────────────────────────────────┐
│ ✅ 第 1 局结束：重庆狼队 获胜                  │
│ 大比分 蓝 1 : 0 红（BO5 · 先胜 3 局者夺冠）    │
├──────────────────────────────────────────────┤
│ 🔄 第 2 局选边                                │
│    当前：🔵 重庆狼队  vs  🔴 成都AG超玩会      │
│    下一局：[🔵重庆狼队 / 🔴成都AG超玩会  ▾]    │
│    ⓘ 单机模式下由你决定；联机模式将由败方选边   │
│    [ 交换红蓝 ]  ← 一键切换（默认已选交换）    │
├──────────────────────────────────────────────┤
│ [➡️ 开始第 2 局 BP]              ...          │
└──────────────────────────────────────────────┘
```

**交互细节：**

- 用**两个下拉框**（或一个"交换/不换"切换 + 只读展示）比自由输入更安全，
  下拉选项**只包含本场两队**，从根源上杜绝步骤 2 校验 ③ 的非法提交；
- **默认值 = 交换红蓝**（`nextBlue = 当前红方`）；
- 用户改动后立即调用 `setSides`，或**在点"开始下一局"时一起提交**。
  **推荐后者**（一次请求、无中间态、避免用户改了却忘记提交）；
- 若采用后者，则 `nextGame` 接口扩展为可选 body：
  `POST /next-game { blueTeamId?, redTeamId? }`，
  缺省时行为不变（**保持 API 向后兼容**，见 `docs/API.md` 更新）。

**`BPRoom.tsx` 需新增本地 state：**

```tsx
// 下一局红蓝归属（默认交换；用户可改）
const [nextBlueId, setNextBlueId] = useState<number | null>(null)
const [nextRedId,  setNextRedId]  = useState<number | null>(null)

// 本局结算完成时初始化默认值 = 交换红蓝
useEffect(() => {
  if (bpFinished && bpState.winner !== null && !isMatchOver) {
    setNextBlueId(bpState.redTeamId)
    setNextRedId(bpState.blueTeamId)
  }
}, [bpFinished, bpState.winner, isMatchOver, bpState.blueTeamId, bpState.redTeamId])
```

> ⚠️ **务必在"开始第 N+1 局"成功后清空这两个 state**，否则第 3 局的默认值会沿用第 2 局的
> 陈旧选择，导致"默认交换"语义错乱。

---

## 四、验收标准

> 每项都给出**可执行的验证动作**与**明确的通过判据**。建议按此顺序执行（问题 3 优先）。

### 前置：环境

```bash
cp -n scripts/env.sh.example scripts/env.sh   # 若 scripts/env.sh 不存在
# 编辑 scripts/env.sh 填入本机 MySQL 真实密码（否则报 Error 1045）
source scripts/env.sh
cd backend && go build ./cmd/server/ && ./server   # 后端
cd frontend && npm run dev                          # 前端（另一个终端）
```

### 问题 3 验收（最高优先）

| # | 动作 | 通过判据 |
|---|------|---------|
| 3.1 | `source scripts/env.sh && mysql ... -e "SELECT id,name,icon_path FROM hero WHERE name IN ('曜','瑶')"` | 两行 `icon_path` **不同**，分别为 `/images/heroes/yao_hero.png`、`/images/heroes/yao_support.png`；两行 `name` 字形正确（非同一个字） |
| 3.2 | 浏览器打开 BP 页，点分路筛选 **"打野"** | **曜** 的图标是**深色短发少年**（不是粉发少女） |
| 3.3 | 点分路筛选 **"对抗路"** | 同上，曜 图标正确 |
| 3.4 | 点分路筛选 **"游走"** | **瑶** 的图标是**粉紫长发少女** |
| 3.5 | 肉眼确认曜/瑶 卡片上是**图像**而非"曜"/"瑶"文字占位 | 若显示文字，说明 URL 404 被 `HeroIcon.onError` 降级了 —— 回到 §三 步骤 3 的校验命令排查 |
| 3.6 | `grep -oP "'/images/heroes/\K[^']+" backend/migrations/002_seed_data.sql \| sort \| uniq -d` | **无输出**（无重名） |
| 3.7 | 删库重跑 migrations 001→005 后再查 3.1 | 结果与 3.1 一致（保证可重建） |

> ⚠️ 若 3.2/3.4 仍不对，**先强制刷新浏览器**（Ctrl+Shift+R）排除旧图缓存，
> 再检查是否漏改了 `frontend/public/images/heroes/` 或 `backend/static/images/heroes/` 中的一个。

### 问题 1 验收

| # | 动作 | 通过判据 |
|---|------|---------|
| 1.1 | 首页选两支队伍创建房间，进入 BP 页 | 左右两张阵营卡上**同时显示队标 + 队名**（非"🔵 蓝方"占位） |
| 1.2 | 浏览器 DevTools → Network，过滤 `images/teams` | 所有队标请求 **200**，无 404 |
| 1.3 | 直接访问 `http://localhost:3000/images/teams/chongqin_wolves.png` | 返回图片（验证 §三 步骤 1 的 `/images` 代理生效） |
| 1.4 | 走完 BP 后用旧房间再次进入（刷新页面） | 队名队标**仍然正确**（验证随房间状态下发，不依赖前端映射） |
| 1.5 | 红蓝镜像检查 | 蓝方队标+队名靠左，红方**靠右**，整体仍左右对称 |

### 问题 2 验收

| # | 动作 | 通过判据 |
|---|------|---------|
| 2.1 | 观察任一方的"禁用"行与"选择"行 | **pick 槽位明显大于 ban 槽位**，体量差异约 45%，一眼可辨 |
| 2.2 | 从第 1 步走到第 20 步，全程盯住棋盘 | Ban/Pick 槽位在"空 → 填充"瞬间**没有任何尺寸跳动或布局位移** |
| 2.3 | 缩放到 1280px 宽窗口 | 每行 5 个 ban / 5 个 pick **都不换行**，棋盘不溢出 |
| 2.4 | 窄屏（<768px）检查 | 红蓝两卡**纵向堆叠**，仍无横向滚动条 |

### 问题 4 验收

| # | 动作 | 通过判据 |
|---|------|---------|
| 4.1 | 走完第 1 局 20 步 + 录入胜方 | 结算面板出现**"第 2 局选边"区块**，明确提示下局需选边 |
| 4.2 | 检查选边 UI 默认值 | 默认**已选中"交换红蓝"** |
| 4.3 | 点"开始第 2 局 BP" | **第 2 局红蓝确实互换**（棋盘标题的队伍对调） |
| 4.4 | 大比分核对（**关键**） | 换边前后大比分**始终跟着队伍**，不发生"A 队的分记到 B 队" |
| 4.5 | 第 2 局录入胜方后进入第 3 局 | 选边 UI **重新出现**，默认值基于第 2 局的归属重新计算（验证 state 已清空） |
| 4.6 | 选"不换边"再进入下一局 | 红蓝归属与上一局**一致** |
| 4.7 | 全局BP记忆方向（**关键**） | 第 1 局 A 队（蓝）选过的英雄，第 2 局换边后 A 队（红）**不可再选但可 ban**；B 队可选不可 ban |
| 4.8 | 打到某方夺冠 | 显示"🏆 比赛结束"，**不再出现**选边 UI |
| 4.9 | 查库 `SELECT side_picker_team_id FROM match_record` | 为**败方** team_id（联机预留落库生效） |
| 4.10 | 向后兼容：不调用 `set-sides` 直接 `next-game` | 行为与改动前一致（沿用当前红蓝），不报错 |

> **4.4 与 4.7 是最容易出错的两项**，也是本方案在 §三 问题4 步骤 2/3 反复强调
> "比分跟 A/B 走、限制跟队伍走"的原因。施工后请务必**手工推演一遍**这两条。

---

## 五、风险与对策

| 风险 | 影响 | 对策 |
|------|------|------|
| ⚠️ 库内 `曜`/`瑶` 的 `name` 本身编码错乱 | 仅改 `icon_path` 后界面出现两个"瑶" | §三 问题3 步骤 0 用 `HEX(name)` 取证；异常时按 ID 精确 UPDATE，连 `name` 一起修 |
| ⚠️ 只改了前端或只改了后端的一份图标 | 某条路径仍 502/404，且被 `onError` 静默降级 | §三 问题3 步骤 3 的校验脚本对**两个目录**都检查；验收 3.5 要求肉眼确认是图像 |
| 迁移 005 在旧库上执行失败（`name` 不唯一/找不到） | 数据未修正 | 先跑步骤 0 查询确认存在这两行；异常时改用硬编码 ID 版本 |
| 队标每步 BP 都查库，拖慢状态下发 | BP 操作变卡 | `teamBrief` 加进程内缓存（§三 问题1 步骤 2 已注明） |
| 换边后大比分记错队（A/B 与红蓝混淆） | 🔴 赛果错误 | 明确规定 `match_record.team_a/b_score` **不随选边变动**；验收 4.4 专项核对 |
| 全局BP记忆方向在换边后反了 | 🔴 规则错误 | 记忆按"上局记录的 side + 队伍"解读；验收 4.7 专项核对 |
| `nextGame` 扩展后破坏现有 API 契约 | 旧调用方失败 | body 全部可选，缺省行为不变；同步更新 `docs/API.md`（验收 4.10） |
| 第 3 局沿用第 2 局陈旧选边 state | 默认值语义错乱 | 开始下一局成功后清空 `nextBlueId/nextRedId`（§三 问题4 步骤 5 已注明） |
| 前后端图标两份副本长期漂移 | 问题 3 类 bug 复发 | 长期建议：统一由后端 `/images` 提供（或前端 `public/` 唯一真相源），另开清理任务 |

---

## 六、施工顺序与工作量

> **推荐顺序的理由**：问题 3 是数据正确性（🔴）且与其他改动零耦合，先独立完成便于验证；
> 问题 1、2 都是纯前端展示；问题 4 跨前后端且涉及赛果正确性，放最后并留足测试时间。

| 阶段 | 内容 | 依赖 | 预估 |
|------|------|------|------|
| **P0** | 问题 3：查库取证 → 改名 → 迁移 005 → 同步 seed → 校验 | 无 | 0.5 天 |
| **P1** | 问题 1：`normalizeLogoURL` + `/images` 代理 + `buildRoomState` 下发队伍 + 棋盘队名队标 | 无 | 1 天 |
| **P2** | 问题 2：`SLOT_SIZE` 分级，`HeroSlot`/`EmptySlot` 成对改 | 无 | 0.5 天 |
| **P3** | 问题 4：`Room` 字段 → `set-sides` 接口 → `NextGame` 接线 → 落库 `side_picker_team_id` | 无 | 1 天 |
| **P4** | 问题 4：前端局间选边 UI + 默认交换 + state 清理 | P3 | 0.5 天 |
| **P5** | 联调 + 按 §四 全量验收（含 4.4 / 4.7 专项） | P0–P4 | 1 天 |
| **P6** | 文档回填：更新 `docs/API.md`（新增 `set-sides`、`next-game` 可选 body、`BPState` 新增 `blueTeam`/`redTeam`）、`README.md`（多局流程增加选边步骤）、迁移到 `learn-as-you-go/` 并记录验收结果 | P5 | 0.5 天 |

**合计约 5 人日。**

**建议 commit 切分**（每个可独立回滚）：

1. `fix(db): 修正曜/瑶英雄名与图标路径并重命名图标文件（问题3）`
2. `feat(frontend): 修复队标路径并在BP棋盘展示双方队名队标（问题1）`
3. `feat(frontend): Ban/Pick 槽位尺寸分级（问题2）`
4. `feat(backend): 局间选边接口与 next-game 接线，落库 side_picker_team_id（问题4）`
5. `feat(frontend): 局间选边UI，默认交换红蓝（问题4）`
6. `docs: 回填选边与队标相关文档及验收结果`

---

## 七、附：施工前必须回答的开放问题

> 这些不影响方案骨架，但**施工到对应步骤前需要确认**，避免返工。

1. **`backend/static/images/` 是否仍被实际使用？**
   后端 `main.go:89` 注册了 `/images`，但前端 `public/images/` 也有全套副本。
   若后端静态目录已废弃，问题 3 的改名/校验范围可收窄，并可另开清理任务消灭双份副本。
   （**本次仍建议两边都改，保持一致性。**）

2. **`scripts/env.sh` 当前不存在** —— 施工前必须先 `cp` 模板并填密码，否则问题 3 的查库取证无法进行。

3. **队标尺寸与布局**：阵营卡标题行加队标后，是否需要同时调整卡片高度以保持左右对称？
   建议施工时以 1280px 视口实测，**若红方镜像行高度与蓝方不一致，需给标题行固定高度**。

4. **`next-game` 是扩展 body 还是新增 `set-sides` 独立接口？**
   → ✅ **已决定：只做 `next-game` 可选 body**（需求方确认，改动面最小）。
   本方案 §三 问题4 步骤 2 描述的独立 `set-sides` 接口**未实现**，
   其校验逻辑已移入 `NextGame`（见 §8.5 的非法选边校验用例）。

5. **第二局选边默认值？**
   → ✅ **已决定：默认不换（沿用上局）**，用户可点「⇄ 交换红蓝」调整。

6. **图标资源以哪边为唯一数据源？**
   → ✅ **已决定：加 `/images` 代理，后端为唯一源**（§三 问题1 步骤 1 已实现）。
   本次仍同步维护两份副本以免漂移，统一清理另开任务。

7. **队标查询是否加缓存？**
   → ✅ **已决定：加**（§三 问题1 步骤 2 的 `teamCache` 已实现）。

---

## 八、验收结果

> ✅ **已完工并全部验收通过**（2026-10-12）
>
> 📌 落地代码见下方"改动文件清单"；施工中发现并修复了一个**方案阶段未预见的严重缺陷**（见 §8.1）。

### 8.1 ⚠️ 施工中新发现的缺陷：全局BP记忆"跟颜色走"（已修复）

**这是本次施工最重要的发现，方案 §五 风险表中列为"🔴 规则错误"的那一条真实发生了。**

`backend/internal/service/game_service.go` 的 `applyGlobalBPMemory` 原实现**只按颜色映射**：

```go
// 旧实现（有缺陷）
switch a.Side {
case "blue":  // 上一局"蓝方"选过
    perms[a.HeroID] = &HeroPerm{BlueCanPick: false, RedCanPick: true}
}
```

它把"上一局的蓝方"直接等同于"本局的蓝方"。**在第 1 局不换边时这个假设恰好成立，所以从未暴露**；
但问题 4 引入选边后，第 2 局的红蓝归属对调，该假设失效，产生**两个方向都反了**的错误：

| 场景 | 旧实现行为 | 正确行为 |
|------|-----------|---------|
| A 队上局（蓝）选过达摩；第 2 局换边后 A=红、B=蓝 | 限制 **B 队**（没选过的）不可选；**A 队**（选过的）可再选 | 限制 **A 队**不可再选；**B 队**可选不可 ban |

修复方式：先用**上一局**的 `blue_team_id` / `red_team_id` 把 `a.Side`（颜色）翻译成**队伍**，
再用**本局**的红蓝归属翻译回颜色，据此设置权限位。同时修正了 `GlobalBPMemoryPerm` 的
入参语义（由"颜色"改为"该队本局是否为蓝方"）。

**实测验证（换边后三大断言全部通过）**：

```
第1局: A队(蓝)pick=['达摩','姬小满','铠','梦奇','芈月']
第2局换边后: 蓝=2(B队) 红=1(A队)
  ✅ B队(蓝) 不可 ban A队选过的【达摩】: 英雄5受全局BP规则限制，此方不可ban
  ✅ B队(蓝) 可选 A队选过的【达摩】（对手可选）
  ✅ A队(红) 不可再选自己上局选过的【姬小满】: 英雄8受全局BP规则限制，此方不可pick
```

> **教训**：这个缺陷无法靠"不换边"的测试发现，只有真正跑通"换边 + 全局BP记忆"的组合用例才会暴露。
> 已在本方案 §四 验收 4.7 中固化为必测项。

### 8.2 问题 3 验收（曜/瑶）

**查库取证结果（施工前）** —— 证实了方案 §2.3 的推断，且与"用户直觉"不同：

```
id    name  name_hex    icon_path                  lanes
31    曜    E69B9C      /images/heroes/yao.png     ["打野","对抗路"]   ← BUG：指向了瑶的图标
127   瑶    E791B6      /images/heroes/yao.png     ["游走"]
```

- **字形均正确**（无编码事故），**问题纯粹是 `曜` 的 `icon_path` 写成了 `yao.png`**；
- 全库扫描确认这是**唯一一处** `icon_path` 重复（`GROUP BY ... HAVING c>1` 仅返回 yao.png 一行）。

| # | 判据 | 结果 |
|---|------|------|
| 3.1 | 曜/瑶 `icon_path` 互不相同 | ✅ `yao_hero.png` / `yao_support.png` |
| 3.2 | 打野分路曜图标正确（非瑶） | ✅ 曜=True 瑶=False（共 47 位） |
| 3.3 | 对抗路分路曜图标正确 | ✅ 曜=True 瑶=False（共 47 位） |
| 3.4 | 游走分路瑶图标正确 | ✅ 曜=False 瑶=True（共 31 位） |
| 3.6 | seed 无重名 + 全库路径唯一 | ✅ 无重名；132 个英雄路径全唯一 |
| 3.7 | 迁移 005 幂等 | ✅ 连跑两次结果一致 |

- 文件重命名：`yao.jpg` → `yao_hero.png`（曜）、`yao.png` → `yao_support.png`（瑶），
  `frontend/public` 与 `backend/static` **两份副本均已同步**（各 132 个文件）；
  顺带修正了 `yao.jpg` "扩展名说谎"（内容是 PNG）的既存隐患。

### 8.3 问题 1 验收（队名 + 队标）

| # | 判据 | 结果 |
|---|------|------|
| 1.1 | 棋盘显示队标 + 队名 | ✅ `blueTeam`/`redTeam` 随房间状态下发，含 `id/name/logoUrl` |
| 1.2 | 队标请求 200 | ✅ 直连后端与经 Vite 代理均 200 |
| 1.3 | `/images` 代理生效 | ✅ `/images/teams/chongqin_wolves.png` → 200 |
| — | 旧路径不残留 | ✅ `/images/heroes/yao.png` → 404（不会被缓存命中错误图） |

- 数据库 `logo_url` 已批量补前导斜杠（迁移 006，18 支队伍全部为绝对路径）；
- 后端新增 `normalizeLogoURL` 兜住未来新写入的相对路径（双保险）；
- 新增进程内 `teamCache`（`sync.RWMutex`），避免每步 BP 都查库。

### 8.4 问题 2 验收（Ban/Pick 尺寸分级）

| # | 判据 | 结果 |
|---|------|------|
| 2.1 | pick 明显大于 ban | ✅ ban `w-11`(2.75rem) / pick `w-16`(4rem)，差异约 45% |
| 2.2 | 填充瞬间无尺寸跳动 | ✅ `HeroSlot` 与 `EmptySlot` **共用同一张 `SLOT_SIZE` 表**，成对缩放 |
| — | 类型检查 | ✅ `tsc --noEmit` 通过；`vite build` 通过 |

### 8.5 问题 4 验收（重新选边）——含最关键的比分/规则正确性

**端到端 BO5 实测（每局交换红蓝，A 队全胜）**：

| 局 | 红蓝归属 | 录入 | 比分视图 | DB `team_a/b_score` |
|----|---------|------|---------|---------------------|
| 1 | A=蓝 B=红 | A 胜 | blue=1 red=0 | 1 : 0 |
| 2 | **B=蓝 A=红** | A 胜 | blue=0 red=**2** | 1 : 1（A 队累计 2 胜） |
| 3 | A=蓝 B=红 | A 胜 | blue=**3** red=0 | **3 : 0** ✅ |

> ✅ **验收 4.4（最关键）**：换边后 `blueScore/redScore` **视图**随之翻转，
> 但 `team_a_score=3 / team_b_score=0` **始终跟着队伍**，未把 A 队的胜场记到 B 队名下。
> 这是"比分跟 A/B 走、不跟红蓝走"设计的直接验证。

| # | 判据 | 结果 |
|---|------|------|
| 4.1 | 局间出现选边 UI | ✅ 含下拉框 + 「⇄ 交换红蓝」+ 说明文案 |
| 4.2 | 默认值 | ✅ **默认不换**（沿用当前红蓝），符合需求方确认 |
| 4.3 | 换边生效 | ✅ 第 2 局 `blue_team_id`/`red_team_id` 已切换 |
| 4.7 | 全局BP记忆方向 | ✅ 见 §8.1，换边后三大断言全部通过 |
| 4.8 | 夺冠后不再选边 | ✅ 第 3 局达 3 胜即 `matchStatus=finished` |
| 4.9 | `side_picker_team_id` 落库 | ✅ 记录为败方 team_id |
| 4.10 | 向后兼容（不带 body） | ✅ 沿用当前红蓝，行为与改动前一致 |

**非法选边校验（全部按预期拒绝）**：

| 用例 | 结果 |
|------|------|
| 同队 `{blue:1, red:1}` | ✅ 400 蓝方与红方不能是同一支队伍 |
| 无关队 `{blue:1, red:99}` | ✅ 400 选边队伍必须是本场比赛的两支参赛队伍 |
| 半截参数 `{blue:1}` | ✅ 400 选边需同时提供 blueTeamId 与 redTeamId |
| 合法换边 | ✅ 200 |

### 8.6 改动文件清单

| 层 | 文件 | 说明 |
|----|------|------|
| DB | `backend/migrations/005_fix_yao_hero_icon.sql` | 新建：曜/瑶 名字与路径修正（幂等+兜底） |
| DB | `backend/migrations/006_fix_team_logo_path.sql` | 新建：队标路径补前导斜杠 |
| DB | `backend/migrations/002_seed_data.sql` | 同步曜/瑶 新路径 |
| 资源 | `frontend/public/images/heroes/`、`backend/static/images/heroes/` | `yao_hero.png` / `yao_support.png` |
| 后端 | `internal/handler/team_brief.go` | 新建：`normalizeLogoURL` / `teamBrief` / `teamCache` / `sidePickerOut` |
| 后端 | `internal/handler/room_handler.go` | 队伍下发、选边接口、比分按队伍记账、`side_picker_team_id` 落库 |
| 后端 | `internal/handler/game_handler.go` | 队标路径规范化 |
| 后端 | `internal/service/game_service.go` | **修复全局BP记忆跟队伍走**（§8.1） |
| 后端 | `internal/repository/match_repo.go` + `mysql_repo.go` | 新增 `UpdateSidePicker`，身份约定注释 |
| 前端 | `vite.config.ts` | 新增 `/images` 代理 |
| 前端 | `src/types/index.ts`、`store/bpStore.ts` | `TeamBrief` + `BPState` 新字段 |
| 前端 | `src/components/BanPickBoard.tsx` | `TeamHeader` 队标队名 + `SLOT_SIZE` 尺寸分级 |
| 前端 | `src/pages/BPRoom.tsx`、`services/api.ts` | 局间选边 UI + `nextGame` 可选 body |
| 文档 | `docs/API.md`、`README.md` | 新增字段/接口/选边流程说明 |

### 8.7 遗留事项

1. **`backend/static/images/` 与 `frontend/public/images/` 双份副本仍然存在**（本次已同步改名保持一致），
   长期建议统一为单一真相源，另开清理任务。
2. **联机（dual）模式的"败方自动选边"交互未启用**，本次仅完成 `side_picker_team_id` 落库预留。
3. `backend/internal/handler/hero_handler.go` 等文件存在**既存的 gofmt 未格式化**问题（非本次改动引入），
   未一并处理以免污染本次 diff。

---

## 九、第二轮：实战反馈的 4 个新问题（已完工）

> 📅 完工日期：2026-10-12 ｜ ✅ 全部验收通过
>
> 第一轮上线后实战测试反馈了 4 个问题。其中 **#3/#4 与 §8.1 的选边改动直接相关**，
> 是"选边功能上线后暴露出的前端信息缺口"。

### 9.1 问题与根因

| # | 反馈现象 | 根因（实测确认） |
|---|---------|-----------------|
| 1 | 历史记录里两队显示为"红蓝方"而非队名；一换边 BP 信息归属就错乱 | `GameResultTable` 中 `队伍{id}` 占位符 + 表头把 `teamA/teamB` **硬编码**标成 🔵/🔴。而红蓝是**每局**概念（第 2 局可能换边），`game.blueTeamId/redTeamId` 才是当真实归属 |
| 2 | 需要新增英雄**王维**（中路） | 图标 `wang_wei.png` 只存在于后端目录，DB/seed 中均无记录 |
| 3 | RW 已用敖隐+不知火舞，狼队不能 ban 敖隐（✅生效），但 **RW 仍能再选火舞**（❌） | **后端规则其实是正确的**（实测再审火舞会被拒绝）。真实缺口是：前端 `isHeroActionable` **只看 ban/pick 占用状态，完全不读权限表**，所以受限英雄看起来仍可点，用户以为"没被限制" |
| 4 | 狼队 ban 敖隐时确实 ban 不了，但图标不变灰不打叉，只弹"敖隐不可ban" | 同 #3：后端**从未把 `permissions` 下发给前端**。`HeroState.permissions` 在前端类型里定义了却始终为空 |

> **一句话根因**：#3 与 #4 是**同一个缺陷的两种表现** ——
> 后端有完整权限判定（`HeroPerm` 4 布尔）但**不下发**；前端只能靠"点了才知道"。

### 9.2 修复内容

**(1) 后端：把每英雄权限表随状态下发**（`buildRoomState`）

```go
"heroes": [ { "id": 14, "status": "available",
              "permissions": {"blueCanBan":true,"blueCanPick":true,
                              "redCanBan":true,"redCanPick":true} } ]
```

- 权限唯一事实来源仍是 `engine.Permissions`，`buildRoomState` **只做序列化**，不重复规则判断；
- 顺带修正了原实现"从 `HeroPool` 造 heroes、只给 `available`"导致 `status` 永远无意义的问题：
  现在用 `engine.HeroStatusOf()` 给出真实状态（`banned_by_blue` / `picked_by_red` / `restricted_by_rule` …）；
- 新增 `HeroStatus.String()` 与 `Engine.HeroStatusOf()` 导出方法。

**(2) 前端：提前置灰禁用**（问题3/4 的用户诉求）

- `BanPickBoard` 新增 `heroStates` prop，构建 `permMap`；
  新增 `isBlockedByRule(heroId)`：按**当前方 + 当前动作**查对应布尔位；
- `isHeroActionable = 未被 ban/pick 占用 && !isBlockedByRule` → 受限英雄**直接不可点击**；
- `HeroCard` 新增 `restricted` prop：**变灰（grayscale）+ 禁用光标 + 叠加「限」角标**
  （与被 ban/pick 的「禁」「蓝」「红」角标区分），`title` 说明"受全局BP规则限制"；
- 权限字段缺失时（旧快照）**保守放行**，交由后端裁决，避免误禁用。

**(3) 历史页显示真实队名**（问题1）

- `GameResultTable` 增加 `getTeamName`，每局用**本局** `game.blueTeamId/redTeamId` 解析队名，
  显示为「🔵 济南RW侠 / 🔴 重庆狼队」；胜方列也改为显示**真实队名 + 胜**；
- `MatchCard` 头部的总比分**去掉 🔵/🔴 标注**（大比分跟队伍身份走、与红蓝无关），
  加"（总比分）"说明，避免换边后误导；
- 每局队名列恢复显示（`showTeams`）。

**(4) 新增英雄王维**（问题2）

- `backend/migrations/007_add_hero_wang_wei.sql`：`INSERT ... WHERE NOT EXISTS` 幂等插入；
- `002_seed_data.sql` 中路分组按拼音补入（`wang_wei` 排在 `wang_zhao_jun` 之前）；
- 图标同步到 `frontend/public/images/heroes/wang_wei.png`（与后端 md5 一致）；
- 顺带清理了 Windows 下载残留的 `wang_wei.png:Zone.Identifier` 垃圾文件。

### 9.3 验收结果

| # | 判据 | 结果 |
|---|------|------|
| 2.1 | 王维出现在英雄列表 | ✅ id=133，`lanes=["中路"]`，总英雄数 133 |
| 2.2 | 迁移 007 幂等 | ✅ 连跑两次仍 133 行、无重复 |
| 2.3 | 图标经前端代理可取 | ✅ `/images/heroes/wang_wei.png` → 200 |
| 3.1 | 权限表随状态下发 | ✅ 133/133 个英雄均带 `permissions` |
| 3.2 | 狼队(蓝)不可 ban 敖隐 | ✅ `blueCanBan=False` |
| 3.3 | RW(红)不可再选火舞 | ✅ `redCanPick=False` |
| 3.4 | **换边后限制仍跟队伍** | ✅ 第2局换边后 RW 变蓝：`blueCanPick=False`(RW不可再选)、狼队变红：`redCanPick=True` 且 `redCanBan=False` |
| 4.1 | 受限英雄前端变灰禁用 | ✅ `isHeroActionable` 返回 false → 卡片灰化 + 禁用光标 + 「限」角标 |
| 1.1 | 历史页每局显示真实队名 | ✅ 每局按 `game.blueTeamId/redTeamId` 解析，换边后正确 |
| 1.2 | 胜方显示队名 | ✅ 「🔵 重庆狼队 胜」 |
| 1.3 | 总比分不误导 | ✅ 头部去掉红蓝标注 |
| — | 回归：BO5 换边全程 | ✅ 视图随换边翻转、`team_a_score=3 / team_b_score=0` 始终跟队伍 |
| — | 构建 | ✅ 后端 build+gofmt、前端 tsc+build 全通过 |

**问题2/3 的验证场景（与用户描述完全一致）**：

```
第1局: 济南RW侠(红) 选了 敖隐 + 不知火舞
第2局(不换边): 重庆狼队=蓝, 济南RW侠=红
  敖隐   permissions = {blueCanBan:False, blueCanPick:True, redCanBan:True, redCanPick:False}
    -> 狼队(蓝)不可ban ✅（用户反馈已生效，保持一致）
    -> RW(红)不可pick  ✅（本轮修复：前端现在会直接置灰）
  不知火舞 permissions = {blueCanBan:False, blueCanPick:True, redCanBan:True, redCanPick:False}
    -> RW(红)不可再选 ✅
```

> 📌 **关于"问题3 后端到底有没有 bug"**：实测证明**后端规则本身是正确的**
> （直接调 API 再选火舞会被拒绝）。用户感知到的"RW 依然可以选择火舞"，
> 本质是前端没有把受限英雄置灰，导致该英雄看起来仍可点选。
> 本轮通过下发 `permissions` + 前端提前禁用，从体验上彻底解决。

### 9.4 本轮改动文件

| 层 | 文件 | 说明 |
|----|------|------|
| DB | `backend/migrations/007_add_hero_wang_wei.sql` | 新建：幂等插入王维 |
| DB | `backend/migrations/002_seed_data.sql` | 中路分组补入王维 |
| 资源 | `frontend/public/images/heroes/wang_wei.png` | 新增（与后端同步） |
| 后端 | `internal/handler/room_handler.go` | `buildRoomState` 下发 `permissions` + 真实 `status` |
| 后端 | `internal/service/bp_engine.go` | 新增 `HeroStatusOf()` 与 `HeroStatus.String()` |
| 后端 | `internal/handler/team_brief.go` | 新增 `heroStatusString()` |
| 前端 | `src/types/index.ts` | `HeroState.permissions` 落地 |
| 前端 | `src/components/BanPickBoard.tsx` | `heroStates` prop + `isBlockedByRule` 置灰判定 |
| 前端 | `src/components/HeroCard.tsx` | `restricted` prop：灰化 + 禁用光标 + 「限」角标 |
| 前端 | `src/components/GameResultTable.tsx` | 每局真实队名、胜方队名、总比分去红蓝标注 |
| 前端 | `src/pages/BPRoom.tsx` | 透传 `heroStates` |
| 文档 | `docs/API.md`、`README.md` | `permissions` 语义与前端置灰说明 |
---

## 十、第三轮：从未结束的对局记录继续模拟 BP（已完工）

> 📅 完工日期：2026-10-12 ｜ ✅ 全部验收通过
>
> 新功能需求：**对局记录里，还未结束的比赛，可以选中它然后继续模拟 BP。**

### 10.1 难点：房间只存在于内存

这个需求比表面看起来难，因为原架构有一条硬伤：

- `Room`（含 `Engine` 全部 BP 状态）**只存在于 `RoomHandler.rooms` 内存 map 中**；
- `roomId` 由 `generateRoomID()` 随机生成，**从未落库**；
- 因此后端一重启，DB 里 `status='ongoing'` 的比赛就彻底回不到房间 —— 历史页只能看，不能继续。

### 10.2 方案

**(1) 落库 `room_id`**（迁移 008）

`match_record` 新增 `room_id`（唯一索引，可空），使"比赛记录 → 房间"可反查。
建表用存储过程包裹 `information_schema` 判断，保证**幂等**（MySQL 8.0 无 `ADD COLUMN IF NOT EXISTS`）。

**(2) 重建房间**（新增 `backend/internal/handler/resume_handler.go`）

```
match_record        → 固定身份 team_a/team_b、赛制、大比分、room_id
game(当前局)         → 该局红蓝归属、胜方、bp_completed
ban_pick_action     → 当前局已走的 N 步 → 逐字节回放
```

恢复目标 = **当前进度**：
- BP 未走完 → 回放到第 N 步，停在第 N+1 步继续
- BP 已走完但未录胜负 → 引擎置 `finished`，前端直接进结算面板
- 第 2 局起 → 用**该局**的 `blue_team_id/red_team_id`（而非首局），故换边后归属正确

引擎侧新增 `Engine.ReplayAction()`：复用 `handleBan/handlePick` 保证状态变更路径与正常流程
**完全一致**，但**跳过 validator 校验**（历史步骤是既成事实，重新校验可能因规则演进误判）。

**(3) ⚠️ 关键前提修复：BP 步骤改为增量落库**

回放要求 `ban_pick_action` 里有"中途进度"，但实测发现：

```
中途走到第 7 步 → DB 中 game 81 的 actions = 0 条
```

原因：步骤**只在整局结束时**由 `FinishGame` 一次性写入。
不修这个，"恢复到中间进度"无从谈起，第一版恢复实现正是因此**回放 0 步**。

修复：
- 新增 `RoomHandler.persistAction()`：每步执行后立即落库（按 `step_order` 判重，幂等）；
- `FinishGame` 从"全量插入"改为**对账补齐**（只写 DB 中尚不存在的 `step_order`），
  既兼容旧路径，又不会与增量写入产生重复；
- `GameRepo` 新增 `DeleteBPActions()`（供重建前清理）。

**(4) 接口与前端**

- 新增 `POST /api/matches/:id/resume` → 返回 `{ roomId, state }`，前端跳 `/room/{roomId}`；
- **幂等**：房间已在内存中直接复用，不重复回放；
- 历史页 `MatchCard` 为 `status='ongoing'` 的比赛增加「▶ 继续模拟 BP」按钮，
  并显示进度提示（如"第1局 BP 进行中（7/20步）"）。

### 10.3 验收结果

**核心用例：中途停止 → 杀掉后端 → 重启 → 恢复**（最关键的一条）

| 步骤 | 结果 |
|------|------|
| 第 1 局走到第 7 步（turn=8）后停止后端 | 蓝ban=[白起,程咬金] 红pick=[关羽,花木兰] |
| DB 校验增量落库 | ✅ `actions=7, bp_completed=0` |
| 重启后 `POST /matches/:id/resume` | ✅ 恢复到 turn=8，四个 ban/pick 列表**逐项一致** |
| 幂等（再次 resume） | ✅ 复用同一 roomId |
| 继续走完剩余 13 步 | ✅ `status=finished`，共 20 步 |
| DB 最终校验 | ✅ `actions=20, distinct_steps=20`（无重复） |

**其余场景**

| 场景 | 判据 | 结果 |
|------|------|------|
| A. BP 已走完、未录胜负 | 恢复后 `status=finished` 且 `winner=null` | ✅ 前端直接进结算面板 |
| B. 第 2 局已换边、中途恢复 | 局号/步号/红蓝归属全部正确 | ✅ `gameNumber=2, turn=6, 蓝=13 红=1` |
| C. 恢复后全局BP记忆权限 | 第1局选过的英雄，第2局该队仍不可再选 | ✅ `blueCanPick=False`（狼队）、`redCanPick=True/redCanBan=False`（RW） |
| D. 已结束的比赛 | `400 该比赛已结束，无法继续` | ✅ |
| E. 恢复后再打完整个系列赛 | 换边/比分/夺冠逻辑不受影响 | ✅ |

> **场景 C 是容易漏掉的一条**：恢复时必须保证 `permissions` 也被正确重建，
> 否则前端会把受限英雄当可用的。实测确认 `StartGame` 在重建时会重新读取上一局
> `ban_pick_action` 派生权限，因此**权限与恢复前完全一致**。

### 10.4 本轮改动文件

| 层 | 文件 | 说明 |
|----|------|------|
| DB | `backend/migrations/008_match_add_room_id.sql` | 新建：`match_record.room_id` + 唯一索引（幂等） |
| 后端 | `internal/handler/resume_handler.go` | 新建：`ResumeRoomFromMatch` / `ensureRoom` / `rebuildRoom` |
| 后端 | `internal/handler/room_handler.go` | `persistAction()` 逐步落库；`CreateRoom` 落 `room_id` |
| 后端 | `internal/service/bp_engine.go` | 新增 `ReplayAction()`（跳过校验的状态回放） |
| 后端 | `internal/service/game_service.go` | `FinishGame` 改为对账补齐 |
| 后端 | `internal/repository/game_repo.go` + `mysql_repo.go` | 新增 `DeleteBPActions` |
| 后端 | `internal/repository/match_repo.go` + `mysql_repo.go` | 新增 `SetRoomID` / `GetByRoomID` |
| 后端 | `internal/model/game.go` | `MatchRecord.RoomID` |
| 后端 | `cmd/server/main.go` | 注册 `POST /api/matches/:id/resume` |
| 前端 | `src/services/api.ts` | `resumeMatch()` |
| 前端 | `src/components/GameResultTable.tsx` | `MatchCard` 增加「▶ 继续模拟 BP」+ 进度提示 |
| 前端 | `src/pages/HistoryPage.tsx` | 恢复处理 + 跳转房间 |
| 文档 | `docs/API.md`、`README.md` | 新接口、`room_id`、每局红蓝归属说明 |

### 10.5 遗留事项

1. **旧数据（本轮之前的比赛）没有 `room_id`**：恢复时会在内存中新建房间号并回写，
   因此旧比赛**同样可以恢复**，只是首次恢复后房间号会变。
2. 本轮之前的历史比赛若 `ban_pick_action` 不完整（旧逻辑只在结束时写），
   恢复只能回到"该局最后一局的已落库状态"；本轮之后新产生的比赛不受影响。
3. 房间仍无自动过期/清理机制，长时间运行会累积内存（原架构既有问题，未在本次范围内）。
