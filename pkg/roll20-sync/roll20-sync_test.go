package roll20_sync

import (
	"context"
	"record-orchestrator/internal/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

type recordingInvoker struct{ method, body, reply string }

func (i *recordingInvoker) InvokeMethodWithContent(_ context.Context, _, method, _ string, c *utils.DataContent) ([]byte, error) {
	i.method, i.body = method, string(c.Data)
	return []byte(i.reply), nil
}

func TestStart_SendsTheDiscordStart(t *testing.T) {
	inv := &recordingInvoker{}
	assert.NoError(t, NewRoll20Sync(inv, "roll20-audio-sync").Start("123", 1712345678901))
	assert.Equal(t, "v1/jukeboxsyncer/start", inv.method)
	assert.JSONEq(t, `{"id":"123","alignTo":1712345678901}`, inv.body)
}

// The key is roll20-audio-sync's: .mkv when the tab's picture was recorded
func TestStop_ReturnsTheUploadedKey(t *testing.T) {
	for _, key := range []string{"123.ogg", "video/123.mkv"} {
		got, err := NewRoll20Sync(&recordingInvoker{reply: `{"key":"` + key + `"}`}, "roll20-audio-sync").Stop("123")
		assert.NoError(t, err)
		assert.Equal(t, key, got)
	}
}

func TestStop_FailsWithoutAKey(t *testing.T) {
	_, err := NewRoll20Sync(&recordingInvoker{reply: ""}, "roll20-audio-sync").Stop("123")
	assert.Error(t, err)
}
