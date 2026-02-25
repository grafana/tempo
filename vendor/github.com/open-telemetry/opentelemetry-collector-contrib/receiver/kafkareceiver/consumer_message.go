// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package kafkareceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver"

import (
	"iter"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// kafkaMessage provides a generic interface for Kafka messages that abstracts
// over Franz-go record types.
type kafkaMessage interface {
	value() []byte
	headers() messageHeaders
	topic() string
	partition() int32
	offset() int64
	timestamp() time.Time
}

type header struct {
	key   string
	value []byte
}

// messageHeaders provides a generic interface for accessing Kafka message headers.
type messageHeaders interface {
	get(key string) (string, bool)
	all() iter.Seq[header]
}

type franzMessage struct {
	record *kgo.Record
}

func wrapFranzMsg(record *kgo.Record) franzMessage {
	return franzMessage{record: record}
}

func (w franzMessage) value() []byte {
	return w.record.Value
}

func (w franzMessage) headers() messageHeaders {
	return franzHeaders{headers: w.record.Headers}
}

func (w franzMessage) topic() string {
	return w.record.Topic
}

func (w franzMessage) partition() int32 {
	return w.record.Partition
}

func (w franzMessage) offset() int64 {
	return w.record.Offset
}

func (w franzMessage) timestamp() time.Time {
	return w.record.Timestamp
}

type franzHeaders struct {
	headers []kgo.RecordHeader
}

func (h franzHeaders) get(key string) (string, bool) {
	for _, header := range h.headers {
		if header.Key == key {
			return string(header.Value), true
		}
	}
	return "", false
}

func (h franzHeaders) all() iter.Seq[header] {
	return func(yield func(header) bool) {
		for _, hdr := range h.headers {
			if !yield(header{key: hdr.Key, value: hdr.Value}) {
				return
			}
		}
	}
}
