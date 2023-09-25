//go:build integration
// +build integration

package memory

import (
	"context"
	"github.com/dapr/go-sdk/client"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

const (
	DEFAULT_STATE_STORE_ID = "object-store"
	DEFAULT_DAPR_PORT      = "50011"
)

var (
	store *Memory[state]
)

func beforeAll() {
	daprClient, err := client.NewClientWithPort(DEFAULT_DAPR_PORT)
	if err != nil {
		log.Fatal(err)
	}
	store = NewMemory[state](daprClient, DEFAULT_STATE_STORE_ID)
}

func teardown() {
	daprClient, err := client.NewClientWithPort(DEFAULT_DAPR_PORT)
	if err != nil {
		log.Fatal(err)
	}
	err = daprClient.DeleteState(context.Background(), DEFAULT_STATE_STORE_ID, "test", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func TestMemory_GetStateOk(t *testing.T) {
	// State store
	defer teardown()
	err := store.Save("test", state{Val: "test"})
	assert.NoError(t, err)
	state, err := store.Get("test")
	assert.NoError(t, err)
	assert.Equal(t, "test", state.Val)
}

func TestMemory_GetNoState(t *testing.T) {
	defer teardown()
	state, err := store.Get("test")
	assert.NoError(t, err)
	assert.Nil(t, state)
}

func TestMemory_SaveState(t *testing.T) {
	defer teardown()
	err := store.Save("test", state{Val: "test"})
	assert.NoError(t, err)
}

func TestMain(m *testing.M) {
	beforeAll()
	m.Run()
}

type state struct {
	Val string
}
