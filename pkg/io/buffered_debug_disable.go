//go:build !debug

package io

type bufferedReadStats struct {
}

func recordStart(len int, offset int64) *bufferedReadStats {
	return nil
}

func recordReaderLockAcquired(s *bufferedReadStats) {}

func recordBufFound(s *bufferedReadStats, buf *readerBuffer) {}

func recordBufLock(s *bufferedReadStats) {}

func recordDone(s *bufferedReadStats) {}

func dumpStats(s *bufferedReadStats) {}
