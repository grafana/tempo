local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';

{
  '#': d.pkg(
    name='ascii',
    url='github.com/jsonnet-libs/xtd/ascii.libsonnet',
    help='`ascii` implements helper functions for ascii characters',
  ),

  local cp(c) = std.codepoint(c),

  '#isLower':: d.fn(
    '`isLower` reports whether ASCII character `c` is a lower case letter',
    [d.arg('c', d.T.string)]
  ),
  isLower(c): cp(c) >= 97 && cp(c) < 123,

  '#isUpper':: d.fn(
    '`isUpper` reports whether ASCII character `c` is a upper case letter',
    [d.arg('c', d.T.string)]
  ),
  isUpper(c): cp(c) >= 65 && cp(c) < 91,

  '#isNumber':: d.fn(
    '`isNumber` reports whether character `c` is a number.',
    [d.arg('c', d.T.string)]
  ),
  isNumber(c): std.isNumber(c) || (cp(c) >= 48 && cp(c) < 58),

  '#isStringNumeric':: d.fn(
    '`isStringNumeric` reports whether string `s` consists only of numeric characters.',
    [d.arg('str', d.T.string)]
  ),
  isStringNumeric(str): std.all(std.map(self.isNumber, std.stringChars(str))),

  '#isStringJSONNumeric':: d.fn(
    '`isStringJSONNumeric` reports whether string `s` is a number as defined by [JSON](https://www.json.org/json-en.html).',
    [d.arg('str', d.T.string)]
  ),
  isStringJSONNumeric(str):
    //                           "1"            "9"
    local onenine(c) = (cp(c) >= 49 && cp(c) <= 57);

    //                         "0"
    local digit(c) = (cp(c) == 48 || onenine(c));

    local digits(str) =
      std.length(str) > 0
      && std.all(
        std.foldl(
          function(acc, c)
            acc + [digit(c)],
          std.stringChars(str),
          [],
        )
      );

    local fraction(str) = str == '' || (str[0] == '.' && digits(str[1:]));

    local sign(c) = (c == '-' || c == '+');

    local exponent(str) =
      str == ''
      || (str[0] == 'E' && digits(str[1:]))
      || (str[0] == 'e' && digits(str[1:]))
      || (std.length(str) > 1 && str[0] == 'E' && sign(str[1]) && digits(str[2:]))
      || (std.length(str) > 1 && str[0] == 'e' && sign(str[1]) && digits(str[2:]));


    local integer(str) =
      (std.length(str) == 1 && digit(str[0]))
      || (std.length(str) > 0 && onenine(str[0]) && digits(str[1:]))
      || (std.length(str) > 1 && str[0] == '-' && digit(str[1]))
      || (std.length(str) > 1 && str[0] == '-' && onenine(str[1]) && digits(str[2:]));

    local expectInteger =
      if std.member(str, '.')
      then std.split(str, '.')[0]
      else if std.member(str, 'e')
      then std.split(str, 'e')[0]
      else if std.member(str, 'E')
      then std.split(str, 'E')[0]
      else str;

    local expectFraction =
      if std.member(str, 'e')
      then std.split(str[std.length(expectInteger):], 'e')[0]
      else if std.member(str, 'E')
      then std.split(str[std.length(expectInteger):], 'E')[0]
      else str[std.length(expectInteger):];

    local expectExponent = str[std.length(expectInteger) + std.length(expectFraction):];

    std.all([
      integer(expectInteger),
      fraction(expectFraction),
      exponent(expectExponent),
    ]),

  '#stringToRFC1123': d.fn(
    |||
      `stringToRFC113` converts a strings to match RFC1123, replacing non-alphanumeric characters with dashes. It'll throw an assertion if the string is too long.

      * RFC 1123. This means the string must:
      * - contain at most 63 characters
      * - contain only lowercase alphanumeric characters or '-'
      * - start with an alphanumeric character
      * - end with an alphanumeric character
    |||,
    [d.arg('str', d.T.string)]
  ),
  stringToRFC1123(str):
    // lowercase alphabetic characters
    local lowercase = std.asciiLower(str);
    // replace non-alphanumeric characters with dashes
    local alphanumeric =
      std.join(
        '',
        std.map(
          function(c)
            if self.isLower(c)
               || self.isNumber(c)
            then c
            else '-',
          std.stringChars(lowercase)
        )
      );
    // remove leading/trailing dashes
    local return = std.stripChars(alphanumeric, '-');
    assert std.length(return) <= 63 : 'String too long';
    return,
}
