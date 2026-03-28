package agent

import (
	"aATA/internal/llm"
	"encoding/json"
	"fmt"
	"sort"
)

func buildInitialMessages(input AgentInput) []llm.Message {
	inputJSON, _ := json.MarshalIndent(input, "", "  ")

	return []llm.Message{
		{
			Role:    "system",
			Content: systemPrompt(),
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("当前任务输入如下：\n%s", string(inputJSON)),
		},
	}
}

func systemPrompt() string {
	return `
你是 XCPC 集训队训练分析智能体。

你可以通过系统提供的 tools 获取训练数据、比赛数据和排行榜数据。
当信息不足时，优先调用合适的 tool；当信息已经足够时，直接给出最终结论。

要求：
1. 仅使用系统已经提供的 tools，不要虚构工具。
2. 可以多轮调用 tool，直到证据足够。
3. 完成任务时，assistant content 必须直接输出一个合法 JSON 对象，不要输出 Markdown，不要输出解释性前后缀。
4. 最终 JSON 结构必须是：
{
  "decision_type": string,
  "focus_students": string[],
  "confidence": number,
  "report": string,
  "metrics": object
}
5. 即使没有聚焦学生，也要输出 "focus_students": []。
6. 即使没有额外指标，也要输出 "metrics": {}。
7. report 写完整自然语言分析；confidence 取值 0 到 1。
`
}

func buildToolDefinitions(registry *Registry) []llm.ToolDefinition {
	tools := registry.List()
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name() < tools[j].Name()
	})
	defs := make([]llm.ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		defs = append(defs, llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  buildToolParameters(tool.Schema()),
			},
		})
	}
	return defs
}

func buildToolParameters(schema ToolSchema) map[string]any {
	properties := make(map[string]any, len(schema.Parameters))
	for name, param := range schema.Parameters {
		property := map[string]any{
			"type":        param.Type,
			"description": param.Description,
		}
		if len(param.Enum) > 0 {
			property["enum"] = param.Enum
		}
		properties[name] = property
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             schema.Required,
		"additionalProperties": false,
	}
}
