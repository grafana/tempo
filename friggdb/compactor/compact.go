package compactor

import "github.com/google/uuid"

type Config struct {
	blocksAtOnce
}

type Compactor struct {
	blockList []blocks
}

func blocksToCompact() []uuid.UUID {

}

func compact(ids []uuid.UUID) {

}
