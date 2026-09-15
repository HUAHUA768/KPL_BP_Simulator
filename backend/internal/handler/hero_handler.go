package handler

import (
	"net/http"

	"kpl-bp-simulator/internal/repository"

	"github.com/gin-gonic/gin"
)

// HeroHandler 英雄相关接口
type HeroHandler struct {
	heroRepo repository.HeroRepo
}

// NewHeroHandler 创建 HeroHandler
func NewHeroHandler(heroRepo repository.HeroRepo) *HeroHandler {
	return &HeroHandler{heroRepo: heroRepo}
}

// ListHeroes GET /api/heroes — 获取英雄列表
// 查询参数: status (可选, "available" | "banned" | "picked")
func (h *HeroHandler) ListHeroes(c *gin.Context) {
	heroes, err := h.heroRepo.ListActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "获取英雄列表失败",
		})
		return
	}

	// 构造响应数据
	type HeroDTO struct {
		ID       int      `json:"id"`
		Name     string   `json:"name"`
		IconPath string   `json:"iconPath"`
		Lanes    []string `json:"lanes"`
		Status   string   `json:"status"`
	}

	result := make([]HeroDTO, 0, len(heroes))
	for _, h := range heroes {
		result = append(result, HeroDTO{
			ID:       h.ID,
			Name:     h.Name,
			IconPath: h.IconPath,
			Lanes:    h.Lanes,
			Status:   "available",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}