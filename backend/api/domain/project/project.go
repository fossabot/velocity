package project

import (
	"encoding/json"
	"time"

	"github.com/gosimple/slug"
	"github.com/velocity-ci/velocity/backend/velocity"
)

// Repository - Implementing repositories will guarantee consistency between persistence objects and virtual objects.
type Repository interface {
	Create(p Project) Project
	Update(p Project) Project
	Delete(p Project)
	GetByID(ID string) (Project, error)
	GetAll(q Query) ([]Project, uint64)
}
type Project struct {
	ID         string                 `json:"id" gorm:"primary_key"`
	Name       string                 `json:"name" gorm:"unique_index"`
	Repository velocity.GitRepository `json:"repository"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Synchronising bool `json:"synchronising"`
}

type Query struct {
	Amount        uint64
	Page          uint64
	Synchronising bool
}

type ManyResponse struct {
	Total  uint64            `json:"total"`
	Result []ResponseProject `json:"result"`
}

func (p *Project) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

func NewProject(name string, repository velocity.GitRepository) Project {
	return Project{
		ID:            slug.Make(name),
		Name:          name,
		Repository:    repository,
		Synchronising: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

type RequestProject struct {
	ID         string `json:"-"`
	Name       string `json:"name" validate:"required,min=3,max=128,projectUnique"`
	Repository string `json:"repository" validate:"required,min=8,max=128"`
	PrivateKey string `json:"key"`
}

type ResponseProject struct {
	ID string `json:"id"`

	Name       string `json:"name"`
	Repository string `json:"repository"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Synchronising bool `json:"synchronising"`
}

func NewResponseProject(p Project) ResponseProject {
	return ResponseProject{
		ID:            p.ID,
		Name:          p.Name,
		Repository:    p.Repository.Address,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		Synchronising: p.Synchronising,
	}
}