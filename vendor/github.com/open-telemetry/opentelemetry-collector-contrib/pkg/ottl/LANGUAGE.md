## Grammar

OTTL grammar includes function invocations, Values and Boolean Expressions. These parts all fit into a Statement, which is the basis of execution in OTTL.

### Design principles

OTTL is intended as a domain-specific language (DSL) for telemetry mutation and generation,
and is not intended to be used as a general-purpose programming language.

OTTL has been designed to work directly with [pdata](https://github.com/open-telemetry/opentelemetry-collector/tree/main/pdata), but can operate on other data formats.

### Editors

Editors are functions that transform the underlying telemetry payload. They may return a value, but typically do not. There must be a single Editor Invocation in each OTTL statement.

An Editor is made up of 2 parts:

- a string identifier. The string identifier must start with a lowercase letter.
- zero or more Values (comma separated) surrounded by parentheses (`()`).

**OTTL has no built-in Editors.**
Users must supply a map between string identifiers and Editor implementations.
OTTL will use this map to determine which implementation to call when executing a Statement.
See [ottlfuncs](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/ottlfuncs#editors) for pre-made, usable Editors.

### Converters

Converters are functions that convert data to a new format before being passed as a function argument or used in a Boolean Expression.
Converters are made up of 3 parts:

- a string identifier. The string identifier must start with an uppercase letter.
- zero or more Values (comma separated) surrounded by parentheses (`()`).
- a combination of zero or more a string key (`["key"]`) or int key (`[0]`)

**OTTL has no built-in Converters.**
Users must include Converters in the same map that Editors are supplied.
OTTL will use this map and reflection to generate Converters that can then be invoked by the user.
See [ottlfuncs](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/ottlfuncs#converters) for pre-made, usable Converters.

When keys are supplied the value returned by the Converter will be indexed by the keys in order.
If keys are supplied to a Converter and the return value cannot be indexed, or if the return value doesn't support the
type of key supplied, OTTL will error. Supported values are:

| Type             | Index Type |
|------------------|------------|
| `pcommon.Map`    | `String`   |
| `map[string]any` | `String`   |
| `pcommon.Slice`  | `Int`      |
| `[]any`          | `Int`      |

Example Converters
- `Int()`
- `IsMatch(field, ".*")`
- `Split(field, ",")[1]`

### Function parameters

The following types are supported for single-value parameters in OTTL functions:

- `Setter`
- `GetSetter`
- `Getter`
- `PMapGetter`
- `FloatGetter`
- `FloatLikeGetter`
- `StringGetter`
- `StringLikeGetter`
- `IntGetter`
- `IntLikeGetter`
- `BoolGetter`
- `BoolLikeGetter`
- `ByteSliceLikeGetter`
- `Enum`
- `string`
- `float64`
- `int64`
- `bool`

For slice parameters, the following types are supported:

- `Getter`
- `PMapGetter`
- `FloatGetter`
- `FloatLikeGetter`
- `StringGetter`
- `StringLikeGetter`
- `IntGetter`
- `IntLikeGetter`
- `string`
- `float64`
- `int64`
- `uint8`. Byte slice literals are parsed as byte slices by OTTL.
- `Getter`

To make a parameter optional, use the `Optional` type, which takes a type argument for the underlying
parameter type. For example, an optional string parameter would be specified as `Optional[string]`.
All optional parameters must be specified after all required parameters.

#### Arguments in invocations

Function arguments must be passed in the order defined in the `Arguments` struct for the function unless they are named, in which case the arguments can come in any order. All named arguments must come after all arguments without
names. Argument names are snake-cased versions of the argument's field name in the function's `Arguments` struct.

When passing optional arguments, all optional arguments preceding a given optional argument must be specified if
the arguments are not named. Passing a named argument allows skipping the preceding optional arguments.

### Values

Values are passed as function parameters or are used in a Boolean Expression. Values can take the form of:

- [Paths](#paths)
- [Lists](#lists)
- [Literals](#literals)
- [Enums](#enums)
- [Converters](#converters)
- [Math Expressions](#math-expressions)
- [Maps](#maps)

### Paths

A Path Value is a reference to a telemetry field. Paths are composed of lowercase identifiers, dots (`.`), and square brackets containing either a string key (`["key"]`), an integer key (`[0]`), or an expression that might be a [Converter](#converters) or another Path.
**The interpretation of a Path is NOT implemented by OTTL**. Instead, users must provide a `PathExpressionParser` that OTTL can use to interpret paths.
As a result, the usage of Path segments is determined by the user. However, it is recommended to use them as follows:

- Identifiers are used to map to a telemetry field.
- Dots (`.`) are used to separate nested fields, please note that the first path segment is interpreted by OTTL as the context identifier.
- Square brackets and keys (`["key"]`) are used to access values within maps.

When accessing a map's value, if the given key does not exist, `nil` will be returned.
This can be used to check for the presence of a key within a map within a [Boolean Expression](#boolean-expressions).

Example Paths
- `metric.name`
- `span.value_double`
- `resource.name`
- `resource.attributes["key"]`
- `log.attributes["nested"]["values"]`
- `datapoint.cache["slice"][1]`

#### Contexts

The package that handles the interpretation of a path is normally called a Context.
Contexts will have an implementation of `PathExpressionParser` that decides how an OTTL Path is interpreted.
The context's implementation will need to make decisions like what a dot (`.`) represents or which paths allow indexing (`["key"]`) and how many indexes.

[There are OpenTelemetry-specific contexts provided for each signal here.](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts)
When using OTTL it is recommended to use these contexts unless you have a specific need.  Check out each context to view the paths it supports.

### Lists

A List Value comprises a sequence of Values.
Currently, list can only be created by the grammar to be used in functions or conditions;
the grammar does not provide an accessor to individual list entries.

Example List Values:
- `[]`
- `[1]`
- `["1", "2", "3"]`
- `["a", attributes["key"], Concat(["a", "b"], "-")]`

### Maps

A Map Value comprises a set of key Value pairs.

Example Map Values:
- `{}`
- `{"foo": "bar"}`
- `{"foo": {"a": 2}}`
- `{"foo": {"a": attributes["key"]}}`

### Literals

Literals are literal interpretations of the Value into a Go value.  Accepted literals are:

- Strings. Strings are represented as literals by surrounding the string in double quotes (`""`).
- Ints.  Ints are represented by any digit, optionally prepended by plus (`+`) or minus (`-`). Internally OTTL represents all ints as `int64`
- Floats.  Floats are represented by digits separated by a dot (`.`), optionally prepended by plus (`+`) or minus (`-`). The leading digit is optional. Internally OTTL represents all Floats as `float64`.
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

### Enums

Enums are uppercase identifiers that get interpreted during parsing and converted to an `int64`. **The interpretation of an Enum is NOT implemented by OTTL.** Instead, the user must provide a `EnumParser` that OTTL can use to interpret the Enum.  The `EnumParser` returns an `int64` instead of a function, which means that the Enum's numeric value is retrieved during parsing instead of during execution.

Within the grammar Enums are always used as `int64`.  As a result, the Enum's symbol can be used as if it is an Int value.

When defining an OTTL function, if the function needs to take an Enum then the function must use the `Enum` type for that argument, not an `int64`.

### Math Expressions

Math Expressions represent arithmetic calculations.  They support `+`, `-`, `*`, and `/`, along with `()` for grouping.

Math Expressions currently support `int64`, `float64`, `time.Time` and `time.Duration`.
For `time.Time` and `time.Duration`, only `+` and `-` are supported with the following rules:
- A `time.Time` `-` a `time.Time` yields a `time.Duration`.
- A `time.Duration` `+` a `time.Time` yields a `time.Time`.
- A `time.Time` `+`  a `time.Duration` yields a `time.Time`.
- A `time.Time` `-`  a `time.Duration` yields a `time.Time`.
- A `time.Duration` `+` a `time.Duration` yields a `time.Duration`.
- A `time.Duration` `-` a `time.Duration` yields a `time.Duration`.

Math Expressions support `Paths` and `Editors` that return supported types.
Note that `*` and `/` take precedence over `+` and `-`.
Also note that `time.Time` and `time.Duration` can only be used with `+` and `-`.
Operations that share the same level of precedence will be executed in the order that they appear in the Math Expression.
Math Expressions can be grouped with parentheses to override evaluation precedence.
Math Expressions that mix `int64` and `float64` will result in an error.
It is up to the function using the Math Expression to determine what to do with that error and the default return value of `nil`.
Division by zero is gracefully handled with an error, but other arithmetic operations that would result in a panic will still result in a panic.
Division of integers results in an integer and follows Go's rules for division of integers.

Since Math Expressions support `Path`s and `Converter`s as input, they are evaluated during data processing.
__As a result, in order for a function to be able to accept an Math Expressions as a parameter it must use a `Getter`.__

Example Math Expressions
- `1 + 1`
- `end_time_unix_nano - end_time_unix_nano`
- `sum([1, 2, 3, 4]) + (10 / 1) - 1`


### Boolean Expressions

Boolean Expressions allow a decision to be made about whether an Editor should be called. Boolean Expressions are optional.  When used, the parsed statement will include a `Condition`, which can be used to evaluate the result of the statement's Boolean Expression. Boolean Expressions always evaluate to a boolean value (true or false).

Boolean Expressions consist of the literal string `where` followed by one or more Booleans (see below).
Booleans can be joined with the literal strings `and` and `or`.
Booleans can be negated with the literal string `not`.
Note that `not` has the highest precedence and `and` Boolean Expressions have higher precedence than `or`.
Boolean Expressions can be grouped with parentheses to override evaluation precedence.

### Booleans

Booleans can be either:
- A literal boolean value (`true` or `false`).
- A Converter that returns a boolean value (`true` or `false`).
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
- `not (IsMatch(name, "http_.*") and kind > 0)`

## Comparison Rules

The table below describes what happens when two Values are compared. Value types are provided by the user of OTTL. All of the value types supported by OTTL are listed in this table.

If numeric values are of different types, they are compared as `float64`.

For numeric values and strings, the comparison rules are those implemented by Go. Numeric values are done with signed comparisons. For binary values, `false` is considered to be less than `true`.

For values that are not one of the basic primitive types, the only valid comparisons are Equal and Not Equal, which are implemented using Go's standard `==` and `!=` operators.

A `not equal` notation in the table below means that the "!=" operator returns true, but any other operator returns false. Note that a nil byte array is considered equivalent to nil.

The `time.Time` and `time.Duration` types are compared using comparison functions from their respective packages. For more details on how those comparisons work, see the [Golang Time package](https://pkg.go.dev/time).

| base type      | bool        | int64               | float64             | string                          | Bytes                    | nil                    | time.Time                            | time.Duration                                        | map[string]any                                               | pcommon.Map                                                  | []any                                                          | pcommon.Slice                                                  |
|----------------|-------------|---------------------|---------------------|---------------------------------|--------------------------|------------------------|--------------------------------------|------------------------------------------------------|--------------------------------------------------------------|--------------------------------------------------------------|----------------------------------------------------------------|----------------------------------------------------------------|
| bool           | normal, T>F | not equal           | not equal           | not equal                       | not equal                | not equal              | not equal                            | not equal                                            | not equal                                                    | not equal                                                    | not equal                                                      | not equal                                                      |
| int64          | not equal   | compared as largest | compared as float64 | not equal                       | not equal                | not equal              | not equal                            | not equal                                            | not equal                                                    | not equal                                                    | not equal                                                      | not equal                                                      |
| float64        | not equal   | compared as float64 | compared as largest | not equal                       | not equal                | not equal              | not equal                            | not equal                                            | not equal                                                    | not equal                                                    | not equal                                                      | not equal                                                      |
| string         | not equal   | not equal           | not equal           | normal (compared as Go strings) | not equal                | not equal              | not equal                            | not equal                                            | not equal                                                    | not equal                                                    | not equal                                                      | not equal                                                      |
| Bytes          | not equal   | not equal           | not equal           | not equal                       | byte-for-byte comparison | []byte(nil) == nil     | not equal                            | not equal                                            | not equal                                                    | not equal                                                    | not equal                                                      | not equal                                                      |
| nil            | not equal   | not equal           | not equal           | not equal                       | []byte(nil) == nil       | true for equality only | not equal                            | not equal                                            | not equal                                                    | not equal                                                    | not equal                                                      | not equal                                                      |
| time.Time      | not equal   | not equal           | not equal           | not equal                       | not equal                | not equal              | uses `time.Equal()`to check equality | not equal                                            | not equal                                                    | not equal                                                    | not equal                                                      | not equal                                                      |
| time.Duration  | not equal   | not equal           | not equal           | not equal                       | not equal                | not equal              | not equal                            | uses `time.Before()` and `time.After` for comparison | not equal                                                    | not equal                                                    | not equal                                                      | not equal                                                      |
| map[string]any | not equal   | not equal           | not equal           | not equal                       | not equal                | not equal              | not equal                            | not equal                                            | uses reflect.DeepEqual for comparison                        | convert to raw map and uses reflect.DeepEqual for comparison | not equal                                                      | not equal                                                      |
| pcommon.Map    | not equal   | not equal           | not equal           | not equal                       | not equal                | not equal              | not equal                            | not equal                                            | convert to raw map and uses reflect.DeepEqual for comparison | uses pcommon.Map Equal for comparison                        | not equal                                                      | not equal                                                      |
| []any          | not equal   | not equal           | not equal           | not equal                       | not equal                | not equal              | not equal                            | not equal                                            | not equal                                                    | not equal                                                    | uses reflect.DeepEqual for comparison                          | convert to raw slice and uses reflect.DeepEqual for comparison |
| pcommon.Slice  | not equal   | not equal           | not equal           | not equal                       | not equal                | not equal              | not equal                            | not equal                                            | not equal                                                    | not equal                                                    | convert to raw slice and uses reflect.DeepEqual for comparison | uses pcommon.Slice Equal for comparison                        |

Examples:
- `name == "a name"`
- `1 < 2`
- `attributes["custom-attr"] != nil`
- `IsMatch(resource.attributes["host.name"], "pod-*")`

## Accessing signal telemetry

Access to signal telemetry is provided to OTTL functions through a `TransformContext` that is created by the user and passed during statement evaluation. To allow functions to operate on the `TransformContext`, OTTL provides `Getter`, `Setter`, and `GetSetter` interfaces.

### Getters and Setters

Getters allow for reading the following types of data. See the respective section of each Value type for how they are interpreted.

- [Paths](#paths).
- [Enums](#enums).
- [Literals](#literals).
- [Converters](#converters).

It is possible to update the Value in a telemetry field using a Setter. For read and write access, the `GetSetter` interface extends both interfaces.

## Logging inside an OTTL function

To emit logs inside an OTTL function, add a parameter of type [`component.TelemetrySettings`](https://pkg.go.dev/go.opentelemetry.io/collector/component#TelemetrySettings) to the function signature. OTTL will then inject the TelemetrySettings that were passed to `NewParser` into the function.  TelemetrySettings can be used to emit logs.
