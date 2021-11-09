{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='eventSeries', url='', help='EventSeries contain information on series of events, i.e. thing that was/is happening continuously for some time.'),
  '#withCount':: d.fn(help='count is the number of occurrences in this series up to the last heartbeat time.', args=[d.arg(name='count', type=d.T.integer)]),
  withCount(count): { count: count },
  '#withLastObservedTime':: d.fn(help='MicroTime is version of Time with microsecond level precision.', args=[d.arg(name='lastObservedTime', type=d.T.string)]),
  withLastObservedTime(lastObservedTime): { lastObservedTime: lastObservedTime },
  '#mixin': 'ignore',
  mixin: self,
}
