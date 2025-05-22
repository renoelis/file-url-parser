package service

import (
	"encoding/csv"
	"errors"
	"file-url-parser/config"
	"file-url-parser/utils"
	"fmt"
	"os"
	"strconv"
)

// ParseCSV 解析CSV文件
func ParseCSV(filePath string, offset int, limit int) (ExcelParseResult, error) {
	// 打开CSV文件
	file, err := os.Open(filePath)
	if err != nil {
		return ExcelParseResult{}, err
	}
	defer file.Close()

	// 创建CSV reader
	reader := csv.NewReader(file)

	// 读取所有行
	rows, err := reader.ReadAll()
	if err != nil {
		return ExcelParseResult{}, err
	}

	// 检查是否有数据
	if len(rows) == 0 {
		// 空文件
		return ExcelParseResult{Data: []map[string]interface{}{}}, nil
	}

	// 查找表格数据的实际起始位置
	startRow, startCol, headerRow := findTableStart(rows)
	if startRow == -1 || headerRow == nil {
		// 找不到有效的表格数据
		return ExcelParseResult{Data: []map[string]interface{}{}}, nil
	}

	// 获取最大允许行数
	maxAllowedRows := config.GetMaxAllowedRows()
	
	// 检查是否有数据行
	totalDataRows := len(rows) - (startRow + 1)
	if totalDataRows <= 0 {
		// 没有数据行，只有表头
		return ExcelParseResult{
			Data:           []map[string]interface{}{},
			Headers:        []string{},
			OriginalHeaders: []string{},
		}, nil
	}
	
	// 处理无限制的情况 (maxAllowedRows = -1)
	hasRowLimit := maxAllowedRows != -1
	
	// 如果有行数限制且数据量超过限制
	if hasRowLimit && totalDataRows > maxAllowedRows {
		return ExcelParseResult{}, errors.New("数据行数超过限制，最多允许 " + strconv.Itoa(maxAllowedRows) + " 行数据")
	}

	// 使用找到的表头行
	originalHeaders := headerRow[startCol:]
	
	// 根据配置决定使用哪种键
	useHeaderAsKey := config.GetUseHeaderAsKey()
	var headers []string
	
	if useHeaderAsKey {
		// 使用原始表头
		headers = originalHeaders
	} else {
		// 使用统一格式的表头 Col_1, Col_2, ...，保持原始顺序
		headers = make([]string, len(originalHeaders))
		for i := range originalHeaders {
			// 使用1-based索引，与Excel列号保持一致
			headers[i] = fmt.Sprintf("Col_%d", i+1)
		}
	}

	// 处理分页参数
	// 计算实际的行索引范围
	startIndex := startRow + 1 + offset // 从表头下一行开始计算偏移
	endIndex := len(rows)
	
	// 如果指定了limit并且大于0，计算结束索引
	if limit > 0 {
		endIndex = startIndex + limit
		if endIndex > len(rows) {
			endIndex = len(rows)
		}
	}
	
	// 检查起始索引是否已经超出数据范围
	if startIndex >= len(rows) {
		// 偏移量超出范围，返回空数据
		return ExcelParseResult{
			Data:           []map[string]interface{}{},
			Headers:        headers,
			OriginalHeaders: originalHeaders,
		}, nil
	}

	// 解析数据，只处理指定范围内的行
	var result []map[string]interface{}
	for i := startIndex; i < endIndex; i++ {
		row := rows[i]
		
		// 跳过空行
		if len(row) <= startCol || isEmptyRow(row, startCol) {
			continue
		}
		
		item := make(map[string]interface{})
		rowData := row[startCol:] // 只处理从起始列开始的数据

		// 确保行数据与表头匹配
		for j := 0; j < len(headers) && j < len(rowData); j++ {
			cellValue := rowData[j]
			
			// 跳过空单元格
			if cellValue == "" {
				continue
			}

			// 尝试解析数值
			if val, err := strconv.ParseFloat(cellValue, 64); err == nil {
				item[headers[j]] = val
				continue
			}

			// 特殊处理日期格式
			if isLikelyDate(cellValue) {
				// 尝试解析为标准格式日期
				if formattedDate, ok := formatDateString(cellValue); ok {
					item[headers[j]] = formattedDate
					continue
				}
			}

			// 处理逗号分隔的内容
			if utils.IsCommaList(cellValue) {
				item[headers[j]] = utils.ProcessCommaList(cellValue)
				continue
			}

			// 默认为字符串
			item[headers[j]] = cellValue
		}

		// 只添加非空的数据项
		if len(item) > 0 {
			result = append(result, item)
		}
	}

	// 对结果中的所有字段再次检查是否有逗号分隔的内容
	for i := range result {
		for key, value := range result[i] {
			if strValue, ok := value.(string); ok {
				if utils.IsCommaList(strValue) {
					result[i][key] = utils.ProcessCommaList(strValue)
				}
			}
		}
	}

	parseResult := ExcelParseResult{
		Data:           result,
		Headers:        headers,
		OriginalHeaders: originalHeaders,
	}

	return parseResult, nil
}

// 注意：以下函数已移至excel_parser.go，在此删除以避免重复声明
// isCommaList、processCommaList、isLikelyDate、formatDateString
