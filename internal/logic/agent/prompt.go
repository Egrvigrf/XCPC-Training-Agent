package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// buildPrompt 组装发送给 LLM 的话
func buildPrompt(state AgentState, registry *Registry) string {

	inputJSON, _ := json.MarshalIndent(state.Input, "", "  ")

	var toolResultText string
	for _, tr := range state.ToolResults {
		resultJSON, _ := json.MarshalIndent(tr.Result, "", "  ")
		toolResultText += "\n工具返回(" + tr.ToolName + "):\n" + string(resultJSON) + "\n"
	}

	// 自动生成工具说明
	var toolDescBuilder strings.Builder

	toolDescBuilder.WriteString("你只能使用以下工具：\n\n")

	for _, t := range registry.List() {

		schema := t.Schema()

		paramsJSON, _ := json.MarshalIndent(schema.Parameters, "", "  ")
		requiredJSON, _ := json.MarshalIndent(schema.Required, "", "  ")

		toolDescBuilder.WriteString(fmt.Sprintf(
			"Name: %s\nDescription: %s\nParameters:\n%s\nRequired:\n%s\n\n",
			t.Name(),
			t.Description(),
			string(paramsJSON),
			string(requiredJSON),
		))
	}

	return fmt.Sprintf(`
你是一个训练数据分析智能体

%s

禁止：
- 发明新工具
- 输出 reasoning_content
- 输出 markdown
- 编造数据

你必须严格输出 JSON，且格式如下：

{
  "action": "call_tool" 或 "finish",
  "tool_name": "...",
  "arguments": {...},
  "reasoning": "解释",
  "final_output": {
    "decision_type": "coach_attention",
    "focus_students": [],
    "confidence": 0.0
  }
}

规则：

如果调用工具：
- action="call_tool"
- final_output 必须是 {}

请你判断，如果你调度的工具已经满足需求，你就按下面的方式结束循环：
- action="finish"
- tool_name=""
- arguments 必须是 {}
- final_output 必须符合 schema

当前输入：
%s

历史工具结果：
%s
`,
		toolDescBuilder.String(),
		string(inputJSON),
		toolResultText,
	)
}
