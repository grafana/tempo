{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='deviceCounterConsumption', url='', help='"DeviceCounterConsumption defines a set of counters that a device will consume from a CounterSet."'),
  '#withCounterSet':: d.fn(help='"CounterSet is the name of the set from which the counters defined will be consumed."', args=[d.arg(name='counterSet', type=d.T.string)]),
  withCounterSet(counterSet): { counterSet: counterSet },
  '#withCounters':: d.fn(help='"Counters defines the counters that will be consumed by the device.\\n\\nThe maximum number counters in a device is 32. In addition, the maximum number of all counters in all devices is 1024 (for example, 64 devices with 16 counters each)."', args=[d.arg(name='counters', type=d.T.object)]),
  withCounters(counters): { counters: counters },
  '#withCountersMixin':: d.fn(help='"Counters defines the counters that will be consumed by the device.\\n\\nThe maximum number counters in a device is 32. In addition, the maximum number of all counters in all devices is 1024 (for example, 64 devices with 16 counters each)."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='counters', type=d.T.object)]),
  withCountersMixin(counters): { counters+: counters },
  '#mixin': 'ignore',
  mixin: self,
}
