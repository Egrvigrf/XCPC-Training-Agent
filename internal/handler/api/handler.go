package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"aATA/internal/svc"
)

type Handler interface {
	InitRegister(*gin.Engine)
}

type handle struct {
	srv        *gin.Engine
	addr       string
	httpServer *http.Server
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

	// 注册路由
	handlers := initHandler(svc)
	for _, handler := range handlers {
		handler.InitRegister(h.srv)
	}

	// 用标准库 http.Server 包住 gin engine
	h.httpServer = &http.Server{
		Addr:    h.addr,
		Handler: h.srv,

		// 给慢连接一些保护
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return h
}

func (h *handle) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		// 异步启动 http 监听，关闭时这里会返回 http.ErrServerClosed
		errCh <- h.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		// 给关闭一个上限时间：等正在处理的请求完成
		// 因为 ctx 已经是 cancelled 状态，如果用 rootCtx 会立刻取消导致 Shutdown 立即返回，达不到等待效果
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = h.httpServer.Shutdown(shutdownCtx)
		return nil

	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
