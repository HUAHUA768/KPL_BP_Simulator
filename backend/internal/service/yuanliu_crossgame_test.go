package service

import (
	"testing"

	"kpl-bp-simulator/internal/model"
)

// ============================================================================
// 元流之子「五选一」锁定 — 跨局 / 换边 / 与全局BP记忆叠加 的集成测试
//
// 覆盖方案 §3.6 的集成类用例。这些用例是本规则最重要的防线：
// 纯函数正确不代表派生链路正确（颜色/队伍翻译、跨局累计、ban 不锁都在这里）。
// ============================================================================

// 测试用固定 ID —— 与真实库一致（33/60/83/105/132），但代码不依赖这些数字。
const (
	ylTank    = 33  // 元流之子·坦克
	ylMage    = 60  // 元流之子·法师
	ylShooter = 83  // 元流之子·射手（元射）
	ylAssasin = 105 // 元流之子·刺客
	ylSupport = 132 // 元流之子·辅助
)

// makeYuanliuHeroModels 构造含五个元流之子 + 若干普通英雄的英雄表。
//
// 普通英雄 ID 覆盖 1..60：范围要足够宽，因为测试辅助函数会用
// 「ban 步 1..20 / pick 步 21..41」的分配策略避免同局内冲突。
// 元流用真实 ID（33/60/83/105/132），模拟真实英雄池的"稀疏"分布。
func makeYuanliuHeroModels() []*model.Hero {
	heroes := make([]*model.Hero, 0, 65)
	for i := 1; i <= 60; i++ {
		// 跳过元流占用的真实 ID，避免生成重名重复英雄
		switch i {
		case ylTank, ylMage, ylShooter, ylAssasin, ylSupport:
			continue
		}
		heroes = append(heroes, &model.Hero{ID: i, Name: "普通英雄"})
	}
	heroes = append(heroes,
		&model.Hero{ID: ylTank, Name: "元流之子·坦克"},
		&model.Hero{ID: ylMage, Name: "元流之子·法师"},
		&model.Hero{ID: ylShooter, Name: "元流之子·射手"},
		&model.Hero{ID: ylAssasin, Name: "元流之子·刺客"},
		&model.Hero{ID: ylSupport, Name: "元流之子·辅助"},
	)
	return heroes
}

// pickHeroAtStep 走完一局，并在指定步序 pick 指定英雄（其余步用普通英雄填充）。
//
// 步序 → 动作方的映射（与 bp_engine.go getTurnInfo 一致）：
//
//	1蓝ban 2红ban 3蓝ban 4红ban 5蓝pick 6红pick 7红pick 8蓝pick 9蓝pick 10红pick
//	11红ban 12蓝ban 13红ban 14蓝ban 15红ban 16蓝ban
//	17红pick 18蓝pick 19蓝pick 20红pick
//
// step 必须是 pick 步（5~10、17~20）。
func (m *mockGameRepo) playGameWithPick(t *testing.T, matchID, gameNumber, blueTeamID, redTeamID, step, heroID int) int {
	t.Helper()

	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, m, map[int]bool{})
	engine, err := gs.StartGame(matchID, gameNumber, blueTeamID, redTeamID)
	if err != nil {
		t.Fatalf("第%d局 StartGame 失败: %v", gameNumber, err)
	}
	engine.Start()

	for i := 1; i <= 20; i++ {
		info := engine.GetTurnInfo()
		useHero := heroID
		if i != step {
			// 用普通英雄填充（ID 1..20），避开元流组
			useHero = i
		}
		if err := engine.ExecuteAction(info.Action, useHero); err != nil {
			t.Fatalf("第%d局第%d步(%v)失败: %v", gameNumber, i, info.Action, err)
		}
		engine.Advance()
	}

	// 建局行 + 落库（与现有测试同一套路：CreateGame 需先于 FinishGame）
	gameID, err := m.CreateGame(&model.Game{
		MatchID: matchID, GameNumber: gameNumber,
		BlueTeamID: blueTeamID, RedTeamID: redTeamID,
	})
	if err != nil {
		t.Fatalf("CreateGame 失败: %v", err)
	}
	if err := gs.FinishGame(gameID, engine, nil); err != nil {
		t.Fatalf("第%d局 FinishGame 失败: %v", gameNumber, err)
	}
	return gameID
}

// banHeroAtStep 走完一局，并在指定步序 ban 指定英雄（**不产生 pick 记录**）。
// 用于验证"ban 不触发锁定"。
func (m *mockGameRepo) playGameWithBan(t *testing.T, matchID, gameNumber, blueTeamID, redTeamID, step, heroID int) int {
	t.Helper()

	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, m, map[int]bool{})
	engine, err := gs.StartGame(matchID, gameNumber, blueTeamID, redTeamID)
	if err != nil {
		t.Fatalf("第%d局 StartGame 失败: %v", gameNumber, err)
	}
	engine.Start()

	isPickStep := func(i int) bool { return (i >= 5 && i <= 10) || (i >= 17 && i <= 20) }

	for i := 1; i <= 20; i++ {
		info := engine.GetTurnInfo()
		useHero := i // ban 步：用 1..20 区间
		if i == step {
			useHero = heroID
		} else if isPickStep(i) {
			// pick 步用 21..41 区间，与 ban 步的 1..20 不重叠，
			// 避免"同一局内某英雄先被 pick 又被 ban"造成冲突
			useHero = 20 + i
		}
		if err := engine.ExecuteAction(info.Action, useHero); err != nil {
			t.Fatalf("第%d局第%d步(%v, 英雄%d)失败: %v", gameNumber, i, info.Action, useHero, err)
		}
		engine.Advance()
	}

	gameID, err := m.CreateGame(&model.Game{
		MatchID: matchID, GameNumber: gameNumber,
		BlueTeamID: blueTeamID, RedTeamID: redTeamID,
	})
	if err != nil {
		t.Fatalf("CreateGame 失败: %v", err)
	}
	if err := gs.FinishGame(gameID, engine, nil); err != nil {
		t.Fatalf("第%d局 FinishGame 失败: %v", gameNumber, err)
	}
	return gameID
}

// startNextGame 在第 N 局已落库后，开第 N+1 局并返回引擎。
func startNextGame(t *testing.T, repo *mockGameRepo, matchID, gameNumber, blueTeamID, redTeamID int) *Engine {
	t.Helper()
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{})
	engine, err := gs.StartGame(matchID, gameNumber, blueTeamID, redTeamID)
	if err != nil {
		t.Fatalf("第%d局 StartGame 失败: %v", gameNumber, err)
	}
	return engine
}

// ============================================================================
// 用例 0（最关键）：**局内**动态锁定 —— 端到端测试暴露出的真实缺陷
// ============================================================================

// TestGameService_YuanliuLock_IntraGame 需求 1 的本义：**同一局之内**
// 「一旦使用了元射，则其它四个元流都不能再选」。
//
// ⚠️ 这个用例是施工中端到端实测发现缺陷后补上的：
// 初版实现只在 StartGame 里按"历史局"派生权限表，导致第 1 局 pick 元射后
// 其余四个仍可 pick（权限表开局即固化，本局内的 pick 无法回溯修改）。
// 修复方式：把互斥组注册到 Engine，在 handlePick 后动态收紧权限。
func TestGameService_YuanliuLock_IntraGame(t *testing.T) {
	repo := newMockGameRepo()
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{})

	engine, err := gs.StartGame(1, 1, 101, 102) // 第 1 局，无任何历史
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	if err := engine.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}

	// --- 局内：pick 之前，其余四个应当可 pick ---
	for _, id := range others {
		if ok, _ := engine.heroCan(Blue, Pick, id); !ok {
			t.Fatalf("pick 元射之前，蓝方应可 pick 英雄%d", id)
		}
	}

	// 走到第5步（蓝pick1）pick 元射
	if err := playTo(engine, 4); err != nil {
		t.Fatalf("走到第4步失败: %v", err)
	}
	if err := engine.ExecuteAction(Pick, ylShooter); err != nil {
		t.Fatalf("蓝方 pick 元射失败: %v", err)
	}
	engine.Advance()

	// --- 局内：pick 元射之后，蓝方对另外四个应当**立刻**不可 pick ---
	for _, id := range others {
		ok, status := engine.heroCan(Blue, Pick, id)
		if ok {
			t.Errorf("【局内锁定失效】蓝方已 pick 元射，仍可 pick 英雄%d", id)
		}
		if status != HeroRestrictedByRule {
			t.Errorf("英雄%d：应以 restricted_by_rule 拒绝，实际 status=%v", id, status)
		}
		// ban 权必须保留（需求 2）
		if okBan, _ := engine.heroCan(Blue, Ban, id); !okBan {
			t.Errorf("英雄%d：局内锁定后蓝方 ban 权仍须保留", id)
		}
	}

	// --- 红方不受影响，仍可五选一（需求 2）---
	for _, id := range others {
		if ok, _ := engine.heroCan(Red, Pick, id); !ok {
			t.Errorf("英雄%d：红方未使用元流，局内仍应可 pick", id)
		}
	}
}

// playTo 从当前步推进到第 target 步（含），用普通英雄填充。
// 用于把引擎推到指定状态后再做关键断言。
//
// 填充策略：ban 步用 1..20，pick 步用 61..62（英雄池 1..60 内的空闲区间），
// 均避开元流 ID 与互斥组，防止提前触发锁定。
func playTo(engine *Engine, target int) error {
	isPickStep := func(t int) bool { return (t >= 5 && t <= 10) || (t >= 17 && t <= 20) }
	for int(engine.CurrentTurn) <= target {
		info := engine.GetTurnInfo()
		hero := int(info.Turn) // ban 步：1..20
		if isPickStep(int(info.Turn)) {
			hero = 50 + int(info.Turn) - 4 // pick 步：51..62，避开元流与 ban 区间
		}
		if err := engine.ExecuteAction(info.Action, hero); err != nil {
			return err
		}
		engine.Advance()
	}
	return nil
}

// TestGameService_YuanliuLock_IntraGame_PersistsAcrossGame 局内锁定 + 跨局累计的衔接：
// 第 1 局 pick 元射 → 第 2 局由 StartGame 的历史派生同样锁定（两条路径结果一致）。
func TestGameService_YuanliuLock_IntraGame_PersistsAcrossGame(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter)

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	// 跨局路径：权限表在开局时就已锁定
	for _, id := range []int{ylTank, ylMage, ylAssasin, ylSupport} {
		if ok, _ := engine2.heroCan(Blue, Pick, id); ok {
			t.Errorf("英雄%d：第2局开局即应被锁（跨局累计路径）", id)
		}
		if ok, _ := engine2.heroCan(Blue, Ban, id); !ok {
			t.Errorf("英雄%d：第2局 ban 权仍须保留", id)
		}
	}
}

// TestGameService_YuanliuLock_RedSideIntraGame 局内锁定对红方同样生效
// （规则不偏向任何一方）。
func TestGameService_YuanliuLock_RedSideIntraGame(t *testing.T) {
	repo := newMockGameRepo()
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	if err := engine.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	// 走到第6步（红pick1）pick 元射
	if err := playTo(engine, 5); err != nil {
		t.Fatalf("走到第5步失败: %v", err)
	}
	if err := engine.ExecuteAction(Pick, ylShooter); err != nil {
		t.Fatalf("红方 pick 元射失败: %v", err)
	}
	engine.Advance()

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}
	for _, id := range others {
		if ok, _ := engine.heroCan(Red, Pick, id); ok {
			t.Errorf("红方已 pick 元射，不应再可 pick 英雄%d", id)
		}
		if ok, _ := engine.heroCan(Red, Ban, id); !ok {
			t.Errorf("英雄%d：红方 ban 权须保留", id)
		}
		// 蓝方不受影响
		if ok, _ := engine.heroCan(Blue, Pick, id); !ok {
			t.Errorf("英雄%d：蓝方未使用元流，应仍可 pick", id)
		}
	}
}

// TestGameService_YuanliuLock_BanDoesNotTriggerIntraGameLock
// 局内 ban 元流**不**触发锁定（与跨局的"ban 不计入"语义一致）。
func TestGameService_YuanliuLock_BanDoesNotTriggerIntraGameLock(t *testing.T) {
	repo := newMockGameRepo()
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	if err := engine.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	// 第1步（蓝ban1）ban 元射
	if err := engine.ExecuteAction(Ban, ylShooter); err != nil {
		t.Fatalf("蓝方 ban 元射失败: %v", err)
	}
	engine.Advance()

	// 蓝方 ban 了元射，但从未 pick 任何元流 → 其余四个仍应可 pick
	for _, id := range []int{ylTank, ylMage, ylAssasin, ylSupport} {
		if ok, _ := engine.heroCan(Blue, Pick, id); !ok {
			t.Errorf("英雄%d：局内 ban 元流不应触发五选一锁定", id)
		}
	}
}

// ============================================================================
// 用例 1：第 2 局 —— 蓝方第 1 局 pick 过元射
// ============================================================================

// TestGameService_YuanliuLock_SecondGame 需求 1 + 需求 2 的核心用例。
//
// 场景：第1局 蓝(101) 在第5步 pick 元射(83)。第2局 不换边，蓝仍是101。
//
//	蓝方（用过元射）：其余四个 {坦克,法师,刺客,辅助} → 不可 pick
//	红方（未用过）  ：四个 → 仍可 pick（B 队依然五选一）
func TestGameService_YuanliuLock_SecondGame(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter) // 第5步=蓝pick1

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}
	for _, id := range others {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		// 需求 1：蓝方已用过元射 → 其余四个不可再选
		if p.BlueCanPick {
			t.Errorf("英雄%d：蓝方已用过元射，本局不可再 pick", id)
		}
		// 需求 2：但蓝方**自身** ban 权完整保留
		if !p.BlueCanBan {
			t.Errorf("英雄%d：蓝方 ban 权必须保留（需求2），实际 BlueCanBan=false", id)
		}
		// 需求 2：B 队（红方）依然可以五选一
		if !p.RedCanPick {
			t.Errorf("英雄%d：红方未用过元流，应仍可 pick（B队依然五选一）", id)
		}
		// 需求 2："B 队不用再 ban 其它四个元流" → 红方**不可 ban**
		// ⚠️ 施工中曾漏做此条，导致"AG 选了元射，狼队仍能 ban 元坦"（§8.9）
		if p.RedCanBan {
			t.Errorf("英雄%d：A 队已用过元流，B 队不应再能 ban 其它四个元流，实际 RedCanBan=true", id)
		}
	}
}

// TestGameService_YuanliuLock_KeepsBanRight 需求 2 专项：
// 已用过元射的蓝方，对**五个元流全部**保有 ban 权。
//
// 这是最容易犯的错误（把 CanBan 一起压掉），因此单独立一个用例把五个都查一遍。
func TestGameService_YuanliuLock_KeepsBanRight(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter)

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	all := []int{ylTank, ylMage, ylShooter, ylAssasin, ylSupport}
	for _, id := range all {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		if !p.BlueCanBan {
			t.Errorf("英雄%d：需求2 明确 A 队依然有 ban 五个元流的权力，BlueCanBan 不得为 false", id)
		}
	}
}

// TestGameService_YuanliuLock_OpponentCanStillPick 需求 2 专项：
// B 队依然可以进行元流五选一 —— 即 B 队对仍未用过的元流保有 pick 权。
func TestGameService_YuanliuLock_OpponentCanStillPick(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter)

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}
	pickable := 0
	for _, id := range others {
		if engine2.Permissions[id].RedCanPick {
			pickable++
		}
	}
	if pickable != len(others) {
		t.Errorf("红方（未用过元流）应能五选一，实际可 pick 的未用元流 = %d/%d", pickable, len(others))
	}
}

// ============================================================================
// 用例 2：换边后限制仍"跟队伍走"（§8.1 回归防护）
// ============================================================================

// TestGameService_YuanliuLock_CrossGame_TeamBased 换边防回归。
//
// 场景：第1局 蓝(101) pick 元射。第2局**换边** → 蓝=102, 红=101。
// 期望：限制跟着 101 走（101 现在红了）→ 红方对四个未用元流不可 pick；
//
//	蓝方（102）不受影响，仍可 pick。
func TestGameService_YuanliuLock_CrossGame_TeamBased(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter) // 101 是蓝，pick 元射

	// 第2局换边：102 变蓝，101 变红
	engine2 := startNextGame(t, repo, 1, 2, 102, 101)

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}
	for _, id := range others {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		// 101 队（本局红方）用过元射 → 红方不可 pick
		if p.RedCanPick {
			t.Errorf("英雄%d：换边后 101 变红，它用过元射，红方本局不可 pick（限制跟队伍走）", id)
		}
		// 101 队的 ban 权仍保留
		if !p.RedCanBan {
			t.Errorf("英雄%d：101 队 ban 权必须保留，实际 RedCanBan=false", id)
		}
		// 102 队（本局蓝方）未用过 → 仍可 pick
		if !p.BlueCanPick {
			t.Errorf("英雄%d：102 队未用过元流，应仍可 pick（限制不得加错队）", id)
		}
	}
}

// ============================================================================
// 用例 3：ban 不触发锁定（需求方确认的边界）
// ============================================================================

// TestGameService_YuanliuLock_BanDoesNotLock 第1局只 ban 元射（未 pick）→ 第2局五选一权力完整。
func TestGameService_YuanliuLock_BanDoesNotLock(t *testing.T) {
	repo := newMockGameRepo()
	// 第1步 = 蓝ban1：蓝方 ban 掉元射，但从未 pick 任何元流
	repo.playGameWithBan(t, 1, 1, 101, 102, 1, ylShooter)

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}
	for _, id := range others {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		// 从未 pick 过元流 → 不锁定，蓝方仍可自由 pick
		if !p.BlueCanPick {
			t.Errorf("英雄%d：第1局只 ban 未 pick，不应触发元流锁定", id)
		}
		if !p.BlueCanBan {
			t.Errorf("英雄%d：ban 权应完整", id)
		}
	}
}

// TestGameService_YuanliuLock_BannedShooterStillAllowsOthers
// 元射被对手 ban 掉、且自己从未用过元流 → 自己仍可选其它四个（被 ban 不锁定）。
//
// 场景：第1局红方(102)在第2步 ban 元射。第2局 102 不用受任何元流锁定。
func TestGameService_YuanliuLock_BannedShooterStillAllowsOthers(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithBan(t, 1, 1, 101, 102, 2, ylShooter) // 第2步=红ban1

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}
	for _, id := range others {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		if !p.BlueCanPick {
			t.Errorf("英雄%d：被 ban 不触发锁定，蓝方应仍可 pick", id)
		}
		if !p.RedCanPick {
			t.Errorf("英雄%d：被 ban 不触发锁定，红方应仍可 pick", id)
		}
	}
}

// ============================================================================
// 用例 4：跨局累计（不止看上一局）
// ============================================================================

// TestGameService_YuanliuLock_AccumulatesAcrossGames 本规则最关键用例之一。
//
// 场景：第1局 蓝(101) pick 坦克(33) → 第2局 蓝(101) 想 pick 法师(60)。
// 期望：第 2 局**就已经**被锁（第1局用过坦克后，其余四个即不可选）。
// 这同时验证「一用过就锁整组剩余」与「累计跨局」两条语义。
func TestGameService_YuanliuLock_AccumulatesAcrossGames(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylTank) // 第1局蓝方 pick 坦克

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	// 第2局：蓝方用过坦克 → 法师/元射/刺客/辅助 全部不可 pick
	for _, id := range []int{ylMage, ylShooter, ylAssasin, ylSupport} {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		if p.BlueCanPick {
			t.Errorf("英雄%d：第1局用过坦克，第2局蓝方不可再 pick 任何其它元流", id)
		}
		if !p.BlueCanBan {
			t.Errorf("英雄%d：ban 权应保留", id)
		}
	}
}

// TestGameService_YuanliuLock_ThreeGames 三局累计：
// 第1局 101 pick 元射 → 第2局（换边后）仍锁定 → 第3局（再换回来）依然锁定。
//
// 验证规则在整个系列赛中持续生效，不因多次换边而丢失。
func TestGameService_YuanliuLock_ThreeGames(t *testing.T) {
	repo := newMockGameRepo()

	// 第1局：101 蓝，pick 元射
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter)
	// 第2局：换边 102蓝/101红 —— 需要第2局也有记录才能进第3局。
	// 用第20步（红方最后一次 pick）pick 普通英雄40，避免与其它步的英雄 ID 冲突。
	repo.playGameWithPick(t, 1, 2, 102, 101, 20, 40)
	// 第3局：换回来 101蓝/102红
	engine3 := startNextGame(t, repo, 1, 3, 101, 102)

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}
	for _, id := range others {
		p := engine3.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		if p.BlueCanPick {
			t.Errorf("英雄%d：第1局用过元射，第3局蓝方(101)仍不可 pick", id)
		}
		if !p.BlueCanBan {
			t.Errorf("英雄%d：第3局 ban 权仍应保留", id)
		}
	}
}

// ============================================================================
// 用例 5：与全局BP记忆叠加的正确性
// ============================================================================

// TestGameService_YuanliuLock_StackedWithMemory 元流锁 + 全局BP记忆叠加。
//
// 场景：第1局 蓝(101) pick 元射 → 第2局：
//   - 元射：101 自己选过 → 记忆规则给 {蓝可ban不可pick}；元流规则对它不新增约束
//   - 其余四个：元流规则把蓝方 pick 压掉；记忆规则未触及（全 true）
//
// 关键断言：两者叠加后蓝方对元射是 {CanBan:true, CanPick:false}，
// 对另外四个也是 {CanBan:true, CanPick:false} —— 形态一致，前端渲染自然统一。
func TestGameService_YuanliuLock_StackedWithMemory(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter)

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	// 元射自身：记忆规则（自己选过）保证不可再 pick，且红方可选
	shooter := engine2.Permissions[ylShooter]
	if shooter == nil {
		t.Fatal("元射应有权限记录")
	}
	if shooter.BlueCanPick {
		t.Error("元射：蓝方自己选过，不可再 pick（记忆规则）")
	}
	if !shooter.BlueCanBan {
		t.Error("元射：蓝方可 ban（记忆规则给的 ban 权）")
	}
	if !shooter.RedCanPick {
		t.Error("元射：对手可选（记忆规则）")
	}

	// 另外四个：元流规则压 pick，ban 全保留
	for _, id := range []int{ylTank, ylMage, ylAssasin, ylSupport} {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		if p.BlueCanPick {
			t.Errorf("英雄%d：元流锁定应压掉蓝方 pick", id)
		}
		if !p.BlueCanBan {
			t.Errorf("英雄%d：元流锁定不得压掉 ban", id)
		}
	}
}

// ============================================================================
// 用例 6：边界与健壮性
// ============================================================================

// TestGameService_YuanliuLock_NoYuanliuInPool 英雄池无元流英雄 → 不 panic、不产生**元流**锁定。
//
// ⚠️ 注意断言口径：第2局必然带有**全局BP记忆**的效果（第1局 pick 过的英雄
// 会变成不对称权限），那是既有规则的正常行为，**不是**元流锁定。
// 因此这里不能用"全部 FullPerm"作为判据，而应逐英雄核对：
//
//	被全局BP记忆规则命中的英雄 → 权限必须与该规则一致（与元流无关）
//	其余英雄                    → 必须保持 FullPerm（证明元流规则没有误伤）
func TestGameService_YuanliuLock_NoYuanliuInPool(t *testing.T) {
	repo := newMockGameRepo()
	gs := NewGameService(&mockHeroRepo{heroes: makeHeroModels(50)}, repo, map[int]bool{})

	// 第1局跑完并落库（用不含元流的英雄池）
	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	ids := make([]int, 20)
	for i := range ids {
		ids[i] = i + 1
	}
	if err := runFullGame(engine, ids); err != nil {
		t.Fatalf("第1局 BP 失败: %v", err)
	}
	gameID, _ := repo.CreateGame(&model.Game{MatchID: 1, GameNumber: 1, BlueTeamID: 101, RedTeamID: 102})
	if err := gs.FinishGame(gameID, engine, nil); err != nil {
		t.Fatalf("FinishGame 失败: %v", err)
	}

	// 第1局被 pick 过的英雄集合（这些会被全局BP记忆规则命中）
	pickedInGame1 := map[int]bool{}
	acts, _ := repo.ListBPActions(gameID)
	for _, a := range acts {
		if a.ActionType == "pick" {
			pickedInGame1[a.HeroID] = true
		}
	}
	// 防御性断言：确认本用例确实存在"被记忆规则命中"的英雄，
	// 否则下面的分支会全部走 FullPerm 路径，用例失去区分力
	if len(pickedInGame1) == 0 {
		t.Fatal("前置条件失败：第1局应有 pick 记录")
	}

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	// 未被第1局 pick 过的英雄（31..50）：必须保持 FullPerm
	for id := 31; id <= 50; id++ {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		if !p.CanBan(Blue) || !p.CanPick(Blue) || !p.CanBan(Red) || !p.CanPick(Red) {
			t.Errorf("英雄%d（未被 pick 过）在无元流英雄池时应为全权限，实际=%+v", id, p)
		}
	}

	// 被第1局 pick 过的英雄：权限应体现记忆规则（一方可ban不可pick、另一方反之），
	// 但绝不能是"双方都不可pick"这种元流锁定才有的形态。
	for id := range pickedInGame1 {
		if id > 50 {
			continue
		}
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		blueCanPick, redCanPick := p.CanPick(Blue), p.CanPick(Red)
		// 记忆规则下，恰有一方能 pick
		if !blueCanPick && !redCanPick {
			t.Errorf("英雄%d：无元流英雄池时不应出现「双方都不可pick」（那是元流锁定的形态）", id)
		}
	}
}

// TestGameService_YuanliuLock_FirstGameNoLock 第 1 局不产生任何元流锁定（五选一权力完整）。
func TestGameService_YuanliuLock_FirstGameNoLock(t *testing.T) {
	repo := newMockGameRepo()
	engine1 := startNextGame(t, repo, 1, 1, 101, 102)

	for _, id := range []int{ylTank, ylMage, ylShooter, ylAssasin, ylSupport} {
		p := engine1.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		if !p.CanBan(Blue) || !p.CanPick(Blue) || !p.CanBan(Red) || !p.CanPick(Red) {
			t.Errorf("第1局英雄%d应为全权限（五选一权力完整），实际=%+v", id, p)
		}
	}
}

// TestGameService_YuanliuLock_TournamentBannedNotRevived 赛事禁用英雄不被元流规则"复活"。
func TestGameService_YuanliuLock_TournamentBannedNotRevived(t *testing.T) {
	repo := newMockGameRepo()
	// 把元射设为赛事禁用
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{ylShooter: true})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	p := engine.Permissions[ylShooter]
	if p == nil {
		t.Fatal("元射应有权限记录")
	}
	if p.CanBan(Blue) || p.CanPick(Blue) || p.CanBan(Red) || p.CanPick(Red) {
		t.Errorf("赛事禁用英雄应保持四布尔全 false，实际=%+v", p)
	}
}
