{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourceStatus', url='', help='"ResourceStatus represents the status of a single resource allocated to a Pod."'),
  '#withName':: d.fn(help='"Name of the resource. Must be unique within the pod and in case of non-DRA resource, match one of the resources from the pod spec. For DRA resources, the value must be \\"claim:<claim_name>/<request>\\". When this status is reported about a container, the \\"claim_name\\" and \\"request\\" must match one of the claims of this container."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withResources':: d.fn(help='"List of unique resources health. Each element in the list contains an unique resource ID and its health. At a minimum, for the lifetime of a Pod, resource ID must uniquely identify the resource allocated to the Pod on the Node. If other Pod on the same Node reports the status with the same resource ID, it must be the same resource they share. See ResourceID type definition for a specific format it has in various use cases."', args=[d.arg(name='resources', type=d.T.array)]),
  withResources(resources): { resources: if std.isArray(v=resources) then resources else [resources] },
  '#withResourcesMixin':: d.fn(help='"List of unique resources health. Each element in the list contains an unique resource ID and its health. At a minimum, for the lifetime of a Pod, resource ID must uniquely identify the resource allocated to the Pod on the Node. If other Pod on the same Node reports the status with the same resource ID, it must be the same resource they share. See ResourceID type definition for a specific format it has in various use cases."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='resources', type=d.T.array)]),
  withResourcesMixin(resources): { resources+: if std.isArray(v=resources) then resources else [resources] },
  '#mixin': 'ignore',
  mixin: self,
}
