// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package cgroups

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// UserHZToNano holds the divisor to convert HZ to Nanoseconds
// It's safe to consider it being 100Hz in userspace mode
const (
	UserHZToNano uint64 = uint64(time.Second) / 100
)

func randToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
