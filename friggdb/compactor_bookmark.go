package friggdb

import "github.com/google/uuid"

type bookmark struct {
	id       uuid.UUID
	location uint64
	index    []byte
	objects  []byte

	currentID     []byte
	currentObject []byte
}

func (b *bookmark) done() bool {
	return len(b.index) == 0 && len(b.objects) == 0 && len(b.currentID) == 0 && len(b.currentObject) == 0
}

func (b *bookmark) clearObject() {
	b.currentID = nil
	b.currentObject = nil
}

func allDone(bookmarks []*bookmark) bool {
	for _, b := range bookmarks {
		if !b.done() {
			return false
		}
	}

	return true
}
