package friggdb

import (
	"github.com/google/uuid"
	"github.com/grafana/frigg/friggdb/backend"
)

type Config struct {
	BlocksAtOnce int `yaml:"blocks-per"`
	BytesAtOnce  int `yaml:"bytes-per"`
}

type Compactor struct {
	cfg *Config

	r backend.Reader
}

func New(cfg *Config, r backend.Reader) *Compactor {
	return &Compactor{
		cfg: cfg,
		r:   r,
	}
}

func (c *Compactor) blocksToCompact(tenantID string) []uuid.UUID {
	return nil
}

func (c *Compactor) compact(ids []uuid.UUID, tenantID string) error {
	bookmarks := make([]*bookmark, 0, len(ids))

	for _, id := range ids {
		index, err := c.r.Index(id, tenantID)
		if err != nil {
			return err
		}

		bookmarks = append(bookmarks, &bookmark{
			id:    id,
			index: index,
		})
	}
}

type bookmark struct {
	id       uuid.UUID
	location uint64
	index    []byte
	objects  []byte
}

func (b *bookmark) newBookmark(id uuid.UUID) (*bookmark, error) {
	return &bookmark{
		id: id,
	}
}

func (b *bookmark) done() bool {
	return len(index) == 0 && len(objects) == 0
}

func (b *bookmark) nextObject() ([]byte, []byte, error) {
	if b.done() {

	}

}
