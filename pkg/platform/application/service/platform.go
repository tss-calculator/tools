package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	applogger "github.com/tss-calculator/go-lib/pkg/application/logger"
	"github.com/tss-calculator/tools/pkg/platform/application/model"
)

type RepositoryProvider interface {
	Exist(repository model.Repository) (bool, error)
	Clone(ctx context.Context, repository model.Repository) error
	Fetch(ctx context.Context, repository model.Repository) error
	Checkout(ctx context.Context, repository model.Repository, branch string) error
	RepositoryPath(id model.RepositoryID) string
	Hash(ctx context.Context, repositoryID model.RepositoryID) (string, error)
	BranchName(ctx context.Context, repositoryID model.RepositoryID) (string, error)
	Reset(ctx context.Context, repositoryID model.RepositoryID) error
	Merge(ctx context.Context, repositoryID model.RepositoryID, branch string) error
	Push(ctx context.Context, repositoryID model.RepositoryID, dryRun bool) (string, error)
}

type RepositoryBuilder interface {
	Build(ctx context.Context, repositories map[model.RepositoryID]model.Repository) error
}

type Platform interface {
	Checkout(ctx context.Context, context model.ContextID) error
	Build(ctx context.Context) error
	ResetContext(ctx context.Context) error
	MergeContext(ctx context.Context, fromContext model.ContextID) error
	PushContext(ctx context.Context, context model.ContextID, force bool) error
}

func NewPlatformService(
	config model.Platform,
	logger applogger.Logger,
	repositoryProvider RepositoryProvider,
	repositoryBuilder RepositoryBuilder,
) Platform {
	return &platform{
		config:             config,
		logger:             logger,
		repositoryProvider: repositoryProvider,
		repositoryBuilder:  repositoryBuilder,
	}
}

type platform struct {
	config model.Platform

	logger             applogger.Logger
	repositoryProvider RepositoryProvider
	repositoryBuilder  RepositoryBuilder
}

func (service platform) Build(ctx context.Context) error {
	repositoryMap := make(map[model.RepositoryID]model.Repository)
	for _, repository := range service.config.Repositories {
		r := repository
		repositoryMap[r.ID] = r
	}
	return service.repositoryBuilder.Build(ctx, repositoryMap)
}

func (service platform) Checkout(ctx context.Context, contextID model.ContextID) error {
	c, ok := service.config.Contexts[contextID]
	if !ok {
		return fmt.Errorf("context with id %v not found", contextID)
	}
	err := service.iterateRepositories(func(repository model.Repository) error {
		return service.checkout(ctx, repository, c.Branches[repository.ID])
	})
	if err != nil {
		return err
	}
	return service.ResetContext(ctx)
}

func (service platform) ResetContext(ctx context.Context) error {
	return service.iterateRepositories(func(repository model.Repository) error {
		return service.repositoryProvider.Reset(ctx, repository.ID)
	})
}

func (service platform) MergeContext(ctx context.Context, fromContext model.ContextID) error {
	from, contextExist := service.config.Contexts[fromContext]
	if !contextExist {
		return fmt.Errorf("context with id %v not found", fromContext)
	}
	return service.iterateRepositories(func(repository model.Repository) error {
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

func (service platform) PushContext(ctx context.Context, contextID model.ContextID, force bool) error {
	c, contextExist := service.config.Contexts[contextID]
	if !contextExist {
		return fmt.Errorf("context with id %v not found", contextID)
	}
	var bC model.Context
	if c.BaseContextID != nil {
		bC = service.config.Contexts[*c.BaseContextID]
	}
	return service.iterateRepositories(func(repository model.Repository) error {
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

func (service platform) checkout(ctx context.Context, repository model.Repository, branch string) error {
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

func (service platform) cloneIfNotExist(ctx context.Context, repository model.Repository) error {
	exist, err := service.repositoryProvider.Exist(repository)
	if err != nil {
		return err
	}
	if !exist {
		return service.repositoryProvider.Clone(ctx, repository)
	}
	return nil
}

func (service platform) iterateRepositories(f func(repository model.Repository) error) error {
	for _, repository := range service.config.Repositories {
		err := f(repository)
		if err != nil {
			return err
		}
	}
	return nil
}
