{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='linuxContainerUser', url='', help='"LinuxContainerUser represents user identity information in Linux containers"'),
  '#withGid':: d.fn(help='"GID is the primary gid initially attached to the first process in the container"', args=[d.arg(name='gid', type=d.T.integer)]),
  withGid(gid): { gid: gid },
  '#withSupplementalGroups':: d.fn(help='"SupplementalGroups are the supplemental groups initially attached to the first process in the container"', args=[d.arg(name='supplementalGroups', type=d.T.array)]),
  withSupplementalGroups(supplementalGroups): { supplementalGroups: if std.isArray(v=supplementalGroups) then supplementalGroups else [supplementalGroups] },
  '#withSupplementalGroupsMixin':: d.fn(help='"SupplementalGroups are the supplemental groups initially attached to the first process in the container"\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='supplementalGroups', type=d.T.array)]),
  withSupplementalGroupsMixin(supplementalGroups): { supplementalGroups+: if std.isArray(v=supplementalGroups) then supplementalGroups else [supplementalGroups] },
  '#withUid':: d.fn(help='"UID is the primary uid initially attached to the first process in the container"', args=[d.arg(name='uid', type=d.T.integer)]),
  withUid(uid): { uid: uid },
  '#mixin': 'ignore',
  mixin: self,
}
