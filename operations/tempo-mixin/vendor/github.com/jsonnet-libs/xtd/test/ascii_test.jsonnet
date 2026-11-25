local ascii = import '../ascii.libsonnet';
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';

test.new(std.thisFile)

+ test.case.new(
  name='all numeric',
  test=test.expect.eq(
    actual=ascii.isStringNumeric('123'),
    expected=true,
  )
)

+ test.case.new(
  name='only beginning numeric',
  test=test.expect.eq(
    actual=ascii.isStringNumeric('123abc'),
    expected=false,
  )
)

+ test.case.new(
  name='only end numeric',
  test=test.expect.eq(
    actual=ascii.isStringNumeric('abc123'),
    expected=false,
  )
)

+ test.case.new(
  name='none numeric',
  test=test.expect.eq(
    actual=ascii.isStringNumeric('abc'),
    expected=false,
  )
)

+ test.case.new(
  name='empty',
  test=test.expect.eq(
    actual=ascii.isStringNumeric(''),
    expected=true,
  )
)

+ std.foldl(
  function(acc, str)
    acc
    + test.case.new(
      name='valid: ' + str,
      test=test.expect.eq(
        actual=ascii.isStringJSONNumeric(str),
        expected=true,
      )
    ),
  [
    '15',
    '1.5',
    '-1.5',
    '1e5',
    '1E5',
    '1.5e5',
    '1.5E5',
    '1.5e-5',
    '1.5E+5',
  ],
  {},
)
+ std.foldl(
  function(acc, str)
    acc
    + test.case.new(
      name='invalid: ' + str,
      test=test.expect.eq(
        actual=ascii.isStringJSONNumeric(str),
        expected=false,
      )
    ),
  [
    '15e',
    '1.',
    '+',
    '+1E5',
    '.5',
    'E5',
    'e5',
    '15e5garbage',
    '1garbag5e5garbage',
    'garbage15e5garbage',
  ],
  {},
)
