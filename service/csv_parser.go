package service

import (
	"encoding/csv"
	"errors"
	"file-url-parser/config"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	var allTableFieldNames []string // 存储所有表格字段名
	var allDetailCols []int // 存储所有明细列，用于兼容旧逻辑
	
	// 检查是否使用表头作为键
	useHeaderAsKey := config.GetUseHeaderAsKey()
	
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
		
		// 精简日志输出
		// fmt.Printf("CSV表格组 %d 字段名: %s, 表头: %v\n", groupIdx, tableFieldName, groupTableHeaders)
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
	}
	
	// 打印调试信息（精简）
	// fmt.Printf("CSV主字段列: %v, 主字段表头: %v\n", mainCols, filteredMainHeaders)
	// fmt.Printf("CSV明细字段列: %v, 明细字段表头: %v\n", detailCols, tableHeaders)
	
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
		
		// 如果不使用表头作为键，使用固定的表格字段名，避免与主字段冲突
		if !useHeaderAsKey {
			// 使用固定的表格字段名"detail_table"，避免与主字段冲突
			tableFieldName = "detail_table"
		} else {
			// 确保当使用表头作为键时，表格字段名不为空
			if strings.TrimSpace(tableFieldName) == "" {
				tableFieldName = "订单明细"
			}
		}
		
		// 精简日志输出
		// fmt.Printf("CSV表格字段名: %s\n", tableFieldName)
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
				} else {
					mainRecord[key] = ""
				}
				
				// 精简日志输出
				// fmt.Printf("CSV设置主字段 %s = %v\n", key, mainRecord[key])
			}
		}
		
		// 处理多表格数据
		tableDataMap := make(map[string][]map[string]interface{})
		
		// 初始化每个表格的数据数组
		for _, name := range allTableFieldNames {
			tableDataMap[name] = []map[string]interface{}{}
		}
		
		// 处理当前行的表格数据
		for groupIdx, group := range tableGroups {
			if len(group) == 0 || groupIdx >= len(allTableFieldNames) {
				continue
			}
			
			tableFieldName := allTableFieldNames[groupIdx]
			groupTableHeaders := multiTableHeaders[tableFieldName]
			
			// 检查当前行是否有该表格的数据
			hasDetailInCurrentRow := false
			for _, j := range group {
				if j < len(rows[i]) && strings.TrimSpace(rows[i][j]) != "" {
					hasDetailInCurrentRow = true
					break
				}
			}
			
			if hasDetailInCurrentRow {
				detailRecord := make(map[string]interface{})
				
				// 填充明细字段
				for idx, j := range group {
					if idx < len(groupTableHeaders) && j < len(rows[i]) {
						// 确定键名
						var key string
						if useHeaderAsKey {
							key = groupTableHeaders[idx]
						} else {
							// 使用统一格式的键名 Col_1, Col_2, ...
							key = fmt.Sprintf("Col_%d", idx+1)
						}
						
						cellValue := rows[i][j]
						if cellValue != "" {
							detailRecord[key] = parseValue(cellValue)
						} else {
							detailRecord[key] = ""
						}
					}
				}
				
				// 添加到对应表格的数据数组
				if len(detailRecord) > 0 {
					tableDataMap[tableFieldName] = append(tableDataMap[tableFieldName], detailRecord)
				}
			}
		}
		
		// 查找后续行中属于同一主记录的明细数据
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
			
			// 处理每个表格组的数据
			for groupIdx, group := range tableGroups {
				if len(group) == 0 || groupIdx >= len(allTableFieldNames) {
					continue
				}
				
				tableFieldName := allTableFieldNames[groupIdx]
				groupTableHeaders := multiTableHeaders[tableFieldName]
				
				// 检查是否有该表格的数据
				hasDetail := false
				for _, j := range group {
					if j < len(rows[nextRow]) && strings.TrimSpace(rows[nextRow][j]) != "" {
						hasDetail = true
						break
					}
				}
				
				if hasDetail {
					detailRecord := make(map[string]interface{})
					
					// 填充明细字段
					for idx, j := range group {
						if idx < len(groupTableHeaders) && j < len(rows[nextRow]) {
							// 确定键名
							var key string
							if useHeaderAsKey {
								key = groupTableHeaders[idx]
							} else {
								// 使用统一格式的键名 Col_1, Col_2, ...
								key = fmt.Sprintf("Col_%d", idx+1)
							}
							
							cellValue := rows[nextRow][j]
							if cellValue != "" {
								detailRecord[key] = parseValue(cellValue)
							} else {
								detailRecord[key] = ""
							}
						}
					}
					
					// 添加到对应表格的数据数组
					if len(detailRecord) > 0 {
						tableDataMap[tableFieldName] = append(tableDataMap[tableFieldName], detailRecord)
					}
				}
			}
			
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
						// 精简日志输出
						// fmt.Printf("CSV更新主字段 %s = %v\n", key, mainRecord[key])
					}
				}
			}
			
			nextRow++
		}
		
		// 将各表格的明细记录添加到主记录中
		for name, detailRecords := range tableDataMap {
			if len(detailRecords) > 0 {
				mainRecord[name] = detailRecords
				// 精简日志输出
				// fmt.Printf("CSV添加表格 %s 的明细记录到主记录，记录数: %d\n", name, len(detailRecords))
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
		TableHeaders:            tableHeaders,
		HasTableHeader:          true,
		OriginalTableHeaders:    originalTableHeaders,
		MultiTableHeaders:       multiTableHeaders,
		MultiTableOriginalHeaders: multiTableOriginalHeaders,
	}
	
	// 应用分页参数
	parseResult.Data = applyPagination(parseResult.Data, offset, limit)
	
	return parseResult, nil
}

// 检查主字段区域是否为空
func isEmptyMainFields(row []string, startCol, endCol int) bool {
	for i := startCol; i < endCol && i < len(row); i++ {
		if strings.TrimSpace(row[i]) != "" {
			return false
		}
	}
	return true
}

// 检查表格字段区域是否为空
func isEmptyTableFields(row []string, startCol int) bool {
	for i := startCol; i < len(row); i++ {
		if strings.TrimSpace(row[i]) != "" {
			return false
		}
	}
	return true
}

// 这里删除重复定义的isLikelyDate函数，使用excel_parser.go中定义的函数
