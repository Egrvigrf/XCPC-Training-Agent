package agenttrace

type NoopSink struct{}

func (NoopSink) Record(step int, eventType EventType, parentID string, payload any) string {
	return ""
}

func (NoopSink) StartSpan(step int, spanType SpanType, parentSpanID string, payload any) string {
	return ""
}

func (NoopSink) FinishSpan(spanID, status string, payload any) {}

func (NoopSink) Result() RunTrace {
	return RunTrace{
		Mode:   ModeSummary,
		Spans:  []Span{},
		Events: []Event{},
	}
}
