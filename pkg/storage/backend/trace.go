package backend

type TraceWriter interface {
	Write(bIndex []byte, bTraces []byte, bBloom []byte) error
}

type TraceReader interface {
	BatchNames() ([]string, error)
	Bloom(name string) ([]byte, error)
	Index(name string) ([]byte, error)
	Trace(name string, start uint64, length uint32) ([]byte, error)
}
