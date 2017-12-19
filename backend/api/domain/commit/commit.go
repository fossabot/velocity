package commit

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Repository interface {
	CreateCommit(c Commit) Commit
	UpdateCommit(c Commit) Commit
	DeleteCommit(c Commit)
	GetCommitByCommitID(commitID string) (Commit, error)
	GetCommitByProjectIDAndCommitHash(projectID string, hash string) (Commit, error)
	GetAllCommitsByProjectID(projectID string, q CommitQuery) ([]Commit, uint64)

	CreateBranch(b Branch) Branch
	UpdateBranch(b Branch) Branch
	DeleteBranch(b Branch)
	GetBranchByProjectIDAndName(projectID string, name string) (Branch, error)
	GetAllBranchesByProjectID(projectID string, q BranchQuery) ([]Branch, uint64)
}

type CommitQuery struct {
	Amount uint64
	Page   uint64
	Branch string
	Author string
}

type BranchQuery struct {
	Amount uint64
	Page   uint64
	Active int // -1: false, 0: all, 1: true
}

type Commit struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	Message   string    `json:"message"`
	Branches  []Branch  `json:"branches"`
}

type Branch struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ProjectID   string    `json:"projectId"`
	LastUpdated time.Time `json:"lastUpdated"`
	Active      bool
}

func NewBranch(projectID string, name string) Branch {
	return Branch{
		ID:          uuid.NewV3(uuid.NewV1(), projectID).String(),
		ProjectID:   projectID,
		Name:        name,
		LastUpdated: time.Now(),
		Active:      true,
	}
}

type QueryBranch struct {
	Amount uint64
	Page   uint64
}

type ManyResponseBranch struct {
	Total  uint64           `json:"total"`
	Result []ResponseBranch `json:"result"`
}

type ResponseBranch struct {
	Name        string    `json:"name"`
	LastUpdated time.Time `json:"lastUpdated"`
	Active      bool      `json:"active"`
}

func NewResponseBranch(b Branch) ResponseBranch {
	return ResponseBranch{
		Name:        b.Name,
		LastUpdated: b.LastUpdated,
		Active:      b.Active,
	}
}

func NewCommit(
	projectID string,
	hash string,
	message string,
	author string,
	date time.Time,
	branches []Branch,
) Commit {
	return Commit{
		ID:        uuid.NewV3(uuid.NewV1(), hash).String(),
		ProjectID: projectID,
		Hash:      hash,
		Message:   message,
		Author:    author,
		CreatedAt: date,
		Branches:  branches,
	}
}

type ResponseCommit struct {
	ID        string    `json:"id"`
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	Message   string    `json:"message"`
	Branches  []string  `json:"branches"`
}

func NewResponseCommit(c Commit) ResponseCommit {
	branches := []string{}
	for _, b := range c.Branches {
		branches = append(branches, b.Name)
	}

	return ResponseCommit{
		ID:        c.ID,
		Hash:      c.Hash,
		Author:    c.Author,
		CreatedAt: c.CreatedAt,
		Message:   c.Message,
		Branches:  branches,
	}
}

type ManyResponseCommit struct {
	Total  uint64           `json:"total"`
	Result []ResponseCommit `json:"result"`
}