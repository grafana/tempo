{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podResourceClaimStatus', url='', help='"PodResourceClaimStatus is stored in the PodStatus for each PodResourceClaim which references a ResourceClaimTemplate. It stores the generated name for the corresponding ResourceClaim."'),
  '#withName':: d.fn(help='"Name uniquely identifies this resource claim inside the pod. This must match the name of an entry in pod.spec.resourceClaims, which implies that the string must be a DNS_LABEL."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withResourceClaimName':: d.fn(help='"ResourceClaimName is the name of the ResourceClaim that was generated for the Pod in the namespace of the Pod. It this is unset, then generating a ResourceClaim was not necessary. The pod.spec.resourceClaims entry can be ignored in this case."', args=[d.arg(name='resourceClaimName', type=d.T.string)]),
  withResourceClaimName(resourceClaimName): { resourceClaimName: resourceClaimName },
  '#mixin': 'ignore',
  mixin: self,
}
