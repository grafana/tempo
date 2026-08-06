package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
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

	TLS           bool   `name:"tls" help:"use TLS transport" default:"false"`
	TLSServerName string `name:"tls-server-name" help:"override the TLS server name (SNI)"`
	TLSCA         string `name:"tls-ca" help:"path to a PEM-encoded CA certificate file"`
	TLSMinVersion string `name:"tls-min-version" default:"VersionTLS13" enum:"VersionTLS10,VersionTLS11,VersionTLS12,VersionTLS13" help:"minimum TLS version (only applies with --tls)"`
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

	resp, err := c.SubmitRedaction(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("submitting redaction: %w", err)
	}
	return resp, nil
}

func (cmd *redactCmd) buildTransportCredentials() (credentials.TransportCredentials, error) {
	return schedulerTransportCredentials(cmd.TLS, cmd.TLSServerName, cmd.TLSCA, cmd.TLSMinVersion)
}

// tlsMinVersions maps the dskit-style version names to their tls constants. Mirrors dskit's
// crypto/tls config values so the CLI flag matches the server's tls_min_version.
var tlsMinVersions = map[string]uint16{
	"VersionTLS10": tls.VersionTLS10,
	"VersionTLS11": tls.VersionTLS11,
	"VersionTLS12": tls.VersionTLS12,
	"VersionTLS13": tls.VersionTLS13,
}

// schedulerTransportCredentials builds gRPC transport credentials for the backend-scheduler
// client, shared by the redact submit and cancel commands. minVersion is a dskit-style version
// name (e.g. "VersionTLS13").
//
// TLS settings are rejected rather than ignored when useTLS is false: discarding them silently would
// send the tenant header and a control-plane call in cleartext while the operator believed the
// connection was protected. The version name is validated either way, so a typo cannot pass unnoticed
// behind a plaintext connection.
func schedulerTransportCredentials(useTLS bool, serverName, ca, minVersion string) (credentials.TransportCredentials, error) {
	if !useTLS {
		if _, err := parseTLSMinVersion(minVersion); err != nil {
			return nil, err
		}
		if ca != "" || serverName != "" {
			return nil, fmt.Errorf("--tls-ca and --tls-server-name require --tls; refusing to connect in cleartext with TLS options set")
		}
		return insecure.NewCredentials(), nil
	}

	cfg, err := schedulerTLSConfig(serverName, ca, minVersion)
	if err != nil {
		return nil, err
	}
	return credentials.NewTLS(cfg), nil
}

// parseTLSMinVersion resolves a dskit-style version name, listing the accepted names on failure so the
// message cannot drift from the table above.
func parseTLSMinVersion(minVersion string) (uint16, error) {
	if v, ok := tlsMinVersions[minVersion]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("unknown minimum TLS version %q (allowed: %s)",
		minVersion, strings.Join(slices.Sorted(maps.Keys(tlsMinVersions)), ", "))
}

// schedulerTLSConfig builds the TLS config for a scheduler connection: the system roots plus an
// optional additional CA, the requested minimum version, and an optional SNI override.
func schedulerTLSConfig(serverName, ca, minVersion string) (*tls.Config, error) {
	minVer, err := parseTLSMinVersion(minVersion)
	if err != nil {
		return nil, err
	}

	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("loading system cert pool: %w", err)
	}
	if certPool == nil {
		certPool = x509.NewCertPool()
	}

	if ca != "" {
		pem, err := os.ReadFile(ca)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert %q: %w", ca, err)
		}
		if !certPool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid certificates found in %q", ca)
		}
	}

	return &tls.Config{
		ServerName: serverName,
		RootCAs:    certPool,
		MinVersion: minVer,
	}, nil
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
