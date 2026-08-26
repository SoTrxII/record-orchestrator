package memory

// Session is one recording in progress, and which instance of the Pandora
// pool is carrying it
type Session struct {
	VcId       string `json:"vcId"`
	R20Id      string `json:"r20Id"`
	InstanceId string `json:"instanceId"`
}

// State is the orchestrator's whole view of what is being recorded, keyed by
// voice channel id.
// It lives under a single key : the orchestrator runs as a single replica, so
// an in-process lock is enough to keep it consistent and the store is only
// there to survive a restart. Scaling the orchestrator out would mean moving
// this to a compare-and-swap on the state store's ETag.
type State struct {
	Sessions map[string]Session `json:"sessions"`
}

type StateStore interface {
	Save(key string, value State) error
	Get(key string) (*State, error)
	Delete(key string) error
}
