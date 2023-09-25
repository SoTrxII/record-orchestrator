package pandora

type DiscordRecorder interface {
	Start(vcId string) error
	Stop(vcId string) ([]string, error)
}

type topics string

const (
	P_Start   topics = "startRecordingDiscord"
	S_Started        = "startRecordingDiscord"
	P_End            = "stopRecordingDiscord"
	S_Ended          = "stoppedRecordingDiscord"
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
