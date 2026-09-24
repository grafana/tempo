{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='counterSet', url='', help='"CounterSet defines a named set of counters that are available to be used by devices defined in the ResourceSlice.\\n\\nThe counters are not allocatable by themselves, but can be referenced by devices. When a device is allocated, the portion of counters it uses will no longer be available for use by other devices."'),
  '#withCounters':: d.fn(help='"Counters defines the counters that will be consumed by the device. The name of each counter must be unique in that set and must be a DNS label.\\n\\nTo ensure this uniqueness, capacities defined by the vendor must be listed without the driver name as domain prefix in their name. All others must be listed with their domain prefix.\\n\\nThe maximum number of counters is 32."', args=[d.arg(name='counters', type=d.T.object)]),
  withCounters(counters): { counters: counters },
  '#withCountersMixin':: d.fn(help='"Counters defines the counters that will be consumed by the device. The name of each counter must be unique in that set and must be a DNS label.\\n\\nTo ensure this uniqueness, capacities defined by the vendor must be listed without the driver name as domain prefix in their name. All others must be listed with their domain prefix.\\n\\nThe maximum number of counters is 32."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='counters', type=d.T.object)]),
  withCountersMixin(counters): { counters+: counters },
  '#withName':: d.fn(help='"CounterSet is the name of the set from which the counters defined will be consumed."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
