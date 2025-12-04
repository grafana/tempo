{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourceHandle', url='', help='"ResourceHandle holds opaque resource data for processing by a specific kubelet plugin."'),
  '#withData':: d.fn(help='"Data contains the opaque data associated with this ResourceHandle. It is set by the controller component of the resource driver whose name matches the DriverName set in the ResourceClaimStatus this ResourceHandle is embedded in. It is set at allocation time and is intended for processing by the kubelet plugin whose name matches the DriverName set in this ResourceHandle.\\n\\nThe maximum size of this field is 16KiB. This may get increased in the future, but not reduced."', args=[d.arg(name='data', type=d.T.string)]),
  withData(data): { data: data },
  '#withDriverName':: d.fn(help="\"DriverName specifies the name of the resource driver whose kubelet plugin should be invoked to process this ResourceHandle's data once it lands on a node. This may differ from the DriverName set in ResourceClaimStatus this ResourceHandle is embedded in.\"", args=[d.arg(name='driverName', type=d.T.string)]),
  withDriverName(driverName): { driverName: driverName },
  '#mixin': 'ignore',
  mixin: self,
}
