package router

import (
	"file-url-parser/config"
	"file-url-parser/controller"
	"file-url-parser/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 添加CORS中间件
	r.Use(corsMiddleware())

	// 创建速率限制器
	rateLimiter := middleware.NewRateLimiter(config.GetRateLimit())

	// 文件处理路由
	fileProcess := r.Group("/fileProcess")
	{
		// 应用速率限制中间件
		fileProcess.Use(middleware.RateLimit(rateLimiter))
		
		// 文件解析接口
		fileProcess.POST("/parse", controller.ParseURLHandler)
	}

	return r
}

// corsMiddleware 处理跨域请求
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
