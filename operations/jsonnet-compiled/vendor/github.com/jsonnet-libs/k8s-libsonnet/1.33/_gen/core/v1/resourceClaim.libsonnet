{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourceClaim', url='', help='"ResourceClaim references one entry in PodSpec.ResourceClaims."'),
  '#withName':: d.fn(help='"Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withRequest':: d.fn(help='"Request is the name chosen for a request in the referenced claim. If empty, everything from the claim is made available, otherwise only the result of this request."', args=[d.arg(name='request', type=d.T.string)]),
  withRequest(request): { request: request },
  '#mixin': 'ignore',
  mixin: self,
}
