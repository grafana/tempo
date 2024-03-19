{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='v1alpha1', url='', help=''),
  ipAddress: (import 'ipAddress.libsonnet'),
  ipAddressSpec: (import 'ipAddressSpec.libsonnet'),
  parentReference: (import 'parentReference.libsonnet'),
  serviceCIDR: (import 'serviceCIDR.libsonnet'),
  serviceCIDRSpec: (import 'serviceCIDRSpec.libsonnet'),
  serviceCIDRStatus: (import 'serviceCIDRStatus.libsonnet'),
}
