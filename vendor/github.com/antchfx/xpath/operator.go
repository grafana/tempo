package xpath

import "math"

// The XPath number operator function list.

type logical func(iterator, string, interface{}, interface{}) bool

var logicalFuncs = [][]logical{
	{cmpBooleanBoolean, nil, nil, nil},
	{nil, cmpNumericNumeric, cmpNumericString, cmpNumericNodeSet},
	{nil, cmpStringNumeric, cmpStringString, cmpStringNodeSet},
	{nil, cmpNodeSetNumeric, cmpNodeSetString, cmpNodeSetNodeSet},
}

// number vs number
func cmpNumberNumberF(op string, a, b float64) bool {
	switch op {
	case "=":
		return a == b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case "!=":
		return a != b
	}
	return false
}

// string vs string
func cmpStringStringF(op string, a, b string) bool {
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	case ">", "<", ">=", "<=":
		return cmpNumberNumberF(op, stringToNumber(a), stringToNumber(b))
	}
	return false
}

func cmpBooleanBooleanF(op string, a, b bool) bool {
	switch op {
	case "or":
		return a || b
	case "and":
		return a && b
	}
	return false
}

func cmpNumericNumeric(t iterator, op string, m, n interface{}) bool {
	a := m.(float64)
	b := n.(float64)
	return cmpNumberNumberF(op, a, b)
}

func cmpNumericString(t iterator, op string, m, n interface{}) bool {
	a := m.(float64)
	b := n.(string)
	return cmpNumberNumberF(op, a, stringToNumber(b))
}

func cmpNumericNodeSet(t iterator, op string, m, n interface{}) bool {
	a := m.(float64)
	b := n.(query)

	for {
		node := b.Select(t)
		if node == nil {
			break
		}
		if cmpNumberNumberF(op, a, stringToNumber(node.Value())) {
			return true
		}
	}
	return false
}

func cmpNodeSetNumeric(t iterator, op string, m, n interface{}) bool {
	a := m.(query)
	b := n.(float64)
	for {
		node := a.Select(t)
		if node == nil {
			break
		}
		if cmpNumberNumberF(op, stringToNumber(node.Value()), b) {
			return true
		}
	}
	return false
}

func cmpNodeSetString(t iterator, op string, m, n interface{}) bool {
	a := m.(query)
	b := n.(string)
	for {
		node := a.Select(t)
		if node == nil {
			break
		}
		if cmpStringStringF(op, node.Value(), b) {
			return true
		}
	}
	return false
}

func cmpNodeSetNodeSet(t iterator, op string, m, n interface{}) bool {
	a := m.(query)
	b := n.(query)
	for {
		x := a.Select(t)
		if x == nil {
			return false
		}

		y := b.Select(t)
		if y == nil {
			return false
		}

		for {
			if cmpStringStringF(op, x.Value(), y.Value()) {
				return true
			}
			if y = b.Select(t); y == nil {
				break
			}
		}
		// reset
		b.Evaluate(t)
	}
}

func cmpStringNumeric(t iterator, op string, m, n interface{}) bool {
	a := m.(string)
	b := n.(float64)
	return cmpNumberNumberF(op, stringToNumber(a), b)
}

func cmpStringString(t iterator, op string, m, n interface{}) bool {
	a := m.(string)
	b := n.(string)
	return cmpStringStringF(op, a, b)
}

func cmpStringNodeSet(t iterator, op string, m, n interface{}) bool {
	a := m.(string)
	b := n.(query)
	for {
		node := b.Select(t)
		if node == nil {
			break
		}
		if cmpStringStringF(op, a, node.Value()) {
			return true
		}
	}
	return false
}

func cmpBooleanBoolean(t iterator, op string, m, n interface{}) bool {
	a := m.(bool)
	b := n.(bool)
	return cmpBooleanBooleanF(op, a, b)
}

// eqFunc is an `=` operator.
func eqFunc(t iterator, m, n interface{}) interface{} {
	t1 := getXPathType(m)
	t2 := getXPathType(n)
	return logicalFuncs[t1][t2](t, "=", m, n)
}

// gtFunc is an `>` operator.
func gtFunc(t iterator, m, n interface{}) interface{} {
	t1 := getXPathType(m)
	t2 := getXPathType(n)
	return logicalFuncs[t1][t2](t, ">", m, n)
}

// geFunc is an `>=` operator.
func geFunc(t iterator, m, n interface{}) interface{} {
	t1 := getXPathType(m)
	t2 := getXPathType(n)
	return logicalFuncs[t1][t2](t, ">=", m, n)
}

// ltFunc is an `<` operator.
func ltFunc(t iterator, m, n interface{}) interface{} {
	t1 := getXPathType(m)
	t2 := getXPathType(n)
	return logicalFuncs[t1][t2](t, "<", m, n)
}

// leFunc is an `<=` operator.
func leFunc(t iterator, m, n interface{}) interface{} {
	t1 := getXPathType(m)
	t2 := getXPathType(n)
	return logicalFuncs[t1][t2](t, "<=", m, n)
}

// neFunc is an `!=` operator.
func neFunc(t iterator, m, n interface{}) interface{} {
	t1 := getXPathType(m)
	t2 := getXPathType(n)
	return logicalFuncs[t1][t2](t, "!=", m, n)
}

// orFunc is an `or` operator.
var orFunc = func(t iterator, m, n interface{}) interface{} {
	t1 := getXPathType(m)
	t2 := getXPathType(n)
	return logicalFuncs[t1][t2](t, "or", m, n)
}

func numericExpr(t iterator, m, n interface{}, cb func(float64, float64) float64) float64 {
	a := asNumber(t, m)
	b := asNumber(t, n)
	return cb(a, b)
}

// plusFunc is an `+` operator.
var plusFunc = func(t iterator, m, n interface{}) interface{} {
	return numericExpr(t, m, n, func(a, b float64) float64 {
		return a + b
	})
}

// minusFunc is an `-` operator.
var minusFunc = func(t iterator, m, n interface{}) interface{} {
	return numericExpr(t, m, n, func(a, b float64) float64 {
		return a - b
	})
}

// mulFunc is an `*` operator.
var mulFunc = func(t iterator, m, n interface{}) interface{} {
	return numericExpr(t, m, n, func(a, b float64) float64 {
		return a * b
	})
}

// divFunc is an `DIV` operator.
var divFunc = func(t iterator, m, n interface{}) interface{} {
	return numericExpr(t, m, n, func(a, b float64) float64 {
		return a / b
	})
}

// modFunc is an 'MOD' operator.
var modFunc = func(t iterator, m, n interface{}) interface{} {
	return numericExpr(t, m, n, func(a, b float64) float64 {
		// XPath 1.0 REC §3.5: truncating IEEE remainder; mod by zero is NaN.
		return math.Mod(a, b)
	})
}
