package platform

type ContextID = string

type RepositoryID = string

type Image = string

type Context struct {
	ID            ContextID
	BaseContextID *ContextID
	Branches      map[RepositoryID]string
}

type Repository struct {
	ID        RepositoryID
	DependsOn []RepositoryID
	GitSrc    string
	Images    []Image
}

type PipelineID = string

type Platform struct {
	RepoSrc      string
	Registry     string
	Contexts     map[ContextID]Context
	Repositories []Repository
	Pipelines    map[PipelineID]string
}
