{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podSchedulingContextStatus', url='', help='"PodSchedulingContextStatus describes where resources for the Pod can be allocated."'),
  '#withResourceClaims':: d.fn(help='"ResourceClaims describes resource availability for each pod.spec.resourceClaim entry where the corresponding ResourceClaim uses \\"WaitForFirstConsumer\\" allocation mode."', args=[d.arg(name='resourceClaims', type=d.T.array)]),
  withResourceClaims(resourceClaims): { resourceClaims: if std.isArray(v=resourceClaims) then resourceClaims else [resourceClaims] },
  '#withResourceClaimsMixin':: d.fn(help='"ResourceClaims describes resource availability for each pod.spec.resourceClaim entry where the corresponding ResourceClaim uses \\"WaitForFirstConsumer\\" allocation mode."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='resourceClaims', type=d.T.array)]),
  withResourceClaimsMixin(resourceClaims): { resourceClaims+: if std.isArray(v=resourceClaims) then resourceClaims else [resourceClaims] },
  '#mixin': 'ignore',
  mixin: self,
}
