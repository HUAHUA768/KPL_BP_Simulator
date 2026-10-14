package handler

import (
	"strings"
	"sync"

	"kpl-bp-simulator/internal/model"
	"kpl-bp-simulator/internal/service"
)

// ============================================================================
// 战队展示辅助（问题1：BP 界面展示双方队名 + 队标）
// ============================================================================

// normalizeLogoURL 确保队标为"绝对路径"，避免前端以当前页面 URL 为基准解析导致 404。
//
// 背景：team 表 logo_url 存的是相对路径（如 "images/teams/x.png"，无前导斜杠），
// 而 BP 页地址为 /room/:id，相对路径会被解析成 /room/images/teams/x.png → 必然 404。
// 英雄图标历来是绝对路径（"/images/heroes/x.png"），此处统一为同一约定。
func normalizeLogoURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") ||
		strings.HasPrefix(raw, "https://") ||
		strings.HasPrefix(raw, "/") {
		return raw
	}
	return "/" + raw
}

// sidePickerOut 把 *int 转为可直接 JSON 序列化的值（nil → null）
func sidePickerOut(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// heroStatusString 把引擎的 HeroStatus 转成前端契约字符串
func heroStatusString(s service.HeroStatus) string {
	return s.String()
}

// teamBrief 返回前端展示所需的队伍摘要（id / name / logoUrl）。
// 查不到时返回 nil，前端据此降级为"🔵 蓝方 / 🔴 红方"。
func teamBrief(t *model.Team) interface{} {
	if t == nil {
		return nil
	}
	return map[string]interface{}{
		"id":      t.ID,
		"name":    t.Name,
		"logoUrl": normalizeLogoURL(t.LogoURL),
	}
}

// ============================================================================
// teamCache — 进程内战队缓存
//
// buildRoomState 在每一步 BP 都会被调用（REST 响应 + WebSocket 广播），
// 若每次都查库会给 BP 操作带来不必要的 DB 往返。战队数据基本静态，
// 因此按 ID 缓存；查不到（含 teamID<=0）也缓存为 nil，避免反复穿透。
// 自身带锁，可安全地被并发调用（buildRoomState 在不同房间的锁下运行）。
// ============================================================================
type teamCache struct {
	mu    sync.RWMutex
	teams map[int]*model.Team
	repo  interface {
		GetByID(id int) (*model.Team, error)
	}
}

func newTeamCache(repo interface {
	GetByID(id int) (*model.Team, error)
}) *teamCache {
	return &teamCache{teams: make(map[int]*model.Team), repo: repo}
}

// get 返回缓存的战队；未命中时回源查询并缓存（含 nil 结果，表示"确认不存在"）。
func (c *teamCache) get(teamID int) *model.Team {
	if teamID <= 0 {
		return nil
	}

	c.mu.RLock()
	t, ok := c.teams[teamID]
	c.mu.RUnlock()
	if ok {
		return t
	}

	// 回源（在锁外查询，避免长时间持锁）
	fetched, err := c.repo.GetByID(teamID)
	if err != nil {
		fetched = nil
	}

	c.mu.Lock()
	c.teams[teamID] = fetched
	c.mu.Unlock()
	return fetched
}
