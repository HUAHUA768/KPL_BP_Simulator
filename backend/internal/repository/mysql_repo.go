package repository

import (
	"kpl-bp-simulator/internal/model"

	"gorm.io/gorm"
)

// ============================================================================
// MySQLHeroRepo — 英雄数据访问
// ============================================================================

type MySQLHeroRepo struct {
	db *gorm.DB
}

func NewMySQLHeroRepo(db *gorm.DB) *MySQLHeroRepo {
	return &MySQLHeroRepo{db: db}
}

func (r *MySQLHeroRepo) ListActive() ([]*model.Hero, error) {
	var heroes []*model.Hero
	err := r.db.Where("is_active = ?", true).Find(&heroes).Error
	return heroes, err
}

func (r *MySQLHeroRepo) GetByID(id int) (*model.Hero, error) {
	var hero model.Hero
	err := r.db.First(&hero, id).Error
	if err != nil {
		return nil, err
	}
	return &hero, nil
}

// ============================================================================
// MySQLTeamRepo — 战队数据访问
// ============================================================================

type MySQLTeamRepo struct {
	db *gorm.DB
}

func NewMySQLTeamRepo(db *gorm.DB) *MySQLTeamRepo {
	return &MySQLTeamRepo{db: db}
}

func (r *MySQLTeamRepo) ListAll() ([]*model.Team, error) {
	var teams []*model.Team
	err := r.db.Find(&teams).Error
	return teams, err
}

func (r *MySQLTeamRepo) GetByID(id int) (*model.Team, error) {
	var team model.Team
	err := r.db.First(&team, id).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// ============================================================================
// MySQLGameRepo — 对局数据访问
// ============================================================================

type MySQLGameRepo struct {
	db *gorm.DB
}

func NewMySQLGameRepo(db *gorm.DB) *MySQLGameRepo {
	return &MySQLGameRepo{db: db}
}

func (r *MySQLGameRepo) CreateGame(g *model.Game) (int, error) {
	if err := r.db.Create(g).Error; err != nil {
		return 0, err
	}
	return g.ID, nil
}

func (r *MySQLGameRepo) SaveBPAction(a *model.BanPickAction) error {
	return r.db.Create(a).Error
}

// DeleteBPActions 删除某局全部 BP 操作（重建/回放前清理，保证不产生重复步骤）
func (r *MySQLGameRepo) DeleteBPActions(gameID int) error {
	return r.db.Where("game_id = ?", gameID).Delete(&model.BanPickAction{}).Error
}

func (r *MySQLGameRepo) UpdateGameBPCompleted(gameID int, winner *string) error {
	updates := map[string]interface{}{
		"bp_completed": true,
	}
	if winner != nil {
		updates["winner"] = *winner
	}
	return r.db.Model(&model.Game{}).Where("id = ?", gameID).Updates(updates).Error
}

func (r *MySQLGameRepo) GetGameByMatchAndNum(matchID, gameNumber int) (*model.Game, error) {
	var game model.Game
	err := r.db.Where("match_id = ? AND game_number = ?", matchID, gameNumber).First(&game).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func (r *MySQLGameRepo) ListGamesByMatch(matchID int) ([]*model.Game, error) {
	var games []*model.Game
	err := r.db.Where("match_id = ?", matchID).Order("game_number ASC").Find(&games).Error
	return games, err
}

func (r *MySQLGameRepo) ListBPActions(gameID int) ([]*model.BanPickAction, error) {
	var actions []*model.BanPickAction
	err := r.db.Where("game_id = ?", gameID).Order("step_order ASC").Find(&actions).Error
	return actions, err
}

// ListAllGames 查询全部对局（按 match_id DESC, game_number ASC）
func (r *MySQLGameRepo) ListAllGames() ([]*model.Game, error) {
	var games []*model.Game
	err := r.db.Order("match_id DESC, game_number ASC").Find(&games).Error
	return games, err
}

// ============================================================================
// MySQLMatchRepo — 系列赛数据访问
// ============================================================================

type MySQLMatchRepo struct {
	db *gorm.DB
}

func NewMySQLMatchRepo(db *gorm.DB) *MySQLMatchRepo {
	return &MySQLMatchRepo{db: db}
}

func (r *MySQLMatchRepo) CreateMatch(m *model.MatchRecord) (int, error) {
	if err := r.db.Create(m).Error; err != nil {
		return 0, err
	}
	return m.ID, nil
}

func (r *MySQLMatchRepo) GetByID(id int) (*model.MatchRecord, error) {
	var m model.MatchRecord
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MySQLMatchRepo) UpdateProgress(id, aScore, bScore, currentGameNum int, status string) error {
	return r.db.Model(&model.MatchRecord{}).Where("id = ?", id).Updates(map[string]interface{}{
		"team_a_score":     aScore,
		"team_b_score":     bScore,
		"current_game_num": currentGameNum,
		"status":           status,
	}).Error
}

func (r *MySQLMatchRepo) UpdateSidePicker(id, teamID int) error {
	return r.db.Model(&model.MatchRecord{}).Where("id = ?", id).
		Update("side_picker_team_id", teamID).Error
}

// SetRoomID 写入房间号（恢复对局用）
func (r *MySQLMatchRepo) SetRoomID(id int, roomID string) error {
	return r.db.Model(&model.MatchRecord{}).Where("id = ?", id).
		Update("room_id", roomID).Error
}

// GetByRoomID 按房间号反查比赛记录
func (r *MySQLMatchRepo) GetByRoomID(roomID string) (*model.MatchRecord, error) {
	var m model.MatchRecord
	if err := r.db.Where("room_id = ?", roomID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MySQLMatchRepo) ListRecent(limit int) ([]*model.MatchRecord, error) {
	var matches []*model.MatchRecord
	err := r.db.Order("id DESC").Limit(limit).Find(&matches).Error
	return matches, err
}
