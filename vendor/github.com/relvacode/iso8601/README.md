A fast ISO8601 date parser for Go

[![GoDoc](https://godoc.org/github.com/relvacode/iso8601?status.svg)](https://godoc.org/github.com/relvacode/iso8601) ![Build Status](https://github.com/relvacode/iso8601/actions/workflows/verify.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/relvacode/iso8601)](https://goreportcard.com/report/github.com/relvacode/iso8601)


```
go get github.com/relvacode/iso8601
```

The built-in RFC3333 time layout in Go is too restrictive to support any ISO8601 date-time.

This library parses any ISO8601 date into a native Go time object without regular expressions.

## Usage

```go
package main

import "github.com/relvacode/iso8601"

// iso8601.Time can be used as a drop-in replacement for time.Time with JSON responses
type ExternalAPIResponse struct {
	Timestamp *iso8601.Time
}


func main() {
	// iso8601.ParseString can also be called directly
	t, err := iso8601.ParseString("2020-01-02T16:20:00")
}
```

## Benchmark

```
goos: linux
goarch: amd64
pkg: github.com/relvacode/iso8601
cpu: AMD Ryzen 7 7840U w/ Radeon 780M Graphics      
BenchmarkParse-16               35880919                30.89 ns/op            0 B/op          0 allocs/op
```
