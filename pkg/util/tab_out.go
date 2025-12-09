package util

import (
	"fmt"
	"strings"
)

func TabOut(s fmt.Stringer) string {
	return strings.ReplaceAll(s.String(), "\n", "\n\t")
}
