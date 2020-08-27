package flagx

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"
)

// MustGetBool returns a bool flag or fatals if an error occurs.
func MustGetBool(cmd *cobra.Command, name string) bool {
	ok, err := cmd.Flags().GetBool(name)
	if err != nil {
		cmdx.Fatalf(err.Error())
	}
	return ok
}

// MustGetString returns a string flag or fatals if an error occurs.
func MustGetString(cmd *cobra.Command, name string) string {
	s, err := cmd.Flags().GetString(name)
	if err != nil {
		cmdx.Fatalf(err.Error())
	}
	return s
}

// MustGetDuration returns a time.Duration flag or fatals if an error occurs.
func MustGetDuration(cmd *cobra.Command, name string) time.Duration {
	d, err := cmd.Flags().GetDuration(name)
	if err != nil {
		cmdx.Fatalf(err.Error())
	}
	return d
}

// MustGetStringSlice returns a []string flag or fatals if an error occurs.
func MustGetStringSlice(cmd *cobra.Command, name string) []string {
	ss, err := cmd.Flags().GetStringSlice(name)
	if err != nil {
		cmdx.Fatalf(err.Error())
	}
	return ss
}

// MustGetInt returns a int flag or fatals if an error occurs.
func MustGetInt(cmd *cobra.Command, name string) int {
	ss, err := cmd.Flags().GetInt(name)
	if err != nil {
		cmdx.Fatalf(err.Error())
	}
	return ss
}
