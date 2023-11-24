package builder

import (
	stdcontext "context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	applogger "github.com/tss-calculator/go-lib/pkg/application/logger"
	"github.com/tss-calculator/tools/pkg/platform/application/model/platform"
	"github.com/tss-calculator/tools/pkg/platform/application/service"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/command"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/config/buildconfig"
)

func NewRepositoryBuilder(
	logger applogger.Logger,
	configLoader *buildconfig.Loader,
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
	configLoader       *buildconfig.Loader
	repositoryProvider service.RepositoryProvider
	runner             command.Runner
}

func (builder repositoryBuilder) Push(ctx stdcontext.Context, registry string, repositories map[platform.RepositoryID]service.RepositoryInfo) error {
	for _, repository := range repositories {
		err := builder.pushDockerImages(ctx, registry, repository)
		if err != nil {
			return err
		}
	}
	return nil
}

func (builder repositoryBuilder) Build(
	ctx stdcontext.Context,
	registry string,
	repositories map[platform.RepositoryID]service.RepositoryInfo,
) error {
	buildMap := make(map[platform.RepositoryID]struct{})
	buildRepository := func(repository service.RepositoryInfo) error {
		if _, ok := buildMap[repository.ID]; ok {
			return nil
		}
		err := builder.buildSources(ctx, repository)
		if err != nil {
			return err
		}
		err = builder.buildDockerImages(ctx, registry, repository, repositories)
		if err != nil {
			return err
		}
		buildMap[repository.ID] = struct{}{}
		return nil
	}

	for _, repository := range repositories {
		for _, repositoryID := range repository.DependsOn {
			err := buildRepository(repositories[repositoryID])
			if err != nil {
				return err
			}
		}
		err := buildRepository(repository)
		if err != nil {
			return err
		}
	}
	return nil
}

func (builder repositoryBuilder) buildSources(ctx stdcontext.Context, repository service.RepositoryInfo) error {
	builder.logger.Info(fmt.Sprintf("start build sources \"%v\"", repository.ID))
	start := time.Now()
	defer func() {
		builder.logger.Info(fmt.Sprintf("done in %v", time.Since(start)))
	}()

	repositoryPath := builder.repositoryProvider.RepositoryPath(repository.ID)
	buildConfig, err := builder.configLoader.Load(repositoryPath + "/platform-build.json")
	if err != nil {
		return err
	}

	output, err := builder.runner.Execute(ctx, command.Command{
		WorkDir:    repositoryPath,
		Executable: buildConfig.Sources.Executable,
		Args:       buildConfig.Sources.Args,
	})
	builder.logger.Debug(output)
	return err
}

func (builder repositoryBuilder) buildDockerImages(
	ctx stdcontext.Context,
	registry string,
	repository service.RepositoryInfo,
	repositories map[platform.RepositoryID]service.RepositoryInfo,
) error {
	builder.logger.Info(fmt.Sprintf("start build docker images for \"%v\"", repository.ID))
	start := time.Now()
	defer func() {
		builder.logger.Info(fmt.Sprintf("done in %v", time.Since(start)))
	}()

	repositoryPath := builder.repositoryProvider.RepositoryPath(repository.ID)
	buildConfig, err := builder.configLoader.Load(repositoryPath + "/platform-build.json")
	if err != nil {
		return err
	}

	args := make(map[string]string)
	args["REGISTRY"] = registry
	for _, depends := range repository.DependsOn {
		r := repositories[depends]
		args[strings.ReplaceAll(depends, "-", "_")] = hex.EncodeToString(r.Hash)
	}

	args[strings.ReplaceAll(repository.ID, "-", "_")] = hex.EncodeToString(repository.Hash)
	for _, image := range buildConfig.Images {
		tags := []string{
			"--tag=" + buildTag(registry, image.Name, hex.EncodeToString(repository.Hash)),
		}
		if repository.Branch != nil {
			tags = append(tags, "--tag="+buildTag(registry, image.Name, *repository.Branch))
		}

		dockerArgs := []string{
			"build",
			image.Context,
			fmt.Sprintf("--file=%v", image.DockerFile),
		}
		dockerArgs = append(dockerArgs, tags...)
		dockerArgs = append(dockerArgs, buildArgs(args)...)
		output, err := builder.runner.Execute(ctx, command.Command{
			WorkDir:    repositoryPath,
			Executable: "docker",
			Args:       dockerArgs,
		})
		builder.logger.Debug(output)
		if err != nil {
			return err
		}
	}
	return nil
}

func (builder repositoryBuilder) pushDockerImages(ctx stdcontext.Context, registry string, repository service.RepositoryInfo) error {
	builder.logger.Info(fmt.Sprintf("start push docker images for \"%v\"", repository.ID))
	start := time.Now()
	defer func() {
		builder.logger.Info(fmt.Sprintf("done in %v", time.Since(start)))
	}()

	repositoryPath := builder.repositoryProvider.RepositoryPath(repository.ID)
	buildConfig, err := builder.configLoader.Load(repositoryPath + "/platform-build.json")
	if err != nil {
		return err
	}

	for _, image := range buildConfig.Images {
		if image.SkipPush {
			builder.logger.Info(fmt.Sprintf("skip push %v/%v", registry, image.Name))
			continue
		}
		err = builder.pushDockerImage(ctx, repository.ID, buildTag(registry, image.Name, hex.EncodeToString(repository.Hash)))
		if err != nil {
			return err
		}
		if repository.Branch != nil {
			err = builder.pushDockerImage(ctx, repository.ID, buildTag(registry, image.Name, *repository.Branch))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (builder repositoryBuilder) pushDockerImage(ctx stdcontext.Context, repositoryID platform.RepositoryID, tag string) error {
	builder.logger.Info(fmt.Sprintf("push image %v", tag))
	output, err := builder.runner.Execute(ctx, command.Command{
		WorkDir:    builder.repositoryProvider.RepositoryPath(repositoryID),
		Executable: "docker",
		Args: []string{
			"push",
			tag,
		},
	})
	builder.logger.Debug(output)
	return err
}

func buildArgs(args map[string]string) []string {
	result := make([]string, 0, len(args))
	for key, value := range args {
		result = append(result, fmt.Sprintf("--build-arg=%v=%v", strings.ToUpper(key), value))
	}
	return result
}

func buildTag(registry, imageName, tag string) string {
	return fmt.Sprintf("%v/%v:%v", registry, imageName, tag)
}
