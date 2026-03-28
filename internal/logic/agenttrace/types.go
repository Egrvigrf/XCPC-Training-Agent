package agenttrace

import "time"

type Mode string

const (
	ModeSummary Mode = "summary" // 摘要
	ModeDebug   Mode = "debug"   // 原文
)

type EventType string
type SpanType string

const (
	EventRunStarted      EventType = "run_started"      // 任务开始
	EventToolsRegistered EventType = "tools_registered" // 工具注册完成
	EventModelCalled     EventType = "model_called"     //  模型调用
	EventModelReturned   EventType = "model_returned"   // 模型返回
	EventToolCalled      EventType = "tool_called"      // 工具调用
	EventToolReturned    EventType = "tool_returned"    // 工具返回
	EventRunFinished     EventType = "run_finished"     // 任务完成
	EventRunFailed       EventType = "run_failed"       // 任务失败
)

const (
	SpanModelCall SpanType = "model_call" // 模型调用阶段
	SpanToolCall  SpanType = "tool_call"  // 工具调用阶段
)

// RunTrace 一次完整性任务
type RunTrace struct {
	RunID      string    `json:"run_id"`
	Mode       Mode      `json:"mode"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Spans      []Span    `json:"spans"`
	Events     []Event   `json:"events"`
}

// Event 阶段里的具体事件（瞬时事实）
type Event struct {
	EventID   string         `json:"event_id"`
	RunID     string         `json:"run_id"`
	ParentID  string         `json:"parent_id,omitempty"`
	Step      int            `json:"step"`
	EventType EventType      `json:"event_type"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

// Span 任务中的一个阶段（摘要）
type Span struct {
	SpanID       string         `json:"span_id"`
	RunID        string         `json:"run_id"`
	ParentSpanID string         `json:"parent_span_id,omitempty"` // 父节点 id，未来可以支持子 span
	Step         int            `json:"step"`
	SpanType     SpanType       `json:"span_type"`
	StartedAt    time.Time      `json:"started_at"`
	FinishedAt   time.Time      `json:"finished_at"`
	Status       string         `json:"status"`
	LatencyMs    int64          `json:"latency_ms"`
	Payload      map[string]any `json:"payload"`
}

type Sink interface {
	// Record 记录一个 Event，返回事件 id
	Record(step int, eventType EventType, parentID string, payload any) string
	// StartSpan 开始一个阶段，返回 Span id
	StartSpan(step int, spanType SpanType, parentSpanID string, payload any) string
	// FinishSpan 结束一个阶段，返回 Span id
	FinishSpan(spanID, status string, payload any)
	// Result 获取完整 RunTrace
	Result() RunTrace
}
