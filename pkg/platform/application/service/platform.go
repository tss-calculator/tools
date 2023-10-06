package service

import (
	"context"
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
	Hash(ctx context.Context, repository model.Repository) (string, error)
}

type RepositoryBuilder interface {
	Build(ctx context.Context, repositories map[model.RepositoryID]model.Repository) error
}

type Platform interface {
	Checkout(ctx context.Context, context string) error
	Build(ctx context.Context) error
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

func (service platform) Checkout(ctx context.Context, contextID string) error {
	c, ok := service.config.Contexts[contextID]
	if !ok {
		return fmt.Errorf("context with id %v not found", contextID)
	}
	return service.iterateRepositories(func(repository model.Repository) error {
		return service.checkout(ctx, repository, c.Branches[repository.ID])
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
