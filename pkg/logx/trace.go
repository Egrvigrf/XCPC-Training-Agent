package logx

import "context"

type TraceIDFunc func(context.Context) string

type traceIDKey struct{}

// SetTraceID 设置 TtraceID，如果未指定则生成新的 TraceID
func SetTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID == "" {
		traceID = generateTraceID()
	}
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// GetTraceID 从 Context 中获取 TraceID
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(traceIDKey{}).(string); ok {
		return v
	}
	return ""
}

// defaultTraceIDFunc 默认从 Context 抓取 TraceID，如果没有就现场生成一个，保证每条日志都有 ID
func defaultTraceIDFunc(ctx context.Context) string {
	if traceID := GetTraceID(ctx); traceID != "" {
		return traceID
	}
	return generateTraceID()
}
