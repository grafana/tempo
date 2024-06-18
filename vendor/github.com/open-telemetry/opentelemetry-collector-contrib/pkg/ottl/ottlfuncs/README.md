# OTTL Functions

The following functions are intended to be used in implementations of the OpenTelemetry Transformation Language that
interact with OTel data via the Collector's internal data model, [pdata](https://github.com/open-telemetry/opentelemetry-collector/tree/main/pdata).

This document contains documentation for both types of OTTL functions:

- [Editors](#editors) that transform telemetry.
- [Converters](#converters) that provide utilities for transforming telemetry.

## Design principles

For the standard OTTL functions described in this document, we specify design principles to ensure they are always
secure and safe for use:

- Built-in OTTL functions may not access the file system, network, or any other I/O devices.
- Built-in OTTL functions may share information only through their parameters and results.
- Built-in OTTL functions must be terminating; they must not loop forever.

OTTL functions are implemented in Go, and so are only limited by what can be implemented in a Go program.
User-defined OTTL functions may therefore not adhere the above principles.

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

- Are allowed to transform telemetry. When a Function is invoked the expectation is that the underlying telemetry is modified in some way.
- May have side effects. Some Functions may generate telemetry and add it to the telemetry payload to be processed in this batch.
- May return values. Although not common and not required, Functions may return values.

Available Editors:

- [delete_key](#delete_key)
- [delete_matching_keys](#delete_matching_keys)
- [flatten](#flatten)
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

The `delete_key` function removes a key from a `pcommon.Map`

`target` is a path expression to a `pcommon.Map` type field. `key` is a string that is a key in the map.

The key will be deleted from the map.

Examples:


- `delete_key(attributes, "http.request.header.authorization")`

- `delete_key(resource.attributes, "http.request.header.authorization")`

### delete_matching_keys

`delete_matching_keys(target, pattern)`

The `delete_matching_keys` function removes all keys from a `pcommon.Map` that match a regex pattern.

`target` is a path expression to a `pcommon.Map` type field. `pattern` is a regex string.

All keys that match the pattern will be deleted from the map.

Examples:


- `delete_matching_keys(attributes, "(?i).*password.*")`

- `delete_matching_keys(resource.attributes, "(?i).*password.*")`

### flatten

`flatten(target, Optional[prefix], Optional[depth])`

The `flatten` function flattens a `pcommon.Map` by moving items from nested maps to the root. 

`target` is a path expression to a `pcommon.Map` type field. `prefix` is an optional string. `depth` is an optional non-negative int.

For example, the following map

```json
{
  "name": "test",
  "address": {
    "street": "first",
    "house": 1234
  },
  "occupants": ["user 1", "user 2"]
}
```

is converted to 

```json
{
    "name": "test",
    "address.street": "first",
    "address.house": 1234,
    "occupants.0": "user 1",
    "occupants.1": "user 2"
}
```

If `prefix` is supplied, it will be appended to the start of the new keys. This can help you namespace the changes. For example, if in the above example a `prefix` of `app` was configured, the result would be

```json
{
    "app.name": "test",
    "app.address.street": "first",
    "app.address.house": 1234,
    "app.occupants.0": "user 1",
    "app.occupants.1": "user 2"
}
```

If `depth` is supplied, the function will only flatten nested maps up to that depth. For example, if a `depth` of `2` was configured, the following map

```json
{
  "0": {
    "1": {
      "2": {
        "3": {
          "4": "value"
        }
      }
    }
  }
}
```

the result would be

```json
{
  "0.1.2": {
    "3": {
      "4": "value"
    }
  }
}
```

A `depth` of `0` means that no flattening will occur.

Examples:

- `flatten(attributes)`


- `flatten(cache, "k8s", 4)`


- `flatten(body, depth=2)`


### keep_keys

`keep_keys(target, keys[])`

The `keep_keys` function removes all keys from the `pcommon.Map` that do not match one of the supplied keys.

`target` is a path expression to a `pcommon.Map` type field. `keys` is a slice of one or more strings.

The map will be changed to only contain the keys specified by the list of strings.

Examples:

- `keep_keys(attributes, ["http.method"])`


- `keep_keys(resource.attributes, ["http.method", "http.route", "http.url"])`

### limit

`limit(target, limit, priority_keys[])`

The `limit` function reduces the number of elements in a `pcommon.Map` to be no greater than the limit.

`target` is a path expression to a `pcommon.Map` type field. `limit` is a non-negative integer.
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

`target` is a `pcommon.Map` type field. `source` is a `pcommon.Map` type field. `strategy` is a string that must be one of `insert`, `update`, or `upsert`.

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

`replace_all_matches(target, pattern, replacement, Optional[function], Optional[replacementFormat])`

The `replace_all_matches` function replaces any matching string value with the replacement string.

`target` is a path expression to a `pcommon.Map` type field. `pattern` is a string following [filepath.Match syntax](https://pkg.go.dev/path/filepath#Match). `replacement` is either a path expression to a string telemetry field or a literal string. `function` is an optional argument that can take in any Converter that accepts a (`replacement`) string and returns a string. An example is a hash function that replaces any matching string with the hash value of `replacement`.
`replacementFormat` is an optional string argument that specifies the format of the replacement. It must contain exactly one `%s` format specifier as shown in the example below. No other format specifiers are supported.

Each string value in `target` that matches `pattern` will get replaced with `replacement`. Non-string values are ignored.

Examples:

- `replace_all_matches(attributes, "/user/*/list/*", "/user/{userId}/list/{listId}")`
- `replace_all_matches(attributes, "/user/*/list/*", "/user/{userId}/list/{listId}", SHA256, "/user/%s")`

### replace_all_patterns

`replace_all_patterns(target, mode, regex, replacement, Optional[function], Optional[replacementFormat])`

The `replace_all_patterns` function replaces any segments in a string value or key that match the regex pattern with the replacement string.

`target` is a path expression to a `pcommon.Map` type field. `regex` is a regex string indicating a segment to replace. `replacement` is either a path expression to a string telemetry field or a literal string.

`mode` determines whether the match and replace will occur on the map's value or key. Valid values are `key` and `value`.

If one or more sections of `target` match `regex` they will get replaced with `replacement`.

The `replacement` string can refer to matched groups using [regexp.Expand syntax](https://pkg.go.dev/regexp#Regexp.Expand). `replacementFormat` is an optional string argument that specifies the format of the replacement. It must contain exactly one `%s` format specifier as shown in the example below. No other format specifiers are supported.

The `function` is an optional argument that can take in any Converter that accepts a (`replacement`) string and returns a string. An example is a hash function that replaces any matching regex pattern with the hash value of `replacement`.

Examples:

- `replace_all_patterns(attributes, "value", "/account/\\d{4}", "/account/{accountId}")`
- `replace_all_patterns(attributes, "key", "/account/\\d{4}", "/account/{accountId}")`
- `replace_all_patterns(attributes, "key", "^kube_([0-9A-Za-z]+_)", "k8s.$$1.")`
- `replace_all_patterns(attributes, "key", "^kube_([0-9A-Za-z]+_)", "$$1.")`
- `replace_all_patterns(attributes, "key", "^kube_([0-9A-Za-z]+_)", "$$1.", SHA256, "k8s.%s")`

Note that when using OTTL within the collector's configuration file, `$` must be escaped to `$$` to bypass
environment variable substitution logic. To input a literal `$` from the configuration file, use `$$$`.
If using OTTL outside of collector configuration, `$` should not be escaped and a literal `$` can be entered using `$$`.

### replace_match

`replace_match(target, pattern, replacement, Optional[function], Optional[replacementFormat])`

The `replace_match` function allows replacing entire strings if they match a glob pattern.

`target` is a path expression to a telemetry field. `pattern` is a string following [filepath.Match syntax](https://pkg.go.dev/path/filepath#Match). `replacement` is either a path expression to a string telemetry field or a literal string.
`replacementFormat` is an optional string argument that specifies the format of the replacement. It must contain exactly one `%s` format specifier as shown in the example below. No other format specifiers are supported.

If `target` matches `pattern` it will get replaced with `replacement`.

The `function` is an optional argument that can take in any Converter that accepts a (`replacement`) string and returns a string. An example is a hash function that replaces any matching glob pattern with the hash value of `replacement`.

Examples:

- `replace_match(attributes["http.target"], "/user/*/list/*", "/user/{userId}/list/{listId}")`
- `replace_match(attributes["http.target"], "/user/*/list/*", "/user/{userId}/list/{listId}", SHA256, "/user/%s")`

### replace_pattern

`replace_pattern(target, regex, replacement, Optional[function], Optional[replacementFormat])`

The `replace_pattern` function allows replacing all string sections that match a regex pattern with a new value.

`target` is a path expression to a telemetry field. `regex` is a regex string indicating a segment to replace. `replacement` is either a path expression to a string telemetry field or a literal string.

If one or more sections of `target` match `regex` they will get replaced with `replacement`.

The `replacement` string can refer to matched groups using [regexp.Expand syntax](https://pkg.go.dev/regexp#Regexp.Expand). `replacementFormat` is an optional string argument that specifies the format of the replacement. It must contain exactly one `%s` format specifier as shown in the example below. No other format specifiers are supported

The `function` is an optional argument that can take in any Converter that accepts a (`replacement`) string and returns a string. An example is a hash function that replaces a matching regex pattern with the hash value of `replacement`.

Examples:

- `replace_pattern(resource.attributes["process.command_line"], "password\\=[^\\s]*(\\s?)", "password=***")`
- `replace_pattern(name, "^kube_([0-9A-Za-z]+_)", "k8s.$$1.")`
- `replace_pattern(name, "^kube_([0-9A-Za-z]+_)", "$$1.", SHA256, "k8s.%s")`

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

The `truncate_all` function truncates all string values in a `pcommon.Map` so that none are longer than the limit.

`target` is a path expression to a `pcommon.Map` type field. `limit` is a non-negative integer.

The map will be mutated such that the number of characters in all string values is less than or equal to the limit. Non-string values are ignored.

Examples:

- `truncate_all(attributes, 100)`


- `truncate_all(resource.attributes, 50)`

## Converters

Converters are pure functions that take OTTL values as input and output a single value for use within a statement.
Unlike functions, they do not modify any input telemetry and always return a value.

Available Converters:

- [Base64Decode](#base64decode)
- [Concat](#concat)
- [ConvertCase](#convertcase)
- [Day](#day)
- [ExtractPatterns](#extractpatterns)
- [FNV](#fnv)
- [Hour](#hour)
- [Hours](#hours)
- [Double](#double)
- [Duration](#duration)
- [Int](#int)
- [IsBool](#isbool)
- [IsDouble](#isdouble)
- [IsInt](#isint)
- [IsMap](#ismap)
- [IsMatch](#ismatch)
- [IsList](#islist)
- [IsString](#isstring)
- [Len](#len)
- [Log](#log)
- [Microseconds](#microseconds)
- [Milliseconds](#milliseconds)
- [Minute](#minute)
- [Minutes](#minutes)
- [Month](#month)
- [Nanoseconds](#nanoseconds)
- [Now](#now)
- [ParseCSV](#parsecsv)
- [ParseJSON](#parsejson)
- [ParseKeyValue](#parsekeyvalue)
- [ParseXML](#parsexml)
- [Seconds](#seconds)
- [SHA1](#sha1)
- [SHA256](#sha256)
- [SpanID](#spanid)
- [Split](#split)
- [String](#string)
- [Substring](#substring)
- [Time](#time)
- [TraceID](#traceid)
- [TruncateTime](#truncatetime)
- [Unix](#unix)
- [UnixMicro](#unixmicro)
- [UnixMilli](#unixmilli)
- [UnixNano](#unixnano)
- [UnixSeconds](#unixseconds)
- [UUID](#UUID)
- [Year](#year)

### Base64Decode

`Base64Decode(value)`

The `Base64Decode` Converter takes a base64 encoded string and returns the decoded string.

`value` is a valid base64 encoded string.

Examples:

- `Base64Decode("aGVsbG8gd29ybGQ=")`


- `Base64Decode(attributes["encoded field"])`

### Concat

`Concat(values[], delimiter)`

The `Concat` Converter takes a sequence of values and a delimiter and concatenates their string representation. Unsupported values, such as lists or maps that may substantially increase payload size, are not added to the resulting string.

`values` is a list of values. It supports paths, primitive values, and byte slices (such as trace IDs or span IDs).

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

### Day

`Day(value)`

The `Day` Converter returns the day component from the specified time using the Go stdlib [`time.Day` function](https://pkg.go.dev/time#Time.Day).

`value` is a `time.Time`. If `value` is another type, an error is returned.

The returned type is `int64`.

Examples:

- `Day(Now())`

### Double

The `Double` Converter converts an inputted `value` into a double.

The returned type is float64.

The input `value` types:
* float64. returns the `value` without changes.
* string. Tries to parse a double from string. If it fails then nil will be returned.
* bool. If `value` is true, then the function will return 1 otherwise 0.
* int64. The function converts the integer to a double.

If `value` is another type or parsing failed nil is always returned.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

Examples:

- `Double(attributes["http.status_code"])`


- `Double("2.0")`

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

### Hour

`Hour(value)`

The `Hour` Converter returns the hour from the specified time.  The Converter [uses the `time.Hour` function](https://pkg.go.dev/time#Time.Hour).

`value` is a `time.Time`. If `value` is another type an error is returned.

The returned type is `int64`.

Examples:

- `Hour(Now())`

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

- float64. Fraction is discharged (truncation towards zero).
- string. Trying to parse an integer from string if it fails then nil will be returned.
- bool. If `value` is true, then the function will return 1 otherwise 0.
- int64. The function returns the `value` without changes.

If `value` is another type or parsing failed nil is always returned.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

Examples:

- `Int(attributes["http.status_code"])`


- `Int("2.0")`

### IsBool

`IsBool(value)`

The `IsBool` Converter evaluates whether the given `value` is a boolean or not.

Specifically, it will return `true` if the provided `value` is one of the following:

1. A Go's native `bool` type.
2. A `pcommon.ValueTypeBool`.

Otherwise, it will return `false`.

Examples:

- `IsBool(false)`


- `IsBool(pcommon.NewValueBool(false))`


- `IsBool(42)`


- `IsBool(attributes["any key"])`

### IsDouble

`IsDouble(value)`

The `IsDouble` Converter returns true if the given value is a double.

The `value` is either a path expression to a telemetry field to retrieve, or a literal.

If `value` is a `float64` or a `pcommon.ValueTypeDouble` then returns `true`, otherwise returns `false`.

Examples:

- `IsDouble(body)`

- `IsDouble(attributes["maybe a double"])`

### IsInt

`IsInt(value)`

The `IsInt` Converter returns true if the given value is a int.

The `value` is either a path expression to a telemetry field to retrieve, or a literal.

If `value` is a `int64` or a `pcommon.ValueTypeInt` then returns `true`, otherwise returns `false`.

Examples:

- `IsInt(body)`

- `IsInt(attributes["maybe a int"])`

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

Examples:

- `IsMatch(attributes["http.path"], "foo")`


- `IsMatch("string", ".*ring")`

### IsList

`IsList(value)`

The `IsList` Converter returns true if the given value is a list.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

If `value` is a `list`, `pcommon.ValueTypeSlice`. `pcommon.Slice`, or any other list type, then returns `true`, otherwise returns `false`.

Examples:

- `IsList(body)`

- `IsList(attributes["maybe a slice"])`

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

The `Log` Converter returns a `float64` that is the logarithm of the `target`.

`target` is either a path expression to a telemetry field to retrieve or a literal.

The function take the logarithm of the target, returning an error if the target is less than or equal to zero.

If target is not a float64, it will be converted to one:

- int64s are converted to float64s
- strings are converted using `strconv`
- booleans are converted using `1` for `true` and `0` for `false`. This means passing `false` to the function will cause an error.
- int, float, string, and bool OTLP Values are converted following the above rules depending on their type. Other types cause an error.

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

### Minute

`Minute(value)`

The `Minute` Converter returns the minute component from the specified time using the Go stdlib [`time.Minute` function](https://pkg.go.dev/time#Time.Minute).

`value` is a `time.Time`. If `value` is another type, an error is returned.

The returned type is `int64`.

Examples:

- `Minute(Now())`

### Minutes

`Minutes(value)`

The `Minutes` Converter returns the duration as a floating point number of minutes.

`value` is a `time.Duration`. If `value` is another type an error is returned.

The returned type is `float64`.

Examples:

- `Minutes(Duration("1h"))`

### Month

`Month(value)`

The `Month` Converter returns the month component from the specified time using the Go stdlib [`time.Month` function](https://pkg.go.dev/time#Time.Month).

`value` is a `time.Time`. If `value` is another type, an error is returned.

The returned type is `int64`.

Examples:

- `Month(Now())`

### Nanoseconds

`Nanoseconds(value)`

The `Nanoseconds` Converter returns the duration as an integer nanosecond count.

`value` is a `time.Duration`. If `value` is another type an error is returned.

The returned type is `int64`.

Examples:

- `Nanoseconds(Duration("1h"))`

### Now

`Now()`

The `Now` function returns the current time as represented by `time.Now()` in Go.

The returned type is `time.Time`.

Examples:

- `UnixSeconds(Now())`
- `set(start_time, Now())`

### ParseCSV

`ParseCSV(target, headers, Optional[delimiter], Optional[headerDelimiter], Optional[mode])`

The `ParseCSV` Converter returns a `pcommon.Map` struct that contains the result of parsing the `target` string as CSV. The resultant map is structured such that it is a mapping of field name -> field value.

`target` is a Getter that returns a string. This string should be a CSV row. if `target` is not a properly formatted CSV row, or if the number of fields in `target` does not match the number of fields in `headers`, `ParseCSV` will return an error. Leading and trailing newlines in `target` will be stripped. Newlines elswhere in `target` are not treated as row delimiters during parsing, and will be treated as though they are part of the field that are placed in.

`headers` is a Getter that returns a string. This string should be a CSV header, specifying the names of the CSV fields.

`delimiter` is an optional string parameter that specifies the delimiter used to split `target` into fields. By default, it is set to `,`.

`headerDelimiter` is an optional string parameter that specified the delimiter used to split `headers` into fields. By default, it is set to the value of `delimiter`.

`mode` is an optional string paramater that specifies the parsing mode. Valid values are `strict`, `lazyQuotes`, and `ignoreQuotes`. By default, it is set to `strict`.
- The `strict` mode provides typical CSV parsing.
- The `lazyQotes` mode provides a relaxed version of CSV parsing where a quote may appear in the middle of a unquoted field.
- The `ignoreQuotes` mode completely ignores any quoting rules for CSV and just splits the row on the delimiter.

Examples:

- `ParseCSV("999-999-9999,Joe Smith,joe.smith@example.com", "phone,name,email")`


- `ParseCSV(body, "phone|name|email", delimiter="|")`


- `ParseCSV(attributes["csv_line"], attributes["csv_headers"], delimiter="|", headerDelimiter=",", mode="lazyQuotes")`


- `ParseCSV("\"555-555-5556,Joe Smith\",joe.smith@example.com", "phone,name,email", mode="ignoreQuotes")`

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

### ParseKeyValue

`ParseKeyValue(target, Optional[delimiter], Optional[pair_delimiter])`

The `ParseKeyValue` Converter returns a `pcommon.Map` that is a result of parsing the target string for key value pairs.

`target` is a Getter that returns a string. If the returned string is empty, an error will be returned. `delimiter` is an optional string that is used to split the key and value in a pair, the default is `=`. `pair_delimiter` is an optional string that is used to split key value pairs, the default is a single space (` `).

For example, the following target `"k1=v1 k2=v2 k3=v3"` will use default delimiters and be parsed into the following map:
```
{ "k1": "v1", "k2": "v2", "k3": "v3" }
```

Examples:

- `ParseKeyValue("k1=v1 k2=v2 k3=v3")`
- `ParseKeyValue("k1!v1_k2!v2_k3!v3", "!", "_")`
- `ParseKeyValue(attributes["pairs"])`


### ParseXML

`ParseXML(target)`

The `ParseXML` Converter returns a `pcommon.Map` struct that is the result of parsing the target string as an XML document.

`target` is a Getter that returns a string. This string should be in XML format.
If `target` is not a string, nil, or cannot be parsed as XML, `ParseXML` will return an error.

Unmarshalling XML is done using the following rules:
1. All character data for an XML element is trimmed, joined, and placed into the `content` field.
2. The tag for an XML element is trimmed, and placed into the `tag` field.
3. The attributes for an XML element is placed as a `pcommon.Map` into the `attribute` field.
4. Processing instructions, directives, and comments are ignored and not represented in the resultant map.
5. All child elements are parsed as above, and placed in a `pcommon.Slice`, which is then placed into the `children` field.

For example, the following XML document:
```xml
<?xml version="1.0" encoding="UTF-8" ?>
<Log>
  <User>
    <ID>00001</ID>
    <Name type="first">Joe</Name>
    <Email>joe.smith@example.com</Email>
  </User>
  <Text>User fired alert A</Text>
</Log>
```

will be parsed as:
```json
{
  "tag": "Log",
  "children": [
    {
      "tag": "User",
      "children": [
        {
          "tag": "ID",
          "content": "00001"
        },
        {
          "tag": "Name",
          "content": "Joe",
          "attributes": {
            "type": "first"
          }
        },
        {
          "tag": "Email",
          "content": "joe.smith@example.com"
        }
      ]
    },
    {
      "tag": "Text",
      "content": "User fired alert A"
    }
  ]
}
```

Examples:

- `ParseXML(body)`

- `ParseXML(attributes["xml"])`

- `ParseXML("<HostInfo hostname=\"example.com\" zone=\"east-1\" cloudprovider=\"aws\" />")`



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

```Split(target, delimiter)```

The `Split` Converter separates a string by the delimiter, and returns an array of substrings.

`target` is a string. `delimiter` is a string.

If the `target` is not a string or does not exist, the `Split` Converter will return an error.

Examples:

- `Split("A|B|C", "|")`

### String

`String(value)`

The `String` Converter converts the `value` to string type.

The returned type is `string`.

- string. The function returns the `value` without changes.
- []byte. The function returns the `value` as a string encoded in hexadecimal.
- map. The function returns the `value` as a key-value-pair of type string.
- slice. The function returns the `value` as a list formatted string.
- pcommon.Value. The function returns the `value` as a string type.

If `value` is of another type it gets marshalled to string type.
If `value` is empty, or parsing failed, nil is always returned.

The `value` is either a path expression to a telemetry field to retrieve, or a literal.

Examples:

- `String("test")`
- `String(attributes["http.method"])`
- `String(span_id)`
- `String([1,2,3])`
- `String(false)`

### Substring

`Substring(target, start, length)`

The `Substring` Converter returns a substring from the given start index to the specified length.

`target` is a string. `start` and `length` are `int64`.

If `target` is not a string or is nil, an error is returned.
If the start/length exceed the length of the `target` string, an error is returned.

Examples:

- `Substring("123456789", 0, 3)`

### Time

`Time(target, format, Optional[location])`

The `Time` Converter takes a string representation of a time and converts it to a Golang `time.Time`.

`target` is a string. `format` is a string, `location` is an optional string.

If either `target` or `format` are nil, an error is returned. The parser used is the parser at [internal/coreinternal/parser](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/internal/coreinternal/timeutils). If the `target` and `format` do not follow the parsing rules used by this parser, an error is returned.

`format` denotes a textual representation of the time value formatted according to ctime-like format string. It follows [standard Go Layout formatting](https://pkg.go.dev/time#pkg-constants) with few additional substitutes:
| substitution | description | examples |
|-----|-----|-----|
|`%Y` | Year as a zero-padded number | 0001, 0002, ..., 2019, 2020, ..., 9999 |
|`%y` | Year, last two digits as a zero-padded number | 01, ..., 99 |
|`%m` | Month as a zero-padded number | 01, 02, ..., 12 |
|`%o` | Month as a space-padded number | 1, 2, ..., 12 |
|`%q` | Month as an unpadded number | 1,2,...,12 |
|`%b`, `%h` | Abbreviated month name | Jan, Feb, ... |
|`%B` | Full month name | January, February, ... |
|`%d` | Day of the month as a zero-padded number | 01, 02, ..., 31 |
|`%e` | Day of the month as a space-padded number| 1, 2, ..., 31 |
|`%g` | Day of the month as a unpadded number | 1,2,...,31 |
|`%a` | Abbreviated weekday name | Sun, Mon, ... |
|`%A` | Full weekday name | Sunday, Monday, ... |
|`%H` | Hour (24-hour clock) as a zero-padded number | 00, ..., 24 |
|`%I` | Hour (12-hour clock) as a zero-padded number | 00, ..., 12 |
|`%l` | Hour 12-hour clock | 0, ..., 24 |
|`%p` | Locale’s equivalent of either AM or PM | AM, PM |
|`%P` | Locale’s equivalent of either am or pm | am, pm |
|`%M` | Minute as a zero-padded number | 00, 01, ..., 59 |
|`%S` | Second as a zero-padded number | 00, 01, ..., 59 |
|`%L` | Millisecond as a zero-padded number | 000, 001, ..., 999 |
|`%f` | Microsecond as a zero-padded number | 000000, ..., 999999 |
|`%s` | Nanosecond as a zero-padded number | 00000000, ..., 99999999 |
|`%z` | UTC offset in the form ±HHMM[SS[.ffffff]] or empty | +0000, -0400 |
|`%Z` | Timezone name or abbreviation or empty | UTC, EST, CST |
|`%D`, `%x` | Short MM/DD/YYYY date, equivalent to %m/%d/%y | 01/21/2031 |
|`%F` | Short YYYY-MM-DD date, equivalent to %Y-%m-%d | 2031-01-21 |
|`%T`,`%X` | ISO 8601 time format (HH:MM:SS), equivalent to %H:%M:%S | 02:55:02 |
|`%r` | 12-hour clock time | 02:55:02 pm |
|`%R` | 24-hour HH:MM time, equivalent to %H:%M | 13:55 |
|`%n` | New-line character ('\n') | |
|`%t` | Horizontal-tab character ('\t') | |
|`%%` | A % sign | |
|`%c` | Date and time representation | Mon Jan 02 15:04:05 2006 |

`location` specifies a default time zone canonical ID to be used for date parsing in case it is not part of `format`.

When loading `location`, this function will look for the IANA Time Zone database in the following locations in order:
- a directory or uncompressed zip file named by the ZONEINFO environment variable
- on a Unix system, the system standard installation location
- $GOROOT/lib/time/zoneinfo.zip
- the `time/tzdata` package, if it was imported. 

When building a Collector binary, importing `time/tzdata` in any Go source file will bundle the database into the binary, which guarantees the lookups will work regardless of the setup on the host setup. Note this will add roughly 500kB to binary size.

Examples:

- `Time("02/04/2023", "%m/%d/%Y")`
- `Time("Feb 15, 2023", "%b %d, %Y")`
- `Time("2023-05-26 12:34:56 HST", "%Y-%m-%d %H:%M:%S %Z")`
- `Time("1986-10-01T00:17:33 MST", "%Y-%m-%dT%H:%M:%S %Z")`
- `Time("2012-11-01T22:08:41+0000 EST", "%Y-%m-%dT%H:%M:%S%z %Z")`
- `Time("2023-05-26 12:34:56", "%Y-%m-%d %H:%M:%S", "America/New_York")`

### TraceID

`TraceID(bytes)`

The `TraceID` Converter returns a `pdata.TraceID` struct from the given byte slice.

`bytes` is a byte slice of exactly 16 bytes.

Examples:

- `TraceID(0x00000000000000000000000000000000)`

### TruncateTime

`TruncateTime(time, duration)`

The `TruncateTime` Converter returns the given time rounded down to a multiple of the given duration. The Converter [uses the `time.Truncate` function](https://pkg.go.dev/time#Time.Truncate).

`time` is a `time.Time`. `duration` is a `time.Duration`. If `time` is not a `time.Time` or if `duration` is not a `time.Duration`, an error will be returned.

While some common paths can return a `time.Time` object, you will most like need to use the [Duration Converter](#duration) to create a `time.Duration`.

Examples:

- `TruncateTime(start_time, Duration("1s"))`

### Unix

`Unix(seconds, Optional[nanoseconds])`

The `Unix` Converter returns an epoch timestamp as a Unix time. Similar to [Golang's Unix function](https://pkg.go.dev/time#Unix).

`seconds` is `int64`. If `seconds` is another type an error is returned.
`nanoseconds` is `int64`. It is optional and its default value is 0. If `nanoseconds` is another type an error is returned.

The returned type is `time.Time`.

Examples:

- `Unix(1672527600)`

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

### Year

`Year(value)`

The `Year` Converter returns the year component from the specified time using the Go stdlib [`time.Year` function](https://pkg.go.dev/time#Time.Year).

`value` is a `time.Time`. If `value` is another type, an error is returned.

The returned type is `int64`.

Examples:

- `Year(Now())`

## Function syntax

Functions should be named and formatted according to the following standards.

- Function names MUST start with a verb unless it is a Factory that creates a new type.
- Converters MUST be UpperCamelCase.
- Function names that contain multiple words MUST separate those words with `_`.
- Functions that interact with multiple items MUST have plurality in the name. Ex: `truncate_all`, `keep_keys`, `replace_all_matches`.
- Functions that interact with a single item MUST NOT have plurality in the name. If a function would interact with multiple items due to a condition, like `where`, it is still considered singular. Ex: `set`, `delete`, `replace_match`.
- Functions that change a specific target MUST set the target as the first parameter.
