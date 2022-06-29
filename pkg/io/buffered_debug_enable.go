//go:build debug

package io

import (
	"fmt"
	"time"

	"github.com/go-logfmt/logfmt"
	"go.uber.org/atomic"
)

var concurrent = atomic.NewInt32(0)
var duration = atomic.NewDuration(0)

type bufferedReadStats struct {
	len    int
	offset int64
	hit    bool

	start      time.Time
	readerLock time.Time
	bufFound   time.Time
	bufLock    time.Time
	done       time.Time
}

func recordStart(len int, offset int64) *bufferedReadStats {
	concurrent.Inc()

	return &bufferedReadStats{
		start:  time.Now(),
		len:    len,
		offset: offset,
	}
}

func recordReaderLockAcquired(s *bufferedReadStats) {
	s.readerLock = time.Now()
}

func recordBufFound(s *bufferedReadStats, buf *readerBuffer) {
	s.hit = buf != nil
	s.bufFound = time.Now()
}

func recordBufLock(s *bufferedReadStats) {
	s.bufLock = time.Now()
}

func recordDone(s *bufferedReadStats) {
	s.done = time.Now()
}

func dumpStats(s *bufferedReadStats) {
	all := s.done.Sub(s.bufLock)
	dur := duration.Add(all)

	b, err := logfmt.MarshalKeyvals("len", s.len,
		"off", s.offset,
		"hit", s.hit,
		"lock1", s.readerLock.Sub(s.start),
		"buf", s.bufFound.Sub(s.readerLock),
		"lock2", s.bufLock.Sub(s.bufFound),
		"all", all,
		"dur", dur,
		"conc", concurrent.Load())
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	concurrent.Dec()
}
