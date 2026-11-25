---
permalink: /number/
---

# number

```jsonnet
local number = import "github.com/jsonnet-libs/xtd/number.libsonnet"
```

`number` implements helper functions for processing number.

## Index

* [`fn inRange(v, from, to)`](#fn-inrange)
* [`fn maxInArray(arr, default=0)`](#fn-maxinarray)
* [`fn minInArray(arr, default=0)`](#fn-mininarray)

## Fields

### fn inRange

```ts
inRange(v, from, to)
```

`inRange` returns true if `v` is in the given from/to range.`

### fn maxInArray

```ts
maxInArray(arr, default=0)
```

`maxInArray` finds the biggest number in an array

### fn minInArray

```ts
minInArray(arr, default=0)
```

`minInArray` finds the smallest number in an array