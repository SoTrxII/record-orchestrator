package memory

import (
	"context"
	"encoding/json"
	"record-orchestrator/internal/utils"
	"sync"
)

type Memory[S interface{}] struct {
	client    utils.StateSaver
	component string
	mu        sync.Mutex
}

func NewMemory[S interface{}](client utils.StateSaver, component string) *Memory[S] {
	return &Memory[S]{client: client, component: component}
}

func (m *Memory[S]) Save(key string, value S) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.client.SaveState(context.Background(), m.component, key, bytes, map[string]string{})
}

func (m *Memory[S]) Get(key string) (*S, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, err := m.client.GetState(context.Background(), m.component, key, map[string]string{})
	if err != nil {
		return nil, err
	}
	if item.Value == nil {
		return nil, nil
	}
	var state S
	err = json.Unmarshal(item.Value, &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (m *Memory[S]) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.client.DeleteState(context.Background(), m.component, key, map[string]string{})
}
