/*
工具白名单注册与调用：

Register(tool Tool)
List() []Tool（给 prompt.go 列工具）
Call(ctx, toolName, args)（controller 只能通过 registry 调工具）
*/

package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	if _, exists := r.tools[t.Name()]; exists {
		panic("duplicate tool: " + t.Name())
	}
	r.tools[t.Name()] = t
}

func (r *Registry) List() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

func (r *Registry) Call(ctx context.Context, name string, raw json.RawMessage) (any, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("未知工具：%s", name)
	}
	return tool.Call(ctx, raw)
}
