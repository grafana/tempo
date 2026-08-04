package main

import (
	"context"
	"fmt"

	"github.com/grafana/dskit/user"

	schedulerclient "github.com/grafana/tempo/modules/backendscheduler/client"
	"github.com/grafana/tempo/pkg/tempopb"
)

type redactCancelCmd struct {
	SchedulerAddr string `arg:"" help:"backend scheduler gRPC address (host:port)"`

	TenantID string `name:"tenant" required:"" help:"tenant ID"`

	TLS           bool   `name:"tls" help:"use TLS transport" default:"false"`
	TLSServerName string `name:"tls-server-name" help:"override the TLS server name (SNI)"`
	TLSCA         string `name:"tls-ca" help:"path to a PEM-encoded CA certificate file"`
}

func (cmd *redactCancelCmd) Run(_ *globalOptions) error {
	transportCred, err := schedulerTransportCredentials(cmd.TLS, cmd.TLSServerName, cmd.TLSCA)
	if err != nil {
		return fmt.Errorf("building transport credentials: %w", err)
	}

	c, err := schedulerclient.NewWithOptions(cmd.SchedulerAddr, defaultSchedulerClientConfig(), transportCred)
	if err != nil {
		return fmt.Errorf("creating scheduler client: %w", err)
	}
	defer c.Close()

	resp, err := cmd.cancel(context.Background(), c)
	if err != nil {
		return err
	}

	fmt.Printf("batch_id:       %s\npending_purged: %d\n", resp.BatchId, resp.PendingPurged)
	fmt.Println("in-flight jobs will finish; the batch is then removed and compaction resumes")
	return nil
}

// cancel injects the tenant org ID into the outgoing gRPC metadata and calls CancelRedaction. The
// tenant is sent exclusively via the X-Scope-OrgID header; the server sources it from the
// authenticated context, never a body field.
func (cmd *redactCancelCmd) cancel(ctx context.Context, c tempopb.BackendSchedulerClient) (*tempopb.CancelRedactionResponse, error) {
	ctx = user.InjectOrgID(ctx, cmd.TenantID)
	ctx, err := user.InjectIntoGRPCRequest(ctx)
	if err != nil {
		return nil, fmt.Errorf("injecting tenant ID into gRPC request: %w", err)
	}

	resp, err := c.CancelRedaction(ctx, &tempopb.CancelRedactionRequest{})
	if err != nil {
		return nil, fmt.Errorf("cancelling redaction: %w", err)
	}
	return resp, nil
}
