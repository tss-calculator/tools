package pipeline

import (
	stdcontext "context"
	"encoding/hex"
	"os"
	"text/template"

	"github.com/pkg/errors"

	"github.com/tss-calculator/tools/pkg/platform/application/model/platform"
	"github.com/tss-calculator/tools/pkg/platform/application/service"
	"github.com/tss-calculator/tools/pkg/platform/infrastructure/command"
)

func NewPipelineExecutor(
	registry string,
	pipelines map[platform.PipelineID]string,
	runner command.Runner,
	repositoryProvider service.RepositoryProvider,
) service.PipelineExecutor {
	return &executor{
		registry:           registry,
		pipelines:          pipelines,
		runner:             runner,
		repositoryProvider: repositoryProvider,
	}
}

type Repository struct {
	Hash      string
	Images    []string
	Directory string
}

type pipelineVariables struct {
	ContextID    string
	Pipeline     string
	Registry     string
	Repositories map[string]Repository
}

type executor struct {
	registry  string
	pipelines map[platform.PipelineID]string

	runner             command.Runner
	repositoryProvider service.RepositoryProvider
}

func (e executor) Execute(
	ctx stdcontext.Context,
	contextID platform.ContextID,
	pipeline platform.PipelineID,
	repositoryMap map[platform.RepositoryID]service.RepositoryInfo,
) error {
	variables := e.loadPipelineVariables(contextID, pipeline, e.registry, repositoryMap)
	pipelineFile, err := e.buildPipeline(pipeline, variables)
	if err != nil {
		return err
	}
	defer os.Remove(pipelineFile.Name())
	_, err = e.runner.Execute(ctx, command.Command{
		Executable: "bash",
		Args:       []string{pipelineFile.Name()},
		Verbose:    true,
	})
	return err
}

func (e executor) loadPipelineVariables(
	contextID platform.ContextID,
	pipeline platform.PipelineID,
	registry string,
	repositoryMap map[platform.RepositoryID]service.RepositoryInfo,
) pipelineVariables {
	repositories := make(map[string]Repository)
	for id, repository := range repositoryMap {
		repositories[id] = Repository{
			Hash:      hex.EncodeToString(repository.Hash),
			Images:    repository.Images,
			Directory: e.repositoryProvider.RepositoryPath(id),
		}
	}
	return pipelineVariables{
		ContextID:    contextID,
		Pipeline:     pipeline,
		Registry:     registry,
		Repositories: repositories,
	}
}

func (e executor) buildPipeline(pipelineID platform.PipelineID, variables pipelineVariables) (*os.File, error) {
	pipelineFile, err := os.CreateTemp(".platform", pipelineID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create temporary file for %v pipeline", pipelineID)
	}
	pipelineTemplate, err := template.ParseFiles(e.pipelines[pipelineID])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %v pipeline template", pipelineID)
	}
	err = pipelineTemplate.Execute(pipelineFile, variables)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute %v pipeline template", pipelineID)
	}
	return pipelineFile, nil
}
