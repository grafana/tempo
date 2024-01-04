package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/middleware"
	"github.com/grafana/dskit/server"
	"github.com/grafana/dskit/services"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	util_log "github.com/grafana/tempo/pkg/util/log"
)

type TempoServer interface {
	HTTP() *mux.Router
	GRPC() *grpc.Server
	Log() log.Logger
	EnableHTTP2()

	StartAndReturnService(cfg server.Config, supportGRPCOnHTTP bool, servicesToWaitFor func() []services.Service) (services.Service, error)
}

type tempoServer struct {
	mux *mux.Router // all tempo http routes are added here

	externalServer *server.Server // the standard server that all HTTP/GRPC requests are served on
	// jpe: put internal server here as well?

	enableHTTP2 sync.Once
}

func newTempoServer() *tempoServer {
	return &tempoServer{
		mux: mux.NewRouter(),
		// externalServer will be initialized in StartService
	}
}

func (s *tempoServer) HTTP() *mux.Router {
	return s.mux
}

func (s *tempoServer) GRPC() *grpc.Server {
	return s.externalServer.GRPC
}

func (s *tempoServer) Log() log.Logger {
	return s.externalServer.Log
}

func (s *tempoServer) EnableHTTP2() {
	s.enableHTTP2.Do(func() {
		s.externalServer.HTTPServer.Handler = h2c.NewHandler(s.externalServer.HTTPServer.Handler, &http2.Server{})
	})
}

func (s *tempoServer) StartAndReturnService(cfg server.Config, supportGRPCOnHTTP bool, servicesToWaitFor func() []services.Service) (services.Service, error) {
	var err error

	metrics := server.NewServerMetrics(cfg)
	// use tempo's mux unless we are doing grpc over http, then we will let the library instantiate its own
	// router and piggy back on it to route grpc requests
	cfg.Router = s.mux
	if supportGRPCOnHTTP {
		cfg.Router = nil
		cfg.DoNotAddDefaultHTTPMiddleware = true // we don't want instrumentation on the "root" router, we want it on our mux
	}
	DisableSignalHandling(&cfg)
	s.externalServer, err = server.NewWithMetrics(cfg, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// now that we have created the server and service let's setup our grpc/http router if necessary
	if supportGRPCOnHTTP {
		s.EnableHTTP2()
		// jpe - this works as well
		// s.externalServer.HTTP.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// 	// route to GRPC server if it's a GRPC request
		// 	if req.ProtoMajor == 2 && strings.Contains(req.Header.Get("Content-Type"), "application/grpc") { // jpe - both? i don't think grafana sends the content-type header
		// 		s.externalServer.GRPC.ServeHTTP(w, req)
		// 		return
		// 	}

		// 	w.WriteHeader(http.StatusNotFound)
		// })

		// recreate dskit instrumentation here
		cfg.DoNotAddDefaultHTTPMiddleware = false
		httpMiddleware, err := server.BuildHTTPMiddleware(cfg, s.mux, metrics, s.externalServer.Log)
		if err != nil {
			return nil, fmt.Errorf("failed to create http middleware: %w", err)
		}
		router := middleware.Merge(httpMiddleware...).Wrap(s.mux)
		s.externalServer.HTTP.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// route to GRPC server if it's a GRPC request
			if req.ProtoMajor == 2 && strings.Contains(req.Header.Get("Content-Type"), "application/grpc") { // jpe - both? i don't think grafana sends the content-type header
				s.externalServer.GRPC.ServeHTTP(w, req)
				return
			}

			// default to standard http server
			router.ServeHTTP(w, req)
		})
	}

	return NewServerService(s.externalServer, servicesToWaitFor), nil
}

// NewServerService constructs service from Server component.
// servicesToWaitFor is called when server is stopping, and should return all
// services that need to terminate before server actually stops.
// N.B.: this function is NOT Cortex specific, please let's keep it that way.
// Passed server should not react on signals. Early return from Run function is considered to be an error.
func NewServerService(serv *server.Server, servicesToWaitFor func() []services.Service) services.Service {
	serverDone := make(chan error, 1)

	runFn := func(ctx context.Context) error {
		go func() {
			defer close(serverDone)
			serverDone <- serv.Run()
		}()

		select {
		case <-ctx.Done():
			return nil
		case err := <-serverDone:
			if err != nil {
				return err
			}
			return fmt.Errorf("server stopped unexpectedly")
		}
	}

	stoppingFn := func(_ error) error {
		// wait until all modules are done, and then shutdown server.
		for _, s := range servicesToWaitFor() {
			_ = s.AwaitTerminated(context.Background())
		}

		// shutdown HTTP and gRPC servers (this also unblocks Run)
		serv.Shutdown()

		// if not closed yet, wait until server stops.
		<-serverDone
		level.Info(util_log.Logger).Log("msg", "server stopped")
		return nil
	}

	return services.NewBasicService(nil, runFn, stoppingFn)
}

// DisableSignalHandling puts a dummy signal handler
func DisableSignalHandling(config *server.Config) {
	config.SignalHandler = make(ignoreSignalHandler)
}

type ignoreSignalHandler chan struct{}

func (dh ignoreSignalHandler) Loop() {
	<-dh
}

func (dh ignoreSignalHandler) Stop() {
	close(dh)
}
