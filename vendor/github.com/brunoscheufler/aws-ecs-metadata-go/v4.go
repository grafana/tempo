package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Based on https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html

const (
	ecsMetadataUriEnvV4 = "ECS_CONTAINER_METADATA_URI_V4"

	labelEcsCluster               = "com.amazonaws.ecs.cluster"
	labelEcsContainerName         = "com.amazonaws.ecs.container-name"
	labelEcsTaskArn               = "com.amazonaws.ecs.task-arn"
	labelEcsTaskDefinitionFamily  = "com.amazonaws.ecs.task-definition-family"
	labelEcsTaskDefinitionVersion = "com.amazonaws.ecs.task-definition-version"
)

type Limits struct {
	CPU    float64 `json:"CPU"`
	Memory int     `json:"Memory"`
}

type LabelsV4 struct {
	EcsCluster               string
	EcsContainerName         string
	EcsTaskArn               string
	EcsTaskDefinitionFamily  string
	EcsTaskDefinitionVersion string

	rest map[string]string
}

func (l LabelsV4) Get(name string) string {
	return l.rest[name]
}

func (l *LabelsV4) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &l.rest); err != nil {
		return err
	}
	if cluster, ok := l.rest[labelEcsCluster]; ok {
		l.EcsCluster = cluster
		delete(l.rest, labelEcsCluster)
	}
	if containerName, ok := l.rest[labelEcsContainerName]; ok {
		l.EcsContainerName = containerName
		delete(l.rest, labelEcsContainerName)
	}
	if taskArn, ok := l.rest[labelEcsTaskArn]; ok {
		l.EcsTaskArn = taskArn
		delete(l.rest, labelEcsTaskArn)
	}
	if family, ok := l.rest[labelEcsTaskDefinitionFamily]; ok {
		l.EcsTaskDefinitionFamily = family
		delete(l.rest, labelEcsTaskDefinitionFamily)
	}
	if version, ok := l.rest[labelEcsTaskDefinitionVersion]; ok {
		l.EcsTaskDefinitionVersion = version
		delete(l.rest, labelEcsTaskDefinitionVersion)
	}
	return nil
}

type ContainerMetadataV4 struct {
	DockerID      string    `json:"DockerId"`
	Name          string    `json:"Name"`
	DockerName    string    `json:"DockerName"`
	Image         string    `json:"Image"`
	ImageID       string    `json:"ImageID"`
	Labels        LabelsV4  `json:"Labels"`
	DesiredStatus string    `json:"DesiredStatus"`
	KnownStatus   string    `json:"KnownStatus"`
	Limits        Limits    `json:"Limits"`
	CreatedAt     time.Time `json:"CreatedAt"`
	StartedAt     time.Time `json:"StartedAt"`
	Type          string    `json:"Type"`
	ContainerARN  string    `json:"ContainerARN"`
	LogDriver     string    `json:"LogDriver"`
	LogOptions    struct {
		AwsLogsCreateGroup string `json:"awslogs-create-group"`
		AwsLogsGroup       string `json:"awslogs-group"`
		AwsLogsStream      string `json:"awslogs-stream"`
		AwsRegion          string `json:"awslogs-region"`
	} `json:"LogOptions"`
	Networks []struct {
		NetworkMode              string   `json:"NetworkMode"`
		IPv4Addresses            []string `json:"IPv4Addresses"`
		AttachmentIndex          int      `json:"AttachmentIndex"`
		IPv4SubnetCIDRBlock      string   `json:"IPv4SubnetCIDRBlock"`
		MACAddress               string   `json:"MACAddress"`
		DomainNameServers        []string `json:"DomainNameServers"`
		DomainNameSearchList     []string `json:"DomainNameSearchList"`
		PrivateDNSName           string   `json:"PrivateDNSName"`
		SubnetGatewayIpv4Address string   `json:"SubnetGatewayIpv4Address"`
	} `json:"Networks"`
}

type TaskMetadataV4 struct {
	Cluster          string                `json:"Cluster"`
	TaskARN          string                `json:"TaskARN"`
	Family           string                `json:"Family"`
	Revision         string                `json:"Revision"`
	DesiredStatus    string                `json:"DesiredStatus"`
	KnownStatus      string                `json:"KnownStatus"`
	Limits           Limits                `json:"Limits"`
	PullStartedAt    time.Time             `json:"PullStartedAt"`
	PullStoppedAt    time.Time             `json:"PullStoppedAt"`
	AvailabilityZone string                `json:"AvailabilityZone"`
	LaunchType       string                `json:"LaunchType"`
	Containers       []ContainerMetadataV4 `json:"Containers"`
}

// Retrieve ECS Task Metadata in V4 format
func GetTaskV4(ctx context.Context, client *http.Client) (*TaskMetadataV4, error) {
	metadataUrl := os.Getenv(ecsMetadataUriEnvV4)
	if metadataUrl == "" {
		return nil, fmt.Errorf("missing metadata uri in environment (%s)", ecsMetadataUriEnvV4)
	}

	taskMetadata := &TaskMetadataV4{}
	body, err := fetch(ctx, client, fmt.Sprintf("%s/task", metadataUrl))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve task metadata v4: %w", err)
	}

	err = json.Unmarshal(body, &taskMetadata)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal into task metadata v4: %w", err)
	}

	return taskMetadata, nil
}

// Retrieve ECS Container Metadata in V4 format
func GetContainerV4(ctx context.Context, client *http.Client) (*ContainerMetadataV4, error) {
	metadataUrl := os.Getenv(ecsMetadataUriEnvV4)
	if metadataUrl == "" {
		return nil, fmt.Errorf("missing metadata uri in environment (%s)", ecsMetadataUriEnvV4)
	}

	containerMetadata := &ContainerMetadataV4{}
	body, err := fetch(ctx, client, metadataUrl)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve container metadata v4: %w", err)
	}

	err = json.Unmarshal(body, &containerMetadata)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal into container metadata v4: %w", err)
	}

	return containerMetadata, nil
}
