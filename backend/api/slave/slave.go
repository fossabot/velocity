package slave

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type RequestSlave struct {
	ID string `json:"id"`
}

type Slave struct {
	ID      string
	State   string // ready, busy, disconnected
	ws      *websocket.Conn
	Command *CommandMessage
}

func NewSlave(ID string) *Slave {
	return &Slave{
		ID:      ID,
		State:   "disconnected",
		Command: nil,
	}
}

func (s *Slave) SetWebSocket(ws *websocket.Conn) {
	s.ws = ws
}

type SlaveMessage struct {
	Type string  `json:"type"`
	Data Message `json:"data"`
}

type Message interface{}

type LogMessage struct {
	ProjectID  string `json:"projectId"`
	CommitHash string `json:"commitHash"`
	BuildID    uint64 `json:"buildId"`
	Step       uint64 `json:"step"`
	Status     string `json:"status"`
	Output     string `json:"output"`
}

func (c *SlaveMessage) UnmarshalJSON(b []byte) error {
	var objMap map[string]*json.RawMessage
	// We'll store the error (if any) so we can return it if necessary
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}

	// Deserialize Command
	err = json.Unmarshal(*objMap["type"], &c.Type)
	if err != nil {
		return err
	}

	// Deserialize Data by command
	var rawData json.RawMessage
	err = json.Unmarshal(*objMap["data"], &rawData)
	if err != nil {
		return err
	}

	if c.Type == "log" {
		d := LogMessage{}
		err := json.Unmarshal(rawData, &d)
		if err != nil {
			return err
		}
		c.Data = &d
	} else {
		return fmt.Errorf("unsupported type in json.Unmarshal: %s", c.Type)
	}

	return nil
}