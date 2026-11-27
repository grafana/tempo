local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  '#': d.pkg(
    name='number',
    url='github.com/jsonnet-libs/xtd/number.libsonnet',
    help='`number` implements helper functions for processing number.',
  ),

  '#inRange':: d.fn(
    '`inRange` returns true if `v` is in the given from/to range.`',
    [
      d.arg('v', d.T.number),
      d.arg('from', d.T.number),
      d.arg('to', d.T.number),
    ]
  ),
  inRange(v, from, to):
    v > from && v <= to,

  '#maxInArray':: d.fn(
    '`maxInArray` finds the biggest number in an array',
    [
      d.arg('arr', d.T.array),
      d.arg('default', d.T.number, default=0),
    ]
  ),
  maxInArray(arr, default=0):
    std.foldl(
      std.max,
      std.set(arr),
      default,
    ),

  '#minInArray':: d.fn(
    '`minInArray` finds the smallest number in an array',
    [
      d.arg('arr', d.T.array),
      d.arg('default', d.T.number, default=0),
    ]
  ),
  minInArray(arr, default=0):
    std.foldl(
      std.min,
      std.set(arr),
      default,
    ),
}
