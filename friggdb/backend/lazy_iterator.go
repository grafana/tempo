package backend

import (
	"github.com/google/uuid"
)

type lazyIterator struct {
}

func NewLazyIterator(tenantID string, blockID uuid.UUID, reader Reader) Iterator {
	return nil
}
