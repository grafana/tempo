package vparquet

import "context"

type Iterator interface {
	Next(context.Context) (*Trace, error)
	Close()
}
