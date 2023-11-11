package dependency

import (
	"context"
	"errors"

	applogger "github.com/tss-calculator/go-lib/pkg/application/logger"
	"github.com/tss-calculator/tools/pkg/platform/application/model/platform"
	"github.com/tss-calculator/tools/pkg/platform/application/service"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/builder"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/command"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/config/buildconfig"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/provider"
)

var dependencyContainer = struct{}{}

type Container interface {
	Platform() service.Platform
	RepositoryProvider() service.RepositoryProvider
}

func NewDependencyContainer(
	logger applogger.Logger,
	platformConfig platform.Platform,
) Container {
	runner := command.NewCommandRunner(logger)
	repositoryProvider := provider.NewRepositoryProvider(platformConfig.RepoSrc, runner)
	repositoryBuilder := builder.NewRepositoryBuilder(logger, buildconfig.NewLoader(), repositoryProvider, runner)
	platformService := service.NewPlatformService(platformConfig, logger, repositoryProvider, repositoryBuilder)

	return &container{
		platform:           platformService,
		repositoryProvider: repositoryProvider,
	}
}

type container struct {
	platform           service.Platform
	repositoryProvider service.RepositoryProvider
}

func (c *container) RepositoryProvider() service.RepositoryProvider {
	return c.repositoryProvider
}

func (c *container) Platform() service.Platform {
	return c.platform
}

func ContainerFromContext(ctx context.Context) (Container, error) {
	v := ctx.Value(dependencyContainer)
	if c, ok := v.(Container); ok {
		return c, nil
	}
	return nil, errors.New("dependency container not found")
}

func ContainerToContext(ctx context.Context, c Container) context.Context {
	return context.WithValue(ctx, dependencyContainer, c)
}
