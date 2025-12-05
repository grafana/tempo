{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='claimSource', url='', help='"ClaimSource describes a reference to a ResourceClaim.\\n\\nExactly one of these fields should be set.  Consumers of this type must treat an empty object as if it has an unknown value."'),
  '#withResourceClaimName':: d.fn(help='"ResourceClaimName is the name of a ResourceClaim object in the same namespace as this pod."', args=[d.arg(name='resourceClaimName', type=d.T.string)]),
  withResourceClaimName(resourceClaimName): { resourceClaimName: resourceClaimName },
  '#withResourceClaimTemplateName':: d.fn(help='"ResourceClaimTemplateName is the name of a ResourceClaimTemplate object in the same namespace as this pod.\\n\\nThe template will be used to create a new ResourceClaim, which will be bound to this pod. When this pod is deleted, the ResourceClaim will also be deleted. The pod name and resource name, along with a generated component, will be used to form a unique name for the ResourceClaim, which will be recorded in pod.status.resourceClaimStatuses.\\n\\nThis field is immutable and no changes will be made to the corresponding ResourceClaim by the control plane after creating the ResourceClaim."', args=[d.arg(name='resourceClaimTemplateName', type=d.T.string)]),
  withResourceClaimTemplateName(resourceClaimTemplateName): { resourceClaimTemplateName: resourceClaimTemplateName },
  '#mixin': 'ignore',
  mixin: self,
}
