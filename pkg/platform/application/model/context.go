package model

type ContextID = string

type Context struct {
	ID       ContextID
	Branches map[RepositoryID]string
}
