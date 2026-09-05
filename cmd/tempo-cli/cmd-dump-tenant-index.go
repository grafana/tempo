package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type dumpTenantIndexCmd struct {
	TenantID string `arg:"" help:"tenant-id within the bucket"`
	backendOptions
}

// Run reads the tenant's protobuf-encoded tenant index and dumps it as JSON
// to stdout, for ad hoc exploration with jq or similar tools. Tempo no longer
// keeps a JSON copy of the tenant index in the backend by default (see
// blocklist_poll_tenant_index_json_fallback), so this recreates one on demand.
func (cmd *dumpTenantIndexCmd) Run(ctx *globalOptions) error {
	r, _, _, err := loadBackend(&cmd.backendOptions, ctx)
	if err != nil {
		return err
	}

	idx, err := r.TenantIndex(context.Background(), cmd.TenantID)
	if err != nil {
		return fmt.Errorf("reading tenant index: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(idx)
}
