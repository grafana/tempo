{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='modifyVolumeStatus', url='', help='"ModifyVolumeStatus represents the status object of ControllerModifyVolume operation"'),
  '#withTargetVolumeAttributesClassName':: d.fn(help='"targetVolumeAttributesClassName is the name of the VolumeAttributesClass the PVC currently being reconciled"', args=[d.arg(name='targetVolumeAttributesClassName', type=d.T.string)]),
  withTargetVolumeAttributesClassName(targetVolumeAttributesClassName): { targetVolumeAttributesClassName: targetVolumeAttributesClassName },
  '#mixin': 'ignore',
  mixin: self,
}
