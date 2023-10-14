package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/tss-calculator/tools/pkg/platform/application/model"
	"github.com/tss-calculator/tools/pkg/platform/application/service"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/command"
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
	_, err := provider.runner.Execute(ctx, command.Command{
		Executable: "git",
		Args:       []string{"clone", repository.GitSrc, provider.RepositoryPath(repository.ID)},
	})
	return errors.Wrapf(err, "failed to clone repository %v", repository.ID)
}

func (provider repositoryProvider) Checkout(ctx context.Context, repository model.Repository, branch string) error {
	if branch == "" {
		return fmt.Errorf("branch for repository %v is empty", repository.ID)
	}
	currentBranch, err := provider.BranchName(ctx, repository.ID)
	if err != nil {
		return err
	}
	if branch == currentBranch {
		return nil
	}
	_, err = provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repository.ID),
		Executable: "git",
		Args:       []string{"checkout", "-b", branch, fmt.Sprintf("origin/%v", branch)},
	})
	return errors.Wrapf(err, "failed to checkout repository %v on branch %v", repository.ID, branch)
}

func (provider repositoryProvider) Fetch(ctx context.Context, repository model.Repository) error {
	_, err := provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repository.ID),
		Executable: "git",
		Args:       []string{"fetch"},
	})
	return errors.Wrapf(err, "failed to fetch repository %v", repository.ID)
}

func (provider repositoryProvider) RepositoryPath(id model.RepositoryID) string {
	return provider.repoDir + "/" + id
}

func (provider repositoryProvider) Hash(ctx context.Context, repositoryID model.RepositoryID) (string, error) {
	hash, err := provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repositoryID),
		Executable: "git",
		Args:       []string{"rev-parse", "HEAD"},
	})
	return strings.TrimSpace(hash), errors.Wrapf(err, "failed to get hash from repository %v", repositoryID)
}

func (provider repositoryProvider) BranchName(ctx context.Context, repositoryID model.RepositoryID) (string, error) {
	branchName, err := provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repositoryID),
		Executable: "git",
		Args:       []string{"rev-parse", "--abbrev-ref", "HEAD"},
	})
	return strings.TrimSpace(branchName), errors.Wrapf(err, "failed to get branch name from repository %v", repositoryID)
}

func (provider repositoryProvider) Reset(ctx context.Context, repositoryID model.RepositoryID) error {
	_, err := provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repositoryID),
		Executable: "git",
		Args:       []string{"reset", "--hard"},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to reset repository %v", repositoryID)
	}
	_, err = provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repositoryID),
		Executable: "git",
		Args:       []string{"clean", "-dxf"},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to clean repository %v", repositoryID)
	}
	return nil
}

func (provider repositoryProvider) Merge(ctx context.Context, repositoryID model.RepositoryID, branch string) error {
	_, err := provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repositoryID),
		Executable: "git",
		Args:       []string{"merge", fmt.Sprintf("origin/%v", branch)},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to merge branch %v from repository %v", branch, repositoryID)
	}
	return nil
}

func (provider repositoryProvider) Push(ctx context.Context, repositoryID model.RepositoryID, dryRun bool) error {
	args := []string{"push"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	out, err := provider.runner.Execute(ctx, command.Command{
		WorkDir:    provider.RepositoryPath(repositoryID),
		Executable: "git",
		Args:       args,
	})
	fmt.Println(out)
	if err != nil {
		return errors.Wrapf(err, "failed to push repository %v", repositoryID)
	}
	return nil
}
