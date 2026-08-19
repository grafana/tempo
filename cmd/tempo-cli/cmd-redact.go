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

// validate resolves the time window and enforces that the request is coherent: exactly one selector
// (an explicit trace ID list or a TraceQL query, never both and never neither), a fully specified and
// ordered window if one is given, and a window only alongside the query selector. The server enforces
// all of it too; checking here fails fast before dialing and before a TLS handshake.
//
// Resolving the window is a side effect: it populates cmd.startNano/endNano for submit().
func (cmd *redactCmd) validate() error {
	// Resolved here, not in submit(): validate() runs before the dial, so a mistyped bound fails without
	// a TLS handshake. One instant for both bounds, so identical relative specs collapse to a zero-width
	// window that the ordering check below rejects.
	now := time.Now()

	startSet, endSet, err := cmd.resolveWindow(now)
	if err != nil {
		return err
	}

	// The server enforces these too; checking here saves a round trip. Requiring both bounds is policy,
	// not a storage limit: a one-sided window is almost always a typo and guessing has no undo.
	if startSet != endSet {
		return errors.New("--start and --end must both be set or both be omitted")
	}
	if startSet && cmd.startNano >= cmd.endNano {
		return fmt.Errorf("--start must be before --end, but %s resolved to %s and %s resolved to %s",
			cmd.Start, time.Unix(0, cmd.startNano).UTC().Format(time.RFC3339Nano),
			cmd.End, time.Unix(0, cmd.endNano).UTC().Format(time.RFC3339Nano))
	}
	// Selector coherence is checked before window/selector compatibility, so a request that gets both
	// wrong is told about the more fundamental problem first.
	hasIDs := len(cmd.TraceIDs) > 0
	hasQuery := cmd.Query != ""
	switch {
	case hasIDs && hasQuery:
		return errors.New("--trace-id and --query are mutually exclusive")
	case !hasIDs && !hasQuery:
		return errors.New("one of --trace-id or --query must be provided")
	}

	// The trace-ID path applies no time bound, so a window would delete each listed trace only from the
	// overlapping blocks and leave the rest behind, reported as complete. The server refuses this too.
	if startSet && hasIDs {
		return errors.New("--start/--end cannot be combined with --trace-id: the window is not applied per trace, " +
			"so only the parts of each trace held by in-window blocks would be removed")
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
	return schedulerTransportCredentials(cmd.TLS, cmd.TLSServerName, cmd.TLSCA)
}

// schedulerTransportCredentials builds the dial credentials for the scheduler, shared by every
// redact subcommand. Hand-rolled rather than routed through the dskit grpcclient.Config the
// scheduler client already embeds; that consolidation is tracked separately.
func schedulerTransportCredentials(useTLS bool, serverName, caPath string) (credentials.TransportCredentials, error) {
	if !useTLS {
		return insecure.NewCredentials(), nil
	}

	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("loading system cert pool: %w", err)
	}
	if certPool == nil {
		certPool = x509.NewCertPool()
	}

	if caPath != "" {
		pem, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert %q: %w", caPath, err)
		}
		if !certPool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid certificates found in %q", caPath)
		}
	}

	return credentials.NewTLS(&tls.Config{
		ServerName: serverName,
		RootCAs:    certPool,
	}), nil
}

// unixNanoMin/unixNanoMax bound what time.Time.UnixNano() can express. The floor is the epoch rather
// than the true int64 minimum (1677-09-21) because every layer rejects a negative bound anyway.
var (
	unixNanoMin = time.Unix(0, 0)
	unixNanoMax = time.Unix(0, math.MaxInt64)
)

// resolveWindow resolves both bounds against a single instant, reporting which were supplied.
func (cmd *redactCmd) resolveWindow(now time.Time) (startSet, endSet bool, err error) {
	// Resolved into locals and assigned together, so a failure on the second bound cannot leave the
	// receiver describing a one-sided window that the returned flags deny.
	startNano, startSet, err := windowBoundNano(cmd.Start, now)
	if err != nil {
		return false, false, fmt.Errorf("parsing --start: %w", err)
	}
	endNano, endSet, err := windowBoundNano(cmd.End, now)
	if err != nil {
		return false, false, fmt.Errorf("parsing --end: %w", err)
	}

	cmd.startNano, cmd.endNano = startNano, endNano
	return startSet, endSet, nil
}

// windowBoundNano resolves a --start/--end value to absolute unix nanoseconds, reporting ok=false when
// the flag was not supplied. Resolving client-side freezes the window at submission so a long redaction
// never chases live ingest, and reuses the parseTime family the query commands already use.
//
// now comes from the caller so both bounds resolve against one instant: per-bound time.Now() puts
// "--start now-7d --end now-7d" a few nanoseconds apart, which passes every ordering check.
func windowBoundNano(spec string, now time.Time) (nano int64, ok bool, err error) {
	if strings.TrimSpace(spec) == "" {
		return 0, false, nil
	}

	t, err := parseTimeAt(spec, now)
	if err != nil {
		return 0, false, err
	}

	// Bound the resolved instant, not its year: UnixNano() wraps silently outside [unixNanoMin,
	// unixNanoMax], and a year-granular check leaks the last ~8 months of 2262 and everything before
	// 1970, all of which resolve negative and are only caught server-side after a TLS handshake.
	if t.Before(unixNanoMin) || t.After(unixNanoMax) {
		return 0, false, fmt.Errorf("time %q resolves to %s, outside the range this command can express (%s .. %s)",
			spec, t.UTC().Format(time.RFC3339Nano),
			unixNanoMin.UTC().Format(time.RFC3339Nano), unixNanoMax.UTC().Format(time.RFC3339Nano))
	}

	// Exactly the epoch collides with the "unbounded" sentinel, widening the window to the whole tenant.
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
