package roll20_sync

import (
	"context"
	"record-orchestrator/internal/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

type recordingInvoker struct{ method, body string }

func (i *recordingInvoker) InvokeMethodWithContent(_ context.Context, _, method, _ string, c *utils.DataContent) ([]byte, error) {
	i.method, i.body = method, string(c.Data)
	return nil, nil
}

func TestStart_SendsTheDiscordStart(t *testing.T) {
	inv := &recordingInvoker{}
	assert.NoError(t, NewRoll20Sync(inv, "roll20-audio-sync").Start("123", 1712345678901))
	assert.Equal(t, "v1/jukeboxsyncer/start", inv.method)
	assert.JSONEq(t, `{"id":"123","alignTo":1712345678901}`, inv.body)
}

func TestStop_ReturnsTheUploadedKey(t *testing.T) {
	key, err := NewRoll20Sync(&recordingInvoker{}, "roll20-audio-sync").Stop("123")
	assert.NoError(t, err)
	assert.Equal(t, "123.ogg", key)
}
