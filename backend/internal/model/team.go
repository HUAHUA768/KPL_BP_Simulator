package model

// Team 战队数据模型（对应表 team）
type Team struct {
	ID       int    `json:"id"       gorm:"primaryKey"`
	Name     string `json:"name"     gorm:"column:name"`
	LogoURL  string `json:"logoUrl"  gorm:"column:logo_url"`
	IsPreset bool   `json:"isPreset" gorm:"column:is_preset"`
}

// TableName 指定表名
func (Team) TableName() string {
	return "team"
}