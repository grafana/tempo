# traceql — Test Specifications (Metrics Math)

This document defines the required tests for the metrics math subsystem of the
`traceql` package. It complements SPECS.md (contracts) and NOTES.md (rationale).

Each test is identified as `TRQL-T-<NN>`.

---

## 1. Parse Tests — Valid Expressions

### TRQL-T-01: TestMetricsMathExpression (existing — parse_test.go:1919)

**Scenario:** Binary arithmetic expressions parse into the correct AST structure.

**Setup:**
- Input strings: `({} | count_over_time()) / ({} | count_over_time())`,
  `({} | rate()) + ({} | rate())`, multiplication, subtraction, nested division,
  grouped division.

**Assertions:**
- Parsed `*RootExpr` deep-equals expected structure built via `newRootExprMath` / `newWrappedMetricsPipeline`.
- `IsMath()` returns `true` for binary expressions.
- `IsMath()` returns `false` for single-leaf `metricsExpression`.

---

### TRQL-T-02: TestMetricsMathExpressionErrors (existing — parse_test.go:2007)

**Scenario:** Invalid math expression syntax produces parse errors.

**Setup:**
- Missing outer parens: `{} | count_over_time() / {} | count_over_time()`
- Incomplete expression: `({} | count_over_time()) / `
- Improper nesting.

**Assertions:**
- `ExtractFetchSpansRequest` returns a non-nil error for each case.

---

### TRQL-T-03: TestMetricsMathExpressionString (existing — parse_test.go:2022)

**Scenario:** `String()` on a math `RootExpr` produces a canonical representation.

**Setup:**
- Parse `({ status = error } | count_over_time()) / ({ true } | count_over_time())`
- Parse a nested expression.

**Assertions:**
- `RootExpr.String()` equals the canonical form.
- Round-trip: `Parse(expr.String())` deep-equals the original.

---

## 2. Evaluation Tests — `applyArithmeticOp`

### TRQL-T-04: TestApplyArithmeticOpBasic (missing — add to ast_metrics_math_test.go)

**Scenario:** Correct scalar arithmetic for all four operators.

**Setup:**
- `(op=OpAdd, lhs=3.0, rhs=2.0)` → expected `5.0`
- `(op=OpSub, lhs=3.0, rhs=2.0)` → expected `1.0`
- `(op=OpMult, lhs=3.0, rhs=2.0)` → expected `6.0`
- `(op=OpDiv, lhs=3.0, rhs=2.0)` → expected `1.5`

**Assertions:**
- Result equals expected within `1e-9` tolerance.

---

### TRQL-T-05: TestApplyArithmeticOpNaN (missing — add to ast_metrics_math_test.go)

**Scenario:** NaN and division-by-zero handling.

**Setup:**
- `(op=OpDiv, lhs=1.0, rhs=0.0)` → expected `NaN`
- `(op=OpAdd, lhs=NaN, rhs=2.0)` → expected `2.0` (NaN coerced to 0)
- `(op=OpSub, lhs=3.0, rhs=NaN)` → expected `3.0`
- `(op=OpMult, lhs=NaN, rhs=NaN)` → expected `0.0`
- Invalid operator → expected `NaN`

**Assertions:**
- `math.IsNaN(result)` for NaN cases.
- Exact value match for coercion cases.

---

## 3. Evaluation Tests — `applyBinaryOp`

### TRQL-T-06: TestApplyBinaryOpMatchingKeys (missing — add to ast_metrics_math_test.go)

**Scenario:** Series with identical label keys are combined correctly.

**Setup:**
- `lhs` SeriesSet: 3 series each with 4 time-step values.
- `rhs` SeriesSet: same 3 keys, different values.
- `op = OpAdd`.

**Assertions:**
- Result has same 3 keys.
- Each value equals `lhsValue + rhsValue` at each time step.
- No extra series in result.

---

### TRQL-T-07: TestApplyBinaryOpMissingKey (missing — add to ast_metrics_math_test.go)

**Scenario:** Series present in one side but not the other are silently dropped.

**Setup:**
- `lhs`: keys A, B, C.
- `rhs`: keys A, B (C absent).
- `op = OpDiv`.

**Assertions:**
- Result contains only keys A and B.
- Key C is absent from result.

---

### TRQL-T-08: TestApplyBinaryOpNoLabels (missing — add to ast_metrics_math_test.go)

**Scenario:** Fallback to no-labels key (`SeriesMapKey{}`) when exact key absent.

**Setup:**
- `lhs`: one series with no labels (empty key).
- `rhs`: one series with labels `{foo="bar"}`.
- `op = OpMult`.

**Assertions:**
- `getTSMatch` falls back to `SeriesMapKey{}`.
- Result series uses values from the no-labels series.

---

## 4. Evaluation Tests — `mathExpression.process`

### TRQL-T-09: TestMathExpressionLeafProcess (missing — add to ast_metrics_math_test.go)

**Scenario:** Leaf node filters series by `__query_fragment` and strips internal labels.

**Setup:**
- Input SeriesSet: 4 series, 2 tagged with `__query_fragment="frag-A"`, 2 with `"frag-B"`.
- Leaf node with `key = "frag-A"`.

**Assertions:**
- Output contains only the 2 series for `"frag-A"`.
- `__query_fragment` label absent from output series.
- `__name__` label absent from output series.

---

### TRQL-T-10: TestMathExpressionBinaryProcess (missing — add to ast_metrics_math_test.go)

**Scenario:** Binary node fan-outs to both children and combines results.

**Setup:**
- Input: 4 series — 2 with `__query_fragment="frag-A"` (values all 4.0), 2 with
  `"frag-B"` (values all 2.0).
- Binary node: `op=OpDiv`, `lhs=leaf(frag-A)`, `rhs=leaf(frag-B)`.
- Matching label keys between frag-A and frag-B series pairs.

**Assertions:**
- Output: 2 series with values all `2.0` (4.0/2.0).
- No `__query_fragment` in output.
- `__name__` set to `"(fragA_name / fragB_name)"` if both had names.

---

## 5. Validation Tests

### TRQL-T-11: TestMathExpressionValidate (missing — add to ast_metrics_math_test.go)

**Scenario:** validate() rejects non-arithmetic operators.

**Setup:**
- Binary node with `op = OpAnd` (non-arithmetic).

**Assertions:**
- `validate()` returns a non-nil error containing the operator name.

---

### TRQL-T-12: TestMathExpressionValidateLeafFilter (missing)

**Scenario:** validate() propagates leaf filter errors.

**Setup:**
- Leaf node with a filter that returns an error from `validate()`.

**Assertions:**
- `mathExpression.validate()` returns that error unchanged.

---

## 6. RootExpr / IsMath Tests

### TRQL-T-13: TestIsMathFalseForSingleLeaf (missing — add to ast_test.go)

**Scenario:** `IsMath()` returns false after `unwrapSingleMathExpr`.

**Setup:**
- Parse `{} | rate()` (single pipeline, no binary op).

**Assertions:**
- `expr.IsMath() == false`

---

### TRQL-T-14: TestIsMathTrueForBinaryExpr (missing — add to ast_test.go)

**Scenario:** `IsMath()` returns true for binary math expressions.

**Setup:**
- Parse `({} | rate()) + ({} | rate())`.

**Assertions:**
- `expr.IsMath() == true`

---

## 7. Coverage Requirements

- Statement coverage for `ast_metrics_math.go`: ≥ 85%.
- All paths through `applyArithmeticOp` must be exercised (add, sub, mult, div, NaN, div-by-zero, invalid op).
- All paths through `applyBinaryOp` must be exercised (match, no-match, no-labels fallback).
- Both leaf and binary node paths of `mathExpression.process` must be exercised.
- `validate()` error and nil paths must both be covered.
- `IsMath()` must be tested for all three cases: binary math, single-leaf, nil.
