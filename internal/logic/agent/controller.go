// agent 运行内核

package agent

import (
	"aATA/internal/llm"
	"aATA/internal/logic/agenttrace"
	"context"
	"encoding/json"
	"errors"
	"strings"
)

type Controller struct {
	LLM      llm.Client
	Registry *Registry
	Trace    agenttrace.Sink
}

func NewController(
	llmClient llm.Client,
	registry *Registry,
	traceSink agenttrace.Sink,
) *Controller {
	if traceSink == nil {
		traceSink = agenttrace.NoopSink{}
	}

	return &Controller{
		LLM:      llmClient,
		Registry: registry,
		Trace:    traceSink,
	}
}

func (c *Controller) Run(ctx context.Context, input AgentInput) (map[string]interface{}, agenttrace.RunTrace, error) {
	state := AgentState{Input: input}
	tools := buildToolDefinitions(c.Registry)
	messages := buildInitialMessages(input)
	observer := newRunObserver(c.Trace, c.LLM, input, c.Registry.List())

	for state.Step = 0; state.Step < 10; state.Step++ {
		req := llm.ChatRequest{
			Messages: messages,
			Tools:    tools,
		}

		completion, err := c.completeStep(ctx, &state, observer, req)
		if err != nil {
			return nil, c.Trace.Result(), err
		}

		if len(completion.ToolCalls) > 0 {
			messages = append(messages, completion.Message)
			c.runToolCalls(ctx, &state, observer, completion.ToolCalls, &messages)
			continue
		}

		final, err := finishOutput(completion.Content)
		if err != nil {
			observer.runFailed(state, observer.lastEventID, "finish_validate", err, map[string]any{
				"summary": "运行失败：模型最终输出不是合法 JSON",
				"content": completion.Content,
			})
			return nil, c.Trace.Result(), err
		}

		observer.runFinished(state, final, finalStringSlice(final["focus_students"]))
		return final, c.Trace.Result(), nil
	}

	err := errors.New("执行步数超过上限")
	observer.runFailed(state, observer.lastEventID, "loop_guard", err, map[string]any{
		"summary": "运行失败：执行步数超过上限",
	})
	return nil, c.Trace.Result(), err
}

func (c *Controller) completeStep(ctx context.Context, state *AgentState, observer *runObserver, req llm.ChatRequest) (*llm.ChatCompletion, error) {
	attempt := observer.startModel(*state, req)
	completion, err := c.LLM.Chat(ctx, req)
	if err != nil {
		observer.failModelCall(*state, attempt, err)
		return nil, err
	}

	var parseErr error
	if len(completion.ToolCalls) == 0 {
		_, parseErr = parseFinalOutput(completion.Content)
	}

	observer.recordModelReturn(*state, attempt, completion, parseErr)
	return completion, nil
}

func (c *Controller) runToolCalls(ctx context.Context, state *AgentState, observer *runObserver, calls []llm.ToolCall, messages *[]llm.Message) {
	for _, call := range calls {
		toolMessage, _ := c.runToolCall(ctx, state, observer, call)
		*messages = append(*messages, toolMessage)
	}
}

func (c *Controller) runToolCall(ctx context.Context, state *AgentState, observer *runObserver, call llm.ToolCall) (llm.Message, error) {
	attempt := observer.startTool(*state, call.Function.Name, call.Function.Arguments, call.ID)

	var (
		result  any
		callErr error
	)
	latencyMs, _ := measureLatency(func() error {
		rawArgs := []byte(strings.TrimSpace(call.Function.Arguments))
		if len(rawArgs) == 0 {
			rawArgs = []byte("{}")
		}
		result, callErr = c.Registry.Call(ctx, call.Function.Name, rawArgs)
		return callErr
	})

	if callErr != nil {
		observer.recordToolReturn(*state, attempt, latencyMs, nil, callErr)
		state.ToolResults = append(state.ToolResults, ToolResult{
			ToolName: call.Function.Name,
			Result:   map[string]string{"error": callErr.Error()},
		})

		payload, _ := json.Marshal(map[string]string{"error": callErr.Error()})
		return llm.Message{
			Role:       "tool",
			ToolCallID: call.ID,
			Content:    string(payload),
		}, callErr
	}

	state.ToolResults = append(state.ToolResults, ToolResult{
		ToolName: call.Function.Name,
		Result:   result,
	})
	observer.recordToolReturn(*state, attempt, latencyMs, result, nil)

	payload, _ := json.Marshal(result)
	return llm.Message{
		Role:       "tool",
		ToolCallID: call.ID,
		Content:    string(payload),
	}, nil
}

func finishOutput(raw string) (map[string]interface{}, error) {
	final, err := parseFinalOutput(raw)
	if err != nil {
		return nil, err
	}
	if final.DecisionType == "" {
		return nil, errors.New("缺少 decision_type")
	}
	if final.Report == "" {
		return nil, errors.New("缺少 report")
	}
	if final.FocusStudents == nil {
		final.FocusStudents = []string{}
	}
	if final.Metrics == nil {
		final.Metrics = map[string]interface{}{}
	}
	return structToMap(&final), nil
}

func parseFinalOutput(raw string) (FinalOutput, error) {
	var final FinalOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &final); err != nil {
		return FinalOutput{}, err
	}
	return final, nil
}

func finalStringSlice(v any) []string {
	values, ok := v.([]string)
	if ok {
		return values
	}
	return nil
}

func structToMap(f *FinalOutput) map[string]interface{} {
	return map[string]interface{}{
		"decision_type":  f.DecisionType,
		"focus_students": f.FocusStudents,
		"confidence":     f.Confidence,
		"report":         f.Report,
		"metrics":        f.Metrics,
	}
}
