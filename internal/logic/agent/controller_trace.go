// 将运行过程 -> trace 结构的映射抽象为一个独立层
// collector 和 observer 分层清晰，trace schema 不再分散于系统各处而是集中在 observer
// 因果链维护集中，其它组件无需维护因果链
// runObserver 不是业务逻辑本身，只是把业务运行过程翻译为观测事件

package agent

import (
	"aATA/internal/llm"
	"aATA/internal/logic/agenttrace"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type runObserver struct {
	trace       agenttrace.Sink
	modelName   string
	lastEventID string
	lastSpanID  string
}

type modelAttempt struct {
	spanID  string
	eventID string
	prompt  string
	repair  bool
}

type toolAttempt struct {
	spanID   string
	eventID  string
	toolName string
}

func newRunObserver(trace agenttrace.Sink, llmClient llm.Client, input AgentInput, tools []Tool) *runObserver {
	observer := &runObserver{
		trace:     trace,
		modelName: modelName(llmClient),
	}

	runStartedID := observer.trace.Record(0, agenttrace.EventRunStarted, "", map[string]any{
		"query":  input.Query,
		"params": input.Params,
	})

	toolNames := make([]string, 0, len(tools))
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name())
	}
	sort.Strings(toolNames)

	observer.lastEventID = observer.trace.Record(0, agenttrace.EventToolsRegistered, runStartedID, map[string]any{
		"status":      "success",
		"entity_type": "registry",
		"entity_name": "agent_tools",
		"summary":     fmt.Sprintf("已注册 %d 个工具", len(toolNames)),
		"tool_names":  toolNames,
		"tool_count":  len(toolNames),
	})

	return observer
}

// startModel 模型调用开始
func (o *runObserver) startModel(state AgentState, prompt string, repair bool) modelAttempt {
	summary := "开始调用模型"
	if repair {
		summary = "开始修复性调用模型"
	}

	spanID := o.trace.StartSpan(state.Step, agenttrace.SpanModelCall, o.lastSpanID, map[string]any{
		"entity_type":    "model",
		"entity_name":    o.modelName,
		"status":         "started",
		"summary":        summary,
		"state_snapshot": buildStateSnapshot(state, len(prompt), nil),
	})

	eventID := o.trace.Record(state.Step, agenttrace.EventModelCalled, o.lastEventID, map[string]any{
		"status":                  "started",
		"entity_type":             "model",
		"entity_name":             o.modelName,
		"summary":                 summary,
		"prompt":                  prompt,
		"prompt_length":           len(prompt),
		"history_tool_result_cnt": len(state.ToolResults),
		"repair_attempt":          repair,
		"state_snapshot":          buildStateSnapshot(state, len(prompt), nil),
	})

	return modelAttempt{
		spanID:  spanID,
		eventID: eventID,
		prompt:  prompt,
		repair:  repair,
	}
}

// failModelCall 模型调用失败
func (o *runObserver) failModelCall(state AgentState, attempt modelAttempt, err error) {
	o.trace.FinishSpan(attempt.spanID, "error", map[string]any{
		"entity_type": "model",
		"entity_name": o.modelName,
		"error":       err.Error(),
		"summary":     "模型调用失败",
	})
	o.runFailed(state, attempt.eventID, "model_called", err, len(attempt.prompt), nil)
}

// recordModelReturn 模型调用返回
func (o *runObserver) recordModelReturn(state AgentState, attempt modelAttempt, completion *llm.Completion, resp *LLMResponse, parseErr error) {
	parseOK := parseErr == nil
	summary := "模型返回了非法 JSON"
	if parseOK {
		summary = buildModelReturnSummary(resp, attempt.repair)
	} else if attempt.repair {
		summary = "修复性模型调用仍然返回了非法 JSON"
	}

	o.trace.FinishSpan(attempt.spanID, "success", map[string]any{
		"entity_type":   "model",
		"entity_name":   o.modelName,
		"summary":       summary,
		"latency_ms":    completion.LatencyMs,
		"finish_reason": completion.FinishReason,
		"parse_ok":      parseOK,
		"input_tokens":  completion.Usage.PromptTokens,
		"output_tokens": completion.Usage.CompletionTokens,
		"total_tokens":  completion.Usage.TotalTokens,
	})

	cleaned := cleanLLMOutput(completion.Content)
	payload := map[string]any{
		"status":         "success",
		"entity_type":    "model",
		"entity_name":    o.modelName,
		"summary":        summary,
		"raw":            completion.Content,
		"cleaned":        cleaned,
		"raw_response":   completion.RawResponse,
		"parse_ok":       parseOK,
		"latency_ms":     completion.LatencyMs,
		"finish_reason":  completion.FinishReason,
		"input_tokens":   completion.Usage.PromptTokens,
		"output_tokens":  completion.Usage.CompletionTokens,
		"total_tokens":   completion.Usage.TotalTokens,
		"state_snapshot": buildStateSnapshot(state, len(attempt.prompt), nil),
		"repair_attempt": attempt.repair,
	}

	if parseErr != nil {
		payload["parse_error"] = parseErr.Error()
		payload["repairing"] = !attempt.repair
	} else {
		payload["response"] = llmResponseSummary(resp)
		payload["repaired"] = attempt.repair
	}

	o.lastSpanID = attempt.spanID
	o.lastEventID = o.trace.Record(state.Step, agenttrace.EventModelReturned, attempt.eventID, payload)
}

// startTool 工具调用开始
func (o *runObserver) startTool(state AgentState, toolName string, arguments map[string]any) toolAttempt {
	spanID := o.trace.StartSpan(state.Step, agenttrace.SpanToolCall, o.lastSpanID, map[string]any{
		"entity_type":    "tool",
		"entity_name":    toolName,
		"status":         "started",
		"summary":        "开始调用工具",
		"state_snapshot": buildStateSnapshot(state, 0, nil),
	})

	eventID := o.trace.Record(state.Step, agenttrace.EventToolCalled, o.lastEventID, map[string]any{
		"status":         "started",
		"entity_type":    "tool",
		"entity_name":    toolName,
		"summary":        "开始调用工具",
		"tool_name":      toolName,
		"arguments":      arguments,
		"state_snapshot": buildStateSnapshot(state, 0, nil),
	})

	return toolAttempt{
		spanID:   spanID,
		eventID:  eventID,
		toolName: toolName,
	}
}

// recordToolReturn 工具调用返回
func (o *runObserver) recordToolReturn(state AgentState, attempt toolAttempt, latencyMs int64, result any, err error) {
	status := "success"
	summary := "工具调用成功"
	resultSummary := buildToolResultSummary(result)
	payload := map[string]any{
		"status":         status,
		"entity_type":    "tool",
		"entity_name":    attempt.toolName,
		"summary":        summary,
		"tool_name":      attempt.toolName,
		"latency_ms":     latencyMs,
		"result_summary": resultSummary,
		"state_snapshot": buildStateSnapshot(state, 0, nil),
	}

	if err != nil {
		status = "error"
		summary = "工具调用失败"
		payload["status"] = status
		payload["summary"] = summary
		payload["error"] = err.Error()
		payload["error_code"] = "工具调用失败"
		resultSummary = map[string]any{"error": err.Error()}
		payload["result_summary"] = resultSummary
	} else {
		payload["result"] = result
	}

	o.trace.FinishSpan(attempt.spanID, status, map[string]any{
		"entity_type":    "tool",
		"entity_name":    attempt.toolName,
		"summary":        summary,
		"error":          errorString(err),
		"latency_ms":     latencyMs,
		"result_summary": resultSummary,
		"state_snapshot": buildStateSnapshot(state, 0, nil),
	})

	o.lastSpanID = attempt.spanID
	o.lastEventID = o.trace.Record(state.Step, agenttrace.EventToolReturned, attempt.eventID, payload)
}

// runFailed 运行时失败
func (o *runObserver) runFailed(state AgentState, parentEventID, stage string, err error, contextChars int, extra map[string]any) {
	payload := map[string]any{
		"status":         "error",
		"entity_type":    "run",
		"entity_name":    "agent_run",
		"stage":          stage,
		"error":          err.Error(),
		"summary":        fmt.Sprintf("运行失败，阶段：%s", stage),
		"state_snapshot": buildStateSnapshot(state, contextChars, nil),
	}
	for k, v := range extra {
		payload[k] = v
	}
	o.trace.Record(state.Step, agenttrace.EventRunFailed, parentEventID, payload)
}

// runFinished 运行时成功
func (o *runObserver) runFinished(state AgentState, final map[string]any, focusStudents []string) {
	o.trace.Record(state.Step, agenttrace.EventRunFinished, o.lastEventID, map[string]any{
		"status":         "success",
		"entity_type":    "run",
		"entity_name":    "agent_run",
		"summary":        "运行成功完成",
		"final_output":   final,
		"state_snapshot": buildStateSnapshot(state, 0, focusStudents),
	})
}

func modelName(client llm.Client) string {
	descriptor, ok := client.(llm.Descriptor)
	if !ok {
		return "未知模型"
	}
	return descriptor.ModelName()
}

func llmResponseSummary(resp *LLMResponse) map[string]any {
	if resp == nil {
		return map[string]any{}
	}

	summary := map[string]any{
		"action":    resp.Action,
		"tool_name": resp.ToolName,
		"reasoning": resp.Reasoning,
	}

	if resp.FinalOutput != nil {
		summary["final_output"] = map[string]any{
			"decision_type":      resp.FinalOutput.DecisionType,
			"focus_students_cnt": len(resp.FinalOutput.FocusStudents),
			"metrics_cnt":        len(resp.FinalOutput.Metrics),
			"report_length":      len(resp.FinalOutput.Report),
			"confidence":         resp.FinalOutput.Confidence,
		}
	}

	if len(resp.Arguments) > 0 {
		summary["arguments"] = resp.Arguments
	}

	return summary
}

func buildStateSnapshot(state AgentState, contextChars int, focusStudents []string) map[string]any {
	evidenceTypes := make([]string, 0, len(state.ToolResults))
	for _, tr := range state.ToolResults {
		evidenceTypes = append(evidenceTypes, tr.ToolName)
	}
	sort.Strings(evidenceTypes)

	snapshot := map[string]any{
		"step":               state.Step,
		"tool_results_count": len(state.ToolResults),
		"evidence_types":     evidenceTypes,
		"context_chars":      contextChars,
	}
	if len(focusStudents) > 0 {
		snapshot["focus_students"] = focusStudents
	}
	return snapshot
}

func buildToolResultSummary(result any) map[string]any {
	if result == nil {
		return map[string]any{
			"type": "nil",
		}
	}

	switch v := result.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return map[string]any{
			"type":      "object",
			"key_count": len(keys),
			"keys":      keys,
		}
	default:
		b, _ := json.Marshal(v)
		return map[string]any{
			"type":         fmt.Sprintf("%T", result),
			"result_chars": len(b),
		}
	}
}

func buildModelReturnSummary(resp *LLMResponse, repaired bool) string {
	if resp == nil {
		return "模型返回为空"
	}
	if resp.Action == "call_tool" {
		if repaired {
			return fmt.Sprintf("修复性模型调用决定调用工具 %s", resp.ToolName)
		}
		return fmt.Sprintf("模型决定调用工具 %s", resp.ToolName)
	}
	if repaired {
		return "修复性模型调用生成了最终答案"
	}
	return "模型生成了最终答案"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func measureLatency(fn func() error) (int64, error) {
	startedAt := time.Now()
	err := fn()
	return time.Since(startedAt).Milliseconds(), err
}
