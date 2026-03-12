package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/grafana/tempo/cmd/tempo/app"
	"github.com/grafana/tempo/modules/distributor"
	"github.com/grafana/tempo/modules/distributor/forwarder"
	"github.com/grafana/tempo/modules/overrides/userconfigurable/client"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run ./cmd/validate-limits/ <limits.json>\n")
		os.Exit(1)
	}

	limitsFile := os.Args[1]

	// Read JSON file
	data, err := os.ReadFile(limitsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Unmarshal into Limits struct
	var limits client.Limits
	if err := json.Unmarshal(data, &limits); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Create config with hardcoded valid forwarder
	cfg := &app.Config{
		Distributor: distributor.Config{
			Forwarders: forwarder.ConfigList{
				{Name: "k6-cloud-insights"},
			},
		},
	}

	// Create validator and validate
	validator := app.NewOverridesValidator(cfg)
	if err := validator.Validate(&limits); err != nil {
		fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Limits validated successfully")
}
