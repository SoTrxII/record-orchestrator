package services

type State struct {
	VcId  string
	R20Id string
}
type StateStore interface {
	Save(key string, value State) error
	Get(key string) (*State, error)
	Delete(key string) error
}
