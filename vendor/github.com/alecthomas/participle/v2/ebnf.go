package participle

import (
	"fmt"
	"strings"
)

// String returns the EBNF for the grammar.
//
// Productions are always upper cased. Lexer tokens are always lower case.
func (p *Parser[G]) String() string {
	return ebnf(p.typeNodes[p.rootType])
}

type ebnfp struct {
	name string
	out  string
}

func ebnf(n node) string {
	outp := []*ebnfp{}
	switch n.(type) {
	case *strct:
		buildEBNF(true, n, map[node]bool{}, nil, &outp)
		out := []string{}
		for _, p := range outp {
			out = append(out, fmt.Sprintf("%s = %s .", p.name, p.out))
		}
		return strings.Join(out, "\n")

	default:
		out := &ebnfp{}
		buildEBNF(true, n, map[node]bool{}, out, &outp)
		return out.out
	}
}

func buildEBNF(root bool, n node, seen map[node]bool, p *ebnfp, outp *[]*ebnfp) {
	switch n := n.(type) {
	case *disjunction:
		if !root {
			p.out += "("
		}
		for i, next := range n.nodes {
			if i > 0 {
				p.out += " | "
			}
			buildEBNF(false, next, seen, p, outp)
		}
		if !root {
			p.out += ")"
		}

	case *union:
		name := strings.ToUpper(n.typ.Name()[:1]) + n.typ.Name()[1:]
		if p != nil {
			p.out += name
		}
		if seen[n] {
			return
		}
		p = &ebnfp{name: name}
		*outp = append(*outp, p)
		seen[n] = true
		for i, next := range n.disjunction.nodes {
			if i > 0 {
				p.out += " | "
			}
			buildEBNF(false, next, seen, p, outp)
		}

	case *custom:
		name := strings.ToUpper(n.typ.Name()[:1]) + n.typ.Name()[1:]
		p.out += name

	case *strct:
		name := strings.ToUpper(n.typ.Name()[:1]) + n.typ.Name()[1:]
		if p != nil {
			p.out += name
		}
		if seen[n] {
			return
		}
		seen[n] = true
		p = &ebnfp{name: name}
		*outp = append(*outp, p)
		buildEBNF(true, n.expr, seen, p, outp)

	case *sequence:
		group := n.next != nil && !root
		if group {
			p.out += "("
		}
		for n != nil {
			buildEBNF(false, n.node, seen, p, outp)
			n = n.next
			if n != nil {
				p.out += " "
			}
		}
		if group {
			p.out += ")"
		}

	case *parseable:
		p.out += n.t.Name()

	case *capture:
		buildEBNF(false, n.node, seen, p, outp)

	case *reference:
		p.out += "<" + strings.ToLower(n.identifier) + ">"

	case *negation:
		p.out += "~"
		buildEBNF(false, n.node, seen, p, outp)

	case *literal:
		p.out += fmt.Sprintf("%q", n.s)

	case *group:
		if child, ok := n.expr.(*group); ok && child.mode == groupMatchOnce {
			buildEBNF(false, child.expr, seen, p, outp)
		} else if child, ok := n.expr.(*capture); ok {
			if grandchild, ok := child.node.(*group); ok && grandchild.mode == groupMatchOnce {
				buildEBNF(false, grandchild.expr, seen, p, outp)
			} else {
				buildEBNF(false, n.expr, seen, p, outp)
			}
		} else {
			buildEBNF(false, n.expr, seen, p, outp)
		}
		switch n.mode {
		case groupMatchNonEmpty:
			p.out += "!"
		case groupMatchZeroOrOne:
			p.out += "?"
		case groupMatchZeroOrMore:
			p.out += "*"
		case groupMatchOneOrMore:
			p.out += "+"
		case groupMatchOnce:
		}

	case *lookaheadGroup:
		if !n.negative {
			p.out += "(?= "
		} else {
			p.out += "(?! "
		}
		buildEBNF(true, n.expr, seen, p, outp)
		p.out += ")"

	default:
		panic(fmt.Sprintf("unsupported node type %T", n))
	}
}
