package index

type Iterator interface {
	Next() (ID, []byte, error)
}

type Finder interface {
	Find(id ID) ([]byte, error)
}

type Appender interface {
	Append(ID, []byte) error
	Complete()
	Records() []*Record
	Length() int
}
