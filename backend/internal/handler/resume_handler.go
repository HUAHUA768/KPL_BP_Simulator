package handler

import (
	"log"
	"net/http"
	"strconv"

	"kpl-bp-simulator/internal/model"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// 恢复对局（"继续模拟 BP"）
//
// 背景：房间（Room）只存在于内存，后端重启后 DB 里 status='ongoing' 的比赛
// 就回不到房间。本文件负责依据已落库的数据把房间**重建**出来：
//
//	match_record  →  固定身份 / 赛制 / 大比分 / 房间号
//	game(各局)    →  每局的红蓝归属、胜方、BP 是否完成
//	ban_pick_action(当前局) → 逐字节回放到第 N 步
//
// 恢复目标 = "当前进度"：
//   - 当前局 BP 未走完 → 回放已存的 N 步，停在第 N+1 步继续
//   - 当前局 BP 已走完但未录胜负 → 引擎置为 finished，前端直接进结算面板
//
// 恢复是**幂等**的：房间已在内存中时直接返回它，不会重复回放。
// ============================================================================

// ResumeRoomFromMatch POST /api/matches/:id/resume — 从对局记录恢复房间继续 BP
//
// 返回：可直接跳转的 roomId + 房间状态快照
//
//	{ "code": 0, "data": { "roomId": "ab12cd34", "state": { ...BPState... } } }
func (h *RoomHandler) ResumeRoomFromMatch(c *gin.Context) {
	matchIDStr := c.Param("id")
	matchID, err := strconv.Atoi(matchIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "matchId 格式错误"})
		return
	}

	match, err := h.matchRepo.GetByID(matchID)
	if err != nil || match == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "message": "比赛记录不存在"})
		return
	}

	// 已结束的比赛不再提供恢复（前端也不应展示入口）
	if match.Status == "finished" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "该比赛已结束，无法继续"})
		return
	}

	roomID, room, err := h.ensureRoom(match)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "恢复对局失败：" + err.Error()})
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"roomId": roomID,
			"state":  h.buildRoomState(roomID, room),
		},
	})
}

// ensureRoom 返回该比赛对应的内存房间；不存在则依据落库数据重建。
//
// 复用规则（先内存、后重建）：
//  1. match.room_id 命中内存房间 → 直接复用（最常见：同一进程内刷新页面）
//  2. 否则重建，并把（可能是新生成的）roomId 写回 match_record
func (h *RoomHandler) ensureRoom(match *model.MatchRecord) (string, *Room, error) {
	// 1) 优先复用内存中的房间
	if match.RoomID != nil && *match.RoomID != "" {
		if existing, ok := h.getRoom(*match.RoomID); ok {
			return *match.RoomID, existing, nil
		}
	}

	// 2) 重建
	roomID := generateRoomID()
	if match.RoomID != nil && *match.RoomID != "" {
		// 保留原房间号，便于前端已保存的链接/书签继续可用
		roomID = *match.RoomID
	}

	room, err := h.rebuildRoom(match, roomID)
	if err != nil {
		return "", nil, err
	}

	h.roomsMu.Lock()
	h.rooms[roomID] = room
	h.roomsMu.Unlock()

	// 回写房间号，保证下次（含重启后）仍能按 room_id 反查
	if match.RoomID == nil || *match.RoomID != roomID {
		if err := h.matchRepo.SetRoomID(match.ID, roomID); err != nil {
			log.Printf("[resume] 回写 room_id 失败 matchID=%d: %v", match.ID, err)
		}
	}

	return roomID, room, nil
}

// rebuildRoom 依据落库数据重建一场比赛的房间与 BP 引擎。
// 调用方需保证 match 未结束。
func (h *RoomHandler) rebuildRoom(match *model.MatchRecord, roomID string) (*Room, error) {
	// --- 1) 定位"当前局" ---
	// 以 current_game_num 为准；若该局不存在（异常数据）则回退到最后一局
	curNum := match.CurrentGameNum
	if curNum <= 0 {
		curNum = 1
	}
	curGame, err := h.gameRepo.GetGameByMatchAndNum(match.ID, curNum)
	if err != nil || curGame == nil {
		games, gerr := h.gameRepo.ListGamesByMatch(match.ID)
		if gerr != nil || len(games) == 0 {
			return nil, errNoGameForMatch
		}
		curGame = games[len(games)-1]
		curNum = curGame.GameNumber
	}

	// --- 2) 造引擎（走 GameService，保证与正常流程同一条构造路径：
	//         会自动读取上一局 BP 记录并派生全局BP记忆权限）---
	engine, err := h.gameService.StartGame(match.ID, curNum, curGame.BlueTeamID, curGame.RedTeamID)
	if err != nil {
		return nil, err
	}
	if err := engine.Start(); err != nil {
		return nil, err
	}

	// --- 3) 回放"当前局"已落库的 BP 步骤 ---
	actions, err := h.gameRepo.ListBPActions(curGame.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range actions {
		if err := engine.ReplayAction(a.HeroID); err != nil {
			// 历史与状态机序列不自洽 → 宁可失败也不给出错误状态
			return nil, err
		}
	}
	bpCompleted := curGame.BPCompleted || engine.IsFinished()

	// --- 4) 组装房间 ---
	room := &Room{
		MatchID:    match.ID,
		GameID:     curGame.ID,
		GameNumber: curNum,
		// 红蓝归属取"当前局"的，而不是首局（第2局起可能已换边）
		BlueTeamID: curGame.BlueTeamID,
		RedTeamID:  curGame.RedTeamID,
		BoFormat:   match.BoFormat,
		Engine:     engine,
		// 固定身份与比分
		TeamAID:    match.TeamAID,
		TeamBID:    match.TeamBID,
		TeamAScore: match.TeamAScore,
		TeamBScore: match.TeamBScore,
		// BP 已落库 → Persisted 置真，避免再次 FinishGame 重复写入
		Persisted:        bpCompleted,
		SidePickerTeamID: match.SidePickerTeamID,
	}
	// 红蓝视图比分按当前归属映射（队伍比分不变）
	room.syncColorScores()

	// 本局胜方已录入 → 恢复 Winner，使结算面板状态正确
	if curGame.Winner != nil && *curGame.Winner != "" {
		w := *curGame.Winner
		room.Winner = &w
	}

	// --- 5) 比赛是否已结束（达到所需胜场）---
	if need := room.winsNeeded(); need > 0 &&
		(room.TeamAScore >= need || room.TeamBScore >= need) {
		room.MatchOver = true
	}

	return room, nil
}

// errNoGameForMatch 表示比赛下没有任何对局记录（异常数据）
var errNoGameForMatch = &resumeError{"该比赛没有任何对局记录"}

type resumeError struct{ msg string }

func (e *resumeError) Error() string { return e.msg }
