package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"shorturl-platform/internal/model"
	auth "shorturl-platform/pkg/jwt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap" // 重新添加导入
	"gorm.io/gorm"
)

// AuthHandler 包含认证相关的处理器
type AuthHandler struct {
	db         *gorm.DB
	redis      *redis.Client
	jwtManager *auth.TokenManager
}

// NewAuthHandler 创建一个新的 AuthHandler
func NewAuthHandler(db *gorm.DB, redis *redis.Client, jwtManager *auth.TokenManager) *AuthHandler {
	return &AuthHandler{db: db, redis: redis, jwtManager: jwtManager}
}

// LoginRequest 定义了登录请求的结构体
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"admin"`
}

// RegisterRequest 定义了注册请求的结构体
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50" example:"newuser"`
	Email    string `json:"email" binding:"required,email" example:"newuser@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

// AuthResponse 定义了认证成功后的响应
type AuthResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// Login godoc
// @Summary 用户登录
// @Description 使用用户名和密码获取 JWT 令牌
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param   account  body   LoginRequest  true  "登录凭据"
// @Success 200 {object} AuthResponse "成功响应"
// @Failure 400 {object} gin.H "请求无效"
// @Failure 401 {object} gin.H "认证失败"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	var user model.User
	userKey := "user:" + req.Username
	if h.redis != nil {
		val, err := h.redis.Get(context.Background(), userKey).Result()
		if err == nil {
			if json.Unmarshal([]byte(val), &user) == nil {
				goto VerifyPassword
			}
		}
	}

	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if h.redis != nil {
		userBytes, _ := json.Marshal(user)
		h.redis.Set(context.Background(), userKey, userBytes, 1*time.Hour)
	}

VerifyPassword:
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "账户已被禁用"})
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		zap.S().Errorf("生成令牌失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	go h.db.Model(&user).Update("last_login", time.Now())
	c.JSON(http.StatusOK, AuthResponse{Token: token})
}

// Register godoc
// @Summary 用户注册
// @Description 创建一个新用户并返回 JWT 令牌
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param   account  body   RegisterRequest  true  "注册信息"
// @Success 201 {object} AuthResponse "成功响应"
// @Failure 400 {object} gin.H "请求无效或用户已存在"
// @Failure 500 {object} gin.H "服务器内部错误"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	var count int64
	h.db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
		return
	}

	user := model.User{Username: req.Username, Email: req.Email, IsActive: true, Role: "user"}
	if err := user.SetPassword(req.Password); err != nil {
		zap.S().Errorf("密码加密失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	if err := h.db.Create(&user).Error; err != nil {
		zap.S().Errorf("创建用户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	if h.redis != nil {
		userKey := "user:" + user.Username
		userBytes, _ := json.Marshal(user)
		h.redis.Set(context.Background(), userKey, userBytes, 1*time.Hour)
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		zap.S().Errorf("注册后生成令牌失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{Token: token})
}

// GetCurrentUser godoc
// @Summary 获取当前用户信息
// @Description 获取当前已登录用户的信息
// @Tags User
// @Security ApiKeyAuth
// @Produce  json
// @Success 200 {object} model.User "成功响应"
// @Failure 401 {object} gin.H "未认证"
// @Failure 404 {object} gin.H "用户不存在"
// @Router /api/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, user)
}
