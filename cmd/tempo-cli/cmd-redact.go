package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/grafana/dskit/user"

	schedulerclient "github.com/grafana/tempo/modules/backendscheduler/client"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util"
)

type redactCmd struct {
	SchedulerAddr string `arg:"" help:"backend scheduler gRPC address (host:port)"`

	TenantID string   `name:"tenant" required:"" help:"tenant ID"`
	TraceIDs []string `name:"trace-id" required:"" help:"trace ID to redact (may be repeated)"`

	TLS           bool   `name:"tls" help:"use TLS transport" default:"false"`
	TLSServerName string `name:"tls-server-name" help:"override the TLS server name (SNI)"`
	TLSCA         string `name:"tls-ca" help:"path to a PEM-encoded CA certificate file"`
}

func (cmd *redactCmd) Run(_ *globalOptions) error {
	traceIDs, err := parseTraceIDs(cmd.TraceIDs)
	if err != nil {
		return err
	}

	transportCred, err := cmd.buildTransportCredentials()
	if err != nil {
		return fmt.Errorf("building transport credentials: %w", err)
	}

	c, err := schedulerclient.NewWithOptions(cmd.SchedulerAddr, defaultSchedulerClientConfig(), transportCred)
	if err != nil {
		return fmt.Errorf("creating scheduler client: %w", err)
	}
	defer c.Close()

	resp, err := cmd.submit(context.Background(), c, traceIDs)
	if err != nil {
		return err
	}

	fmt.Printf("batch_id:     %s\njobs_created: %d\n", resp.BatchId, resp.JobsCreated)
	return nil
}

// submit injects the tenant org ID into the outgoing gRPC metadata and calls SubmitRedaction.
func (cmd *redactCmd) submit(ctx context.Context, c tempopb.BackendSchedulerClient, traceIDs [][]byte) (*tempopb.SubmitRedactionResponse, error) {
	ctx = user.InjectOrgID(ctx, cmd.TenantID)
	ctx, err := user.InjectIntoGRPCRequest(ctx)
	if err != nil {
		return nil, fmt.Errorf("injecting tenant ID into gRPC request: %w", err)
	}

	resp, err := c.SubmitRedaction(ctx, &tempopb.SubmitRedactionRequest{
		TenantId: cmd.TenantID,
		TraceIds: traceIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("submitting redaction: %w", err)
	}
	return resp, nil
}

func (cmd *redactCmd) buildTransportCredentials() (credentials.TransportCredentials, error) {
	if !cmd.TLS {
		return insecure.NewCredentials(), nil
	}

	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("loading system cert pool: %w", err)
	}
	if certPool == nil {
		certPool = x509.NewCertPool()
	}

	if cmd.TLSCA != "" {
		pem, err := os.ReadFile(cmd.TLSCA)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert %q: %w", cmd.TLSCA, err)
		}
		if !certPool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid certificates found in %q", cmd.TLSCA)
		}
	}

	return credentials.NewTLS(&tls.Config{
		ServerName: cmd.TLSServerName,
		RootCAs:    certPool,
	}), nil
}

// parseTraceIDs converts a slice of hex trace ID strings to raw byte slices.
func parseTraceIDs(hexIDs []string) ([][]byte, error) {
	traceIDs := make([][]byte, 0, len(hexIDs))
	for _, id := range hexIDs {
		b, err := util.HexStringToTraceID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid trace ID %q: %w", id, err)
		}
		traceIDs = append(traceIDs, b)
	}
	return traceIDs, nil
}

// defaultSchedulerClientConfig returns a zero-value Config suitable for CLI use.
func defaultSchedulerClientConfig() schedulerclient.Config {
	var cfg schedulerclient.Config
	cfg.GRPCClientConfig.RegisterFlagsWithPrefix("backendscheduler.client", &flag.FlagSet{})
	return cfg
}
