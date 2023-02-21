package utils

import (
	"fmt"
	"os"
	"os/exec"
)

// ExecCmdFunc is a function type that matches exec.Command's signature
type ExecCmdFunc = func(name string, arg ...string) *exec.Cmd

// fakeExecCmd is a function that initialises a new exec.Cmd, one which will
// simply call the testName function rather than the command it is provided. It will
// also pass through as arguments to the testName function the testRunName, the command and its
// arguments.
func fakeExecCmd(testName string, testRunName string, command string, args ...string) *exec.Cmd {
	cs := []string{fmt.Sprintf("-test.run=%s", testName), "--", testRunName}
	cs = append(cs, command)
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

// BuildFakeExecCmd returns a fakeExecCmd for the testName and testRunName.
// See platform/platform_common_test.go for an example of how to use it to mock exec.Cmd in tests.
func BuildFakeExecCmd(testName string, testRunName string) ExecCmdFunc {
	return func(command string, args ...string) *exec.Cmd {
		return fakeExecCmd(testName, testRunName, command, args...)
	}
}

// ParseFakeExecCmdArgs parses the CLI's os.Args as passed by fakeExecCmd and returns the
// testRunName, and cmdList.
// Meant to be used from test functions that are called by a fakeExecCmd built with
// BuildFakeExecCmd.
func ParseFakeExecCmdArgs() (string, []string) {
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	return args[0], args[1:]
}
