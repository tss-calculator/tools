package main

import (
	stdcontext "context"
	"errors"

	"github.com/tss-calculator/tools/pkg/platform/infrastructure/dependency"
)

func executePipeline(ctx stdcontext.Context, context string, pipelines []string) error {
	if context == "" {
		return errors.New("context not provided")
	}
	dependencyContainer, err := dependency.ContainerFromContext(ctx)
	if err != nil {
		return err
	}
	return dependencyContainer.Platform().ExecutePipelines(ctx, context, pipelines)
}
