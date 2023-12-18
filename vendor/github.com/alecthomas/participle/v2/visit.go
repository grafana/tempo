package participle

import "fmt"

// Visit all nodes.
//
// Cycles are deliberately not detected, it is up to the visitor function to handle this.
func visit(n node, visitor func(n node, next func() error) error) error {
	return visitor(n, func() error {
		switch n := n.(type) {
		case *disjunction:
			for _, child := range n.nodes {
				if err := visit(child, visitor); err != nil {
					return err
				}
			}
			return nil
		case *strct:
			return visit(n.expr, visitor)
		case *custom:
			return nil
		case *union:
			for _, member := range n.disjunction.nodes {
				if err := visit(member, visitor); err != nil {
					return err
				}
			}
			return nil
		case *sequence:
			if err := visit(n.node, visitor); err != nil {
				return err
			}
			if n.next != nil {
				return visit(n.next, visitor)
			}
			return nil
		case *parseable:
			return nil
		case *capture:
			return visit(n.node, visitor)
		case *reference:
			return nil
		case *negation:
			return visit(n.node, visitor)
		case *literal:
			return nil
		case *group:
			return visit(n.expr, visitor)
		case *lookaheadGroup:
			return visit(n.expr, visitor)
		default:
			panic(fmt.Sprintf("%T", n))
		}
	})
}
