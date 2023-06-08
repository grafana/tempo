local default = import 'environments/default/main.jsonnet';
local base = import 'outputs/base.json';
local test = import 'testonnet/main.libsonnet';

test.new(std.thisFile)
+ test.case.new(
  'Basic',
  test.expect.eq(
    default,
    base
  )
)
