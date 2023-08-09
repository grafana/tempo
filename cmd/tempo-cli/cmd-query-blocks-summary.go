package main

import (

	"context"
	"fmt"




	"github.com/grafana/tempo/pkg/model/trace"

	//"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
	// "github.com/grafana/tempo/tempodb"
	// "github.com/grafana/tempo/tempodb/backend"
	// "github.com/grafana/tempo/tempodb/encoding"
	// "github.com/grafana/tempo/tempodb/encoding/common"
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

	var (
		combiner   = trace.NewCombiner()
		// marshaller = new(jsonpb.Marshaler)
		// jsonBytes  = bytes.Buffer{}
	)

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
	for _, attr := range rootSpanResource.Attributes {
		if "service.name" == attr.Key {
			rootServiceName = attr.Value.GetStringValue()
		}
	}

	for _, b := range combinedTrace.Batches {
		size += b.Size()
		for _, attr := range b.Resource.Attributes {
			if "service.name" == attr.Key {
				serviceNameMap[attr.Value.GetStringValue()] ++
			}
		}
		for _, scope := range b.ScopeSpans {
			spanCount += len(scope.Spans)
			for _, span := range scope.Spans {
				if span.EndTimeUnixNano > lastEndTime {
					lastEndTime = span.EndTimeUnixNano
				}
			}
		}
	}


	// get top 5 most frequent service names
	lowest := 0
	topFiveName := make([]string, 5)
	topFiveFreq := make([]int, 5)
	for name, freq := range serviceNameMap {
		if freq > lowest {
			for i := 3; i >= 0; i-- {
				if freq < topFiveFreq[i] || i == 0 {
					position := i + 1
					if freq > topFiveFreq[0] {
						position = 0
					}

					for y := 4; y >= position+1; y-- {
						topFiveFreq[y] = topFiveFreq[y-1]
						topFiveName[y] = topFiveName[y-1]
					}
					topFiveFreq[position] = freq
					topFiveName[position] = name
					break
				}
			}
			lowest = topFiveFreq[4]
		}
	}

	duration := lastEndTime - firstStartTime
	durationSecond := duration/1000000000

	fmt.Printf("Number of blocks: %d \n", len(results))
	fmt.Printf("Span count: %d \n", spanCount)
	fmt.Printf("Trace size: %d MB \n", size/1000000)
	fmt.Printf("Trace duration: %d seconds \n", durationSecond)
	fmt.Printf("Root service name: %s \n", rootServiceName)
	fmt.Println("Root span info:")
	fmt.Println(rootSpan)
	fmt.Println("top 5 frequent service.names: ")
	fmt.Println(topFiveName)

	fmt.Println("combined:")
	// err = marshaller.Marshal(&jsonBytes, combinedTrace)
	// if err != nil {
	// 	fmt.Println("failed to marshal to json: ", err)
	// 	return nil
	// }
	// fmt.Println(jsonBytes.String())
	return nil
}