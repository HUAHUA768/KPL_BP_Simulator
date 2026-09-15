package repository

import (
	"kpl-bp-simulator/internal/model"
)

// HeroRepo 英雄数据访问接口
type HeroRepo interface {
	// ListActive 列出所有活跃英雄（用于初始化英雄池）
	ListActive() ([]*model.Hero, error)
	// GetByID 按 ID 查询英雄
	GetByID(id int) (*model.Hero, error)
}
