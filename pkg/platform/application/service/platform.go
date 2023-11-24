package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	applogger "github.com/tss-calculator/go-lib/pkg/application/logger"
	platformconfig "github.com/tss-calculator/tools/pkg/platform/application/model/platform"
)

type RepositoryProvider interface {
	Exist(repository platformconfig.Repository) (bool, error)
	Clone(ctx context.Context, repository platformconfig.Repository) error
	Fetch(ctx context.Context, repository platformconfig.Repository) error
	Checkout(ctx context.Context, repository platformconfig.Repository, branch string) error
	RepositoryPath(id platformconfig.RepositoryID) string
	Hash(ctx context.Context, repositoryID platformconfig.RepositoryID) (string, error)
	BranchName(ctx context.Context, repositoryID platformconfig.RepositoryID) (string, error)
	Reset(ctx context.Context, repositoryID platformconfig.RepositoryID) error
	Merge(ctx context.Context, repositoryID platformconfig.RepositoryID, branch string) error
	Push(ctx context.Context, repositoryID platformconfig.RepositoryID, dryRun bool) (string, error)
}

type RepositoryInfo struct {
	platformconfig.Repository
	Hash   []byte
	Branch *string
}

type RepositoryBuilder interface {
	Build(ctx context.Context, registry string, repositories map[platformconfig.RepositoryID]RepositoryInfo) error
	Push(ctx context.Context, registry string, repositories map[platformconfig.RepositoryID]RepositoryInfo) error
}

type PipelineExecutor interface {
	Execute(
		ctx context.Context,
		contextID platformconfig.ContextID,
		pipeline platformconfig.PipelineID,
		repositoryMap map[platformconfig.RepositoryID]RepositoryInfo,
	) error
}

type Platform interface {
	Checkout(ctx context.Context, context platformconfig.ContextID) error
	Build(ctx context.Context, pushImages bool) error
	ResetContext(ctx context.Context) error
	MergeContext(ctx context.Context, fromContext platformconfig.ContextID) error
	PushContext(ctx context.Context, context platformconfig.ContextID, force bool) error
	ExecutePipelines(ctx context.Context, contextID platformconfig.ContextID, pipelines []string) error
}

func NewPlatformService(
	config platformconfig.Platform,
	logger applogger.Logger,
	repositoryProvider RepositoryProvider,
	repositoryBuilder RepositoryBuilder,
	pipelineExecutor PipelineExecutor,
) Platform {
	return &platform{
		config:             config,
		logger:             logger,
		repositoryProvider: repositoryProvider,
		repositoryBuilder:  repositoryBuilder,
		repositoryMap:      buildRepositoryMap(config),
		pipelineExecutor:   pipelineExecutor,
	}
}

type platform struct {
	config        platformconfig.Platform
	repositoryMap map[platformconfig.RepositoryID]platformconfig.Repository

	logger             applogger.Logger
	repositoryProvider RepositoryProvider
	repositoryBuilder  RepositoryBuilder
	pipelineExecutor   PipelineExecutor
}

func (service platform) ExecutePipelines(ctx context.Context, contextID platformconfig.ContextID, pipelines []string) error {
	repositoryMap := make(map[platformconfig.RepositoryID]RepositoryInfo)
	err := service.iterateRepositories(func(repository platformconfig.Repository) error {
		hash, err := service.buildRepositoryHash(ctx, repository)
		if err != nil {
			return err
		}
		branch, err := service.buildRepositoryBranch(ctx, repository)
		if err != nil {
			return err
		}
		repositoryMap[repository.ID] = RepositoryInfo{
			repository,
			hash,
			branch,
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, pipeline := range pipelines {
		err = service.pipelineExecutor.Execute(ctx, contextID, pipeline, repositoryMap)
		if err != nil {
			return err
		}
	}
	return nil
}

func (service platform) Build(ctx context.Context, pushImages bool) error {
	repositoryMap := make(map[platformconfig.RepositoryID]RepositoryInfo)
	err := service.iterateRepositories(func(repository platformconfig.Repository) error {
		hash, err := service.buildRepositoryHash(ctx, repository)
		if err != nil {
			return err
		}
		branch, err := service.buildRepositoryBranch(ctx, repository)
		if err != nil {
			return err
		}
		repositoryMap[repository.ID] = RepositoryInfo{
			repository,
			hash,
			branch,
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = service.repositoryBuilder.Build(ctx, service.config.Registry, repositoryMap)
	if err != nil {
		return err
	}
	if pushImages {
		return service.repositoryBuilder.Push(ctx, service.config.Registry, repositoryMap)
	}
	return nil
}

func (service platform) Checkout(ctx context.Context, contextID platformconfig.ContextID) error {
	c, ok := service.config.Contexts[contextID]
	if !ok {
		return fmt.Errorf("context with id %v not found", contextID)
	}
	err := service.iterateRepositories(func(repository platformconfig.Repository) error {
		return service.checkout(ctx, repository, c.Branches[repository.ID])
	})
	if err != nil {
		return err
	}
	return service.ResetContext(ctx)
}

func (service platform) ResetContext(ctx context.Context) error {
	return service.iterateRepositories(func(repository platformconfig.Repository) error {
		return service.repositoryProvider.Reset(ctx, repository.ID)
	})
}

func (service platform) MergeContext(ctx context.Context, fromContext platformconfig.ContextID) error {
	from, contextExist := service.config.Contexts[fromContext]
	if !contextExist {
		return fmt.Errorf("context with id %v not found", fromContext)
	}
	return service.iterateRepositories(func(repository platformconfig.Repository) error {
		currentBranch, err := service.repositoryProvider.BranchName(ctx, repository.ID)
		if err != nil {
			return err
		}
		branch, branchExist := from.Branches[repository.ID]
		if !branchExist || currentBranch == branch {
			service.logger.Info(fmt.Sprintf("skip merge branch from repository \"%v\"", repository.ID))
			return nil
		}
		service.logger.Info(fmt.Sprintf("merge branch \"%v\" to \"%v\" from repository \"%v\"", branch, currentBranch, repository.ID))
		err = service.repositoryProvider.Merge(ctx, repository.ID, branch)
		if err != nil {
			service.logger.Error(err, fmt.Sprintf("failed merge repository \"%v\"", repository.ID))
			resetErr := service.repositoryProvider.Reset(ctx, repository.ID)
			return errors.Join(err, resetErr)
		}
		return nil
	})
}

func (service platform) PushContext(ctx context.Context, contextID platformconfig.ContextID, force bool) error {
	c, contextExist := service.config.Contexts[contextID]
	if !contextExist {
		return fmt.Errorf("context with id %v not found", contextID)
	}
	var bC platformconfig.Context
	if c.BaseContextID != nil {
		bC = service.config.Contexts[*c.BaseContextID]
	}
	return service.iterateRepositories(func(repository platformconfig.Repository) error {
		baseBranch := bC.Branches[repository.ID]
		branch, branchExist := c.Branches[repository.ID]
		if !branchExist || branch == baseBranch {
			service.logger.Info(fmt.Sprintf("skip push repository \"%v\"", repository.ID))
			return nil
		}
		service.logger.Info(fmt.Sprintf("push repository \"%v\" (dry-run = %v)", repository.ID, !force))
		output, err := service.repositoryProvider.Push(ctx, repository.ID, !force)
		service.logger.Info(output)
		return err
	})
}

func (service platform) checkout(ctx context.Context, repository platformconfig.Repository, branch string) error {
	service.logger.Info(fmt.Sprintf("checkout \"%v\" to branch \"%v\"...", repository.ID, branch))
	start := time.Now()
	defer func() {
		service.logger.Info(fmt.Sprintf("done in %v", time.Since(start).String()))
	}()

	err := service.cloneIfNotExist(ctx, repository)
	if err != nil {
		return err
	}
	err = service.repositoryProvider.Fetch(ctx, repository)
	if err != nil {
		return err
	}
	return service.repositoryProvider.Checkout(ctx, repository, branch)
}

func (service platform) cloneIfNotExist(ctx context.Context, repository platformconfig.Repository) error {
	exist, err := service.repositoryProvider.Exist(repository)
	if err != nil {
		return err
	}
	if !exist {
		return service.repositoryProvider.Clone(ctx, repository)
	}
	return nil
}

func (service platform) iterateRepositories(f func(repository platformconfig.Repository) error) error {
	for _, repository := range service.config.Repositories {
		err := f(repository)
		if err != nil {
			return err
		}
	}
	return nil
}

func (service platform) buildRepositoryHash(ctx context.Context, repository platformconfig.Repository) ([]byte, error) {
	hash := sha256.New()
	commit, err := service.repositoryProvider.Hash(ctx, repository.ID)
	if err != nil {
		return nil, err
	}
	hash.Write([]byte(commit))
	for _, depends := range repository.DependsOn {
		repositoryHash, err := service.buildRepositoryHash(ctx, service.repositoryMap[depends])
		if err != nil {
			return nil, err
		}
		hash.Write(repositoryHash)
	}
	return hash.Sum(nil), nil
}

func (service platform) buildRepositoryBranch(ctx context.Context, repository platformconfig.Repository) (*string, error) {
	b, err := service.repositoryProvider.BranchName(ctx, repository.ID)
	if err != nil {
		return nil, err
	}
	repositoryBranch := &b
	for _, depends := range repository.DependsOn {
		branch, err := service.buildRepositoryBranch(ctx, service.repositoryMap[depends])
		if err != nil {
			return nil, err
		}
		if repositoryBranch != branch {
			return nil, nil
		}
	}
	return repositoryBranch, nil
}

func buildRepositoryMap(config platformconfig.Platform) map[platformconfig.RepositoryID]platformconfig.Repository {
	repositoryMap := make(map[platformconfig.RepositoryID]platformconfig.Repository)
	for _, repository := range config.Repositories {
		r := repository
		repositoryMap[r.ID] = r
	}
	return repositoryMap
}
