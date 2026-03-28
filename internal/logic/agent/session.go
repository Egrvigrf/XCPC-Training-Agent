package agent

import "fmt"

const maxSessionItems = 8

func newSessionSnapshot(input AgentInput) SessionSnapshot {
	return SessionSnapshot{
		Goal:           input.Query,
		ConfirmedFacts: []string{},
		DoneItems:      []string{},
		TodoItems:      []string{},
		Artifacts:      []string{},
	}
}

func (s *SessionSnapshot) recordToolResult(toolName string, success bool) {
	if toolName == "" {
		return
	}

	if success {
		s.DoneItems = appendLimited(s.DoneItems, fmt.Sprintf("已调用工具 %s", toolName))
		s.ConfirmedFacts = appendLimited(s.ConfirmedFacts, fmt.Sprintf("工具 %s 已返回可用数据", toolName))
		s.Artifacts = appendLimited(s.Artifacts, "tool:"+toolName)
		return
	}

	s.TodoItems = appendLimited(s.TodoItems, fmt.Sprintf("需要重新评估工具 %s 的调用结果", toolName))
	// 失败信息不进入长期事实，只保留在当前待办。
}

func appendLimited(items []string, value string) []string {
	if value == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}

	items = append(items, value)
	if len(items) <= maxSessionItems {
		return items
	}
	return append([]string(nil), items[len(items)-maxSessionItems:]...)
}
