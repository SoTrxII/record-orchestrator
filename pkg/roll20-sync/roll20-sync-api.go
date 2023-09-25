package roll20_sync

type R20Recorder interface {
	Start(r20Id string) error
	Stop(r20Id string) (string, error)
}
