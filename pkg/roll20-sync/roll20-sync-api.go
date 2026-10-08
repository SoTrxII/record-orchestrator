package roll20_sync

type R20Recorder interface {
	// alignTo is the unix ms the Discord recording started at, so the jukebox
	// audio can be placed on its timeline
	Start(r20Id string, alignTo int64) error
	Stop(r20Id string) (string, error)
}
