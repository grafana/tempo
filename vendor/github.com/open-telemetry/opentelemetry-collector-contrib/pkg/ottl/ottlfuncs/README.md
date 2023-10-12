# OTTL Functions

The following functions are intended to be used in implementations of the OpenTelemetry Transformation Language that
interact with OTel data via the Collector's internal data model, [pdata](https://github.com/open-telemetry/opentelemetry-collector/tree/main/pdata).

This document contains documentation for both types of OTTL functions:

- [Functions](#functions) that transform telemetry.
- [Converters](#converters) that provide utilities for transforming telemetry.

## Working with functions

Functions generally expect specific types to be returned by `Paths`.
For these functions, if that type is not returned or if `nil` is returned, the function will error.
Some functions are able to handle different types and will generally convert those types to their desired type.
In these situations the function will error if it does not know how to do the conversion.
Use `ErrorMode` to determine how the `Statement` handles these errors.
See the component-specific guides for how each uses error mode:
- [filterprocessor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/filterprocessor#ottl)
- [routingprocessor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/routingprocessor#tech-preview-opentelemetry-transformation-language-statements-as-routing-conditions)
- [transformprocessor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor#config)

## Editors

Editors are what OTTL uses to transform telemetry.

Editors:

- Are allowed to transform telemetry.  When a Function is invoked the expectation is that the underlying telemetry is modified in some way.
- May have side effects.  Some Functions may generate telemetry and add it to the telemetry payload to be processed in this batch.
- May return values.  Although not common and not required, Functions may return values.

Available Editors:
- [delete_key](#delete_key)
- [delete_matching_keys](#delete_matching_keys)
- [keep_keys](#keep_keys)
- [limit](#limit)
- [merge_maps](#merge_maps)
- [replace_all_matches](#replace_all_matches)
- [replace_all_patterns](#replace_all_patterns)
- [replace_match](#replace_match)
- [replace_pattern](#replace_pattern)
- [set](#set)
- [truncate_all](#truncate_all)

### delete_key

`delete_key(target, key)`

The `delete_key` function removes a key from a `pdata.Map`

`target` is a path expression to a `pdata.Map` type field. `key` is a string that is a key in the map.

The key will be deleted from the map.

Examples:

- `delete_key(attributes, "http.request.header.authorization")`


- `delete_key(resource.attributes, "http.request.header.authorization")`

### delete_matching_keys

`delete_matching_keys(target, pattern)`

The `delete_matching_keys` function removes all keys from a `pdata.Map` that match a regex pattern.

`target` is a path expression to a `pdata.Map` type field. `pattern` is a regex string.

All keys that match the pattern will be deleted from the map.

Examples:

- `delete_key(attributes, "http.request.header.authorization")`


- `delete_key(resource.attributes, "http.request.header.authorization")`

### keep_keys

`keep_keys(target, keys[])`

The `keep_keys` function removes all keys from the `pdata.Map` that do not match one of the supplied keys.

`target` is a path expression to a `pdata.Map` type field. `keys` is a slice of one or more strings.

The map will be changed to only contain the keys specified by the list of strings.

Examples:

- `keep_keys(attributes, ["http.method"])`


- `keep_keys(resource.attributes, ["http.method", "http.route", "http.url"])`

### limit

`limit(target, limit, priority_keys[])`

The `limit` function reduces the number of elements in a `pdata.Map` to be no greater than the limit.

`target` is a path expression to a `pdata.Map` type field. `limit` is a non-negative integer.
`priority_keys` is a list of strings of attribute keys that won't be dropped during limiting.

The number of priority keys must be less than the supplied `limit`.

The map will be mutated such that the number of items does not exceed the limit.
The map is not copied or reallocated.

Which items are dropped is random, provide keys in `priority_keys` to preserve required keys.

Examples:

- `limit(attributes, 100, [])`


- `limit(resource.attributes, 50, ["http.host", "http.method"])`

### merge_maps

`merge_maps(target, source, strategy)`

The `merge_maps` function merges the source map into the target map using the supplied strategy to handle conflicts.

`target` is a `pdata.Map` type field. `source` is a `pdata.Map` type field. `strategy` is a string that must be one of `insert`, `update`, or `upsert`.

If strategy is:
- `insert`: Insert the value from `source` into `target` where the key does not already exist.
- `update`: Update the entry in `target` with the value from `source` where the key does exist.
- `upsert`: Performs insert or update. Insert the value from `source` into `target` where the key does not already exist and update the entry in `target` with the value from `source` where the key does exist.

`merge_maps` is a special case of the [`set` function](#set). If you need to completely override `target`, use `set` instead.

Examples:

- `merge_maps(attributes, ParseJSON(body), "upsert")`


- `merge_maps(attributes, ParseJSON(attributes["kubernetes"]), "update")`


- `merge_maps(attributes, resource.attributes, "insert")`

### replace_all_matches

`replace_all_matches(target, pattern, replacement)`

The `replace_all_matches` function replaces any matching string value with the replacement string.

`target` is a path expression to a `pdata.Map` type field. `pattern` is a string following [filepath.Match syntax](https://pkg.go.dev/path/filepath#Match). `replacement` is either a path expression to a string telemetry field or a literal string.

Each string value in `target` that matches `pattern` will get replaced with `replacement`. Non-string values are ignored.

There is currently a bug with OTTL that does not allow the pattern to end with `\\"`.
[See Issue 23238 for details](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/23238).

Examples:

- `replace_all_matches(attributes, "/user/*/list/*", "/user/{userId}/list/{listId}")`

### replace_all_patterns

`replace_all_patterns(target, mode, regex, replacement)`

The `replace_all_patterns` function replaces any segments in a string value or key that match the regex pattern with the replacement string.

`target` is a path expression to a `pdata.Map` type field. `regex` is a regex string indicating a segment to replace. `replacement` is either a path expression to a string telemetry field or a literal string.

`mode` determines whether the match and replace will occur on the map's value or key. Valid values are `key` and `value`.

If one or more sections of `target` match `regex` they will get replaced with `replacement`.

The `replacement` string can refer to matched groups using [regexp.Expand syntax](https://pkg.go.dev/regexp#Regexp.Expand).

There is currently a bug with OTTL that does not allow the pattern to end with `\\"`.
If your pattern needs to end with backslashes, add something inconsequential to the end of the pattern such as `{1}`, `$`, or `.*`.
[See Issue 23238 for details](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/23238).

Examples:

- `replace_all_patterns(attributes, "value", "/account/\\d{4}", "/account/{accountId}")`
- `replace_all_patterns(attributes, "key", "/account/\\d{4}", "/account/{accountId}")`
- `replace_all_patterns(attributes, "key", "^kube_([0-9A-Za-z]+_)", "k8s.$$1.")`

Note that when using OTTL within the collector's configuration file, `$` must be escaped to `$$` to bypass
environment variable substitution logic. To input a literal `$` from the configuration file, use `$$$`.
If using OTTL outside of collector configuration, `$` should not be escaped and a literal `$` can be entered using `$$`.

### replace_match

`replace_match(target, pattern, replacement)`

The `replace_match` function allows replacing entire strings if they match a glob pattern.

`target` is a path expression to a telemetry field. `pattern` is a string following [filepath.Match syntax](https://pkg.go.dev/path/filepath#Match). `replacement` is either a path expression to a string telemetry field or a literal string.

If `target` matches `pattern` it will get replaced with `replacement`.

There is currently a bug with OTTL that does not allow the pattern to end with `\\"`.
[See Issue 23238 for details](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/23238).

Examples:

- `replace_match(attributes["http.target"], "/user/*/list/*", "/user/{userId}/list/{listId}")`

### replace_pattern

`replace_pattern(target, regex, replacement)`

The `replace_pattern` function allows replacing all string sections that match a regex pattern with a new value.

`target` is a path expression to a telemetry field. `regex` is a regex string indicating a segment to replace. `replacement` is either a path expression to a string telemetry field or a literal string.

If one or more sections of `target` match `regex` they will get replaced with `replacement`.

The `replacement` string can refer to matched groups using [regexp.Expand syntax](https://pkg.go.dev/regexp#Regexp.Expand).

There is currently a bug with OTTL that does not allow the pattern to end with `\\"`.
If your pattern needs to end with backslashes, add something inconsequential to the end of the pattern such as `{1}`, `$`, or `.*`.
[See Issue 23238 for details](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/23238).

Examples:

- `replace_pattern(resource.attributes["process.command_line"], "password\\=[^\\s]*(\\s?)", "password=***")`
- `replace_pattern(name, "^kube_([0-9A-Za-z]+_)", "k8s.$$1.")`

Note that when using OTTL within the collector's configuration file, `$` must be escaped to `$$` to bypass
environment variable substitution logic. To input a literal `$` from the configuration file, use `$$$`.
If using OTTL outside of collector configuration, `$` should not be escaped and a literal `$` can be entered using `$$`.

### set

`set(target, value)`

The `set` function allows users to set a telemetry field using a value.

`target` is a path expression to a telemetry field. `value` is any value type. If `value` resolves to `nil`, e.g. it references an unset map value, there will be no action.

How the underlying telemetry field is updated is decided by the path expression implementation provided by the user to the `ottl.ParseStatements`.

Examples:

- `set(attributes["http.path"], "/foo")`


- `set(name, attributes["http.route"])`


- `set(trace_state["svc"], "example")`


- `set(attributes["source"], trace_state["source"])`

### truncate_all

`truncate_all(target, limit)`

The `truncate_all` function truncates all string values in a `pdata.Map` so that none are longer than the limit.

`target` is a path expression to a `pdata.Map` type field. `limit` is a non-negative integer.

The map will be mutated such that the number of characters in all string values is less than or equal to the limit. Non-string values are ignored.

Examples:

- `truncate_all(attributes, 100)`


- `truncate_all(resource.attributes, 50)`

## Converters

Converters are pure functions that take OTTL values as input and output a single value for use within a statement.
Unlike functions, they do not modify any input telemetry and always return a value.

Available Converters:
- [Concat](#concat)
- [ConvertCase](#convertcase)
- [ExtractPatterns](#extractpatterns)
- [FNV](#fnv)
- [Hours](#hours)
- [Duration](#duration)
- [Int](#int)
- [IsMap](#ismap)
- [IsMatch](#ismatch)
- [IsString](#isstring)
- [Len](#len)
- [Log](#log)
- [Microseconds](#microseconds)
- [Milliseconds](#milliseconds)
- [Minutes](#minutes)
- [Nanoseconds](#nanoseconds)
- [ParseJSON](#parsejson)
- [Seconds](#seconds)
- [SHA1](#sha1)
- [SHA256](#sha256)
- [SpanID](#spanid)
- [Split](#split)
- [Substring](#substring)
- [Time](#time)
- [UnixMicro](#unixmicro)
- [UnixMilli](#unixmilli)
- [UnixNano](#unixnano)
- [UnixSeconds](#unixseconds)
- [UUID](#UUID)

### Concat

`Concat(values[], delimiter)`

The `Concat` Converter takes a delimiter and a sequence of values and concatenates their string representation. Unsupported values, such as lists or maps that may substantially increase payload size, are not added to the resulting string.

`values` is a list of values passed as arguments. It supports paths, primitive values, and byte slices (such as trace IDs or span IDs).

`delimiter` is a string value that is placed between strings during concatenation. If no delimiter is desired, then simply pass an empty string.

Examples:

- `Concat([attributes["http.method"], attributes["http.path"]], ": ")`


- `Concat([name, 1], " ")`


- `Concat(["HTTP method is: ", attributes["http.method"]], "")`

### ConvertCase

`ConvertCase(target, toCase)`

The `ConvertCase` Converter converts the `target` string into the desired case `toCase`.

`target` is a string. `toCase` is a string.

If the `target` is not a string or does not exist, the `ConvertCase` Converter will return an error.

`toCase` can be:

- `lower`: Converts the `target` string to lowercase (e.g. `MY_METRIC` to `my_metric`)
- `upper`: Converts the `target` string to uppercase (e.g. `my_metric` to `MY_METRIC`)
- `snake`: Converts the `target` string to snakecase (e.g. `myMetric` to `my_metric`)
- `camel`: Converts the `target` string to camelcase (e.g. `my_metric` to `MyMetric`)

If `toCase` is any value other than the options above, the `ConvertCase` Converter will return an error during collector startup.

Examples:

- `ConvertCase(metric.name, "snake")`

### Duration

`Duration(duration)`

The `Duration` Converter takes a string representation of a duration and converts it to a [Golang `time.duration`](https://pkg.go.dev/time#ParseDuration).

`duration` is a string. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

If either `duration` is nil or is in a format that cannot be converted to Golang `time.duration`, an error is returned.

Examples:

- `Duration("3s")`
- `Duration("333ms")`
- `Duration("1000000h")`

### ExtractPatterns

`ExtractPatterns(target, pattern)`

The `ExtractPatterns` Converter returns a `pcommon.Map` struct that is a result of extracting named capture groups from the target string. If not matches are found then an empty `pcommon.Map` is returned.

`target` is a Getter that returns a string. `pattern` is a regex string. 

If `target` is not a string or nil `ExtractPatterns` will return an error. If `pattern` does not contain at least 1 named capture group then `ExtractPatterns` will error on startup. 

Examples:

- `ExtractPatterns(attributes["k8s.change_cause"], "GIT_SHA=(?P<git.sha>\w+)")`

- `ExtractPatterns(body, "^(?P<timestamp>\\w+ \\w+ [0-9]+:[0-9]+:[0-9]+) (?P<hostname>([A-Za-z0-9-_]+)) (?P<process>\\w+)(\\[(?P<pid>\\d+)\\])?: (?P<message>.*)$")`

### FNV

`FNV(value)`

The `FNV` Converter converts the `value` to an FNV hash/digest.

The returned type is int64.

`value` is either a path expression to a string telemetry field or a literal string. If `value` is another type an error is returned.

If an error occurs during hashing it will be returned.

Examples:

- `FNV(attributes["device.name"])`


- `FNV("name")`

### Hours

`Hours(value)`

The `Hours` Converter returns the duration as a floating point number of hours.

`value` is a `time.Duration`. If `value` is another type an error is returned.

The returned type is `float64`.

Examples:

- `Hours(Duration("1h"))`

### Int

`Int(value)`

The `Int` Converter converts the `value` to int type.

The returned type is int64.

The input `value` types:
* float64. Fraction is discharged (truncation towards zero).
* string. Trying to parse an integer from string if it fails then nil will be returned.
* bool. If `value` is true, then the function will return 1 otherwise 0.
* int64. The function returns the `value` without changes.

If `value` is another type or parsing failed nil is always returned.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

Examples:

- `Int(attributes["http.status_code"])`


- `Int("2.0")`

### IsMap

`IsMap(value)`

The `IsMap` Converter returns true if the given value is a map.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

If `value` is a `map[string]any` or a `pcommon.ValueTypeMap` then returns `true`, otherwise returns `false`.

Examples:

- `IsMap(body)`


- `IsMap(attributes["maybe a map"])`

### IsMatch

`IsMatch(target, pattern)`

The `IsMatch` Converter returns true if the `target` matches the regex `pattern`.

`target` is either a path expression to a telemetry field to retrieve or a literal string. `pattern` is a regexp pattern.
The matching semantics are identical to `regexp.MatchString`.

The function matches the target against the pattern, returning true if the match is successful and false otherwise.
If target is not a string, it will be converted to one:

- booleans, ints and floats will be converted using `strconv`
- byte slices will be encoded using base64
- OTLP Maps and Slices will be JSON encoded
- other OTLP Values will use their canonical string representation via `AsString`

If target is nil, false is always returned.

There is currently a bug with OTTL that does not allow the target string to end with `\\"`.
[See Issue 23238 for details](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/23238).

Examples:

- `IsMatch(attributes["http.path"], "foo")`


- `IsMatch("string", ".*ring")`

### IsString

`IsString(value)`

The `IsString` Converter returns true if the given value is a string.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

If `value` is a `string` or a `pcommon.ValueTypeStr` then returns `true`, otherwise returns `false`.

Examples:

- `IsString(body)`

- `IsString(attributes["maybe a string"])`

### Len

`Len(target)`

The `Len` Converter returns the int64 length of the target string or slice.

`target` is either a `string`, `slice`, `map`, `pcommon.Slice`, `pcommon.Map`, or `pcommon.Value` with type `pcommon.ValueTypeStr`, `pcommon.ValueTypeSlice`, or `pcommon.ValueTypeMap`.

If the `target` is not an acceptable type, the `Len` Converter will return an error.

Examples:

- `Len(body)`

### Log

`Log(value)`

The `Log` Converter returns the logarithm of the `target`.

`target` is either a path expression to a telemetry field to retrieve or a literal.

The function take the logarithm of the target, returning an error if the target is less than or equal to zero.

If target is not a float64, it will be converted to one:

- int64s are converted to float64s
- strings are converted using `strconv`
- booleans are converted using `1` for `true` and `0` for `false`.  This means passing `false` to the function will cause an error.
- int, float, string, and bool OTLP Values are converted following the above rules depending on their type.  Other types cause an error.

If target is nil an error is returned.

Examples:

- `Log(attributes["duration_ms"])`


- `Int(Log(attributes["duration_ms"])`

### Microseconds

`Microseconds(value)`

The `Microseconds` Converter returns the duration as an integer millisecond count.

`value` is a `time.Duration`. If `value` is another type an error is returned.

The returned type is `int64`.

Examples:

- `Microseconds(Duration("1h"))`

### Milliseconds

`Milliseconds(value)`

The `Milliseconds` Converter returns the duration as an integer millisecond count.

`value` is a `time.Duration`. If `value` is another type an error is returned.

The returned type is `int64`.

Examples:

- `Milliseconds(Duration("1h"))`

### Minutes

`Minutes(value)`

The `Minutes` Converter returns the duration as a floating point number of minutes.

`value` is a `time.Duration`. If `value` is another type an error is returned.

The returned type is `float64`.

Examples:

- `Minutes(Duration("1h"))`

### Nanoseconds

`Nanoseconds(value)`

The `Nanoseconds` Converter returns the duration as an integer nanosecond count.

`value` is a `time.Duration`. If `value` is another type an error is returned.

The returned type is `int64`.

Examples:

- `Nanoseconds(Duration("1h"))`

### ParseJSON

`ParseJSON(target)`

The `ParseJSON` Converter returns a `pcommon.Map` struct that is a result of parsing the target string as JSON

`target` is a Getter that returns a string. This string should be in json format.
If `target` is not a string, nil, or cannot be parsed as JSON, `ParseJSON` will return an error.

Unmarshalling is done using [jsoniter](https://github.com/json-iterator/go).
Each JSON type is converted into a `pdata.Value` using the following map:

```
JSON boolean -> bool
JSON number  -> float64
JSON string  -> string
JSON null    -> nil
JSON arrays  -> pdata.SliceValue
JSON objects -> map[string]any
```

Examples:

- `ParseJSON("{\"attr\":true}")`


- `ParseJSON(attributes["kubernetes"])`


- `ParseJSON(body)`

### Seconds

`Seconds(value)`

The `Seconds` Converter returns the duration as a floating point number of seconds.

`value` is a `time.Duration`. If `value` is another type an error is returned.

The returned type is `float64`.

Examples:

- `Seconds(Duration("1h"))`

### SHA1

`SHA1(value)`

The `SHA1` Converter converts the `value` to a sha1 hash/digest.

The returned type is string.

`value` is either a path expression to a string telemetry field or a literal string. If `value` is another type an error is returned.

If an error occurs during hashing it will be returned.

Examples:

- `SHA1(attributes["device.name"])`


- `SHA1("name")`

**Note:** According to the National Institute of Standards and Technology (NIST), SHA1 is no longer a recommended hash function. It should be avoided except when required for compatibility. New uses should prefer FNV whenever possible.

### SHA256

`SHA256(value)`

The `SHA256` Converter converts the `value` to a sha256 hash/digest.

The returned type is string.

`value` is either a path expression to a string telemetry field or a literal string. If `value` is another type an error is returned.

If an error occurs during hashing it will be returned.

Examples:

- `SHA256(attributes["device.name"])`


- `SHA256("name")`

**Note:** According to the National Institute of Standards and Technology (NIST), SHA256 is no longer a recommended hash function. It should be avoided except when required for compatibility. New uses should prefer FNV whenever possible.

### SpanID

`SpanID(bytes)`

The `SpanID` Converter returns a `pdata.SpanID` struct from the given byte slice.

`bytes` is a byte slice of exactly 8 bytes.

Examples:

- `SpanID(0x0000000000000000)`

### Split

`Split(target, delimiter)`

The `Split` Converter separates a string by the delimiter, and returns an array of substrings.

`target` is a string. `delimiter` is a string.

If the `target` is not a string or does not exist, the `Split` Converter will return an error.

There is currently a bug with OTTL that does not allow the target string to end with `\\"`.
[See Issue 23238 for details](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/23238).

Examples:

- ```Split("A|B|C", "|")```

### Substring

`Substring(target, start, length)`

The `Substring` Converter returns a substring from the given start index to the specified length.

`target` is a string. `start` and `length` are `int64`.

If `target` is not a string or is nil, an error is returned.
If the start/length exceed the length of the `target` string, an error is returned.

Examples:

- `Substring("123456789", 0, 3)`

### Time

The `Time` Converter takes a string representation of a time and converts it to a Golang `time.Time`.

`time` is a string. `format` is a string.

If either `time` or `format` are nil, an error is returned. The parser used is the parser at [internal/coreinternal/parser](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/internal/coreinternal/timeutils). If the time and format do not follow the parsing rules used by this parser, an error is returned.

Examples:

- `Time("02/04/2023", "%m/%d/%Y")`

### TraceID

`TraceID(bytes)`

The `TraceID` Converter returns a `pdata.TraceID` struct from the given byte slice.

`bytes` is a byte slice of exactly 16 bytes.

Examples:

- `TraceID(0x00000000000000000000000000000000)`

### UnixMicro

`UnixMicro(value)`

The `UnixMicro` Converter returns the time as a Unix time, the number of microseconds elapsed since January 1, 1970 UTC.

`value` is a `time.Time`. If `value` is another type an error is returned.

The returned type is `int64`.

Examples:

- `UnixMicro(Time("02/04/2023", "%m/%d/%Y"))`

### UnixMilli

`UnixMilli(value)`

The `UnixMilli` Converter returns the time as a Unix time, the number of milliseconds elapsed since January 1, 1970 UTC.

`value` is a `time.Time`. If `value` is another type an error is returned.

The returned type is `int64`.

Examples:

- `UnixMilli(Time("02/04/2023", "%m/%d/%Y"))`

### UnixNano

`UnixNano(value)`

The `UnixNano` Converter returns the time as a Unix time, the number of nanoseconds elapsed since January 1, 1970 UTC.

`value` is a `time.Time`. If `value` is another type an error is returned.

The returned type is `int64`.

Examples:

- `UnixNano(Time("02/04/2023", "%m/%d/%Y"))`

### UnixSeconds

`UnixSeconds(value)`

The `UnixSeconds` Converter returns the time as a Unix time, the number of seconds elapsed since January 1, 1970 UTC.

`value` is a `time.Time`. If `value` is another type an error is returned.

The returned type is `int64`.

Examples:

- `UnixSeconds(Time("02/04/2023", "%m/%d/%Y"))`

### UUID

`UUID()`

The `UUID` function generates a v4 uuid string.

## Function syntax

Functions should be named and formatted according to the following standards.
- Function names MUST start with a verb unless it is a Factory that creates a new type.
- Converters MUST be UpperCamelCase.
- Function names that contain multiple words MUST separate those words with `_`.
- Functions that interact with multiple items MUST have plurality in the name.  Ex: `truncate_all`, `keep_keys`, `replace_all_matches`.
- Functions that interact with a single item MUST NOT have plurality in the name.  If a function would interact with multiple items due to a condition, like `where`, it is still considered singular.  Ex: `set`, `delete`, `replace_match`.
- Functions that change a specific target MUST set the target as the first parameter.
