package service

import (
	"errors"
	"file-url-parser/config"
	"file-url-parser/model"
	"file-url-parser/utils"
	"strings"
)

// ParseURLContent 解析URL内容
func ParseURLContent(url string, offset int, limit int) (interface{}, error) {
	// 下载文件
	data, fileInfo, err := utils.DownloadFile(url, config.GetMaxFileSize())
	if err != nil {
		return nil, err
	}

	// 检查文件类型是否支持
	if !isSupportedFileType(fileInfo.FileType) {
		return nil, errors.New("不支持的文件类型: " + fileInfo.FileType)
	}

	// 保存临时文件
	tempFilePath, err := utils.SaveTempFile(data, fileInfo.FileName)
	if err != nil {
		return nil, err
	}
	defer utils.CleanupTempFile(tempFilePath)

	// 根据文件类型处理
	switch {
	case fileInfo.IsExcel():
		// 解析Excel
		result, err := ParseExcel(tempFilePath, offset, limit)
		if err != nil {
			return nil, err
		}
		// 使用有序响应
		return model.OrderedExcelResponse{
			Data:           result.Data,
			Headers:        result.Headers,
			OriginalHeaders: result.OriginalHeaders,
		}, nil
	case fileInfo.IsCSV():
		// 解析CSV
		result, err := ParseCSV(tempFilePath, offset, limit)
		if err != nil {
			return nil, err
		}
		// 使用有序响应
		return model.OrderedExcelResponse{
			Data:           result.Data,
			Headers:        result.Headers,
			OriginalHeaders: result.OriginalHeaders,
		}, nil
	default:
		// 解析其他文件类型
		content, err := ParseComplexFile(tempFilePath, fileInfo)
		if err != nil {
			return nil, err
		}
		return model.TextResponse{Content: content}, nil
	}
}

// isSupportedFileType 检查文件类型是否支持
func isSupportedFileType(fileType string) bool {
	for _, allowedType := range config.GetAllowedFormats() {
		if strings.EqualFold(fileType, allowedType) {
			return true
		}
	}
	return false
}
