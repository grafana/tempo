// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

//go:build !linux
// +build !linux

package api

import (
	"context"
	"net"
	"net/http"

	"github.com/DataDog/datadog-agent/pkg/trace/api/internal/header"
)

// connContext is unimplemented for non-linux builds.
func connContext(ctx context.Context, c net.Conn) context.Context {
	return ctx
}

// IDProvider implementations are able to look up a container ID given a ctx and http header.
type IDProvider interface {
	GetContainerID(context.Context, http.Header) string
}

type idProvider struct{}

// NewIDProvider initializes an IDProvider instance, in non-linux environments the procRoot arg is unused.
func NewIDProvider(_ string) IDProvider {
	return &idProvider{}
}

func (_ *idProvider) GetContainerID(_ context.Context, h http.Header) string {
	return h.Get(header.ContainerID)
}
