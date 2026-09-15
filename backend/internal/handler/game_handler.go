package handler

import (
	"net/http"
	"strconv"

	"kpl-bp-simulator/internal/repository"

	"github.com/gin-gonic/gin"
)

// GameHandler 对局相关接口
type GameHandler struct {
	gameRepo repository.GameRepo
	teamRepo repository.TeamRepo
}

// NewGameHandler 创建 GameHandler
func NewGameHandler(gameRepo repository.GameRepo, teamRepo repository.TeamRepo) *GameHandler {
	return &GameHandler{
		gameRepo: gameRepo,
		teamRepo: teamRepo,
	}
}

// ListGames GET /api/games — 查询历史对局
// 查询参数: matchId (可选)
func (h *GameHandler) ListGames(c *gin.Context) {
	matchIDStr := c.Query("matchId")

	var games []*struct {
		ID          int    `json:"id"`
		MatchID     int    `json:"matchId"`
		GameNumber  int    `json:"gameNumber"`
		BlueTeamID  int    `json:"blueTeamId"`
		RedTeamID   int    `json:"redTeamId"`
		Winner      string `json:"winner"`
		BPCompleted bool   `json:"bpCompleted"`
		Actions     []gin.H `json:"banPickActions"`
	}

	if matchIDStr != "" {
		matchID, err := strconv.Atoi(matchIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "matchId 格式错误"})
			return
		}

		gameList, err := h.gameRepo.ListGamesByMatch(matchID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询对局失败"})
			return
		}

		for _, g := range gameList {
			actions, _ := h.gameRepo.ListBPActions(g.ID)
			actionList := make([]gin.H, 0, len(actions))
			for _, a := range actions {
				actionList = append(actionList, gin.H{
					"stepOrder":  a.StepOrder,
					"actionType": a.ActionType,
					"side":       a.Side,
					"heroId":     a.HeroID,
				})
			}
			winner := ""
			if g.Winner != nil {
				winner = *g.Winner
			}
			games = append(games, &struct {
				ID          int    `json:"id"`
				MatchID     int    `json:"matchId"`
				GameNumber  int    `json:"gameNumber"`
				BlueTeamID  int    `json:"blueTeamId"`
				RedTeamID   int    `json:"redTeamId"`
				Winner      string `json:"winner"`
				BPCompleted bool   `json:"bpCompleted"`
				Actions     []gin.H `json:"banPickActions"`
			}{
				ID:          g.ID,
				MatchID:     g.MatchID,
				GameNumber:  g.GameNumber,
				BlueTeamID:  g.BlueTeamID,
				RedTeamID:   g.RedTeamID,
				Winner:      winner,
				BPCompleted: g.BPCompleted,
				Actions:     actionList,
			})
		}
	}

	if games == nil {
		games = []*struct {
			ID          int    `json:"id"`
			MatchID     int    `json:"matchId"`
			GameNumber  int    `json:"gameNumber"`
			BlueTeamID  int    `json:"blueTeamId"`
			RedTeamID   int    `json:"redTeamId"`
			Winner      string `json:"winner"`
			BPCompleted bool   `json:"bpCompleted"`
			Actions     []gin.H `json:"banPickActions"`
		}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": games,
	})
}