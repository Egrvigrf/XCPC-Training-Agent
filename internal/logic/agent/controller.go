// agent 运行内核

package agent

import (
	"aATA/internal/llm"
	"aATA/internal/logic/agentmemory"
	"aATA/internal/logic/agenttrace"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
)

type Controller struct {
	LLM      llm.Client
	Registry *Registry
	Summary  ToolSummarizer
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
		Summary:  NewDefaultToolSummarizer(),
		Trace:    traceSink,
	}
}

func (c *Controller) Run(ctx context.Context, input AgentInput) (map[string]interface{}, agenttrace.RunTrace, error) {
	// 解析记忆路径，推导出本轮任务应该加载哪些 memory 文件，从全量注入变成了按任务定向注入
	resolvedPaths := input.MemoryPaths()
	// 加载记忆包，做本轮运行前的外部记忆检索与装配
	memoryBundle, err := agentmemory.NewLoader(os.Getenv("AGENT_MEMORY_DIR")).Load(resolvedPaths)
	if err != nil {
		return nil, c.Trace.Result(), err
	}

	// 初始化模型状态
	state := AgentState{
		Input: input,
		// 引入会话快照，上下文不再等同于全部历史消息，而是等于“基础消息+当前快照+局部对话窗口”
		// 这代表快照是给下一步决策使用的状态骨架
		Snapshot: newSessionSnapshot(input),
		// 把本轮匹配到的 memory 路径也存进状态里，state 不只是会话状态，还保存了一部分上下文信息
		ResolvedPaths: resolvedPaths,
		// 记忆审核，不仅加载 memory，还把本轮应该用哪些记忆条目记忆下来了
		AppliedMemories: appliedMemoryNames(memoryBundle),
	}
	tools := buildToolDefinitions(c.Registry)                            // 注册工具表
	baseMessages := buildBaseMessages(input, memoryBundle)               // 构造基础消息
	conversation := make([]llm.Message, 0, 16)                           // 初始化动态消息队列
	observer := newRunObserver(c.Trace, c.LLM, input, c.Registry.List()) // 初始化观测模块

	for state.Step = 0; state.Step < 10; state.Step++ {
		// 通过基础消息、执行快照、动态消息队列组装请求
		req := llm.ChatRequest{
			Messages: buildRequestMessages(baseMessages, state.Snapshot, conversation),
			Tools:    tools,
		}

		// 调用模型完成当前 step
		completion, err := c.completeStep(ctx, &state, observer, req)
		if err != nil {
			return nil, c.Trace.Result(), err
		}

		// 模型发起工具调用
		if len(completion.ToolCalls) > 0 {
			conversation = append(conversation, completion.Message)                    // 把决策放入动态消息队列
			c.runToolCalls(ctx, &state, observer, completion.ToolCalls, &conversation) // 执行工具并且返回调用结果
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

	err = errors.New("执行步数超过上限")
	observer.runFailed(state, observer.lastEventID, "loop_guard", err, map[string]any{
		"summary": "运行失败：执行步数超过上限",
	})
	return nil, c.Trace.Result(), err
}

func appliedMemoryNames(bundle agentmemory.Bundle) []string {
	names := make([]string, 0, len(bundle.Rules)+1)
	if bundle.Project != "" {
		names = append(names, "project")
	}
	for _, rule := range bundle.Rules {
		if rule.Name != "" {
			names = append(names, "rule:"+rule.Name)
		}
	}
	return names
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
		summary := c.Summary.Summarize(call.Function.Name, map[string]any{"error": callErr.Error()})
		observer.recordToolReturn(*state, attempt, latencyMs, nil, callErr)
		state.Snapshot.recordToolResult(call.Function.Name, false)
		state.ToolResults = append(state.ToolResults, ToolResult{
			ToolName: call.Function.Name,
			Result:   map[string]string{"error": callErr.Error()},
			Summary:  summary,
		})

		payload, _ := json.Marshal(summary)
		return llm.Message{
			Role:       "tool",
			ToolCallID: call.ID,
			Content:    string(payload),
		}, callErr
	}

	summary := c.Summary.Summarize(call.Function.Name, result)
	state.ToolResults = append(state.ToolResults, ToolResult{
		ToolName: call.Function.Name,
		Result:   result,
		Summary:  summary,
	})
	state.Snapshot.recordToolResult(call.Function.Name, true)
	observer.recordToolReturn(*state, attempt, latencyMs, result, nil)

	payload, _ := json.Marshal(summary)
	return llm.Message{
		Role:       "tool",
		ToolCallID: call.ID,
		Content:    string(payload),
	}, nil
}

func (in AgentInput) MemoryPaths() []string {
	if in.Params == nil {
		return nil
	}

	keys := []string{"memory_paths", "context_paths", "paths"}
	paths := make([]string, 0, 4)
	for _, key := range keys {
		raw, ok := in.Params[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case []string:
			for _, item := range v {
				item = strings.TrimSpace(item)
				if item != "" {
					paths = append(paths, item)
				}
			}
		case []any:
			for _, item := range v {
				text, ok := item.(string)
				if !ok {
					continue
				}
				text = strings.TrimSpace(text)
				if text != "" {
					paths = append(paths, text)
				}
			}
		case string:
			for _, item := range strings.Split(v, ",") {
				item = strings.TrimSpace(item)
				if item != "" {
					paths = append(paths, item)
				}
			}
		}
	}

	if len(paths) == 0 {
		return nil
	}
	return paths
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
