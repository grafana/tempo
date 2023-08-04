# OTTL Functions

The following functions are intended to be used in implementations of the OpenTelemetry Transformation Language that interact with otel data via the collector's internal data model, [pdata](https://github.com/open-telemetry/opentelemetry-collector/tree/main/pdata). These functions may make assumptions about the types of the data returned by Paths.

## Functions

Functions are the way that components that use OTTL transform telemetry.

Functions:
- Are allowed to transform telemetry.  When a Function is invoked the expectation is that the underlying telemetry is modified in some way.
- May have side effects.  Some Functions may generate telemetry and add it to the telemetry payload to be processed in this batch.
- May return values.  Although not common, Functions may return values, but they do not have to.

List of available Functions:
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

## Converters

Converters are functions that help translate between the OTTL grammar and the underlying pdata structure.
They manipulate the OTTL grammar value into a form that will make working with the telemetry easier or more efficient.

Converters:
- Are pure functions.  They should never change the underlying telemetry and the same inputs should always result in the same output.
- Always return something.

List of available Converters:
- [Concat](#concat)
- [ConvertCase](#convertcase)
- [Int](#int)
- [IsMatch](#ismatch)
- [ParseJSON](#ParseJSON)
- [SpanID](#spanid)
- [Split](#split)
- [TraceID](#traceid)
- [Substring](#substring)

### Concat

`Concat(values[], delimiter)`

The `Concat` factory function takes a delimiter and a sequence of values and concatenates their string representation. Unsupported values, such as lists or maps that may substantially increase payload size, are not added to the resulting string.

`values` is a list of values passed as arguments. It supports paths, primitive values, and byte slices (such as trace IDs or span IDs).

`delimiter` is a string value that is placed between strings during concatenation. If no delimiter is desired, then simply pass an empty string.

Examples:

- `Concat([attributes["http.method"], attributes["http.path"]], ": ")`


- `Concat([name, 1], " ")`


- `Concat(["HTTP method is: ", attributes["http.method"]], "")`

### ConvertCase

`ConvertCase(target, toCase)`

The `ConvertCase` factory function converts the `target` string into the desired case `toCase`.

`target` is a string. `toCase` is a string.

If the `target` is not a string or does not exist, the `ConvertCase` factory function will return `nil`.

`toCase` can be:

- `lower`: Converts the `target` string to lowercase (e.g. `MY_METRIC` to `my_metric`)
- `upper`: Converts the `target` string to uppercase (e.g. `my_metric` to `MY_METRIC`)
- `snake`: Converts the `target` string to snakecase (e.g. `myMetric` to `my_metric`)
- `camel`: Converts the `target` string to camelcase (e.g. `my_metric` to `MyMetric`)

If `toCase` is any value other than the options above, the `ConvertCase` factory function will return an error during collector startup.

Examples:

- `ConvertCase(metric.name, "snake")`

### Int

`Int(value)`

The `Int` factory function converts the `value` to int type.

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

### IsMatch

`IsMatch(target, pattern)`

The `IsMatch` factory function returns true if the `target` matches the regex `pattern`.

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

### ParseJSON

`ParseJSON(target)`

The `ParseJSON` factory function returns a `pcommon.Map` struct that is a result of parsing the target string as JSON

`target` is a Getter that returns a string. This string should be in json format.

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

### SpanID

`SpanID(bytes)`

The `SpanID` factory function returns a `pdata.SpanID` struct from the given byte slice.

`bytes` is a byte slice of exactly 8 bytes.

Examples:

- `SpanID(0x0000000000000000)`

### Split

`Split(target, delimiter)`

The `Split` factory function separates a string by the delimiter, and returns an array of substrings.

`target` is a string. `delimiter` is a string.

If the `target` is not a string or does not exist, the `Split` factory function will return `nil`.

Examples:

- ```Split("A|B|C", "|")```

### TraceID

`TraceID(bytes)`

The `TraceID` factory function returns a `pdata.TraceID` struct from the given byte slice.

`bytes` is a byte slice of exactly 16 bytes.

Examples:

- `TraceID(0x00000000000000000000000000000000)`

### Substring

`Substring(target, start, length)`

The `Substring` Converter returns a substring from the given start index to the specified length.

`target` is a string. `start` and `length` are `int64`.

The `Substring` Converter will return `nil` if the given parameters are invalid, e.x. `target` is not a string, or the start/length exceed the length of the `target` string.

Examples:

- `Substring("123456789", 0, 3)`

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

`target` is a path expression to a `pdata.Map` type field. `pattern` is a string following [filepath.Match syntax](https://pkg.go.dev/path/filepath#Match). `replacement` is a string.

Each string value in `target` that matches `pattern` will get replaced with `replacement`. Non-string values are ignored.

Examples:

- `replace_all_matches(attributes, "/user/*/list/*", "/user/{userId}/list/{listId}")`

### replace_all_patterns

`replace_all_patterns(target, mode, regex, replacement)`

The `replace_all_patterns` function replaces any segments in a string value or key that match the regex pattern with the replacement string.

`target` is a path expression to a `pdata.Map` type field. `regex` is a regex string indicating a segment to replace. `replacement` is a string.

`mode` determines whether the match and replace will occur on the map's value or key. Valid values are `key` and `value`.

If one or more sections of `target` match `regex` they will get replaced with `replacement`.

The `replacement` string can refer to matched groups using [regexp.Expand syntax](https://pkg.go.dev/regexp#Regexp.Expand).

Examples:

- `replace_all_patterns(attributes, "value", "/account/\\d{4}", "/account/{accountId}")`
- `replace_all_patterns(attributes, "key", "/account/\\d{4}", "/account/{accountId}")`
- `replace_all_patterns(attributes, "key", "^kube_([0-9A-Za-z]+_)", "k8s.$$1.")`

Note that when using OTTL within the collector's configuration file, `$` must be escaped to `$$` to bypass
environment variable substitution logic. To input a literal `$` from the configuration file, use `$$$`.
If using OTTL outside of collector configuration, `$` should not be escaped and a literal `$` can be entered using `$$`.

### replace_pattern

`replace_pattern(target, regex, replacement)`

The `replace_pattern` function allows replacing all string sections that match a regex pattern with a new value.

`target` is a path expression to a telemetry field. `regex` is a regex string indicating a segment to replace. `replacement` is a string.

If one or more sections of `target` match `regex` they will get replaced with `replacement`.

The `replacement` string can refer to matched groups using [regexp.Expand syntax](https://pkg.go.dev/regexp#Regexp.Expand).

Examples:

- `replace_pattern(resource.attributes["process.command_line"], "password\\=[^\\s]*(\\s?)", "password=***")`
- `replace_pattern(name, "^kube_([0-9A-Za-z]+_)", "k8s.$$1.")`

Note that when using OTTL within the collector's configuration file, `$` must be escaped to `$$` to bypass
environment variable substitution logic. To input a literal `$` from the configuration file, use `$$$`.
If using OTTL outside of collector configuration, `$` should not be escaped and a literal `$` can be entered using `$$`.

### replace_match

`replace_match(target, pattern, replacement)`

The `replace_match` function allows replacing entire strings if they match a glob pattern.

`target` is a path expression to a telemetry field. `pattern` is a string following [filepath.Match syntax](https://pkg.go.dev/path/filepath#Match). `replacement` is a string.

If `target` matches `pattern` it will get replaced with `replacement`.

Examples:

- `replace_match(attributes["http.target"], "/user/*/list/*", "/user/{userId}/list/{listId}")`

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

## Function syntax

Functions should be named and formatted according to the following standards.
- Function names MUST start with a verb unless it is a Factory that creates a new type.
- Factory functions MUST be UpperCamelCase.
- Function names that contain multiple words MUST separate those words with `_`.
- Functions that interact with multiple items MUST have plurality in the name.  Ex: `truncate_all`, `keep_keys`, `replace_all_matches`.
- Functions that interact with a single item MUST NOT have plurality in the name.  If a function would interact with multiple items due to a condition, like `where`, it is still considered singular.  Ex: `set`, `delete`, `replace_match`.
- Functions that change a specific target MUST set the target as the first parameter.
