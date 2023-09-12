package main

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/google/uuid"

	"github.com/grafana/tempo/pkg/boundedwaitgroup"
	v1resource "github.com/grafana/tempo/pkg/tempopb/resource/v1"
	v1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/common"
)

type queryTraceSummaryCmd struct {
	backendOptions

	TraceID  string `arg:"" help:"trace ID to retrieve"`
	TenantID string `arg:"" help:"tenant ID to search"`

	Percentage float32 `help:"percentage of blocks to scan e.g..1 for 10%"`
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

	traceSummary, err := queryBucketForSummary(context.Background(), cmd.Percentage, r, c, cmd.TenantID, id)
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
	fmt.Printf("Trace size: %d B \n", traceSummary.TraceSize)
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

func queryBucketForSummary(ctx context.Context, percentage float32, r backend.Reader, c backend.Compactor, tenantID string, traceID common.ID) (*TraceSummary, error) {
	blockIDs, err := r.Blocks(context.Background(), tenantID)
	if err != nil {
		return nil, err
	}

	if percentage > 0 {
		// shuffle
		rand.Shuffle(len(blockIDs), func(i, j int) { blockIDs[i], blockIDs[j] = blockIDs[j], blockIDs[i] })

		// get the first n%
		total := len(blockIDs)
		total = int(float32(total) * percentage)
		blockIDs = blockIDs[:total]
	}

	fmt.Println("total blocks to search: ", len(blockIDs))

	// Load in parallel
	wg := boundedwaitgroup.New(50)
	resultsCh := make(chan *queryResults)

	go func() {
		for blockNum, id := range blockIDs {
			wg.Add(1)

			go func(blockNum2 int, id2 uuid.UUID) {
				defer wg.Done()

				// search here
				q, err := queryBlock(ctx, r, c, blockNum2, id2, tenantID, traceID)
				if err != nil {
					fmt.Println("Error querying block:", err)
					return
				}

				if q != nil {
					resultsCh <- q
				}
			}(blockNum, id)
		}
	}()

	// cheap way to let the wait group get at least one .Add()
	time.Sleep(time.Second)

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var rootSpan *v1.Span
	var rootSpanResource *v1resource.Resource

	numBlock := 0
	size := 0
	spanCount := 0

	firstStartTime := uint64(math.MaxUint64)
	lastEndTime := uint64(0)

	rootServiceName := ""
	serviceNameMap := make(map[string]int)

	for q := range resultsCh {
		numBlock++
		for _, b := range q.trace.Batches {
			size += b.Size()
			for _, attr := range b.Resource.Attributes {
				if "service.name" == attr.Key {
					serviceNameMap[attr.Value.GetStringValue()]++
					break
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
					if len(span.ParentSpanId) == 0 {
						rootSpan = span
						rootSpanResource = b.Resource
					}
				}
			}
		}

		if rootSpanResource != nil {
			for _, attr := range rootSpanResource.Attributes {
				if attr.Key == "service.name" {
					rootServiceName = attr.Value.GetStringValue()
					break
				}
			}
		}
	}

	duration := lastEndTime - firstStartTime
	durationSecond := duration / 1000000000

	// get top 5 most frequent service names
	topFiveSortedPL := sortServiceNames(serviceNameMap)
	topFiveServiceName := make([]string, 5)
	length := len(topFiveSortedPL)
	if length > 5 {
		length = 5
	}
	for index := 0; index < length; index++ {
		topFiveServiceName[index] = topFiveSortedPL[index].Key
	}

	return &TraceSummary{
		NumBlock:         numBlock,
		SpanCount:        spanCount,
		TraceSize:        size,
		TraceDuration:    durationSecond,
		RootServiceName:  rootServiceName,
		RootSpan:         rootSpan,
		RootSpanResource: rootSpanResource,
		ServiceNames:     topFiveServiceName,
	}, nil
}

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
