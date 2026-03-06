// Package bra implements the branch rewriting filter for binaries.
package bra

type converter interface {
	Size() int
	Convert(b []byte, encoding bool) int
}
