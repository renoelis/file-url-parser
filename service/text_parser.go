package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"file-url-parser/config"
	"file-url-parser/model"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// ParseTextFile 解析文本文件
func ParseTextFile(filePath string) (string, error) {
	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// 返回文本内容
	return string(data), nil
}

// ParseComplexFile 解析复杂文件（Word、PDF等）
func ParseComplexFile(filePath string, fileInfo *model.FileInfo) (string, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", errors.New("文件不存在")
	}

	// 尝试使用Go处理
	if fileInfo.IsText() {
		return ParseTextFile(filePath)
	}

	// 对于复杂文件，调用Python辅助服务
	return callPythonService(filePath, fileInfo)
}

// callPythonService 调用Python辅助服务解析文件
func callPythonService(filePath string, fileInfo *model.FileInfo) (string, error) {
	pythonServiceURL := config.GetPythonServiceURL() + "/parse"

	// 创建multipart表单
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 添加文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}

	// 添加文件类型
	_ = writer.WriteField("file_type", fileInfo.FileType)

	// 完成multipart表单
	err = writer.Close()
	if err != nil {
		return "", err
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", pythonServiceURL, &requestBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("Python服务返回错误，状态码: " + resp.Status)
	}

	// 解析响应
	var result struct {
		Content string `json:"content"`
		Error   string `json:"error"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", errors.New(result.Error)
	}

	return result.Content, nil
}
