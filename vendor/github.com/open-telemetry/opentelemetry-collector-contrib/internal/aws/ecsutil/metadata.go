// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ecsutil // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil"

// TaskMetadata defines task metadata for a task
type TaskMetadata struct {
	AvailabilityZone string              `json:"AvailabilityZone,omitempty"`
	Cluster          string              `json:"Cluster,omitempty"`
	Containers       []ContainerMetadata `json:"Containers,omitempty"`
	Family           string              `json:"Family,omitempty"`
	KnownStatus      string              `json:"KnownStatus,omitempty"`
	LaunchType       string              `json:"LaunchType,omitempty"`
	Limits           Limits              `json:"Limits,omitempty"`
	PullStartedAt    string              `json:"PullStartedAt,omitempty"`
	PullStoppedAt    string              `json:"PullStoppedAt,omitempty"`
	Revision         string              `json:"Revision,omitempty"`
	TaskARN          string              `json:"TaskARN,omitempty"`
}

// ContainerMetadata defines container metadata for a container
type ContainerMetadata struct {
	ContainerARN  string            `json:"ContainerARN,omitempty"`
	ContainerName string            `json:"Name,omitempty"`
	CreatedAt     string            `json:"CreatedAt,omitempty"`
	DockerID      string            `json:"DockerId,omitempty"`
	DockerName    string            `json:"DockerName,omitempty"`
	ExitCode      *int64            `json:"ExitCode,omitempty"`
	FinishedAt    string            `json:"FinishedAt,omitempty"`
	Image         string            `json:"Image,omitempty"`
	ImageID       string            `json:"ImageID,omitempty"`
	KnownStatus   string            `json:"KnownStatus,omitempty"`
	Labels        map[string]string `json:"Labels,omitempty"`
	Limits        Limits            `json:"Limits,omitempty"`
	LogDriver     string            `json:"LogDriver,omitempty"`
	LogOptions    LogOptions        `json:"LogOptions,omitempty"`
	Networks      []Network         `json:"Networks,omitempty"`
	StartedAt     string            `json:"StartedAt,omitempty"`
	Type          string            `json:"Type,omitempty"`
}

// Limits defines the Cpu and Memory limits
type Limits struct {
	CPU    *float64 `json:"CPU,omitempty"`
	Memory *uint64  `json:"Memory,omitempty"`
}

// LogOptions defines the CloudWatch configuration
type LogOptions struct {
	LogGroup string `json:"awslogs-group,omitempty"`
	Region   string `json:"awslogs-region,omitempty"`
	Stream   string `json:"awslogs-stream,omitempty"`
}

type Network struct {
	IPv4Addresses []string `json:"IPv4Addresses,omitempty"`
	NetworkMode   string   `json:"NetworkMode,omitempty"`
}
