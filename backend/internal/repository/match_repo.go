package repository

import (
	"kpl-bp-simulator/internal/model"
)

// MatchRepo 系列赛（一场 BO 比赛）数据访问接口
// 约定：team_a_id = 蓝方队伍，team_b_id = 红方队伍（本模拟器全程不换边）
type MatchRepo interface {
	// CreateMatch 创建一场比赛，返回自增 ID
	CreateMatch(m *model.MatchRecord) (int, error)
	// GetByID 按 ID 查询
	GetByID(id int) (*model.MatchRecord, error)
	// UpdateProgress 一次性更新比分 / 当前局号 / 状态
	UpdateProgress(id, aScore, bScore, currentGameNum int, status string) error
	// ListRecent 查询最近 limit 场比赛（按 id DESC）
	ListRecent(limit int) ([]*model.MatchRecord, error)
}