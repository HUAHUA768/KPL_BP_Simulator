package service

import "testing"

// ============================================================================
// 回归测试：用户实测报告的两个缺陷（§8.9）
//
// 报告原文（AG=用过元流的一方，狼队=对手）：
//
//	1. 前端问题：AG 本局选了元射，但是狼队后续还能 ban 元坦（**局内**路径）
//	2. AG 第一局用了元射，狼队第二把依旧能 ban 元坦
//	   （正常情况下是所有元流狼队都不能 ban）（**跨局**路径）
//
// 共同根因：需求 2 的"若 A 队已使用元射，则 **B 队不用再 ban 其它四个元流**"
// 被理解成了"对 A 队 ban 权的描述"，而它其实是一条**对 B 队的禁止规则**。
// 修复：两条路径都补上「对手 CanBan = false」。
// ============================================================================

// TestRegression_WolfCannotBanTank_CrossGame 报告 2：跨局路径。
//
// 场景：AG(101) 第1局蓝方 pick 元射 → 第2局 狼队(102) 对元坦不应有 ban 权。
func TestRegression_WolfCannotBanTank_CrossGame(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter) // AG=蓝101 用元射

	engine2 := startNextGame(t, repo, 1, 2, 101, 102) // 第2局不换边

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}
	for _, id := range others {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		// 报告 2 的核心断言：狼队（红方）不可 ban
		if p.RedCanBan {
			t.Errorf("【报告2未修复】英雄%d：AG 已用过元射，狼队不应再能 ban 它", id)
		}
		// 顺手确认：AG 自己的 ban 权仍在（需求2 的另一半）
		if !p.BlueCanBan {
			t.Errorf("英雄%d：AG 自身的 ban 权必须保留", id)
		}
		// AG 不可 pick、狼队仍可 pick（五选一）
		if p.BlueCanPick {
			t.Errorf("英雄%d：AG 已用过元射，不可再 pick", id)
		}
		if !p.RedCanPick {
			t.Errorf("英雄%d：狼队仍应有五选一权力", id)
		}
	}
}

// TestRegression_WolfCannotBanTank_IntraGame 报告 1：局内路径。
//
// 场景：AG(蓝) 在本局 pick 元射 → **同一局内**狼队(红) 对元坦不应有 ban 权。
func TestRegression_WolfCannotBanTank_IntraGame(t *testing.T) {
	repo := newMockGameRepo()
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	if err := engine.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	others := []int{ylTank, ylMage, ylAssasin, ylSupport}

	// pick 之前：狼队对元坦是可 ban 的
	if ok, _ := engine.heroCan(Red, Ban, ylTank); !ok {
		t.Fatal("前置条件：AG 未使用元流前，狼队应可 ban 元坦")
	}

	// AG(蓝) 在第5步 pick 元射
	if err := playTo(engine, 4); err != nil {
		t.Fatalf("走到第4步失败: %v", err)
	}
	if err := engine.ExecuteAction(Pick, ylShooter); err != nil {
		t.Fatalf("AG pick 元射失败: %v", err)
	}
	engine.Advance()

	// 报告 1 的核心断言：局内，狼队（红方）不可 ban
	for _, id := range others {
		okBan, status := engine.heroCan(Red, Ban, id)
		if okBan {
			t.Errorf("【报告1未修复】英雄%d：AG 本局已选元射，狼队不应再能 ban 它", id)
		}
		// 应以 restricted_by_rule 拒绝（而不是"已被占用"等其它原因）
		if status != HeroRestrictedByRule {
			t.Errorf("英雄%d：应以 restricted_by_rule 拒绝狼队 ban，实际 status=%v", id, status)
		}
		// AG 自己的 ban 权仍在
		if okBanBlue, _ := engine.heroCan(Blue, Ban, id); !okBanBlue {
			t.Errorf("英雄%d：AG 自身 ban 权必须保留", id)
		}
		// 狼队仍可 pick（五选一）
		if okPick, _ := engine.heroCan(Red, Pick, id); !okPick {
			t.Errorf("英雄%d：狼队仍应有五选一权力", id)
		}
	}
}

// TestRegression_RedUsedYuanliu_BlueCannotBan 对称性：
// 若用过元流的是**红方**，则蓝方（对手）不可 ban。
//
// 规则不得偏向任何一方。
func TestRegression_RedUsedYuanliu_BlueCannotBan(t *testing.T) {
	repo := newMockGameRepo()
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	if err := engine.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	// 走到第6步（红pick1）由红方 pick 元射
	if err := playTo(engine, 5); err != nil {
		t.Fatalf("走到第5步失败: %v", err)
	}
	if err := engine.ExecuteAction(Pick, ylShooter); err != nil {
		t.Fatalf("红方 pick 元射失败: %v", err)
	}
	engine.Advance()

	for _, id := range []int{ylTank, ylMage, ylAssasin, ylSupport} {
		// 蓝方（对手）不可 ban
		if ok, _ := engine.heroCan(Blue, Ban, id); ok {
			t.Errorf("英雄%d：红方已用过元流，蓝方不应再能 ban", id)
		}
		// 红方（pick 方）保留 ban 权
		if ok, _ := engine.heroCan(Red, Ban, id); !ok {
			t.Errorf("英雄%d：红方自身 ban 权必须保留", id)
		}
		// 蓝方仍可 pick
		if ok, _ := engine.heroCan(Blue, Pick, id); !ok {
			t.Errorf("英雄%d：蓝方仍应有五选一权力", id)
		}
	}
}

// TestRegression_NoLockWhenUnused 对照：双方都未用过元流时，
// 双方对五个元流都保有 ban 权（修复不得误伤正常场景）。
func TestRegression_NoLockWhenUnused(t *testing.T) {
	repo := newMockGameRepo()
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	if err := engine.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	for _, id := range []int{ylTank, ylMage, ylShooter, ylAssasin, ylSupport} {
		if ok, _ := engine.heroCan(Blue, Ban, id); !ok {
			t.Errorf("英雄%d：未使用元流时蓝方应可 ban", id)
		}
		if ok, _ := engine.heroCan(Red, Ban, id); !ok {
			t.Errorf("英雄%d：未使用元流时红方应可 ban", id)
		}
		if ok, _ := engine.heroCan(Blue, Pick, id); !ok {
			t.Errorf("英雄%d：未使用元流时蓝方应可 pick", id)
		}
		if ok, _ := engine.heroCan(Red, Pick, id); !ok {
			t.Errorf("英雄%d：未使用元流时红方应可 pick", id)
		}
	}
}

// TestRegression_BanDoesNotRemoveOpponentBan 对照：ban 不触发锁定，
// 因此也不应剥夺对手的 ban 权。
func TestRegression_BanDoesNotRemoveOpponentBan(t *testing.T) {
	repo := newMockGameRepo()
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	if err := engine.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	// 蓝方 ban 元射（不是 pick）
	if err := engine.ExecuteAction(Ban, ylShooter); err != nil {
		t.Fatalf("蓝方 ban 元射失败: %v", err)
	}
	engine.Advance()

	// 其余四个：双方 ban 权都应完好
	for _, id := range []int{ylTank, ylMage, ylAssasin, ylSupport} {
		if ok, _ := engine.heroCan(Red, Ban, id); !ok {
			t.Errorf("英雄%d：ban 元射不应剥夺红方对其它元流的 ban 权", id)
		}
		if ok, _ := engine.heroCan(Blue, Ban, id); !ok {
			t.Errorf("英雄%d：蓝方 ban 权应完好", id)
		}
		if ok, _ := engine.heroCan(Blue, Pick, id); !ok {
			t.Errorf("英雄%d：ban 不触发锁定，蓝方应仍可 pick", id)
		}
	}
}

// TestRegression_CrossGame_SideSwap_WolfCannotBan 换边后依然成立：
// 用过元流的队换到红方 → 蓝方（对手）不可 ban。
func TestRegression_CrossGame_SideSwap_WolfCannotBan(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter) // AG=101 蓝，用元射

	// 第2局换边：AG 变红(101)，狼队变蓝(102)
	engine2 := startNextGame(t, repo, 1, 2, 102, 101)

	for _, id := range []int{ylTank, ylMage, ylAssasin, ylSupport} {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		// 狼队（现为蓝方）不可 ban
		if p.BlueCanBan {
			t.Errorf("英雄%d：换边后狼队(蓝)仍不应能 ban（限制跟队伍走）", id)
		}
		// AG（现为红方）自身 ban 权保留
		if !p.RedCanBan {
			t.Errorf("英雄%d：AG(红) 自身 ban 权必须保留", id)
		}
	}
}

// ============================================================================
// 回归测试：第二轮实战反馈 —— "换边后，用过元射的队能 ban 自己用过的元射"（§9.7）
//
// 报告原文：狼队第一把选了元射，但是 AG 第三把依然能 ban 元射
//           （其它四个是不能 ban 的，说明其它四个没有bug）
//
// 真实数据（DB match_record id=60）：
//
//	第1局 blue=1(狼队) red=2(AG)  → 狼队 pick 元射(83)
//	第2局 blue=2(AG)   red=1(狼队) → AG pick 元坦(33)      ← 换边
//	第3局 blue=1(狼队) red=2(AG)  → 狼队 ban 元射(83) 成功 ← BUG
//
// 根因：`YuanliuLocked` 刻意把"用过的那个"排除在锁定集外，理由是其 pick 由
// 全局BP记忆处理。但**记忆只看上一局**（第2局狼队在红方没选元流），
// 第3局记忆已不再覆盖元射 → 元射既没被锁 pick、也没被剥夺对手 ban。
// 修复：`applyYuanliuLock` 的"对手 ban 剥夺"必须覆盖**整组五个**（含用过的那个）。
// ============================================================================

// TestRegression02_TeamCanBanOwnUsedYuanliu_Match60 精确复现 DB match 60。
func TestRegression02_TeamCanBanOwnUsedYuanliu_Match60(t *testing.T) {
	repo := newMockGameRepo()

	// 第1局：狼队(1) 蓝方，pick 元射
	repo.playGameWithPick(t, 60, 1, 1, 2, 5, ylShooter)
	// 第2局：换边，AG(2) 蓝方，pick 元坦
	repo.playGameWithPick(t, 60, 2, 2, 1, 5, ylTank)

	// 第3局：换回，狼队(1) 蓝方、AG(2) 红方
	engine3 := startNextGame(t, repo, 60, 3, 1, 2)

	// ── 核心断言：狼队（蓝，第1局用过元射）不得 ban 元射 ──
	if p := engine3.Permissions[ylShooter]; p == nil {
		t.Fatal("元射应有权限记录")
	} else if p.BlueCanBan {
		t.Error("【报告未修复】狼队第1局用过元射，第3局不应能 ban 元射")
	}

	// ── 狼队用过元射 → 对整组五个元流都不得 ban ──
	for _, id := range []int{ylTank, ylMage, ylShooter, ylAssasin, ylSupport} {
		p := engine3.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		if p.BlueCanBan {
			t.Errorf("英雄%d：狼队用过元流，不应能 ban 任何元流", id)
		}
	}

	// ── AG（红，第2局用过元坦）同样不得 ban 任何元流 ──
	for _, id := range []int{ylTank, ylMage, ylShooter, ylAssasin, ylSupport} {
		if p := engine3.Permissions[id]; p != nil && p.RedCanBan {
			t.Errorf("英雄%d：AG 用过元坦，不应能 ban 任何元流", id)
		}
	}

	// ── 双方都用过 → 双方对整组都不可 pick（"只能选一个"）──
	for _, id := range []int{ylTank, ylMage, ylShooter, ylAssasin, ylSupport} {
		p := engine3.Permissions[id]
		if p == nil {
			continue
		}
		if p.BlueCanPick {
			t.Errorf("英雄%d：狼队已用过元流，不可再 pick", id)
		}
		if p.RedCanPick {
			t.Errorf("英雄%d：AG 已用过元流，不可再 pick", id)
		}
	}
}

// TestRegression02_UsedHeroBanLockedInGame 局内路径：对手也不得 ban "刚被用的那个"。
//
// 局内 pick 后，该英雄已被移出 HeroPool，但对手对它仍可能有 ban 意图；
// 权限位必须同步置 false（与跨局路径保持一致）。
func TestRegression02_UsedHeroBanLockedInGame(t *testing.T) {
	repo := newMockGameRepo()
	gs := NewGameService(&mockHeroRepo{heroes: makeYuanliuHeroModels()}, repo, map[int]bool{})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame 失败: %v", err)
	}
	if err := engine.Start(); err != nil {
		t.Fatalf("Start 失败: %v", err)
	}

	// 蓝方第5步 pick 元射
	if err := playTo(engine, 4); err != nil {
		t.Fatalf("走到第4步失败: %v", err)
	}
	if err := engine.ExecuteAction(Pick, ylShooter); err != nil {
		t.Fatalf("蓝方 pick 元射失败: %v", err)
	}
	engine.Advance()

	// 元射自身：对手（红）不得 ban
	p := engine.Permissions[ylShooter]
	if p == nil {
		t.Fatal("元射应有权限记录")
	}
	if p.RedCanBan {
		t.Error("【报告未修复】蓝方已 pick 元射，红方不应能 ban 元射（含用过的那个）")
	}
	// 蓝方（用过方）自己的 ban 权保留
	if !p.BlueCanBan {
		t.Error("蓝方自身对元射的 ban 权应保留")
	}

	// 其余四个：红方同样不得 ban
	for _, id := range []int{ylTank, ylMage, ylAssasin, ylSupport} {
		if q := engine.Permissions[id]; q != nil && q.RedCanBan {
			t.Errorf("英雄%d：红方不应能 ban", id)
		}
	}
}

// TestRegression02_OnlyOneSideUsed 只有一方用过时：未用过方保留完整 ban 权？
// —— 不，规则是"对手不得 ban"，因此未用过方作为对手也不得 ban。
// 但未用过方**自己**仍是 pick 方时保有 ban 权（用于后续轮次）。
func TestRegression02_OnlyOneSideUsed(t *testing.T) {
	repo := newMockGameRepo()
	repo.playGameWithPick(t, 1, 1, 101, 102, 5, ylShooter) // 仅 101 用过元射

	engine2 := startNextGame(t, repo, 1, 2, 101, 102)

	for _, id := range []int{ylTank, ylMage, ylShooter, ylAssasin, ylSupport} {
		p := engine2.Permissions[id]
		if p == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		// 用过方(蓝=101)：ban 权保留、pick 权被锁
		if !p.BlueCanBan {
			t.Errorf("英雄%d：用过方自身 ban 权应保留", id)
		}
		if p.BlueCanPick {
			t.Errorf("英雄%d：用过方不可再 pick", id)
		}
		// 对手(红=102)：可 pick（五选一）、不可 ban
		if !p.RedCanPick {
			t.Errorf("英雄%d：对手仍应有五选一权力", id)
		}
		if p.RedCanBan {
			t.Errorf("英雄%d：对手不应能 ban", id)
		}
	}
}
