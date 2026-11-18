// Command assign_codeowner assigns unassigned pull requests using assigned-PR load.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func requiredEnv(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("missing required environment variable: %s", name)
	}
	return value, nil
}

func run(ctx context.Context) error {
	token, err := requiredEnv("GH_TOKEN")
	if err != nil {
		return err
	}
	client, err := newGitHubClient(token)
	if err != nil {
		return err
	}
	return sweep(ctx, client)
}

func workflowError(err error) {
	message := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(err.Error())
	fmt.Printf("::error::%s\n", message)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		workflowError(err)
		os.Exit(1)
	}
}
