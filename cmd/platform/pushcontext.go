package main

import (
	stdcontext "context"
	"errors"

	"github.com/tss-calculator/tools/pkg/platform/infrastructure/dependency"
)

func pushContext(ctx stdcontext.Context, context string, force bool) error {
	if context == "" {
		return errors.New("context not provided")
	}
	dependencyContainer, err := dependency.ContainerFromContext(ctx)
	if err != nil {
		return err
	}
	return dependencyContainer.Platform().PushContext(ctx, context, force)
}
