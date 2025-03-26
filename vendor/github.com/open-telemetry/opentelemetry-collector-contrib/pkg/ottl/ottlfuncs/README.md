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

- [filterprocessor](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/processor/filterprocessor/README.md#configuration)
- [routingconnector](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/connector/routingconnector/README.md#configuration)
- [transformprocessor](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/processor/transformprocessor/README.md#config)

## Editors

Editors are what OTTL uses to transform telemetry.

Editors:

- Are allowed to transform telemetry. When an Editor is invoked the expectation is that the underlying telemetry is modified in some way.
- May have side effects. Some Editors may generate telemetry and add it to the telemetry payload to be processed in this batch.
- May return values. Although not common and not required, Editors may return values.

Available Editors:

- [append](#append)
- [delete_key](#delete_key)
- [delete_matching_keys](#delete_matching_keys)
- [keep_matching_keys](#keep_matching_keys)
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

### append

`append(target, Optional[value], Optional[values])`

The `append` function appends single or multiple string values to `target`. 
`append` converts scalar values into an array if the field exists but is not an array, and creates an array containing the provided values if the field doesn’t exist.

Resulting field is always of type `pcommon.Slice` and will not convert the types of existing or new items in the slice. This means that it is possible to create a slice whose elements have different types.  Be careful when using `append` to set attribute values, as this will produce values that are not possible to create through OpenTelemetry APIs [according to](https://opentelemetry.io/docs/specs/otel/common/#attribute) the OpenTelemetry specification.

- `append(log.attributes["tags"], "prod")`
- `append(log.attributes["tags"], values = ["staging", "staging:east"])`
- `append(log.attributes["tags_copy"], log.attributes["tags"])`

### delete_key

`delete_key(target, key)`

The `delete_key` function removes a key from a `pcommon.Map`

`target` is a path expression to a `pcommon.Map` type field. `key` is a string that is a key in the map.

The key will be deleted from the map.

Examples:


- `delete_key(log.attributes, "http.request.header.authorization")`

- `delete_key(resource.attributes, "http.request.header.authorization")`

### delete_matching_keys

`delete_matching_keys(target, pattern)`

The `delete_matching_keys` function removes all keys from a `pcommon.Map` that match a regex pattern.

`target` is a path expression to a `pcommon.Map` type field. `pattern` is a regex string.

All keys that match the pattern will be deleted from the map.

Examples:


- `delete_matching_keys(log.attributes, "(?i).*password.*")`

- `delete_matching_keys(resource.attributes, "(?i).*password.*")`

### keep_matching_keys

`keep_matching_keys(target, pattern)`

The `keep_matching_keys` function keeps all keys from a `pcommon.Map` that match a regex pattern.

`target` is a path expression to a `pcommon.Map` type field. `pattern` is a regex string.

All keys that match the pattern will remain in the map, while non matching keys will be removed.

Examples:


- `keep_matching_keys(log.attributes, "(?i).*version.*")`

- `keep_matching_keys(resource.attributes, "(?i).*version.*")`

### flatten

`flatten(target, Optional[prefix], Optional[depth], Optional[resolveConflicts])`

The `flatten` function flattens a `pcommon.Map` by moving items from nested maps to the root. 

`target` is a path expression to a `pcommon.Map` type field. `prefix` is an optional string. `depth` is an optional non-negative int, `resolveConflicts` resolves the potential conflicts in the map keys by adding a number suffix starting with `0` from the first duplicated key.


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

If `resolveConflicts` is set to `true`, conflicts within the map will be resolved

```json
{
  "address": {
    "street": {
      "number": "first",
    },
    "house": "1234",
  },
  "address.street": {
    "number": ["second", "third"],
  },
  "address.street.number": "fourth",
  "occupants": [
    "user 1",
    "user 2",
  ],
}
```

the result would be

```json
{
  "address.street.number":   "first",
  "address.house":           "1234",
  "address.street.number.0": "second",
  "address.street.number.1": "third",
  "occupants":               "user 1",
  "occupants.0":             "user 2",
  "address.street.number.2": "fourth",
}

```

**Note:**
Please note that when the `resolveConflicts` parameter is set to `true`, the flattening of arrays is managed differently.
With conflict resolution enabled, arrays and any potentially conflicting keys are handled in a standardized manner. Specifically, a `.<number>` suffix is added to the first conflicting key, with the `number` incrementing for each additional conflict.

Examples:

- `flatten(resource.attributes)`


- `flatten(metric.cache, "k8s", 4)`


- `flatten(log.body, depth=2)`


- `flatten(body, resolveConflicts=true)`


### keep_keys

`keep_keys(target, keys[])`

The `keep_keys` function removes all keys from the `pcommon.Map` that do not match one of the supplied keys.

`target` is a path expression to a `pcommon.Map` type field. `keys` is a slice of one or more strings.

The map will be changed to only contain the keys specified by the list of strings.

Examples:

- `keep_keys(log.attributes, ["http.method"])`


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

- `limit(log.attributes, 100, [])`


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

- `merge_maps(log.attributes, ParseJSON(log.body), "upsert")`


- `merge_maps(log.attributes, ParseJSON(log.attributes["kubernetes"]), "update")`


- `merge_maps(log.attributes, resource.attributes, "insert")`

### replace_all_matches

`replace_all_matches(target, pattern, replacement, Optional[function], Optional[replacementFormat])`

The `replace_all_matches` function replaces any matching string value with the replacement string.

`target` is a path expression to a `pcommon.Map` type field. `pattern` is a string following [filepath.Match syntax](https://pkg.go.dev/path/filepath#Match). `replacement` is either a path expression to a string telemetry field or a literal string. `function` is an optional argument that can take in any Converter that accepts a (`replacement`) string and returns a string. An example is a hash function that replaces any matching string with the hash value of `replacement`.
`replacementFormat` is an optional string argument that specifies the format of the replacement. It must contain exactly one `%s` format specifier as shown in the example below. No other format specifiers are supported.

Each string value in `target` that matches `pattern` will get replaced with `replacement`. Non-string values are ignored.

Examples:

- `replace_all_matches(resource.attributes, "/user/*/list/*", "/user/{userId}/list/{listId}")`
- `replace_all_matches(resource.attributes, "/user/*/list/*", "/user/{userId}/list/{listId}", SHA256, "/user/%s")`

### replace_all_patterns

`replace_all_patterns(target, mode, regex, replacement, Optional[function], Optional[replacementFormat])`

The `replace_all_patterns` function replaces any segments in a string value or key that match the regex pattern with the replacement string.

`target` is a path expression to a `pcommon.Map` type field. `regex` is a regex string indicating a segment to replace. `replacement` is either a path expression to a string telemetry field or a literal string.

`mode` determines whether the match and replace will occur on the map's value or key. Valid values are `key` and `value`.

If one or more sections of `target` match `regex` they will get replaced with `replacement`.

The `replacement` string can refer to matched groups using [regexp.Expand syntax](https://pkg.go.dev/regexp#Regexp.Expand). `replacementFormat` is an optional string argument that specifies the format of the replacement. It must contain exactly one `%s` format specifier as shown in the example below. No other format specifiers are supported.

The `function` is an optional argument that can take in any Converter that accepts a (`replacement`) string and returns a string. An example is a hash function that replaces any matching regex pattern with the hash value of `replacement`.

Examples:

- `replace_all_patterns(resource.attributes, "value", "/account/\\d{4}", "/account/{accountId}")`
- `replace_all_patterns(resource.attributes, "key", "/account/\\d{4}", "/account/{accountId}")`
- `replace_all_patterns(resource.attributes, "key", "^kube_([0-9A-Za-z]+_)", "k8s.$$1.")`
- `replace_all_patterns(resource.attributes, "key", "^kube_([0-9A-Za-z]+_)", "$$1.")`
- `replace_all_patterns(resource.attributes, "key", "^kube_([0-9A-Za-z]+_)", "$$1.", SHA256, "k8s.%s")`

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

- `replace_match(span.attributes["http.target"], "/user/*/list/*", "/user/{userId}/list/{listId}")`
- `replace_match(span.attributes["http.target"], "/user/*/list/*", "/user/{userId}/list/{listId}", SHA256, "/user/%s")`

### replace_pattern

`replace_pattern(target, regex, replacement, Optional[function], Optional[replacementFormat])`

The `replace_pattern` function allows replacing all string sections that match a regex pattern with a new value.

`target` is a path expression to a telemetry field. `regex` is a regex string indicating a segment to replace. `replacement` is either a path expression to a string telemetry field or a literal string.

If one or more sections of `target` match `regex` they will get replaced with `replacement`.

The `replacement` string can refer to matched groups using [regexp.Expand syntax](https://pkg.go.dev/regexp#Regexp.Expand). `replacementFormat` is an optional string argument that specifies the format of the replacement. It must contain exactly one `%s` format specifier as shown in the example below. No other format specifiers are supported

The `function` is an optional argument that can take in any Converter that accepts a (`replacement`) string and returns a string. An example is a hash function that replaces a matching regex pattern with the hash value of `replacement`.

Examples:

- `replace_pattern(resource.attributes["process.command_line"], "password\\=[^\\s]*(\\s?)", "password=***")`
- `replace_pattern(metric.name, "^kube_([0-9A-Za-z]+_)", "k8s.$$1.")`
- `replace_pattern(metric.name, "^kube_([0-9A-Za-z]+_)", "$$1.", SHA256, "k8s.%s")`

Note that when using OTTL within the collector's configuration file, `$` must be escaped to `$$` to bypass
environment variable substitution logic. To input a literal `$` from the configuration file, use `$$$`.
If using OTTL outside of collector configuration, `$` should not be escaped and a literal `$` can be entered using `$$`.

### set

`set(target, value)`

The `set` function allows users to set a telemetry field using a value.

`target` is a path expression to a telemetry field. `value` is any value type. If `value` resolves to `nil`, e.g. it references an unset map value, there will be no action.

How the underlying telemetry field is updated is decided by the path expression implementation provided by the user to the `ottl.ParseStatements`.

Examples:

- `set(resource.attributes["http.path"], "/foo")`


- `set(metric.name, resource.attributes["http.route"])`


- `set(span.trace_state["svc"], "example")`


- `set(span.attributes["source"], span.trace_state["source"])`

### truncate_all

`truncate_all(target, limit)`

The `truncate_all` function truncates all string values in a `pcommon.Map` so that none are longer than the limit.

`target` is a path expression to a `pcommon.Map` type field. `limit` is a non-negative integer.

The map will be mutated such that the number of characters in all string values is less than or equal to the limit. Non-string values are ignored.

Examples:

- `truncate_all(log.attributes, 100)`


- `truncate_all(resource.attributes, 50)`

## Converters

Converters are pure functions that take OTTL values as input and output a single value for use within a statement.
Unlike functions, they do not modify any input telemetry and always return a value.

Available Converters:

- [Base64Decode](#base64decode)
- [Decode](#decode)
- [Concat](#concat)
- [ConvertCase](#convertcase)
- [ConvertAttributesToElementsXML](#convertattributestoelementsxml)
- [ConvertTextToElementsXML](#converttexttoelementsxml)
- [Day](#day)
- [Double](#double)
- [Duration](#duration)
- [ExtractPatterns](#extractpatterns)
- [ExtractGrokPatterns](#extractgrokpatterns)
- [FNV](#fnv)
- [Format](#format)
- [FormatTime](#formattime)
- [GetXML](#getxml)
- [Hex](#hex)
- [Hour](#hour)
- [Hours](#hours)
- [InsertXML](#insertxml)
- [Int](#int)
- [IsBool](#isbool)
- [IsDouble](#isdouble)
- [IsInt](#isint)
- [IsRootSpan](#isrootspan)
- [IsMap](#ismap)
- [IsMatch](#ismatch)
- [IsList](#islist)
- [IsString](#isstring)
- [Len](#len)
- [Log](#log)
- [IsValidLuhn](#isvalidluhn)
- [MD5](#md5)
- [Microseconds](#microseconds)
- [Milliseconds](#milliseconds)
- [Minute](#minute)
- [Minutes](#minutes)
- [Month](#month)
- [Murmur3Hash](#murmur3hash)
- [Murmur3Hash128](#murmur3hash128)
- [Nanosecond](#nanosecond)
- [Nanoseconds](#nanoseconds)
- [Now](#now)
- [ParseCSV](#parsecsv)
- [ParseJSON](#parsejson)
- [ParseKeyValue](#parsekeyvalue)
- [ParseSimplifiedXML](#parsesimplifiedxml)
- [ParseXML](#parsexml)
- [RemoveXML](#removexml)
- [Second](#second)
- [Seconds](#seconds)
- [SHA1](#sha1)
- [SHA256](#sha256)
- [SHA512](#sha512)
- [SliceToMap](#slicetomap)
- [Sort](#sort)
- [SpanID](#spanid)
- [Split](#split)
- [String](#string)
- [Substring](#substring)
- [Time](#time)
- [ToCamelCase](#tocamelcase)
- [ToKeyValueString](#tokeyvaluestring)
- [ToLowerCase](#tolowercase)
- [ToSnakeCase](#tosnakecase)
- [ToUpperCase](#touppercase)
- [TraceID](#traceid)
- [TruncateTime](#truncatetime)
- [Unix](#unix)
- [UnixMicro](#unixmicro)
- [UnixMilli](#unixmilli)
- [UnixNano](#unixnano)
- [UnixSeconds](#unixseconds)
- [UserAgent](#useragent)
- [UUID](#UUID)
- [Weekday](#weekday)
- [Year](#year)

### Base64Decode (Deprecated)

*This function has been deprecated. Please use the [Decode](#decode) function instead.*

`Base64Decode(value)`

The `Base64Decode` Converter takes a base64 encoded string and returns the decoded string.

`value` is a valid base64 encoded string.

Examples:

- `Base64Decode("aGVsbG8gd29ybGQ=")`


- `Base64Decode(resource.attributes["encoded field"])`

### Decode

`Decode(value, encoding)`

The `Decode` Converter takes a string or byte array encoded with the specified encoding and returns the decoded string.

`value` is a valid encoded string or byte array.
`encoding` is a valid encoding name included in the [IANA encoding index](https://www.iana.org/assignments/character-sets/character-sets.xhtml).

Examples:

- `Decode("aGVsbG8gd29ybGQ=", "base64")`


- `Decode(resource.attributes["encoded field"], "us-ascii")`

### Concat

`Concat(values[], delimiter)`

The `Concat` Converter takes a sequence of values and a delimiter and concatenates their string representation. Unsupported values, such as lists or maps that may substantially increase payload size, are not added to the resulting string.

`values` is a list of values. It supports paths, primitive values, and byte slices (such as trace IDs or span IDs).

`delimiter` is a string value that is placed between strings during concatenation. If no delimiter is desired, then simply pass an empty string.

Examples:

- `Concat([span.attributes["http.method"], span.attributes["http.path"]], ": ")`


- `Concat([metric.name, 1], " ")`


- `Concat(["HTTP method is: ", span.attributes["http.method"]], "")`

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

### ConvertAttributesToElementsXML

`ConvertAttributesToElementsXML(target, Optional[xpath])`

The `ConvertAttributesToElementsXML` Converter returns an edited version of an XML string where attributes are converted into child elements.

`target` is a Getter that returns a string. This string should be in XML format.
If `target` is not a string, nil, or cannot be parsed as XML, `ConvertAttributesToElementsXML` will return an error.

`xpath` (optional) is a string that specifies an [XPath](https://www.w3.org/TR/1999/REC-xpath-19991116/) expression that
selects one or more elements. Attributes will only be converted within the result(s) of the xpath.

For example, `<a foo="bar"><b>baz</b></a>` will be converted to `<a><b>baz</b><foo>bar</foo></a>`.

Examples:

Convert all attributes in a document

- `ConvertAttributesToElementsXML(log.body)`

Convert only attributes within "Record" elements

- `ConvertAttributesToElementsXML(log.body, "/Log/Record")`

### ConvertTextToElementsXML

`ConvertTextToElementsXML(target, Optional[xpath], Optional[elementName])`

The `ConvertTextToElementsXML` Converter returns an edited version of an XML string where all text belongs to a dedicated element.

`target` is a Getter that returns a string. This string should be in XML format.
If `target` is not a string, nil, or cannot be parsed as XML, `ConvertTextToElementsXML` will return an error.

`xpath` (optional) is a string that specifies an [XPath](https://www.w3.org/TR/1999/REC-xpath-19991116/) expression that
selects one or more elements. Content will only be converted within the result(s) of the xpath. The default is `/`.

`elementName` (optional) is a string that is used for any element tags that are created to wrap content.
The default is `"value"`.

For example, `<a><b>foo</b>bar</a>` will be converted to `<a><b>foo</b><value>bar</value></a>`.

Examples:

Ensure all text content in a document is wrapped in a dedicated element

- `ConvertTextToElementsXML(log.body)`

Use a custom name for any new elements

- `ConvertTextToElementsXML(log.body, elementName = "custom")`

Convert only part of the document

- `ConvertTextToElementsXML(log.body, "/some/part/", "value")`

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

- `Double(log.attributes["http.status_code"])`


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

- `ExtractPatterns(resource.attributes["k8s.change_cause"], "GIT_SHA=(?P<git.sha>\w+)")`

- `ExtractPatterns(log.body, "^(?P<timestamp>\\w+ \\w+ [0-9]+:[0-9]+:[0-9]+) (?P<hostname>([A-Za-z0-9-_]+)) (?P<process>\\w+)(\\[(?P<pid>\\d+)\\])?: (?P<message>.*)$")`

### ExtractGrokPatterns

`ExtractGrokPatterns(target, pattern, Optional[namedCapturesOnly], Optional[patternDefinitions])`

The `ExtractGrokPatterns` Converter parses unstructured data into a format that is structured and queryable. 
It returns a `pcommon.Map` struct that is a result of extracting named capture groups from the target string. If no matches are found then an empty `pcommon.Map` is returned.

- `target` is a Getter that returns a string. 
- `pattern` is a grok pattern string. 
- `namedCapturesOnly` (optional) specifies if non-named captures should be returned. 
- `patternDefinitions` (optional) is a list of custom pattern definition strings used inside `pattern` in the form of `PATTERN_NAME=PATTERN`. 
This parameter lets you define your own custom patterns to improve readability when the extracted `pattern` is not part of the default set or when you need custom naming. 

If `target` is not a string or nil `ExtractGrokPatterns` returns an error. If `pattern` does not contain at least 1 named capture group and `namedCapturesOnly` is set to `true` then `ExtractPatterns` errors on startup.

Parsing is done using [Elastic Go-Grok](https://github.com/elastic/go-grok?tab=readme-ov-file) library.
Grok is a regular expression dialect that supports reusable aliased expressions. It sits on `re2` regex library so any valid `re2` expressions are valid in grok.
Grok uses this regular expression language to allow naming existing patterns and combining them into more complex patterns that match your fields

Pattern can be specified in either of these forms:
 - `%{SYNTAX}` - e.g {NUMBER}
 - `%{SYNTAX:ID}` - e.g {NUMBER:MY_AGE}
 - `%{SYNTAX:ID:TYPE}` - e.g {NUMBER:MY_AGE:INT}

Where `SYNTAX` is a pattern that will match your text, `ID` is identifier you give to the piece of text being matched and `TYPE` data type you want to cast your named field.
Supported types are `int`, `long`, `double`, `float` and boolean

The [Elastic Go-Grok](https://github.com/elastic/go-grok) ships with numerous predefined grok patterns that simplify working with grok.
In collector Complete set is included consisting of a default set and all additional sets adding product/tool specific capabilities (like [aws](https://github.com/elastic/go-grok/blob/main/patterns/aws.go) or [java](https://github.com/elastic/go-grok/blob/main/patterns/java.go) patterns).


Default set consists of:

| Name | Example |
|-----|-----|
| WORD |  "hello", "world123", "test_data" |
| NOTSPACE | "example", "text-with-dashes", "12345" |
| SPACE | " ", "\t", "  " |
| INT | "123", "-456", "+789" |
| NUMBER | "123", "456.789", "-0.123" |
| BOOL |"true", "false", "true" |
| BASE10NUM | "123", "-123.456", "0.789" |
| BASE16NUM | "1a2b", "0x1A2B", "-0x1a2b3c" |
| BASE16FLOAT |  "0x1.a2b3", "-0x1A2B3C.D" |
| POSINT | "123", "456", "789" |
| NONNEGINT | "0", "123", "456" |
| GREEDYDATA |"anything goes", "literally anything", "123 #@!" |
| QUOTEDSTRING | "\"This is a quote\"", "'single quoted'" |
| UUID |"123e4567-e89b-12d3-a456-426614174000" |
| URN | "urn:isbn:0451450523", "urn:ietf:rfc:2648" |

and many more. Complete list can be found [here](https://github.com/elastic/go-grok/blob/main/patterns/default.go).

Examples:

- _Uses regex pattern with named captures to extract_:

  `ExtractGrokPatterns(resource.attributes["k8s.change_cause"], "GIT_SHA=(?P<git.sha>\w+)")`

- _Uses regex pattern with named captures to extract_:

  `ExtractGrokPatterns(log.body, "^(?P<timestamp>\\w+ \\w+ [0-9]+:[0-9]+:[0-9]+) (?P<hostname>([A-Za-z0-9-_]+)) (?P<process>\\w+)(\\[(?P<pid>\\d+)\\])?: (?P<message>.*)$")`

- _Uses `URI` from default set to extract URI and includes only named captures_:

  `ExtractGrokPatterns(log.body, "%{URI}", true)`

- _Uses more complex pattern consisting of elements from default set and includes only named captures_:
  
  `ExtractGrokPatterns(log.body, "%{DATESTAMP:timestamp} %{TZ:event.timezone} %{DATA:user.name} %{GREEDYDATA:postgresql.log.connection_id} %{POSINT:process.pid:int}", true)`

- _Uses `LOGLINE` pattern defined in `patternDefinitions` passed as last argument_:
  
  `ExtractGrokPatterns(log.body, "%{LOGLINE}", true, ["LOGLINE=%{DATESTAMP:timestamp} %{TZ:event.timezone} %{DATA:user.name} %{GREEDYDATA:postgresql.log.connection_id} %{POSINT:process.pid:int}"])`

- Add custom patterns to parse the password from `/etc/passwd` and making `pattern` readable:

  - `pattern`: `%{USERNAME:user.name}:%{PASSWORD:user.password}:%{USERINFO}`
  - `patternDefinitions`:
    - `PASSWORD=%{WORD}`
    - `USERINFO=%{GREEDYDATA}`

    Note that `USERNAME` is in the default pattern set and does not need to be redefined.

  - Target: `smith:pass123:1001:1000:J Smith,1234,(234)567-8910,(234)567-1098,email:/home/smith:/bin/sh` 

  - Return values: 
     - `user.name`: smith
     - `user.password`: pass123


### FNV

`FNV(value)`

The `FNV` Converter converts the `value` to an FNV hash/digest.

The returned type is int64.

`value` is either a path expression to a string telemetry field or a literal string. If `value` is another type an error is returned.

If an error occurs during hashing it will be returned.

Examples:

- `FNV(resource.attributes["device.name"])`


- `FNV("name")`

### Format

```Format(formatString, []formatArguments)```

The `Format` Converter takes the given format string and formats it using `fmt.Sprintf` and the given arguments.

`formatString` is a string. `formatArguments` is an array of values.

If the `formatString` is not a string or does not exist, the `Format` Converter will return an error.
If any of the `formatArgs` are incorrect (e.g. missing, or an incorrect type for the corresponding format specifier), then a string will still be returned, but with Go's default error handling for `fmt.Sprintf`.

Format specifiers that can be used in `formatString` are documented in Go's [fmt package documentation](https://pkg.go.dev/fmt#hdr-Printing)

Examples:

- `Format("%02d", [log.attributes["priority"]])`
- `Format("%04d-%02d-%02d", [Year(Now()), Month(Now()), Day(Now())])`
- `Format("%s/%s/%04d-%02d-%02d.log", [resource.attributes["hostname"], log.body["program"], Year(Now()), Month(Now()), Day(Now())])`

### FormatTime

`FormatTime(time, format)`

The `FormatTime` Converter takes a `time.Time` and converts it to a human-readable string representation of the time according to the specified format.

`time` is `time.Time`. If `time` is another type an error is returned. `format` is a string.

If either `time` or `format` are nil, an error is returned. The parser used is the parser at [internal/coreinternal/parser](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/internal/coreinternal/timeutils). If `format` does not follow the parsing rules used by this parser, an error is returned.

`format` denotes a human-readable textual representation of the resulting time value formatted according to ctime-like format string. It follows [standard Go Layout formatting](https://pkg.go.dev/time#pkg-constants) with few additional substitutes:
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
|`%i` | Timezone as +/-HH | -07 |
|`%j` | Timezone as +/-HH:MM | -07:00 |
|`%k` | Timezone as +/-HH:MM:SS | -07:00:00 |
|`%w` | Timezone as +/-HHMMSS | -070000 |
|`%D`, `%x` | Short MM/DD/YYYY date, equivalent to %m/%d/%y | 01/21/2031 |
|`%F` | Short YYYY-MM-DD date, equivalent to %Y-%m-%d | 2031-01-21 |
|`%T`,`%X` | ISO 8601 time format (HH:MM:SS), equivalent to %H:%M:%S | 02:55:02 |
|`%r` | 12-hour clock time | 02:55:02 pm |
|`%R` | 24-hour HH:MM time, equivalent to %H:%M | 13:55 |
|`%n` | New-line character ('\n') | |
|`%t` | Horizontal-tab character ('\t') | |
|`%%` | A % sign | |
|`%c` | Date and time representation | Mon Jan 02 15:04:05 2006 |

Examples:

- `FormatTime(Time("02/04/2023", "%m/%d/%Y"), "%A %h %e %Y")`
- `FormatTime(UnixNano(span.attributes["time_nanoseconds"]), "%b %d %Y %H:%M:%S")`
- `FormatTime(TruncateTime(spanevent.time, Duration("10h 20m"))), "%Y-%m-%d %H:%M:%S")`

### GetXML

`GetXML(target, xpath)`

The `GetXML` Converter returns an XML string with selected elements.

`target` is a Getter that returns a string. This string should be in XML format.
If `target` is not a string, nil, or is not valid xml, `GetXML` will return an error.

`xpath` is a string that specifies an [XPath](https://www.w3.org/TR/1999/REC-xpath-19991116/) expression that
selects one or more elements. Currently, this converter only supports selecting elements.

Examples:

Get all elements at the root of the document with tag "a"

- `GetXML(log.body, "/a")`

Gel all elements anywhere in the document with tag "a"

- `GetXML(log.body, "//a")`

Get the first element at the root of the document with tag "a"

- `GetXML(log.body, "/a[1]")`

Get all elements in the document with tag "a" that have an attribute "b" with value "c"

- `GetXML(log.body, "//a[@b='c']")`

Get `foo` from `<a>foo</a>`

- `GetXML(log.body, "/a/text()")`

Get `hello` from `<a><![CDATA[hello]]></a>`

- `GetXML(log.body, "/a/text()")`

Get `bar` from `<a foo="bar"/>`

- `GetXML(log.body, "/a/@foo")`

### Hex

`Hex(value)`

The `Hex` converter converts the `value` to its hexadecimal representation.

The returned type is string representation of the hexadecimal value.

The input `value` types:

- float64 (`1.1` will result to `0x3ff199999999999a`)
- string (`"1"` will result in `0x31`)
- bool (`true` will result in `0x01`; `false` to `0x00`)
- int64 (`12` will result in `0xC`)
- []byte (without any changes - `0x02` will result to `0x02`)

If `value` is another type or parsing failed nil is always returned.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

Examples:

- `Hex(span.attributes["http.status_code"])`


- `Hex(2.0)`

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

### InsertXML

`InsertXML(target, xpath, value)`

The `InsertXML` Converter returns an edited version of an XML string with child elements added to selected elements.

`target` is a Getter that returns a string. This string should be in XML format and represents the document which will
be modified. If `target` is not a string, nil, or is not valid xml, `InsertXML` will return an error.

`xpath` is a string that specifies an [XPath](https://www.w3.org/TR/1999/REC-xpath-19991116/) expression that
selects one or more elements.

`value` is a Getter that returns a string. This string should be in XML format and represents the document which will
be inserted into `target`. If `value` is not a string, nil, or is not valid xml, `InsertXML` will return an error.

Examples:

Add an element "foo" to the root of the document

- `InsertXML(log.body, "/", "<foo/>")`

Add an element "bar" to any element called "foo"

- `InsertXML(log.body, "//foo", "<bar/>")`

Fetch and insert an xml document into another

- `InsertXML(log.body, "/subdoc", log.attributes["subdoc"])`

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

- `Int(log.attributes["http.status_code"])`


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


- `IsBool(resource.attributes["any key"])`

### IsDouble

`IsDouble(value)`

The `IsDouble` Converter returns true if the given value is a double.

The `value` is either a path expression to a telemetry field to retrieve, or a literal.

If `value` is a `float64` or a `pcommon.ValueTypeDouble` then returns `true`, otherwise returns `false`.

Examples:

- `IsDouble(log.body)`

- `IsDouble(log.attributes["maybe a double"])`

### IsInt

`IsInt(value)`

The `IsInt` Converter returns true if the given value is a int.

The `value` is either a path expression to a telemetry field to retrieve, or a literal.

If `value` is a `int64` or a `pcommon.ValueTypeInt` then returns `true`, otherwise returns `false`.

Examples:

- `IsInt(log.body)`

- `IsInt(log.attributes["maybe a int"])`

### IsRootSpan

`IsRootSpan()`

The `IsRootSpan` Converter returns `true` if the span in the corresponding context is root, which means
its `parent_span_id` is equal to hexadecimal representation of zero.

This function is supported with [OTTL span context](../contexts/ottlspan/README.md). In any other context it is not supported.

The function returns `false` in all other scenarios, including `span.parent_span_id == ""` or `span.parent_span_id == nil`.

Examples:

- `IsRootSpan()`

- `set(span.attributes["isRoot"], "true") where IsRootSpan()`

### IsMap

`IsMap(value)`

The `IsMap` Converter returns true if the given value is a map.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

If `value` is a `map[string]any` or a `pcommon.ValueTypeMap` then returns `true`, otherwise returns `false`.

Examples:

- `IsMap(log.body)`


- `IsMap(log.attributes["maybe a map"])`

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

- `IsMatch(span.attributes["http.path"], "foo")`


- `IsMatch("string", ".*ring")`

### IsList

`IsList(value)`

The `IsList` Converter returns true if the given value is a list.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

If `value` is a `list`, `pcommon.ValueTypeSlice`. `pcommon.Slice`, or any other list type, then returns `true`, otherwise returns `false`.

Examples:

- `IsList(log.body)`

- `IsList(resource.attributes["maybe a slice"])`

### IsString

`IsString(value)`

The `IsString` Converter returns true if the given value is a string.

The `value` is either a path expression to a telemetry field to retrieve or a literal.

If `value` is a `string` or a `pcommon.ValueTypeStr` then returns `true`, otherwise returns `false`.

Examples:

- `IsString(log.body)`

- `IsString(resource.attributes["maybe a string"])`

### Len

`Len(target)`

The `Len` Converter returns the int64 length of the target string or slice.

`target` is either a `string`, `slice`, `map`, `pcommon.Slice`, `pcommon.Map`, or `pcommon.Value` with type `pcommon.ValueTypeStr`, `pcommon.ValueTypeSlice`, or `pcommon.ValueTypeMap`.

If the `target` is not an acceptable type, the `Len` Converter will return an error.

Examples:

- `Len(log.body)`

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

- `Log(span.attributes["duration_ms"])`


- `Int(Log(span.attributes["duration_ms"])`

### IsValidLuhn

`IsValidLuhn(value)`

The `IsValidLuhn` converter returns a `boolean` value that indicates whether the value is a valid identification number,
such as a credit card number according to the [Luhn algorithm](https://en.wikipedia.org/wiki/Luhn_algorithm).

The value must either be a `string` consisting of digits only, or an `integer` number. If it is neither, an error will be returned.

Examples:

- `IsValidLuhn(span.attributes["credit_card_number"])`

- `IsValidLuhn("17893729974")`

### MD5

`MD5(value)`

The `MD5` Converter converts the `value` to a md5 hash/digest.

The returned type is string.

`value` is either a path expression to a string telemetry field or a literal string. If `value` is another type an error is returned.

If an error occurs during hashing it will be returned.

Examples:

- `MD5(resource.attributes["device.name"])`

- `MD5("name")`

**Note:** According to the National Institute of Standards and Technology (NIST), MD5 is no longer a recommended hash function. It should be avoided except when required for compatibility. New uses should prefer a SHA-2 family function (e.g. SHA-256, SHA-512) whenever possible.

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

### Murmur3Hash

`Murmur3Hash(target)`

The `Murmur3Hash` Converter converts the `target` string to a hexadecimal string in little-endian of the 32-bit Murmur3 hash.

`target` is a Getter that returns a string.

The returned type is `string`.

Examples:

- `Murmur3Hash(attributes["order.productId"])`

### Murmur3Hash128

`Murmur3Hash128(target)`

The `Murmur3Hash128` Converter converts the `target` string to a hexadecimal string in little-endian of the 128-bit Murmur3 hash.

`target` is a Getter that returns a string.

The returned type is `string`.

Examples:

- `Murmur3Hash128(attributes["order.productId"])`

### Nanosecond

`Nanosecond(value)`

The `Nanosecond` Converter returns the nanosecond component from the specified time using the Go stdlib [`time.Nanosecond` function](https://pkg.go.dev/time#Time.Nanosecond).

`value` is a `time.Time`. If `value` is another type, an error is returned.

The returned type is `int64`.

Examples:

- `Nanosecond(Now())`

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
- `set(span.start_time, Now())`

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


- `ParseCSV(log.body, "phone|name|email", delimiter="|")`


- `ParseCSV(log.attributes["csv_line"], log.attributes["csv_headers"], delimiter="|", headerDelimiter=",", mode="lazyQuotes")`


- `ParseCSV("\"555-555-5556,Joe Smith\",joe.smith@example.com", "phone,name,email", mode="ignoreQuotes")`

### ParseJSON

`ParseJSON(target)`

The `ParseJSON` Converter returns a `pcommon.Map` or `pcommon.Slice` struct that is a result of parsing the target string as JSON

`target` is a Getter that returns a string. This string should be in json format.
If `target` is not a string, nil, or cannot be parsed as JSON, `ParseJSON` will return an error.

Unmarshalling is done using [goccy/go-json](https://github.com/goccy/go-json).
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


- `ParseJSON("[\"attr1\",\"attr2\"]")`


- `ParseJSON(resource.attributes["kubernetes"])`


- `ParseJSON(log.body)`

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
- `ParseKeyValue(log.attributes["pairs"])`

### ParseSimplifiedXML

`ParseSimplifiedXML(target)`

The `ParseSimplifiedXML` Converter returns a `pcommon.Map` struct that is the result of parsing the target string without preservation of attributes or extraneous text content.

The goal of this Converter is to produce a more user-friendly representation of XML data than the [`ParseXML`](#parsexml) Converter.
This Converter should be preferred over `ParseXML` when minor semantic details (e.g. order of elements) are not critically important, when subsequent processing or querying of the result is expected, or when human-readability is a concern.

This Converter disregards certain aspects of XML, specifically attributes and extraneous text content, in order to produce
a direct representation of XML data. Users are encouraged to simplify their XML documents prior to using `ParseSimplifiedXML`.

See other functions which may be useful for preparing XML documents:

- [`ConvertAttributesToElementsXML`](#convertattributestoelementsxml)
- [`ConvertTextToElementsXML`](#converttexttoelementsxml)
- [`RemoveXML`](#removexml)
- [`InsertXML`](#insertxml)
- [`GetXML`](#getxml)

#### Formal Definitions

A "Simplified XML" document contains no attributes and no extraneous text content.

An element has "extraneous text content" when it contains both text and element content. e.g.

```xml
<foo>
    bar <!-- extraneous text content -->
    <hello>world</hello> <!-- element content -->
</foo>
```

#### Parsing logic

1. Declaration elements, attributes, comments, and extraneous text content are ignored.
2. Elements which contain a value are converted into key/value pairs.
   e.g. `<foo>bar</foo>` becomes `"foo": "bar"`
3. Elements which contain child elements are converted into a key/value pair where the value is a map.
   e.g. `<foo> <bar>baz</bar> </foo>` becomes `"foo": { "bar": "baz" }`
4. Sibling elements that share the same tag will be combined into a slice.
   e.g. `<a> <b>1</b> <c>2</c> <c>3</c> </foo>` becomes `"a": { "b": "1", "c": [ "2", "3" ] }`.
5. Empty elements are dropped, but they can determine whether a value should be a slice or map.
   e.g. `<a> <b>1</b> <b/> </a>` becomes `"a": { "b": [ "1" ] }` instead of `"a": { "b": "1" }`

#### Examples

Parse a Simplified XML document from the body:

```xml
<event>
    <id>1</id>
    <user>jane</user>
    <details>
      <time>2021-10-01T12:00:00Z</time>
      <description>Something happened</description>
      <cause>unknown</cause>
    </details>
</event>
```

```json
{
  "event": {
    "id": 1,
    "user": "jane",
    "details": {
      "time": "2021-10-01T12:00:00Z",
      "description": "Something happened",
      "cause": "unknown"
    }
  }
}
```

Parse a Simplified XML document with unique child elements:

```xml
<x>
  <y>1</y>
  <z>2</z>
</x>
```

```json
{
  "x": {
    "y": "1",
    "z": "2"
  }
}
```

Parse a Simplified XML document with multiple elements of the same tag:

```xml
<a>
  <b>1</b>
  <b>2</b>
</a>
```

```json
{
  "a": {
    "b": ["1", "2"]
  }
}
```

Parse a Simplified XML document with CDATA element:

```xml
<a>
  <b>1</b>
  <b><![CDATA[2]]></b>
</a>
```

```json
{
  "a": {
    "b": ["1", "2"]
  }
}
```

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

- `ParseXML(log.body)`

- `ParseXML(log.attributes["xml"])`

- `ParseXML("<HostInfo hostname=\"example.com\" zone=\"east-1\" cloudprovider=\"aws\" />")`

### RemoveXML

`RemoveXML(target, xpath)`

The `RemoveXML` Converter returns an edited version of an XML string with selected elements removed.

`target` is a Getter that returns a string. This string should be in XML format.
If `target` is not a string, nil, or is not valid xml, `RemoveXML` will return an error.

`xpath` is a string that specifies an [XPath](https://www.w3.org/TR/1999/REC-xpath-19991116/) expression that
selects one or more elements to remove from the XML document.

For example, the XPath `/Log/Record[./Name/@type="archive"]` applied to the following XML document:

```xml
<?xml version="1.0" encoding="UTF-8" ?>
<Log>
  <Record>
    <ID>00001</ID>
    <Name type="archive"></Name>
    <Data>Some data</Data>
  </Record>
  <Record>
    <ID>00002</ID>
    <Name type="user"></Name>
    <Data>Some data</Data>
  </Record>
</Log>
```

will return:

```xml
<?xml version="1.0" encoding="UTF-8" ?>
<Log>
  <Record>
    <ID>00002</ID>
    <Name type="user"></Name>
    <Data>Some data</Data>
  </Record>
</Log>
```

Examples:

Delete the attribute "foo" from the elements with tag "a"

- `RemoveXML(log.body, "/a/@foo")`

Delete all elements with tag "b" that are children of elements with tag "a"

- `RemoveXML(log.body, "/a/b")`

Delete all elements with tag "b" that are children of elements with tag "a" and have the attribute "foo" with value "bar"

- `RemoveXML(log.body, "/a/b[@foo='bar']")`

Delete all comments

- `RemoveXML(log.body, "//comment()")`

Delete text from nodes that contain the word "sensitive"

- `RemoveXML(log.body, "//*[contains(text(), 'sensitive')]")`

### Second

`Second(value)`

The `Second` Converter returns the second component from the specified time using the Go stdlib [`time.Second` function](https://pkg.go.dev/time#Time.Second).

`value` is a `time.Time`. If `value` is another type, an error is returned.

The returned type is `int64`.

Examples:

- `Second(Now())`

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

- `SHA1(resource.attributes["device.name"])`


- `SHA1("name")`

**Note:** [According to the National Institute of Standards and Technology (NIST)](https://csrc.nist.gov/projects/hash-functions), SHA1 is no longer a recommended hash function. It should be avoided except when required for compatibility. New uses should prefer a SHA-2 family function (such as SHA-256 or SHA-512) whenever possible.

### SHA256

`SHA256(value)`

The `SHA256` Converter converts the `value` to a sha256 hash/digest.

The returned type is string.

`value` is either a path expression to a string telemetry field or a literal string. If `value` is another type an error is returned.

If an error occurs during hashing it will be returned.

Examples:

- `SHA256(resource.attributes["device.name"])`

- `SHA256("name")`

### SHA512

`SHA512(input)`

The `SHA512` converter calculates sha512 hash value/digest of the `input`.

The returned type is string.

`input` is either a path expression to a string telemetry field or a literal string. If `input` is another type, converter raises an error.
If an error occurs during hashing, the error will be returned.

Examples:

- `SHA512(resource.attributes["device.name"])`

- `SHA512("name")`

### SliceToMap

`SliceToMap(target, keyPath, Optional[valuePath])`

The `SliceToMap` converter converts a slice of objects to a map. The arguments are as follows:

- `target`: A list of maps containing the entries to be converted.
- `keyPath`: A string array that determines the name of the keys for the map entries by pointing to the value of an attribute within each slice item. Note that
the `keyPath` must resolve to a string value, otherwise the converter will not be able to convert the item
to a map entry.
- `valuePath`: This optional string array determines which attribute should be used as the value for the map entry. If no
`valuePath` is defined, the value of the map entry will be the same as the original slice item.

Examples:

The examples below will convert the following input: 

```yaml
attributes:
  hello: world
  things:
    - name: foo
      value: 2
    - name: bar
      value: 5
```

- `SliceToMap(resource.attributes["things"], ["name"])`:

This converts the input above to the following:

```yaml
attributes:
  hello: world
  things:
    foo:
      name: foo
      value: 2
    bar:
      name: bar
      value: 5
```

- `SliceToMap(resource.attributes["things"], ["name"], ["value"])`:

This converts the input above to the following:

```yaml
attributes:
  hello: world
  things:
    foo: 2
    bar: 5
```

Once the `SliceToMap` function has been applied to a value, the converted entries are addressable via their keys:

- `set(resource.attributes["thingsMap"], SliceToMap(resource.attributes["things"], ["name"]))`
- `set(resource.attributes["element_1"], resource.attributes["thingsMap"]["foo'])`
- `set(resource.attributes["element_2"], resource.attributes["thingsMap"]["bar'])`

### Sort

`Sort(target, Optional[order])`

The `Sort` Converter sorts the `target` array in either ascending or descending order.

`target` is an array or `pcommon.Slice` typed field containing the elements to be sorted. 

`order` is a string specifying the sort order. Must be either `asc` or `desc`. The default value is `asc`.

The Sort Converter preserves the data type of the original elements while sorting. 
The behavior varies based on the types of elements in the target slice:

| Element Types | Sorting Behavior                    | Return Value |
|---------------|-------------------------------------|--------------|
| Integers | Sorts as integers                   | Sorted array of integers |
| Doubles | Sorts as doubles                    | Sorted array of doubles |
| Integers and doubles | Converts all to doubles, then sorts | Sorted array of integers and doubles |
| Strings | Sorts as strings                    | Sorted array of strings |
| Booleans | Converts all to strings, then sorts | Sorted array of booleans |
| Mix of integers, doubles, booleans, and strings | Converts all to strings, then sorts | Sorted array of mixed types |
| Any other types | N/A                                 | Returns an error |

Examples:

- `Sort(resource.attributes["device.tags"])`
- `Sort(resource.attributes["device.tags"], "desc")`

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

### Trim

```Trim(target, Optional[replacement])```

The `Trim` Converter removes the leading and trailing character (default: a space character).

If the `target` is not a string or does not exist, the `Trim` Converter will return an error.

`target` is a string.
`replacement` is an optional string representing the character to replace with (default: a space character).

Examples:

- `Trim(" this is a test ", " ")`
- `Trim("!!this is a test!!", "!!")`

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
- `String(span.attributes["http.method"])`
- `String(span.span_id)`
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

`Time(target, format, Optional[location], Optional[locale])`

The `Time` Converter takes a string representation of a time and converts it to a Golang `time.Time`.

`target` is a string. `format` is a string, `location` is an optional string, `locale` is an optional string.

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
|`%i` | Timezone as +/-HH | -07 |
|`%j` | Timezone as +/-HH:MM | -07:00 |
|`%k` | Timezone as +/-HH:MM:SS | -07:00:00 |
|`%w` | Timezone as +/-HHMMSS | -070000 |
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

`locale` specifies the input language of the `target` value. It is used to interpret timestamp values written in a specific language, 
ensuring that the function can correctly parse the localized month names, day names, and periods of the day based on the provided language.

The value must be a well-formed BCP 47 language tag, and a known [CLDR](https://cldr.unicode.org) v45 locale.
If not supplied, English (`en`) is used.

Examples:

- `Time("mercoledì set 4 2024", "%A %h %e %Y", "", "it")`
- `Time("Febrero 25 lunes, 2002, 02:03:04 p.m.", "%B %d %A, %Y, %r", "America/New_York", "es-ES")`

### ToCamelCase

`ToCamelCase(target)`

The `ToCamelCase` Converter converts the `target` string into camel case (e.g. `my_metric_name` to `MyMetricName`).

`target` is a string.

Examples:

- `ToCamelCase(metric.name)`

### ToKeyValueString

`ToKeyValueString(target, Optional[delimiter], Optional[pair_delimiter], Optional[sort_output])`

The `ToKeyValueString` Converter takes a `pcommon.Map` and converts it to a `string` of key value pairs.

- `target` is a Getter that returns a `pcommon.Map`. 
- `delimiter` is an optional string that is used to join keys and values, the default is `=`. 
- `pair_delimiter` is an optional string that is used to join key value pairs, the default is a single space (` `).
- `sort_output` is an optional bool that is used to deterministically sort the keys of the output string. It should only be used if the output is required to be in the same order each time, as it introduces some performance overhead. 

For example, the following map `{"k1":"v1","k2":"v2","k3":"v3"}` will use default delimiters and be converted into the following string:

```
`k1=v1 k2=v2 k3=v3`
```

**Note:** Any nested arrays or maps will be represented as a JSON string. It is recommended to [flatten](#flatten) `target` before using this function. 

For example, `{"k1":"v1","k2":{"k3":"v3","k4":["v4","v5"]}}` will be converted to:

```
`k1=v1 k2={\"k3\":\"v3\",\"k4\":[\"v4\",\"v5\"]}`
```

**Note:** If any keys or values contain either delimiter, they will be double quoted. If any double quotes are present in the quoted value, they will be escaped.

For example, `{"k1":"v1","k2":"v=2","k3"="\"v=3\""}` will be converted to:

```
`k1=v1 k2="v=2" k3="\"v=3\""`
```

Examples:

- `ToKeyValueString(log.body)`
- `ToKeyValueString(log.body, ":", ",", true)`

### ToLowerCase

`ToLowerCase(target)`

The `ToLowerCase` Converter converts the `target` string into lower case (e.g. `MyMetricName` to `mymetricmame`).

`target` is a string.

Examples:

- `ToLowerCase(metric.name)`

### ToSnakeCase

`ToSnakeCase(target)`

The `ToSnakeCase` Converter converts the `target` string into snake case (e.g. `MyMetricName` to `my_metric_name`).

`target` is a string.

Examples:

- `ToSnakeCase(metric.name)`

### ToUpperCase

`ToUpperCase(target)`

The `ToUpperCase` Converter converts the `target` string into upper case (e.g. `MyMetricName` to `MYMETRICNAME`).

`target` is a string.

Examples:

- `ToUpperCase(metric.name)`

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

- `TruncateTime(span.start_time, Duration("1s"))`

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

### UserAgent

`UserAgent(value)`

The `UserAgent` Converter parses the string argument trying to match it against well-known user-agent strings.

`value` is a string or a path to a string.  If `value` is not a string an error is returned.

The results of the parsing are returned as a map containing `user_agent.name`, `user_agent.version` and `user_agent.original`
as defined in semconv v1.25.0.

Parsing is done using the [uap-go package](https://github.com/ua-parser/uap-go). The specific formats it recognizes can be found [here](https://github.com/ua-parser/uap-core/blob/master/regexes.yaml).

Examples:

- `UserAgent("curl/7.81.0")`
  ```yaml
  "user_agent.name": "curl"
  "user_agent.version": "7.81.0"
  "user_agent.original": "curl/7.81.0"
  ```
- `Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0`
  ```yaml
  "user_agent.name": "Firefox"
  "user_agent.version": "126.0"
  "user_agent.original": "Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0"
  ```

### URL

`URL(url_string)`

Parses a Uniform Resource Locator (URL) string and extracts its components as an object.
This URL object includes properties for the URL’s domain, path, fragment, port, query, scheme, user info, username, and password.

`original`, `domain`, `scheme`, and `path` are always present. Other properties are present only if they have corresponding values.

`url_string` is a `string`.

- `URL("http://www.example.com")`

results in 
```
  "url.original": "http://www.example.com",
  "url.scheme":   "http",
  "url.domain":   "www.example.com",
  "url.path":     "",
```

- `URL("http://myusername:mypassword@www.example.com:80/foo.gif?key1=val1&key2=val2#fragment")`

results in 
```
  "url.path":      "/foo.gif",
  "url.fragment":  "fragment",
  "url.extension": "gif",
  "url.password":  "mypassword",
  "url.original":  "http://myusername:mypassword@www.example.com:80/foo.gif?key1=val1&key2=val2#fragment",
  "url.scheme":    "http",
  "url.port":      80,
  "url.user_info": "myusername:mypassword",
  "url.domain":    "www.example.com",
  "url.query":     "key1=val1&key2=val2",
  "url.username":  "myusername",
```

### UUID

`UUID()`

The `UUID` function generates a v4 uuid string.

### Weekday

`Weekday(value)`

The `Weekday` Converter returns the day of the week component from the specified time using the Go stdlib [`time.Weekday` function](https://pkg.go.dev/time#Time.Weekday).

`value` is a `time.Time`. If `value` is another type, an error is returned.

The returned type is `int64`.

The returned range is 0-6 (Sun-Sat)

Examples:

- `Weekday(Now())`

### Year

`Year(value)`

The `Year` Converter returns the year component from the specified time using the Go stdlib [`time.Year` function](https://pkg.go.dev/time#Time.Year).

`value` is a `time.Time`. If `value` is another type, an error is returned.

The returned type is `int64`.

Examples:

- `Year(Now())`
