package cmdx

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ErrNilDependency is returned if a dependency is missing.
var ErrNilDependency = errors.New("a dependency was expected to be defined but is nil. Please open an issue with the stack trace")

// Must fatals with the optional message if err is not nil.
func Must(err error, message string, args ...interface{}) {
	if err == nil {
		return
	}

	_, _ = fmt.Fprintf(os.Stderr, message+"\n", args...)
	os.Exit(1)
}

// CheckResponse fatals if err is nil or the response.StatusCode does not match the expectedStatusCode
func CheckResponse(err error, expectedStatusCode int, response *http.Response) {
	Must(err, "Command failed because error occurred: %s", err)

	if response.StatusCode != expectedStatusCode {
		out, err := ioutil.ReadAll(response.Body)
		if err != nil {
			out = []byte{}
		}
		pretty, err := json.MarshalIndent(json.RawMessage(out), "", "\t")
		if err == nil {
			out = pretty
		}

		Fatalf(
			`Command failed because status code %d was expected but code %d was received.

Response payload:

%s`,
			expectedStatusCode,
			response.StatusCode,
			out,
		)
	}
}

// FormatResponse takes an object and prints a json.MarshalIdent version of it or fatals.
func FormatResponse(o interface{}) string {
	out, err := json.MarshalIndent(o, "", "\t")
	Must(err, `Command failed because an error occurred while prettifying output: %s`, err)
	return string(out)
}

// Fatalf prints to os.Stderr and exists with code 1.
func Fatalf(message string, args ...interface{}) {
	if len(args) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, message+"\n", args...)
	} else {
		_, _ = fmt.Fprintln(os.Stderr, message)
	}
	os.Exit(1)
}

// ExpectDependency expects every dependency to be not nil or it fatals.
func ExpectDependency(logger logrus.FieldLogger, dependencies ...interface{}) {
	if logger == nil {
		panic("missing logger for dependency check")
	}
	for _, d := range dependencies {
		if d == nil {
			logger.WithError(errors.WithStack(ErrNilDependency)).Fatalf("A fatal issue occurred.")
		}
	}
}
