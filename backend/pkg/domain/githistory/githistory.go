package githistory

import (
	"time"

	"github.com/velocity-ci/velocity/backend/pkg/domain/project"
)

type Commit struct {
	ID        string           `json:"id"`
	Project   *project.Project `json:"project"`
	Hash      string           `json:"hash"`
	Author    string           `json:"author"`
	CreatedAt time.Time        `json:"createdAt"`
	Message   string           `json:"message"`
	Signed    string           `json:"signed"`
}

type CommitQuery struct {
	Limit    int      `json:"amount" query:"amount"`
	Page     int      `json:"page" query:"page"`
	Branches []string `json:"branches" query:"branches"`
	Branch   string   `json:"branch" query:"branch"`
}

func NewCommitQuery() *CommitQuery {
	return &CommitQuery{
		Limit: 10,
		Page:  1,
	}
}

type Branch struct {
	ID          string           `json:"id"`
	Project     *project.Project `json:"project"`
	Name        string           `json:"name"`
	LastUpdated time.Time        `json:"lastUpdated"`
	Active      bool             `json:"active"`
}
