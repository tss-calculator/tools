package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
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
	container := dependency.NewDependencyContainer(mainLogger, platformConfig, os.Getenv("SILENT") != "")
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
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "push-images",
					},
				},
				Before: func(c *cli.Context) error {
					return checkout(c.Context, c.String("context"))
				},
				Action: func(c *cli.Context) error {
					return build(c.Context, c.Bool("push-images"))
				},
			},
			&cli.Command{
				Name: "reset-context",
				Action: func(c *cli.Context) error {
					return resetContext(c.Context)
				},
			},
			&cli.Command{
				Name: "merge-context",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "from-context",
						Required: true,
					},
				},
				Before: func(c *cli.Context) error {
					return checkout(c.Context, c.String("context"))
				},
				Action: func(c *cli.Context) error {
					return mergeContext(c.Context, c.String("from-context"))
				},
			},
			&cli.Command{
				Name: "push-context",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "force",
					},
				},
				Action: func(c *cli.Context) error {
					return pushContext(c.Context, c.String("context"), c.Bool("force"))
				},
			},
			&cli.Command{
				Name: "execute",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:     "pipelines",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return executePipeline(c.Context, c.String("context"), c.StringSlice("pipelines"))
				},
			},
		},
	}
	err = app.RunContext(ctx, os.Args)
	if err != nil {
		mainLogger.FatalError(err, "failed execute command "+strings.Join(os.Args, " "))
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
