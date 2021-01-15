package index

type ID []byte

type Record struct {
	ID     ID
	Start  uint64
	Length uint32
}
