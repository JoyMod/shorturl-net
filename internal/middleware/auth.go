package middleware

import (
	"net/http"
	auth "shorturl-platform/pkg/jwt"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(jwtManager *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过认证的路由
		if shouldSkipAuth(c.Request.URL.Path) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证令牌"})
			c.Abort()
			return
		}

		// 提取Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminMiddleware 管理员权限中间件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// 不需要认证的路由
func shouldSkipAuth(path string) bool {
	skipPaths := []string{
		"/",
		"/admin",
		"/health",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/:code", // 短链接重定向
	}

	for _, skipPath := range skipPaths {
		if path == skipPath || strings.HasPrefix(path, "/static/") {
			return true
		}
	}
	return false
}
