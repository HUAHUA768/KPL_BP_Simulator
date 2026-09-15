package repository

import (
	"kpl-bp-simulator/internal/model"
)

// GameRepo 对局数据访问接口（由具体实现如 MySQL 版本提供）
type GameRepo interface {
	// CreateGame 创建一局比赛，返回自增 ID
	CreateGame(g *model.Game) (int, error)
	// SaveBPAction 持久化单步 BP 操作
	SaveBPAction(a *model.BanPickAction) error
	// UpdateGameBPCompleted 标记该局 BP 完成
	UpdateGameBPCompleted(gameID int, winner *string) error
	// GetGameByMatchAndNum 按比赛ID+局号查询
	GetGameByMatchAndNum(matchID, gameNumber int) (*model.Game, error)
	// ListGamesByMatch 查询一场比赛的全部对局
	ListGamesByMatch(matchID int) ([]*model.Game, error)
	// ListBPActions 查询某局全部 BP 操作（按 step_order 排序）
	ListBPActions(gameID int) ([]*model.BanPickAction, error)
}
