package es

// IndexInterface note index info interface define.
type IndexInterface interface {
	FetchIndex() ([]NoteIndex, error)
	Name() string
}
