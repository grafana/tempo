---
permalink: /ascii/
---

# ascii

```jsonnet
local ascii = import "github.com/jsonnet-libs/xtd/ascii.libsonnet"
```

`ascii` implements helper functions for ascii characters

## Index

* [`fn isLower(c)`](#fn-islower)
* [`fn isNumber(c)`](#fn-isnumber)
* [`fn isStringJSONNumeric(str)`](#fn-isstringjsonnumeric)
* [`fn isStringNumeric(str)`](#fn-isstringnumeric)
* [`fn isUpper(c)`](#fn-isupper)
* [`fn stringToRFC1123(str)`](#fn-stringtorfc1123)

## Fields

### fn isLower

```ts
isLower(c)
```

`isLower` reports whether ASCII character `c` is a lower case letter

### fn isNumber

```ts
isNumber(c)
```

`isNumber` reports whether character `c` is a number.

### fn isStringJSONNumeric

```ts
isStringJSONNumeric(str)
```

`isStringJSONNumeric` reports whether string `s` is a number as defined by [JSON](https://www.json.org/json-en.html).

### fn isStringNumeric

```ts
isStringNumeric(str)
```

`isStringNumeric` reports whether string `s` consists only of numeric characters.

### fn isUpper

```ts
isUpper(c)
```

`isUpper` reports whether ASCII character `c` is a upper case letter

### fn stringToRFC1123

```ts
stringToRFC1123(str)
```

`stringToRFC113` converts a strings to match RFC1123, replacing non-alphanumeric characters with dashes. It'll throw an assertion if the string is too long.

* RFC 1123. This means the string must:
* - contain at most 63 characters
* - contain only lowercase alphanumeric characters or '-'
* - start with an alphanumeric character
* - end with an alphanumeric character
