package logx

type Logger interface {
	Log(entry Entry)
}

// SetOutput 允许在初始化时更换输出端
func SetOutput(logger Logger) {
	if logger != nil {
		defaultLogger.output = logger
	}
}

func SetTraceIDFunc(fn TraceIDFunc) {
	if fn != nil {
		defaultLogger.traceIDFunc = fn
	}
}
