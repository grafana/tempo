//go:build !unix

package main

func peakRSSBytes() uint64 { return 0 }
