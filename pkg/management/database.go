package management

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tidwall/buntdb"
	"gitlab.com/tozd/go/errors"
)

type Database struct {
	db   *buntdb.DB
	lock map[string]*sync.RWMutex
}

func NewDatabase(file string) (*Database, error) {
	db, err := buntdb.Open(file)
	if err != nil {
		return nil, err
	}

	dbd := &Database{
		db:   db,
		lock: make(map[string]*sync.RWMutex),
	}

	// make sure the agent index exists
	err = dbd.db.CreateIndex("agent_id", "agent-*", buntdb.IndexString)
	if err != nil && err != buntdb.ErrIndexExists {
		return nil, err
	}

	return dbd, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) SaveAgent(agent RegisteredAgent) error {
	d.lock[agent.ID].Lock()
	defer d.lock[agent.ID].Unlock()

	byt, err := json.Marshal(agent)
	if err != nil {
		return errors.Errorf("marshalling agent: %w", err)
	}

	return d.db.Update(func(tx *buntdb.Tx) error {
		_, replaced, err := tx.Set(agent.ID, string(byt), nil)
		if err != nil {
			return errors.Errorf("setting agent: %w", err)
		}
		if replaced {
			fmt.Printf("Agent %s updated\n", agent.ID)
		}
		return nil
	})
}

func (d *Database) GetAgent(id string) (RegisteredAgent, bool, error) {
	if _, ok := d.lock[id]; !ok {
		d.lock[id] = &sync.RWMutex{}
		// d.lock[id].Lock()
	}
	d.lock[id].RLock()
	defer d.lock[id].RUnlock()

	var agent *RegisteredAgent
	err := d.db.View(func(tx *buntdb.Tx) error {
		byt, err := tx.Get(id)
		if err != nil {
			if err == buntdb.ErrNotFound {
				return nil
			}
			return errors.Errorf("getting agent: %w", err)
		}
		return json.Unmarshal([]byte(byt), &agent)
	})
	if err != nil {
		return RegisteredAgent{}, false, errors.Errorf("viewing agent: %w", err)
	}
	if agent == nil {
		return RegisteredAgent{}, false, nil
	}
	return *agent, true, nil
}

func (d *Database) GetAllAgents() ([]RegisteredAgent, error) {
	for _, lock := range d.lock {
		lock.RLock()
	}
	defer func() {
		for _, lock := range d.lock {
			lock.RUnlock()
		}
	}()
	var agents []RegisteredAgent
	err := d.db.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("agent_id", func(key, value string) bool {
			var agent RegisteredAgent
			err := json.Unmarshal([]byte(value), &agent)
			if err != nil {
				return false
			}
			agents = append(agents, agent)
			if _, ok := d.lock[agent.ID]; !ok {
				d.lock[agent.ID] = &sync.RWMutex{}
				d.lock[agent.ID].Lock()
			}

			return true
		})
	})
	if err != nil {
		return nil, errors.Errorf("viewing agents: %w", err)
	}

	return agents, nil
}
