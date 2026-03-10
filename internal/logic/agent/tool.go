/*
工具接口定义（建议带 schema，便于 prompt 自动生成）：

Name() string
Description() string
ArgsSchema() map[string]any（或你定义一个 struct）
Call(ctx, args) (any, error)
*/

package agent

import (
	"context"
	"encoding/json"
)

type ToolSchema struct {
	Parameters map[string]Parameter
	Required   []string
}

type Parameter struct {
	Type        string // "string" | "number" | "integer" | "boolean"
	Description string
	Enum        []string // optional
}

type Tool interface {
	Name() string                                                 // 对于 LLM 来说就是函数签名
	Description() string                                          // 让 LLM 理解这个工具的“语义”
	Schema() ToolSchema                                           // 提供参数结构，而且可以做参数自动校验
	Call(ctx context.Context, input json.RawMessage) (any, error) // 调用接口
}
