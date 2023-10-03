// This code was forked from Loki.
// https://github.com/grafana/loki/tree/caf7dde32be8b085549273fc6c238e3b6b656a3a/pkg/usagestats
package usagestats

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/multierror"
	"github.com/grafana/dskit/services"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/tempo/cmd/tempo/build"
	"github.com/grafana/tempo/tempodb/backend"
)

const (
	// attemptNumber how many times we will try to read a corrupted cluster seed before deleting it.
	attemptNumber = 4
	// seedKey is the key for the cluster seed to use with the kv store.
	seedKey = "usagestats_token"
)

var (
	reportCheckInterval = time.Minute
	reportInterval      = 4 * time.Hour

	stabilityCheckInterval   = 5 * time.Second
	stabilityMinimumRequired = 6
)

type Reporter struct {
	logger log.Logger
	reader backend.RawReader
	writer backend.RawWriter
	reg    prometheus.Registerer

	services.Service

	conf       Config
	kvConfig   kv.Config
	cluster    *ClusterSeed
	lastReport time.Time
}

func NewReporter(config Config, kvConfig kv.Config, reader backend.RawReader, writer backend.RawWriter, logger log.Logger, reg prometheus.Registerer) (*Reporter, error) {
	if !config.Enabled {
		return nil, nil
	}
	r := &Reporter{
		logger:   logger,
		reader:   reader,
		writer:   writer,
		conf:     config,
		kvConfig: kvConfig,
		reg:      reg,
	}
	r.Service = services.NewBasicService(nil, r.running, nil)
	return r, nil
}

func (rep *Reporter) initLeader(ctx context.Context) *ClusterSeed {
	kvClient, err := kv.NewClient(rep.kvConfig, JSONCodec, nil, rep.logger)
	if err != nil {
		level.Warn(rep.logger).Log("msg", "failed to create kv client", "err", err)
		return nil
	}
	// Try to become leader via the kv client
	backoff := backoff.New(ctx, rep.conf.Backoff)
	for backoff.Ongoing() {
		// create a new cluster seed
		seed := ClusterSeed{
			UID:               uuid.NewString(),
			PrometheusVersion: build.GetVersion(),
			CreatedAt:         time.Now(),
		}
		copySeed := seed
		if err := kvClient.CAS(ctx, seedKey, func(in interface{}) (out interface{}, retry bool, err error) {
			// The key is already set, so we don't need to do anything
			if in != nil {
				if kvSeed, ok := in.(*ClusterSeed); ok && kvSeed != nil && kvSeed.UID != copySeed.UID {
					copySeed = *kvSeed
					return nil, false, nil
				}
			}
			return &copySeed, true, nil
		}); err != nil {
			level.Warn(rep.logger).Log("msg", "failed to CAS cluster seed key", "err", err)
			continue
		}
		// ensure stability of the cluster seed
		stableSeed := ensureStableKey(ctx, kvClient, rep.logger)
		seed = *stableSeed
		// Fetch the remote cluster seed.
		remoteSeed, err := rep.fetchSeed(ctx,
			func(err error) bool {
				// we only want to retry if the error is not an object not found error or a bad see file error
				return !errors.Is(err, backend.ErrDoesNotExist) && !errors.Is(err, backend.ErrBadSeedFile)
			})
		if err != nil {
			if errors.Is(err, backend.ErrDoesNotExist) || errors.Is(err, backend.ErrBadSeedFile) {
				// we are the leader and we need to save the file.
				if err := rep.writeSeedFile(ctx, seed); err != nil {
					level.Warn(rep.logger).Log("msg", "failed to CAS cluster seed key", "err", err)
					backoff.Wait()
					continue
				}
				return &seed
			}
			backoff.Wait()
			continue
		}
		return remoteSeed
	}
	return nil
}

// ensureStableKey ensures that the cluster seed is stable for at least 30seconds.
// This is required when using gossiping kv client like memberlist which will never have the same seed
// but will converge eventually.
func ensureStableKey(ctx context.Context, kvClient kv.Client, logger log.Logger) *ClusterSeed {
	var (
		previous    *ClusterSeed
		stableCount int
	)
	for {
		time.Sleep(stabilityCheckInterval)
		value, err := kvClient.Get(ctx, seedKey)
		if err != nil {
			level.Debug(logger).Log("msg", "failed to get cluster seed key for stability check", "err", err)
			continue
		}
		if seed, ok := value.(*ClusterSeed); ok && seed != nil {
			if previous == nil {
				previous = seed
				continue
			}
			if previous.UID != seed.UID {
				previous = seed
				stableCount = 0
				continue
			}
			stableCount++
			if stableCount > stabilityMinimumRequired {
				return seed
			}
		}
	}
}

func (rep *Reporter) init(ctx context.Context) {
	if rep.conf.Leader {
		rep.cluster = rep.initLeader(ctx)
		return
	}
	// follower only wait for the cluster seed to be set.
	// it will try forever to fetch the cluster seed.
	seed, _ := rep.fetchSeed(ctx, nil)
	rep.cluster = seed
}

// fetchSeed fetches the cluster seed from the object store and try until it succeeds.
// continueFn allow you to decide if we should continue retrying. Nil means always retry
func (rep *Reporter) fetchSeed(ctx context.Context, continueFn func(err error) bool) (*ClusterSeed, error) {
	var (
		backoff    = backoff.New(ctx, rep.conf.Backoff)
		readingErr = 0
	)
	for backoff.Ongoing() {
		seed, err := rep.readSeedFile(ctx)
		if err != nil {
			if !errors.Is(err, backend.ErrDoesNotExist) {
				readingErr++
			}
			level.Debug(rep.logger).Log("msg", "failed to read cluster seed file", "err", err)
			if readingErr > attemptNumber {
				if errors.Is(err, backend.ErrBadSeedFile) {
					level.Debug(rep.logger).Log("msg", "seed file corrupted")
				}
			}
			if continueFn == nil || continueFn(err) {
				backoff.Wait()
				continue
			}
			return nil, err
		}
		return seed, nil
	}
	return nil, backoff.Err()
}

// readSeedFile reads the cluster seed file from the object store.
func (rep *Reporter) readSeedFile(ctx context.Context) (*ClusterSeed, error) {
	reader, _, err := rep.reader.Read(ctx, backend.ClusterSeedFileName, backend.KeyPath{}, false)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := reader.Close(); err != nil {
			level.Error(rep.logger).Log("msg", "failed to close reader", "err", closeErr)
		}
	}()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	seed, err := JSONCodec.Decode(data)
	if err != nil {
		return nil, backend.ErrBadSeedFile
	}
	return seed.(*ClusterSeed), nil
}

// writeSeedFile writes the cluster seed to the object store.
func (rep *Reporter) writeSeedFile(ctx context.Context, seed ClusterSeed) error {
	data, err := JSONCodec.Encode(seed)
	if err != nil {
		return err
	}
	return rep.writer.Write(ctx, backend.ClusterSeedFileName, []string{}, bytes.NewReader(data), -1, false)
}

// running inits the reporter seed and start sending report for every interval
func (rep *Reporter) running(ctx context.Context) error {
	rep.init(ctx)

	if rep.cluster == nil {
		<-ctx.Done()
		return ctx.Err()
	}
	// check every minute if we should report.
	ticker := time.NewTicker(reportCheckInterval)
	defer ticker.Stop()

	// find  when to send the next report.
	next := nextReport(reportInterval, rep.cluster.CreatedAt, time.Now())
	if rep.lastReport.IsZero() {
		// if we never reported assumed it was the last interval.
		rep.lastReport = next.Add(-reportInterval)
	}
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			if !next.Equal(now) && now.Sub(rep.lastReport) < reportInterval {
				continue
			}
			level.Info(rep.logger).Log("msg", "reporting cluster stats", "date", time.Now())
			if err := rep.reportUsage(ctx, next); err != nil {
				level.Info(rep.logger).Log("msg", "failed to report usage", "err", err)
				continue
			}
			rep.lastReport = next
			next = next.Add(reportInterval)
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
			return ctx.Err()
		}
	}
}

// reportUsage reports the usage to grafana.com.
func (rep *Reporter) reportUsage(ctx context.Context, interval time.Time) error {
	backoff := backoff.New(ctx, backoff.Config{
		MinBackoff: time.Second,
		MaxBackoff: 30 * time.Second,
		MaxRetries: 5,
	})
	var errs multierror.MultiError
	for backoff.Ongoing() {
		if err := sendReport(ctx, rep.cluster, interval); err != nil {
			level.Info(rep.logger).Log("msg", "failed to send usage report", "retries", backoff.NumRetries(), "err", err)
			errs.Add(err)
			backoff.Wait()
			continue
		}
		level.Debug(rep.logger).Log("msg", "usage report sent with success")
		return nil
	}
	return errs.Err()
}

// nextReport compute the next report time based on the interval.
// The interval is based off the creation of the cluster seed to avoid all cluster reporting at the same time.
func nextReport(interval time.Duration, createdAt, now time.Time) time.Time {
	// createdAt * (x * interval ) >= now
	return createdAt.Add(time.Duration(math.Ceil(float64(now.Sub(createdAt))/float64(interval))) * interval)
}
