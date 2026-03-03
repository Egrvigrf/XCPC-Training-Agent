package logx

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type ConsoleLogger struct{}

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}

// Log 将日志对象 Entry 序列化为 JSON 格式并且打印
func (c *ConsoleLogger) Log(entry Entry) {
	b, err := json.Marshal(entry)
	if err != nil {
		fmt.Println("log marshal error:", err)
		return
	}
	fmt.Println(string(b))
}

func mergeFields(fields ...Fields) Fields {
	m := make(Fields)
	for _, f := range fields {
		for k, v := range f {
			m[k] = v
		}
	}
	return m
}

func generateTraceID() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "trace-fallback"
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
