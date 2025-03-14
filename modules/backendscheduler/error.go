package backendscheduler

import "errors"

var (
	ErrFlushFailed = errors.New("failed to flush cache to store")
	ErrNoJobsFound = errors.New("no jobs found")
)
