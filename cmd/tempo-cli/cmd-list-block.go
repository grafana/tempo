package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/pkg/tempopb"
	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	"github.com/grafana/tempo/pkg/util"
	tempodb_backend "github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type valueStats struct {
	count int
}
type values struct {
	all   map[string]valueStats
	key   string
	count int
}
type kvPairs map[string]values

type listBlockCmd struct {
	backendOptions

	TenantID string `arg:"" help:"tenant-id within the bucket"`
	BlockID  string `arg:"" help:"block ID to list"`
	Scan     bool   `help:"scan contents of block for duplicate trace IDs and other info (warning, can be intense)"`
}

func (cmd *listBlockCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	return dumpBlock(r, c, cmd.TenantID, time.Hour, cmd.BlockID, cmd.Scan)
}

func dumpBlock(r tempodb_backend.Reader, c tempodb_backend.Compactor, tenantID string, windowRange time.Duration, blockID string, scan bool) error {
	id := uuid.MustParse(blockID)

	meta, err := r.BlockMeta(context.TODO(), id, tenantID)
	if err != nil && err != tempodb_backend.ErrDoesNotExist {
		return err
	}

	compactedMeta, err := c.CompactedBlockMeta(id, tenantID)
	if err != nil && err != tempodb_backend.ErrDoesNotExist {
		return err
	}

	if meta == nil && compactedMeta == nil {
		fmt.Println("Unable to load any meta for block", blockID)
		return nil
	}

	unifiedMeta := getMeta(meta, compactedMeta, windowRange)

	fmt.Println("ID            : ", unifiedMeta.BlockID)
	fmt.Println("Version       : ", unifiedMeta.Version)
	fmt.Println("Total Objects : ", unifiedMeta.TotalObjects)
	fmt.Println("Data Size     : ", humanize.Bytes(unifiedMeta.Size))
	fmt.Println("Encoding      : ", unifiedMeta.Encoding)
	fmt.Println("Level         : ", unifiedMeta.CompactionLevel)
	fmt.Println("Window        : ", unifiedMeta.window)
	fmt.Println("Start         : ", unifiedMeta.StartTime)
	fmt.Println("End           : ", unifiedMeta.EndTime)
	fmt.Println("Duration      : ", fmt.Sprint(unifiedMeta.EndTime.Sub(unifiedMeta.StartTime).Round(time.Second)))
	fmt.Println("Age           : ", fmt.Sprint(time.Since(unifiedMeta.EndTime).Round(time.Second)))

	if scan {
		fmt.Println("Scanning block contents.  Press CRTL+C to quit ...")

		block, err := encoding.NewBackendBlock(&unifiedMeta.BlockMeta, r)
		if err != nil {
			return err
		}

		iter, err := block.Iterator(uint32(2 * 1024 * 1024))
		if err != nil {
			return err
		}
		defer iter.Close()

		// Scanning stats
		i := 0
		dupe := 0
		maxObjSize := 0
		minObjSize := 0
		maxObjID := common.ID{}
		minObjID := common.ID{}

		allKVP := kvPairs{}
		printStats := func() {
			fmt.Println()
			fmt.Println("Scanning results:")
			fmt.Println("Objects scanned : ", i)
			fmt.Println("Duplicates      : ", dupe)
			fmt.Println("Smallest object : ", humanize.Bytes(uint64(minObjSize)), " : ", util.TraceIDToHexString(minObjID))
			fmt.Println("Largest object  : ", humanize.Bytes(uint64(maxObjSize)), " : ", util.TraceIDToHexString(maxObjID))
			fmt.Println("")
			printKVPairs(allKVP)
		}

		// Print stats on ctrl+c
		c := make(chan os.Signal)
		// nolint:govet
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			printStats()
			os.Exit(0)
		}()

		ctx := context.Background()
		prevID := make([]byte, 16)
		for {
			objID, obj, err := iter.Next(ctx)
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			if len(obj) > maxObjSize {
				maxObjSize = len(obj)
				maxObjID = objID
			}

			if len(obj) < minObjSize || minObjSize == 0 {
				minObjSize = len(obj)
				minObjID = objID
			}

			if bytes.Equal(objID, prevID) {
				dupe++
			}

			copy(prevID, objID)

			trace, err := model.MustNewDecoder(meta.DataEncoding).PrepareForRead(obj)
			if err != nil {
				return err
			}
			kvp := extractKVPairs(trace)
			for k, vs := range kvp {
				addKey(allKVP, k, 1)
				for v := range vs.all {
					addVal(allKVP, k, v, 1)
				}
			}

			i++
			if i%100000 == 0 {
				fmt.Println("Record: ", i)
			}
		}

		printStats()
	}

	return nil
}

// helper methods for calculating label stats
func printKVPairs(kvp kvPairs) {
	allValues := make([]values, 0, len(kvp))
	for _, vs := range kvp {
		allValues = append(allValues, vs)
	}
	sort.Slice(allValues, func(i, j int) bool {
		return relativeValue(allValues[i]) > relativeValue(allValues[j])
	})
	for _, vs := range allValues {
		fmt.Println("key:", vs.key, "count:", vs.count, "len:", len(vs.all), "value:", relativeValue(vs))
		for a, c := range vs.all {
			fmt.Printf(" %s:\t%.2f\n", a, float64(c.count)/float64(vs.count))
		}
	}
}

// attempts to calculate the "value" that storing a given label would provide by. currently (number of times appeared)^2 / cardinality
//  this is not researched and could definitely be improved
func relativeValue(v values) float64 {
	return (float64(v.count) * float64(v.count)) / float64(len(v.all))
}
func extractKVPairs(t *tempopb.Trace) kvPairs {
	kvp := kvPairs{}
	for _, b := range t.Batches {
		spanCount := 0
		for _, ils := range b.InstrumentationLibrarySpans {
			for _, s := range ils.Spans {
				spanCount++
				for _, a := range s.Attributes {
					val, yay := stringVal(a.Value)
					if !yay {
						continue
					}
					addKey(kvp, a.Key, 1)
					addVal(kvp, a.Key, val, 1)
				}
			}
		}
		for _, a := range b.Resource.Attributes {
			val, yay := stringVal(a.Value)
			if !yay {
				continue
			}
			addKey(kvp, a.Key, spanCount)
			addVal(kvp, a.Key, val, spanCount)
		}
	}
	return kvp
}
func addKey(kvp kvPairs, key string, count int) {
	v, ok := kvp[key]
	if !ok {
		v = values{
			all: map[string]valueStats{},
			key: key,
		}
	}
	v.count += count
	kvp[key] = v
}
func addVal(kvp kvPairs, key string, val string, count int) {
	v := kvp[key]
	stats, ok := v.all[val]
	if !ok {
		stats = valueStats{
			count: 0,
		}
	}
	stats.count += count
	v.all[val] = stats
	kvp[key] = v
}
func stringVal(v *v1.AnyValue) (string, bool) {
	if sVal, ok := v.Value.(*v1.AnyValue_StringValue); ok {
		return sVal.StringValue, true
	}
	if nVal, ok := v.Value.(*v1.AnyValue_IntValue); ok {
		return strconv.FormatInt(nVal.IntValue, 10), true
	}
	if dVal, ok := v.Value.(*v1.AnyValue_DoubleValue); ok {
		return fmt.Sprintf("%f", dVal.DoubleValue), true
		// strconv.FormatFloat()
	}
	if bVal, ok := v.Value.(*v1.AnyValue_BoolValue); ok {
		if bVal.BoolValue {
			return "true", true
		}
		return "false", true
	}
	// todo? add support for these?
	// if kVal, ok := v.Value.(*v1.AnyValue_KvlistValue); ok {
	//  return fmt.Sprintf("kvval %v", kVal.KvlistValue) // better way?
	// }
	// if aVal, ok := v.Value.(*v1.AnyValue_ArrayValue); ok {
	//  return fmt.Sprintf("arrayval %v", aVal.ArrayValue) // better way?
	// }
	return "", false
}
