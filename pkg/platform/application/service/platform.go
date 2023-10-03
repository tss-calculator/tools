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
}

type Platform interface {
	Checkout(ctx context.Context, context string) error
}

func NewPlatformService(
	contexts map[model.ContextID]model.Context,
	repositories []model.Repository,
	logger applogger.Logger,
	repositoryProvider RepositoryProvider,
) Platform {
	return &platform{
		contexts:           contexts,
		repositories:       repositories,
		logger:             logger,
		repositoryProvider: repositoryProvider,
	}
}

type platform struct {
	contexts     map[model.ContextID]model.Context
	repositories []model.Repository

	logger             applogger.Logger
	repositoryProvider RepositoryProvider
}

func (service platform) Checkout(ctx context.Context, contextID string) error {
	c, ok := service.contexts[contextID]
	if !ok {
		return fmt.Errorf("context with id %v not found", contextID)
	}
	for _, repository := range service.repositories {
		err := service.checkout(ctx, repository, c.Branches[repository.ID])
		if err != nil {
			return err
		}
	}
	return nil
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
