package service

import (
	"errors"
	"file-url-parser/config"
	"file-url-parser/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExcelParseResult Excel解析结果
type ExcelParseResult struct {
	Data           []map[string]interface{} // 解析后的数据
	Headers        []string                 // 使用的表头（可能是原始表头或统一格式）
	OriginalHeaders []string                // 原始表头
}

// ParseExcel 解析Excel文件
func ParseExcel(filePath string, offset int, limit int) (ExcelParseResult, error) {
	// 打开Excel文件
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return ExcelParseResult{}, err
	}
	defer f.Close()

	// 获取第一个工作表
	sheetName := f.GetSheetList()[0]

	// 获取所有单元格
	rows, err := f.GetRows(sheetName)
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

// findTableStart 查找表格数据的实际起始位置
// 返回表头行索引、起始列索引和表头行内容
func findTableStart(rows [][]string) (int, int, []string) {
	// 至少需要两行数据（表头+数据）
	if len(rows) < 2 {
		return -1, -1, nil
	}
	
	// 查找第一个非空单元格，这可能是表头的起始位置
	for rowIdx, row := range rows {
		for colIdx, cell := range row {
			if strings.TrimSpace(cell) != "" {
				// 找到第一个非空单元格，检查下一行是否也有内容（表示这是表头行）
				if rowIdx+1 < len(rows) && len(rows[rowIdx+1]) > colIdx && strings.TrimSpace(rows[rowIdx+1][colIdx]) != "" {
					// 验证这是否是一个有效的表头行（检查是否有足够的连续非空单元格）
					headerCount := countConsecutiveNonEmptyCells(row, colIdx)
					if headerCount >= 1 {
						return rowIdx, colIdx, row
					}
				}
			}
		}
	}
	
	// 如果没有找到符合条件的表头行，使用第一行作为表头（如果有数据）
	if len(rows) > 0 && len(rows[0]) > 0 {
		return 0, 0, rows[0]
	}
	
	return -1, -1, nil
}

// countConsecutiveNonEmptyCells 计算从指定位置开始的连续非空单元格数量
func countConsecutiveNonEmptyCells(row []string, startCol int) int {
	count := 0
	for i := startCol; i < len(row); i++ {
		if strings.TrimSpace(row[i]) != "" {
			count++
		} else {
			// 如果遇到空单元格，检查后面是否还有非空单元格
			// 如果有，则继续计数，否则停止
			hasMoreNonEmpty := false
			for j := i + 1; j < len(row); j++ {
				if strings.TrimSpace(row[j]) != "" {
					hasMoreNonEmpty = true
					break
				}
			}
			if !hasMoreNonEmpty {
				break
			}
		}
	}
	return count
}

// isEmptyRow 检查行是否为空（从指定列开始）
func isEmptyRow(row []string, startCol int) bool {
	for i := startCol; i < len(row); i++ {
		if strings.TrimSpace(row[i]) != "" {
			return false
		}
	}
	return true
}

// isLikelyDate 检查字符串是否可能是日期
func isLikelyDate(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	// 检查是否包含日期分隔符
	if strings.Contains(value, "-") || strings.Contains(value, "/") || strings.Contains(value, ".") {
		return true
	}

	// 检查是否包含年份标记
	if strings.Contains(value, "年") {
		return true
	}

	return false
}

// formatDateString 格式化日期字符串
func formatDateString(value string) (string, bool) {
	value = strings.TrimSpace(value)

	// 首先检查是否为标准格式（带有"-"的格式）
	// 如果已经是标准格式，直接返回原值
	if strings.Contains(value, "-") {
		return value, true
	}

	// 只处理带有"/"的日期格式
	if !strings.Contains(value, "/") {
		// 非"/"分隔的日期格式，保持原样
		return value, true
	}

	// 处理带有"/"的日期格式
	slashFormats := []string{
		// 年月日时分秒格式
		"2006/1/2 15:04:05",
		"2006/01/02 15:04:05",
		
		// 年月日时分格式
		"2006/1/2 15:04",
		"2006/01/02 15:04",
		
		// 年月日时格式
		"2006/1/2 15",
		"2006/01/02 15",
		
		// 年月日格式
		"2006/1/2",
		"2006/01/02",
		"2006/1/2 0:00",
		"2006/01/02 0:00",
		
		// 年月格式
		"2006/1",
		"2006/01",
		
		// 年格式
		"2006/",
	}

	// 尝试解析"/"格式的日期
	for _, format := range slashFormats {
		if t, err := time.Parse(format, value); err == nil {
			// 根据原始格式的组成部分决定输出格式
			if strings.Contains(format, "15:04:05") || t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 {
				// 包含时分秒
				return t.Format("2006-01-02 15:04:05"), true
			} else if strings.Contains(format, "15:04") || t.Hour() != 0 || t.Minute() != 0 {
				// 包含时分
				return t.Format("2006-01-02 15:04"), true
			} else if strings.Contains(format, "15") || t.Hour() != 0 {
				// 只包含时
				return t.Format("2006-01-02 15"), true
			} else if strings.Contains(format, "1/2") || strings.Contains(format, "01/02") {
				// 包含年月日
				return t.Format("2006-01-02"), true
			} else if strings.Contains(format, "2006/1") || strings.Contains(format, "2006/01") {
				// 只包含年月
				return t.Format("2006-01"), true
			} else if strings.Contains(format, "2006/") {
				// 只包含年
				return t.Format("2006"), true
			}
			
			// 默认返回年月日格式
			return t.Format("2006-01-02"), true
		}
	}

	// 如果无法解析为日期，保持原样
	return value, true
}

// isCommaList 检查字符串是否为逗号分隔的列表 (已被utils.IsCommaList替代，但保留以兼容现有代码)
func isCommaList(value string) bool {
	return utils.IsCommaList(value)
}

// processCommaList 将逗号分隔的字符串转换为字符串数组 (已被utils.ProcessCommaList替代，但保留以兼容现有代码)
func processCommaList(value string) []string {
	return utils.ProcessCommaList(value)
}
