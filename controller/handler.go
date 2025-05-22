package controller

import (
	"file-url-parser/config"
	"file-url-parser/model"
	"file-url-parser/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ParseURLHandler 处理URL解析请求
func ParseURLHandler(c *gin.Context) {
	var request model.URLRequest

	// 绑定请求参数
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "无效的请求参数: " + err.Error(),
		})
		return
	}

	// 验证URL
	if request.URL == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "URL不能为空",
		})
		return
	}

	// 设置是否使用表头作为键
	var useHeaderAsKey bool
	if request.UseHeaderAsKey != nil {
		// 如果请求中指定了，则使用请求中的值
		useHeaderAsKey = *request.UseHeaderAsKey
		// 临时覆盖全局配置
		config.SetUseHeaderAsKey(useHeaderAsKey)
	}

	// 解析URL内容
	result, err := service.ParseURLContent(request.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "解析失败: " + err.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, result)
}
