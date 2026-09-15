package model

import "time"

// Game 单局数据模型（对应表 game）
type Game struct {
	ID          int       `json:"id"          gorm:"primaryKey"`
	MatchID     int       `json:"matchId"     gorm:"column:match_id"`
	GameNumber  int       `json:"gameNumber"  gorm:"column:game_number"`
	BlueTeamID  int       `json:"blueTeamId"  gorm:"column:blue_team_id"`
	RedTeamID   int       `json:"redTeamId"   gorm:"column:red_team_id"`
	Winner      *string   `json:"winner"      gorm:"column:winner"`
	BPCompleted bool      `json:"bpCompleted" gorm:"column:bp_completed"`
	CreatedAt   time.Time `json:"createdAt"   gorm:"column:created_at"`
}

// TableName 指定表名
func (Game) TableName() string {
	return "game"
}

// MatchRecord 一场比赛（BO系列赛）数据模型（对应表 match_record）
type MatchRecord struct {
	ID               int       `json:"id"               gorm:"primaryKey"`
	TeamAID          int       `json:"teamAId"          gorm:"column:team_a_id"`
	TeamBID          int       `json:"teamBId"          gorm:"column:team_b_id"`
	TeamAScore       int       `json:"teamAScore"       gorm:"column:team_a_score"`
	TeamBScore       int       `json:"teamBScore"       gorm:"column:team_b_score"`
	CurrentGameNum   int       `json:"currentGameNum"   gorm:"column:current_game_num"`
	SidePickerTeamID *int      `json:"sidePickerTeamId" gorm:"column:side_picker_team_id"`
	Status           string    `json:"status"           gorm:"column:status"`
	CreatedAt        time.Time `json:"createdAt"        gorm:"column:created_at"`
}

// TableName 指定表名
func (MatchRecord) TableName() string {
	return "match_record"
}

// BanPickAction 单步 BP 操作记录（对应表 ban_pick_action）
type BanPickAction struct {
	ID         int       `json:"id"         gorm:"primaryKey"`
	GameID     int       `json:"gameId"     gorm:"column:game_id"`
	StepOrder  int       `json:"stepOrder"  gorm:"column:step_order"`
	ActionType string    `json:"actionType" gorm:"column:action_type"`
	Side       string    `json:"side"       gorm:"column:side"`
	HeroID     int       `json:"heroId"     gorm:"column:hero_id"`
	CreatedAt  time.Time `json:"createdAt"  gorm:"column:created_at"`
}

// TableName 指定表名
func (BanPickAction) TableName() string {
	return "ban_pick_action"
}