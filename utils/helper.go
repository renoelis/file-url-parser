package utils

import (
	"errors"
	"file-url-parser/model"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadFile 从URL下载文件
func DownloadFile(url string, maxSize int64) ([]byte, *model.FileInfo, error) {
	// 创建HTTP请求
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, nil, errors.New("下载失败，状态码: " + resp.Status)
	}

	// 获取文件信息
	contentDisposition := resp.Header.Get("Content-Disposition")
	contentType := resp.Header.Get("Content-Type")
	contentLength := resp.ContentLength

	// 检查文件大小
	if contentLength > maxSize {
		return nil, nil, errors.New("文件太大，超过最大限制")
	}

	// 从URL或Content-Disposition中提取文件名
	fileName := extractFileName(url, contentDisposition)
	fileType := filepath.Ext(strings.ToLower(fileName))

	// 读取文件内容
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return nil, nil, err
	}

	fileInfo := &model.FileInfo{
		FileName:    fileName,
		FileType:    fileType,
		ContentType: contentType,
		Size:        int64(len(data)),
	}

	return data, fileInfo, nil
}

// SaveTempFile 保存临时文件
func SaveTempFile(data []byte, fileName string) (string, error) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "url-parser-*"+filepath.Ext(fileName))
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// 写入数据
	if _, err := tempFile.Write(data); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

// CleanupTempFile 清理临时文件
func CleanupTempFile(filePath string) {
	os.Remove(filePath)
}

// extractFileName 从URL或Content-Disposition中提取文件名
func extractFileName(url, contentDisposition string) string {
	// 尝试从Content-Disposition中提取
	if contentDisposition != "" {
		if strings.Contains(contentDisposition, "filename=") {
			parts := strings.Split(contentDisposition, "filename=")
			if len(parts) > 1 {
				fileName := strings.Trim(parts[1], `"' `)
				if fileName != "" {
					return fileName
				}
			}
		}
	}

	// 从URL中提取
	urlPath := strings.Split(url, "?")[0] // 移除查询参数
	segments := strings.Split(urlPath, "/")
	if len(segments) > 0 {
		fileName := segments[len(segments)-1]
		if fileName != "" {
			return fileName
		}
	}

	// 默认文件名
	return "downloaded_file"
}

// IsCommaList 检查字符串是否为逗号分隔的列表
func IsCommaList(value string) bool {
	// 如果字符串包含逗号，则认为是逗号分隔的列表
	return strings.Contains(value, ",")
}

// ProcessCommaList 将逗号分隔的字符串转换为字符串数组
func ProcessCommaList(value string) []string {
	items := strings.Split(value, ",")
	// 去除每个项目的前后空格
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}
	return items
}
