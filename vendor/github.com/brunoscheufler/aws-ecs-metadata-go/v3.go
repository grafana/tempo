package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Based on https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v3.html

const (
	ecsMetadataUriEnvV3 = "ECS_CONTAINER_METADATA_URI"
)

type ContainerMetadataV3 struct {
	DockerID   string `json:"DockerId"`
	Name       string `json:"Name"`
	DockerName string `json:"DockerName"`
	Image      string `json:"Image"`
	ImageID    string `json:"ImageID"`
	Labels     struct {
		EcsCluster               string `json:"com.amazonaws.ecs.cluster"`
		EcsContainerName         string `json:"com.amazonaws.ecs.container-name"`
		EcsTaskArn               string `json:"com.amazonaws.ecs.task-arn"`
		EcsTaskDefinitionFamily  string `json:"com.amazonaws.ecs.task-definition-family"`
		EcsTaskDefinitionVersion string `json:"com.amazonaws.ecs.task-definition-version"`
	} `json:"Labels"`
	DesiredStatus string `json:"DesiredStatus"`
	KnownStatus   string `json:"KnownStatus"`
	Limits        struct {
		CPU    int `json:"CPU"`
		Memory int `json:"Memory"`
	} `json:"Limits"`
	CreatedAt time.Time `json:"CreatedAt"`
	StartedAt time.Time `json:"StartedAt,omitempty"`
	Type      string    `json:"Type"`
	Networks  []struct {
		NetworkMode   string   `json:"NetworkMode"`
		IPv4Addresses []string `json:"IPv4Addresses"`
	} `json:"Networks"`
}

type TaskMetadataV3 struct {
	Cluster       string                `json:"Cluster"`
	TaskARN       string                `json:"TaskARN"`
	Family        string                `json:"Family"`
	Revision      string                `json:"Revision"`
	DesiredStatus string                `json:"DesiredStatus"`
	KnownStatus   string                `json:"KnownStatus"`
	Containers    []ContainerMetadataV3 `json:"Containers"`
	Limits        struct {
		CPU    float64 `json:"CPU"`
		Memory int     `json:"Memory"`
	} `json:"Limits"`
	PullStartedAt time.Time `json:"PullStartedAt"`
	PullStoppedAt time.Time `json:"PullStoppedAt"`
}

// Retrieve ECS Task Metadata in V3 format
func GetTaskV3(ctx context.Context, client *http.Client) (*TaskMetadataV3, error) {
	metadataUrl := os.Getenv(ecsMetadataUriEnvV3)
	if metadataUrl == "" {
		return nil, fmt.Errorf("missing metadata uri in environment (%s)", ecsMetadataUriEnvV3)
	}

	taskMetadata := &TaskMetadataV3{}
	body, err := fetch(ctx, client, fmt.Sprintf("%s/task", metadataUrl))

	err = json.Unmarshal(body, &taskMetadata)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal into task metadata (v3): %w", err)
	}

	return taskMetadata, nil
}

// Retrieve ECS Container Metadata in V3 format
func GetContainerV3(ctx context.Context, client *http.Client) (*ContainerMetadataV3, error) {
	metadataUrl := os.Getenv(ecsMetadataUriEnvV3)
	if metadataUrl == "" {
		return nil, fmt.Errorf("missing metadata uri in environment (%s)", ecsMetadataUriEnvV3)
	}

	contaienrMetadata := &ContainerMetadataV3{}
	body, err := fetch(ctx, client, fmt.Sprintf("%s", metadataUrl))

	err = json.Unmarshal(body, &contaienrMetadata)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal into container metadata (v3): %w", err)
	}

	return contaienrMetadata, nil
}
