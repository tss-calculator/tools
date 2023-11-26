package main

import (
	stdcontext "context"

	"github.com/tss-calculator/tools/pkg/platform/infrastructure/dependency"
)

func executePipeline(ctx stdcontext.Context, context string, pipelines []string) error {
	dependencyContainer, err := dependency.ContainerFromContext(ctx)
	if err != nil {
		return err
	}
	return dependencyContainer.Platform().ExecutePipelines(ctx, context, pipelines)
}
