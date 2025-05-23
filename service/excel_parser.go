package service

import (
	"errors"
	"file-url-parser/config"
	"file-url-parser/utils"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ExcelParseResult Excel解析结果
type ExcelParseResult struct {
	Data           []map[string]interface{} // 解析后的数据
	Headers        []string                 // 使用的表头（可能是原始表头或统一格式）
	OriginalHeaders []string                // 原始表头
	TableHeaders   []string                 // 表格行表头（兼容旧版本，存储第一个表格的表头）
	HasTableHeader bool                     // 是否包含表格行表头
	OriginalTableHeaders []string           // 原始表格表头（兼容旧版本，存储第一个表格的原始表头）
	
	// 多表格支持
	MultiTableHeaders map[string][]string   // 多表格表头映射，键为表格字段名，值为表头
	MultiTableOriginalHeaders map[string][]string // 多表格原始表头映射，键为表格字段名，值为原始表头
}

// splitCols 根据第二行表头，返回主字段列索引列表 & 明细字段列索引列表
// 返回：主字段列索引列表，明细字段列索引列表，明细字段分组（每组表示一个表格）
func splitCols(header2 []string, header1 []string) (mainCols []int, detailCols []int, tableGroups [][]int) {
	// 精简调试日志
	// fmt.Printf("第一行表头: %v\n", header1)
	// fmt.Printf("第二行表头: %v\n", header2)
	
	// 检查每一列
	for j, cell := range header2 {
		// 如果第二行表头为空，则该列是主字段列
		if strings.TrimSpace(cell) == "" {
			mainCols = append(mainCols, j)
		} else {
			// 否则是明细字段列
			detailCols = append(detailCols, j)
		}
	}
	
	// 如果没有明细字段，直接返回
	if len(detailCols) == 0 {
		return mainCols, detailCols, tableGroups
	}
	
	// 分析明细字段的连续区域，每个连续区域视为一个表格
	sort.Ints(detailCols) // 确保按列索引排序
	
	// 初始化第一个表格组
	currentGroup := []int{detailCols[0]}
	
	// 遍历所有明细列，按连续性分组
	for i := 1; i < len(detailCols); i++ {
		// 如果当前列与前一列相邻，添加到当前组
		if detailCols[i] == detailCols[i-1] + 1 {
			currentGroup = append(currentGroup, detailCols[i])
		} else {
			// 否则，结束当前组，开始新组
			tableGroups = append(tableGroups, currentGroup)
			currentGroup = []int{detailCols[i]}
		}
	}
	
	// 添加最后一个组
	if len(currentGroup) > 0 {
		tableGroups = append(tableGroups, currentGroup)
	}
	
	// 将明细字段范围之外的所有列都视为主字段
	for j := 0; j < len(header1); j++ {
		// 检查该列是否已经在主字段或明细字段中
		if indexOf(mainCols, j) != -1 || indexOf(detailCols, j) != -1 {
			continue
		}
		
		// 只有当第一行表头不为空时，才将其添加为主字段
		if j < len(header1) && strings.TrimSpace(header1[j]) != "" {
			mainCols = append(mainCols, j)
			// 精简调试日志
			// fmt.Printf("添加额外主字段: 索引=%d, 名称=%s\n", j, header1[j])
		}
	}
	
	// 精简调试日志
	// fmt.Printf("分割结果 - 主字段列: %v, 明细字段列: %v\n", mainCols, detailCols)
	// fmt.Printf("表格分组: %v\n", tableGroups)
	
	return mainCols, detailCols, tableGroups
}

// indexOf 找数组中某个元素的下标
func indexOf(xs []int, x int) int {
	for i, v := range xs {
		if v == x {
			return i
		}
	}
	return -1
}

// parseValue 处理单元格值的类型转换
func parseValue(cellValue string) interface{} {
	// 跳过空单元格
	if cellValue == "" {
		return ""
	}
	
	// 调试输出（仅保留关键信息）
	// fmt.Printf("处理单元格值: %s\n", cellValue)
	
	// 特殊处理URL列表（包含http://或https://并且包含逗号）
	if (strings.Contains(cellValue, "http://") || strings.Contains(cellValue, "https://")) && strings.Contains(cellValue, ",") {
		// fmt.Printf("  检测到URL列表，包含逗号，直接处理为数组\n")
		result := utils.ProcessCommaList(cellValue)
		// fmt.Printf("  转换为URL数组: %v\n", result)
		return result
	}
	
	// 特殊处理日期格式
	if isLikelyDate(cellValue) {
		// 使用formatDateString函数处理所有日期格式
		if formattedDate, ok := formatDateString(cellValue); ok {
			// fmt.Printf("  转换为日期: %s\n", formattedDate)
			return formattedDate
		}
		// 如果formatDateString无法处理，保持原始日期格式
		return cellValue
	}
	
	// 尝试解析数值
	if val, err := strconv.ParseFloat(cellValue, 64); err == nil {
		// 检查是否是整数
		if val == float64(int(val)) {
			// fmt.Printf("  转换为整数: %d\n", int(val))
			return int(val)
		}
		// fmt.Printf("  转换为浮点数: %f\n", val)
		return val
	}
	
	// 处理逗号分隔的内容
	if utils.IsCommaList(cellValue) {
		// fmt.Printf("  检测到逗号分隔的内容: %v\n", cellValue)
		result := utils.ProcessCommaList(cellValue)
		// fmt.Printf("  转换为数组: %v\n", result)
		return result
	}
	
	// 默认为字符串
	// fmt.Printf("  保持为字符串: %s\n", cellValue)
	return cellValue
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
	startRow, startCol, _ := findTableStart(rows)
	if startRow == -1 {
		// 找不到有效的表格数据
		return ExcelParseResult{Data: []map[string]interface{}{}}, nil
	}
	
	// 检查是否有足够的行来进行两行表头处理
	if startRow+1 >= len(rows) {
		// 没有足够的行，返回空结果
		return ExcelParseResult{Data: []map[string]interface{}{}}, nil
	}

	// 获取两行表头
	header1 := rows[startRow]
	header2 := rows[startRow+1]
	
	// 确保表头行有足够的列
	if len(header1) <= startCol || len(header2) <= startCol {
		return ExcelParseResult{Data: []map[string]interface{}{}}, nil
	}
	
	// 动态拆分主字段列 & 明细列
	mainCols, detailCols, tableGroups := splitCols(header2, header1)
	
	// 检查是否启用了表格行表头检测
	hasTableHeader := config.GetHasTableHeader()
	
	// 如果启用了表格行表头检测，但没有检测到明细字段，返回错误
	if hasTableHeader && len(detailCols) == 0 {
		return ExcelParseResult{}, errors.New("没有检测到任何明细字段，无法解析子表")
	}
	
	// 如果没有启用表格行表头检测，或者没有检测到明细字段，使用标准处理逻辑
	if !hasTableHeader || len(detailCols) == 0 {
		result, err := parseExcelStandard(rows, startRow, startCol)
		if err != nil {
			return ExcelParseResult{}, err
		}
		
		// 应用分页参数
		result.Data = applyPagination(result.Data, offset, limit)
		return result, nil
	}
	
	// 过滤掉空的表头字段
	var filteredMainHeaders []string
	var mainHeaderIndices []int
	
	for _, j := range mainCols {
		if j < len(header1) && strings.TrimSpace(header1[j]) != "" {
			filteredMainHeaders = append(filteredMainHeaders, header1[j])
			mainHeaderIndices = append(mainHeaderIndices, j)
		}
	}
	
	// 初始化多表格支持结构
	multiTableHeaders := make(map[string][]string)
	multiTableOriginalHeaders := make(map[string][]string)
	
	// 处理每个表格组
    // 检查是否使用表头作为键
    useHeaderAsKey := config.GetUseHeaderAsKey()
    	var allTableFieldNames []string // 存储所有表格字段名
	var allDetailCols []int // 存储所有明细列，用于兼容旧逻辑
	
	// 如果没有表格组，使用全部明细列作为一个组（兼容旧逻辑）
	if len(tableGroups) == 0 && len(detailCols) > 0 {
		tableGroups = append(tableGroups, detailCols)
	}
	
	// 处理每个表格组
	for groupIdx, group := range tableGroups {
		if len(group) == 0 {
			continue
		}
		
		// 添加到全部明细列
		allDetailCols = append(allDetailCols, group...)
		
		// 提取该组的表头
		var groupTableHeaders []string
		for _, j := range group {
			if j < len(header2) && strings.TrimSpace(header2[j]) != "" {
				groupTableHeaders = append(groupTableHeaders, header2[j])
			}
		}
		
		// 保存原始表格表头
		originalGroupTableHeaders := make([]string, len(groupTableHeaders))
		copy(originalGroupTableHeaders, groupTableHeaders)
		
		// 找到表格字段名
		var tableFieldName string
		tableColStart := group[0]
		if tableColStart < len(header1) {
			tableFieldName = header1[tableColStart]
			if strings.TrimSpace(tableFieldName) == "" {
				// 如果表格名为空，使用默认名称
				tableFieldName = fmt.Sprintf("表格%d", groupIdx+1)
			}
		} else {
			tableFieldName = fmt.Sprintf("表格%d", groupIdx+1)
		}
		
		// 统一表格字段名处理
		if !useHeaderAsKey {
			// 对于多表格，使用 detail_table_1, detail_table_2 等格式
			tableFieldName = fmt.Sprintf("detail_table_%d", groupIdx+1)
		} else {
			// 当使用表头作为键时，确保表格名称一致
			// 如果是第一个表格，使用"table"，其他表格使用"table1", "table2"等
			if groupIdx == 0 {
				tableFieldName = "table"
			} else {
				tableFieldName = fmt.Sprintf("table%d", groupIdx)
			}
		}
		
		// 添加到表格字段名列表
		allTableFieldNames = append(allTableFieldNames, tableFieldName)
		
		// 存储表格表头
		multiTableHeaders[tableFieldName] = groupTableHeaders
		multiTableOriginalHeaders[tableFieldName] = originalGroupTableHeaders
		
		// 精简调试日志
		// fmt.Printf("表格组 %d 字段名: %s, 表头: %v\n", groupIdx, tableFieldName, groupTableHeaders)
	}
	
	// 兼容旧逻辑：使用第一个表格的表头
	var tableHeaders []string
	var originalTableHeaders []string
	var tableFieldName string
	
	if len(tableGroups) > 0 && len(multiTableHeaders) > 0 {
		// 使用第一个表格的信息
		tableFieldName = allTableFieldNames[0]
		tableHeaders = multiTableHeaders[tableFieldName]
		originalTableHeaders = multiTableOriginalHeaders[tableFieldName]
		
		// 精简调试日志
		// fmt.Printf("使用第一个表格的表头: %v\n", tableHeaders)
	} else {
		// 明细字段表头（旧逻辑）
		for _, j := range detailCols {
			if j < len(header2) && strings.TrimSpace(header2[j]) != "" {
				tableHeaders = append(tableHeaders, header2[j])
			}
		}
		
		// 保存原始表格表头
		originalTableHeaders = make([]string, len(tableHeaders))
		copy(originalTableHeaders, tableHeaders)
		
		// 找到明细表格的字段名
		if len(detailCols) > 0 {
			tableColStart := detailCols[0]
			if tableColStart < len(header1) {
				tableFieldName = header1[tableColStart]
				if strings.TrimSpace(tableFieldName) == "" {
					tableFieldName = "表格"
				}
			} else {
				tableFieldName = "表格"
			}
			
			// 统一表格字段名处理
			if !useHeaderAsKey {
				// 使用固定的表格字段名"detail_table"，避免与主字段冲突
				tableFieldName = "detail_table"
			} else {
				// 确保当使用表头作为键时，表格字段名为"table"
				tableFieldName = "table"
			}
			
			// 精简调试日志
			// fmt.Printf("表格字段名: %s\n", tableFieldName)
		}
	}
	
	// 获取最大允许行数
	maxAllowedRows := config.GetMaxAllowedRows()
	
	// 检查是否有数据行
	totalDataRows := len(rows) - (startRow + 2) // 减去两行表头
	if totalDataRows <= 0 {
		// 没有数据行，只有表头
		return ExcelParseResult{
			Data:           []map[string]interface{}{},
			Headers:        filteredMainHeaders,
			OriginalHeaders: filteredMainHeaders,
			TableHeaders:   tableHeaders,
			HasTableHeader: true,
			OriginalTableHeaders: originalTableHeaders,
		}, nil
	}
	
	// 处理无限制的情况 (maxAllowedRows = -1)
	hasRowLimit := maxAllowedRows != -1
	
	// 如果有行数限制且数据量超过限制
	if hasRowLimit && totalDataRows > maxAllowedRows {
		return ExcelParseResult{}, errors.New("数据行数超过限制，最多允许 " + strconv.Itoa(maxAllowedRows) + " 行数据")
	}
	
	// 检查是否使用表头作为键
	useHeaderAsKey = config.GetUseHeaderAsKey()
	
	// 如果不使用表头作为键，创建统一格式的键名
	var mainOutputHeaders []string
	var tableOutputHeaders []string
	
	if useHeaderAsKey {
		mainOutputHeaders = filteredMainHeaders
		tableOutputHeaders = tableHeaders
	} else {
		// 使用统一格式的表头 Col_1, Col_2, ...
		mainOutputHeaders = make([]string, len(filteredMainHeaders))
		for i := range filteredMainHeaders {
			mainOutputHeaders[i] = fmt.Sprintf("Col_%d", i+1)
		}
		
		tableOutputHeaders = make([]string, len(tableHeaders))
		for i := range tableHeaders {
			tableOutputHeaders[i] = fmt.Sprintf("Col_%d", i+1)
		}
		
		// 同时更新多表格表头映射
		for tableName, headers := range multiTableHeaders {
			formattedHeaders := make([]string, len(headers))
			for i := range headers {
				formattedHeaders[i] = fmt.Sprintf("Col_%d", i+1)
			}
			multiTableHeaders[tableName] = formattedHeaders
		}
	}
	
	// 处理数据行
	var result []map[string]interface{}
	
	// 按照主记录分组处理数据
	i := startRow + 2 // 从第三行开始（跳过两行表头）
	for i < len(rows) {
		// 检查是否是新的主记录（主字段有值）
		isMainRecord := false
		for _, j := range mainCols {
			if j < 3 && j < len(rows[i]) && strings.TrimSpace(rows[i][j]) != "" {
				isMainRecord = true
				break
			}
		}
		
		if !isMainRecord {
			// 如果不是主记录，跳过
			i++
			continue
		}
		
		// 创建新的主记录
		mainRecord := make(map[string]interface{})
		
		// 填充主字段
		for idx, j := range mainHeaderIndices {
			if j < len(rows[i]) {
				headerName := filteredMainHeaders[idx]
				key := headerName
				if !useHeaderAsKey {
					key = mainOutputHeaders[idx]
				}
				
				cellValue := rows[i][j]
				if cellValue != "" {
					mainRecord[key] = parseValue(cellValue)
					// 精简调试日志
					// fmt.Printf("设置主字段 %s = %v\n", key, mainRecord[key])
				} else {
					mainRecord[key] = ""
				}
			}
		}
		
		// 处理多表格数据
		tableDataMap := make(map[string][]map[string]interface{})
		
		// 初始化每个表格的数据数组
		for _, name := range allTableFieldNames {
			tableDataMap[name] = []map[string]interface{}{}
		}
		
		// 打印所有表格名称，用于调试
		// fmt.Printf("所有表格名称: %v\n", allTableFieldNames)
		// for idx, name := range allTableFieldNames {
		// 	fmt.Printf("表格 %d: 名称=%s, 表头=%v\n", idx, name, multiTableHeaders[name])
		// }
		
		// 处理当前行和后续行的所有表格数据
		processCurrentRow := func(rowData []string, isCurrentRow bool) {
			// 处理每个表格组的数据
			for groupIdx, group := range tableGroups {
				if len(group) == 0 || groupIdx >= len(allTableFieldNames) {
					continue
				}
				
				tableFieldName := allTableFieldNames[groupIdx]
				groupTableHeaders := multiTableHeaders[tableFieldName]
				
				// 精简调试日志
				// fmt.Printf("处理表格 %s (组索引: %d), 表头: %v\n", tableFieldName, groupIdx, groupTableHeaders)
				
				// 检查是否有该表格的数据
				hasDetail := false
				for _, j := range group {
					if j < len(rowData) && strings.TrimSpace(rowData[j]) != "" {
						hasDetail = true
						break
					}
				}
				
				if hasDetail {
					detailRecord := make(map[string]interface{})
					
					// 填充明细字段
					for idx, j := range group {
						if idx < len(groupTableHeaders) && j < len(rowData) {
							headerKey := groupTableHeaders[idx]
							
							// 确定键名
							var key string
							if useHeaderAsKey {
								key = headerKey
							} else {
								// 使用统一格式的键名 Col_1, Col_2, ...
								key = fmt.Sprintf("Col_%d", idx+1)
							}
							
							cellValue := rowData[j]
							if cellValue != "" {
								detailRecord[key] = parseValue(cellValue)
								// 精简调试日志
								// fmt.Printf("  设置表格字段 %s = %v\n", key, detailRecord[key])
							} else {
								detailRecord[key] = ""
							}
						}
					}
					
					// 添加到对应表格的数据数组
					if len(detailRecord) > 0 {
						tableDataMap[tableFieldName] = append(tableDataMap[tableFieldName], detailRecord)
						if isCurrentRow {
							// 精简调试日志
							// fmt.Printf("添加当前行的表格 %s 数据: %v\n", tableFieldName, detailRecord)
						} else {
							// 精简调试日志
							// fmt.Printf("添加后续行的表格 %s 数据: %v\n", tableFieldName, detailRecord)
						}
					}
				} else {
					// 精简调试日志
					// fmt.Printf("  表格 %s 在当前行没有数据\n", tableFieldName)
				}
			}
		}
		
		// 处理当前行的表格数据
		processCurrentRow(rows[i], true)
		
		// 处理后续行中属于同一主记录的明细数据
		nextRow := i + 1
		for nextRow < len(rows) {
			// 检查是否是新的主记录
			isNextMainRecord := false
			for _, j := range mainCols {
				if j < 3 && j < len(rows[nextRow]) && strings.TrimSpace(rows[nextRow][j]) != "" {
					isNextMainRecord = true
					break
				}
			}
			
			if isNextMainRecord {
				// 如果是新的主记录，停止处理当前主记录的明细
				break
			}
			
			// 处理后续行的表格数据
			processCurrentRow(rows[nextRow], false)
			
			// 检查这一行是否有主字段的值
			for idx, j := range mainHeaderIndices {
				// 只处理不在任何表格组中的主字段
				inDetailCols := false
				for _, group := range tableGroups {
					if indexOf(group, j) != -1 {
						inDetailCols = true
						break
					}
				}
				
				if !inDetailCols && j < len(rows[nextRow]) {
					key := filteredMainHeaders[idx]
					if !useHeaderAsKey {
						key = mainOutputHeaders[idx]
					}
					
					cellValue := rows[nextRow][j]
					if cellValue != "" {
						mainRecord[key] = parseValue(cellValue)
						// 精简调试日志
						// fmt.Printf("更新主字段 %s = %v\n", key, mainRecord[key])
					}
				}
			}
			
			nextRow++
		}
		
		// 将各表格的明细记录添加到主记录中
		for name, detailRecords := range tableDataMap {
			if len(detailRecords) > 0 {
				// 检查是否有实际数据（非空值）
				hasRealData := false
				for _, record := range detailRecords {
					for _, value := range record {
						if value != nil && value != "" {
							hasRealData = true
							break
						}
					}
					if hasRealData {
						break
					}
				}
				
				if hasRealData {
					mainRecord[name] = detailRecords
					// 精简调试日志
					// fmt.Printf("添加表格 %s 的明细记录到主记录，记录数: %d\n", name, len(detailRecords))
				} else {
					// 精简调试日志
					// fmt.Printf("表格 %s 没有实际数据，不添加到主记录\n", name)
				}
			}
		}
		
		// 将主记录添加到结果中
		result = append(result, mainRecord)
		
		// 跳到下一个主记录
		i = nextRow
	}
	
	// 最终输出
	parseResult := ExcelParseResult{
		Data:                    result,
		Headers:                 mainOutputHeaders,
		OriginalHeaders:         filteredMainHeaders,
		TableHeaders:            tableOutputHeaders,  // 使用处理后的表格表头
		HasTableHeader:          true,
		OriginalTableHeaders:    originalTableHeaders,
		MultiTableHeaders:       multiTableHeaders,
		MultiTableOriginalHeaders: multiTableOriginalHeaders,
	}
	
	// 打印多表格表头信息，用于调试
	// fmt.Printf("多表格表头信息:\n")
	// for tableName, headers := range multiTableHeaders {
	// 	fmt.Printf("表格 %s 的表头: %v\n", tableName, headers)
	// }
	
	// 应用分页参数
	parseResult.Data = applyPagination(parseResult.Data, offset, limit)
	
	return parseResult, nil
}

// applyPagination 应用分页参数到数据集
func applyPagination(data []map[string]interface{}, offset int, limit int) []map[string]interface{} {
	// 如果数据为空，直接返回
	if len(data) == 0 {
		return data
	}
	
	// 如果偏移量超出范围，返回空数组
	if offset >= len(data) {
		return []map[string]interface{}{}
	}
	
	// 计算结束索引
	endIndex := len(data)
	if limit > 0 && offset+limit < endIndex {
		endIndex = offset + limit
	}
	
	// 应用分页
	return data[offset:endIndex]
}

// parseExcelStandard 使用标准方式解析Excel（无表格行表头）
func parseExcelStandard(rows [][]string, startRow int, startCol int) (ExcelParseResult, error) {
	// 使用找到的表头行
	headerRow := rows[startRow]
	originalHeaders := headerRow[startCol:]
	
	// 过滤掉空的表头字段
	var filteredOriginalHeaders []string
	for _, h := range originalHeaders {
		if strings.TrimSpace(h) != "" {
			filteredOriginalHeaders = append(filteredOriginalHeaders, h)
		}
	}
	
	// 根据配置决定使用哪种键
	useHeaderAsKey := config.GetUseHeaderAsKey()
	var headers []string
	
	if useHeaderAsKey {
		// 使用原始表头
		headers = filteredOriginalHeaders
	} else {
		// 使用统一格式的表头 Col_1, Col_2, ...，保持原始顺序
		headers = make([]string, len(filteredOriginalHeaders))
		for i := range filteredOriginalHeaders {
			// 使用1-based索引，与Excel列号保持一致
			headers[i] = fmt.Sprintf("Col_%d", i+1)
		}
	}
	
	// 获取最大允许行数
	maxAllowedRows := config.GetMaxAllowedRows()
	
	// 检查是否有数据行
	totalDataRows := len(rows) - (startRow + 1)
	if totalDataRows <= 0 {
		// 没有数据行，只有表头
		return ExcelParseResult{
			Data:           []map[string]interface{}{},
			Headers:        headers,
			OriginalHeaders: filteredOriginalHeaders,
			TableHeaders:   []string{},
			HasTableHeader: false,
			OriginalTableHeaders: []string{},
		}, nil
	}
	
	// 处理无限制的情况 (maxAllowedRows = -1)
	hasRowLimit := maxAllowedRows != -1
	
	// 如果有行数限制且数据量超过限制
	if hasRowLimit && totalDataRows > maxAllowedRows {
		return ExcelParseResult{}, errors.New("数据行数超过限制，最多允许 " + strconv.Itoa(maxAllowedRows) + " 行数据")
	}
	
	// 处理分页参数
	startIndex := startRow + 1 // 从表头下一行开始
	endIndex := len(rows)
	
	// 解析数据
	var result []map[string]interface{}
	
	// 标准处理逻辑（无表格行）
	for i := startIndex; i < endIndex; i++ {
		row := rows[i]
		
		// 跳过空行
		if len(row) <= startCol || isEmptyRow(row, startCol) {
			continue
		}
		
		item := make(map[string]interface{})
		rowData := row[startCol:] // 只处理从起始列开始的数据

		// 确保行数据与表头匹配
		for j := 0; j < len(filteredOriginalHeaders) && j < len(rowData); j++ {
			cellValue := ""
			
			// 找到对应的原始列索引
			originalIndex := -1
			count := 0
			for k, h := range originalHeaders {
				if strings.TrimSpace(h) != "" {
					if count == j {
						originalIndex = k
						break
					}
					count++
				}
			}
			
			// 如果找到了原始列索引，获取对应的单元格值
			if originalIndex >= 0 && originalIndex < len(rowData) {
				cellValue = rowData[originalIndex]
			}
			
			// 跳过空单元格
			if cellValue == "" {
				continue
			}

			// 使用parseValue处理所有字段值
			if useHeaderAsKey {
				// 使用原始表头作为键
				item[filteredOriginalHeaders[j]] = parseValue(cellValue)
			} else {
				// 使用统一格式的键名
				item[headers[j]] = parseValue(cellValue)
			}
		}

		// 只添加非空的数据项
		if len(item) > 0 {
			result = append(result, item)
		}
	}
	
	parseResult := ExcelParseResult{
		Data:           result,
		Headers:        headers,
		OriginalHeaders: filteredOriginalHeaders,
		TableHeaders:   []string{},
		HasTableHeader: false,
		OriginalTableHeaders: []string{},
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
	
	// 首先检查是否是从左上角开始的表格（最常见的情况）
	// 如果第一行和第二行的第一个单元格都有内容，很可能是表格结构
	if len(rows[0]) > 0 && strings.TrimSpace(rows[0][0]) != "" &&
	   len(rows[1]) > 0 && strings.TrimSpace(rows[1][0]) != "" {
		// 检查第一行是否有足够的连续非空单元格
		headerCount := countConsecutiveNonEmptyCells(rows[0], 0)
		if headerCount >= 1 {
			return 0, 0, rows[0]
		}
	}
	
	// 如果不是从左上角开始，则查找第一个非空单元格
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
	if value == "" {
		return value, false
	}

	// 处理标准日期格式 yyyy-MM-dd 及其变体
	if strings.Contains(value, "-") {
		// 检查是否是标准格式
		parts := strings.Split(value, " ")
		datePart := parts[0]
		timePart := ""
		if len(parts) > 1 {
			timePart = parts[1]
		}
		
		// 处理日期部分
		dateParts := strings.Split(datePart, "-")
		if len(dateParts) >= 2 {
			// 确保是有效的年月格式
			if year, err := strconv.Atoi(dateParts[0]); err == nil && year >= 1900 && year <= 2100 {
				if month, err := strconv.Atoi(dateParts[1]); err == nil && month >= 1 && month <= 12 {
					// 已经是破折号格式，保持原样
					// 但确保月份是两位数
					formattedMonth := fmt.Sprintf("%02d", month)
					
					// 构建格式化的日期部分
					formattedDatePart := fmt.Sprintf("%d-%s", year, formattedMonth)
					
					// 如果有日部分，添加日
					if len(dateParts) > 2 {
						if day, err := strconv.Atoi(dateParts[2]); err == nil && day >= 1 && day <= 31 {
							formattedDay := fmt.Sprintf("%02d", day)
							formattedDatePart = fmt.Sprintf("%s-%s", formattedDatePart, formattedDay)
						}
					}
					
					// 如果有时间部分，添加时间
					if timePart != "" {
						return fmt.Sprintf("%s %s", formattedDatePart, timePart), true
					}
					
					return formattedDatePart, true
				}
			}
		}
	}
	
	// 处理斜杠日期格式 yyyy/MM/dd 及其变体
	if strings.Contains(value, "/") {
		// 检查是否是斜杠格式
		parts := strings.Split(value, " ")
		datePart := parts[0]
		timePart := ""
		if len(parts) > 1 {
			timePart = parts[1]
		}
		
		// 处理日期部分
		dateParts := strings.Split(datePart, "/")
		if len(dateParts) >= 2 {
			// 确保是有效的年月格式
			if year, err := strconv.Atoi(dateParts[0]); err == nil && year >= 1900 && year <= 2100 {
				if month, err := strconv.Atoi(dateParts[1]); err == nil && month >= 1 && month <= 12 {
					// 转换为破折号格式
					// 确保月份是两位数
					formattedMonth := fmt.Sprintf("%02d", month)
					
					// 构建格式化的日期部分
					formattedDatePart := fmt.Sprintf("%d-%s", year, formattedMonth)
					
					// 如果有日部分，添加日
					if len(dateParts) > 2 {
						if day, err := strconv.Atoi(dateParts[2]); err == nil && day >= 1 && day <= 31 {
							formattedDay := fmt.Sprintf("%02d", day)
							formattedDatePart = fmt.Sprintf("%s-%s", formattedDatePart, formattedDay)
						}
					}
					
					// 如果有时间部分，添加时间
					if timePart != "" {
						return fmt.Sprintf("%s %s", formattedDatePart, timePart), true
					}
					
					return formattedDatePart, true
				}
			}
		}
	}
	
	// 特殊处理短年份格式的日期（如MM-DD-YY）
	if strings.Contains(value, "-") && len(value) <= 8 {
		parts := strings.Split(value, "-")
		if len(parts) == 3 {
			// 尝试将各部分解析为数字
			if month, err := strconv.Atoi(parts[0]); err == nil && month >= 1 && month <= 12 {
				if day, err := strconv.Atoi(parts[1]); err == nil && day >= 1 && day <= 31 {
					if year, err := strconv.Atoi(parts[2]); err == nil {
						// 处理两位数年份
						if year < 100 {
							// 如果年份小于50，假设是21世纪（20xx）
							// 如果年份大于等于50，假设是20世纪（19xx）
							if year < 50 {
								year += 2000
							} else {
								year += 1900
							}
						}
						// 构造标准日期格式
						formattedMonth := fmt.Sprintf("%02d", month)
						formattedDay := fmt.Sprintf("%02d", day)
						return fmt.Sprintf("%d-%s-%s", year, formattedMonth, formattedDay), true
					}
				}
			}
		}
	}
	
	// 特殊处理短年份格式的日期（如MM/DD/YY）
	if strings.Contains(value, "/") && len(value) <= 8 {
		parts := strings.Split(value, "/")
		if len(parts) == 3 {
			// 尝试将各部分解析为数字
			if month, err := strconv.Atoi(parts[0]); err == nil && month >= 1 && month <= 12 {
				if day, err := strconv.Atoi(parts[1]); err == nil && day >= 1 && day <= 31 {
					if year, err := strconv.Atoi(parts[2]); err == nil {
						// 处理两位数年份
						if year < 100 {
							// 如果年份小于50，假设是21世纪（20xx）
							// 如果年份大于等于50，假设是20世纪（19xx）
							if year < 50 {
								year += 2000
							} else {
								year += 1900
							}
						}
						// 构造标准日期格式
						formattedMonth := fmt.Sprintf("%02d", month)
						formattedDay := fmt.Sprintf("%02d", day)
						return fmt.Sprintf("%d-%s-%s", year, formattedMonth, formattedDay), true
					}
				}
			}
		}
	}

	// 如果无法解析为指定的日期格式，保持原样
	return value, false
}

// isNumeric 检查字符串是否全部为数字
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isCommaList 检查字符串是否为逗号分隔的列表 (已被utils.IsCommaList替代，但保留以兼容现有代码)
func isCommaList(value string) bool {
	return utils.IsCommaList(value)
}

// processCommaList 将逗号分隔的字符串转换为字符串数组 (已被utils.ProcessCommaList替代，但保留以兼容现有代码)
func processCommaList(value string) []string {
	return utils.ProcessCommaList(value)
}
