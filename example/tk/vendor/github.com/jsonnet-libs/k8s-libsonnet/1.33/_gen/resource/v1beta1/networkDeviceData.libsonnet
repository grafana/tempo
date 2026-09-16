{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='networkDeviceData', url='', help='"NetworkDeviceData provides network-related details for the allocated device. This information may be filled by drivers or other components to configure or identify the device within a network context."'),
  '#withHardwareAddress':: d.fn(help="\"HardwareAddress represents the hardware address (e.g. MAC Address) of the device's network interface.\\n\\nMust not be longer than 128 characters.\"", args=[d.arg(name='hardwareAddress', type=d.T.string)]),
  withHardwareAddress(hardwareAddress): { hardwareAddress: hardwareAddress },
  '#withInterfaceName':: d.fn(help='"InterfaceName specifies the name of the network interface associated with the allocated device. This might be the name of a physical or virtual network interface being configured in the pod.\\n\\nMust not be longer than 256 characters."', args=[d.arg(name='interfaceName', type=d.T.string)]),
  withInterfaceName(interfaceName): { interfaceName: interfaceName },
  '#withIps':: d.fn(help="\"IPs lists the network addresses assigned to the device's network interface. This can include both IPv4 and IPv6 addresses. The IPs are in the CIDR notation, which includes both the address and the associated subnet mask. e.g.: \\\"192.0.2.5/24\\\" for IPv4 and \\\"2001:db8::5/64\\\" for IPv6.\\n\\nMust not contain more than 16 entries.\"", args=[d.arg(name='ips', type=d.T.array)]),
  withIps(ips): { ips: if std.isArray(v=ips) then ips else [ips] },
  '#withIpsMixin':: d.fn(help="\"IPs lists the network addresses assigned to the device's network interface. This can include both IPv4 and IPv6 addresses. The IPs are in the CIDR notation, which includes both the address and the associated subnet mask. e.g.: \\\"192.0.2.5/24\\\" for IPv4 and \\\"2001:db8::5/64\\\" for IPv6.\\n\\nMust not contain more than 16 entries.\"\n\n**Note:** This function appends passed data to existing values", args=[d.arg(name='ips', type=d.T.array)]),
  withIpsMixin(ips): { ips+: if std.isArray(v=ips) then ips else [ips] },
  '#mixin': 'ignore',
  mixin: self,
}
