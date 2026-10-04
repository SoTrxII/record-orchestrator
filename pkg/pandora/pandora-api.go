package pandora

type DiscordRecorder interface {
	Start(instanceId, vcId string) error
	Stop(instanceId, vcId string) (Recording, error)
}

// Recording is what a stopped recording session leaves behind
type Recording struct {
	// Keys of the records made. There are several when Pandora had to
	// reconnect during the session
	Ids []string
	// Discord ids of everyone heard in these records
	Participants []string
}

// Topics used to talk to Pandora, following a request/reply pattern.
// The P_* topics are the ones we publish on and Pandora subscribes to. They
// are suffixed with an instance id to address one instance of the pool, see
// RequestTopic.
// The S_* topics are the ones Pandora replies on and we subscribe to. They are
// shared by the whole pool : replies are told apart by correlation id.
// These names must stay in sync with Pandora's PubSubBroker.TOPICS.
const (
	P_Start = "startRecordingDiscord"
	P_End   = "stopRecordingDiscord"

	S_Started = "startedRecordingDiscord"
	S_Ended   = "stoppedRecordingDiscord"
)

// Every request carries a correlation id that Pandora echoes back on its
// reply, so several recording sessions can be in flight without their replies
// getting mixed up. Pandora keeps it in the controller state, which means it
// survives a disaster recovery restart.
type StartPandoraRequest struct {
	VoiceChannelId string `json:"voiceChannelId"`
	CorrelationId  string `json:"correlationId"`
}

type StartPandoraReply struct {
	VoiceChannelId string `json:"voiceChannelId"`
	CorrelationId  string `json:"correlationId"`
}

type StopPandoraRequest struct {
	VoiceChannelId string `json:"voiceChannelId"`
	CorrelationId  string `json:"correlationId"`
}

type StopPandoraReply struct {
	Ids []string `json:"ids"`
	// Absent from the replies of a Pandora older than 3.2.0, which reads as
	// "nobody" rather than as an error
	Participants  []string `json:"participants"`
	CorrelationId string   `json:"correlationId"`
}

type PandoraReply struct {
	Started *StartPandoraReply
	Stopped *StopPandoraReply
	Error   error
}
