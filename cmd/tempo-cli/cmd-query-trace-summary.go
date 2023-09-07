package main

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/gogo/protobuf/jsonpb"

	v1resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

type queryTraceSummaryCmd struct {
	backendOptions

	TraceID  string `arg:"" help:"trace ID to retrieve"`
	TenantID string `arg:"" help:"tenant ID to search"`
}

func (cmd *queryTraceSummaryCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	id, err := util.HexStringToTraceID(cmd.TraceID)
	if err != nil {
		return err
	}

	traceSummary, err := queryBucketNotCombined(context.Background(), r, c, cmd.TenantID, id)
	if err != nil {
		return err
	}

	var (
		marshaller = new(jsonpb.Marshaler)
		jsonBytes  = bytes.Buffer{}
	)

	// jsonify rootspan
	err = marshaller.Marshal(&jsonBytes, traceSummary.RootSpan)
	if err != nil {
		fmt.Println("failed to marshal to json: ", err)
		return nil
	}

	fmt.Printf("Number of blocks: %d \n", traceSummary.NumBlock)
	fmt.Printf("Span count: %d \n", traceSummary.SpanCount)
	fmt.Printf("Trace size: %d MB \n", traceSummary.TraceSize)
	fmt.Printf("Trace duration: %d seconds \n", traceSummary.TraceDuration)
	fmt.Printf("Root service name: %s \n", traceSummary.RootServiceName)
	fmt.Println("Root span info:")
	fmt.Println(jsonBytes.String())
	fmt.Println("top frequent service.names: ")
	fmt.Println(traceSummary.ServiceNames)

	return nil
}

type TraceSummary struct {
	NumBlock         int
	SpanCount        int
	TraceSize        int
	TraceDuration    uint64
	RootServiceName  string
	RootSpan         *v1.Span
	RootSpanResource *v1resource.Resource
	ServiceNames     []string
}

func sortServiceNames(nameFrequencies map[string]int) PairList {
	pl := make(PairList, len(nameFrequencies))
	i := 0
	for k, v := range nameFrequencies {
		pl[i] = Pair{k, v}
		i++
	}
	sort.Sort(sort.Reverse(pl))
	return pl
}

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
