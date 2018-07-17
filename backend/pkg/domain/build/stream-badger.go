package build

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/velocity-ci/velocity/backend/pkg/domain"
)

// key: streamLine-<streamID>-<line-num>
// key: stream-<stepID>-<id>

type streamBadgerDB struct {
	*badger.DB
}

func newStreamBadgerDB(db *badger.DB) *streamBadgerDB {
	return &streamBadgerDB{db}
}

func getKeyForStream(s *Stream) []byte {
	return []byte(fmt.Sprintf("stream-%s-%s", s.Step.ID, s.ID))
}

func getValueForStream(s *Stream) []byte {
	res, err := json.Marshal(s)
	if err != nil {
		fmt.Printf("json.Marshal error %v", err)
	}
	return res
}

func getKeyForStreamLine(s *StreamLine) []byte {
	return []byte(fmt.Sprintf("streamLine-%s-%d", s.StreamID, s.LineNumber))
}

func getValueForStreamLine(s *StreamLine) []byte {
	res, err := json.Marshal(s)
	if err != nil {
		fmt.Printf("json.Marshal error %v", err)
	}
	return res
}

func (db *streamBadgerDB) save(s *Stream) error {
	err := db.Update(func(tx *badger.Txn) error {
		return tx.Set(getKeyForStream(s), getValueForStream(s))
	})

	return err
}

func (db *streamBadgerDB) getStreamByID(id string) (st *Stream, err error) {
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		var foundK []byte
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			sK := string(k)

			if sK[len(sK)-len(id):] == id {
				foundK = k
				break
			}
		}
		if foundK != nil {
			item, _ := txn.Get(foundK)
			v, _ := item.Value()
			var s Stream
			err = json.Unmarshal(v, &s)
			st = &s
			return nil
		}

		return fmt.Errorf("could not find stream: %s", id)
	})

	return st, err
}

func (db *streamBadgerDB) getStreamsByStepID(stepID string) (r []*Stream) {
	db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		keys := [][]byte{}
		needle := fmt.Sprintf("stream-%s", stepID)

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if string(k)[0:len(needle)] == needle {
				keys = append(keys, k)
			}
		}

		for _, k := range keys {
			item, err := txn.Get(k)
			fmt.Println(err)
			v, _ := item.Value()
			var stream Stream
			json.Unmarshal(v, &stream)
			r = append(r, &stream)
		}

		return nil
	})

	return r
}

func (db *streamBadgerDB) saveStreamLine(sL *StreamLine) error {
	return db.Update(func(tx *badger.Txn) error {
		return tx.Set(getKeyForStreamLine(sL), getValueForStreamLine(sL))
	})
}

func (db *streamBadgerDB) getLinesByStream(s *Stream, pQ *domain.PagingQuery) (r []*StreamLine, t int) {
	db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		keys := [][]byte{}
		needle := fmt.Sprintf("streamLine-%s", s.ID)

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if string(k)[0:len(needle)] == needle {
				keys = append(keys, k)
			}
		}

		for _, k := range keys {
			item, _ := txn.Get(k)
			v, _ := item.Value()
			var sL StreamLine
			json.Unmarshal(v, &sL)
			r = append(r, &sL)
		}

		return nil
	})

	return r, len(r)
}
