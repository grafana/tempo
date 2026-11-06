# Experimental OTTL Functions

The converters in this package are experimental. They are not covered by OTTL's stability guarantees, so their
names, arguments, and behavior may change, and they may be removed, in any release. They are not part of the
[standard functions](../../ottlfuncs/README.md) returned by `StandardFuncs` and `StandardConverters`. Components make
them available by merging `ExperimentalConverters` into their function set, for example with
`WithExperimentalConverters`.

## Converters

Available Converters:

- [All](#all)
- [Any](#any)
- [Filter](#filter)
- [Find](#find)
- [MapEach](#mapeach)
- [MapKeys](#mapkeys)
- [Reduce](#reduce)
- [When](#when)

### All

> [!IMPORTANT]
> This function is experimental and may change in future releases. It requires the [`ottl.functions.enableLambda`](../../documentation.md#feature-gates) feature gate to be enabled.

`All(source, predicate)`

The `All` converter returns `true` if `predicate` evaluates to `true` for every element in `source`.

`source` is a path expression or another getter that resolves to a slice or map.

`predicate` is a lambda expression with exactly two parameters and a boolean result. 
The first parameter is the element index when evaluating a slice (`int64`), or the element 
key when evaluating a map (`string`). The second parameter is the element value.
Use `_` as a parameter name to ignore unused parameters.

An empty slice or map returns `true`.

If `source` is not a slice or map, or if `predicate` does not return a boolean, it returns an error.

Examples:

Check that every slice element matches:

- `All(log.attributes["tags"], (_, v) => v == "prod")`

Check that every map key matches:

- `All(log.attributes, (k, _) => HasPrefix(k, "http."))`

Use in a condition:

- `set(log.attributes["all_prod"], true) where All(log.attributes["tags"], (_, v) => v == "prod")`

### Any

> [!IMPORTANT]
> This function is experimental and may change in future releases. It requires the [`ottl.functions.enableLambda`](../../documentation.md#feature-gates) feature gate to be enabled.

`Any(source, predicate)`

The `Any` converter returns `true` if `predicate` evaluates to `true` for at least one element in `source`.

`source` is a path expression or another getter that resolves to a slice or map.

`predicate` is a lambda expression with exactly two parameters and a boolean result. 
The first parameter is the element index when evaluating a slice (`int64`), or the element 
key when evaluating a map (`string`). The second parameter is the element value.
Use `_` as a parameter name to ignore unused parameters.

An empty slice or map returns `false`.

If `source` is not a slice or map, or if `predicate` does not return a boolean, it returns an error.

Examples:

Check whether any slice element matches:

- `Any(log.attributes["tags"], (_, v) => v == "prod")`

Check whether any map key matches:

- `Any(log.attributes, (k, _) => HasPrefix(k, "http."))`

Use in a condition:

- `set(log.attributes["has_prod"], true) where Any(log.attributes["tags"], (_, v) => v == "prod")`

### Filter

> [!IMPORTANT]
> This function is experimental and may change in future releases. It requires the [`ottl.functions.enableLambda`](../../documentation.md#feature-gates) feature gate to be enabled.

`Filter(source, predicate)`

The `Filter` converter returns a new `pcommon.Slice` or `pcommon.Map` containing only the elements for which
`predicate` evaluates to `true`.

`source` is a path expression or another getter that resolves to a slice or map.

`predicate` is a lambda expression with exactly two parameters and a boolean result. The first parameter is
the element index when filtering a slice (`int64`), or the element key when filtering a map (`string`). The
second parameter is the element value. Use `_` as a parameter name to ignore unused parameters.

If `source` is not a slice or map, or if `predicate` does not return a boolean, it returns an error.

Examples:

Filter a slice by value:

- `Filter(log.attributes["tags"], (_, v) => v == "prod")`

Filter a map by key:

- `Filter(log.attributes, (k, _) => HasPrefix(k, "http."))`

Store the filtered result:

- `set(log.attributes["prod_tags"], Filter(log.attributes["tags"], (_, v) => v == "prod"))`

### Find

> [!IMPORTANT]
> This function is experimental and may change in future releases. It requires the [`ottl.functions.enableLambda`](../../documentation.md#feature-gates) feature gate to be enabled.

`Find(source, predicate, Optional[mapper])`

The `Find` converter returns the value of the first element in `source` for which `predicate` evaluates to `true`. 
If no element matches, it returns `nil`.

`source` is a path expression or another getter that resolves to a slice or map.

`predicate` is a lambda expression with exactly two parameters and a boolean result. 
The first parameter is the element index when searching a slice (`int64`), or the element 
key when searching a map (`string`). The second parameter is the element value.
Use `_` as a parameter name to ignore unused parameters.

`mapper` is an optional lambda expression with exactly two parameters. When provided, 
it transforms the matched element before returning it. The first parameter is the found element 
index or key, and the second parameter is the element value. When omitted, the matched value 
is returned as-is.

If `source` is not a slice or map, or if `predicate` does not return a boolean, it returns an error.

Examples:

Find a slice element by value:

- `Find(log.attributes["tags"], (_, v) => v == "prod")`

Find a map element by key:

- `Find(log.attributes, (k, _) => k == "http.method")`

Find a map element key instead of value:

- `Find(log.attributes, (_, v) => v == "prod", (k, _) => k)`

Transform the found element:

- `Find(log.attributes, (_, v) => v == "prod", (_, v) => String(v))`

Store the found value:

- `set(log.attributes["first_prod"], Find(log.attributes["tags"], (_, v) => v == "prod"))`

### MapEach

> [!IMPORTANT]
> This function is experimental and may change in future releases. It requires the [`ottl.functions.enableLambda`](../../documentation.md#feature-gates) feature gate to be enabled.

`MapEach(source, mapper)`

The `MapEach` converter returns a new `pcommon.Slice` or `pcommon.Map` with each element value 
transformed by `mapper`.

`source` is a path expression or another getter that resolves to a slice or map.

`mapper` is a lambda expression with exactly two parameters. The first parameter is the element
index when mapping a slice (`int64`), or the element key when mapping a map (`string`). The
second parameter is the element value. Use `_` as a parameter name to ignore unused parameters.

If `source` is not a slice or map, it returns an error.

Examples:

Mapping slice values:

- `MapEach(log.attributes["counts"], (_, v) => Int(v) * 2)`

Stringify map values:

- `MapEach(log.attributes, (_, v) => String(v))`

Store the mapped result:

- `set(log.attributes["doubled"], MapEach(log.attributes["counts"], (_, v) => Int(v)))`

### MapKeys

> [!IMPORTANT]
> This function is experimental and may change in future releases. It requires the [`ottl.functions.enableLambda`](../../documentation.md#feature-gates) feature gate to be enabled.

`MapKeys(source, keyMapper)`

The `MapKeys` converter returns a new `pcommon.Map` with each key transformed by `keyMapper`. Values are unchanged.

`source` is a path expression or another getter that resolves to a map.

`keyMapper` is a lambda expression with exactly two parameters that returns a `string`.
The first parameter is the element key (`string`). The second parameter is the element value.
Use `_` as a parameter name to ignore unused parameters.

If `keyMapper` produces duplicate keys, only one value is retained and which one is unspecified. 
Keys are processed in the order they appear in the `source` map, though this is not guaranteed.

If `source` is not a map, or if `keyMapper` does not return a `string`, it returns an error.

Examples:

Prefix map keys:

- `MapKeys(log.attributes, (k, _) => Concat(["http.", k], ""))`

Derive keys from key and value:

- `MapKeys(log.attributes, (k, v) => Concat([k, ":", String(v)], ""))`

Store the result:

- `set(log.attributes["prefixed"], MapKeys(log.attributes, (k, _) => Concat(["http.", k], "")))`

### Reduce

> [!IMPORTANT]
> This function is experimental and may change in future releases. It requires the [`ottl.functions.enableLambda`](../../documentation.md#feature-gates) feature gate to be enabled.

`Reduce(source, seed, accumulator)`

The `Reduce` converter folds `source` into a single value, starting from `seed` and applying `accumulator` to each element.

`source` is a path expression or another getter that resolves to a slice or map.

`seed` is the initial accumulator value.

`accumulator` is a lambda expression with exactly three parameters. 
The first parameter is the current accumulator value. 
The second parameter is the element index when reducing a slice (`int64`), or the element 
key when reducing a map (`string`). The third parameter is the element value.
Use `_` as a parameter name to ignore unused parameters.

An empty slice or map returns `seed` unchanged.

For maps, element processing order follows map iteration and is not guaranteed to be stable. 
This matters when `accumulator` is not commutative.

If `source` is not a slice or map, it returns an error.

Examples:

Sum a slice of numbers:

- `Reduce(log.attributes["counts"], 0, (acc, _, v) => acc + Int(v))`

Build a semicolon-separated key=value string:

- `Reduce(log.attributes["labels"], "", (acc, k, v) => Concat([acc, k, "=", String(v), ";"], ""))`

Store the result:

- `set(log.attributes["total"], Reduce(log.attributes["counts"], 0, (acc, _, v) => acc + Int(v)))`

### When

> [!IMPORTANT]
> This function is experimental and may change in future releases. It requires the [`ottl.functions.enableLambda`](../../documentation.md#feature-gates) feature gate to be enabled.

`When(condition, trueValue, falseValue)`

The `When` converter returns `trueValue` when `condition` evaluates to true, otherwise it returns `falseValue`.

`condition` is a lambda expression with no parameters that returns a `boolean`.

`trueValue` and `falseValue` are OTTL expressions or literal values.

If `condition` does not return a `boolean`, it returns an error.

Examples:

Select a value based on a type check:

- `When(() => IsMap(log.attributes), "map", "not map")`

Select a value based on a comparison:

- `When(() => attributes["int_value"] > 0, "positive", "negative")`

Store the result:

- `set(log.attributes["result"], When(() => IsMap(log.attributes), "yes", "no"))`
