{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='statefulSetPersistentVolumeClaimRetentionPolicy', url='', help='"StatefulSetPersistentVolumeClaimRetentionPolicy describes the policy used for PVCs created from the StatefulSet VolumeClaimTemplates."'),
  '#withWhenDeleted':: d.fn(help='"WhenDeleted specifies what happens to PVCs created from StatefulSet VolumeClaimTemplates when the StatefulSet is deleted. The default policy of `Retain` causes PVCs to not be affected by StatefulSet deletion. The `Delete` policy causes those PVCs to be deleted."', args=[d.arg(name='whenDeleted', type=d.T.string)]),
  withWhenDeleted(whenDeleted): { whenDeleted: whenDeleted },
  '#withWhenScaled':: d.fn(help='"WhenScaled specifies what happens to PVCs created from StatefulSet VolumeClaimTemplates when the StatefulSet is scaled down. The default policy of `Retain` causes PVCs to not be affected by a scaledown. The `Delete` policy causes the associated PVCs for any excess pods above the replica count to be deleted."', args=[d.arg(name='whenScaled', type=d.T.string)]),
  withWhenScaled(whenScaled): { whenScaled: whenScaled },
  '#mixin': 'ignore',
  mixin: self,
}
