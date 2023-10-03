package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/tss-calculator/tools/pkg/platform/application/model"
	"github.com/tss-calculator/tools/pkg/platform/application/service"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/command"

	"github.com/pkg/errors"
)

func NewRepositoryProvider(
	repoDir string,
	runner command.Runner,
) service.RepositoryProvider {
	return &repositoryProvider{
		repoDir: repoDir,
		runner:  runner,
	}
}

type repositoryProvider struct {
	repoDir string
	runner  command.Runner
}

func (provider repositoryProvider) Exist(repository model.Repository) (bool, error) {
	_, err := os.Stat(provider.RepositoryPath(repository.ID))
	if err == nil {
		return true, nil
	}
	if err != nil && os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (provider repositoryProvider) Clone(ctx context.Context, repository model.Repository) error {
	err := provider.runner.Execute(ctx, command.Command{
		Executable: "git",
		Args:       []string{"clone", repository.GitSrc, provider.RepositoryPath(repository.ID)},
	})
	return errors.Wrapf(err, "failed to clone repository %v", repository.ID)
}

func (provider repositoryProvider) Checkout(ctx context.Context, repository model.Repository, branch string) error {
	if branch == "" {
		return fmt.Errorf("branch for repository %v is empty", repository.ID)
	}
	err := provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repository.ID),
		Executable: "git",
		Args:       []string{"checkout", fmt.Sprintf("origin/%v", branch)},
	})
	return errors.Wrapf(err, "failed to checkout repository %v on branch %v", repository.ID, branch)
}

func (provider repositoryProvider) Fetch(ctx context.Context, repository model.Repository) error {
	err := provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repository.ID),
		Executable: "git",
		Args:       []string{"fetch"},
	})
	return errors.Wrapf(err, "failed to fetch repository %v", repository.ID)
}

func (provider repositoryProvider) RepositoryPath(id model.RepositoryID) string {
	return provider.repoDir + "/" + id
}
