package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/joe-elliott/frigg/pkg/storage"
)

var (
	dir           string
	modeDumpIndex bool
)

func init() {
	flag.StringVar(&dir, "dir", "", "dir to find chunks")
	flag.BoolVar(&modeDumpIndex, "dump-index", false, "dumps the index at the path")
}

func main() {
	flag.Parse()

	if len(dir) == 0 {
		fmt.Println("-dir is required")
		return
	}

	if !modeDumpIndex {
		fmt.Println("One of -dump-index ... is required")
		return
	}

	var err error

	if modeDumpIndex {
		err = dumpIndex(dir)
	}

	if err != nil {
		fmt.Printf("%v", err)
	}
}

func dumpIndex(dir string) error {

	bytes, err := ioutil.ReadFile(path.Join(dir, "index"))
	if err != nil {
		return err
	}

	records, err := storage.DecodeRecords(bytes)
	if err != nil {
		return err
	}

	fmt.Println("records: %d", len(records))
	for i, r := range records {
		fmt.Printf("%4d : %v %v %v\n", i, r.TraceID, r.Start, r.Length)
	}

	return nil
}
