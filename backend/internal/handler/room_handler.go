package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"sync"

	"kpl-bp-simulator/internal/model"
	"kpl-bp-simulator/internal/repository"
	"kpl-bp-simulator/internal/service"
	ws "kpl-bp-simulator/internal/websocket"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Room — 一个房间 = 一场系列赛（BO5 / BO7 / 无限制）的全部内存态
// ============================================================================
//
// 生命周期：CreateRoom 时建立（同时落库 match_record + 第1局 game 行），
// 随"进入下一局"更换 Engine/GameID，随"决出胜方"或"手动结束"终止。
// 并发安全：任何字段读写都必须先持有 mu（见 withRoom / 各 Handler）。
type Room struct {
	mu         sync.Mutex
	MatchID    int
	GameID     int // 当前局的 game 行 ID
	GameNumber int // 当前第几局
	BlueTeamID int
	RedTeamID  int
	BoFormat   int // 5 / 7 / 0
	BlueScore  int
	RedScore   int
	Winner     *string // 当前局已提交的胜方（未提交为 nil）
	MatchOver  bool
	Persisted  bool // 当前局 BP 是否已落库（双保险，防重复写入）
	Engine     *service.Engine

	// 系列赛固定身份：team_a / team_b 不随选边变化（问题4）
	// 比分以队伍身份记账，换边不会把 A 队的胜场写到 B 队名下。
	TeamAID int
	TeamBID int
	// 当前局的比分快照（按队伍身份），用于持久化到 team_a_score / team_b_score
	TeamAScore int
	TeamBScore int

	// 下一局的红蓝归属（由用户在局间选边 UI 指定；未指定时沿用当前局）
	NextBlueTeamID *int
	NextRedTeamID  *int
	// 选边权归属队伍（联机模式 = 败方；单机模式仅落库预留）
	SidePickerTeamID *int
}

// isFixedTeam 判断某队伍是否属于本场比赛的固定两队（team_a / team_b）
func (r *Room) isFixedTeam(teamID int) bool {
	return teamID == r.TeamAID || teamID == r.TeamBID
}

// addWin 按队伍身份累加比分并同步红蓝视图，避免换边导致比分记错队（问题4）
func (r *Room) addWin(teamID int) {
	if teamID == r.TeamAID {
		r.TeamAScore++
	} else if teamID == r.TeamBID {
		r.TeamBScore++
	}
	r.syncColorScores()
}

// syncColorScores 依据"当前局红蓝归属"把队伍比分映射为红蓝比分视图
func (r *Room) syncColorScores() {
	if r.BlueTeamID == r.TeamAID {
		r.BlueScore, r.RedScore = r.TeamAScore, r.TeamBScore
	} else {
		r.BlueScore, r.RedScore = r.TeamBScore, r.TeamAScore
	}
}

// winsNeeded 返回该赛制下夺冠所需胜场；无限制赛制(BoFormat=0)返回 0 表示永不自动结束
func (r *Room) winsNeeded() int {
	if r.BoFormat <= 0 {
		return 0
	}
	return r.BoFormat/2 + 1
}

// RoomHandler 房间相关接口
type RoomHandler struct {
	heroRepo         repository.HeroRepo
	gameRepo         repository.GameRepo
	teamRepo         repository.TeamRepo
	matchRepo        repository.MatchRepo
	gameService      *service.GameService
	hub              *ws.Hub
	rooms            map[string]*Room // roomID -> Room
	roomsMu          sync.Mutex       // 保护 rooms map 本身
	tournamentBanned map[int]bool
	teams            *teamCache // 战队信息进程内缓存（队名/队标展示用）
}

// NewRoomHandler 创建 RoomHandler
func NewRoomHandler(
	heroRepo repository.HeroRepo,
	gameRepo repository.GameRepo,
	teamRepo repository.TeamRepo,
	matchRepo repository.MatchRepo,
	gameService *service.GameService,
	hub *ws.Hub,
	tournamentBanned map[int]bool,
) *RoomHandler {
	h := &RoomHandler{
		heroRepo:         heroRepo,
		gameRepo:         gameRepo,
		teamRepo:         teamRepo,
		matchRepo:        matchRepo,
		gameService:      gameService,
		hub:              hub,
		rooms:            make(map[string]*Room),
		tournamentBanned: tournamentBanned,
		teams:            newTeamCache(teamRepo),
	}

	// 注册 WebSocket 操作处理器
	hub.SetActionHandler(h.handleWSAction)

	return h
}

// getRoom 安全取房间（先锁 map，再交给调用方锁 Room）
func (h *RoomHandler) getRoom(roomID string) (*Room, bool) {
	h.roomsMu.Lock()
	defer h.roomsMu.Unlock()
	room, ok := h.rooms[roomID]
	return room, ok
}

// handleWSAction 处理来自 WebSocket 的 ban/pick 操作
func (h *RoomHandler) handleWSAction(roomID string, side string, msg *ws.IncomingMessage) error {
	room, ok := h.getRoom(roomID)
	if !ok {
		return nil // 房间不存在，静默忽略
	}

	var actionType service.ActionType
	switch msg.Type {
	case "ban":
		actionType = service.Ban
	case "pick":
		actionType = service.Pick
	default:
		return nil
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if err := room.Engine.ExecuteAction(actionType, msg.HeroID); err != nil {
		return err
	}

	room.Engine.Advance()
	h.persistIfFinished(room)

	// 广播更新
	h.broadcastUpdate(roomID, room)
	return nil
}

// CreateRoom POST /api/rooms — 创建房间（= 创建一场系列赛）
func (h *RoomHandler) CreateRoom(c *gin.Context) {
	var req struct {
		BlueTeamID int  `json:"blueTeamId"`
		RedTeamID  int  `json:"redTeamId"`
		BoFormat   *int `json:"boFormat"` // 可选，缺省 5
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	// 赛制枚举校验：仅接受 5 / 7 / 0
	boFormat := 5
	if req.BoFormat != nil {
		boFormat = *req.BoFormat
	}
	if boFormat != 5 && boFormat != 7 && boFormat != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "赛制仅支持 BO5 / BO7 / 无限制"})
		return
	}

	// 校验战队存在
	if _, err := h.teamRepo.GetByID(req.BlueTeamID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "蓝方战队不存在"})
		return
	}
	if _, err := h.teamRepo.GetByID(req.RedTeamID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "红方战队不存在"})
		return
	}

	// 生成房间号
	roomID := generateRoomID()

	// 1. 落库：一场系列赛（team_a = 蓝方，team_b = 红方）
	matchID, err := h.matchRepo.CreateMatch(&model.MatchRecord{
		TeamAID:        req.BlueTeamID,
		TeamBID:        req.RedTeamID,
		CurrentGameNum: 1,
		BoFormat:       boFormat,
		Status:         "ongoing",
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "创建比赛记录失败"})
		return
	}

	// 2. 落库：第一局
	gameID, err := h.gameRepo.CreateGame(&model.Game{
		MatchID:    matchID,
		GameNumber: 1,
		BlueTeamID: req.BlueTeamID,
		RedTeamID:  req.RedTeamID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "创建对局记录失败"})
		return
	}

	// 3. 造引擎（经 GameService，保证与后续各局同一条构造路径）
	engine, err := h.gameService.StartGame(matchID, 1, req.BlueTeamID, req.RedTeamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "加载英雄池失败"})
		return
	}
	if err := engine.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "启动 BP 流程失败"})
		return
	}

	room := &Room{
		MatchID:    matchID,
		GameID:     gameID,
		GameNumber: 1,
		BlueTeamID: req.BlueTeamID,
		RedTeamID:  req.RedTeamID,
		BoFormat:   boFormat,
		Engine:     engine,
		// 固定身份：team_a = 首局蓝方，team_b = 首局红方（后续换边不变）
		TeamAID:    req.BlueTeamID,
		TeamBID:    req.RedTeamID,
		TeamAScore: 0,
		TeamBScore: 0,
	}

	h.roomsMu.Lock()
	h.rooms[roomID] = room
	h.roomsMu.Unlock()

	state := h.buildRoomState(roomID, room)
	c.JSON(http.StatusCreated, gin.H{"code": 0, "data": state})
}

// JoinRoom POST /api/rooms/:id/join — 加入房间
func (h *RoomHandler) JoinRoom(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		TeamID int    `json:"teamId"`
		Side   string `json:"side"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	if _, ok := h.getRoom(roomID); !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"roomId": roomID,
			"side":   req.Side,
		},
	})
}

// GetRoomStatus GET /api/rooms/:id/status — 获取当前 BP 状态
func (h *RoomHandler) GetRoomStatus(c *gin.Context) {
	roomID := c.Param("id")

	room, ok := h.getRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": h.buildRoomState(roomID, room)})
}

// Ban POST /api/rooms/:id/ban — 执行 ban 操作
func (h *RoomHandler) Ban(c *gin.Context) {
	h.handleAction(c, service.Ban)
}

// Pick POST /api/rooms/:id/pick — 执行 pick 操作
func (h *RoomHandler) Pick(c *gin.Context) {
	h.handleAction(c, service.Pick)
}

// handleAction ban/pick 共用的处理流程
func (h *RoomHandler) handleAction(c *gin.Context, actionType service.ActionType) {
	roomID := c.Param("id")

	var req struct {
		HeroID int `json:"heroId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}

	room, ok := h.getRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if err := room.Engine.ExecuteAction(actionType, req.HeroID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "message": err.Error()})
		return
	}

	room.Engine.Advance()
	h.persistIfFinished(room)

	// 广播状态更新
	h.broadcastUpdate(roomID, room)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": h.buildRoomState(roomID, room)})
}

// SubmitResult POST /api/rooms/:id/result — 提交本局胜方
// body: {"winner": "blue" | "red"}
func (h *RoomHandler) SubmitResult(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		Winner string `json:"winner"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "参数错误"})
		return
	}
	if req.Winner != "blue" && req.Winner != "red" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "winner 仅支持 blue / red"})
		return
	}

	room, ok := h.getRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if !room.Engine.IsFinished() {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "本局 BP 尚未完成，不能提交胜负"})
		return
	}
	if room.Winner != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "本局胜方已提交"})
		return
	}
	if room.MatchOver {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "比赛已结束"})
		return
	}

	// 幂等写 winner 列
	if err := h.gameRepo.UpdateGameBPCompleted(room.GameID, &req.Winner); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "保存本局胜方失败"})
		return
	}

	winner := req.Winner
	room.Winner = &winner

	// 比分按"队伍身份"累加（问题4）：换边后不能把 A 队的胜场记到 B 队名下。
	// winner 是颜色，先映射回该局实际队伍，再落到 team_a / team_b 记账。
	winnerTeamID := room.BlueTeamID
	if winner == "red" {
		winnerTeamID = room.RedTeamID
	}
	room.addWin(winnerTeamID)

	// 联机模式（预留）：败方获得下一局选边权。本次仅落库，交互留待 dual 模式启用。
	loserID := room.BlueTeamID
	if winner == "blue" {
		loserID = room.RedTeamID
	}
	room.SidePickerTeamID = &loserID
	// 选边权落库失败不应阻断胜负录入主流程，仅记录日志
	if err := h.matchRepo.UpdateSidePicker(room.MatchID, loserID); err != nil {
		log.Printf("[warn] 更新选边权失败 matchID=%d: %v", room.MatchID, err)
	}

	// 达到所需胜场 → 比赛结束（无限制赛制永不为真）
	status := "ongoing"
	if need := room.winsNeeded(); need > 0 && (room.TeamAScore >= need || room.TeamBScore >= need) {
		room.MatchOver = true
		status = "finished"
	}

	// 持久化用"队伍身份"比分（team_a_score / team_b_score），与红蓝视图解耦
	if err := h.matchRepo.UpdateProgress(room.MatchID, room.TeamAScore, room.TeamBScore, room.GameNumber, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "更新比赛进度失败"})
		return
	}

	h.broadcastUpdate(roomID, room)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": h.buildRoomState(roomID, room)})
}

// NextGame POST /api/rooms/:id/next-game — 进入下一局（触发全局BP记忆规则）
//
// 请求体可选（问题4：第二局起重新选边）：
//
//	{ "blueTeamId": 12, "redTeamId": 7 }
//
// 不传 / 传 null 时沿用当前局红蓝归属，行为与改动前完全一致（向后兼容）。
func (h *RoomHandler) NextGame(c *gin.Context) {
	roomID := c.Param("id")

	// 选边参数全部可选：无 body 或空 body 都应正常走默认路径
	var req struct {
		BlueTeamID *int `json:"blueTeamId"`
		RedTeamID  *int `json:"redTeamId"`
	}
	// 忽略绑定错误（含 EOF / 空 body），由后续"两者是否都给"决定是否采用
	_ = c.ShouldBindJSON(&req)

	room, ok := h.getRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if !room.Engine.IsFinished() {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "本局 BP 尚未完成，不能进入下一局"})
		return
	}
	if room.Winner == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "请先提交本局胜方"})
		return
	}
	if room.MatchOver {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "比赛已结束，不能进入下一局"})
		return
	}
	// 上一局必须已落库，否则下一局读不到 BP 记录 → 全局BP记忆规则静默失效
	if !room.Persisted {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "上一局 BP 记录尚未保存完成，请稍后重试"})
		return
	}

	// 解析下一局红蓝归属：默认沿用当前局
	nextBlue, nextRed := room.BlueTeamID, room.RedTeamID

	// 只有在"两个 ID 都给了"时才视为一次选边请求，避免半截参数被误用
	if req.BlueTeamID != nil && req.RedTeamID != nil {
		b, r := *req.BlueTeamID, *req.RedTeamID

		if b == r {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "蓝方与红方不能是同一支队伍"})
			return
		}
		// 必须恰好是本场比赛的两支固定队伍（防止无关队伍被塞进来）
		if !room.isFixedTeam(b) || !room.isFixedTeam(r) || b == r {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "选边队伍必须是本场比赛的两支参赛队伍"})
			return
		}
		if !((b == room.TeamAID && r == room.TeamBID) || (b == room.TeamBID && r == room.TeamAID)) {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "选边队伍必须是本场比赛的两支参赛队伍"})
			return
		}

		nextBlue, nextRed = b, r
	} else if req.BlueTeamID != nil || req.RedTeamID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "选边需同时提供 blueTeamId 与 redTeamId"})
		return
	}

	gameNumber := room.GameNumber + 1

	// 落库新一局（采用选边后的红蓝归属）
	gameID, err := h.gameRepo.CreateGame(&model.Game{
		MatchID:    room.MatchID,
		GameNumber: gameNumber,
		BlueTeamID: nextBlue,
		RedTeamID:  nextRed,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "创建对局记录失败"})
		return
	}

	// 新引擎：GameService 会自动读取上一局 BP 记录并应用全局BP记忆
	// 注意：全局BP限制"跟队伍走"而非"跟颜色走"，因此换边后规则依然正确
	// （applyGlobalBPMemory 按上一局记录的 side 字段解读，side 已绑定到具体队伍）。
	engine, err := h.gameService.StartGame(room.MatchID, gameNumber, nextBlue, nextRed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "开始下一局失败：" + err.Error()})
		return
	}
	if err := engine.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "启动 BP 流程失败"})
		return
	}

	room.Engine = engine
	room.GameID = gameID
	room.GameNumber = gameNumber
	room.Winner = nil
	room.Persisted = false

	// 提交本局选边：红蓝归属切换为新一局的归属（问题4）
	room.BlueTeamID = nextBlue
	room.RedTeamID = nextRed
	// 红蓝视图比分随归属重新映射（队伍比分 TeamAScore/TeamBScore 始终不变）
	room.syncColorScores()
	// 清理一次性的选边意图，避免污染后续局
	room.NextBlueTeamID = nil
	room.NextRedTeamID = nil

	// 持久化仍使用"队伍身份"比分
	if err := h.matchRepo.UpdateProgress(room.MatchID, room.TeamAScore, room.TeamBScore, gameNumber, "ongoing"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "更新比赛进度失败"})
		return
	}

	h.broadcastUpdate(roomID, room)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": h.buildRoomState(roomID, room)})
}

// FinishMatch POST /api/rooms/:id/finish-match — 手动结束比赛（无限制赛制专用）
func (h *RoomHandler) FinishMatch(c *gin.Context) {
	roomID := c.Param("id")

	room, ok := h.getRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "房间不存在"})
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.MatchOver {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "比赛已结束"})
		return
	}

	room.MatchOver = true

	if err := h.matchRepo.UpdateProgress(room.MatchID, room.TeamAScore, room.TeamBScore, room.GameNumber, "finished"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "结束比赛失败"})
		return
	}

	h.broadcastUpdate(roomID, room)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": h.buildRoomState(roomID, room)})
}

// WebSocketHandler GET /ws — WebSocket 升级入口
func (h *RoomHandler) WebSocketHandler(c *gin.Context) {
	roomID := c.Query("roomId")
	side := c.Query("side") // "blue" | "red" | "spectator"
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "缺少 roomId"})
		return
	}
	if side == "" {
		side = "spectator"
	}

	ws.ServeWS(h.hub, c.Writer, c.Request, roomID, side)
}

// ============================================================================
// 内部辅助方法
// ============================================================================

// persistIfFinished 一局 BP 刚完成时落库（20 步 Actions + bp_completed）。
// 调用方必须已持有 room.mu。引擎完成后 ExecuteAction 会拒绝后续操作，
// 天然防重；Persisted 标志是双保险。
func (h *RoomHandler) persistIfFinished(room *Room) {
	if !room.Engine.IsFinished() || room.Persisted {
		return
	}
	if err := h.gameService.FinishGame(room.GameID, room.Engine, nil); err != nil {
		// 落库失败不阻塞 BP 流程，但必须留痕
		log.Printf("[room] 第%d局 FinishGame 落库失败: %v", room.GameNumber, err)
		return
	}
	room.Persisted = true
}

func (h *RoomHandler) buildRoomState(roomID string, room *Room) gin.H {
	engine := room.Engine
	info := engine.GetTurnInfo()

	// 构建英雄状态列表
	heroes := make([]gin.H, 0)
	for heroID := range engine.HeroPool {
		heroes = append(heroes, gin.H{
			"id":     heroID,
			"status": "available",
		})
	}
	// 补充 Permissions 中的英雄
	for heroID, perm := range engine.Permissions {
		found := false
		for _, h := range heroes {
			if h["id"] == heroID {
				found = true
				break
			}
		}
		if !found {
			status := "available"
			if !perm.CanBan(service.Blue) && !perm.CanPick(service.Blue) &&
				!perm.CanBan(service.Red) && !perm.CanPick(service.Red) {
				status = "tournament_banned"
			}
			heroes = append(heroes, gin.H{
				"id":     heroID,
				"status": status,
			})
		}
	}

	status := "ongoing"
	if engine.IsFinished() {
		status = "finished"
	}

	sideStr := "blue"
	if info.Side == service.Red {
		sideStr = "red"
	}

	matchStatus := "ongoing"
	if room.MatchOver {
		matchStatus = "finished"
	}

	var winner interface{}
	if room.Winner != nil {
		winner = *room.Winner
	}

	return gin.H{
		"roomId":     roomID,
		"status":     status,
		"matchId":    room.MatchID,
		"gameNumber": room.GameNumber,
		"boFormat":   room.BoFormat,
		"blueTeamId": engine.BlueTeamID,
		"redTeamId":  engine.RedTeamID,
		// 队名 + 队标（问题1）：前端棋盘直接用，避免再靠前端 id→name 映射
		"blueTeam":  teamBrief(h.teams.get(engine.BlueTeamID)),
		"redTeam":   teamBrief(h.teams.get(engine.RedTeamID)),
		"blueScore": room.BlueScore,
		"redScore":  room.RedScore,
		// 固定身份与选边权（问题4）：前端据此渲染局间选边 UI
		"teamAId":          room.TeamAID,
		"teamBId":          room.TeamBID,
		"sidePickerTeamId": sidePickerOut(room.SidePickerTeamID),
		"nextBlueTeamId":   room.NextBlueTeamID,
		"nextRedTeamId":    room.NextRedTeamID,
		"winner":           winner,
		"matchStatus":      matchStatus,
		"currentTurn":      int(engine.CurrentTurn),
		"round":            int(engine.Round),
		"action":           info.Action.String(),
		"side":             sideStr,
		"blueBanned":       engine.BlueBanned,
		"redBanned":        engine.RedBanned,
		"bluePicked":       engine.BluePicked,
		"redPicked":        engine.RedPicked,
		"heroes":           heroes,
	}
}

func (h *RoomHandler) broadcastUpdate(roomID string, room *Room) {
	state := h.buildRoomState(roomID, room)

	msgType := "bp_update"
	if room.Engine.IsFinished() {
		msgType = "bp_finished"
	}

	h.hub.Broadcast(roomID, &ws.OutgoingMessage{
		Type: msgType,
		Data: state,
	})
}

func generateRoomID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
