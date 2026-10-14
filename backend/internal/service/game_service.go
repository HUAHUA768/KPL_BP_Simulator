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
// 若非第一局，读取上一局 BP 记录 → 按全局BP记忆规则派生本局权限 →
// 按元流之子"五选一"规则**跨局累计**派生本局权限 → 创建 Engine
func (s *GameService) StartGame(matchID, gameNumber, blueTeamID, redTeamID int) (*Engine, error) {
	heroes, err := s.heroRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("加载英雄池失败: %w", err)
	}

	// 0. 元流之子组（"五选一"锁定范围）——按名字前缀识别，不硬编码 ID
	yuanliuIDs := YuanliuGroupIDs(heroes)

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

	// 3. 元流之子"五选一"锁定（跨局累计）。
	// ⚠️ 与第 2 步的关键差异：全局BP记忆只看"上一局"，而元流规则必须看**整场比赛**，
	//    因此这里独立遍历全部历史局（详见 collectYuanliuUsage）。
	// ⚠️ 顺序要求：必须在 applyGlobalBPMemory **之后**执行 —— 本规则只"压 pick 位"，
	//    不重置 ban 位，叠加在既有权限之上（见 applyYuanliuLock 注释）。
	//
	// ⚠️ 注意这里**不加 gameNumber > 1 的限制**：
	//    第 1 局虽然历史为空（不产生跨局锁定），但互斥组仍需注册到 Engine 上，
	//    由引擎在**本局内**动态生效（"一旦使用了元射，则其它四个都不能再选"）。
	if len(yuanliuIDs) > 0 {
		usedBlue, usedRed, err := s.collectYuanliuUsage(matchID, gameNumber, yuanliuIDs, blueTeamID, redTeamID)
		if err != nil {
			return nil, err
		}
		s.applyYuanliuLock(perms, yuanliuIDs, usedBlue, usedRed)
	}

	engine := NewBPEngineWithPerms(blueTeamID, redTeamID, perms, s.tournamentBanned, nil)

	// 4. 注册互斥组：让引擎能在**本局内**于某方 pick 后立即锁定组内其余英雄。
	//    跨局部分已由第 3 步写入初始权限表，本步负责"局内动态收紧"。
	if len(yuanliuIDs) > 0 {
		engine.SetExclusiveGroups([][]int{yuanliuIDs})
	}

	return engine, nil
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

// ============================================================================
// 元流之子「五选一」锁定（规则定义见 yuanliu.go）
// ============================================================================

// collectYuanliuUsage 统计本场比赛中双方各自"已 pick 过的元流英雄"，供五选一锁定使用。
//
// 返回值语义：usedBlue / usedRed 分别对应**本局**的蓝队与红队身份。
//
// ⚠️ 语义关键（与 applyGlobalBPMemory 完全一致）：限制**跟队伍走，不跟颜色走**。
// 每一局记录里的 Side 是"颜色"，而每局的红蓝归属可能已因选边而交换，因此必须：
//  1. 用【该局自己】的 blue/red 归属，把 a.Side（颜色）翻译成"是哪个队 pick 的"；
//  2. 再用【本局】的 blue/red 归属，把"该队"翻译回本局它是蓝还是红。
//
// ⚠️ 与全局BP记忆的两点差异：
//   - 范围：遍历**全部历史局**（含更早的局），而非只看上一局 —— 规则是"本场比赛累计"；
//   - 过滤：只有 pick 计入，**ban 不计入**（需求方确认：ban 掉元射不会锁死其它四个）。
//
// 性能：局数 ≤ 9、每局动作 ≤ 20，总计 ≤ 180 次遍历 + 每局一次 ListBPActions，
// 该量级远小于既有"每步 BP 都落库"的开销，无需缓存。
func (s *GameService) collectYuanliuUsage(
	matchID, currentGameNumber int, yuanliuIDs []int,
	curBlueTeamID, curRedTeamID int,
) (usedBlue, usedRed map[int]bool, err error) {
	usedBlue, usedRed = map[int]bool{}, map[int]bool{}

	inGroup := make(map[int]bool, len(yuanliuIDs))
	for _, id := range yuanliuIDs {
		inGroup[id] = true
	}

	games, err := s.gameRepo.ListGamesByMatch(matchID)
	if err != nil {
		return nil, nil, fmt.Errorf("读取比赛历史局失败: %w", err)
	}

	for _, g := range games {
		if g == nil {
			continue
		}
		// 只统计"本局之前"的局。本局尚未开始、其记录必然为空；
		// 用 >= 跳过也更稳健（防脏数据把本局记录算进来导致重复计数）。
		if g.GameNumber >= currentGameNumber {
			continue
		}

		actions, aerr := s.gameRepo.ListBPActions(g.ID)
		if aerr != nil {
			return nil, nil, fmt.Errorf("读取第%d局BP记录失败: %w", g.GameNumber, aerr)
		}

		for _, a := range actions {
			if a == nil {
				continue
			}
			// 只认 pick 的元流英雄（ban 不计入）
			if a.ActionType != "pick" || !inGroup[a.HeroID] {
				continue
			}

			// 1) 颜色 → 该局实际队伍
			var pickedByTeamID int
			switch a.Side {
			case "blue":
				pickedByTeamID = g.BlueTeamID
			case "red":
				pickedByTeamID = g.RedTeamID
			default:
				continue
			}

			// 2) 该队在本局是蓝还是红（选边后可能已交换）
			switch pickedByTeamID {
			case curBlueTeamID:
				usedBlue[a.HeroID] = true
			case curRedTeamID:
				usedRed[a.HeroID] = true
			}
			// 两队都不是（理论上不可达）：保守跳过，不给出错误锁定
		}
	}

	return usedBlue, usedRed, nil
}

// applyYuanliuLock 把元流之子「五选一」锁定施加到权限表上（就地修改 perms）。
//
// 规则公式（见 yuanliu.go / 方案 §2.2）：
//
//	T 本局能否 pick h ⇔ usedYuanliu(T) 为空 或 h ∈ usedYuanliu(T)
//
// ⚠️ 五条必须严格遵守的语义（需求方逐条确认）：
//  1. **锁仅作用于 pick** —— 用过元流的队伍，其自身 ban 权完整保留
//     （需求 2："A 队依然有 ban 五个元流的权力"）。
//  2. **对手的 ban 权被剥夺** —— 需求 2："若 A 队已使用元射，则 B 队**不用再 ban**
//     其它四个元流"。这是对 B 的**授权缺失**：A 用过之后，这些元流对 B 已无 pick 威胁，
//     因此 B 不需要（也不允许）再 ban 它们。→ 置 `对手CanBan = false`。
//  3. **锁是按队伍的** —— A 队被锁不影响 B 队的五选一权力（usedBlue / usedRed 独立计算）。
//  4. **"用过的那一个"同样要处理对手 ban** —— ⚠️ 这是实战暴露的第二处缺陷（见 §9.7）：
//     初版依赖"全局BP记忆会处理已用过的那一个"，但**记忆只看上一局**，
//     若该队在中途换过边（如第1局用元射、第2局打红），第3局记忆已不再覆盖元射
//     → 该队竟能 ban 自己用过的元射。因此本函数必须**独立处理已用过的那一个**：
//     它自身的 `本队CanPick` 交给记忆规则（或本就已被 pick 移出池），
//     但 `对手CanBan` 必须由本规则置 false。
//  5. **不复活赛事禁用** —— NonePerm（四布尔全 false）的条目直接跳过。
//
// ⚠️ 前提：FullPerm()/NonePerm() 每次调用都返回**新指针**（见 bp_engine.go），
// 因此这里就地修改 p.BlueCanPick 不会串改其他英雄。**该前提不得被破坏**
// （不要把 FullPerm 改成共享单例）。
func (s *GameService) applyYuanliuLock(
	perms map[int]*HeroPerm, groupIDs []int,
	usedBlue, usedRed map[int]bool,
) {
	// 每支队伍：本场用过的元流集合。空集 → 该队仍保有完整五选一权力，不做任何限制。
	type sideUsage struct {
		used    map[int]bool
		ownPick func(p *HeroPerm)
		oppBan  func(p *HeroPerm)
	}
	sides := []sideUsage{
		{
			used:    usedBlue,
			ownPick: func(p *HeroPerm) { p.BlueCanPick = false },
			oppBan:  func(p *HeroPerm) { p.RedCanBan = false },
		},
		{
			used:    usedRed,
			ownPick: func(p *HeroPerm) { p.RedCanPick = false },
			oppBan:  func(p *HeroPerm) { p.BlueCanBan = false },
		},
	}

	// 逐队处理。对该队而言，一旦用过元流，则：
	//   (a) 对手对**整组五个**都失去 ban 权
	//   (b) 该队对**整组五个**都失去 pick 权 —— 含**它用过的那一个**
	//
	// ⚠️ (b) 必须覆盖"用过的那个"，不能交给全局BP记忆：
	// 记忆只看上一局，若该队中途换过边，记忆会"忘记"更早的 pick，
	// 导致它竟能重复 pick 自己用过的元流（实测缺陷，见 §9.7）。
	// 本规则是"本场比赛只能选一个"，必须自给自足、独立保证。
	for _, su := range sides {
		if len(su.used) == 0 {
			continue // 未用过任何元流 → 不产生任何限制
		}
		for _, id := range groupIDs {
			p := perms[id]
			if p == nil {
				continue
			}
			// 赛事禁用（四布尔全 false）：保持原样，不得被本规则"复活"任何位
			if !p.BlueCanBan && !p.BlueCanPick && !p.RedCanBan && !p.RedCanPick {
				continue
			}

			su.oppBan(p)  // (a) 对手不可 ban（整组）
			su.ownPick(p) // (b) 本队不可再 pick（整组，含用过的那个）
		}
	}
}

// FinishGame 一局 BP 完成后收尾：
//  1. 补齐落库 BP 操作记录（与已增量落库的步骤对账，不重复插入）
//  2. 更新 game 表的 bp_completed = true 及 winner（如有）
func (s *GameService) FinishGame(gameID int, engine *Engine, winner *string) error {
	if gameID <= 0 {
		return fmt.Errorf("无效的 gameID=%d", gameID)
	}
	if !engine.IsFinished() {
		return fmt.Errorf("BP 尚未完成，不能收尾")
	}

	// 1. 读取已落库步骤，按 step_order 建索引
	existing, err := s.gameRepo.ListBPActions(gameID)
	if err != nil {
		return fmt.Errorf("读取已落库BP记录失败: %w", err)
	}
	seen := make(map[int]bool, len(existing))
	for _, a := range existing {
		seen[a.StepOrder] = true
	}

	// 2. 补齐缺失步骤（幂等）。
	// 房间层现在会**每步增量落库**（见 RoomHandler.persistAction），
	// 因此这里是对账补齐语义，只写入 DB 中尚不存在的 step_order，
	// 既兼容旧的"结束时一次性写"路径，也不会产生重复步骤。
	for _, rec := range engine.Actions {
		if seen[rec.StepOrder] {
			continue
		}
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

	// 3. 标记 BP 完成
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
