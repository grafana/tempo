# OpenTelemetry Transformation Language

The OpenTelemetry Transformation Language is a language for transforming open telemetry data based on the [OpenTelemetry Collector Processing Exploration](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/processing.md).

This package reads in OTTL statements and converts them to invokable Booleans and functions based on the OTTL's grammar.

The OTTL is signal agnostic; it is not aware of the type of telemetry on which it will operate.  Instead, the Booleans and functions returned by the package must be passed a `TransformContext`, which provide access to the signal's telemetry. Telemetry data can be accessed and updated through [Getters and Setters](#getters-and-setters).

## Grammar

The OTTL grammar includes Invocations, Values and Boolean Expressions.

### Invocations

Invocations represent a function call that transform the underlying telemetry payload. Invocations are made up of 2 parts:

- a string identifier. The string identifier must start with a lowercase letter.
- zero or more Values (comma separated) surrounded by parentheses (`()`).

**The OTTL does not define any function implementations.**
Users must supply a map between string identifiers and the actual function implementation.
The OTTL will use this map and reflection to generate Invocations, that can then be invoked by the user.
See [ottlfuncs](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/ottlfuncs) for pre-made, usable functions.

Example Invocations
- `route()`
- `set(field, 1)`

#### Invocation parameters

The OTTL will use reflection to determine parameter types when parsing an invocation within a statement.

When developing functions that represent invocations, the following types are supported for single parameter values:
- `Setter`
- `GetSetter`
- `Getter`
- `Enum`
- `string`
- `float64`
- `int64`
- `bool`

For slice parameters, the following types are supported:
- `string`
- `float64`
- `int64`
- `uint8`. Byte slice literals are parsed as byte slices by the OTTL.
- `Getter`

### Values

Values are passed as input to an Invocation or are used in a Boolean Expression. Values can take the form of:
- [Paths](#paths)
- [Lists](#lists)
- [Literals](#literals)
- [Enums](#enums)
- [Converters](#converters)
- [Math Expressions](#math_expressions)

#### Paths

A Path Value is a reference to a telemetry field.  Paths are made up of lowercase identifiers, dots (`.`), and square brackets combined with a string key (`["key"]`).  **The interpretation of a Path is NOT implemented by the OTTL.**  Instead, the user must provide a `PathExpressionParser` that the OTTL can use to interpret paths.  As a result, how the Path parts are used is up to the user.  However, it is recommended, that the parts be used like so:

- Identifiers are used to map to a telemetry field.
- Dots (`.`) are used to separate nested fields.
- Square brackets and keys (`["key"]`) are used to access values within maps.

When accessing a map's value, if the given key does not exist, `nil` will be returned.
This can be used to check for the presence of a key within a map within a [Boolean Expression](#boolean_expressions).

Example Paths
- `name`
- `value_double`
- `resource.name`
- `resource.attributes["key"]`

#### Lists

A List Value comprises a sequence of Values.
Currently, list can only be created by the grammar to be used in functions or conditions;
the grammar does not provide an accessor to individual list entries.

Example List Values:
- `[]`
- `[1]`
- `["1", "2", "3"]`
- `["a", attributes["key"], Concat(["a", "b"], "-")]`

#### Literals

Literals are literal interpretations of the Value into a Go value.  Accepted literals are:

- Strings. Strings are represented as literals by surrounding the string in double quotes (`""`).
- Ints.  Ints are represented by any digit, optionally prepended by plus (`+`) or minus (`-`). Internally the OTTL represents all ints as `int64`
- Floats.  Floats are represented by digits separated by a dot (`.`), optionally prepended by plus (`+`) or minus (`-`). The leading digit is optional. Internally the OTTL represents all Floats as `float64`.
- Bools.  Bools are represented by the exact strings `true` and `false`.
- Nil.  Nil is represented by the exact string `nil`.
- Byte slices.  Byte slices are represented via a hex string prefaced with `0x`

Example Literals
- `"a string"`
- `1`, `-1`
- `1.5`, `-.5`
- `true`, `false`
- `nil`,
- `0x0001`

#### Enums

Enums are uppercase identifiers that get interpreted during parsing and converted to an `int64`. **The interpretation of an Enum is NOT implemented by the OTTL.** Instead, the user must provide a `EnumParser` that the OTTL can use to interpret the Enum.  The `EnumParser` returns an `int64` instead of a function, which means that the Enum's numeric value is retrieved during parsing instead of during execution.

Within the grammar Enums are always used as `int64`.  As a result, the Enum's symbol can be used as if it is an Int value.

When defining a function that will be used as an Invocation by the OTTL, if the function needs to take an Enum then the function must use the `Enum` type for that argument, not an `int64`.

#### Converters

Converters are special functions that convert data to a new format before being passed to an Invocation or Boolean Expression.
Like Invocations, Converters are made up of 2 parts:

- a string identifier. The string identifier must start with an uppercase letter.
- zero or more Values (comma separated) surrounded by parentheses (`()`).

**The OTTL does not define any converter implementations.**
Users must include converters in the same map that invocations are supplied.
The OTTL will use this map and reflection to generate Converters that can then be invoked by the user.
See [ottlfuncs](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/ottlfuncs#converters) for pre-made, usable Converters.

Example Converters
- `Int()`
- `IsMatch(field, ".*")`


#### Math Expressions

Math Expressions represent arithmetic calculations.  They support `+`, `-`, `*`, and `/`, along with `()` for grouping.

Math Expressions currently only support `int64` and `float64`.
Math Expressions support `Paths` and `Invocations` that return supported types.
Note that `*` and `/` take precedence over `+` and `-`.
Operations that share the same level of precedence will be executed in the order that they appear in the Math Expression.
Math Expressions can be grouped with parentheses to override evaluation precedence.
Math Expressions that mix `int64` and `float64` will result in an error.
It is up to the function using the Math Expression to determine what to do with that error and the default return value of `nil`.
Division by zero is gracefully handled with an error, but other arithmetic operations that would result in a panic will still result in a panic.
Division of integers results in an integer and follows Go's rules for division of integers.

Since Math Expressions support `Path` and `Invocation`, they are evaluated during data processing.
__As a result, in order for a function to be able to accept an Math Expressions as a parameter it must use a `Getter`.__

Example Math Expressions
- `1 + 1`
- `end_time_unix_nano - end_time_unix_nano`
- `sum([1, 2, 3, 4]) + (10 / 1) - 1`


### Boolean Expressions

Boolean Expressions allow a decision to be made about whether an Invocation should be called. Boolean Expressions are optional.  When used, the parsed statement will include a `Condition`, which can be used to evaluate the result of the statement's Boolean Expression. Boolean Expressions always evaluate to a boolean value (true or false).

Boolean Expressions consist of the literal string `where` followed by one or more Booleans (see below).
Booleans can be joined with the literal strings `and` and `or`.
Booleans can be negated with the literal string `not`.
Note that `not` has the highest precedence and `and` Boolean Expressions have higher precedence than `or`.
Boolean Expressions can be grouped with parentheses to override evaluation precedence.

### Booleans

Booleans can be either:
- A literal boolean value (`true` or `false`).
- A Comparison, made up of a left Value, an operator, and a right Value. See [Values](#values) for details on what a Value can be.

Operators determine how the two Values are compared.

The valid operators are:

- Equal (`==`). Tests if the left and right Values are equal (see the Comparison Rules below).
- Not Equal (`!=`).  Tests if the left and right Values are not equal.
- Less Than (`<`). Tests if left is less than right.
- Greater Than (`>`). Tests if left is greater than right.
- Less Than or Equal To (`<=`). Tests if left is less than or equal to right.
- Greater Than or Equal to (`>=`). Tests if left is greater than or equal to right.

Booleans can be negated with the `not` keyword such as
- `not true`
- `not name == "foo"`   
- `not (IsMatch(name, "http_.*") == true and kind > 0)`

### Comparison Rules

The table below describes what happens when two Values are compared. Value types are provided by the user of OTTL. All of the value types supported by OTTL are listed in this table.

If numeric values are of different types, they are compared as `float64`.

For numeric values and strings, the comparison rules are those implemented by Go. Numeric values are done with signed comparisons. For binary values, `false` is considered to be less than `true`.

For values that are not one of the basic primitive types, the only valid comparisons are Equal and Not Equal, which are implemented using Go's standard `==` and `!=` operators.

A `not equal` notation in the table below means that the "!=" operator returns true, but any other operator returns false. Note that a nil byte array is considered equivalent to nil.


| base type | bool        | int64               | float64             | string                          | Bytes                    | nil                    |
|-----------|-------------|---------------------|---------------------|---------------------------------|--------------------------|------------------------|
| bool      | normal, T>F | not equal           | not equal           | not equal                       | not equal                | not equal              |
| int64     | not equal   | compared as largest | compared as float64 | not equal                       | not equal                | not equal              |
| float64   | not equal   | compared as float64 | compared as largest | not equal                       | not equal                | not equal              |
| string    | not equal   | not equal           | not equal           | normal (compared as Go strings) | not equal                | not equal              |
| Bytes     | not equal   | not equal           | not equal           | not equal                       | byte-for-byte comparison | []byte(nil) == nil     |
| nil       | not equal   | not equal           | not equal           | not equal                       | []byte(nil) == nil       | true for equality only |

Examples:
- `name == "a name"`
- `1 < 2`
- `attributes["custom-attr"] != nil`
- `IsMatch(resource.attributes["host.name"], "pod-*") == true`

## Accessing signal telemetry

Access to signal telemetry is provided to OTTL functions through a `TransformContext` that is created by the user and passed during statement evaluation. To allow functions to operate on the `TransformContext`, the OTTL provides `Getter`, `Setter`, and `GetSetter` interfaces.

### Getters and Setters

Getters allow for reading the following types of data. See the respective section of each Value type for how they are interpreted.
- [Paths](#paths).
- [Enums](#enums).
- [Literals](#literals).
- [Invocations](#invocations).

It is possible to update the Value in a telemetry field using a Setter. For read and write access, the `GetSetter` interface extends both interfaces.

## Logging inside a OTTL function

To emit logs inside a OTTL function, add a parameter of type [`component.TelemetrySettings`](https://pkg.go.dev/go.opentelemetry.io/collector/component#TelemetrySettings) to the function signature. The OTTL will then inject the TelemetrySettings that were passed to `NewParser` into the function.  TelemetrySettings can be used to emit logs.

## Examples

These examples contain a SQL-like declarative language.  Applied statements interact with only one signal, but statements can be declared across multiple signals.  Functions used in examples are indicative of what could be useful, but are not implemented by the OTTL itself.

### Remove a forbidden attribute

```
traces:
  delete(attributes["http.request.header.authorization"])
metrics:
  delete(attributes["http.request.header.authorization"])
logs:
  delete(attributes["http.request.header.authorization"])
```

### Remove all attributes except for some

```
traces:
  keep_keys(attributes, ["http.method", "http.status_code"])
metrics:
  keep_keys(attributes, ["http.method", "http.status_code"])
logs:
  keep_keys(attributes, ["http.method", "http.status_code"])
```

### Reduce cardinality of an attribute

```
traces:
  replace_match(attributes["http.target"], "/user/*/list/*", "/user/{userId}/list/{listId}")
```

### Reduce cardinality of a span name

```
traces:
  replace_match(name, "GET /user/*/list/*", "GET /user/{userId}/list/{listId}")
```

### Reduce cardinality of any matching attribute

```
traces:
  replace_all_matches(attributes, "/user/*/list/*", "/user/{userId}/list/{listId}")
```

### Decrease the size of the telemetry payload

```
traces:
  delete(resource.attributes["process.command_line"])
metrics:
  delete(resource.attributes["process.command_line"])
logs:
  delete(resource.attributes["process.command_line"])
```

### Attach information from resource into telemetry

```
metrics:
  set(attributes["k8s_pod"], resource.attributes["k8s.pod.name"])
```

### Decorate error spans with additional information

```
traces:
  set(attributes["whose_fault"], "theirs") where attributes["http.status"] == 400 or attributes["http.status"] == 404
  set(attributes["whose_fault"], "ours") where attributes["http.status"] == 500
```

### Update a spans ID

```
logs:
  set(span_id, SpanID(0x0000000000000000))
traces:
  set(span_id, SpanID(0x0000000000000000))
```

### Convert metric name to snake case

```
metrics:
  set(metric.name, ConvertCase(metric.name, "snake"))
```

### Check if an attribute exists

```
traces:
  set(attributes["test-passed"], true) where attributes["target-attribute"] != nil
```
