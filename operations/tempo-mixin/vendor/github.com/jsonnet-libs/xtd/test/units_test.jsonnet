local units = import '../units.libsonnet';
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';

test.new(std.thisFile)
// parseCPU(10) = parseCPU("10") = 10
// parseCPU(4.5) = parse("4.5") = 4.5
// parseCPU("3000m") = 3000 / 1000
// parseCPU("3580m") = 3580 / 1000
// parseCPU("3980.7m") = 3980.7 / 1000
// parseCPU(0.5) = parse("0.5") = parse("500m") = 0.5
+ test.case.new(
  name='parseCPU - integer',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU(10),
    expected=10,
  )
)
+ test.case.new(
  name='parseCPU - string integer',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU('10'),
    expected=10,
  )
)
+ test.case.new(
  name='parseCPU - float',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU(4.5),
    expected=4.5,
  )
)
+ test.case.new(
  name='parseCPU - string float',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU('4.5'),
    expected=4.5,
  )
)
+ test.case.new(
  name='parseCPU - string millicores',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU('3000m'),
    expected=3,
  )
)
+ test.case.new(
  name='parseCPU - string millicores',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU('3580m'),
    expected=3.58,
  )
)
+ test.case.new(
  name='parseCPU - string millicores',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU('3980.7m'),
    expected=3980.7 / 1000,  // Use a division to avoid floating point precision issues
  )
)
+ test.case.new(
  name='parseCPU - fraction',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU(0.5),
    expected=0.5,
  )
)
+ test.case.new(
  name='parseCPU - string fraction',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU('0.5'),
    expected=0.5,
  )
)
+ test.case.new(
  name='parseCPU - fraction millicores',
  test=test.expect.eq(
    actual=units.parseKubernetesCPU('500m'),
    expected=0.5,
  )
)

// siToBytes tests
+ test.case.new(
  name='siToBytes - plain number',
  test=test.expect.eq(
    actual=units.siToBytes('1024'),
    expected=1024,
  )
)
+ test.case.new(
  name='siToBytes - Ki',
  test=test.expect.eq(
    actual=units.siToBytes('1Ki'),
    expected=1024,
  )
)
+ test.case.new(
  name='siToBytes - Mi',
  test=test.expect.eq(
    actual=units.siToBytes('1Mi'),
    expected=1048576,
  )
)
+ test.case.new(
  name='siToBytes - Gi',
  test=test.expect.eq(
    actual=units.siToBytes('2Gi'),
    expected=2147483648,
  )
)
+ test.case.new(
  name='siToBytes - Ti',
  test=test.expect.eq(
    actual=units.siToBytes('1Ti'),
    expected=1099511627776,
  )
)
+ test.case.new(
  name='siToBytes - fractional Ki',
  test=test.expect.eq(
    actual=units.siToBytes('1.5Ki'),
    expected=1536,
  )
)

// parseDuration tests
+ test.case.new(
  name='parseDuration - milliseconds',
  test=test.expect.eq(
    actual=units.parseDuration('500ms'),
    expected=0.5,
  )
)
+ test.case.new(
  name='parseDuration - seconds',
  test=test.expect.eq(
    actual=units.parseDuration('30s'),
    expected=30,
  )
)
+ test.case.new(
  name='parseDuration - minutes',
  test=test.expect.eq(
    actual=units.parseDuration('5m'),
    expected=300,
  )
)
+ test.case.new(
  name='parseDuration - hours',
  test=test.expect.eq(
    actual=units.parseDuration('2h'),
    expected=7200,
  )
)
+ test.case.new(
  name='parseDuration - combined minutes and seconds',
  test=test.expect.eq(
    actual=units.parseDuration('4m30s'),
    expected=270,
  )
)
+ test.case.new(
  name='parseDuration - combined hours and fractional seconds',
  test=test.expect.eq(
    actual=units.parseDuration('1h30.5s'),
    expected=3600 + 30.5,
  )
)
+ test.case.new(
  name='parseDuration - combined hours and minutes',
  test=test.expect.eq(
    actual=units.parseDuration('1h30m'),
    expected=5400,
  )
)
+ test.case.new(
  name='parseDuration - combined hours and milliseconds',
  test=test.expect.eq(
    actual=units.parseDuration('1h500ms'),
    expected=3600 + 0.5,
  )
)

// formatDuration tests
+ test.case.new(
  name='formatDuration - milliseconds',
  test=test.expect.eq(
    actual=units.formatDuration(0.5),
    expected='500ms',
  )
)
+ test.case.new(
  name='formatDuration - seconds',
  test=test.expect.eq(
    actual=units.formatDuration(45),
    expected='45s',
  )
)
+ test.case.new(
  name='formatDuration - minutes',
  test=test.expect.eq(
    actual=units.formatDuration(300),
    expected='5m',
  )
)
+ test.case.new(
  name='formatDuration - hours',
  test=test.expect.eq(
    actual=units.formatDuration(7200),
    expected='2h',
  )
)
+ test.case.new(
  name='formatDuration - combined minutes and seconds',
  test=test.expect.eq(
    actual=units.formatDuration(270),
    expected='4m30s',
  )
)
+ test.case.new(
  name='formatDuration - combined hours and minutes',
  test=test.expect.eq(
    actual=units.formatDuration(5400),
    expected='1h30m',
  )
)
+ test.case.new(
  name='formatDuration - combined hours and minutes and seconds',
  test=test.expect.eq(
    actual=units.formatDuration(5400 + 30),
    expected='1h30m30s',
  )
)
+ test.case.new(
  name='formatDuration - combined hours and minutes and seconds and milliseconds',
  test=test.expect.eq(
    actual=units.formatDuration(5400 + 30 + 0.5),
    expected='1h30m30s500ms',
  )
)

+ test.case.new(
  name='formatDuration - combined hours and milliseconds',
  test=test.expect.eq(
    actual=units.formatDuration(3600 + 0.5),
    expected='1h500ms',
  )
)
