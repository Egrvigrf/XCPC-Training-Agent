// 定义了 Trace 对象是怎么被创建、维护、收集、裁剪并最终产出的
// 把观测数据的组织与生成从主流程中隔离出来，让其变为一个统一的旁路组件

package agenttrace

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Collector 一次 run 对应的 trace 收集上下文
type Collector struct {
	mode      Mode
	runID     string
	startedAt time.Time

	mu          sync.Mutex
	events      []Event          // Event 列表
	spans       []Span           // Span 列表
	activeSpans map[string]*Span // 正在运行中的 Span
}

var idSeq uint64

func NewCollector(mode Mode) *Collector {
	if mode != ModeDebug { // 默认 Summary 模式
		mode = ModeSummary
	}

	now := time.Now()
	return &Collector{
		mode:        mode,
		runID:       newID("run"),
		startedAt:   now,
		events:      make([]Event, 0, 16),
		spans:       make([]Span, 0, 8),
		activeSpans: make(map[string]*Span),
	}
}

// Record 是 Event 的打包入口，接受步骤、事件类型、父事件 id 和概要，打包完成后返回事件 id
func (c *Collector) Record(step int, eventType EventType, parentID string, payload any) string {
	eventID := newID("evt")
	event := Event{
		EventID:   eventID,
		RunID:     c.runID,
		ParentID:  parentID,
		Step:      step,
		EventType: eventType,
		Timestamp: time.Now(),
		Payload:   summarizePayload(c.mode, eventType, payload),
	}

	c.mu.Lock()
	c.events = append(c.events, event)
	c.mu.Unlock()

	return eventID // 返回事件 ID，将来可以建立因果链或给后续事件挂 ParentID
}

// StartSpan 是 Span 生命周期的开始
func (c *Collector) StartSpan(step int, spanType SpanType, parentSpanID string, payload any) string {
	spanID := newID("span")
	now := time.Now()

	span := &Span{
		SpanID:       spanID,
		RunID:        c.runID,
		ParentSpanID: parentSpanID,
		Step:         step,
		SpanType:     spanType,
		StartedAt:    now,
		Status:       "running", // Span 从一开始就有状态，而不是结束后才形成
		Payload:      summarizeSpanPayload(c.mode, payload),
	}

	c.mu.Lock()
	c.activeSpans[spanID] = span // 丢入运行中 Span
	c.mu.Unlock()

	return spanID
}

// FinishSpan 是 Span 生命周期的结束
func (c *Collector) FinishSpan(spanID, status string, payload any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 找到正在运行中的 span
	span, ok := c.activeSpans[spanID]
	if !ok {
		return
	}

	// 补充结束时间和耗时，防止空状态
	span.FinishedAt = time.Now()
	span.LatencyMs = span.FinishedAt.Sub(span.StartedAt).Milliseconds()
	if status == "" {
		status = "unknown"
	}
	span.Status = status

	// 合并 Payload，由三部分构成：start、finish 以及 collector 强制补上的字段
	span.Payload = mergeSummary(span.Payload, summarizeSpanPayload(c.mode, payload))
	span.Payload["status"] = status
	span.Payload["latency_ms"] = span.LatencyMs

	c.spans = append(c.spans, *span)
	delete(c.activeSpans, spanID) // 从 active 移入 finished
}

// Result 导出最终结果
func (c *Collector) Result() RunTrace {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 深拷贝，防止外部污染内部状态
	events := make([]Event, len(c.events))
	copy(events, c.events)

	spans := make([]Span, len(c.spans))
	copy(spans, c.spans)

	// 生成最终 RunTrace
	return RunTrace{
		RunID:      c.runID,
		Mode:       c.mode,
		StartedAt:  c.startedAt,
		FinishedAt: time.Now(),
		Spans:      spans,
		Events:     events,
	}
}

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), atomic.AddUint64(&idSeq, 1))
}

// 摘要生成流程：先把输入统一为 map，然后对值做压缩式保留，如果是 debug 模式再补充原文

func summarizePayload(mode Mode, eventType EventType, payload any) map[string]any {
	summary := normalizeEnvelope(payload)
	summary["event_kind"] = string(eventType)
	summary["storage_mode"] = string(mode)
	summary = summarizeMap(summary)

	if mode == ModeDebug {
		summary["debug"] = rawPayload(payload)
	}
	return summary
}

func summarizeSpanPayload(mode Mode, payload any) map[string]any {
	summary := summarizeMap(normalizeEnvelope(payload))
	summary["storage_mode"] = string(mode)

	if mode == ModeDebug {
		summary["debug"] = rawPayload(payload)
	}
	return summary
}

func normalizeEnvelope(payload any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if m, ok := payload.(map[string]any); ok {
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	return map[string]any{
		"summary": payload,
	}
}

func summarizeMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for k, v := range input {
		switch k {
		case "status", "latency_ms", "error", "error_code", "entity_type", "entity_name", "finish_reason", "parse_ok":
			output[k] = v
		case "summary":
			output[k] = summarizeValue(v)
		default:
			output[k] = summarizeValue(v)
		}
	}
	return output
}

func mergeSummary(base, extra map[string]any) map[string]any {
	if len(base) == 0 {
		return extra
	}
	merged := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}

func summarizeValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		return map[string]any{
			"type":    "string",
			"length":  len(x),
			"preview": preview(x, 200),
		}
	case []string:
		return map[string]any{
			"type":    "string_array",
			"count":   len(x),
			"preview": x,
		}
	case []any:
		return map[string]any{
			"type":  "array",
			"count": len(x),
		}
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return map[string]any{
			"type":         "object",
			"key_count":    len(keys),
			"keys":         keys,
			"json_length":  jsonLen(x),
			"json_preview": previewJSON(x, 240),
		}
	case error:
		return map[string]any{
			"type":    "error",
			"message": x.Error(),
		}
	case bool, int, int64, float64:
		return x
	default:
		return map[string]any{
			"type":         fmt.Sprintf("%T", v),
			"json_length":  jsonLen(v),
			"json_preview": previewJSON(v, 240),
		}
	}
}

func rawPayload(v any) any {
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok {
		return s
	}

	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return json.RawMessage(b)
}

func jsonLen(v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		return len(fmt.Sprintf("%v", v))
	}
	return len(b)
}

func previewJSON(v any, limit int) string {
	b, err := json.Marshal(v)
	if err != nil {
		return preview(fmt.Sprintf("%v", v), limit)
	}
	return preview(string(b), limit)
}

func preview(s string, limit int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
