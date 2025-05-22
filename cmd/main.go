package main

import (
	"file-url-parser/config"
	"file-url-parser/router"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// 设置Gin为发布模式
	gin.SetMode(gin.ReleaseMode)
	os.Setenv("GIN_MODE", "release")

	// 初始化配置
	config.InitConfig()

	// 设置路由
	r := router.SetupRouter()

	// 获取端口
	port := config.GetPort()

	// 启动服务
	log.Printf("服务启动，监听端口：%s，模式：%s", port, gin.Mode())
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
