// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"bytes"
	"context"
	"crypto/sha1" //gosec:disable G505 -- SHA1 is intentionally used for generating unique identifiers
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

var communityIDProtocols = map[string]uint8{
	"ICMP":  1,
	"TCP":   6,
	"UDP":   17,
	"RSVP":  46,
	"ICMP6": 58,
	"SCTP":  132,
}

type CommunityIDArguments[K any] struct {
	SourceIP        ottl.StringGetter[K]
	SourcePort      ottl.IntGetter[K]
	DestinationIP   ottl.StringGetter[K]
	DestinationPort ottl.IntGetter[K]
	Protocol        ottl.Optional[ottl.StringGetter[K]]
	Seed            ottl.Optional[ottl.IntGetter[K]]
}

func NewCommunityIDFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("CommunityID", &CommunityIDArguments[K]{}, createCommunityIDFunction[K])
}

func createCommunityIDFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*CommunityIDArguments[K])

	if !ok {
		return nil, errors.New("CommunityIDFactory args must be of type *CommunityIDArguments[K]")
	}

	return communityID(
		args.SourceIP,
		args.SourcePort,
		args.DestinationIP,
		args.DestinationPort,
		args.Protocol,
		args.Seed,
	), nil
}

type communityIDHash struct {
	srcIPBytes []byte
	dstIPBytes []byte
	srcPort    uint16
	dstPort    uint16
	protocol   uint8
	seed       uint16
}

func (h *communityIDHash) normalize() {
	shouldSwap := false
	if len(h.srcIPBytes) != len(h.dstIPBytes) {
		shouldSwap = len(h.srcIPBytes) > len(h.dstIPBytes)
	} else if cmp := bytes.Compare(h.srcIPBytes, h.dstIPBytes); cmp > 0 {
		shouldSwap = true
	} else if cmp == 0 && h.srcPort > h.dstPort {
		shouldSwap = true
	}
	if shouldSwap {
		h.srcIPBytes, h.dstIPBytes = h.dstIPBytes, h.srcIPBytes
		h.srcPort, h.dstPort = h.dstPort, h.srcPort
	}
}

func (h *communityIDHash) compute() string {
	// Add seed (2 bytes, network order)
	flowTuple := make([]byte, 2)
	binary.BigEndian.PutUint16(flowTuple, h.seed)

	// Add source, destination IPs and 1-byte protocol
	flowTuple = append(flowTuple, h.srcIPBytes...)
	flowTuple = append(flowTuple, h.dstIPBytes...)
	flowTuple = append(flowTuple, h.protocol, 0)

	// Add source and destination ports (2 bytes each, network order)
	srcPortBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(srcPortBytes, h.srcPort)
	flowTuple = append(flowTuple, srcPortBytes...)

	dstPortBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(dstPortBytes, h.dstPort)
	flowTuple = append(flowTuple, dstPortBytes...)

	// Generate the SHA1 hash
	//gosec:disable G401 -- we are not using SHA1 for security, but for generating unique identifier, conflicts will be solved with the seed
	hashBytes := sha1.Sum(flowTuple)

	// Add version prefix (1) and return
	return "1:" + base64.StdEncoding.EncodeToString(hashBytes[:])
}

func communityID[K any](
	sourceIP ottl.StringGetter[K],
	sourcePort ottl.IntGetter[K],
	destinationIP ottl.StringGetter[K],
	destinationPort ottl.IntGetter[K],
	protocol ottl.Optional[ottl.StringGetter[K]],
	seed ottl.Optional[ottl.IntGetter[K]],
) ottl.ExprFunc[K] {
	return func(ctx context.Context, tCtx K) (any, error) {
		hash, err := makeCommunityIDHash(ctx, tCtx, sourceIP, sourcePort, destinationIP, destinationPort, protocol, seed)
		if err != nil {
			return nil, err
		}

		hash.normalize()
		return hash.compute(), nil
	}
}

func extractIPAndPort[K any](
	ctx context.Context,
	tCtx K,
	ipGetter ottl.StringGetter[K],
	portGetter ottl.IntGetter[K],
	endpointName string,
) (net.IP, int64, error) {
	ipValue, err := ipGetter.Get(ctx, tCtx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get %s IP: %w", endpointName, err)
	}

	port, err := portGetter.Get(ctx, tCtx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get %s port: %w", endpointName, err)
	}

	if port < 0 || port > 65535 {
		return nil, 0, fmt.Errorf("%s port must be between 0 and 65535, got %d", endpointName, port)
	}

	ipAddr := net.ParseIP(ipValue)
	if ipAddr == nil {
		return nil, 0, fmt.Errorf("invalid %s IP: %s", endpointName, ipValue)
	}

	return ipAddr, port, nil
}

func makeCommunityIDHash[K any](
	ctx context.Context,
	tCtx K,
	sourceIP ottl.StringGetter[K],
	sourcePort ottl.IntGetter[K],
	destinationIP ottl.StringGetter[K],
	destinationPort ottl.IntGetter[K],
	protocol ottl.Optional[ottl.StringGetter[K]],
	seed ottl.Optional[ottl.IntGetter[K]],
) (*communityIDHash, error) {
	srcIPAddr, srcPort, err := extractIPAndPort(ctx, tCtx, sourceIP, sourcePort, "source")
	if err != nil {
		return nil, err
	}

	dstIPAddr, dstPort, err := extractIPAndPort(ctx, tCtx, destinationIP, destinationPort, "destination")
	if err != nil {
		return nil, err
	}

	protocolValue := communityIDProtocols["TCP"] // defaults to TCP
	if !protocol.IsEmpty() {
		protocolStr, err := protocol.Get().Get(ctx, tCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to get protocol: %w", err)
		}

		var ok bool
		protocolValue, ok = communityIDProtocols[protocolStr]
		if !ok {
			return nil, fmt.Errorf("unsupported protocol: %s", protocolStr)
		}
	}

	// Get seed value (default: 0) if applied
	seedValue := uint16(0)
	if !seed.IsEmpty() {
		seedInt, err := seed.Get().Get(ctx, tCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to get seed: %w", err)
		}
		if seedInt < 0 || seedInt > 65535 {
			return nil, fmt.Errorf("seed must be between 0 and 65535, got %d", seedInt)
		}
		seedValue = uint16(seedInt)
	}

	srcIPBytes := srcIPAddr.To4()
	dstIPBytes := dstIPAddr.To4()

	if srcIPBytes == nil {
		srcIPBytes = srcIPAddr.To16()
	}

	if dstIPBytes == nil {
		dstIPBytes = dstIPAddr.To16()
	}

	return &communityIDHash{
		srcIPBytes: srcIPBytes,
		dstIPBytes: dstIPBytes,
		srcPort:    uint16(srcPort),
		dstPort:    uint16(dstPort),
		protocol:   protocolValue,
		seed:       seedValue,
	}, nil
}
