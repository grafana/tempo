package term

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
)

var ErrConfirmationFailed = errors.New("aborted by user")

// Confirm asks the user for confirmation
func Confirm(msg, approval string) error {
	return confirmFrom(os.Stdin, os.Stdout, msg, approval)
}

func confirmFrom(r io.Reader, w io.Writer, msg, approval string) error {
	reader := bufio.NewScanner(r)
	_, err := fmt.Fprintln(w, msg)
	if err != nil {
		return errors.Wrap(err, "writing to stdout")
	}

	_, err = fmt.Fprintf(w, "Please type '%s' to confirm: ", approval)
	if err != nil {
		return errors.Wrap(err, "writing to stdout")
	}

	if !reader.Scan() {
		if err := reader.Err(); err != nil {
			return errors.Wrap(err, "reading from stdin")
		}

		return ErrConfirmationFailed
	}

	if reader.Text() != approval {
		return ErrConfirmationFailed
	}

	return nil
}
