package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/util"
)

type queryBlocksSummaryCmd struct {
	backendOptions

	TraceID  string `arg:"" help:"trace ID to retrieve"`
	TenantID string `arg:"" help:"tenant ID to search"`
}

func (cmd *queryBlocksSummaryCmd) Run(ctx *globalOptions) error {
	r, _, c, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	id, err := util.HexStringToTraceID(cmd.TraceID)
	if err != nil {
		return err
	}

	results, err := queryBucket(context.Background(), r, c, cmd.TenantID, id)
	if err != nil {
		return err
	}

	var combiner = trace.NewCombiner()

	fmt.Println()
	for i, result := range results {
		combiner.ConsumeWithFinal(result.trace, i == len(results)-1)
	}

	combinedTrace, _ := combiner.Result()
	size := 0
	spanCount := 0
	firstStartTime := combinedTrace.Batches[0].ScopeSpans[0].Spans[0].StartTimeUnixNano
	lastEndTime := combinedTrace.Batches[0].ScopeSpans[0].Spans[0].EndTimeUnixNano
	rootSpan := combinedTrace.Batches[0].ScopeSpans[0].Spans[0]
	rootSpanResource := combinedTrace.Batches[0].Resource
	rootServiceName := ""
	serviceNameMap := make(map[string]int)

	for _, b := range combinedTrace.Batches {
		size += b.Size()
		for _, attr := range b.Resource.Attributes {
			if "service.name" == attr.Key {
				serviceNameMap[attr.Value.GetStringValue()]++
			}
		}
		for _, scope := range b.ScopeSpans {
			spanCount += len(scope.Spans)
			for _, span := range scope.Spans {
				if span.StartTimeUnixNano < firstStartTime {
					firstStartTime = span.StartTimeUnixNano
				}
				if span.EndTimeUnixNano > lastEndTime {
					lastEndTime = span.EndTimeUnixNano
				}
				if span.ParentSpanId == nil {
					rootSpan = span
					rootSpanResource = b.Resource
				}
			}
		}
	}

	for _, attr := range rootSpanResource.Attributes {
		if "service.name" == attr.Key {
			rootServiceName = attr.Value.GetStringValue()
		}
	}

	// get top 5 most frequent service names
	topTenSortedPL := sortServiceNames(serviceNameMap)
	topTenServiceName := make([]string, 10)
	length := len(topTenSortedPL)
	if length > 10 {
		length = 10
	}
	for index := 0; index < length; index++ {
		topTenServiceName[index] = topTenSortedPL[index].Key
	}

	duration := lastEndTime - firstStartTime
	durationSecond := duration / 1000000000

	fmt.Printf("Number of blocks: %d \n", len(results))
	fmt.Printf("Span count: %d \n", spanCount)
	fmt.Printf("Trace size: %d MB \n", size/1000000)
	fmt.Printf("Trace duration: %d seconds \n", durationSecond)
	fmt.Printf("Root service name: %s \n", rootServiceName)
	fmt.Println("Root span info:")
	fmt.Println(rootSpan)
	fmt.Println("top 5 frequent service.names: ")
	fmt.Println(topTenServiceName)

	return nil
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
