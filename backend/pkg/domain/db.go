package domain

import (
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger"

	"github.com/velocity-ci/velocity/backend/pkg/velocity"
	"go.uber.org/zap"

	"github.com/asdine/storm"
)

func NewStormDB(path string) *storm.DB {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, os.ModePerm)
	db, err := storm.Open(path)
	if err != nil {
		velocity.GetLogger().Fatal("could not create storm DB", zap.Error(err))
	}

	return db
}

func NewBadgerDB(path string) *badger.DB {
	opts := badger.DefaultOptions
	opts.Dir = path
	opts.ValueDir = path
	db, err := badger.Open(opts)
	if err != nil {
		velocity.GetLogger().Fatal("could not open badger DB", zap.Error(err))
	}

	return db
}

type PagingQuery struct {
	Limit int `json:"amount" query:"amount"`
	Page  int `json:"page" query:"page"`
}

func NewPagingQuery() *PagingQuery {
	return &PagingQuery{
		Limit: 10,
		Page:  1,
	}
}
