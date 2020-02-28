package compactor

import "github.com/grafana/frigg/friggdb/backend"

type bookmark struct {
	iter backend.Iterator

	currentID     []byte
	currentObject []byte
}

func newBookmark(iter backend.Iterator) *bookmark {
	return &bookmark{
		iter: iter,
	}
}

func (b *bookmark) current() ([]byte, []byte, error) {
	if len(b.currentID) != 0 && len(b.currentObject) != 0 {
		return b.currentID, b.currentObject, nil
	}

	var err error
	b.currentID, b.currentObject, err = b.iter.Next()
	if err != nil {
		return nil, nil, err
	}

	return b.currentID, b.currentObject, nil
}

func (b *bookmark) done() bool {
	_, _, err := b.current()

	return err != nil
}

func (b *bookmark) clear() {
	b.currentID = nil
	b.currentObject = nil
}
