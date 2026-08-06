package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"os"
	"time"

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
	TraceIDs []string `name:"trace-id" help:"trace ID to redact (may be repeated; mutually exclusive with --query)"`
	Query    string   `name:"query" help:"TraceQL query selecting traces to redact (mutually exclusive with --trace-id)"`
	DryRun   bool     `name:"dry-run" default:"false" help:"evaluate and report match counts without rewriting any blocks"`
	Start    string   `name:"start" help:"restrict redaction to blocks/traces at or after this time. 'now', 'now-<dur>' (e.g. now-7d), or RFC3339. Empty = unbounded"`
	End      string   `name:"end" help:"restrict redaction to blocks/traces at or before this time. Same forms as --start. Empty = unbounded"`

	TLS           bool   `name:"tls" help:"use TLS transport" default:"false"`
	TLSServerName string `name:"tls-server-name" help:"override the TLS server name (SNI)"`
	TLSCA         string `name:"tls-ca" help:"path to a PEM-encoded CA certificate file"`
}

func (cmd *redactCmd) Run(_ *globalOptions) error {
	if err := cmd.validate(); err != nil {
		return err
	}

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
	if cmd.DryRun {
		fmt.Println("mode:         dry-run (jobs will report match counts; no blocks will be rewritten)")
	}
	return nil
}

// validate enforces that exactly one selector is provided: an explicit trace ID list or a
// TraceQL query, never both and never neither. The server enforces the same, but checking
// here fails fast before dialing.
func (cmd *redactCmd) validate() error {
	hasIDs := len(cmd.TraceIDs) > 0
	hasQuery := cmd.Query != ""
	switch {
	case hasIDs && hasQuery:
		return fmt.Errorf("--trace-id and --query are mutually exclusive")
	case !hasIDs && !hasQuery:
		return fmt.Errorf("one of --trace-id or --query must be provided")
	}
	return nil
}

// submit injects the tenant org ID into the outgoing gRPC metadata and calls SubmitRedaction.
// The tenant is sent exclusively via the X-Scope-OrgID header; the server sources it from
// the authenticated context and ignores any tenant_id field on the request body.
func (cmd *redactCmd) submit(ctx context.Context, c tempopb.BackendSchedulerClient, traceIDs [][]byte) (*tempopb.SubmitRedactionResponse, error) {
	ctx = user.InjectOrgID(ctx, cmd.TenantID)
	ctx, err := user.InjectIntoGRPCRequest(ctx)
	if err != nil {
		return nil, fmt.Errorf("injecting tenant ID into gRPC request: %w", err)
	}

	req := &tempopb.SubmitRedactionRequest{}
	if cmd.Query != "" {
		req.Selector = &tempopb.SubmitRedactionRequest_Query{
			Query: &tempopb.TraceQLSelector{Query: cmd.Query},
		}
	} else {
		req.TraceIds = traceIDs
	}
	if cmd.DryRun {
		req.Mode = tempopb.RedactionMode_REDACTION_MODE_DRY_RUN
	}

	// Resolve any relative window (now-7d, …) to absolute nanos client-side, so the server gets a
	// frozen window. Empty stays 0 (unbounded on that side).
	now := time.Now()
	if req.StartTimeUnixNano, err = parseTimeSpec(cmd.Start, now); err != nil {
		return nil, fmt.Errorf("parsing --start: %w", err)
	}
	if req.EndTimeUnixNano, err = parseTimeSpec(cmd.End, now); err != nil {
		return nil, fmt.Errorf("parsing --end: %w", err)
	}

	resp, err := c.SubmitRedaction(ctx, req)
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
