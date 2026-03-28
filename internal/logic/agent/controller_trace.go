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
	"strings"
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
	req     llm.ChatRequest
}

type toolAttempt struct {
	spanID     string
	eventID    string
	toolName   string
	toolCallID string
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

func (o *runObserver) startModel(state AgentState, req llm.ChatRequest) modelAttempt {
	snapshot := buildStateSnapshot(state, buildRequestContextSize(req), nil)

	spanID := o.trace.StartSpan(state.Step, agenttrace.SpanModelCall, o.lastSpanID, map[string]any{
		"entity_type":    "model",
		"entity_name":    o.modelName,
		"status":         "started",
		"summary":        "开始调用模型",
		"state_snapshot": snapshot,
	})

	eventID := o.trace.Record(state.Step, agenttrace.EventModelCalled, o.lastEventID, map[string]any{
		"status":                  "started",
		"entity_type":             "model",
		"entity_name":             o.modelName,
		"summary":                 "开始调用模型",
		"messages":                req.Messages,
		"message_count":           len(req.Messages),
		"tool_count":              len(req.Tools),
		"history_tool_result_cnt": len(state.ToolResults),
		"state_snapshot":          snapshot,
	})

	return modelAttempt{
		spanID:  spanID,
		eventID: eventID,
		req:     req,
	}
}

func (o *runObserver) failModelCall(state AgentState, attempt modelAttempt, err error) {
	o.trace.FinishSpan(attempt.spanID, "error", map[string]any{
		"entity_type": "model",
		"entity_name": o.modelName,
		"error":       err.Error(),
		"summary":     "模型调用失败",
	})
	o.runFailed(state, attempt.eventID, "model_called", err, map[string]any{
		"message_count": len(attempt.req.Messages),
		"tool_count":    len(attempt.req.Tools),
	})
}

func (o *runObserver) recordModelReturn(state AgentState, attempt modelAttempt, completion *llm.ChatCompletion, parseErr error) {
	parseOK := parseErr == nil
	summary := buildModelReturnSummary(completion, parseErr)

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

	payload := map[string]any{
		"status":         "success",
		"entity_type":    "model",
		"entity_name":    o.modelName,
		"summary":        summary,
		"content":        completion.Content,
		"tool_calls":     completion.ToolCalls,
		"raw_response":   completion.RawResponse,
		"parse_ok":       parseOK,
		"latency_ms":     completion.LatencyMs,
		"finish_reason":  completion.FinishReason,
		"input_tokens":   completion.Usage.PromptTokens,
		"output_tokens":  completion.Usage.CompletionTokens,
		"total_tokens":   completion.Usage.TotalTokens,
		"state_snapshot": buildStateSnapshot(state, buildRequestContextSize(attempt.req), nil),
	}

	if parseErr != nil {
		payload["parse_error"] = parseErr.Error()
	} else {
		payload["response"] = chatCompletionSummary(completion)
	}

	o.lastSpanID = attempt.spanID
	o.lastEventID = o.trace.Record(state.Step, agenttrace.EventModelReturned, attempt.eventID, payload)
}

func (o *runObserver) startTool(state AgentState, toolName, arguments, toolCallID string) toolAttempt {
	argsJSON := tryDecodeJSON(arguments)

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
		"tool_call_id":   toolCallID,
		"arguments":      argsJSON,
		"arguments_raw":  arguments,
		"state_snapshot": buildStateSnapshot(state, 0, nil),
	})

	return toolAttempt{
		spanID:     spanID,
		eventID:    eventID,
		toolName:   toolName,
		toolCallID: toolCallID,
	}
}

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
		"tool_call_id":   attempt.toolCallID,
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

func (o *runObserver) runFailed(state AgentState, parentEventID, stage string, err error, extra map[string]any) {
	payload := map[string]any{
		"status":         "error",
		"entity_type":    "run",
		"entity_name":    "agent_run",
		"stage":          stage,
		"error":          err.Error(),
		"summary":        fmt.Sprintf("运行失败，阶段：%s", stage),
		"state_snapshot": buildStateSnapshot(state, 0, nil),
	}
	for k, v := range extra {
		payload[k] = v
	}
	o.trace.Record(state.Step, agenttrace.EventRunFailed, parentEventID, payload)
}

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

func chatCompletionSummary(resp *llm.ChatCompletion) map[string]any {
	if resp == nil {
		return map[string]any{}
	}

	summary := map[string]any{
		"finish_reason":  resp.FinishReason,
		"content_length": len(resp.Content),
	}

	if len(resp.ToolCalls) > 0 {
		calls := make([]map[string]any, 0, len(resp.ToolCalls))
		for _, call := range resp.ToolCalls {
			calls = append(calls, map[string]any{
				"id":               call.ID,
				"type":             call.Type,
				"name":             call.Function.Name,
				"arguments_length": len(call.Function.Arguments),
			})
		}
		summary["tool_calls"] = calls
		return summary
	}

	if final, err := parseFinalOutput(resp.Content); err == nil {
		summary["final_output"] = map[string]any{
			"decision_type":      final.DecisionType,
			"focus_students_cnt": len(final.FocusStudents),
			"metrics_cnt":        len(final.Metrics),
			"report_length":      len(final.Report),
			"confidence":         final.Confidence,
		}
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

func buildModelReturnSummary(resp *llm.ChatCompletion, parseErr error) string {
	if resp == nil {
		return "模型返回为空"
	}
	if len(resp.ToolCalls) > 0 {
		if len(resp.ToolCalls) == 1 {
			return fmt.Sprintf("模型决定调用工具 %s", resp.ToolCalls[0].Function.Name)
		}
		return fmt.Sprintf("模型决定并行调用 %d 个工具", len(resp.ToolCalls))
	}
	if parseErr != nil {
		return "模型返回了无法解析的最终答案"
	}
	return "模型生成了最终答案"
}

func buildRequestContextSize(req llm.ChatRequest) int {
	size := 0
	for _, msg := range req.Messages {
		size += len(msg.Content)
		for _, call := range msg.ToolCalls {
			size += len(call.Function.Name)
			size += len(call.Function.Arguments)
		}
	}
	return size
}

func tryDecodeJSON(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}

	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	return v
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
