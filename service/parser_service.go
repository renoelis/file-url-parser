package service

import (
	"errors"
	"file-url-parser/config"
	"file-url-parser/model"
	"file-url-parser/utils"
	"fmt"
	"regexp"
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
		
		// 检查是否使用表头作为键
		useHeaderAsKey := config.GetUseHeaderAsKey()
		
		// 使用有序响应
		response := model.OrderedExcelResponse{
			Data:           result.Data,
			Headers:        result.Headers,
			OriginalHeaders: result.OriginalHeaders,
			TableHeaders:   result.TableHeaders,
			OriginalTableHeaders: result.OriginalTableHeaders,
			MultiTableHeaders: result.MultiTableHeaders,
			MultiTableOriginalHeaders: result.MultiTableOriginalHeaders,
		}
		
		// 查找表格字段名
		tableFieldName := findTableFieldName(result)
		if tableFieldName != "" {
			response.TableFieldName = tableFieldName
		}
		
		// 如果不使用表头作为键，但需要在响应中显示原始表头
		if !useHeaderAsKey {
			// 使用原始表头作为显示用的表头
			response.Headers = result.Headers
			response.OriginalHeaders = result.OriginalHeaders
			
			// 如果有表格表头，也使用原始表格表头
			if len(result.TableHeaders) > 0 && result.HasTableHeader {
				// 获取原始表格表头
				originalTableHeaders := getOriginalTableHeaders(result)
				if len(originalTableHeaders) > 0 {
					response.OriginalTableHeaders = originalTableHeaders
				}
			}
		}
		
		return response, nil
	case fileInfo.IsCSV():
		// 解析CSV
		result, err := ParseCSV(tempFilePath, offset, limit)
		if err != nil {
			return nil, err
		}
		
		// 检查是否使用表头作为键
		useHeaderAsKey := config.GetUseHeaderAsKey()
		
		// 使用有序响应
		response := model.OrderedExcelResponse{
			Data:           result.Data,
			Headers:        result.Headers,
			OriginalHeaders: result.OriginalHeaders,
			TableHeaders:   result.TableHeaders,
			OriginalTableHeaders: result.OriginalTableHeaders,
			MultiTableHeaders: result.MultiTableHeaders,
			MultiTableOriginalHeaders: result.MultiTableOriginalHeaders,
		}
		
		// 查找表格字段名
		tableFieldName := findTableFieldName(result)
		if tableFieldName != "" {
			response.TableFieldName = tableFieldName
		}
		
		// 如果不使用表头作为键，但需要在响应中显示原始表头
		if !useHeaderAsKey {
			// 使用原始表头作为显示用的表头
			response.Headers = result.Headers
			response.OriginalHeaders = result.OriginalHeaders
			
			// 如果有表格表头，也使用原始表格表头
			if len(result.TableHeaders) > 0 && result.HasTableHeader {
				// 获取原始表格表头
				originalTableHeaders := getOriginalTableHeaders(result)
				if len(originalTableHeaders) > 0 {
					response.OriginalTableHeaders = originalTableHeaders
				}
			}
		}
		
		return response, nil
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

// getOriginalTableHeaders 获取原始表格表头
func getOriginalTableHeaders(result ExcelParseResult) []string {
	// 如果没有表格表头，返回空
	if len(result.TableHeaders) == 0 || !result.HasTableHeader {
		return nil
	}
	
	// 如果有原始表格表头，直接返回
	if len(result.OriginalTableHeaders) > 0 {
		return result.OriginalTableHeaders
	}
	
	// 检查第一个表头是否是 Col_X 格式
	colPattern := regexp.MustCompile(`^Col_\d+$`)
	if !colPattern.MatchString(result.TableHeaders[0]) {
		// 如果不是 Col_X 格式，说明已经是原始表头
		return result.TableHeaders
	}
	
	// 如果无法获取原始表格表头，返回默认值
	originalHeaders := make([]string, len(result.TableHeaders))
	for i := range result.TableHeaders {
		colIndex := i + 1
		originalHeaders[i] = fmt.Sprintf("明细字段%d", colIndex)
	}
	return originalHeaders
}

// findTableFieldName 查找表格字段名
func findTableFieldName(result ExcelParseResult) string {
	// 如果没有表格表头，返回空
	if len(result.TableHeaders) == 0 || !result.HasTableHeader {
		return ""
	}
	
	// 检查数据中是否有表格字段
	if len(result.Data) > 0 {
		for key, val := range result.Data[0] {
			// 检查是否是表格数据
			if _, ok := val.([]map[string]interface{}); ok {
				// 找到表格字段
				return key
			}
		}
	}
	
	// 如果在数据中找不到，尝试从原始表头中推断
	if len(result.OriginalHeaders) > 0 {
		// 假设表格字段可能是"订单明细"、"明细"、"表格"等
		possibleNames := []string{"订单明细", "明细", "表格", "详情", "子表"}
		for _, name := range possibleNames {
			for _, header := range result.OriginalHeaders {
				if header == name {
					return header
				}
			}
		}
	}
	
	return ""
}
