package service

import (
	"strings"
	"testing"

	"kpl-bp-simulator/internal/model"
)

// ============================================================================
// mock 依赖：内存版 HeroRepo / GameRepo，用于测试 GameService 编排逻辑
// ============================================================================

type mockHeroRepo struct {
	heroes []*model.Hero
}

func (m *mockHeroRepo) ListActive() ([]*model.Hero, error) {
	return m.heroes, nil
}

func (m *mockHeroRepo) GetByID(id int) (*model.Hero, error) {
	for _, h := range m.heroes {
		if h.ID == id {
			return h, nil
		}
	}
	return nil, nil
}

type mockGameRepo struct {
	games   map[int]*model.Game            // gameID -> game
	actions map[int][]*model.BanPickAction // gameID -> actions（按时序追加）
}

func newMockGameRepo() *mockGameRepo {
	return &mockGameRepo{
		games:   make(map[int]*model.Game),
		actions: make(map[int][]*model.BanPickAction),
	}
}

func (m *mockGameRepo) CreateGame(g *model.Game) (int, error) {
	id := len(m.games) + 1
	g.ID = id
	m.games[id] = g
	return id, nil
}

func (m *mockGameRepo) SaveBPAction(a *model.BanPickAction) error {
	m.actions[a.GameID] = append(m.actions[a.GameID], a)
	return nil
}

func (m *mockGameRepo) UpdateGameBPCompleted(gameID int, winner *string) error {
	g, ok := m.games[gameID]
	if !ok {
		return nil
	}
	g.BPCompleted = true
	g.Winner = winner
	return nil
}

func (m *mockGameRepo) GetGameByMatchAndNum(matchID, gameNumber int) (*model.Game, error) {
	for _, g := range m.games {
		if g.MatchID == matchID && g.GameNumber == gameNumber {
			return g, nil
		}
	}
	return nil, nil
}

func (m *mockGameRepo) ListGamesByMatch(matchID int) ([]*model.Game, error) {
	var out []*model.Game
	for _, g := range m.games {
		if g.MatchID == matchID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (m *mockGameRepo) ListBPActions(gameID int) ([]*model.BanPickAction, error) {
	return m.actions[gameID], nil
}

// ============================================================================
// 工具函数：构造 GameService + 跑完一局 BP
// ============================================================================

// makeHeroModels 构造 N 个英雄
func makeHeroModels(n int) []*model.Hero {
	heroes := make([]*model.Hero, 0, n)
	for i := 1; i <= n; i++ {
		heroes = append(heroes, &model.Hero{ID: i, Name: "英雄"})
	}
	return heroes
}

// runFullGame 用 engine 完整跑完 20 步，返回 engine
// heroIDs 按步序传入（第 i 步用 heroIDs[i]）
func runFullGame(engine *Engine, heroIDs []int) error {
	engine.Start()
	for i := 0; i < 20; i++ {
		info := engine.GetTurnInfo()
		if err := engine.ExecuteAction(info.Action, heroIDs[i]); err != nil {
			return err
		}
		engine.Advance()
	}
	return nil
}

// ============================================================================
// 测试：StartGame 每局都能拿到可用引擎
// ============================================================================

func TestGameService_StartGame_FirstGame(t *testing.T) {
	heroRepo := &mockHeroRepo{heroes: makeHeroModels(50)}
	gameRepo := newMockGameRepo()
	gs := NewGameService(heroRepo, gameRepo, map[int]bool{})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame(第1局)失败: %v", err)
	}
	if engine == nil {
		t.Fatal("Engine 不应为 nil")
	}

	// 第一局：所有英雄全权限（FullPerm）
	for id := 1; id <= 10; id++ {
		perm := engine.Permissions[id]
		if perm == nil {
			t.Fatalf("英雄%d应有权限记录", id)
		}
		if !perm.CanBan(Blue) || !perm.CanPick(Blue) || !perm.CanBan(Red) || !perm.CanPick(Red) {
			t.Errorf("第1局英雄%d应为全权限，实际=%+v", id, perm)
		}
	}
}

func TestGameService_StartGame_TournamentBanned(t *testing.T) {
	heroRepo := &mockHeroRepo{heroes: makeHeroModels(50)}
	gameRepo := newMockGameRepo()
	// 雅典娜=99 赛事禁用（虽然不在英雄表里，但规则上要禁）
	gs := NewGameService(heroRepo, gameRepo, map[int]bool{7: true})

	engine, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("StartGame失败: %v", err)
	}

	// 英雄7 应被全禁
	perm := engine.Permissions[7]
	if perm == nil {
		t.Fatalf("英雄7应有权限记录（NonePerm）")
	}
	if perm.CanBan(Blue) || perm.CanPick(Blue) || perm.CanBan(Red) || perm.CanPick(Red) {
		t.Errorf("赛事禁用英雄7应全权限=false，实际=%+v", perm)
	}
}

// ============================================================================
// 测试：全局BP记忆（跨局权限派生）
// ============================================================================

func TestGameService_GlobalBPMemory(t *testing.T) {
	heroRepo := &mockHeroRepo{heroes: makeHeroModels(50)}
	gameRepo := newMockGameRepo()
	gs := NewGameService(heroRepo, gameRepo, map[int]bool{})

	// ---- 第1局 ----
	g1, err := gs.StartGame(1, 1, 101, 102)
	if err != nil {
		t.Fatalf("第1局 StartGame 失败: %v", err)
	}
	// 用 20 个英雄跑完，并让红方在第6步 pick 英雄6（红方第一次pick=Turn6）
	heroIDs := []int{11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30}
	if err := runFullGame(g1, heroIDs); err != nil {
		t.Fatalf("第1局BP失败: %v", err)
	}
	// 落库 + 标记完成
	gameRepo.CreateGame(&model.Game{MatchID: 1, GameNumber: 1, BlueTeamID: 101, RedTeamID: 102})
	if err := gs.FinishGame(1, g1, strPtr("red")); err != nil {
		t.Fatalf("第1局收尾失败: %v", err)
	}
	// 确认第1局记录：红方选过英雄16
	acts, _ := gameRepo.ListBPActions(1)
	redPicked := map[int]bool{}
	for _, a := range acts {
		if a.ActionType == "pick" && a.Side == "red" {
			redPicked[a.HeroID] = true
		}
	}
	if !redPicked[16] {
		t.Fatalf("第1局红方应pick过16（Turn6是红方首次pick，英雄16），实际=%v", redPicked)
	}

	// ---- 第2局（全局BP记忆生效）----
	engine2, err := gs.StartGame(1, 2, 101, 102)
	if err != nil {
		t.Fatalf("第2局 StartGame 失败: %v", err)
	}

	// 规则：上局红方选了16 → 本局 蓝方可选不可ban、红方可ban不可选
	perm16 := engine2.Permissions[16]
	if perm16 == nil {
		t.Fatalf("第2局英雄16应有权限记录")
	}
	if perm16.BlueCanBan {
		t.Error("上局红方所选的16，蓝方本局不可ban")
	}
	if !perm16.BlueCanPick {
		t.Error("上局红方所选的16，蓝方本局可pick")
	}
	if !perm16.RedCanBan {
		t.Error("上局红方所选的16，红方本局可ban")
	}
	if perm16.RedCanPick {
		t.Error("上局红方所选的16，红方本局不可pick（不能连续两局自己选）")
	}

	// 确认不存在的英雄仍全权限（对照）
	perm1 := engine2.Permissions[1]
	if !perm1.CanBan(Blue) || !perm1.CanPick(Red) {
		t.Error("未被pick过的普通英雄，第2局应恢复正常全权限")
	}
}

// ============================================================================
// 测试：FinishGame 落库
// ============================================================================

func TestGameService_FinishGame_PersistsActions(t *testing.T) {
	heroRepo := &mockHeroRepo{heroes: makeHeroModels(50)}
	gameRepo := newMockGameRepo()
	gs := NewGameService(heroRepo, gameRepo, map[int]bool{})

	engine, _ := gs.StartGame(1, 1, 101, 102)
	heroIDs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	if err := runFullGame(engine, heroIDs); err != nil {
		t.Fatalf("BP失败: %v", err)
	}

	gameRepo.CreateGame(&model.Game{MatchID: 1, GameNumber: 1, BlueTeamID: 101, RedTeamID: 102})
	err := gs.FinishGame(1, engine, strPtr("blue"))
	if err != nil {
		t.Fatalf("FinishGame失败: %v", err)
	}

	acts, _ := gameRepo.ListBPActions(1)
	if len(acts) != 20 {
		t.Fatalf("应有20条BP记录，实际=%d", len(acts))
	}

	// 第1步应为 蓝ban、英雄1
	if acts[0].StepOrder != 1 || acts[0].ActionType != "ban" || acts[0].Side != "blue" || acts[0].HeroID != 1 {
		t.Errorf("第1步记录不正确: %+v", acts[0])
	}
	// 第6步应为 红pick、英雄6
	if acts[5].StepOrder != 6 || acts[5].ActionType != "pick" || acts[5].Side != "red" || acts[5].HeroID != 6 {
		t.Errorf("第6步记录不正确: %+v", acts[5])
	}

	// game 状态已标记完成
	g, _ := gameRepo.GetGameByMatchAndNum(1, 1)
	if g == nil || !g.BPCompleted {
		t.Error("game 应标记 bp_completed=true")
	}
	if g.Winner == nil || *g.Winner != "blue" {
		t.Errorf("winner 应为 blue，实际=%v", g.Winner)
	}
}

// ============================================================================
// 测试：FinishGame 前置校验
// ============================================================================

func TestGameService_FinishGame_NotFinished(t *testing.T) {
	heroRepo := &mockHeroRepo{heroes: makeHeroModels(50)}
	gameRepo := newMockGameRepo()
	gs := NewGameService(heroRepo, gameRepo, map[int]bool{})

	engine, _ := gs.StartGame(1, 1, 101, 102)
	engine.Start() // 只启动，不跑完

	err := gs.FinishGame(1, engine, nil)
	if err == nil {
		t.Fatal("BP未完成时 FinishGame 应报错")
	}
	if !strings.Contains(err.Error(), "尚未完成") {
		t.Errorf("错误信息应说明'尚未完成'，实际: %v", err)
	}
}

func TestGameService_FinishGame_InvalidGameID(t *testing.T) {
	heroRepo := &mockHeroRepo{heroes: makeHeroModels(50)}
	gameRepo := newMockGameRepo()
	gs := NewGameService(heroRepo, gameRepo, map[int]bool{})

	engine, _ := gs.StartGame(1, 1, 101, 102)
	runFullGame(engine, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})

	err := gs.FinishGame(0, engine, nil)
	if err == nil {
		t.Fatal("无效 gameID 时应报错")
	}
}

// strPtr 返回 *string
func strPtr(s string) *string {
	return &s
}
