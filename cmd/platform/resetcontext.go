package main

import (
	stdcontext "context"

	"github.com/tss-calculator/tools/pkg/platform/infrastructure/dependency"
)

func resetContext(ctx stdcontext.Context) error {
	dependencyContainer, err := dependency.ContainerFromContext(ctx)
	if err != nil {
		return err
	}
	return dependencyContainer.Platform().ResetContext(ctx)
}
