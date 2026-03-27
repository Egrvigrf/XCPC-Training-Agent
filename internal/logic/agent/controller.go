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
	observer := newRunObserver(c.Trace, c.LLM, input, c.Registry.List())

	for state.Step = 0; state.Step < 10; state.Step++ {
		prompt := buildPrompt(state, c.Registry)

		resp, err := c.completeStep(ctx, &state, observer, prompt)
		if err != nil {
			return nil, c.Trace.Result(), err
		}

		switch resp.Action {
		case "call_tool":
			if err := c.runTool(ctx, &state, observer, resp); err != nil {
				continue
			}
			continue
		case "finish":
			final, err := finishOutput(resp)
			if err != nil {
				observer.runFailed(state, observer.lastEventID, "finish_validate", err, 0, nil)
				return nil, c.Trace.Result(), err
			}
			observer.runFinished(state, final, resp.FinalOutput.FocusStudents)
			return final, c.Trace.Result(), nil
		default:
			err := errors.New("模型返回了不支持的动作")
			observer.runFailed(state, observer.lastEventID, "validate_action", err, len(prompt), map[string]any{
				"action":  resp.Action,
				"summary": "运行失败：模型返回了不支持的动作",
			})
			return nil, c.Trace.Result(), err
		}
	}

	err := errors.New("执行步数超过上限")
	observer.runFailed(state, observer.lastEventID, "loop_guard", err, 0, map[string]any{
		"summary": "运行失败：执行步数超过上限",
	})
	return nil, c.Trace.Result(), err
}

func (c *Controller) completeStep(ctx context.Context, state *AgentState, observer *runObserver, prompt string) (*LLMResponse, error) {
	attempt := observer.startModel(*state, prompt, false)
	completion, err := c.LLM.Complete(ctx, prompt)
	if err != nil {
		observer.failModelCall(*state, attempt, err)
		return nil, err
	}

	resp, parseErr := parseLLMResponse(completion.Content)
	observer.recordModelReturn(*state, attempt, completion, resp, parseErr)
	if parseErr == nil {
		return resp, nil
	}

	repairPrompt := prompt + "\nYour previous output was invalid JSON. Please output strictly valid JSON only."
	repairAttempt := observer.startModel(*state, repairPrompt, true)
	repairCompletion, err := c.LLM.Complete(ctx, repairPrompt)
	if err != nil {
		observer.failModelCall(*state, repairAttempt, err)
		return nil, err
	}

	resp, parseErr = parseLLMResponse(repairCompletion.Content)
	observer.recordModelReturn(*state, repairAttempt, repairCompletion, resp, parseErr)
	if parseErr != nil {
		finalErr := errors.New("模型连续两次返回了非法 JSON")
		observer.runFailed(*state, observer.lastEventID, "model_returned", finalErr, len(repairPrompt), map[string]any{
			"summary": "运行失败：模型连续两次返回了非法 JSON",
		})
		return nil, finalErr
	}

	return resp, nil
}

func (c *Controller) runTool(ctx context.Context, state *AgentState, observer *runObserver, resp *LLMResponse) error {
	attempt := observer.startTool(*state, resp.ToolName, resp.Arguments)

	var (
		result  any
		callErr error
	)
	latencyMs, _ := measureLatency(func() error {
		rawArgs, _ := json.Marshal(resp.Arguments)
		result, callErr = c.Registry.Call(ctx, resp.ToolName, rawArgs)
		return callErr
	})

	if callErr != nil {
		observer.recordToolReturn(*state, attempt, latencyMs, nil, callErr)
		state.ToolResults = append(state.ToolResults, ToolResult{
			ToolName: resp.ToolName,
			Result:   map[string]string{"error": callErr.Error()},
		})
		return callErr
	}

	state.ToolResults = append(state.ToolResults, ToolResult{
		ToolName: resp.ToolName,
		Result:   result,
	})
	observer.recordToolReturn(*state, attempt, latencyMs, result, nil)
	return nil
}

func parseLLMResponse(raw string) (*LLMResponse, error) {
	var resp LLMResponse
	cleaned := cleanLLMOutput(raw)
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func finishOutput(resp *LLMResponse) (map[string]interface{}, error) {
	if resp == nil || resp.FinalOutput == nil {
		return nil, errors.New("缺少 final_output")
	}
	if resp.FinalOutput.DecisionType == "" {
		return nil, errors.New("缺少 decision_type")
	}
	if resp.FinalOutput.Report == "" {
		return nil, errors.New("缺少 report")
	}
	return structToMap(resp.FinalOutput), nil
}

func cleanLLMOutput(raw string) string {
	raw = strings.TrimSpace(raw)

	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 3 {
			raw = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}
	return raw
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
