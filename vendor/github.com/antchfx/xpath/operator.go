package xpath

import "math"

// The XPath number operator function list.

type logical func(iterator, string, interface{}, interface{}) bool

var logicalFuncs = [][]logical{
	{cmpBooleanBoolean, cmpBooleanNumeric, cmpBooleanString, cmpBooleanNodeSet},
	{cmpNumberBoolean, cmpNumericNumeric, cmpNumericString, cmpNumericNodeSet},
	{cmpStringBoolean, cmpStringNumeric, cmpStringString, cmpStringNodeSet},
	{cmpNodeSetBoolean, cmpNodeSetNumeric, cmpNodeSetString, cmpNodeSetNodeSet},
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
	case "=":
		return a == b
	case "!=":
		return a != b
	case "or":
		return a || b
	case "and":
		return a && b
	}
	return false
}

func cmpNumberBoolean(t iterator, op string, m, n interface{}) bool {
	a := m.(float64)
	b := 0.0
	if n.(bool) {
		b = 1.0
	}
	return cmpNumericNumeric(t, op, a, b)
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

	node := b.Select(t)
	if node == nil {
		right := math.NaN()
		return cmpNumberNumberF(op, a, right)
	}

	right := stringToNumber(node.Value())
	return cmpNumberNumberF(op, a, right)
}

func cmpNodeSetBoolean(t iterator, op string, m, n interface{}) bool {
	q := m.(query)
	b := 0.0
	if n.(bool) {
		b = 1.0
	}
	node := q.Select(t)
	if node == nil {
		return cmpNumericNumeric(t, op, math.NaN(), b)
	}
	a := stringToNumber(node.Value())
	return cmpNumericNumeric(t, op, a, b)
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

func cmpStringBoolean(t iterator, op string, m, n interface{}) bool {
	a := stringToNumber(m.(string))
	b := 0.0
	if n.(bool) {
		b = 1.0
	}
	return cmpNumericNumeric(t, op, a, b)
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

	left := stringToNumber(a)

	node := b.Select(t)
	if node == nil {
		return cmpNumberNumberF(op, left, math.NaN())
	}

	right := stringToNumber(node.Value())
	return cmpNumberNumberF(op, left, right)
}

func cmpBooleanBoolean(t iterator, op string, m, n interface{}) bool {
	a := m.(bool)
	b := n.(bool)
	return cmpBooleanBooleanF(op, a, b)
}

func cmpBooleanNumeric(t iterator, op string, m, n interface{}) bool {
	a := n.(float64)
	b := 0.0
	if m.(bool) {
		b = 1.0
	}
	return cmpNumberNumberF(op, a, b)
}

func cmpBooleanString(t iterator, op string, m, n interface{}) bool {
	a := 0.0
	if m.(bool) {
		a = 1.0
	}
	b := stringToNumber(n.(string))
	return cmpNumberNumberF(op, a, b)
}

func cmpBooleanNodeSet(t iterator, op string, m, n interface{}) bool {
	a := m.(bool)
	q := n.(query)

	node := q.Select(t)
	if node == nil {
		return cmpBooleanNumeric(t, op, a, math.NaN())
	}
	num := stringToNumber(node.Value())
	return cmpBooleanNumeric(t, op, a, num)
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
