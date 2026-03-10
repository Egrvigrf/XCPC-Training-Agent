package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

type AliyunQwenClient struct {
	apiKey  string
	baseURL string
	model   string
}

func NewAliyunQwenClient(model string) *AliyunQwenClient {
	return &AliyunQwenClient{
		apiKey:  os.Getenv("DASHSCOPE_API_KEY"),
		baseURL: os.Getenv("DASHSCOPE_BASE_URL"),
		model:   model,
	}
}

// Complete 向阿里云 DashScope 发送请求，并且返回模型输出文本
func (c *AliyunQwenClient) Complete(ctx context.Context, prompt string) (string, error) {

	// 基本参数校验，防止返回空配置
	if c.apiKey == "" || c.baseURL == "" {
		return "", errors.New("missing DASHSCOPE_API_KEY or DASHSCOPE_BASE_URL")
	}

	// 构造请求体
	body := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": prompt,
			},
		},
		"temperature": 0,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/chat/completions",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "", err
	}

	// 设置 Header，标准 OpenAI 兼容格式
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 执行 HTTP 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 解析响应为弱类型 map
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// 提取 choices
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", errors.New("invalid response format")
	}

	// 提取 message.content
	message := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	content := message["content"].(string)

	return content, nil
}
