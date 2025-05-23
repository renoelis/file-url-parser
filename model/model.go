package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// URLRequest 请求结构
type URLRequest struct {
	URL           string `json:"url" binding:"required"`
	UseHeaderAsKey *bool  `json:"use_header_as_key,omitempty"` // 是否使用表头作为键，null表示使用默认配置
	MaxRows       *int   `json:"max_rows,omitempty"`          // 最大行数限制，null表示使用默认配置，-1表示无限制
	Offset        *int   `json:"offset,omitempty"`            // 数据偏移量，从0开始，表示从第几行开始获取数据（不包括表头）
	Limit         *int   `json:"limit,omitempty"`             // 每次获取的数据行数，不传或为null表示不限制
	HasTableHeader *bool  `json:"has_table_header,omitempty"` // 是否包含表格行表头，null表示使用默认配置
}

// ExcelResponse Excel解析响应
type ExcelResponse struct {
	Data []map[string]interface{} `json:"data"`
}

// OrderedExcelResponse 按表头顺序输出的Excel解析响应
type OrderedExcelResponse struct {
	Data           []map[string]interface{} `json:"data"`
	Headers        []string                 `json:"headers,omitempty"` // 表头顺序
	OriginalHeaders []string                `json:"original_headers,omitempty"` // 原始表头（当使用统一格式键名时）
	TableHeaders   []string                 `json:"table_headers,omitempty"` // 表格行表头（当存在表格行时）
	OriginalTableHeaders []string           `json:"original_table_headers,omitempty"` // 原始表格表头
	TableFieldName string                   `json:"table_field_name,omitempty"` // 表格字段名称
	TableColHeaders map[string][]string     `json:"table_col_headers,omitempty"` // 表格列表头映射
	TableColOriginalHeaders map[string][]string `json:"table_col_original_headers,omitempty"` // 表格列原始表头映射
	
	// 多表格支持
	MultiTableHeaders map[string][]string   `json:"multi_table_headers,omitempty"` // 多表格表头映射
	MultiTableOriginalHeaders map[string][]string `json:"multi_table_original_headers,omitempty"` // 多表格原始表头映射
}

// 正则表达式匹配 Col_数字 格式
var colPattern = regexp.MustCompile(`^Col_(\d+)$`)

// 自定义的有序JSON对象，用于确保按指定顺序输出字段
type OrderedJSONObject struct {
	Keys   []string
	Values map[string]interface{}
}

// MarshalJSON 为OrderedJSONObject实现自定义JSON序列化
func (o OrderedJSONObject) MarshalJSON() ([]byte, error) {
	var buf strings.Builder
	buf.WriteString("{")
	
	for i, key := range o.Keys {
		if i > 0 {
			buf.WriteString(",")
		}
		// 序列化键
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		
		buf.WriteString(":")
		
		// 序列化值
		if val, ok := o.Values[key]; ok {
			valJSON, err := json.Marshal(val)
			if err != nil {
				return nil, err
			}
			buf.Write(valJSON)
		} else {
			// 如果键不存在，输出null
			buf.WriteString("null")
		}
	}
	
	buf.WriteString("}")
	return []byte(buf.String()), nil
}

// MarshalJSON 自定义JSON序列化，确保按表头顺序输出
func (r OrderedExcelResponse) MarshalJSON() ([]byte, error) {
	// 创建一个新的结构体用于输出
	type Output struct {
		Data []json.RawMessage `json:"data"`
		Headers []string       `json:"headers,omitempty"`
		OriginalHeaders []string `json:"original_headers,omitempty"`
		TableHeaders []string   `json:"table_headers,omitempty"`
		OriginalTableHeaders []string `json:"original_table_headers,omitempty"`
	}

	out := Output{
		Headers: r.Headers,
		OriginalHeaders: r.OriginalHeaders,
		TableHeaders: r.TableHeaders,
		OriginalTableHeaders: r.OriginalTableHeaders,
		Data: make([]json.RawMessage, len(r.Data)),
	}

	// 检查是否需要对表头进行数字排序
	if len(r.Headers) > 0 {
		// 检查第一个表头是否是 Col_X 格式
		if colPattern.MatchString(r.Headers[0]) {
			// 如果是 Col_X 格式，按数字排序表头
			sortedHeaders := make([]string, len(r.Headers))
			copy(sortedHeaders, r.Headers)
			
			sort.Slice(sortedHeaders, func(i, j int) bool {
				// 提取数字部分
				numI := extractColNumber(sortedHeaders[i])
				numJ := extractColNumber(sortedHeaders[j])
				return numI < numJ
			})
			
			// 更新表头顺序
			out.Headers = sortedHeaders
		}
	}

	// 检查是否使用Col_X格式的键名（use_header_as_key=false的情况）
	isUsingColFormat := len(r.Headers) > 0 && colPattern.MatchString(r.Headers[0])

	// 创建输出的最终结构
	finalOutput := make(map[string]interface{})
	finalOutput["data"] = make([]json.RawMessage, len(r.Data))
	finalOutput["headers"] = out.Headers
	if len(r.OriginalHeaders) > 0 {
		finalOutput["original_headers"] = r.OriginalHeaders
	}

	// 处理表格相关的表头
	if isUsingColFormat {
		// 如果使用Col_X格式的键名，处理表头映射
		// 只有当没有多表格表头映射时，才添加通用表头
		if len(r.TableHeaders) > 0 && len(r.MultiTableHeaders) == 0 {
			finalOutput["table_headers"] = r.TableHeaders
		}
		// 只有当没有多表格原始表头映射时，才添加通用原始表头
		if len(r.OriginalTableHeaders) > 0 && len(r.MultiTableOriginalHeaders) == 0 {
			finalOutput["original_table_headers"] = r.OriginalTableHeaders
		}
	} else {
		// 使用原始表头作为键的情况
		// 只有当没有多表格表头映射时，才添加通用表头
		if len(r.TableHeaders) > 0 && len(r.MultiTableHeaders) == 0 {
			finalOutput["table_headers"] = r.TableHeaders
		}
		// 只有当没有多表格原始表头映射时，才添加通用原始表头
		if len(r.OriginalTableHeaders) > 0 && len(r.MultiTableOriginalHeaders) == 0 {
			finalOutput["original_table_headers"] = r.OriginalTableHeaders
		}
	}

	// 处理多表格表头映射
	if len(r.MultiTableHeaders) > 0 {
		for tableName, headers := range r.MultiTableHeaders {
			if len(headers) > 0 {
				// 使用表格名称作为键
				finalOutput[fmt.Sprintf("table_%s_headers", tableName)] = headers
			}
		}
	}

	// 处理多表格原始表头映射
	if len(r.MultiTableOriginalHeaders) > 0 {
		for tableName, headers := range r.MultiTableOriginalHeaders {
			if len(headers) > 0 {
				// 使用表格名称作为键
				finalOutput[fmt.Sprintf("table_%s_original_headers", tableName)] = headers
			}
		}
	}

	// 按表头顺序处理每一行数据
	for i, item := range r.Data {
		// 创建有序的对象
		orderedObj := OrderedJSONObject{
			Keys:   out.Headers,
			Values: make(map[string]interface{}),
		}
		
		// 填充值
		for key, val := range item {
			// 处理表格行数据（如果存在）
			if tableData, ok := val.([]map[string]interface{}); ok {
				// 创建有序的表格数据
				orderedTableData := make([]json.RawMessage, len(tableData))
				
				// 获取正确的表头
				var tableHeaders []string
				
				// 尝试从多表格表头映射中获取
				if headers, ok := r.MultiTableHeaders[key]; ok && len(headers) > 0 {
					tableHeaders = headers
					fmt.Printf("使用多表格表头: %s -> %v\n", key, tableHeaders)
				} else if len(r.TableHeaders) > 0 && key == "table" {
					// 如果是第一个表格，使用默认表头
					tableHeaders = r.TableHeaders
					fmt.Printf("使用默认表格表头: %v\n", tableHeaders)
				} else {
					// 如果找不到表头，使用空表头
					tableHeaders = []string{}
					fmt.Printf("找不到表头，使用空表头: %s\n", key)
				}
				
				// 处理每一行表格数据
				for j, tableItem := range tableData {
					// 创建有序的表格行对象
					orderedTableItem := OrderedJSONObject{
						Keys:   tableHeaders,
						Values: make(map[string]interface{}),
					}
					
					// 填充表格行值
					for tableKey, tableVal := range tableItem {
						orderedTableItem.Values[tableKey] = tableVal
					}
					
					// 序列化有序表格行对象
					tableItemJSON, err := json.Marshal(orderedTableItem)
					if err != nil {
						return nil, err
					}
					
					orderedTableData[j] = tableItemJSON
				}
				
				// 将有序表格数据添加到主数据项
				orderedObj.Values[key] = orderedTableData
				
				// 确保表格字段被包含在Keys中
				if !contains(orderedObj.Keys, key) {
					orderedObj.Keys = append(orderedObj.Keys, key)
				}
			} else {
				orderedObj.Values[key] = val
				
				// 确保所有字段都被包含在Keys中
				if !contains(orderedObj.Keys, key) {
					orderedObj.Keys = append(orderedObj.Keys, key)
				}
			}
		}
		
		// 序列化有序对象
		jsonData, err := json.Marshal(orderedObj)
		if err != nil {
			return nil, err
		}
		
		// 添加到最终数据
		finalOutput["data"].([]json.RawMessage)[i] = jsonData
	}

	return json.Marshal(finalOutput)
}

// contains 检查字符串是否在切片中
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// extractColNumber 从 Col_X 格式的字符串中提取数字部分
func extractColNumber(colStr string) int {
	matches := colPattern.FindStringSubmatch(colStr)
	if len(matches) == 2 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}
	return 0
}

// TextResponse 文本解析响应
type TextResponse struct {
	Content string `json:"content"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// FileInfo 文件信息
type FileInfo struct {
	FileName    string
	FileType    string
	ContentType string
	Size        int64
}

// IsExcel 判断是否为Excel文件
func (f *FileInfo) IsExcel() bool {
	return f.FileType == ".xlsx" || f.FileType == ".xls"
}

// IsWord 判断是否为Word文件
func (f *FileInfo) IsWord() bool {
	return f.FileType == ".docx" || f.FileType == ".doc"
}

// IsPDF 判断是否为PDF文件
func (f *FileInfo) IsPDF() bool {
	return f.FileType == ".pdf"
}

// IsText 判断是否为文本文件
func (f *FileInfo) IsText() bool {
	return f.FileType == ".txt" || f.FileType == ".md"
}

// IsCSV 判断是否为CSV文件
func (f *FileInfo) IsCSV() bool {
	return f.FileType == ".csv"
}
