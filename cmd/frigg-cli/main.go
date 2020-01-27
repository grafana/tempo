package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/grafana/frigg/pkg/ingester/wal"
	"github.com/grafana/frigg/pkg/storage"
	"github.com/grafana/frigg/pkg/storage/trace_backend/local"
)

var (
	dir           string
	modeDumpIndex bool
	findTraceID   string
)

func init() {
	flag.StringVar(&dir, "dir", "", "dir to find chunks")
	flag.BoolVar(&modeDumpIndex, "dump-index", false, "dumps the index at the path")
	flag.StringVar(&findTraceID, "find-trace", "", "finds a trace by id.  expected to be a hex value")
}

func main() {
	flag.Parse()

	if len(dir) == 0 {
		fmt.Println("-dir is required")
		return
	}

	if !modeDumpIndex && len(findTraceID) == 0 {
		fmt.Println("One of -dump-index, -find-trace, ... is required")
		return
	}

	var err error

	if modeDumpIndex {
		err = dumpIndex(dir)
	}

	if len(findTraceID) > 0 {
		err = findTraceByID(dir, findTraceID)
	}

	if err != nil {
		fmt.Printf("%v", err)
	}
}

func findTraceByID(dir string, id string) error {
	byteID, err := hex.DecodeString(id)
	if err != nil {
		return err
	}

	size := len(byteID)
	if size < 16 {
		byteID = append(make([]byte, 16-size), byteID...)
	}

	reader, _, err := storage.NewTraceStore(storage.TraceConfig{
		Backend:                  "local",
		BloomFilterFalsePositive: .01,
		Local: local.Config{
			Path: dir,
		},
	})
	if err != nil {
		return err
	}

	start := time.Now()
	trace, metrics, err := reader.FindTrace("fake", byteID)
	elapsed := time.Since(start)
	if err != nil {
		return err
	}

	fmt.Println("------trace--------")
	if trace != nil {
		fmt.Printf("%+v", trace)
	} else {
		fmt.Println("trace not found")
	}
	fmt.Println("------metrics--------")
	fmt.Printf("%+v\n", metrics)
	fmt.Printf("This is meaningless but: %s", elapsed)

	return nil
}

func dumpIndex(dir string) error {

	bytes, err := ioutil.ReadFile(path.Join(dir, "index"))
	if err != nil {
		return err
	}

	records, err := wal.UnmarshalRecords(bytes)
	if err != nil {
		return err
	}

	fmt.Printf("records: %d\n", len(records))
	for i, r := range records {
		fmt.Printf("%4d : %v %v %v\n", i, r.ID, r.Start, r.Length)
	}

	return nil
}
