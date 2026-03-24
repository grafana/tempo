package app

import (
	"net/http"
	"testing"

	"github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/kv/consul"
	"github.com/grafana/dskit/ring"
	"github.com/grafana/dskit/server"
	"github.com/grafana/dskit/services"
	"github.com/grafana/tempo/modules/overrides"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type fakeTempoServer struct {
	router *mux.Router
	grpc   *grpc.Server
}

func (f *fakeTempoServer) HTTPRouter() *mux.Router   { return f.router }
func (f *fakeTempoServer) HTTPHandler() http.Handler { return f.router }
func (f *fakeTempoServer) GRPC() *grpc.Server        { return f.grpc }
func (f *fakeTempoServer) Log() log.Logger           { return log.NewNopLogger() }
func (f *fakeTempoServer) EnableHTTP2()              {}
func (f *fakeTempoServer) SetKeepAlivesEnabled(bool) {}
func (f *fakeTempoServer) StartAndReturnService(server.Config, bool, func() []services.Service) (services.Service, error) {
	return services.NewIdleService(nil, nil), nil
}

func TestInitLiveStoreSingleBinaryUsesLocalIngest(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Target = SingleBinary
	cfg.LiveStore.WAL.Filepath = t.TempDir()
	cfg.LiveStore.ShutdownMarkerDir = t.TempDir()
	cfg.LiveStore.Ring.InstanceID = "single-binary"

	partitionRingStore, partitionRingCloser := consul.NewInMemoryClient(ring.GetPartitionRingCodec(), log.NewNopLogger(), nil)
	defer partitionRingCloser.Close()
	readRingStore, readRingCloser := consul.NewInMemoryClient(ring.GetCodec(), log.NewNopLogger(), nil)
	defer readRingCloser.Close()
	cfg.LiveStore.PartitionRing.KVStore.Mock = partitionRingStore
	cfg.LiveStore.Ring.KVStore.Mock = readRingStore

	overridesSvc, err := overrides.NewOverrides(cfg.Overrides, nil, prometheus.NewRegistry())
	require.NoError(t, err)

	app := &App{
		cfg:       *cfg,
		Server:    &fakeTempoServer{router: mux.NewRouter(), grpc: grpc.NewServer()},
		Overrides: overridesSvc,
	}

	svc, err := app.initLiveStore()
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.NotNil(t, app.liveStore)
	require.False(t, app.cfg.LiveStore.ConsumeFromKafka)
}
