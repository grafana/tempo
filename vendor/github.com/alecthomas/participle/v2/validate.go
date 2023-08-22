package participle

import (
	"fmt"
	"strings"
)

// Perform some post-construction validation. This currently does:
//
// Checks for left recursion.
func validate(n node) error {
	checked := map[*strct]bool{}
	seen := map[node]bool{}

	return visit(n, func(n node, next func() error) error {
		if n, ok := n.(*strct); ok {
			if !checked[n] && isLeftRecursive(n) {
				return fmt.Errorf("left recursion detected on\n\n%s", indent(n.String()))
			}
			checked[n] = true
			if seen[n] {
				return nil
			}
		}
		seen[n] = true
		return next()
	})
}

func isLeftRecursive(root *strct) (found bool) {
	defer func() { _ = recover() }()
	seen := map[node]bool{}
	_ = visit(root.expr, func(n node, next func() error) error {
		if found {
			return nil
		}
		switch n := n.(type) {
		case *strct:
			if root.typ == n.typ {
				found = true
			}

		case *sequence:
			if !n.head {
				panic("done")
			}
		}
		if seen[n] {
			return nil
		}
		seen[n] = true
		return next()
	})
	return
}

func indent(s string) string {
	return "  " + strings.Join(strings.Split(s, "\n"), "\n  ")
}
