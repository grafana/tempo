package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-jsonnet/formatter"
)

func pageln(i ...interface{}) error {
	return fPageln(strings.NewReader(fmt.Sprint(i...)))
}

// fPageln invokes the systems pager with the supplied data.
// If the PAGER environment variable is empty, no pager is used.
// If the PAGER environment variable is unset, use GNU less with convenience flags.
func fPageln(r io.Reader) error {
	pager, ok := os.LookupEnv("TANKA_PAGER")
	if !ok {
		pager, ok = os.LookupEnv("PAGER")
	}
	if !ok {
		// --RAW-CONTROL-CHARS  Honors colors from diff. Must be in all caps, otherwise display issues occur.
		// --quit-if-one-screen Closer to the git experience.
		// --no-init            Don't clear the screen when exiting.
		pager = "less --RAW-CONTROL-CHARS --quit-if-one-screen --no-init"
	}

	if interactive && pager != "" {
		cmd := exec.Command("sh", "-c", pager)
		cmd.Stdin = r
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	_, err := io.Copy(os.Stdout, r)
	return err
}

// writeJSON writes the given object to the path as a JSON file
func writeJSON(i interface{}, path string) error {
	out, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling: %s", err)
	}

	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		return fmt.Errorf("writing %s: %s", path, err)
	}

	return nil
}

// writeJsonnet writes the given object to the path as a formatted Jsonnet file
func writeJsonnet(i interface{}, path string) error {
	out, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling: %s", err)
	}

	main, err := formatter.Format(path, string(out), formatter.DefaultOptions())
	if err != nil {
		return fmt.Errorf("formatting %s: %s", path, err)
	}

	if err := os.WriteFile(path, []byte(main), 0644); err != nil {
		return fmt.Errorf("writing %s: %s", path, err)
	}

	return nil
}
