package middleware

import (
	"net/http"
	"shorturl-platform/internal/config"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RateLimit 全局限流中间件
func RateLimit(redisClient *redis.Client, limitConfig *config.Limit) gin.HandlerFunc {
	if !limitConfig.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 基于内存的限流器
	limiter := rate.NewLimiter(rate.Limit(limitConfig.Requests), int(limitConfig.Burst))
	var mu sync.Mutex

	return func(c *gin.Context) {
		// 跳过特定路径
		for _, path := range limitConfig.SkipPaths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		mu.Lock()
		defer mu.Unlock()

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
