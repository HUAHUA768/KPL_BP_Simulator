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

	// 比分 +1（team_a = 蓝方，team_b = 红方）
	if winner == "blue" {
		room.BlueScore++
	} else {
		room.RedScore++
	}

	// 达到所需胜场 → 比赛结束（无限制赛制永不为真）
	status := "ongoing"
	if need := room.winsNeeded(); need > 0 && (room.BlueScore >= need || room.RedScore >= need) {
		room.MatchOver = true
		status = "finished"
	}

	if err := h.matchRepo.UpdateProgress(room.MatchID, room.BlueScore, room.RedScore, room.GameNumber, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "更新比赛进度失败"})
		return
	}

	h.broadcastUpdate(roomID, room)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": h.buildRoomState(roomID, room)})
}

// NextGame POST /api/rooms/:id/next-game — 进入下一局（触发全局BP记忆规则）
func (h *RoomHandler) NextGame(c *gin.Context) {
	roomID := c.Param("id")

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

	gameNumber := room.GameNumber + 1

	// 落库新一局
	gameID, err := h.gameRepo.CreateGame(&model.Game{
		MatchID:    room.MatchID,
		GameNumber: gameNumber,
		BlueTeamID: room.BlueTeamID,
		RedTeamID:  room.RedTeamID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "创建对局记录失败"})
		return
	}

	// 新引擎：GameService 会自动读取上一局 BP 记录并应用全局BP记忆
	engine, err := h.gameService.StartGame(room.MatchID, gameNumber, room.BlueTeamID, room.RedTeamID)
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

	if err := h.matchRepo.UpdateProgress(room.MatchID, room.BlueScore, room.RedScore, gameNumber, "ongoing"); err != nil {
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

	if err := h.matchRepo.UpdateProgress(room.MatchID, room.BlueScore, room.RedScore, room.GameNumber, "finished"); err != nil {
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
		"roomId":      roomID,
		"status":      status,
		"matchId":     room.MatchID,
		"gameNumber":  room.GameNumber,
		"boFormat":    room.BoFormat,
		"blueTeamId":  engine.BlueTeamID,
		"redTeamId":   engine.RedTeamID,
		"blueScore":   room.BlueScore,
		"redScore":    room.RedScore,
		"winner":      winner,
		"matchStatus": matchStatus,
		"currentTurn": int(engine.CurrentTurn),
		"round":       int(engine.Round),
		"action":      info.Action.String(),
		"side":        sideStr,
		"blueBanned":  engine.BlueBanned,
		"redBanned":   engine.RedBanned,
		"bluePicked":  engine.BluePicked,
		"redPicked":   engine.RedPicked,
		"heroes":      heroes,
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
