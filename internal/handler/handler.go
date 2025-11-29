package handler

import (
	"context"
	"net/http"
	"shorturl-platform/internal/model"
	"shorturl-platform/internal/shortcode" // 导入 shortcode 包
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ShortLinkHandler 处理器
type ShortLinkHandler struct {
	db            *gorm.DB
	redis         *redis.Client
	codeGenerator *shortcode.Generator // 添加 codeGenerator 字段
}

// NewShortLinkHandler 创建处理器实例
func NewShortLinkHandler(db *gorm.DB, redisClient *redis.Client, codeGenerator *shortcode.Generator) *ShortLinkHandler {
	return &ShortLinkHandler{
		db:            db,
		redis:         redisClient,
		codeGenerator: codeGenerator, // 初始化 codeGenerator
	}
}

// IndexPage ... (保持不变)
func (h *ShortLinkHandler) IndexPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

// HealthCheck ... (保持不变)
func (h *ShortLinkHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now()})
}

// CreateShortLinkRequest ... (保持不变)
type CreateShortLinkRequest struct {
	URL string `json:"url" binding:"required,url" example:"https://github.com/gin-gonic/gin"`
}

// CreateShortLinkResponse ... (保持不变)
type CreateShortLinkResponse struct {
	ShortURL string `json:"short_url" example:"http://localhost:8080/xxxxxx"`
}

// CreateShortLink godoc
// @Summary 创建短链接
// @Description 为一个长 URL 创建一个新的短链接
// @Tags ShortLink
// @Security ApiKeyAuth
// @Accept  json
// @Produce  json
// @Param   url  body   CreateShortLinkRequest  true  "长链接 URL"
// @Success 201 {object} CreateShortLinkResponse "成功响应"
// @Failure 400 {object} gin.H "请求无效"
// @Failure 500 {object} gin.H "服务器内部错误"
// @Router /api/shorten [post]
func (h *ShortLinkHandler) CreateShortLink(c *gin.Context) {
	var req CreateShortLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 从预生成通道获取短码，这是一个高性能操作
	shortCode := h.codeGenerator.GetCode()

	shortLink := model.ShortLink{ShortCode: shortCode, OriginalURL: req.URL, IsActive: true}
	if err := h.db.Create(&shortLink).Error; err != nil {
		// 注意：在高并发下，如果通道耗尽且生成速度跟不上，这里可能会因为短码重复而失败。
		// 一个更健壮的系统会在这里实现重试逻辑，或者返回一个 "稍后重试" 的错误。
		// 为简化起见，我们暂时只记录错误。
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建短链接失败，可能是数据库错误或短码冲突"})
		return
	}

	if h.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		// 设置缓存，可以考虑将过期时间配置化
		h.redis.Set(ctx, "shortlink:"+shortCode, req.URL, 24*time.Hour)
	}

	c.JSON(http.StatusCreated, CreateShortLinkResponse{ShortURL: "http://" + c.Request.Host + "/" + shortCode})
}

// RedirectToOriginal ... (保持不变)
func (h *ShortLinkHandler) RedirectToOriginal(c *gin.Context) {
	code := c.Param("code")
	if h.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if cachedURL, err := h.redis.Get(ctx, "shortlink:"+code).Result(); err == nil {
			go h.incrementClickCount(code)
			c.Redirect(http.StatusFound, cachedURL)
			return
		}
	}

	var link model.ShortLink
	if err := h.db.Where("short_code = ? AND is_active = ?", code, true).First(&link).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "链接不存在或已禁用"})
		return
	}

	go h.incrementClickCount(code)
	if h.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		h.redis.Set(ctx, "shortlink:"+code, link.OriginalURL, 24*time.Hour)
	}
	c.Redirect(http.StatusFound, link.OriginalURL)
}

// GetAllLinks ... (保持不变)
func (h *ShortLinkHandler) GetAllLinks(c *gin.Context) {
	var links []model.ShortLink
	if err := h.db.Order("created_at DESC").Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取链接失败"})
		return
	}
	c.JSON(http.StatusOK, links)
}

// incrementClickCount ... (保持不变)
func (h *ShortLinkHandler) incrementClickCount(code string) {
	h.db.Model(&model.ShortLink{}).Where("short_code = ?", code).Update("click_count", gorm.Expr("click_count + 1"))
}

// GetStats ... (保持不变)
func (h *ShortLinkHandler) GetStats(c *gin.Context) {
	var stats struct {
		TotalLinks  int64 `json:"total_links"`
		TotalClicks int64 `json:"total_clicks"`
		ActiveLinks int64 `json:"active_links"`
	}
	h.db.Model(&model.ShortLink{}).Count(&stats.TotalLinks)
	h.db.Model(&model.ShortLink{}).Select("COALESCE(SUM(click_count), 0)").Scan(&stats.TotalClicks)
	h.db.Model(&model.ShortLink{}).Where("is_active = ?", true).Count(&stats.ActiveLinks)
	c.JSON(http.StatusOK, stats)
}

// ToggleLink ... (保持不变)
func (h *ShortLinkHandler) ToggleLink(c *gin.Context) {
	code := c.Param("code")
	var link model.ShortLink
	if err := h.db.Where("short_code = ?", code).First(&link).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "链接不存在"})
		return
	}
	newStatus := !link.IsActive
	h.db.Model(&link).Update("is_active", newStatus)
	if h.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h.redis.Del(ctx, "shortlink:"+code)
	}
	c.JSON(http.StatusOK, gin.H{"message": "状态更新成功", "is_active": newStatus})
}

// DeleteLink ... (保持不变)
func (h *ShortLinkHandler) DeleteLink(c *gin.Context) {
	code := c.Param("code")
	if h.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h.redis.Del(ctx, "shortlink:"+code)
	}
	if err := h.db.Where("short_code = ?", code).Delete(&model.ShortLink{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
