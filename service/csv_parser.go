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
func ParseCSV(filePath string) (ExcelParseResult, error) {
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

	// 检查行数是否超过限制（只计算实际数据行）
	if len(rows) - startRow > maxAllowedRows + 1 { // +1 是因为还有一行是表头
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

	// 解析数据
	var result []map[string]interface{}
	for i := startRow + 1; i < len(rows); i++ {
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
