package repository

import (
	"kpl-bp-simulator/internal/model"
)

// MatchRepo 系列赛（一场 BO 比赛）数据访问接口
//
// 身份约定（重要，问题4 起语义收紧）：
//   - team_a_id / team_b_id 是**本场比赛的固定两队身份**，创建后永不改变；
//   - team_a_score / team_b_score 是**按固定身份记账**的总比分；
//   - 具体的"谁蓝谁红"是每局概念，存在 game.blue_team_id / game.red_team_id，
//     第 2 局起可由用户重新选边（单机）或由败方选边（联机）。
//
// 因此换边**不会**、也**不允许**改动 team_a/team_b 或其比分。
type MatchRepo interface {
	// CreateMatch 创建一场比赛，返回自增 ID
	CreateMatch(m *model.MatchRecord) (int, error)
	// GetByID 按 ID 查询
	GetByID(id int) (*model.MatchRecord, error)
	// UpdateProgress 一次性更新比分 / 当前局号 / 状态
	// 注意：aScore/bScore 必须是**队伍身份**比分（team_a_score / team_b_score），
	// 而非红蓝视图比分，否则换边后会把 A 队的胜场写到 B 队名下。
	UpdateProgress(id, aScore, bScore, currentGameNum int, status string) error
	// UpdateSidePicker 更新"当前拥有选边权的队伍"（联机模式 = 败方；单机为预留）
	UpdateSidePicker(id, teamID int) error
	// ListRecent 查询最近 limit 场比赛（按 id DESC）
	ListRecent(limit int) ([]*model.MatchRecord, error)
}
