package backend

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
)

const (
	azuriteImage  = "mcr.microsoft.com/azure-storage/azurite:3.35.0"
	gcsImage      = "fsouza/fake-gcs-server:1.52.2"
	seaweedfsImage = "chrislusf/seaweedfs:latest"
)

func parsePort(endpoint string) (int, error) {
	substrings := strings.Split(endpoint, ":")
	portStrings := strings.Split(substrings[len(substrings)-1], "/")
	port, err := strconv.Atoi(portStrings[0])
	if err != nil {
		return 0, err
	}
	return port, nil
}

func New(scenario *e2e.Scenario, cfg app.Config) (*e2e.HTTPService, error) {
	var backendService *e2e.HTTPService
	switch cfg.StorageConfig.Trace.Backend {
	case backend.S3:
		port, err := parsePort(cfg.StorageConfig.Trace.S3.Endpoint)
		if err != nil {
			return nil, err
		}
		backendService = NewSeaweedFS(port, "tempo")
		if backendService == nil {
			return nil, fmt.Errorf("error creating seaweedfs backend")
		}
		err = scenario.StartAndWaitReady(backendService)
		if err != nil {
			return nil, err
		}
	case backend.Azure:
		port, err := parsePort(cfg.StorageConfig.Trace.Azure.Endpoint)
		if err != nil {
			return nil, err
		}
		backendService = NewAzurite(port)
		err = scenario.StartAndWaitReady(backendService)
		if err != nil {
			return nil, err
		}
		cfg.StorageConfig.Trace.Azure.Endpoint = backendService.Endpoint(port)
		_, err = azure.CreateContainer(context.TODO(), cfg.StorageConfig.Trace.Azure)
		if err != nil {
			return nil, err
		}
	case backend.GCS:
		port, err := parsePort(cfg.StorageConfig.Trace.GCS.Endpoint)
		if err != nil {
			return nil, err
		}
		backendService = NewGCS(port)
		if backendService == nil {
			return nil, fmt.Errorf("error creating gcs backend")
		}
		err = scenario.StartAndWaitReady(backendService)
		if err != nil {
			return nil, err
		}
	}

	return backendService, nil
}

func NewAzurite(port int) *e2e.HTTPService {
	s := e2e.NewHTTPService(
		"azurite",
		azuriteImage, // Create the azurite container
		e2e.NewCommandWithoutEntrypoint("sh", "-c", "azurite -l /data --blobHost 0.0.0.0"),
		e2e.NewHTTPReadinessProbe(port, "/devstoreaccount1?comp=list", 403, 403), // If we get 403 the Azurite is ready
		port, // blob storage port
	)

	s.SetBackoff(util.TempoBackoff())

	return s
}

func NewGCS(port int) *e2e.HTTPService {
	commands := []string{
		"mkdir -p /data/tempo",
		"/bin/fake-gcs-server -data /data -public-host=tempo_e2e-gcs -port=4443",
	}
	s := e2e.NewHTTPService(
		"gcs",
		gcsImage, // Create the the gcs container
		e2e.NewCommandWithoutEntrypoint("sh", "-c", strings.Join(commands, " && ")),
		e2e.NewHTTPReadinessProbe(port, "/", 400, 400), // for lack of a better way, readiness probe does not support https at the moment
		port,
	)

	s.SetBackoff(util.TempoBackoff())

	return s
}

func NewSeaweedFS(port int, bktNames ...string) *e2e.HTTPService {
	// Create S3 config JSON content
	s3Config := `{"identities":[{"name":"tempo","credentials":[{"accessKey":"Cheescake","secretKey":"supersecret"}],"actions":["Admin","Read","Write","List","Tagging"]}]}`

	// Command to start weed server with S3
	commands := []string{
		fmt.Sprintf("echo '%s' > /tmp/s3.json", s3Config),
		fmt.Sprintf("weed server -s3 -s3.port=%d -s3.config=/tmp/s3.json -dir=/data", port),
	}

	s := e2e.NewHTTPService(
		fmt.Sprintf("seaweedfs-%v", port),
		seaweedfsImage,
		e2e.NewCommandWithoutEntrypoint("sh", "-c", strings.Join(commands, " && ")),
		e2e.NewHTTPReadinessProbe(9333, "/cluster/status", 200, 200), // Master health check
		port,
	)

	s.SetBackoff(util.TempoBackoff())

	return s
}
