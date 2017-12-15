package knownhost

import (
	"github.com/jinzhu/gorm"
	"github.com/velocity-ci/velocity/backend/api/websocket"
)

type Manager struct {
	gormRepository   *gormRepository
	websocketManager *websocket.Manager
}

func NewManager(
	db *gorm.DB,
	websocketManager *websocket.Manager,
) *Manager {
	return &Manager{
		gormRepository:   newGORMRepository(db),
		websocketManager: websocketManager,
	}
}

func (m *Manager) Create(k KnownHost) KnownHost {
	m.gormRepository.Save(k)
	m.websocketManager.EmitAll(&websocket.PhoenixMessage{
		Topic:   "knownhosts",
		Event:   websocket.VNewBranch,
		Payload: NewResponseKnownHost(k),
	})

	return k
}

func (m *Manager) Delete(k KnownHost) {
	m.gormRepository.Delete(k)
	m.websocketManager.EmitAll(&websocket.PhoenixMessage{
		Topic:   "knownhosts",
		Event:   websocket.VDeleteCommit,
		Payload: NewResponseKnownHost(k),
	})
}

func (m *Manager) GetByID(id string) (KnownHost, error) {
	return m.gormRepository.GetCommitByCommitID(id)
}

func (m *Manager) GetAll() ([]KnownHost, uint64) {
	return m.gormRepository.GetAllBranchesByProjectID(projectID, q)
}
