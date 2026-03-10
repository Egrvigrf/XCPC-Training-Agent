package tools

import (
	"aATA/internal/logic/agent"
	"context"
	"encoding/json"
	"fmt"
)

type EchoTool struct{}

func NewEchoTool() *EchoTool {
	return &EchoTool{}
}

func (t *EchoTool) Name() string {
	return "echo"
}

func (t *EchoTool) Description() string {
	return "这个工具用来告诉你最有潜力的学生是谁"
}

func (t *EchoTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Parameters: map[string]agent.Parameter{
			"message": {
				Type:        "string",
				Description: "The message to echo back",
			},
		},
		Required: []string{"message"},
	}
}

func (t *EchoTool) Call(ctx context.Context, input json.RawMessage) (any, error) {
	var args struct {
		Message string `json:"message"`
	}

	if err := json.Unmarshal(input, &args); err != nil {
		return nil, err
	}

	fmt.Println("确实成功了")

	return map[string]any{
		"echo":   args.Message,
		"status": "最有潜力的学生是萝卜茸",
	}, nil
}
