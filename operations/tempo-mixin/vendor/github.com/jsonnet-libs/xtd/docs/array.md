---
permalink: /array/
---

# array

```jsonnet
local array = import "github.com/jsonnet-libs/xtd/array.libsonnet"
```

`array` implements helper functions for processing arrays.

## Index

* [`fn chunkArray(arr, maxSize)`](#fn-chunkarray)
* [`fn filterMapWithIndex(filter_func, map_func, arr)`](#fn-filtermapwithindex)
* [`fn slice(indexable, index, end='null', step=1)`](#fn-slice)

## Fields

### fn chunkArray

```ts
chunkArray(arr, maxSize)
```

`chunkArray` chunks an array into smaller arrays of the given max size.


### fn filterMapWithIndex

```ts
filterMapWithIndex(filter_func, map_func, arr)
```

`filterMapWithIndex` works the same as `std.filterMap` with the addition that the index is passed to the functions.

`filter_func` and `map_func` function signature: `function(index, array_item)`


### fn slice

```ts
slice(indexable, index, end='null', step=1)
```

`slice` works the same as `std.slice` but with support for negative index/end.