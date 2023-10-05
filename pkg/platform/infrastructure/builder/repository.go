package builder

import (
	stdcontext "context"
	"fmt"
	"time"

	applogger "github.com/tss-calculator/go-lib/pkg/application/logger"

	"github.com/tss-calculator/tools/pkg/platform/application/model"
	"github.com/tss-calculator/tools/pkg/platform/application/service"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/command"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/config/buildconfig"
)

func NewRepositoryBuilder(
	logger applogger.Logger,
	configLoader buildconfig.ConfigLoader,
	repositoryProvider service.RepositoryProvider,
	runner command.Runner,
) service.RepositoryBuilder {
	return &repositoryBuilder{
		logger:             logger,
		configLoader:       configLoader,
		repositoryProvider: repositoryProvider,
		runner:             runner,
	}
}

type repositoryBuilder struct {
	logger             applogger.Logger
	configLoader       buildconfig.ConfigLoader
	repositoryProvider service.RepositoryProvider
	runner             command.Runner
}

func (builder repositoryBuilder) Build(
	ctx stdcontext.Context,
	repositories map[model.RepositoryID]model.Repository,
) error {
	buildRepository := func(repository model.Repository, buildConfig model.BuildConfig) error {
		output, err := builder.buildSources(ctx, repository.ID, buildConfig)
		if err != nil {
			builder.logger.Debug(output)
			return err
		}
		output, err = builder.buildDockerImage(ctx, repository.ID, buildConfig)
		builder.logger.Debug(output)
		if err != nil {
			return err
		}
		return nil
	}
	buildConfigMap := make(map[model.RepositoryID]model.BuildConfig)
	for repositoryID := range repositories {
		buildConfig, err := builder.configLoader.Load(builder.repositoryProvider.RepositoryPath(repositoryID) + "/platform-build.json")
		if err != nil {
			return err
		}
		buildConfigMap[repositoryID] = buildConfig
	}
	buildMap := make(map[model.RepositoryID]struct{})
	for repositoryID, repository := range repositories {
		if _, ok := buildMap[repositoryID]; ok {
			continue
		}
		for _, depends := range buildConfigMap[repositoryID].DockerImage.DependsOn {
			err := buildRepository(repositories[depends], buildConfigMap[depends])
			if err != nil {
				return err
			}
		}
		err := buildRepository(repository, buildConfigMap[repositoryID])
		if err != nil {
			return err
		}
	}
	return nil
}

func (builder repositoryBuilder) buildSources(ctx stdcontext.Context, repositoryID model.RepositoryID, buildConfig model.BuildConfig) (string, error) {
	builder.logger.Info(fmt.Sprintf("start build sources \"%v\"...", repositoryID))
	start := time.Now()
	defer func() {
		builder.logger.Info(fmt.Sprintf("done in %v", time.Since(start).String()))
	}()
	return builder.runner.Execute(ctx, command.Command{
		WorkDir:    builder.repositoryProvider.RepositoryPath(repositoryID),
		Executable: buildConfig.Sources.Executable,
		Args:       buildConfig.Sources.Args,
	})
}

func (builder repositoryBuilder) buildDockerImage(ctx stdcontext.Context, repositoryID model.RepositoryID, buildConfig model.BuildConfig) (string, error) {
	builder.logger.Info(fmt.Sprintf("start build docker image \"%v\"...", repositoryID))
	start := time.Now()
	defer func() {
		builder.logger.Info(fmt.Sprintf("done in %v", time.Since(start).String()))
	}()
	return builder.runner.Execute(ctx, command.Command{
		WorkDir:    builder.repositoryProvider.RepositoryPath(repositoryID),
		Executable: buildConfig.DockerImage.Executable,
		Args:       buildConfig.DockerImage.Args,
	})
}
