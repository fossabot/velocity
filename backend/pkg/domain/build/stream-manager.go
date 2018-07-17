package build

import (
	"fmt"
	"time"

	"github.com/asdine/storm"
	"github.com/dgraph-io/badger"
	uuid "github.com/satori/go.uuid"
	"github.com/velocity-ci/velocity/backend/pkg/domain"
	"github.com/velocity-ci/velocity/backend/pkg/velocity"
	"go.uber.org/zap"
)

const (
	EventStreamLineCreate = "streamLine:new"
)

type StreamManager struct {
	db  *streamStormDB
	bdb *streamBadgerDB

	brokers []domain.Broker
}

func NewStreamManager(
	db *storm.DB,
	bdb *badger.DB,
) *StreamManager {
	m := &StreamManager{
		db:      newStreamStormDB(db),
		bdb:     newStreamBadgerDB(bdb),
		brokers: []domain.Broker{},
	}
	return m
}

func (m *StreamManager) AddBroker(b domain.Broker) {
	m.brokers = append(m.brokers, b)
}

func (m *StreamManager) create(
	s *Step,
	name string,
) *Stream {
	stream := &Stream{
		ID:   uuid.NewV3(uuid.NewV1(), s.ID).String(),
		Step: s,
		Name: name,
	}

	m.bdb.save(stream)
	return stream
}

func (m *StreamManager) update(s *Stream) error {
	return m.bdb.save(s)
}

func (m *StreamManager) GetByID(id string) (*Stream, error) {
	// return GetStreamByID(m.db.DB, id)
	return m.bdb.getStreamByID(id)
}

func (m *StreamManager) CreateStreamLine(
	stream *Stream,
	lineNumber int,
	timestamp time.Time,
	output string,
) *StreamLine {
	sL := &StreamLine{
		StreamID:   stream.ID,
		LineNumber: lineNumber,
		Timestamp:  timestamp,
		Output:     fmt.Sprintf("%s", output),
	}
	m.bdb.saveStreamLine(sL)
	for _, b := range m.brokers {
		b.EmitAll(&domain.Emit{
			Topic:   fmt.Sprintf("stream:%s", stream.ID),
			Event:   EventStreamLineCreate,
			Payload: sL,
		})
	}
	return sL
}

func (m *StreamManager) GetStreamsForStep(s *Step) []*Stream {
	// return getStreamsByStepID(m.db.DB, s.ID)
	return m.bdb.getStreamsByStepID(s.ID)
}

func (m *StreamManager) GetStreamLines(s *Stream, q *domain.PagingQuery) ([]*StreamLine, int) {
	return m.bdb.getLinesByStream(s, q)
}

func (m *StreamManager) Update(stream *Stream) error {
	if err := m.bdb.save(stream); err != nil {
		return err
	}
	// for _, b := range m.brokers {
	// 	b.EmitAll(&domain.Emit{
	// 		Event:   ,
	// 		Payload: s,
	// 	})
	// }

	return nil
}

func GetStreamByID(db *storm.DB, id string) (*Stream, error) {
	var sS StormStream
	if err := db.One("ID", id, &sS); err != nil {
		velocity.GetLogger().Error("error", zap.Error(err))
		return nil, err
	}
	return sS.toStream(db), nil
}
