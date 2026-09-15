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