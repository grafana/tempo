// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package api

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/DataDog/datadog-agent/pkg/trace/api/internal/header"
	"github.com/DataDog/datadog-agent/pkg/util/cgroups"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type ucredKey struct{}

// connContext injects a Unix Domain Socket's User Credentials into the
// context.Context object provided. This is useful as the connContext member of an http.Server, to
// provide User Credentials to HTTP handlers.
//
// If the connection c is not a *net.UnixConn, the unchanged context is returned.
func connContext(ctx context.Context, c net.Conn) context.Context {
	s, ok := c.(*net.UnixConn)
	if !ok {
		return ctx
	}
	file, err := s.File()
	if err != nil {
		log.Debugf("Failed to obtain unix socket file: %v", err)
		return ctx
	}
	fd := int(file.Fd())
	ucred, err := syscall.GetsockoptUcred(fd, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		log.Debugf("Failed to read credentials from unix socket: %v", err)
		return ctx
	}
	return context.WithValue(ctx, ucredKey{}, ucred)
}

// cacheExpiration determines how long a pid->container ID mapping is considered valid. This value is
// somewhat arbitrarily chosen, but just needs to be large enough to reduce latency and I/O load
// caused by frequently reading mappings, and small enough that pid-reuse doesn't cause mismatching
// of pids with container ids. A one minute cache means the latency and I/O should be low, and
// there would have to be thousands of containers spawned and dying per second to cause a mismatch.
const cacheExpiration = time.Minute

// IDProvider implementations are able to look up a container ID given a ctx and http header.
type IDProvider interface {
	GetContainerID(context.Context, http.Header) string
}

// noCgroupsProvider is a fallback IDProvider that only looks in the http header for a container ID.
type noCgroupsProvider struct{}

func (i *noCgroupsProvider) GetContainerID(_ context.Context, h http.Header) string {
	return h.Get(header.ContainerID)
}

// NewIDProvider initializes an IDProvider instance using the provided procRoot to perform cgroups lookups in linux environments.
func NewIDProvider(procRoot string) IDProvider {
	reader, err := cgroups.NewReader()
	if err != nil {
		log.Warnf("Failed to identify cgroups version due to err: %v. APM data may be missing containerIDs.", err)
		return &noCgroupsProvider{}
	}
	cgroupController := ""
	if reader.CgroupVersion() == 1 {
		cgroupController = "memory" // The 'memory' controller is used by the cgroupv1 utils in the agent to parse the procfs.
	}
	c := NewCache(1 * time.Minute)
	return &cgroupIDProvider{
		procRoot:   procRoot,
		controller: cgroupController,
		cache:      c,
	}
}

type cgroupIDProvider struct {
	procRoot   string
	controller string
	cache      *Cache
}

// GetContainerID returns the container ID in the http.Header,
// otherwise looks for a PID in the ctx which is used to search cgroups for a container ID.
func (c *cgroupIDProvider) GetContainerID(ctx context.Context, h http.Header) string {
	if id := h.Get(header.ContainerID); id != "" {
		return id
	}
	ucred, ok := ctx.Value(ucredKey{}).(*syscall.Ucred)
	if !ok || ucred == nil {
		return ""
	}
	cid, err := c.getCachedContainerID(strconv.Itoa(int(ucred.Pid)))
	if err != nil {
		log.Debugf("Could not get container ID from pid: %d: %v\n", ucred.Pid, err)
		return ""
	}
	return cid
}

func (c *cgroupIDProvider) getCachedContainerID(pid string) (string, error) {
	currentTime := time.Now()
	entry, found, err := c.cache.Get(currentTime, pid, cacheExpiration)
	if found {
		if err != nil {
			return "", err
		}

		return entry.(string), nil
	}

	// No cache, cacheValidity is 0 or too old value
	val, err := cgroups.IdentiferFromCgroupReferences(c.procRoot, pid, c.controller, cgroups.ContainerFilter)
	if err != nil {
		c.cache.Store(currentTime, pid, nil, err)
		return "", err
	}

	c.cache.Store(currentTime, pid, val, nil)
	return val, nil
}

// The below cache is copied from /pkg/util/containers/v2/metrics/provider/cache.go. It is not
// imported to avoid making the datadog-agent module a dependency of the pkg/trace module. The
// datadog-agent module contains replace directives which are not inherited by packages that
// require it, and cannot be guaranteed to function correctly as a dependency.
type cacheEntry struct {
	value     interface{}
	err       error
	timestamp time.Time
}

// Cache provides a caching mechanism based on staleness toleration provided by requestor
type Cache struct {
	cache       map[string]cacheEntry
	cacheLock   sync.RWMutex
	gcInterval  time.Duration
	gcTimestamp time.Time
}

// NewCache returns a new cache dedicated to a collector
func NewCache(gcInterval time.Duration) *Cache {
	return &Cache{
		cache:      make(map[string]cacheEntry),
		gcInterval: gcInterval,
	}
}

// Get retrieves data from cache, returns not found if cacheValidity == 0
func (c *Cache) Get(currentTime time.Time, key string, cacheValidity time.Duration) (interface{}, bool, error) {
	if cacheValidity <= 0 {
		return nil, false, nil
	}

	c.cacheLock.RLock()
	entry, found := c.cache[key]
	c.cacheLock.RUnlock()

	if !found || currentTime.Sub(entry.timestamp) > cacheValidity {
		return nil, false, nil
	}

	if entry.err != nil {
		return nil, true, entry.err
	}

	return entry.value, true, nil
}

// Store sets data in the cache, it also clears the cache if the gcInterval has passed
func (c *Cache) Store(currentTime time.Time, key string, value interface{}, err error) {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()

	if currentTime.Sub(c.gcTimestamp) > c.gcInterval {
		c.cache = make(map[string]cacheEntry, len(c.cache))
		c.gcTimestamp = currentTime
	}

	c.cache[key] = cacheEntry{value: value, timestamp: currentTime, err: err}
}
