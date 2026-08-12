package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
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
	Start    string   `name:"start" help:"start of the redaction window: 'now', 'now-<dur>' (e.g. now-7d), or RFC3339. Must be given with --end; omit both for the whole tenant"`
	End      string   `name:"end" help:"end of the redaction window. Same forms as --start. Must be given with --start"`

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
	//
	// Both bounds resolve against this one instant, so identical relative specs produce identical
	// bounds and are caught by the ordering check below rather than becoming a nanoseconds-wide window.
	now := time.Now()

	startSet, endSet, err := cmd.resolveWindow(now)
	if err != nil {
		return err
	}

	// The server enforces both of these; checking here saves a round trip. Requiring both bounds is a
	// deliberate policy choice rather than a storage constraint: a one-sided window is almost always a
	// typo, and on a destructive command the cost of guessing wrong has no undo.
	if startSet != endSet {
		return errors.New("--start and --end must both be set or both be omitted")
	}
	if startSet && cmd.startNano >= cmd.endNano {
		return fmt.Errorf("--start must be before --end, but %s resolved to %s and %s resolved to %s",
			cmd.Start, time.Unix(0, cmd.startNano).UTC().Format(time.RFC3339Nano),
			cmd.End, time.Unix(0, cmd.endNano).UTC().Format(time.RFC3339Nano))
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

// unixNanoMin/unixNanoMax are the instants time.Time.UnixNano() is defined between. unixNanoMin is the
// epoch rather than the true int64 floor (1677-09-21) because every layer rejects a negative bound, so
// a pre-epoch bound cannot be submitted regardless.
var (
	unixNanoMin = time.Unix(0, 0)
	unixNanoMax = time.Unix(0, math.MaxInt64)
)

// resolveWindow resolves both bounds against a single instant, reporting which were supplied.
func (cmd *redactCmd) resolveWindow(now time.Time) (startSet, endSet bool, err error) {
	if cmd.startNano, startSet, err = windowBoundNano(cmd.Start, now); err != nil {
		return false, false, fmt.Errorf("parsing --start: %w", err)
	}
	if cmd.endNano, endSet, err = windowBoundNano(cmd.End, now); err != nil {
		return false, false, fmt.Errorf("parsing --end: %w", err)
	}
	return startSet, endSet, nil
}

// windowBoundNano resolves a --start/--end value to absolute unix nanoseconds, reporting ok=false when
// the flag was not supplied.
//
// Resolution reuses the parseTime family, the helpers the query commands already use, so the accepted
// forms (now, now-<dur> with Prometheus units, RFC3339) stay consistent across the CLI. Resolving
// client-side means the window is frozen at submission and a long redaction never chases live ingest.
//
// now is supplied by the caller so both bounds of one window resolve against a single instant. Letting
// each bound take its own time.Now() puts "--start now-7d --end now-7d" a few nanoseconds apart, which
// passes every ordering check and submits a window nothing can match.
func windowBoundNano(spec string, now time.Time) (nano int64, ok bool, err error) {
	if strings.TrimSpace(spec) == "" {
		return 0, false, nil
	}

	t, err := parseTimeAt(spec, now)
	if err != nil {
		return 0, false, err
	}

	// Bound the resolved instant, not its year. time.Time.UnixNano() is only defined between
	// unixNanoMin and unixNanoMax and wraps silently outside them, and a year-granular check leaks
	// the ~8 months of 2262 past the limit as well as everything before 1970 — all of which resolve
	// negative. The scheduler rejects a negative bound, but only after a dial and TLS handshake,
	// which defeats the point of checking here.
	if t.Before(unixNanoMin) || t.After(unixNanoMax) {
		return 0, false, fmt.Errorf("time %q resolves to %s, outside the range this command can express (%s .. %s)",
			spec, t.UTC().Format(time.RFC3339),
			unixNanoMin.UTC().Format(time.RFC3339), unixNanoMax.UTC().Format(time.RFC3339))
	}

	// Exactly the epoch collides with the sentinel that every layer reads as "unbounded", so a
	// window asking for a single instant at the epoch would widen to the whole tenant.
	if nano = t.UnixNano(); nano == 0 {
		return 0, false, fmt.Errorf("time %q resolves to the unix epoch, which this command reserves to mean "+
			"'no bound'; use 1970-01-01T00:00:00.000000001Z instead", spec)
	}

	return nano, true, nil
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
