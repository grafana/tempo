{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='volumeMountStatus', url='', help='"VolumeMountStatus shows status of volume mounts."'),
  '#withMountPath':: d.fn(help='"MountPath corresponds to the original VolumeMount."', args=[d.arg(name='mountPath', type=d.T.string)]),
  withMountPath(mountPath): { mountPath: mountPath },
  '#withName':: d.fn(help='"Name corresponds to the name of the original VolumeMount."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withReadOnly':: d.fn(help='"ReadOnly corresponds to the original VolumeMount."', args=[d.arg(name='readOnly', type=d.T.boolean)]),
  withReadOnly(readOnly): { readOnly: readOnly },
  '#withRecursiveReadOnly':: d.fn(help='"RecursiveReadOnly must be set to Disabled, Enabled, or unspecified (for non-readonly mounts). An IfPossible value in the original VolumeMount must be translated to Disabled or Enabled, depending on the mount result."', args=[d.arg(name='recursiveReadOnly', type=d.T.string)]),
  withRecursiveReadOnly(recursiveReadOnly): { recursiveReadOnly: recursiveReadOnly },
  '#mixin': 'ignore',
  mixin: self,
}
