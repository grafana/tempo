// Copyright The OpenTelemetry Authors
// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package udpserver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver/internal/udpserver"

import (
	"context"
	"fmt"
	"sync"

	"github.com/apache/thrift/lib/go/thrift"
	"go.uber.org/zap"
)

// ThriftProcessor is a server that processes spans using a TBuffered Server
type ThriftProcessor struct {
	server        *UDPServer
	handler       AgentProcessor
	protocolPool  *sync.Pool
	numProcessors int
	processing    sync.WaitGroup
	logger        *zap.Logger
}

// AgentProcessor handler used by the processor to process thrift and call the reporter
// with the deserialized struct. This interface is implemented directly by Thrift generated
// code, e.g. jaegerThrift.NewAgentProcessor(handler), where handler implements the Agent
// Thrift service interface, which is invoked with the deserialized struct.
type AgentProcessor interface {
	Process(ctx context.Context, iprot, oprot thrift.TProtocol) (success bool, err thrift.TException)
}

// NewThriftProcessor creates a TBufferedServer backed ThriftProcessor
func NewThriftProcessor(
	server *UDPServer,
	numProcessors int,
	factory thrift.TProtocolFactory,
	handler AgentProcessor,
	logger *zap.Logger,
) (*ThriftProcessor, error) {
	if numProcessors <= 0 {
		return nil, fmt.Errorf(
			"number of processors must be greater than 0, called with %d", numProcessors)
	}
	protocolPool := &sync.Pool{
		New: func() any {
			trans := &TBufferedReadTransport{}
			return factory.GetProtocol(trans)
		},
	}

	res := &ThriftProcessor{
		server:        server,
		handler:       handler,
		protocolPool:  protocolPool,
		logger:        logger,
		numProcessors: numProcessors,
	}
	res.processing.Add(res.numProcessors)
	for i := 0; i < res.numProcessors; i++ {
		go func() {
			res.processBuffer()
			res.processing.Done()
		}()
	}
	return res, nil
}

// Serve starts serving traffic
func (s *ThriftProcessor) Serve() {
	s.server.Serve()
}

// IsServing indicates whether the server is currently serving traffic
func (s *ThriftProcessor) IsServing() bool {
	return s.server.IsServing()
}

// Stop stops the serving of traffic and waits until the queue is
// emptied by the readers
func (s *ThriftProcessor) Stop() {
	s.server.Stop()
	s.processing.Wait()
}

// processBuffer reads data off the channel and puts it into a custom transport for
// the processor to process
func (s *ThriftProcessor) processBuffer() {
	for buf := range s.server.DataChan() {
		protocol := s.protocolPool.Get().(thrift.TProtocol)
		_, _ = buf.WriteTo(protocol.Transport()) // writes to memory transport don't fail
		s.logger.Debug("Span(s) received by the agent", zap.Int("bytes-received", buf.Len()))

		// NB: oddly, thrift-gen/agent/agent.go:L156 does this: `return true, thrift.WrapTException(err2)`
		// So we check for both OK and error.
		if ok, err := s.handler.Process(context.Background(), protocol, protocol); !ok || err != nil {
			s.logger.Error("Processor failed", zap.Error(err))
		}
		s.protocolPool.Put(protocol)
		s.server.DataRecd(buf) // acknowledge receipt and release the buffer
	}
}
