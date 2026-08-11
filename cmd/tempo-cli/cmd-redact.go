package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"os"
	"strings"

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

	// startNano/endNano hold the window resolved by validate().
	startNano int64
	endNano   int64

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
	// Resolve the window here, not in submit(): validate() runs before the scheduler is dialed, so a
	// mistyped bound fails immediately rather than after a TLS handshake.
	var err error
	if cmd.startNano, err = windowBoundNano(cmd.Start); err != nil {
		return fmt.Errorf("parsing --start: %w", err)
	}
	if cmd.endNano, err = windowBoundNano(cmd.End); err != nil {
		return fmt.Errorf("parsing --end: %w", err)
	}
	// The server enforces both of these; checking here saves a round trip. A one-sided window is
	// refused because the storage layer only bounds the per-block scan when both bounds are set.
	if (cmd.startNano == 0) != (cmd.endNano == 0) {
		return fmt.Errorf("--start and --end must both be set or both be omitted")
	}
	if cmd.startNano != 0 && cmd.startNano >= cmd.endNano {
		return fmt.Errorf("--start must be before --end")
	}

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

	// The window was resolved and validated in validate().
	req.StartTimeUnixNano = cmd.startNano
	req.EndTimeUnixNano = cmd.endNano

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

// unixNanoMinYear/unixNanoMaxYear bound what time.Time.UnixNano() can represent. Outside this range
// it is undefined and wraps silently, so a bound written as "far future" would resolve negative — and a
// negative window selects every block and then scans each one unbounded. Reject rather than wrap.
const (
	unixNanoMinYear = 1678
	unixNanoMaxYear = 2262
)

// windowBoundNano resolves a --start/--end value to absolute unix nanoseconds, or 0 when omitted.
//
// Resolution reuses parseTime, the helper the query commands already use, so the accepted forms (now,
// now-<dur> with Prometheus units, RFC3339) stay consistent across the CLI. Resolving client-side means
// the window is frozen at submission and a long redaction never chases live ingest.
func windowBoundNano(spec string) (int64, error) {
	if strings.TrimSpace(spec) == "" {
		return 0, nil
	}
	t, err := parseTime(spec)
	if err != nil {
		return 0, err
	}
	if y := t.Year(); y < unixNanoMinYear || y > unixNanoMaxYear {
		return 0, fmt.Errorf("time %q is outside the representable range (%d-%d)", spec, unixNanoMinYear, unixNanoMaxYear)
	}
	return t.UnixNano(), nil
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
