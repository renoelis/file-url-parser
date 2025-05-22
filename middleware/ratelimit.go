package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 定义速率限制器
type RateLimiter struct {
	requestCount int           // 当前请求数量
	lastReset    time.Time     // 上次重置时间
	limit        int           // 每秒允许的最大请求数
	mu           sync.Mutex    // 互斥锁，确保并发安全
}

// NewRateLimiter 创建一个新的速率限制器
func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		requestCount: 0,
		lastReset:    time.Now(),
		limit:        limit,
	}
}

// RateLimit 创建速率限制中间件
func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter.mu.Lock()
		defer limiter.mu.Unlock()

		// 检查是否需要重置计数器（过了1秒）
		now := time.Now()
		if now.Sub(limiter.lastReset) >= time.Second {
			limiter.requestCount = 0
			limiter.lastReset = now
		}

		// 检查是否超过限制
		if limiter.requestCount >= limiter.limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "接口调用频率超过限制，请稍后再试",
				"limit": limiter.limit,
				"unit":  "次/秒",
			})
			c.Abort()
			return
		}

		// 递增请求计数
		limiter.requestCount++

		// 继续处理请求
		c.Next()
	}
} 