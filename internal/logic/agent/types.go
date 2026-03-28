/*
定义整个 agent 系统的统一数据结构：

AgentInput：任务输入
AgentState：输入 + 已有工具结果 + step count
FinalOutput：Agent 最终输出
*/

package agent

type AgentInput struct {
	Query  string                 `json:"query"`
	Params map[string]interface{} `json:"params"`
}

type ToolResult struct {
	ToolName string
	Result   interface{}
}

type AgentState struct {
	Input       AgentInput
	ToolResults []ToolResult
	Step        int
}

type FinalOutput struct {
	DecisionType  string                 `json:"decision_type"`
	FocusStudents []string               `json:"focus_students"`
	Confidence    float64                `json:"confidence"`
	Report        string                 `json:"report"`
	Metrics       map[string]interface{} `json:"metrics"`
}
