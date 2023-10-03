package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/tss-calculator/tools/pkg/platform/application/model"
)

type Context struct {
	BaseContext string            `json:"baseContext,omitempty"`
	Branches    map[string]string `json:"branches"`
}

type Repository struct {
	GitSrc string `json:"gitSrc"`
}

type Config struct {
	RepoSrc      string                `json:"repoSrc"`
	Contexts     map[string]Context    `json:"contexts"`
	Repositories map[string]Repository `json:"repositories"`
}

func Load(filePath string) (ApplicationConfig, error) {
	configFile, err := os.Open(filePath)
	if err != nil {
		return ApplicationConfig{}, err
	}
	defer configFile.Close()
	configBody, err := io.ReadAll(configFile)
	if err != nil {
		return ApplicationConfig{}, err
	}

	var config Config
	err = json.Unmarshal(configBody, &config)
	if err != nil {
		return ApplicationConfig{}, err
	}
	err = assertRepositories(config)
	if err != nil {
		return ApplicationConfig{}, err
	}

	for contextID, context := range config.Contexts {
		if context.BaseContext != "" {
			baseContext, ok := config.Contexts[context.BaseContext]
			if !ok {
				return ApplicationConfig{}, fmt.Errorf(
					"base context %v for context %v not found", context.BaseContext, contextID,
				)
			}
			context = mergeContextBranches(baseContext, context)
		}
	}

	return GetApplicationConfig(config), nil
}

type ApplicationConfig struct {
	RepoSrc      string
	Contexts     map[model.ContextID]model.Context
	Repositories []model.Repository
}

func GetApplicationConfig(config Config) ApplicationConfig {
	contexts := make(map[model.ContextID]model.Context, len(config.Contexts))
	for contextID, context := range config.Contexts {
		contexts[contextID] = model.Context{
			ID:       contextID,
			Branches: context.Branches,
		}
	}

	repositories := make([]model.Repository, 0, len(config.Repositories))
	for repositoryID, repository := range config.Repositories {
		repositories = append(repositories, model.Repository{
			ID:     repositoryID,
			GitSrc: repository.GitSrc,
		})
	}

	return ApplicationConfig{
		RepoSrc:      config.RepoSrc,
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
