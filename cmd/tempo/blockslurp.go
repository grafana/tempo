package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/pkg/model"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/encoding"
)

func blockSlurp(bucket string, blockID uuid.UUID, tenantID string, chunk uint32, slurpBuffer int) {
	ctx := context.Background()
	r, _, _, err := loadBackend(bucket)
	if err != nil {
		level.Error(log.Logger).Log("msg", "backend", "err", err)
		os.Exit(1)
	}

	meta, err := r.BlockMeta(ctx, blockID, tenantID)
	if err != nil {
		level.Error(log.Logger).Log("msg", "meta", "err", err)
		os.Exit(1)
	}

	block, err := encoding.NewBackendBlock(meta, r)
	if err != nil {
		level.Error(log.Logger).Log("msg", "block", "err", err)
		os.Exit(1)
	}

	iter, err := block.Iterator(chunk)
	if err != nil {
		level.Error(log.Logger).Log("msg", "iter", "err", err)
		os.Exit(1)
	}
	defer iter.Close()

	prefetchIter := encoding.NewPrefetchIterator(ctx, iter, slurpBuffer)
	defer prefetchIter.Close()

	time.Sleep(3 * time.Second)

	count := 0
	start := time.Now()
	prev := start
	for {
		_, obj, err := prefetchIter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			level.Error(log.Logger).Log("msg", "next", "err", err)
			os.Exit(1)
		}

		// marshal
		trace, err := model.Unmarshal(obj, meta.DataEncoding)
		if err != nil {
			level.Error(log.Logger).Log("msg", "unmarshal", "err", err)
			os.Exit(1)
		}

		// pretend to look for something
		for _, b := range trace.Batches {
			for _, ils := range b.InstrumentationLibrarySpans {
				for _, s := range ils.Spans {
					for _, a := range s.Attributes {
						if a.Key == "asdfasdfasdf" && a.Value.GetStringValue() == "asdfasdfs" {
							fmt.Println("this exists?")
						}
					}
				}
			}
		}

		count++
		if count%1000 == 0 {
			fmt.Println("count", count, time.Since(prev))
			prev = time.Now()
		}
	}

	fmt.Println("meta:", meta)
	fmt.Println("count:", count)
	fmt.Println("time:", time.Since(start))
}

func blockDeslurp(bucket string, blockID uuid.UUID, tenantID string, chunk uint32) {
	ctx := context.Background()
	_, w, _, err := loadBackend(bucket)
	if err != nil {
		level.Error(log.Logger).Log("msg", "backend", "err", err)
		os.Exit(1)
	}

	meta := backend.NewBlockMeta(tenantID, blockID, "v2", backend.EncNone, "blerg")

	block, err := encoding.NewStreamingBlock(&encoding.BlockConfig{
		IndexDownsampleBytes: 1024 * 1024,
		BloomFP:              .05,
		Encoding:             backend.EncNone,
		IndexPageSizeBytes:   1024,
		BloomShardSizeBytes:  100000,
	}, blockID, tenantID, []*backend.BlockMeta{meta}, 10)
	if err != nil {
		level.Error(log.Logger).Log("msg", "block", "err", err)
		os.Exit(1)
	}

	for i := 0; i < 10; i++ {
		id := make([]byte, 16)
		obj := make([]byte, 10)

		rand.Read(id)
		rand.Read(obj)

		err = block.AddObject(id, obj)
		if err != nil {
			level.Error(log.Logger).Log("msg", "add", "err", err)
			os.Exit(1)
		}
	}

	var tracker backend.AppendTracker
	tracker, _, err = block.FlushBuffer(ctx, tracker, w)
	if err != nil {
		level.Error(log.Logger).Log("msg", "flush", "err", err)
		os.Exit(1)
	}
	_, err = block.Complete(ctx, tracker, w)
	if err != nil {
		level.Error(log.Logger).Log("msg", "complete", "err", err)
		os.Exit(1)
	}
}

func loadBackend(bucket string) (backend.Reader, backend.Writer, backend.Compactor, error) {
	// Defaults
	cfg := app.Config{}
	cfg.RegisterFlagsAndApplyDefaults("", &flag.FlagSet{})

	cfg.StorageConfig.Trace.Backend = "gcs"
	cfg.StorageConfig.Trace.GCS.BucketName = bucket

	r, w, c, err := gcs.New(cfg.StorageConfig.Trace.GCS)
	if err != nil {
		return nil, nil, nil, err
	}

	return backend.NewReader(r), backend.NewWriter(w), c, nil
}
