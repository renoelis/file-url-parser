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

	// 设置是否包含表格行表头
	if request.HasTableHeader != nil {
		// 如果请求中指定了，则使用请求中的值
		hasTableHeader := *request.HasTableHeader
		// 临时覆盖全局配置
		config.SetHasTableHeader(hasTableHeader)
	}

	// 设置最大行数限制
	if request.MaxRows != nil {
		// 如果请求中指定了，则使用请求中的值
		maxRows := *request.MaxRows
		// 验证最大行数是否有效
		if maxRows < -1 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Error: "最大行数必须大于等于-1，-1表示无限制",
			})
			return
		}
		// 临时覆盖全局配置
		config.SetMaxAllowedRows(maxRows)
	}

	// 设置偏移量和每页数据量
	offset := 0
	if request.Offset != nil {
		offset = *request.Offset
		if offset < 0 {
			offset = 0
		}
	}

	limit := -1 // 默认不限制
	if request.Limit != nil {
		limit = *request.Limit
		if limit < 0 {
			limit = -1 // 无限制
		}
	}

	// 解析URL内容
	result, err := service.ParseURLContent(request.URL, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error: "解析失败: " + err.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, result)
}
