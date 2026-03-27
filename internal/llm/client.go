/*
定义统一接口，避免 controller 绑定具体模型实现：
Complete(ctx, prompt) (*Completion, error)
*/

package llm

import "context"

type Client interface {
	Complete(ctx context.Context, prompt string) (*Completion, error)
}

type Descriptor interface {
	ModelName() string
}

type Completion struct {
	Content      string          `json:"content"`
	FinishReason string          `json:"finish_reason"`
	LatencyMs    int64           `json:"latency_ms"`
	Usage        CompletionUsage `json:"usage"`
	RawResponse  string          `json:"raw_response"`
}

type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
