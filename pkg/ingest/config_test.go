package ingest

import (
	"context"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kfake"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestEnsureTopicPartitions(t *testing.T) {
	tests := []struct {
		name                     string
		topic                    string
		desiredPartitions        int
		existingPartitions       int
		topicExists              bool
		expectCreate             bool
		expectUpdate             bool
		expectError              bool
		expectedFinalPartitions  int
	}{
		{
			name:                    "create new topic",
			topic:                   "test-topic-create",
			desiredPartitions:       100,
			topicExists:             false,
			expectCreate:            true,
			expectUpdate:            false,
			expectError:             false,
			expectedFinalPartitions: 100,
		},
		{
			name:                    "topic exists with correct partitions",
			topic:                   "test-topic-correct",
			desiredPartitions:       100,
			existingPartitions:      100,
			topicExists:             true,
			expectCreate:            false,
			expectUpdate:            false,
			expectError:             false,
			expectedFinalPartitions: 100,
		},
		{
			name:                    "topic exists with fewer partitions - should update",
			topic:                   "test-topic-update",
			desiredPartitions:       100,
			existingPartitions:      10,
			topicExists:             true,
			expectCreate:            false,
			expectUpdate:            true,
			expectError:             false,
			expectedFinalPartitions: 100,
		},
		{
			name:                    "topic exists with more partitions - no update",
			topic:                   "test-topic-more",
			desiredPartitions:       10,
			existingPartitions:      100,
			topicExists:             true,
			expectCreate:            false,
			expectUpdate:            false,
			expectError:             false,
			expectedFinalPartitions: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster, err := kfake.NewCluster(kfake.NumBrokers(1))
			require.NoError(t, err)
			t.Cleanup(cluster.Close)

			addrs := cluster.ListenAddrs()
			require.Len(t, addrs, 1)

			// Create the topic if it should exist
			if tt.topicExists {
				cl, err := kgo.NewClient(kgo.SeedBrokers(addrs[0]))
				require.NoError(t, err)
				defer cl.Close()

				adm := kadm.NewClient(cl)
				defer adm.Close()

				const defaultReplication = 1
				_, err = adm.CreateTopic(context.Background(), int32(tt.existingPartitions), defaultReplication, nil, tt.topic)
				require.NoError(t, err)
			}

			cfg := KafkaConfig{
				Address:                          addrs[0],
				Topic:                            tt.topic,
				AutoCreateTopicDefaultPartitions: tt.desiredPartitions,
			}

			err = cfg.EnsureTopicPartitions(log.NewNopLogger())
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify the final partition count
			cl, err := kgo.NewClient(kgo.SeedBrokers(addrs[0]))
			require.NoError(t, err)
			defer cl.Close()

			adm := kadm.NewClient(cl)
			defer adm.Close()

			td, err := adm.ListTopics(context.Background(), tt.topic)
			require.NoError(t, err)
			require.NoError(t, td.Error())

			actualPartitions := len(td[tt.topic].Partitions.Numbers())
			require.Equal(t, tt.expectedFinalPartitions, actualPartitions, "partition count mismatch")
		})
	}
}
