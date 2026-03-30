# TraceQL Metrics Math — Test Specifications

This document defines the required tests for the metrics math subsystem.
Each test is identified as `TRQL-T-<NN>`.

---

## 1. Parse Tests

### TRQL-T-01: Valid binary arithmetic expressions

**Scenario:** Binary arithmetic expressions parse into the correct structure.

**Inputs:**
- Division: `({} | count_over_time()) / ({} | count_over_time())`
- Addition: `({} | rate()) + ({} | rate())`
- Multiplication, subtraction, nested division, grouped division.

**Assertions:**
- Parsed expression matches the expected structure.
- A binary expression is recognized as a math query.
- A single parenthesized pipeline (no binary operator) is not recognized as math.

---

### TRQL-T-02: Invalid syntax produces parse errors

**Scenario:** Malformed math expressions are rejected.

**Inputs:**
- Missing outer parentheses: `{} | count_over_time() / {} | count_over_time()`
- Incomplete expression: `({} | count_over_time()) /`
- Improper nesting.

**Assertions:**
- Each input returns a non-nil parse error.

---

### TRQL-T-03: String representation is canonical and round-trips

**Scenario:** Converting a math expression to a string and parsing it again
produces an equivalent expression.

**Inputs:**
- `({ status = error } | count_over_time()) / ({ true } | count_over_time())`
- A nested arithmetic expression.

**Assertions:**
- The string form matches the expected canonical representation.
- Parsing the string form produces a structure equal to the original.

---

## 2. Scalar Arithmetic

### TRQL-T-04: Basic operations

**Scenario:** All four arithmetic operators produce correct results for simple inputs.

**Inputs and expected outputs:**
- 3.0 + 2.0 → 5.0
- 3.0 - 2.0 → 1.0
- 3.0 × 2.0 → 6.0
- 3.0 ÷ 2.0 → 1.5

**Assertions:** Results match expected values within floating-point tolerance.

---

### TRQL-T-05: NaN and division-by-zero handling

**Scenario:** Edge cases in scalar arithmetic never panic and produce well-defined results.

**Inputs and expected outputs:**
- 1.0 ÷ 0.0 → NaN
- NaN + 2.0 → 2.0 (NaN treated as 0)
- 3.0 - NaN → 3.0
- NaN × NaN → 0.0
- Invalid operator → NaN

**Assertions:**
- NaN cases produce NaN results.
- NaN-input coercion produces the stated numeric result.

---

## 3. Element-wise Set Combination

### TRQL-T-06: Matching keys are combined

**Scenario:** Series with identical label keys are combined element-wise.

**Setup:**
- Left set: 3 series, each with 4 time-step values.
- Right set: same 3 keys, different values.
- Operator: addition.

**Assertions:**
- Result contains the same 3 keys.
- Each value equals left + right at each time step.
- No extra series in result.

---

### TRQL-T-07: Unmatched series are dropped

**Scenario:** A series present in only one operand is silently dropped.

**Setup:**
- Left set: keys A, B, C.
- Right set: keys A, B (C absent).
- Operator: division.

**Assertions:**
- Result contains only keys A and B.
- Key C is absent.

---

### TRQL-T-08: No-labels fallback

**Scenario:** A no-labels series on one side matches any key on the other side.

**Setup:**
- Left set: one series with no labels.
- Right set: one series with labels `{foo="bar"}`.
- Operator: multiplication.

**Assertions:**
- The result contains one series using the no-labels series as the matching operand.

---

## 4. Math Expression Evaluation

### TRQL-T-09: Leaf node filters and strips labels

**Scenario:** A leaf node selects only its own fragment and strips internal labels.

**Setup:**
- Input: 4 series — 2 tagged with fragment key "A", 2 tagged with fragment key "B".
- Leaf configured for key "A".

**Assertions:**
- Output contains only the 2 series for key "A".
- Fragment label is absent from all output series.
- Name label is absent from all output series.

---

### TRQL-T-10: Binary node fans out and combines

**Scenario:** A binary node passes the full input to both children and combines results.

**Setup:**
- Input: 4 series — 2 tagged with "A" (values 4.0), 2 tagged with "B" (values 2.0).
- Binary node: division, left = leaf("A"), right = leaf("B").
- Left and right series share matching label keys.

**Assertions:**
- Output: 2 series with values 2.0 (4.0 ÷ 2.0).
- Fragment label absent from output.
- Name label set to combined form if both sides had names.

---

### TRQL-T-11: Validation rejects non-arithmetic operators

**Scenario:** A binary node with a non-arithmetic operator is invalid.

**Setup:**
- Binary node with a logical/comparison operator.

**Assertions:**
- Validate returns a non-nil error containing the operator name.

---

### TRQL-T-12: Validation propagates leaf filter errors

**Scenario:** A leaf node propagates errors from its filter.

**Setup:**
- Leaf with a filter that returns an error from validate.

**Assertions:**
- Math expression validate returns that error unchanged.

---

## 5. Math Query Detection

### TRQL-T-13: Single-leaf expression is not math

**Scenario:** A plain metrics query wrapped in parentheses is not a math query.

**Setup:**
- Parse `{} | rate()`.

**Assertions:**
- The expression is not identified as a math query.

---

### TRQL-T-14: Binary expression is math

**Scenario:** A query with a binary arithmetic operator is a math query.

**Setup:**
- Parse `({} | rate()) + ({} | rate())`.

**Assertions:**
- The expression is identified as a math query.

---

## 6. Coverage Requirements

- Statement coverage for the math evaluation module: ≥ 85%.
- All scalar arithmetic paths must be exercised: add, subtract, multiply, divide,
  NaN coercion, division by zero, invalid operator.
- All element-wise combination paths: match, no-match, no-labels fallback.
- Both leaf and binary evaluation paths.
- Validate error and nil paths.
- Math query detection for: binary expression, single-leaf, nil second-stage.
