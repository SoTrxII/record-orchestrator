package utils

type DiscordRecorder interface {
	Start(vcId string) error
	Stop(vcId string) ([]string, error)
}

type R20Recorder interface {
	Start(r20Id string) error
	Stop(r20Id string) (string, error)
}
