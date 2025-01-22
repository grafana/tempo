package generator

import (
	"context"
	"errors"

	"github.com/go-kit/log/level"
	"github.com/grafana/tempo/pkg/ingest"
	"github.com/grafana/tempo/pkg/tempopb"
)

func (g *Generator) startKafka() {
	// Create context that will be used to stop the goroutines.
	var ctx context.Context
	ctx, g.kafkaStop = context.WithCancel(context.Background())

	g.kafkaWG.Add(1)
	go g.listenKafka(ctx)
}

func (g *Generator) stopKafka() {
	g.kafkaStop()
	g.kafkaWG.Wait()
}

func (g *Generator) listenKafka(ctx context.Context) {
	defer g.kafkaWG.Done()

	level.Info(g.logger).Log("msg", "generator now listening to kafka")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if g.readOnly.Load() {
				// Starting up or shutting down
				continue
			}
			err := g.readKafka(ctx)
			if err != nil {
				level.Error(g.logger).Log("msg", "readKafka failed", "err", err)
				continue
			}
		}
	}
}

func (g *Generator) readKafka(ctx context.Context) error {
	d := ingest.NewDecoder()

	fetches := g.kafkaClient.PollFetches(ctx)
	fetches.EachError(func(_ string, _ int32, err error) {
		if !errors.Is(err, context.Canceled) {
			level.Error(g.logger).Log("msg", "failed to fetch records", "err", err)
		}
	})

	for iter := fetches.RecordIter(); !iter.Done(); {
		r := iter.Next()

		tenant := string(r.Key)

		i, err := g.getOrCreateInstance(tenant)
		if err != nil {
			return err
		}

		d.Reset()
		req, err := d.Decode(r.Value)
		if err != nil {
			return err
		}

		for _, tr := range req.Traces {
			trace := &tempopb.Trace{}
			err = trace.Unmarshal(tr.Slice)
			if err != nil {
				return err
			}

			i.pushSpansFromQueue(ctx, &tempopb.PushSpansRequest{
				Batches: trace.ResourceSpans,
			})
		}
	}

	return nil
}
