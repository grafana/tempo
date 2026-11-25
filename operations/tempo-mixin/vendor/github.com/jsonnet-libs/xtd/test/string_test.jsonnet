local string = import '../string.libsonnet';
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';

test.new(std.thisFile)

+ test.case.new(
  name='strReplaceMulti',
  test=test.expect.eq(
    actual=string.strReplaceMulti('hello world', [['hello', 'goodbye'], ['world', 'universe']]),
    expected='goodbye universe',
  )
)
+ test.case.new(
  name='strReplaceMulti - chained',
  test=test.expect.eq(
    actual=string.strReplaceMulti('hello world', [['hello', 'goodbye'], ['goodbye', 'hi again']]),
    expected='hi again world',
  )
)
+ test.case.new(
  name='strReplaceMulti - not found',
  test=test.expect.eq(
    actual=string.strReplaceMulti('hello world', [['first', 'second'], ['third', 'fourth']]),
    expected='hello world',
  )
)
+ test.case.new(
  name='strReplaceMulti - empty replacements',
  test=test.expect.eq(
    actual=string.strReplaceMulti('hello world', []),
    expected='hello world',
  )
)
