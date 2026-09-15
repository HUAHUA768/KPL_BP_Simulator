package repository

import (
	"kpl-bp-simulator/internal/model"
)

// TeamRepo 战队数据访问接口
type TeamRepo interface {
	// ListAll 列出所有战队
	ListAll() ([]*model.Team, error)
	// GetByID 按 ID 查询战队
	GetByID(id int) (*model.Team, error)
}
