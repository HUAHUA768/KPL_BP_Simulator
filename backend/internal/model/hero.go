package model

// Hero 英雄数据模型（对应表 hero）
type Hero struct {
	ID       int      `json:"id"        gorm:"primaryKey"`
	Name     string   `json:"name"      gorm:"column:name"`
	IconPath string   `json:"iconPath"  gorm:"column:icon_path"`
	Lanes    []string `json:"lanes"     gorm:"column:lanes;serializer:json"`
	IsActive bool     `json:"isActive"  gorm:"column:is_active"`
}

// TableName 指定表名
func (Hero) TableName() string {
	return "hero"
}