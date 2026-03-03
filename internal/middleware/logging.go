package middleware

import (
	"aATA/pkg/logx"
	"time"

	"github.com/gin-gonic/gin"
)

type LoggingMid struct{}

func NewLoggingMid() *LoggingMid {
	return &LoggingMid{}
}

func (m *LoggingMid) Handler(ctx *gin.Context) {
	// 1. 注入 TraceID
	reqCtx := ctx.Request.Context()
	reqCtx = logx.SetTraceID(reqCtx, "") // 从 logx 工具库中生成关于这条记录的 TraceID
	ctx.Request = ctx.Request.WithContext(reqCtx)

	start := time.Now() // 记录请求时间

	// 2. 放行请求
	ctx.Next()

	// 3. 记录访问日志
	logx.Infos(
		ctx.Request.Context(),
		"",
		"http_request",
		logx.Fields{
			"log_type": "access",
			"method":   ctx.Request.Method,
			"path":     ctx.FullPath(),
			"status":   ctx.Writer.Status(),
			"latency":  time.Since(start).Milliseconds(),
			"ip":       ctx.ClientIP(),
		},
	)
}
