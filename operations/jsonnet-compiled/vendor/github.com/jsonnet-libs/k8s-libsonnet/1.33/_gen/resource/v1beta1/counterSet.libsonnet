{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='counterSet', url='', help='"CounterSet defines a named set of counters that are available to be used by devices defined in the ResourceSlice.\\n\\nThe counters are not allocatable by themselves, but can be referenced by devices. When a device is allocated, the portion of counters it uses will no longer be available for use by other devices."'),
  '#withCounters':: d.fn(help='"Counters defines the set of counters for this CounterSet The name of each counter must be unique in that set and must be a DNS label.\\n\\nThe maximum number of counters is 32."', args=[d.arg(name='counters', type=d.T.object)]),
  withCounters(counters): { counters: counters },
  '#withCountersMixin':: d.fn(help='"Counters defines the set of counters for this CounterSet The name of each counter must be unique in that set and must be a DNS label.\\n\\nThe maximum number of counters is 32."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='counters', type=d.T.object)]),
  withCountersMixin(counters): { counters+: counters },
  '#withName':: d.fn(help='"Name defines the name of the counter set. It must be a DNS label."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
