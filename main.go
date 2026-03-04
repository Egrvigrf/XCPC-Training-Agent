package main

import (
	"aATA/internal/app"
	"aATA/internal/config"
	"aATA/internal/handler/api"
	"aATA/internal/handler/task"
	"aATA/internal/svc"
	"aATA/pkg/conf"
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

type Serve interface {
	Run(ctx context.Context) error
}

const (
	Api = "api"
)

var (
	configFile = flag.String("f", "./etc/local/api.yaml", "the config file")
	modeType   = flag.String("m", "api", "server run mod")
)

func main() {
	flag.Parse()

	// 根的 ctx 必须从 main 里面统一产生，所有 goroutine 共享一个退出信号
	// 任何后台任务不允许私自 contextg.Background() 否则不可控
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var cfg config.Config
	conf.MustLoad(*configFile, &cfg)

	app.InitErrorHandler()

	svcCtx, err := svc.NewServiceContext(rootCtx, cfg)
	if err != nil {
		panic(err)
	}

	var srv Serve

	switch *modeType {
	case Api:
		runner := task.NewRunner(svcCtx) // 启动自动任务
		srv = api.NewHandle(svcCtx)      // 启动控制器

		g, ctx := errgroup.WithContext(rootCtx) //  创建自动化任务，前者是协程管理，后者是停止信号灯

		// 让 runner 在后台跑起来，在独立的协程中运行
		g.Go(func() error {
			return runner.Start(ctx)
		})

		// 在独立的协程中让 Web 服务器也跑起来
		g.Go(func() error {
			return srv.Run(ctx)
		})

		// 只要 runner 和 srv 都在正常运行，程序就一直停在这，直到人为取消或者程序崩溃
		if err := g.Wait(); err != nil && err != context.Canceled {
			panic(err)
		}
		return

	default:
		panic("请指定正确的服务")
	}
}
