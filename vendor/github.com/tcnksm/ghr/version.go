package main

import (
	"bytes"
	"fmt"
	"time"

	latest "github.com/tcnksm/go-latest"
)

// Name is application name
const Name = "ghr"

// Version is application version
const Version string = "0.13.0"

// GitCommit describes latest commit hash.
// This is automatically extracted by git describe --always.
var GitCommit string

// OutputVersion checks the current version and compares it against releases
// available on GitHub. If there is a newer version available, it prints an
// update warning.
func OutputVersion() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s version v%s", Name, Version)
	if len(GitCommit) != 0 {
		fmt.Fprintf(&buf, " (%s)", GitCommit)
	}
	fmt.Fprintf(&buf, "\n")

	// Check latest version is release or not.
	verCheckCh := make(chan *latest.CheckResponse)
	go func() {
		githubTag := &latest.GithubTag{
			Owner:      "tcnksm",
			Repository: "ghr",
		}

		res, err := latest.Check(githubTag, Version)
		if err != nil {
			// Don't return error
			Debugf("[ERROR] Check latest version is failed: %s", err)
			return
		}
		verCheckCh <- res
	}()

	select {
	case <-time.After(defaultCheckTimeout):
	case res := <-verCheckCh:
		if res.Outdated {
			fmt.Fprintf(&buf,
				"Latest version of ghr is v%s, please upgrade!\n",
				res.Current)
		}
	}

	return buf.String()
}
