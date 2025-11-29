package main

import (
	"errors"
	"fmt"
	"net/http"
	"shorturl-platform/internal/config"
	"shorturl-platform/internal/handler"
	"shorturl-platform/internal/middleware"
	"shorturl-platform/internal/model"
	"shorturl-platform/internal/shortcode" // 导入新的 shortcode 包
	"shorturl-platform/pkg/database"
	auth "shorturl-platform/pkg/jwt"
	"shorturl-platform/pkg/logger"
	"shorturl-platform/pkg/redis"
	"time"

	_ "shorturl-platform/docs"

	"github.com/gin-gonic/gin"
	redisClient "github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ... (swagger 注释保持不变)

func main() {
	logger.InitLogger()
	defer func() {
		if err := logger.Logger.Sync(); err != nil {
			fmt.Println("日志同步失败:", err)
		}
	}()
	sugaredLogger := zap.S()

	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		sugaredLogger.Fatalf("配置加载失败: %v", err)
	}

	db, err := database.InitMySQL(cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name)
	if err != nil {
		sugaredLogger.Fatalf("数据库初始化失败: %v", err)
	}
	sugaredLogger.Info("✅ 数据库连接成功")

	err = db.AutoMigrate(&model.User{}, &model.ShortLink{})
	if err != nil {
		sugaredLogger.Fatalf("数据库迁移失败: %v", err)
	}
	sugaredLogger.Info("✅ 数据库迁移成功")

	var rdb *redisClient.Client
	if cfg.Cache.Host != "" {
		rdb, err = redis.NewRedisClient(&redis.Options{
			Host: cfg.Cache.Host, Port: cfg.Cache.Port, Password: cfg.Cache.Password, DB: cfg.Cache.DB,
		})
		if err != nil {
			sugaredLogger.Warnf("缓存连接失败: %v", err)
		} else {
			defer func() {
				if err := rdb.Close(); err != nil {
					sugaredLogger.Errorf("关闭 Redis 连接失败: %v", err)
				}
			}()
			sugaredLogger.Info("✅ 缓存连接成功")
		}
	}

	// 初始化并启动短码生成器
	shortcodeGenerator := shortcode.NewGenerator(db, sugaredLogger)
	shortcodeGenerator.Start()
	defer shortcodeGenerator.Stop()
	sugaredLogger.Info("✅ 短码生成器已启动")

	tokenManager := auth.NewManager(cfg.Auth.Secret, cfg.Auth.Issuer, cfg.Auth.ExpirationHours)
	sugaredLogger.Info("✅ 认证管理器初始化成功")

	if err := createAdminUser(db); err != nil {
		sugaredLogger.Errorf("创建管理员失败: %v", err)
	}

	if cfg.App.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.GinZapRecovery(logger.Logger, true))
	router.Use(middleware.GinZapLogger(logger.Logger))

	router.LoadHTMLGlob("web/templates/*")
	router.Static("/static", "./web/static")

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authMiddleware := middleware.AuthMiddleware(tokenManager)
	adminMiddleware := middleware.AdminMiddleware()
	rateLimitMiddleware := middleware.RateLimit(rdb, &cfg.RateLimit)
	router.Use(rateLimitMiddleware)

	// 将生成器注入到 Handler
	urlHandler := handler.NewShortLinkHandler(db, rdb, shortcodeGenerator)
	authHandler := handler.NewAuthHandler(db, rdb, tokenManager)

	registerRoutes(router, urlHandler, authHandler, authMiddleware, adminMiddleware)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	sugaredLogger.Infof("🚀 服务启动成功, 访问 http://localhost:%d", cfg.Server.Port)
	sugaredLogger.Infof("📚 Swagger 文档地址: http://localhost:%d/swagger/index.html", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		sugaredLogger.Fatalf("服务启动失败: %v", err)
	}
}

// ... (registerRoutes 和 createAdminUser 函数保持不变)
func registerRoutes(
	router *gin.Engine,
	urlHandler *handler.ShortLinkHandler,
	authHandler *handler.AuthHandler,
	authMiddleware, adminMiddleware gin.HandlerFunc,
) {
	router.GET("/", urlHandler.IndexPage)
	router.GET("/health", urlHandler.HealthCheck)
	router.GET("/:code", urlHandler.RedirectToOriginal)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/register", authHandler.Register)
	}

	api := router.Group("/api")
	api.Use(authMiddleware)
	{
		api.GET("/me", authHandler.GetCurrentUser)
		api.POST("/shorten", urlHandler.CreateShortLink)
		api.GET("/links", urlHandler.GetAllLinks)
		api.GET("/stats", urlHandler.GetStats)
	}

	admin := api.Group("")
	admin.Use(adminMiddleware)
	{
		admin.PUT("/links/:code", urlHandler.ToggleLink)
		admin.DELETE("/links/:code", urlHandler.DeleteLink)
	}
}

func createAdminUser(db *gorm.DB) error {
	var existing model.User
	if err := db.Where("username = ?", "admin").First(&existing).Error; err == nil {
		return nil
	}

	admin := model.User{Username: "admin", Email: "admin@shorturl.com", Role: "admin", IsActive: true}
	if err := admin.SetPassword("admin"); err != nil {
		return err
	}
	if err := db.Create(&admin).Error; err != nil {
		return err
	}
	zap.S().Info("✅ 默认管理员创建成功", "username", "admin")
	return nil
}
