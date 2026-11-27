---
permalink: /string/
---

# string

```jsonnet
local string = import "github.com/jsonnet-libs/xtd/string.libsonnet"
```

`string` implements helper functions for processing strings.

## Index

* [`fn splitEscape(str, c, escape='\\')`](#fn-splitescape)
* [`fn strReplaceMulti(str, replacements)`](#fn-strreplacemulti)

## Fields

### fn splitEscape

```ts
splitEscape(str, c, escape='\\')
```

`split` works the same as `std.split` but with support for escaping the dividing
string `c`.


### fn strReplaceMulti

```ts
strReplaceMulti(str, replacements)
```

`strReplaceMulti` replaces multiple substrings in a string.

Example:
```jsonnet
strReplaceMulti('hello world', [['hello', 'goodbye'], ['world', 'universe']])
// 'goodbye universe'
```
