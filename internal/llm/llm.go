package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type AliyunQwenClient struct {
	apiKey  string
	baseURL string
	model   string
}

func (c *AliyunQwenClient) ModelName() string {
	return c.model
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error any `json:"error"`
}

func NewAliyunQwenClient(model string) *AliyunQwenClient {
	return &AliyunQwenClient{
		apiKey:  os.Getenv("DASHSCOPE_API_KEY"),
		baseURL: normalizeChatCompletionsURL(os.Getenv("DASHSCOPE_BASE_URL")),
		model:   model,
	}
}

// Complete 向阿里云 DashScope 发送请求，并且返回模型输出文本
func (c *AliyunQwenClient) Complete(ctx context.Context, prompt string) (*Completion, error) {
	startedAt := time.Now()

	// 基本参数校验，防止返回空配置
	if c.apiKey == "" || c.baseURL == "" {
		return nil, errors.New("missing DASHSCOPE_API_KEY or DASHSCOPE_BASE_URL")
	}

	// 构造请求体
	body := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "你是一个严格遵循指令的智能助手。",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0,
		"stream":      false,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, err
	}

	// 设置 Header，标准 OpenAI 兼容格式
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 执行 HTTP 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("llm request failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result chatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode llm response: %w", err)
	}

	if len(result.Choices) == 0 {
		if result.Error != nil {
			return nil, fmt.Errorf("llm returned no choices: %v", result.Error)
		}
		return nil, errors.New("llm returned no choices")
	}

	content := result.Choices[0].Message.Content
	if content == "" {
		return nil, errors.New("llm returned empty message content")
	}

	return &Completion{
		Content:      content,
		FinishReason: result.Choices[0].FinishReason,
		LatencyMs:    time.Since(startedAt).Milliseconds(),
		Usage: CompletionUsage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
		RawResponse: string(respBody),
	}, nil
}

func normalizeChatCompletionsURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return raw
	}
	if strings.HasSuffix(raw, "/chat/completions") {
		return raw
	}
	return raw + "/chat/completions"
}
