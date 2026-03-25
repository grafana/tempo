{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourceHealth', url='', help='"ResourceHealth represents the health of a resource. It has the latest device health information. This is a part of KEP https://kep.k8s.io/4680."'),
  '#withHealth':: d.fn(help="\"Health of the resource. can be one of:\\n - Healthy: operates as normal\\n - Unhealthy: reported unhealthy. We consider this a temporary health issue\\n              since we do not have a mechanism today to distinguish\\n              temporary and permanent issues.\\n - Unknown: The status cannot be determined.\\n            For example, Device Plugin got unregistered and hasn't been re-registered since.\\n\\nIn future we may want to introduce the PermanentlyUnhealthy Status.\"", args=[d.arg(name='health', type=d.T.string)]),
  withHealth(health): { health: health },
  '#withResourceID':: d.fn(help='"ResourceID is the unique identifier of the resource. See the ResourceID type for more information."', args=[d.arg(name='resourceID', type=d.T.string)]),
  withResourceID(resourceID): { resourceID: resourceID },
  '#mixin': 'ignore',
  mixin: self,
}
