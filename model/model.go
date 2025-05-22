package model

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// URLRequest 请求结构
type URLRequest struct {
	URL           string `json:"url" binding:"required"`
	UseHeaderAsKey *bool  `json:"use_header_as_key,omitempty"` // 是否使用表头作为键，null表示使用默认配置
	MaxRows       *int   `json:"max_rows,omitempty"`          // 最大行数限制，null表示使用默认配置
}

// ExcelResponse Excel解析响应
type ExcelResponse struct {
	Data []map[string]interface{} `json:"data"`
}

// OrderedExcelResponse 按表头顺序输出的Excel解析响应
type OrderedExcelResponse struct {
	Data       []map[string]interface{} `json:"data"`
	Headers    []string                 `json:"headers,omitempty"` // 表头顺序
	OriginalHeaders []string            `json:"original_headers,omitempty"` // 原始表头（当使用统一格式键名时）
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
	}

	out := Output{
		Headers: r.Headers,
		OriginalHeaders: r.OriginalHeaders,
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

	// 按表头顺序处理每一行数据
	for i, item := range r.Data {
		// 创建有序的对象
		orderedObj := OrderedJSONObject{
			Keys:   out.Headers,
			Values: make(map[string]interface{}),
		}
		
		// 填充值
		for key, val := range item {
			orderedObj.Values[key] = val
		}
		
		// 序列化有序对象
		jsonData, err := json.Marshal(orderedObj)
		if err != nil {
			return nil, err
		}
		out.Data[i] = jsonData
	}

	return json.Marshal(out)
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
