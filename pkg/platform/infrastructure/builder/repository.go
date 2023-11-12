package builder

import (
	stdcontext "context"
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

func (builder repositoryBuilder) Push(ctx stdcontext.Context, registry string, repositories map[platform.RepositoryID]platform.Repository) error {
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
	repositories map[platform.RepositoryID]platform.Repository,
) error {
	buildMap := make(map[platform.RepositoryID]struct{})
	buildRepository := func(repository platform.Repository) error {
		if _, ok := buildMap[repository.ID]; ok {
			return nil
		}
		err := builder.buildRepository(ctx, registry, repository)
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

func (builder repositoryBuilder) buildRepository(ctx stdcontext.Context, registry string, repository platform.Repository) error {
	err := builder.buildSources(ctx, repository)
	if err != nil {
		return err
	}
	return builder.buildDockerImages(ctx, registry, repository)
}

func (builder repositoryBuilder) buildSources(ctx stdcontext.Context, repository platform.Repository) error {
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

func (builder repositoryBuilder) buildDockerImages(ctx stdcontext.Context, registry string, repository platform.Repository) error {
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
		hash, err2 := builder.repositoryProvider.Hash(ctx, depends)
		if err2 != nil {
			return err2
		}
		args[strings.ReplaceAll(depends, "-", "_")] = hash
	}

	repositoryHash, err := builder.repositoryProvider.Hash(ctx, repository.ID)
	if err != nil {
		return err
	}
	args[strings.ReplaceAll(repository.ID, "-", "_")] = repositoryHash
	for _, image := range buildConfig.Images {
		var hash string
		var branch string
		if image.TagBy == nil {
			hash, branch, err = builder.tagInfo(ctx, repository.ID)
		} else {
			hash, branch, err = builder.tagInfo(ctx, *image.TagBy)
		}
		if err != nil {
			return err
		}
		output, err := builder.runner.Execute(ctx, command.Command{
			WorkDir:    repositoryPath,
			Executable: "docker",
			Args: append([]string{
				"build",
				image.Context,
				fmt.Sprintf("--tag=%v", buildTag(registry, image.Name, hash)),
				fmt.Sprintf("--tag=%v", buildTag(registry, image.Name, branch)),
				fmt.Sprintf("--file=%v", image.DockerFile),
			}, buildArgs(args)...),
		})
		builder.logger.Debug(output)
		if err != nil {
			return err
		}
	}
	return nil
}

func (builder repositoryBuilder) pushDockerImages(ctx stdcontext.Context, registry string, repository platform.Repository) error {
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
		var hash string
		var branch string
		if image.TagBy == nil {
			hash, branch, err = builder.tagInfo(ctx, repository.ID)
		} else {
			hash, branch, err = builder.tagInfo(ctx, *image.TagBy)
		}
		if err != nil {
			return err
		}

		if image.SkipPush {
			builder.logger.Info(fmt.Sprintf("skip push %v/%v", registry, image.Name))
			continue
		}
		err = builder.pushDockerImage(ctx, repository.ID, buildTag(registry, image.Name, hash))
		if err != nil {
			return err
		}
		err = builder.pushDockerImage(ctx, repository.ID, buildTag(registry, image.Name, branch))
		if err != nil {
			return err
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

func (builder repositoryBuilder) tagInfo(ctx stdcontext.Context, repositoryID platform.RepositoryID) (hash, branch string, err error) {
	hash, err = builder.repositoryProvider.Hash(ctx, repositoryID)
	if err != nil {
		return "", "", err
	}
	branch, err = builder.repositoryProvider.BranchName(ctx, repositoryID)
	if err != nil {
		return "", "", err
	}
	return hash, branch, nil
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
