{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podResourceClaim', url='', help='"PodResourceClaim references exactly one ResourceClaim, either directly or by naming a ResourceClaimTemplate which is then turned into a ResourceClaim for the pod.\\n\\nIt adds a name to it that uniquely identifies the ResourceClaim inside the Pod. Containers that need access to the ResourceClaim reference it with this name."'),
  '#withName':: d.fn(help='"Name uniquely identifies this resource claim inside the pod. This must be a DNS_LABEL."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withResourceClaimName':: d.fn(help='"ResourceClaimName is the name of a ResourceClaim object in the same namespace as this pod.\\n\\nExactly one of ResourceClaimName and ResourceClaimTemplateName must be set."', args=[d.arg(name='resourceClaimName', type=d.T.string)]),
  withResourceClaimName(resourceClaimName): { resourceClaimName: resourceClaimName },
  '#withResourceClaimTemplateName':: d.fn(help='"ResourceClaimTemplateName is the name of a ResourceClaimTemplate object in the same namespace as this pod.\\n\\nThe template will be used to create a new ResourceClaim, which will be bound to this pod. When this pod is deleted, the ResourceClaim will also be deleted. The pod name and resource name, along with a generated component, will be used to form a unique name for the ResourceClaim, which will be recorded in pod.status.resourceClaimStatuses.\\n\\nThis field is immutable and no changes will be made to the corresponding ResourceClaim by the control plane after creating the ResourceClaim.\\n\\nExactly one of ResourceClaimName and ResourceClaimTemplateName must be set."', args=[d.arg(name='resourceClaimTemplateName', type=d.T.string)]),
  withResourceClaimTemplateName(resourceClaimTemplateName): { resourceClaimTemplateName: resourceClaimTemplateName },
  '#mixin': 'ignore',
  mixin: self,
}
