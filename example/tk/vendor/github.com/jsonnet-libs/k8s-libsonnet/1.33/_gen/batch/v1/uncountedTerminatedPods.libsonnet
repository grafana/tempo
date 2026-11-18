{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='uncountedTerminatedPods', url='', help="\"UncountedTerminatedPods holds UIDs of Pods that have terminated but haven't been accounted in Job status counters.\""),
  '#withFailed':: d.fn(help='"failed holds UIDs of failed Pods."', args=[d.arg(name='failed', type=d.T.array)]),
  withFailed(failed): { failed: if std.isArray(v=failed) then failed else [failed] },
  '#withFailedMixin':: d.fn(help='"failed holds UIDs of failed Pods."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='failed', type=d.T.array)]),
  withFailedMixin(failed): { failed+: if std.isArray(v=failed) then failed else [failed] },
  '#withSucceeded':: d.fn(help='"succeeded holds UIDs of succeeded Pods."', args=[d.arg(name='succeeded', type=d.T.array)]),
  withSucceeded(succeeded): { succeeded: if std.isArray(v=succeeded) then succeeded else [succeeded] },
  '#withSucceededMixin':: d.fn(help='"succeeded holds UIDs of succeeded Pods."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='succeeded', type=d.T.array)]),
  withSucceededMixin(succeeded): { succeeded+: if std.isArray(v=succeeded) then succeeded else [succeeded] },
  '#mixin': 'ignore',
  mixin: self,
}
