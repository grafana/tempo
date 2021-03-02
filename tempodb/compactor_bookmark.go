package tempodb

import (
	"context"

	"github.com/grafana/tempo/tempodb/encoding"
)

type bookmark struct {
	iter encoding.Iterator

	currentID     []byte
	currentObject []byte
}

func newBookmark(iter encoding.Iterator) *bookmark {
	return &bookmark{
		iter: iter,
	}
}

func (b *bookmark) current(ctx context.Context) ([]byte, []byte, error) {
	if len(b.currentID) != 0 && len(b.currentObject) != 0 {
		return b.currentID, b.currentObject, nil
	}

	var err error
	b.currentID, b.currentObject, err = b.iter.Next(ctx)
	if err != nil {
		return nil, nil, err
	}

	return b.currentID, b.currentObject, nil
}

func (b *bookmark) done(ctx context.Context) bool {
	_, _, err := b.current(ctx)

	return err != nil
}

func (b *bookmark) clear() {
	b.currentID = nil
	b.currentObject = nil
}
