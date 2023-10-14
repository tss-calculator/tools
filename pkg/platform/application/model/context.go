package model

type ContextID = string

type Context struct {
	ID            ContextID
	BaseContextID *ContextID
	Branches      map[RepositoryID]string
}
