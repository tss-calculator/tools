package main

import (
	stdcontext "context"

	"github.com/tss-calculator/tools/pkg/platform/infrastructure/dependency"
)

func mergeContext(ctx stdcontext.Context, fromContext string) error {
	dependencyContainer, err := dependency.ContainerFromContext(ctx)
	if err != nil {
		return err
	}
	return dependencyContainer.Platform().MergeContext(ctx, fromContext)
}
