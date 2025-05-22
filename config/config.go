package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 应用配置
type Config struct {
	Port             string
	PythonServiceURL string
	MaxFileSize      int64
	AllowedFormats   []string
	MaxAllowedRows   int // 添加最大允许行数配置
	UseHeaderAsKey   bool // 是否使用表头作为键
	RateLimit        int  // 接口调用频率限制（次/秒）
}

var appConfig *Config

// InitConfig 初始化配置
func InitConfig() *Config {
	if appConfig == nil {
		port := os.Getenv("PORT")
		if port == "" {
			port = "4001"
		}

		pythonServiceURL := os.Getenv("PYTHON_SERVICE_URL")
		if pythonServiceURL == "" {
			pythonServiceURL = "http://localhost:4002"
		}

		maxFileSizeStr := os.Getenv("MAX_FILE_SIZE")
		maxFileSize := int64(10 * 1024 * 1024) // 默认10MB
		if maxFileSizeStr != "" {
			if size, err := strconv.ParseInt(maxFileSizeStr, 10, 64); err == nil {
				maxFileSize = size
			}
		}

		// 从环境变量读取最大允许行数
		maxAllowedRowsStr := os.Getenv("MAX_ALLOWED_ROWS")
		maxAllowedRows := 200 // 默认200行
		if maxAllowedRowsStr != "" {
			if rows, err := strconv.Atoi(maxAllowedRowsStr); err == nil {
				maxAllowedRows = rows
			}
		}

		// 从环境变量读取是否使用表头作为键
		useHeaderAsKeyStr := os.Getenv("USE_HEADER_AS_KEY")
		useHeaderAsKey := true // 默认使用表头作为键
		if useHeaderAsKeyStr != "" {
			useHeaderAsKey = strings.ToLower(useHeaderAsKeyStr) == "true"
		}

		// 从环境变量读取接口调用频率限制
		rateLimitStr := os.Getenv("RATE_LIMIT")
		rateLimit := 240 // 默认240次/秒
		if rateLimitStr != "" {
			if limit, err := strconv.Atoi(rateLimitStr); err == nil {
				rateLimit = limit
			}
		}

		appConfig = &Config{
			Port:             port,
			PythonServiceURL: pythonServiceURL,
			MaxFileSize:      maxFileSize,
			MaxAllowedRows:   maxAllowedRows,
			UseHeaderAsKey:   useHeaderAsKey,
			RateLimit:        rateLimit,
			AllowedFormats: []string{
				".xlsx", ".xls", // Excel
				".csv",          // CSV
				".docx", ".doc", // Word
				".pdf", // PDF
				".txt", // Text
				".md",  // Markdown
			},
		}
	}
	return appConfig
}

// GetPort 获取端口号
func GetPort() string {
	if appConfig == nil {
		InitConfig()
	}
	return appConfig.Port
}

// GetPythonServiceURL 获取Python服务URL
func GetPythonServiceURL() string {
	if appConfig == nil {
		InitConfig()
	}
	return appConfig.PythonServiceURL
}

// GetMaxFileSize 获取最大文件大小
func GetMaxFileSize() int64 {
	if appConfig == nil {
		InitConfig()
	}
	return appConfig.MaxFileSize
}

// GetAllowedFormats 获取允许的文件格式
func GetAllowedFormats() []string {
	if appConfig == nil {
		InitConfig()
	}
	return appConfig.AllowedFormats
}

// GetMaxAllowedRows 获取最大允许行数
func GetMaxAllowedRows() int {
	if appConfig == nil {
		InitConfig()
	}
	return appConfig.MaxAllowedRows
}

// SetMaxAllowedRows 设置最大允许行数（临时覆盖）
func SetMaxAllowedRows(value int) {
	if appConfig == nil {
		InitConfig()
	}
	appConfig.MaxAllowedRows = value
}

// GetUseHeaderAsKey 获取是否使用表头作为键
func GetUseHeaderAsKey() bool {
	if appConfig == nil {
		InitConfig()
	}
	return appConfig.UseHeaderAsKey
}

// SetUseHeaderAsKey 设置是否使用表头作为键（临时覆盖）
func SetUseHeaderAsKey(value bool) {
	if appConfig == nil {
		InitConfig()
	}
	appConfig.UseHeaderAsKey = value
}

// GetRateLimit 获取接口调用频率限制
func GetRateLimit() int {
	if appConfig == nil {
		InitConfig()
	}
	return appConfig.RateLimit
}
