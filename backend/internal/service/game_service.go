package service

import (
	"fmt"

	"kpl-bp-simulator/internal/model"
	"kpl-bp-simulator/internal/repository"
)

// ============================================================================
// GameService 对局编排器（Game Orchestrator）
// ============================================================================

// GameService 负责一场比赛（match）的多局（game）串行编排：
//  1. 每局开始时：加载英雄池 → 派生该局英雄权限 → 创建新 BPEngine
//  2. 跨局状态派生（全局BP记忆规则）：读取上一局 BP 结果，
//     按规则计算本局每个英雄对蓝/红两队的 ban/pick 权限
//  3. 协调落库：一局 BP 完成后把 Actions 持久化为 ban_pick_action 记录
//
// 设计原则：Engine 实例的使命 = 一局 BP，结束后即丢弃（或存档）；
// 跨局"恢复原状"的责任在本编排器，不在状态机内部。
type GameService struct {
	heroRepo         repository.HeroRepo
	gameRepo         repository.GameRepo
	tournamentBanned map[int]bool // 赛事禁用英雄（如雅典娜），全局面不可用
}

// NewGameService 创建对局编排器
// heroRepo: 英雄数据访问（提供英雄池）
// gameRepo: 对局数据访问（创建局、存 BP 记录）
// tournamentBanned: 赛事禁用英雄集合
func NewGameService(heroRepo repository.HeroRepo, gameRepo repository.GameRepo, tournamentBanned map[int]bool) *GameService {
	if tournamentBanned == nil {
		tournamentBanned = make(map[int]bool)
	}
	return &GameService{
		heroRepo:         heroRepo,
		gameRepo:         gameRepo,
		tournamentBanned: tournamentBanned,
	}
}

// StartGame 开始新一局的 BP 流程，返回准备好的 Engine
// matchID: 所属比赛ID
// gameNumber: 第几局（1,2,3...）
// blueTeamID / redTeamID: 本局蓝红队
//
// 职责链：加载全部英雄 → 构造基础权限（赛事禁用=全禁，其余=双方全可）→
// 若非第一局，读取上一局 BP 记录 → 按全局BP记忆规则派生本局权限 → 创建 Engine
func (s *GameService) StartGame(matchID, gameNumber, blueTeamID, redTeamID int) (*Engine, error) {
	heroes, err := s.heroRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("加载英雄池失败: %w", err)
	}

	// 1. 基础权限表：所有英雄默认 FullPerm，赛事禁用置 NonePerm
	perms := make(map[int]*HeroPerm, len(heroes))
	for _, h := range heroes {
		if s.tournamentBanned[h.ID] {
			perms[h.ID] = NonePerm()
		} else {
			perms[h.ID] = FullPerm()
		}
	}

	// 2. 非第一局：应用全局BP记忆规则（依据上一局结果）
	if gameNumber > 1 {
		prevGame, err := s.gameRepo.GetGameByMatchAndNum(matchID, gameNumber-1)
		if err != nil {
			return nil, fmt.Errorf("读取上一局(%d局)失败: %w", gameNumber-1, err)
		}
		if prevGame != nil && prevGame.BPCompleted {
			actions, err := s.gameRepo.ListBPActions(prevGame.ID)
			if err != nil {
				return nil, fmt.Errorf("读取上一局BP记录失败: %w", err)
			}
			// 全局BP记忆"跟队伍走，不跟颜色走"（问题4）：
			// 上一局记录里的 side 是"颜色"，必须先用上一局的红蓝归属把它翻译成"队伍"，
			// 再按本局的红蓝归属翻译回"颜色"。否则换边后限制会加在错误的队伍身上。
			s.applyGlobalBPMemory(perms, actions, prevGame, blueTeamID, redTeamID)
		}
	}

	return NewBPEngineWithPerms(blueTeamID, redTeamID, perms, s.tournamentBanned, nil), nil
}

// applyGlobalBPMemory 应用全局BP记忆规则：
//
//	上局某队选了 x → 本局 x：该队可 ban 但不可选；对手可选但不可 ban
//
// ⚠️ 语义关键（问题4 选边后尤其重要）：限制**跟着队伍走，不跟着颜色走**。
// 上一局记录里的 Side 是"颜色"，本局的红蓝归属可能已经交换，
// 因此必须：
//  1. 用【上一局】的 blue/red 归属，把 a.Side（颜色）翻译成"是哪个队选了这个英雄"；
//  2. 再用【本局】的 blue/red 归属，把"该队"翻译回本局它是蓝还是红；
//  3. 据此设置本局的权限位。
//
// 若只用颜色直接映射（旧实现），换边后会出现两个错误：
// 自己上局选过的英雄本局竟可再选；而对手却被禁掉了。
//
// 其余英雄保持原有权限（赛事禁用、普通英雄不受影响）
func (s *GameService) applyGlobalBPMemory(
	perms map[int]*HeroPerm,
	actions []*model.BanPickAction,
	prevGame *model.Game,
	curBlueTeamID, curRedTeamID int,
) {
	for _, a := range actions {
		if a.ActionType != "pick" {
			continue // 只有 pick 过的英雄产生"记忆"
		}

		// 1) 颜色 → 上一局实际队伍
		var pickedByTeamID int
		switch a.Side {
		case "blue":
			pickedByTeamID = prevGame.BlueTeamID
		case "red":
			pickedByTeamID = prevGame.RedTeamID
		default:
			continue
		}

		// 2) 该队在本局是蓝还是红（选边后可能已交换）
		pickedByIsBlueNow := pickedByTeamID == curBlueTeamID
		if pickedByTeamID != curBlueTeamID && pickedByTeamID != curRedTeamID {
			// 理论上不可达（两队固定），保守起见跳过，避免给出错误权限
			continue
		}

		// 3) 设置权限：选过的一方"可ban不可选"，对手"可选不可ban"
		if pickedByIsBlueNow {
			perms[a.HeroID] = &HeroPerm{
				BlueCanBan:  true,
				BlueCanPick: false, // 自己选过 → 不可再选
				RedCanBan:   false,
				RedCanPick:  true, // 对手可选 → 不可 ban
			}
		} else {
			perms[a.HeroID] = &HeroPerm{
				BlueCanBan:  false,
				BlueCanPick: true,
				RedCanBan:   true,
				RedCanPick:  false,
			}
		}
	}
}

// FinishGame 一局 BP 完成后收尾：
//  1. 把 Engine.Actions 落库为 ban_pick_action 记录（按步序）
//  2. 更新 game 表的 bp_completed = true 及 winner（如有）
func (s *GameService) FinishGame(gameID int, engine *Engine, winner *string) error {
	if gameID <= 0 {
		return fmt.Errorf("无效的 gameID=%d", gameID)
	}
	if !engine.IsFinished() {
		return fmt.Errorf("BP 尚未完成，不能收尾")
	}

	// 1. 落库 BP 操作记录
	for _, rec := range engine.Actions {
		actionType := rec.Action.String()
		side := "blue"
		if rec.Side == Red {
			side = "red"
		}
		err := s.gameRepo.SaveBPAction(&model.BanPickAction{
			GameID:     gameID,
			StepOrder:  rec.StepOrder,
			ActionType: actionType,
			Side:       side,
			HeroID:     rec.HeroID,
		})
		if err != nil {
			return fmt.Errorf("保存第%d步BP记录失败: %w", rec.StepOrder, err)
		}
	}

	// 2. 标记 BP 完成
	return s.gameRepo.UpdateGameBPCompleted(gameID, winner)
}

// GlobalBPMemoryPerm 导出规则说明（供文档/调试）。
//
// 返回"某队在上局 pick 了 x"时，本局 x 针对**该队当前所处颜色**的权限。
//
// ⚠️ 注意入参语义（问题4）：pickedByTeamIsBlueNow 表示"上局选过 x 的那支队，
// 在本局是蓝方吗"。限制跟队伍走，不跟颜色走 —— 换边后同一支队只是换了颜色，
// 它依然不可再选自己上局选过的英雄。
func GlobalBPMemoryPerm(pickedByTeamIsBlueNow bool) *HeroPerm {
	if pickedByTeamIsBlueNow {
		// 选过 x 的队本局是蓝方：蓝方可ban不可选；红方（对手）可选不可ban
		return &HeroPerm{BlueCanBan: true, BlueCanPick: false, RedCanBan: false, RedCanPick: true}
	}
	// 选过 x 的队本局是红方：红方可ban不可选；蓝方（对手）可选不可ban
	return &HeroPerm{BlueCanBan: false, BlueCanPick: true, RedCanBan: true, RedCanPick: false}
}
