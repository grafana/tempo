local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  '#': d.pkg(
    name='array',
    url='github.com/jsonnet-libs/xtd/array.libsonnet',
    help='`array` implements helper functions for processing arrays.',
  ),

  '#slice':: d.fn(
    '`slice` works the same as `std.slice` but with support for negative index/end.',
    [
      d.arg('indexable', d.T.array),
      d.arg('index', d.T.number),
      d.arg('end', d.T.number, default='null'),
      d.arg('step', d.T.number, default=1),
    ]
  ),
  slice(indexable, index, end=null, step=1):
    local invar = {
      index:
        if index != null
        then
          if index < 0
          then std.length(indexable) + index
          else index
        else 0,
      end:
        if end != null
        then
          if end < 0
          then std.length(indexable) + end
          else end
        else std.length(indexable),
    };
    indexable[invar.index:invar.end:step],

  '#filterMapWithIndex':: d.fn(
    |||
      `filterMapWithIndex` works the same as `std.filterMap` with the addition that the index is passed to the functions.

      `filter_func` and `map_func` function signature: `function(index, array_item)`
    |||,
    [
      d.arg('filter_func', d.T.func),
      d.arg('map_func', d.T.func),
      d.arg('arr', d.T.array),
    ],
  ),
  filterMapWithIndex(filter_func, map_func, arr): [
    map_func(i, arr[i])
    for i in std.range(0, std.length(arr) - 1)
    if filter_func(i, arr[i])
  ],

  '#chunkArray':: d.fn(
    |||
      `chunkArray` chunks an array into smaller arrays of the given max size.
    |||,
    [
      d.arg('arr', d.T.array),
      d.arg('maxSize', d.T.number),
    ]
  ),
  chunkArray(arr, maxSize): [
    arr[i * maxSize:std.min((i + 1) * maxSize, std.length(arr))]
    for i in std.range(0, std.ceil(std.length(arr) / maxSize) - 1)
  ],
}
