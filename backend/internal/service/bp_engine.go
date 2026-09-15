package service

import (
	"errors"
	"fmt"
)

// ============================================================================
// 类型定义 — 枚举 + 常量的 Go 惯用法
// ============================================================================

// BPRound 表示 BP 的第几个循环
type BPRound int

const (
	Round1FirstCycle  BPRound = iota + 1 // 第一循环（Blue先Ban）
	Round2SecondCycle                    // 第二循环（Red先Ban）
)

func (r BPRound) String() string {
	switch r {
	case Round1FirstCycle:
		return "第一循环"
	case Round2SecondCycle:
		return "第二循环"
	default:
		return fmt.Sprintf("未知轮次(%d)", r)
	}
}

// TeamSide 队伍颜色方
type TeamSide int

const (
	Blue TeamSide = iota
	Red
)

func (s TeamSide) String() string {
	switch s {
	case Blue:
		return "蓝队"
	case Red:
		return "红队"
	default:
		return fmt.Sprintf("未知方(%d)", s)
	}
}

// ActionType 操作类型
type ActionType int

const (
	Ban ActionType = iota
	Pick
)

func (a ActionType) String() string {
	switch a {
	case Ban:
		return "ban"
	case Pick:
		return "pick"
	default:
		return fmt.Sprintf("未知动作(%d)", a)
	}
}

// ============================================================================
// BP 状态机 — 枚举 + switch 驱动
// ============================================================================

// BPTurn 表示 BP 流程中的每一个步骤（共20步）
// 使用 iota 自增，每一步对应一个唯一的 turn ID
type BPTurn int

const (
	// --- 第一循环：Blue先Ban，共10步（4ban+6pick）---
	// Ban阶段：蓝ban, 红ban, 蓝ban, 红ban
	Turn1_BlueBan1 BPTurn = iota + 1 // 蓝ban
	Turn2_RedBan1                    // 红ban
	Turn3_BlueBan2                   // 蓝ban
	Turn4_RedBan2                    // 红ban
	// Pick阶段：蓝pick, 红pick, 红pick, 蓝pick, 蓝pick, 红pick
	Turn5_BluePick1 // 蓝pick
	Turn6_RedPick1  // 红pick
	Turn7_RedPick2  // 红pick
	Turn8_BluePick2 // 蓝pick
	Turn9_BluePick3 // 蓝pick
	Turn10_RedPick3 // 红pick

	// --- 第二循环：Red先Ban，共10步（6ban+4pick）---
	// Ban阶段：红ban, 蓝ban, 红ban, 蓝ban, 红ban, 蓝ban
	Turn11_RedBan3  // 红ban（继承 iota+1 = 11，不能写 = iota+11！）
	Turn12_BlueBan4 // 蓝ban
	Turn13_RedBan4  // 红ban
	Turn14_BlueBan5 // 蓝ban
	Turn15_RedBan5  // 红ban
	Turn16_BlueBan6 // 蓝ban
	// Pick阶段：红pick, 蓝pick, 蓝pick, 红pick
	Turn17_RedPick4  // 红pick
	Turn18_BluePick4 // 蓝pick
	Turn19_BluePick5 // 蓝pick
	Turn20_RedPick5  // 红pick

	// 边界值
	TurnInvalid BPTurn = 0
	TurnMax     BPTurn = 20
)

// TurnInfo 描述每一步的元信息
type TurnInfo struct {
	Turn   BPTurn
	Round  BPRound
	Side   TeamSide
	Action ActionType
	Index  int // 该侧第几次此动作（如蓝ban1的Index=1）
}

// ============================================================================
// Turn 元数据查询 — 集中管理，避免散落在多处
// ============================================================================

// getTurnInfo 根据 turn ID 返回该步的完整信息
// 这个函数是纯数据映射，不含业务逻辑，极易测试
func getTurnInfo(turn BPTurn) TurnInfo {
	switch turn {
	// --- 第一循环（Blue先Ban，4ban+6pick）---
	case Turn1_BlueBan1:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Blue, Action: Ban, Index: 1}
	case Turn2_RedBan1:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Red, Action: Ban, Index: 1}
	case Turn3_BlueBan2:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Blue, Action: Ban, Index: 2}
	case Turn4_RedBan2:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Red, Action: Ban, Index: 2}
	case Turn5_BluePick1:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Blue, Action: Pick, Index: 1}
	case Turn6_RedPick1:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Red, Action: Pick, Index: 1}
	case Turn7_RedPick2:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Red, Action: Pick, Index: 2}
	case Turn8_BluePick2:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Blue, Action: Pick, Index: 2}
	case Turn9_BluePick3:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Blue, Action: Pick, Index: 3}
	case Turn10_RedPick3:
		return TurnInfo{Turn: turn, Round: Round1FirstCycle, Side: Red, Action: Pick, Index: 3}

	// --- 第二循环（Red先Ban，6ban+4pick）---
	case Turn11_RedBan3:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Red, Action: Ban, Index: 3}
	case Turn12_BlueBan4:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Blue, Action: Ban, Index: 4}
	case Turn13_RedBan4:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Red, Action: Ban, Index: 4}
	case Turn14_BlueBan5:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Blue, Action: Ban, Index: 5}
	case Turn15_RedBan5:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Red, Action: Ban, Index: 5}
	case Turn16_BlueBan6:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Blue, Action: Ban, Index: 6}
	case Turn17_RedPick4:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Red, Action: Pick, Index: 4}
	case Turn18_BluePick4:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Blue, Action: Pick, Index: 4}
	case Turn19_BluePick5:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Blue, Action: Pick, Index: 5}
	case Turn20_RedPick5:
		return TurnInfo{Turn: turn, Round: Round2SecondCycle, Side: Red, Action: Pick, Index: 5}

	default:
		return TurnInfo{Turn: turn, Round: 0, Side: -1, Action: -1, Index: 0}
	}
}

// ============================================================================
// EngineState 引擎整体状态
// ============================================================================

type EngineState int

const (
	StateIdle     EngineState = iota // 空闲/未开始
	StateRunning                     // 进行中
	StateFinished                    // 已完成
	StateAborted                     // 已中止
)

// HeroStatus 英雄当前可用状态
// 注意：普通对局中 HeroPool 是双方共享的，任何被 ban 或被 pick 的英雄都会从中移除，
// 因此一队 pick 过的英雄，另一队自然无法再 ban（"对面已使用"由共享池天然保证）。
// 全局BP记忆模式下，英雄对蓝/红两队的 ban/pick 权限由 HeroPerm 按队伍区分。
type HeroStatus int

const (
	HeroAvailable        HeroStatus = iota // 可用，可被 ban/pick
	HeroTournamentBanned                   // 赛事禁用（如雅典娜），不可ban也不可pick
	HeroBannedByBlue                       // 已被蓝方禁用
	HeroBannedByRed                        // 已被红方禁用
	HeroPickedByBlue                       // 已被蓝方选择
	HeroPickedByRed                        // 已被红方选择
	HeroRestrictedByRule                   // 被全局BP记忆规则限制（某方仅部分权限）
	HeroUnknown                            // 英雄ID不存在（不属于任何列表）
)

// HeroPerm 单个英雄对蓝/红两队的操作权限（4个独立布尔）
// 支持全局BP记忆模式下的不对称权限，例如：
//
//	上局红方选了 x → 下局 x：蓝方可选不可ban，红方可ban不可选
//	  BlueCanBan=false, BlueCanPick=true, RedCanBan=true, RedCanPick=false
type HeroPerm struct {
	BlueCanBan  bool // 蓝方能否 ban 此英雄
	BlueCanPick bool // 蓝方能否 pick 此英雄
	RedCanBan   bool // 红方能否 ban 此英雄
	RedCanPick  bool // 红方能否 pick 此英雄
}

// FullPerm 返回全权限（双方均可 ban/pick）
func FullPerm() *HeroPerm {
	return &HeroPerm{BlueCanBan: true, BlueCanPick: true, RedCanBan: true, RedCanPick: true}
}

// NonePerm 返回无权限（双方均不可操作，用于赛事禁用）
func NonePerm() *HeroPerm {
	return &HeroPerm{}
}

// CanBan 判断某方能否 ban 此英雄
func (p *HeroPerm) CanBan(side TeamSide) bool {
	switch side {
	case Blue:
		return p.BlueCanBan
	case Red:
		return p.RedCanBan
	default:
		return false
	}
}

// CanPick 判断某方能否 pick 此英雄
func (p *HeroPerm) CanPick(side TeamSide) bool {
	switch side {
	case Blue:
		return p.BlueCanPick
	case Red:
		return p.RedCanPick
	default:
		return false
	}
}

// Validator 校验器接口 — 将校验逻辑抽离，便于测试和替换
type Validator interface {
	ValidateBan(heroID int, side TeamSide) error
	ValidatePick(heroID int, side TeamSide) error
}

// NoopValidator 默认放行所有操作的校验器（兜底，nil validator 时使用）
type NoopValidator struct{}

func (v *NoopValidator) ValidateBan(_ int, _ TeamSide) error  { return nil }
func (v *NoopValidator) ValidatePick(_ int, _ TeamSide) error { return nil }

// ============================================================================
// Engine — BP 状态机核心结构体
// ============================================================================

type Engine struct {
	CurrentTurn      BPTurn            // 当前轮到哪一步
	Round            BPRound           // 当前在第几循环
	State            EngineState       // 引擎整体状态
	BlueTeamID       int               // 蓝队ID
	RedTeamID        int               // 红队ID
	BlueBanned       []int             // 蓝队已ban的英雄ID列表
	RedBanned        []int             // 红队已ban的英雄ID列表
	BluePicked       []int             // 蓝队已pick的英雄ID列表
	RedPicked        []int             // 红队已pick的英雄ID列表
	HeroPool         map[int]bool      // 可用英雄池（兼容保留，true=仍在本局可操作）
	TournamentBanned map[int]bool      // 赛事禁用英雄池（如雅典娜），不可ban也不可pick
	Permissions      map[int]*HeroPerm // 按队伍区分的操作权限（支持全局BP记忆）
	Actions          []ActionRecord    // 操作日志（按步序记录，供落库）
	validator        Validator         // 校验器接口
}

// ActionRecord 单步操作的顺序记录（供 game_service 落库还原）
type ActionRecord struct {
	StepOrder int        // 1~20
	Action    ActionType // ban / pick
	Side      TeamSide   // blue / red
	HeroID    int        // 英雄ID
}

// ============================================================================
// 构造函数
// ============================================================================

// NewBPEngine 创建一个新的 BP 状态机（标准模式：双方共享英雄池）
// heroPool: 初始可用英雄池（不含赛事禁用英雄）。注意：Engine 会直接持有该 map，
// ban/pick 会从中 delete，外部传入者不应再直接复用。
// tournamentBanned: 赛事禁用英雄集合（如雅典娜），这些英雄不可ban也不可pick
// 内部会自动把英雄池转为"双方均可ban/pick"的 FullPerm
func NewBPEngine(blueTeamID, redTeamID int, heroPool map[int]bool, tournamentBanned map[int]bool, validator Validator) *Engine {
	perms := make(map[int]*HeroPerm, len(heroPool))
	for id := range heroPool {
		perms[id] = FullPerm()
	}
	return newEngine(blueTeamID, redTeamID, heroPool, perms, tournamentBanned, validator)
}

// NewBPEngineWithPerms 创建一个 BP 状态机（支持全局BP记忆模式）
// permissions: 每个英雄对蓝/红两队的独立操作权限
//   - FullPerm()：双方均可 ban/pick（常规英雄）
//   - 不对称权限：例如上局红方选了 x → 下局 x 为
//     {BlueCanBan:false, BlueCanPick:true, RedCanBan:true, RedCanPick:false}
//   - NonePerm()：赛事禁用（如雅典娜）
//
// tournamentBanned: 赛事禁用英雄集合（冗余保留，用于错误信息区分）
func NewBPEngineWithPerms(blueTeamID, redTeamID int, permissions map[int]*HeroPerm, tournamentBanned map[int]bool, validator Validator) *Engine {
	pool := make(map[int]bool, len(permissions))
	for id, perm := range permissions {
		if perm != nil {
			pool[id] = true
		}
	}
	return newEngine(blueTeamID, redTeamID, pool, permissions, tournamentBanned, validator)
}

func newEngine(blueTeamID, redTeamID int, heroPool map[int]bool, perms map[int]*HeroPerm, tournamentBanned map[int]bool, validator Validator) *Engine {
	if validator == nil {
		validator = &NoopValidator{} // nil 兜底：默认放行所有（状态机内部已有校验收口）
	}
	return &Engine{
		CurrentTurn:      Turn1_BlueBan1,
		Round:            Round1FirstCycle,
		State:            StateIdle,
		BlueTeamID:       blueTeamID,
		RedTeamID:        redTeamID,
		HeroPool:         heroPool,
		TournamentBanned: tournamentBanned,
		Permissions:      perms,
		validator:        validator,
		BlueBanned:       make([]int, 0),
		RedBanned:        make([]int, 0),
		BluePicked:       make([]int, 0),
		RedPicked:        make([]int, 0),
		Actions:          make([]ActionRecord, 0, 20),
	}
}

// ============================================================================
// 核心方法 — switch 驱动状态流转
// ============================================================================

// Start 启动 BP 流程
func (e *Engine) Start() error {
	if e.State != StateIdle {
		return errors.New("BP流程已在运行或已结束")
	}
	e.State = StateRunning
	return nil
}

// NextTurn 返回下一步应该执行的 Turn
func (e *Engine) NextTurn() BPTurn {
	return e.CurrentTurn
}

// GetTurnInfo 获取当前 step 的详细信息
func (e *Engine) GetTurnInfo() TurnInfo {
	return getTurnInfo(e.CurrentTurn)
}

// ExecuteAction 执行 ban 或 pick 操作
// 这是整个状态机的入口方法，handler 层调用此方法
func (e *Engine) ExecuteAction(actionType ActionType, heroID int) error {
	if e.State != StateRunning {
		return fmt.Errorf("BP流程未运行，当前状态=%v", e.State)
	}
	if e.CurrentTurn == TurnInvalid || e.CurrentTurn > TurnMax {
		return errors.New("无效的操作序号")
	}

	info := getTurnInfo(e.CurrentTurn)

	// 1. 校验：动作类型（ban还是pick）必须匹配
	if actionType != info.Action {
		return fmt.Errorf("期望的动作是 %v，实际传入 %v", info.Action, actionType)
	}

	// 2. 校验：通过 validator 接口验证操作合法性
	var err error
	switch actionType {
	case Ban:
		err = e.validator.ValidateBan(heroID, info.Side)
	case Pick:
		err = e.validator.ValidatePick(heroID, info.Side)
	}
	if err != nil {
		return err
	}

	// 3. 执行操作 — 委托给独立的 handler 函数
	switch actionType {
	case Ban:
		return e.handleBan(info.Side, heroID)
	case Pick:
		return e.handlePick(info.Side, heroID)
	}

	return nil
}

// Advance 推进到下一步（ExecuteAction 成功后调用）
func (e *Engine) Advance() {
	e.CurrentTurn++
	if e.CurrentTurn > TurnMax {
		e.State = StateFinished
		return
	}
	// 根据 turn 判断并更新当前到达第几轮循环
	info := getTurnInfo(e.CurrentTurn)
	e.Round = info.Round
}

// IsFinished 判断是否完成
func (e *Engine) IsFinished() bool {
	return e.State == StateFinished
}

// ============================================================================
// 内部 handler 函数 — 将业务逻辑与 switch 解耦
// ============================================================================

// heroStatus 返回英雄当前状态（全局视角），用于区别"不可ban/pick"的具体原因
// 判定顺序：赛事禁用 → 已被禁用 → 已被选择 → 未知
func (e *Engine) heroStatus(heroID int) HeroStatus {
	if e.TournamentBanned[heroID] {
		return HeroTournamentBanned
	}
	if contains(e.BlueBanned, heroID) {
		return HeroBannedByBlue
	}
	if contains(e.RedBanned, heroID) {
		return HeroBannedByRed
	}
	if contains(e.BluePicked, heroID) {
		return HeroPickedByBlue
	}
	if contains(e.RedPicked, heroID) {
		return HeroPickedByRed
	}
	if _, ok := e.HeroPool[heroID]; ok {
		return HeroAvailable
	}
	return HeroUnknown
}

// heroCan 判断"某方对某英雄执行某操作"是否被允许（按队伍区分权限）
// 返回 (是否允许, 不允许时的具体原因)
// 判定顺序：全局状态（赛事禁用/已被ban/已被pick）→ 权限表 → 英雄池
func (e *Engine) heroCan(side TeamSide, action ActionType, heroID int) (bool, HeroStatus) {
	// 1. 全局状态：赛事禁用 / 已被禁用 / 已被选择
	if status := e.heroStatus(heroID); status != HeroAvailable {
		return false, status
	}
	// 2. 权限表：按队伍区分的 ban/pick 权限（全局BP记忆）
	if perm, ok := e.Permissions[heroID]; ok {
		var allowed bool
		switch action {
		case Ban:
			allowed = perm.CanBan(side)
		case Pick:
			allowed = perm.CanPick(side)
		}
		if !allowed {
			return false, HeroRestrictedByRule
		}
	}
	// 3. 英雄池兜底
	if e.HeroPool[heroID] {
		return true, HeroAvailable
	}
	return false, HeroUnknown
}

// contains 判断切片中是否包含某元素
func contains(s []int, v int) bool {
	for _, i := range s {
		if i == v {
			return true
		}
	}
	return false
}

// heroUnavailableError 生成"不可ban/pick"的具体原因错误
func heroUnavailableError(heroID int, status HeroStatus, action ActionType) error {
	verb := map[ActionType]string{Ban: "ban", Pick: "pick"}[action]
	switch status {
	case HeroTournamentBanned:
		return fmt.Errorf("英雄%d为赛事禁用英雄，不可%s", heroID, verb)
	case HeroBannedByBlue, HeroBannedByRed:
		return fmt.Errorf("英雄%d已被禁用，不可%s", heroID, verb)
	case HeroPickedByBlue:
		return fmt.Errorf("英雄%d已被蓝方选择，不可%s", heroID, verb)
	case HeroPickedByRed:
		return fmt.Errorf("英雄%d已被红方选择，不可%s", heroID, verb)
	case HeroRestrictedByRule:
		return fmt.Errorf("英雄%d受全局BP规则限制，此方不可%s", heroID, verb)
	default:
		return fmt.Errorf("英雄%d不存在，不可%s", heroID, verb)
	}
}

// handleBan 处理 ban 操作
func (e *Engine) handleBan(side TeamSide, heroID int) error {
	// 按队伍区分：检查"该方"能否 ban 此英雄（覆盖全局BP记忆的不对称权限）
	ok, status := e.heroCan(side, Ban, heroID)
	if !ok {
		return heroUnavailableError(heroID, status, Ban)
	}
	delete(e.HeroPool, heroID)
	e.Actions = append(e.Actions, ActionRecord{
		StepOrder: int(e.CurrentTurn),
		Action:    Ban,
		Side:      side,
		HeroID:    heroID,
	})

	switch side {
	case Blue:
		e.BlueBanned = append(e.BlueBanned, heroID)
	case Red:
		e.RedBanned = append(e.RedBanned, heroID)
	}
	return nil
}

// handlePick 处理 pick 操作
func (e *Engine) handlePick(side TeamSide, heroID int) error {
	// 按队伍区分：检查"该方"能否 pick 此英雄（覆盖全局BP记忆的不对称权限）
	ok, status := e.heroCan(side, Pick, heroID)
	if !ok {
		return heroUnavailableError(heroID, status, Pick)
	}
	delete(e.HeroPool, heroID)
	e.Actions = append(e.Actions, ActionRecord{
		StepOrder: int(e.CurrentTurn),
		Action:    Pick,
		Side:      side,
		HeroID:    heroID,
	})

	switch side {
	case Blue:
		e.BluePicked = append(e.BluePicked, heroID)
	case Red:
		e.RedPicked = append(e.RedPicked, heroID)
	}
	return nil
}

// ============================================================================
// Getter 方法 — 供 handler/hub 层读取状态
// ============================================================================

func (e *Engine) GetCurrentSide() TeamSide {
	info := getTurnInfo(e.CurrentTurn)
	return info.Side
}

func (e *Engine) GetCurrentAction() ActionType {
	info := getTurnInfo(e.CurrentTurn)
	return info.Action
}

func (e *Engine) GetRound() BPRound {
	return e.Round
}

func (e *Engine) GetAvailableHeroes() map[int]bool {
	// 返回副本，防止外部修改
	heroes := make(map[int]bool, len(e.HeroPool))
	for k, v := range e.HeroPool {
		heroes[k] = v
	}
	return heroes
}
