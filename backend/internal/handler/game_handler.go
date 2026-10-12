package handler

import (
	"net/http"
	"strconv"

	"kpl-bp-simulator/internal/model"
	"kpl-bp-simulator/internal/repository"

	"github.com/gin-gonic/gin"
)

// GameHandler 对局相关接口
type GameHandler struct {
	gameRepo  repository.GameRepo
	teamRepo  repository.TeamRepo
	matchRepo repository.MatchRepo
}

// NewGameHandler 创建 GameHandler
func NewGameHandler(gameRepo repository.GameRepo, teamRepo repository.TeamRepo, matchRepo repository.MatchRepo) *GameHandler {
	return &GameHandler{
		gameRepo:  gameRepo,
		teamRepo:  teamRepo,
		matchRepo: matchRepo,
	}
}

// gameToJSON 把 model.Game 转成前端契约的对象（含 banPickActions）
func (h *GameHandler) gameToJSON(g *model.Game) gin.H {
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
	return gin.H{
		"id":             g.ID,
		"matchId":        g.MatchID,
		"gameNumber":     g.GameNumber,
		"blueTeamId":     g.BlueTeamID,
		"redTeamId":      g.RedTeamID,
		"winner":         winner,
		"bpCompleted":    g.BPCompleted,
		"banPickActions": actionList,
	}
}

// ListGames GET /api/games — 查询历史对局
// 查询参数: matchId (可选)；不传时返回全部对局（match_id DESC, game_number ASC）
func (h *GameHandler) ListGames(c *gin.Context) {
	matchIDStr := c.Query("matchId")

	var gameList []*model.Game
	var err error

	if matchIDStr != "" {
		matchID, convErr := strconv.Atoi(matchIDStr)
		if convErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "matchId 格式错误"})
			return
		}
		gameList, err = h.gameRepo.ListGamesByMatch(matchID)
	} else {
		// 无参：返回全部对局（修复此前直接返回空数组的问题）
		gameList, err = h.gameRepo.ListAllGames()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询对局失败"})
		return
	}

	games := make([]gin.H, 0, len(gameList))
	for _, g := range gameList {
		games = append(games, h.gameToJSON(g))
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": games})
}

// ListMatches GET /api/matches — 系列赛历史（最近 50 场，含每局 BP）
func (h *GameHandler) ListMatches(c *gin.Context) {
	matches, err := h.matchRepo.ListRecent(50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询比赛失败"})
		return
	}

	result := make([]gin.H, 0, len(matches))
	for _, m := range matches {
		games, _ := h.gameRepo.ListGamesByMatch(m.ID)
		gameList := make([]gin.H, 0, len(games))
		for _, g := range games {
			gameList = append(gameList, h.gameToJSON(g))
		}
		result = append(result, gin.H{
			"id":             m.ID,
			"teamAId":        m.TeamAID,
			"teamBId":        m.TeamBID,
			"teamAScore":     m.TeamAScore,
			"teamBScore":     m.TeamBScore,
			"boFormat":       m.BoFormat,
			"status":         m.Status,
			"currentGameNum": m.CurrentGameNum,
			"createdAt":      m.CreatedAt,
			"games":          gameList,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListTeams GET /api/teams — 战队列表（替代前端硬编码）
func (h *GameHandler) ListTeams(c *gin.Context) {
	teams, err := h.teamRepo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询战队失败"})
		return
	}

	result := make([]gin.H, 0, len(teams))
	for _, t := range teams {
		result = append(result, gin.H{
			"id":      t.ID,
			"name":    t.Name,
			"logoUrl": normalizeLogoURL(t.LogoURL),
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}
