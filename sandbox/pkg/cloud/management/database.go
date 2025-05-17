package management

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gitlab.com/tozd/go/errors"
)

type Database struct {
	filePath string
	agents   map[string]RegisteredAgent
	lock     sync.RWMutex
}

func NewDatabase(filePath string) (*Database, error) {
	db := &Database{
		filePath: filePath,
		agents:   make(map[string]RegisteredAgent),
		lock:     sync.RWMutex{},
	}

	// If file exists, load the data
	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, errors.Errorf("reading database file: %w", err)
		}

		if len(data) > 0 {
			if err := json.Unmarshal(data, &db.agents); err != nil {
				return nil, errors.Errorf("unmarshalling database: %w", err)
			}
		}
	} else {
		fmt.Printf("Database file does not exist, creating new database\n")
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return nil, errors.Errorf("creating database directory: %w", err)
		}
		if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
			return nil, errors.Errorf("writing database file: %w", err)
		}
	}

	return db, nil
}

func (d *Database) Close() error {
	// Ensure data is saved before closing
	d.lock.RLock()
	defer d.lock.RUnlock()

	data, err := json.MarshalIndent(d.agents, "", "  ")
	if err != nil {
		return errors.Errorf("marshalling database: %w", err)
	}

	if err := os.WriteFile(d.filePath, data, 0644); err != nil {
		return errors.Errorf("writing database file: %w", err)
	}

	return nil
}

func (d *Database) SaveAgent(agent RegisteredAgent) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	// Check if we're updating an existing agent
	_, exists := d.agents[agent.ID]

	// Update in memory
	d.agents[agent.ID] = agent

	// Persist to file
	data, err := json.MarshalIndent(d.agents, "", "  ")
	if err != nil {
		return errors.Errorf("marshalling database: %w", err)
	}

	if err := os.WriteFile(d.filePath, data, 0644); err != nil {
		return errors.Errorf("writing database file: %w", err)
	}

	// We could log this information if needed
	_ = exists // Using exists to avoid linter error, would be useful for logging

	return nil
}

func (d *Database) GetAgent(id string) (RegisteredAgent, bool, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	agent, exists := d.agents[id]
	return agent, exists, nil
}

func (d *Database) GetAllAgents() ([]RegisteredAgent, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	agents := make([]RegisteredAgent, 0, len(d.agents))
	for _, agent := range d.agents {
		agents = append(agents, agent)
	}

	return agents, nil
}
