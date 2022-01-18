package store

type Callback func(e *Edge)

type Store interface {
	UpsertEdge(string, Callback) (*Edge, error)
	Expire()
}
