{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='sleepAction', url='', help='"SleepAction describes a \\"sleep\\" action."'),
  '#withSeconds':: d.fn(help='"Seconds is the number of seconds to sleep."', args=[d.arg(name='seconds', type=d.T.integer)]),
  withSeconds(seconds): { seconds: seconds },
  '#mixin': 'ignore',
  mixin: self,
}
