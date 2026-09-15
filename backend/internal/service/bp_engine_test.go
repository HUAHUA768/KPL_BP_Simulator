package service

import (
	"fmt"
	"strings"
	"testing"
)

// ============================================================================
// 测试用 Validator — 简单实现：只允许英雄ID在1~100范围内
// ============================================================================

type mockValidator struct{}

func (m *mockValidator) ValidateBan(heroID int, side TeamSide) error {
	if heroID < 1 || heroID > 100 {
		return fmt.Errorf("英雄%d不在有效范围", heroID)
	}
	return nil
}

func (m *mockValidator) ValidatePick(heroID int, side TeamSide) error {
	if heroID < 1 || heroID > 100 {
		return fmt.Errorf("英雄%d不在有效范围", heroID)
	}
	return nil
}

// ============================================================================
// 辅助函数：构建初始英雄池
// ============================================================================

func makeHeroPool(ids ...int) map[int]bool {
	pool := make(map[int]bool)
	for _, id := range ids {
		pool[id] = true
	}
	return pool
}

// ============================================================================
// getTurnInfo 单元测试 — 覆盖全部20个turn
// ============================================================================

func TestGetTurnInfo(t *testing.T) {
	tests := []struct {
		name       string
		turn       BPTurn
		wantTurn   BPTurn
		wantRound  BPRound
		wantSide   TeamSide
		wantAction ActionType
		wantIndex  int
	}{
		// --- 第一循环（Blue先Ban，4ban+6pick）---
		{"Turn1_BlueBan1", Turn1_BlueBan1, Turn1_BlueBan1, Round1FirstCycle, Blue, Ban, 1},
		{"Turn2_RedBan1", Turn2_RedBan1, Turn2_RedBan1, Round1FirstCycle, Red, Ban, 1},
		{"Turn3_BlueBan2", Turn3_BlueBan2, Turn3_BlueBan2, Round1FirstCycle, Blue, Ban, 2},
		{"Turn4_RedBan2", Turn4_RedBan2, Turn4_RedBan2, Round1FirstCycle, Red, Ban, 2},
		{"Turn5_BluePick1", Turn5_BluePick1, Turn5_BluePick1, Round1FirstCycle, Blue, Pick, 1},
		{"Turn6_RedPick1", Turn6_RedPick1, Turn6_RedPick1, Round1FirstCycle, Red, Pick, 1},
		{"Turn7_RedPick2", Turn7_RedPick2, Turn7_RedPick2, Round1FirstCycle, Red, Pick, 2},
		{"Turn8_BluePick2", Turn8_BluePick2, Turn8_BluePick2, Round1FirstCycle, Blue, Pick, 2},
		{"Turn9_BluePick3", Turn9_BluePick3, Turn9_BluePick3, Round1FirstCycle, Blue, Pick, 3},
		{"Turn10_RedPick3", Turn10_RedPick3, Turn10_RedPick3, Round1FirstCycle, Red, Pick, 3},

		// --- 第二循环（Red先Ban，6ban+4pick）---
		{"Turn11_RedBan3", Turn11_RedBan3, Turn11_RedBan3, Round2SecondCycle, Red, Ban, 3},
		{"Turn12_BlueBan4", Turn12_BlueBan4, Turn12_BlueBan4, Round2SecondCycle, Blue, Ban, 4},
		{"Turn13_RedBan4", Turn13_RedBan4, Turn13_RedBan4, Round2SecondCycle, Red, Ban, 4},
		{"Turn14_BlueBan5", Turn14_BlueBan5, Turn14_BlueBan5, Round2SecondCycle, Blue, Ban, 5},
		{"Turn15_RedBan5", Turn15_RedBan5, Turn15_RedBan5, Round2SecondCycle, Red, Ban, 5},
		{"Turn16_BlueBan6", Turn16_BlueBan6, Turn16_BlueBan6, Round2SecondCycle, Blue, Ban, 6},
		{"Turn17_RedPick4", Turn17_RedPick4, Turn17_RedPick4, Round2SecondCycle, Red, Pick, 4},
		{"Turn18_BluePick4", Turn18_BluePick4, Turn18_BluePick4, Round2SecondCycle, Blue, Pick, 4},
		{"Turn19_BluePick5", Turn19_BluePick5, Turn19_BluePick5, Round2SecondCycle, Blue, Pick, 5},
		{"Turn20_RedPick5", Turn20_RedPick5, Turn20_RedPick5, Round2SecondCycle, Red, Pick, 5},

		// 无效值
		{"Invalid", TurnInvalid, TurnInvalid, 0, -1, -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := getTurnInfo(tt.turn)
			if info.Turn != tt.wantTurn {
				t.Errorf("Turn = %v, want %v", info.Turn, tt.wantTurn)
			}
			if info.Round != tt.wantRound {
				t.Errorf("Round = %v, want %v", info.Round, tt.wantRound)
			}
			if info.Side != tt.wantSide {
				t.Errorf("Side = %v, want %v", info.Side, tt.wantSide)
			}
			if info.Action != tt.wantAction {
				t.Errorf("Action = %v, want %v", info.Action, tt.wantAction)
			}
			if info.Index != tt.wantIndex {
				t.Errorf("Index = %v, want %v", info.Index, tt.wantIndex)
			}
		})
	}
}

// ============================================================================
// Engine 基本流程测试 — 完整20步
// ============================================================================

func TestEngine_StartAndFinish(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
	engine := NewBPEngine(1, 2, pool, make(map[int]bool), &mockValidator{})

	// 初始状态
	if engine.State != StateIdle {
		t.Errorf("初始状态应为 Idle，实际=%v", engine.State)
	}
	if engine.CurrentTurn != Turn1_BlueBan1 {
		t.Errorf("初始turn应为 Turn1_BlueBan1，实际=%v", engine.CurrentTurn)
	}

	// 启动
	if err := engine.Start(); err != nil {
		t.Fatalf("Start() 失败: %v", err)
	}
	if engine.State != StateRunning {
		t.Errorf("启动后状态应为 Running，实际=%v", engine.State)
	}

	// 模拟完整20步操作
	for i := 1; i <= 20; i++ {
		info := engine.GetTurnInfo()
		heroID := i // 用英雄ID=步序来简化测试

		err := engine.ExecuteAction(info.Action, heroID)
		if err != nil {
			t.Fatalf("第%d步失败: %v", i, err)
		}

		engine.Advance()

		expectedTurn := BPTurn(i + 1)
		if engine.CurrentTurn != expectedTurn && i < 20 {
			t.Errorf("第%d步后CurrentTurn=%v，期望=%v", i, engine.CurrentTurn, expectedTurn)
		}
	}

	// 全部完成后
	if !engine.IsFinished() {
		t.Errorf("20步完成后应处于 Finished 状态")
	}
	if engine.CurrentTurn != TurnMax+1 {
		t.Errorf("最终CurrentTurn应为%d，实际=%v", TurnMax+1, engine.CurrentTurn)
	}
}

// ============================================================================
// 第一轮流程验证 — ban ban ban ban → pick pick pick pick pick pick
// ============================================================================

func TestEngine_Round1_Flow(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
	engine := NewBPEngine(1, 2, pool, make(map[int]bool), &mockValidator{})
	engine.Start()

	// 步骤1-4: ban阶段（蓝ban1, 红ban1, 蓝ban2, 红ban2）
	banOrder := []struct {
		turn BPTurn
		side TeamSide
	}{
		{Turn1_BlueBan1, Blue},
		{Turn2_RedBan1, Red},
		{Turn3_BlueBan2, Blue},
		{Turn4_RedBan2, Red},
	}
	for _, expected := range banOrder {
		info := engine.GetTurnInfo()
		if info.Turn != expected.turn {
			t.Fatalf("当前turn=%v，期望=%v", info.Turn, expected.turn)
		}
		if info.Side != expected.side {
			t.Fatalf("侧别=%v，期望=%v", info.Side, expected.side)
		}
		if info.Action != Ban {
			t.Fatalf("动作=%v，期望=Ban", info.Action)
		}
		if info.Round != Round1FirstCycle {
			t.Fatalf("轮次=%v，期望=Round1FirstCycle", info.Round)
		}
		engine.ExecuteAction(Ban, int(engine.CurrentTurn))
		engine.Advance()
	}

	// 步骤5-10: pick阶段（蓝pick, 红pick, 红pick, 蓝pick, 蓝pick, 红pick）
	pickOrder := []struct {
		turn BPTurn
		side TeamSide
	}{
		{Turn5_BluePick1, Blue},
		{Turn6_RedPick1, Red},
		{Turn7_RedPick2, Red},
		{Turn8_BluePick2, Blue},
		{Turn9_BluePick3, Blue},
		{Turn10_RedPick3, Red},
	}
	for _, expected := range pickOrder {
		info := engine.GetTurnInfo()
		if info.Turn != expected.turn {
			t.Fatalf("当前turn=%v，期望=%v", info.Turn, expected.turn)
		}
		if info.Side != expected.side {
			t.Fatalf("侧别=%v，期望=%v", info.Side, expected.side)
		}
		if info.Action != Pick {
			t.Fatalf("动作=%v，期望=Pick", info.Action)
		}
		if info.Round != Round1FirstCycle {
			t.Fatalf("轮次=%v，期望=Round1FirstCycle", info.Round)
		}
		engine.ExecuteAction(Pick, int(engine.CurrentTurn))
		engine.Advance()
	}

	// 第一轮结束后，应进入第二轮
	if engine.Round != Round2SecondCycle {
		t.Fatalf("第一轮结束后Round应为Round2SecondCycle，实际=%v", engine.Round)
	}
}

// ============================================================================
// 第二轮流程验证 — ban ban ban ban ban ban → pick pick pick pick
// ============================================================================

func TestEngine_Round2_Flow(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
	engine := NewBPEngine(1, 2, pool, make(map[int]bool), &mockValidator{})
	engine.Start()

	// 先走完第一轮10步
	for i := 1; i <= 10; i++ {
		info := engine.GetTurnInfo()
		engine.ExecuteAction(info.Action, int(engine.CurrentTurn))
		engine.Advance()
	}

	// 步骤11-16: ban阶段（红ban3, 蓝ban4, 红ban4, 蓝ban5, 红ban5, 蓝ban6）
	banOrder := []struct {
		turn BPTurn
		side TeamSide
	}{
		{Turn11_RedBan3, Red},
		{Turn12_BlueBan4, Blue},
		{Turn13_RedBan4, Red},
		{Turn14_BlueBan5, Blue},
		{Turn15_RedBan5, Red},
		{Turn16_BlueBan6, Blue},
	}
	for _, expected := range banOrder {
		info := engine.GetTurnInfo()
		if info.Turn != expected.turn {
			t.Fatalf("当前turn=%v，期望=%v", info.Turn, expected.turn)
		}
		if info.Side != expected.side {
			t.Fatalf("侧别=%v，期望=%v", info.Side, expected.side)
		}
		if info.Action != Ban {
			t.Fatalf("动作=%v，期望=Ban", info.Action)
		}
		if info.Round != Round2SecondCycle {
			t.Fatalf("轮次=%v，期望=Round2SecondCycle", info.Round)
		}
		engine.ExecuteAction(Ban, int(engine.CurrentTurn))
		engine.Advance()
	}

	// 步骤17-20: pick阶段（红pick, 蓝pick, 蓝pick, 红pick）
	pickOrder := []struct {
		turn BPTurn
		side TeamSide
	}{
		{Turn17_RedPick4, Red},
		{Turn18_BluePick4, Blue},
		{Turn19_BluePick5, Blue},
		{Turn20_RedPick5, Red},
	}
	for _, expected := range pickOrder {
		info := engine.GetTurnInfo()
		if info.Turn != expected.turn {
			t.Fatalf("当前turn=%v，期望=%v", info.Turn, expected.turn)
		}
		if info.Side != expected.side {
			t.Fatalf("侧别=%v，期望=%v", info.Side, expected.side)
		}
		if info.Action != Pick {
			t.Fatalf("动作=%v，期望=Pick", info.Action)
		}
		if info.Round != Round2SecondCycle {
			t.Fatalf("轮次=%v，期望=Round2SecondCycle", info.Round)
		}
		engine.ExecuteAction(Pick, int(engine.CurrentTurn))
		engine.Advance()
	}

	// 全部完成后
	if !engine.IsFinished() {
		t.Error("20步完成后应处于 Finished 状态")
	}
}

// ============================================================================
// ban/pick 逻辑测试
// ============================================================================

func TestEngine_BanLogic(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5)
	engine := NewBPEngine(1, 2, pool, make(map[int]bool), &mockValidator{})
	engine.Start()

	info := engine.GetTurnInfo()
	if info.Action != Ban || info.Side != Blue {
		t.Fatalf("第一步应该是蓝ban，实际=侧%v 动作%v", info.Side, info.Action)
	}

	err := engine.ExecuteAction(Ban, 1)
	if err != nil {
		t.Fatalf("ban失败: %v", err)
	}
	engine.Advance()

	if len(engine.BlueBanned) != 1 || engine.BlueBanned[0] != 1 {
		t.Errorf("蓝队ban列表应为[1]，实际=%v", engine.BlueBanned)
	}
	if pool[1] {
		t.Error("英雄1应从可用池中移除")
	}
}

func TestEngine_PickLogic(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
	engine := NewBPEngine(1, 2, pool, make(map[int]bool), &mockValidator{})
	engine.Start()

	// 手动推进到Turn5（蓝pick1，蓝队首次选择）
	for i := 1; i < 5; i++ {
		info := engine.GetTurnInfo()
		engine.ExecuteAction(info.Action, i)
		engine.Advance()
	}

	info := engine.GetTurnInfo()
	if info.Action != Pick || info.Side != Blue {
		t.Fatalf("第5步应该是蓝pick，实际=侧%v 动作%v", info.Side, info.Action)
	}

	err := engine.ExecuteAction(Pick, 6)
	if err != nil {
		t.Fatalf("pick失败: %v", err)
	}
	engine.Advance()

	if len(engine.BluePicked) != 1 || engine.BluePicked[0] != 6 {
		t.Errorf("蓝队pick列表应为[6]，实际=%v", engine.BluePicked)
	}
}

func TestEngine_InvalidAction(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5)
	engine := NewBPEngine(1, 2, pool, make(map[int]bool), &mockValidator{})

	err := engine.ExecuteAction(Pick, 1)
	if err == nil {
		t.Fatal("期望报错，实际未报错")
	}
}

func TestEngine_DoubleBan(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5)
	engine := NewBPEngine(1, 2, pool, make(map[int]bool), &mockValidator{})

	engine.ExecuteAction(Ban, 1)
	engine.Advance()

	err := engine.ExecuteAction(Ban, 1)
	if err == nil {
		t.Fatal("ban已禁用的英雄应该报错")
	}
}

func TestEngine_StartTwice(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5)
	engine := NewBPEngine(1, 2, pool, make(map[int]bool), &mockValidator{})

	engine.Start()
	err := engine.Start()
	if err == nil {
		t.Fatal("重复启动应该报错")
	}
}

// ============================================================================
// 赛事禁用英雄测试（如雅典娜，不可ban也不可pick）
// ============================================================================

func TestEngine_TournamentBannedHero(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
	var athenaID = 99 // 雅典娜不在 heroPool 中，但在赛事禁用列表中
	engine := NewBPEngine(1, 2, pool, map[int]bool{athenaID: true}, &mockValidator{})
	engine.Start()

	// 第一步是蓝ban，尝试ban赛事禁用英雄 → 应该报"不可ban"
	err := engine.ExecuteAction(Ban, athenaID)
	if err == nil {
		t.Fatal("ban赛事禁用英雄应该报错")
	}
	if !strings.Contains(err.Error(), "不可ban") {
		t.Errorf("错误信息应包含'不可ban'，实际: %v", err)
	}
	if !strings.Contains(err.Error(), "赛事禁用") {
		t.Errorf("错误信息应说明'赛事禁用'，实际: %v", err)
	}

	// 赛事禁用英雄不应进入蓝方ban列表
	if len(engine.BlueBanned) != 0 {
		t.Errorf("赛事禁用英雄不应被ban，蓝队ban列表=%v", engine.BlueBanned)
	}
}

func TestEngine_TournamentBannedHero_NotPickable(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
	var athenaID = 99
	engine := NewBPEngine(1, 2, pool, map[int]bool{athenaID: true}, &mockValidator{})
	engine.Start()

	// 推进到第5步（蓝pick）
	for i := 1; i < 5; i++ {
		info := engine.GetTurnInfo()
		engine.ExecuteAction(info.Action, int(engine.CurrentTurn))
		engine.Advance()
	}

	err := engine.ExecuteAction(Pick, athenaID)
	if err == nil {
		t.Fatal("pick赛事禁用英雄应该报错")
	}
	if !strings.Contains(err.Error(), "不可pick") {
		t.Errorf("错误信息应包含'不可pick'，实际: %v", err)
	}
	if len(engine.BluePicked) != 0 {
		t.Errorf("赛事禁用英雄不应被pick，蓝队pick列表=%v", engine.BluePicked)
	}
}

// ============================================================================
// 对面已使用英雄测试（对方pick过的英雄，本方不能再ban）
// ============================================================================

func TestEngine_CannotBanOpponentPickedHero(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
	engine := NewBPEngine(1, 2, pool, make(map[int]bool), &mockValidator{})
	engine.Start()

	// 蓝方第5步 pick 英雄5
	for i := 1; i < 5; i++ {
		info := engine.GetTurnInfo()
		engine.ExecuteAction(info.Action, int(engine.CurrentTurn))
		engine.Advance()
	}
	engine.ExecuteAction(Pick, 5)
	engine.Advance()

	// 第6步轮到红方pick，此时红方想ban已被蓝方pick的英雄5 → 应该报错
	// 但第6步是pick不是ban，所以这个场景要直接测 handleBan 的行为
	err := engine.handleBan(Red, 5)
	if err == nil {
		t.Fatal("ban对面已使用的英雄应该报错")
	}
	if !strings.Contains(err.Error(), "已被蓝方选择") {
		t.Errorf("错误信息应说明'已被蓝方选择'，实际: %v", err)
	}
}

// ============================================================================
// heroStatus 状态查询测试
// ============================================================================

func TestHeroStatus(t *testing.T) {
	pool := makeHeroPool(1, 2, 3, 4, 5)
	engine := NewBPEngine(1, 2, pool, map[int]bool{99: true}, &mockValidator{})

	// 初始：全部可用
	if status := engine.heroStatus(1); status != HeroAvailable {
		t.Errorf("英雄1初始应为HeroAvailable，实际=%v", status)
	}
	// 赛事禁用
	if status := engine.heroStatus(99); status != HeroTournamentBanned {
		t.Errorf("英雄99应为HeroTournamentBanned，实际=%v", status)
	}
	// 不存在的英雄
	if status := engine.heroStatus(1000); status != HeroUnknown {
		t.Errorf("英雄1000应为HeroUnknown，实际=%v", status)
	}

	// 蓝方ban英雄1
	engine.BlueBanned = append(engine.BlueBanned, 1)
	if status := engine.heroStatus(1); status != HeroBannedByBlue {
		t.Errorf("英雄1应为HeroBannedByBlue，实际=%v", status)
	}

	// 红方pick英雄2
	engine.RedPicked = append(engine.RedPicked, 2)
	if status := engine.heroStatus(2); status != HeroPickedByRed {
		t.Errorf("英雄2应为HeroPickedByRed，实际=%v", status)
	}
}

// ============================================================================
// heroCan 权限视角测试（按队伍区分，支持全局BP记忆）
// ============================================================================

func TestHeroCanPerms(t *testing.T) {
	perms := make(map[int]*HeroPerm)
	for i := 1; i <= 10; i++ {
		perms[i] = FullPerm()
	}
	// 上局红方选了 x=5 → 下局 x 蓝方可选不可ban、红方可ban不可选
	perms[5] = &HeroPerm{BlueCanBan: false, BlueCanPick: true, RedCanBan: true, RedCanPick: false}
	// 上局蓝方选了 y=6 → 下局 y 红方可选不可ban、蓝方可ban不可选
	perms[6] = &HeroPerm{BlueCanBan: true, BlueCanPick: false, RedCanBan: false, RedCanPick: true}

	engine := NewBPEngineWithPerms(1, 2, perms, nil, &mockValidator{})

	// x=5：蓝方可选不可ban
	if ok, _ := engine.heroCan(Blue, Ban, 5); ok {
		t.Error("蓝方不应能ban x=5（上局红方所选）")
	}
	if ok, _ := engine.heroCan(Blue, Pick, 5); !ok {
		t.Error("蓝方应能pick x=5")
	}
	// x=5：红方可ban不可选
	if ok, _ := engine.heroCan(Red, Ban, 5); !ok {
		t.Error("红方应能ban x=5")
	}
	if ok, _ := engine.heroCan(Red, Pick, 5); ok {
		t.Error("红方不应能pick x=5（上局自己选的不能再用）")
	}

	// y=6：红方可选不可ban
	if ok, _ := engine.heroCan(Red, Pick, 6); !ok {
		t.Error("红方应能pick y=6（上局蓝方所选）")
	}
	if ok, _ := engine.heroCan(Red, Ban, 6); ok {
		t.Error("红方不应能ban y=6")
	}
	// y=6：蓝方可ban不可选
	if ok, _ := engine.heroCan(Blue, Ban, 6); !ok {
		t.Error("蓝方应能ban y=6")
	}
	if ok, _ := engine.heroCan(Blue, Pick, 6); ok {
		t.Error("蓝方不应能pick y=6（上局蓝方自己选的）")
	}

	// 普通英雄1：双方均可
	if ok, _ := engine.heroCan(Blue, Ban, 1); !ok {
		t.Error("普通英雄1蓝方应能ban")
	}
	if ok, _ := engine.heroCan(Red, Pick, 1); !ok {
		t.Error("普通英雄1红方应能pick")
	}
}

// ============================================================================
// 全局BP记忆端到端测试：用不对称权限构造引擎并执行
// ============================================================================

func TestGlobalBPMemory_AsymmetricBan(t *testing.T) {
	perms := make(map[int]*HeroPerm)
	for i := 1; i <= 20; i++ {
		perms[i] = FullPerm()
	}
	// 上局红方选了 x=5 → 蓝方不可ban x
	perms[5] = &HeroPerm{BlueCanBan: false, BlueCanPick: true, RedCanBan: true, RedCanPick: false}

	engine := NewBPEngineWithPerms(1, 2, perms, nil, &mockValidator{})
	engine.Start()

	// 第1步蓝方ban，尝试ban x=5 → 应被全局BP规则拒绝
	err := engine.ExecuteAction(Ban, 5)
	if err == nil {
		t.Fatal("蓝方ban上局红方所选英雄应报错")
	}
	if !strings.Contains(err.Error(), "全局BP规则") {
		t.Errorf("错误信息应提到'全局BP规则'，实际: %v", err)
	}
	// 且英雄池里 x 应该仍在（没被ban掉）
	if _, ok := engine.HeroPool[5]; !ok {
		t.Error("ban被拒绝后，英雄5不应从池中移除")
	}

	// 蓝方ban普通英雄1 → 成功
	if err := engine.ExecuteAction(Ban, 1); err != nil {
		t.Fatalf("蓝方ban普通英雄应成功: %v", err)
	}
	if len(engine.BlueBanned) != 1 || engine.BlueBanned[0] != 1 {
		t.Errorf("蓝方ban列表应为[1]，实际=%v", engine.BlueBanned)
	}
}

func TestGlobalBPMemory_PickOpponentHero(t *testing.T) {
	perms := make(map[int]*HeroPerm)
	for i := 1; i <= 20; i++ {
		perms[i] = FullPerm()
	}
	// 上局蓝方选了 y=6 → 红方不可pick y
	perms[6] = &HeroPerm{BlueCanBan: true, BlueCanPick: false, RedCanBan: false, RedCanPick: true}

	engine := NewBPEngineWithPerms(1, 2, perms, nil, &mockValidator{})
	engine.Start()

	// 第1步蓝方ban普通英雄，推进到红方ban（第2步）
	if err := engine.ExecuteAction(Ban, 1); err != nil {
		t.Fatalf("第1步蓝ban失败: %v", err)
	}
	engine.Advance()

	// 第2步红方ban y=6（RedCanBan=false）→ 应被拒绝
	err := engine.ExecuteAction(Ban, 6)
	if err == nil {
		t.Fatal("红方ban上局蓝方所选英雄应报错")
	}
	if !strings.Contains(err.Error(), "全局BP规则") {
		t.Errorf("错误信息应提到'全局BP规则'，实际: %v", err)
	}
}
