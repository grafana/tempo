// Command assign_codeowner assigns unassigned pull requests using assigned-PR load.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func run(ctx context.Context, command string) error {
	if command != "sweep" && command != "report-failure" {
		return fmt.Errorf("unknown command %q; expected sweep or report-failure", command)
	}
	cfg, err := configFromEnv()
	if err != nil {
		return err
	}
	token, err := requiredEnv("GH_TOKEN")
	if err != nil {
		return err
	}
	client := newGitHubClient(token)
	if command == "report-failure" {
		runURL, err := requiredEnv("RUN_URL")
		if err != nil {
			return err
		}
		return reportFailure(ctx, client, cfg.repo, runURL)
	}
	return sweep(ctx, client, cfg)
}

func queryWith(keyValues ...string) url.Values {
	if len(keyValues)%2 != 0 {
		panic("queryWith requires key/value pairs")
	}
	values := make(url.Values, len(keyValues)/2)
	for index := 0; index < len(keyValues); index += 2 {
		values.Set(keyValues[index], keyValues[index+1])
	}
	return values
}

func workflowError(err error) {
	message := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(err.Error())
	fmt.Printf("::error::%s\n", message)
}

func main() {
	command := "sweep"
	if len(os.Args) > 2 {
		workflowError(errors.New("usage: assign_codeowner [sweep|report-failure]"))
		os.Exit(2)
	}
	if len(os.Args) == 2 {
		command = os.Args[1]
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, command); err != nil {
		workflowError(err)
		os.Exit(1)
	}
}
