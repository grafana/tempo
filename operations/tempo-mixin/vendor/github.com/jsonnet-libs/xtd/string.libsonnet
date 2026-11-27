local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  '#': d.pkg(
    name='string',
    url='github.com/jsonnet-libs/xtd/string.libsonnet',
    help='`string` implements helper functions for processing strings.',
  ),

  // BelRune is a string of the Ascii character BEL which made computers ring in ancient times.
  // We use it as "magic" char to temporarily replace an escaped string as it is a non printable
  // character and thereby will unlikely be in a valid key by accident. Only when we include it.
  local BelRune = std.char(7),

  '#splitEscape':: d.fn(
    |||
      `split` works the same as `std.split` but with support for escaping the dividing
      string `c`.
    |||,
    [
      d.arg('str', d.T.string),
      d.arg('c', d.T.string),
      d.arg('escape', d.T.string, default='\\'),
    ]
  ),
  splitEscape(str, c, escape='\\'):
    std.map(
      function(i)
        std.strReplace(i, BelRune, escape + c),
      std.split(
        std.strReplace(str, escape + c, BelRune),
        c,
      )
    ),

  '#strReplaceMulti':: d.fn(
    |||
      `strReplaceMulti` replaces multiple substrings in a string.

      Example:
      ```jsonnet
      strReplaceMulti('hello world', [['hello', 'goodbye'], ['world', 'universe']])
      // 'goodbye universe'
      ```
    |||,
    [
      d.arg('str', d.T.string),
      d.arg('replacements', d.T.array),
    ]
  ),
  strReplaceMulti(str, replacements):
    assert std.isString(str) : 'str must be a string';
    assert std.isArray(replacements) : 'replacements must be an array';
    assert std.all([std.isArray(r) && std.length(r) == 2 && std.isString(r[0]) && std.isString(r[1]) for r in replacements]) : 'replacements must be an array of arrays of strings';
    std.foldl(
      function(acc, replacement)
        std.strReplace(acc, replacement[0], replacement[1]),
      replacements,
      str,
    ),
}
