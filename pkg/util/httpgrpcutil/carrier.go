package httpgrpcutil

import (
	"github.com/grafana/dskit/httpgrpc"
)

// Used to transfer trace information from/to HTTP request.
type HttpgrpcHeadersCarrier httpgrpc.HTTPRequest

func (c *HttpgrpcHeadersCarrier) Get(key string) string {
	for _, h := range c.Headers {
		if h.Key == key {
			return h.Values[0]
		}
	}
	return ""
}

func (c *HttpgrpcHeadersCarrier) Set(key, val string) {
	c.Headers = append(c.Headers, &httpgrpc.Header{
		Key:    key,
		Values: []string{val},
	})
}

func (c *HttpgrpcHeadersCarrier) Keys() []string {
	k := make([]string, 0, len(c.Headers))
	for _, h := range c.Headers {
		k = append(k, h.Key)
	}
	return k
}
