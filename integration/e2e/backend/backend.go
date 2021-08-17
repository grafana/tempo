package backend

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cortexproject/cortex/integration/e2e"
	cortex_e2e_db "github.com/cortexproject/cortex/integration/e2e/db"
	"github.com/grafana/tempo/cmd/tempo/app"
	util "github.com/grafana/tempo/integration"
	"github.com/grafana/tempo/tempodb/backend/azure"
)

func parsePort(endpoint string) (int, error) {
	substrings := strings.Split(endpoint, ":")
	port, err := strconv.Atoi(substrings[len(substrings)-1])
	if err != nil {
		return 0, err
	}
	return port, nil
}

func New(scenario *e2e.Scenario, cfg app.Config) (*e2e.HTTPService, error) {
	var backendService *e2e.HTTPService
	switch cfg.StorageConfig.Trace.Backend {
	case "s3":
		port, err := parsePort(cfg.StorageConfig.Trace.S3.Endpoint)
		if err != nil {
			return nil, err
		}
		backendService = cortex_e2e_db.NewMinio(port, "tempo")
		if backendService == nil {
			return nil, fmt.Errorf("error creating minio backend")
		}
		err = scenario.StartAndWaitReady(backendService)
		if err != nil {
			return nil, err
		}
	case "azure":
		port, err := parsePort(cfg.StorageConfig.Trace.Azure.Endpoint)
		if err != nil {
			return nil, err
		}
		backendService = util.NewAzurite(port)
		err = scenario.StartAndWaitReady(backendService)
		if err != nil {
			return nil, err
		}
		cfg.StorageConfig.Trace.Azure.Endpoint = backendService.Endpoint(port)
		_, err = azure.CreateContainer(context.TODO(), cfg.StorageConfig.Trace.Azure)
		if err != nil {
			return nil, err
		}
	}

	return backendService, nil
}
