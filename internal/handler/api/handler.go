package api

import (
	"context"

	"github.com/gin-gonic/gin"

	"aATA/internal/handler"
	"aATA/internal/svc"
	"aATA/pkg/httpx"
)

type Handler interface {
	InitRegister(*gin.Engine)
}

type handle struct {
	srv  *gin.Engine
	addr string
}

// NewHandle 初始化 API，配置 HTTP 服务器实例并且完成业务路由的注册
func NewHandle(svc *svc.ServiceContext) *handle {
	h := &handle{
		srv:  gin.Default(),
		addr: "0.0.0.0:8080",
	}
	if len(svc.Config.Addr) > 0 { // 如果外部传进来地址了，就不使用默认地址
		h.addr = svc.Config.Addr
	}

	httpx.SetErrorHandler(handler.ErrorHandler) // 初始化错误管理

	// 注册路由
	handlers := initHandler(svc)
	for _, handler := range handlers {
		handler.InitRegister(h.srv)
	}

	return h
}

func (h *handle) Run(ctx context.Context) error {
	return h.srv.Run(h.addr)
}
