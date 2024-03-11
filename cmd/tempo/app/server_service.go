package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
	"github.com/grafana/dskit/middleware"
	"github.com/grafana/dskit/server"
	"github.com/grafana/dskit/services"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	util_log "github.com/grafana/tempo/pkg/util/log"
)

type TempoServer interface {
	HTTPRouter() *mux.Router
	HTTPHandler() http.Handler
	GRPC() *grpc.Server
	Log() log.Logger
	EnableHTTP2()
	SetKeepAlivesEnabled(enabled bool)

	StartAndReturnService(cfg server.Config, supportGRPCOnHTTP bool, servicesToWaitFor func() []services.Service) (services.Service, error)
}

// todo: evaluate whether the internal server should be included as part of this
type tempoServer struct {
	mux     *mux.Router  // all tempo http routes are added here
	handler http.Handler // the final handler which includes the router and any middleware

	externalServer  *server.Server // the standard server that all HTTP/GRPC requests are served on
	enableHTTP2Once sync.Once
}

func newTempoServer() *tempoServer {
	return &tempoServer{
		mux: mux.NewRouter(),
		// externalServer will be initialized in StartService
	}
}

func (s *tempoServer) HTTPRouter() *mux.Router {
	return s.mux
}

func (s *tempoServer) HTTPHandler() http.Handler {
	return s.handler
}

func (s *tempoServer) GRPC() *grpc.Server {
	return s.externalServer.GRPC
}

func (s *tempoServer) Log() log.Logger {
	return s.externalServer.Log
}

func (s *tempoServer) EnableHTTP2() {
	s.enableHTTP2Once.Do(func() {
		s.externalServer.HTTPServer.Handler = h2c.NewHandler(s.externalServer.HTTPServer.Handler, &http2.Server{})
	})
}

func (s *tempoServer) SetKeepAlivesEnabled(enabled bool) {
	s.externalServer.HTTPServer.SetKeepAlivesEnabled(enabled)
}

func (s *tempoServer) StartAndReturnService(cfg server.Config, supportGRPCOnHTTP bool, servicesToWaitFor func() []services.Service) (services.Service, error) {
	var err error

	// the configured write timeout. this is the amount of time we actually want to cancel the request after.
	actualWriteTimeout := cfg.HTTPServerWriteTimeout
	cfg.HTTPServerWriteTimeout = 0 // set to 0 b/c we are going to enforce
	timeoutMiddleware := middleware.Func(func(h http.Handler) http.Handler {
		return http.TimeoutHandler(h, actualWriteTimeout, "request timed out")
	})

	metrics := server.NewServerMetrics(cfg)
	DisableSignalHandling(&cfg)

	if !supportGRPCOnHTTP {
		// add timeout middleware
		if actualWriteTimeout > 0 {
			cfg.HTTPMiddleware = []middleware.Interface{timeoutMiddleware}
		}

		// We don't do any GRPC handling, let the library handle all routing for us
		cfg.Router = s.mux

		s.externalServer, err = server.NewWithMetrics(cfg, metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to create server: %w", err)
		}
		s.handler = s.externalServer.HTTPServer.Handler
	} else {
		// We want to route both GRPC and HTTP requests on the same endpoint
		cfg.Router = nil
		cfg.DoNotAddDefaultHTTPMiddleware = true // we don't want instrumentation on the "root" router, we want it on our mux. it will be added below.

		if actualWriteTimeout > 0 {
			cfg.GRPCMiddleware = append(cfg.GRPCMiddleware, unaryTimeoutInterceptor(actualWriteTimeout))
			cfg.GRPCStreamMiddleware = append(cfg.GRPCStreamMiddleware, streamTimeoutInterceptor(actualWriteTimeout))
		}

		s.externalServer, err = server.NewWithMetrics(cfg, metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to create server: %w", err)
		}

		// now that we have created the server and service let's setup our grpc/http router if necessary
		// for grpc to work we must enable h2c on the external server
		s.EnableHTTP2()

		// recreate dskit instrumentation here
		cfg.DoNotAddDefaultHTTPMiddleware = false
		httpMiddleware, err := server.BuildHTTPMiddleware(cfg, s.mux, metrics, s.externalServer.Log)
		if err != nil {
			return nil, fmt.Errorf("failed to create http middleware: %w", err)
		}

		// add timeout middleware and add to the handler
		if actualWriteTimeout > 0 {
			httpMiddleware = append(httpMiddleware, timeoutMiddleware)
		}
		s.handler = middleware.Merge(httpMiddleware...).Wrap(s.mux)

		s.externalServer.HTTP.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// route to GRPC server if it's a GRPC request
			if req.ProtoMajor == 2 && strings.Contains(req.Header.Get("Content-Type"), "application/grpc") {
				// http is handled by the http.TimeoutHandler. manually write a timeout for grpc
				s.externalServer.GRPC.ServeHTTP(w, req)

				return
			}

			// default to standard http server
			s.handler.ServeHTTP(w, req)
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

// jpe - revert to main
func unaryTimeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return handler(ctx, req)
	}
}

func streamTimeoutInterceptor(timeout time.Duration) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, cancel := context.WithTimeout(ss.Context(), timeout)
		defer cancel()

		return handler(srv, &grpc_middleware.WrappedServerStream{
			ServerStream:   ss,
			WrappedContext: ctx,
		})
	}
}
