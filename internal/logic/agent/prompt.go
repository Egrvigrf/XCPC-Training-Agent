package agent

import (
	"aATA/internal/llm"
	"aATA/internal/logic/agentmemory"
	"encoding/json"
	"fmt"
	"sort"
)

const recentConversationLimit = 6

func buildBaseMessages(input AgentInput, bundle agentmemory.Bundle) []llm.Message {
	messages := []llm.Message{
		{
			Role:    "system",
			Content: systemPrompt(),
		},
	}

	if bundle.Project != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: "Project Memory:\n" + bundle.Project,
		})
	}

	for _, rule := range bundle.Rules {
		if rule.Content == "" {
			continue
		}
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: fmt.Sprintf("Path Rule (%s):\n%s", rule.Name, rule.Content),
		})
	}

	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: fmt.Sprintf("当前任务输入如下：\n%s", string(inputJSON)),
	})

	return messages
}

func buildRequestMessages(baseMessages []llm.Message, snapshot SessionSnapshot, conversation []llm.Message) []llm.Message {
	out := make([]llm.Message, 0, len(baseMessages)+1+recentConversationLimit)
	if len(baseMessages) == 0 {
		return out
	}

	out = append(out, baseMessages[:len(baseMessages)-1]...)
	out = append(out, llm.Message{
		Role:    "system",
		Content: buildSnapshotMessage(snapshot),
	})
	out = append(out, baseMessages[len(baseMessages)-1])
	out = append(out, recentConversation(conversation)...)
	return out
}

func systemPrompt() string {
	return `
你是 XCPC 集训队训练分析智能体。

规则：
1. 当信息不足时，优先调用已提供的工具获取数据。
2. 不要虚构工具、数据或结论。
3. 当证据充分时，直接输出最终结果。
4. 最终输出必须是一个合法 JSON 对象，不要输出 Markdown，不要输出额外说明。

最终 JSON 结构：
{
  "decision_type": string,
  "focus_students": string[],
  "confidence": number,
  "report": string,
  "metrics": object
}
`
}

func buildSnapshotMessage(snapshot SessionSnapshot) string {
	body, _ := json.MarshalIndent(snapshot, "", "  ")
	return "Session Snapshot:\n" + string(body)
}

func recentConversation(messages []llm.Message) []llm.Message {
	if len(messages) <= recentConversationLimit {
		return append([]llm.Message(nil), messages...)
	}
	return append([]llm.Message(nil), messages[len(messages)-recentConversationLimit:]...)
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
