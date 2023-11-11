package platformconfig

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tss-calculator/tools/pkg/platform/application/model/platform"
)

type Context struct {
	BaseContext string            `json:"baseContext,omitempty"`
	Branches    map[string]string `json:"branches"`
}

type Repository struct {
	GitSrc    string   `json:"gitSrc"`
	DependsOn []string `json:"dependsOn"`
	Images    []string `json:"images"`
}

type Config struct {
	RepoSrc      string                `json:"repoSrc"`
	Registry     string                `json:"registry"`
	Contexts     map[string]Context    `json:"contexts"`
	Repositories map[string]Repository `json:"repositories"`
}

func Load(path string) (platform.Platform, error) {
	configBody, err := os.ReadFile(path)
	if err != nil {
		return platform.Platform{}, err
	}

	var config Config
	err = json.Unmarshal(configBody, &config)
	if err != nil {
		return platform.Platform{}, err
	}
	err = assertRepositories(config)
	if err != nil {
		return platform.Platform{}, err
	}

	for contextID, context := range config.Contexts {
		if context.BaseContext != "" {
			baseContext, ok := config.Contexts[context.BaseContext]
			if !ok {
				return platform.Platform{}, fmt.Errorf(
					"base context %v for context %v not found", context.BaseContext, contextID,
				)
			}
			context = mergeContextBranches(baseContext, context)
		}
	}

	return mapToPlatformConfig(config), nil
}

func mapToPlatformConfig(config Config) platform.Platform {
	contexts := make(map[platform.ContextID]platform.Context)
	for contextID, context := range config.Contexts {
		contexts[contextID] = platform.Context{
			ID:            contextID,
			BaseContextID: toOptString(context.BaseContext),
			Branches:      context.Branches,
		}
	}

	repositories := make([]platform.Repository, 0, len(config.Repositories))
	for repositoryID, repository := range config.Repositories {
		repositories = append(repositories, platform.Repository{
			ID:        repositoryID,
			GitSrc:    repository.GitSrc,
			Images:    repository.Images,
			DependsOn: repository.DependsOn,
		})
	}

	return platform.Platform{
		RepoSrc:      config.RepoSrc,
		Registry:     config.Registry,
		Contexts:     contexts,
		Repositories: repositories,
	}
}

func mergeContextBranches(baseContext, context Context) Context {
	for repositoryID, branch := range baseContext.Branches {
		if _, ok := context.Branches[repositoryID]; !ok {
			context.Branches[repositoryID] = branch
		}
	}
	return context
}

func assertRepositories(config Config) error {
	for _, context := range config.Contexts {
		for repositoryID := range context.Branches {
			if _, ok := config.Repositories[repositoryID]; !ok {
				return fmt.Errorf("unexpected repository %v", repositoryID)
			}
		}
	}
	return nil
}

func toOptString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
