package client

import (
	"context"
	"fmt"
	"io"

	"github.com/mark3labs/mcp-go/client/transport"
)

// NewStdioMCPClient creates a new stdio-based MCP client that communicates with a subprocess.
// It launches the specified command with given arguments and sets up stdin/stdout pipes for communication.
// Returns an error if the subprocess cannot be started or the pipes cannot be created.
//
// NOTICE: NewStdioMCPClient will start the connection automatically. Don't call the Start method manually.
// This is for backward compatibility.
func NewStdioMCPClient(
	command string,
	env []string,
	args ...string,
) (*Client, error) {

	stdioTransport := transport.NewStdio(command, env, args...)
	err := stdioTransport.Start(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to start stdio transport: %w", err)
	}

	return NewClient(stdioTransport), nil
}

// GetStderr returns a reader for the stderr output of the subprocess.
// This can be used to capture error messages or logs from the subprocess.
func GetStderr(c *Client) (io.Reader, bool) {
	t := c.GetTransport()

	stdio, ok := t.(*transport.Stdio)
	if !ok {
		return nil, false
	}

	return stdio.Stderr(), true
}
