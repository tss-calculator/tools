package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/tss-calculator/go-lib/pkg/infrastructure/logger"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/config/platformconfig"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/dependency"

	"github.com/urfave/cli/v2"
)

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	ctx = listenOSKillSignalsContext(ctx)
	mainLogger := logger.NewTextLogger()

	platformConfig, err := platformconfig.Load("platform.json")
	if err != nil {
		mainLogger.FatalError(err, "failed load platform config")
	}
	container := dependency.NewDependencyContainer(mainLogger, platformConfig)
	ctx = dependency.ContainerToContext(ctx, container)

	app := &cli.App{
		Name: "platform",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "context",
				Required: true,
			},
		},
		Commands: cli.Commands{
			&cli.Command{
				Name: "checkout",
				Action: func(c *cli.Context) error {
					return checkout(c.Context, c.String("context"))
				},
			},
			&cli.Command{
				Name: "build",
				Before: func(c *cli.Context) error {
					return checkout(c.Context, c.String("context"))
				},
				Action: func(c *cli.Context) error {
					return build(c.Context)
				},
			},
		},
	}
	err = app.RunContext(ctx, os.Args)
	if err != nil {
		mainLogger.FatalError(err)
	}
}

func listenOSKillSignalsContext(ctx context.Context) context.Context {
	var cancelFunc context.CancelFunc
	ctx, cancelFunc = context.WithCancel(ctx)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
		select {
		case <-ch:
			cancelFunc()
		case <-ctx.Done():
			return
		}
	}()
	return ctx
}
