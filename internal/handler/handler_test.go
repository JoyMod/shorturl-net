package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shorturl-platform/internal/model"
	"shorturl-platform/internal/shortcode"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTest 为集成测试初始化一个干净的环境
// 它返回一个配置好的 gin.Engine 和一个清理函数
func setupTest() (*gin.Engine, func(), *ShortLinkHandler) {
	// 1. 设置测试模式
	gin.SetMode(gin.TestMode)

	// 2. 初始化内存数据库
	db, err := gorm.Open(sqlite.open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("无法连接到内存数据库: " + err.Error())
	}

	// 3. 自动迁移
	err = db.AutoMigrate(&model.ShortLink{}, &model.User{})
	if err != nil {
		panic("数据库迁移失败: " + err.Error())
	}

	// 4. 初始化 Handler
	// 注意：在测试中我们不依赖 Redis，所以传入 nil
	// 我们也不需要真实的日志记录器
	logger, _ := zap.NewDevelopment()
	sugaredLogger := logger.Sugar()

	// 使用一个简单的、非后台运行的短码生成器进行测试
	// 这样可以避免在测试期间 goroutine 泄漏
	mockGenerator := shortcode.NewGenerator(db, sugaredLogger)

	linkHandler := NewShortLinkHandler(db, nil, mockGenerator)

	// 5. 设置路由
	router := gin.Default()
	router.POST("/api/shorten", linkHandler.CreateShortLink)
	router.GET("/:code", linkHandler.RedirectToOriginal)

	// 6. 定义清理函数
	cleanup := func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
		mockGenerator.Stop() // 确保生成器被停止
	}

	return router, cleanup, linkHandler
}

// TestShortLinkHandler_Integration 测试创建和重定向的完整流程
func TestShortLinkHandler_Integration(t *testing.T) {
	router, cleanup, _ := setupTest()
	defer cleanup()

	// === 步骤 1: 创建一个新的短链接 ===
	originalURL := "https://www.google.com/very/long/path/that/needs/shortening"

	// 准备请求体
	reqBody := CreateShortLinkRequest{URL: originalURL}
	bodyBytes, _ := json.Marshal(reqBody)

	// 发起 POST 请求
	req, _ := http.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 断言创建成功
	assert.Equal(t, http.StatusCreated, w.Code, "创建短链接时，状态码应为 201 Created")

	// 解析响应
	var createResp CreateShortLinkResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	assert.NoError(t, err, "解析创建响应时不应出错")
	assert.NotEmpty(t, createResp.ShortURL, "响应中应包含短链接 URL")

	// 从 URL 中提取短码
	// 例如: http://example.com/abcdef -> abcdef
	shortCode := createResp.ShortURL[len(createResp.ShortURL)-7:]

	// === 步骤 2: 访问短链接并验证重定向 ===

	// 发起 GET 请求
	req, _ = http.NewRequest(http.MethodGet, "/"+shortCode, nil)

	// 创建一个新的 recorder
	w = httptest.NewRecorder()

	// 为了测试重定向，我们需要一个能处理重定向的客户端
	// httptest.Recorder 不会自动跟随重定向，但它会记录重定向信息
	router.ServeHTTP(w, req)

	// 断言重定向
	assert.Equal(t, http.StatusFound, w.Code, "访问短码时，状态码应为 302 Found")

	// 验证重定向的目标地址
	redirectURL := w.Header().Get("Location")
	assert.Equal(t_testing.T, originalURL, redirectURL, "重定向的 URL 应与原始 URL 匹配")
}
