package model

type RepositoryID = string

type Repository struct {
	ID     RepositoryID
	GitSrc string
}
