package model

type Platform struct {
	RepoSrc      string
	Contexts     map[ContextID]Context
	Repositories []Repository
}
