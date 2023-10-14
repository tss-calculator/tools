package main

import (
	stdcontext "context"
	"errors"

	"github.com/tss-calculator/tools/pkg/platform/infrastructure/dependency"
)

func mergeContext(ctx stdcontext.Context, fromContext string) error {
	if fromContext == "" {
		return errors.New("context not provided")
	}
	dependencyContainer, err := dependency.ContainerFromContext(ctx)
	if err != nil {
		return err
	}
	return dependencyContainer.Platform().MergeContext(ctx, fromContext)
}
