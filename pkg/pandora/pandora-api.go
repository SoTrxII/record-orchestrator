package pandora

type DiscordRecorder interface {
	Start(vcId string) error
	Stop(vcId string) ([]string, error)
}

// Topics used to talk to Pandora, following a request/reply pattern.
// The P_* topics are the ones we publish on and Pandora subscribes to.
// The S_* topics are the ones Pandora replies on and we subscribe to.
// These names must stay in sync with Pandora's PubSubBroker.TOPICS.
const (
	P_Start = "startRecordingDiscord"
	P_End   = "stopRecordingDiscord"

	S_Started = "startedRecordingDiscord"
	S_Ended   = "stoppedRecordingDiscord"
)

type StartPandoraRequest struct {
	VoiceChannelId string `json:"voiceChannelId"`
}

type StartPandoraReply struct {
	VoiceChannelId string `json:"voiceChannelId"`
}

type StopPandoraRequest struct {
	VoiceChannelId string `json:"voiceChannelId"`
}

type StopPandoraReply struct {
	Ids []string `json:"ids"`
}

type PandoraReply struct {
	Started *StartPandoraReply
	Stopped *StopPandoraReply
	Error   error
}
