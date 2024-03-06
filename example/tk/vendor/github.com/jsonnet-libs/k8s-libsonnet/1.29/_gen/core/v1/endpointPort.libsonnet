{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='endpointPort', url='', help='"EndpointPort is a tuple that describes a single port."'),
  '#withAppProtocol':: d.fn(help="\"The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand. This field follows standard Kubernetes label syntax. Valid values are either:\\n\\n* Un-prefixed protocol names - reserved for IANA standard service names (as per RFC-6335 and https://www.iana.org/assignments/service-names).\\n\\n* Kubernetes-defined prefixed names:\\n  * 'kubernetes.io/h2c' - HTTP/2 prior knowledge over cleartext as described in https://www.rfc-editor.org/rfc/rfc9113.html#name-starting-http-2-with-prior-\\n  * 'kubernetes.io/ws'  - WebSocket over cleartext as described in https://www.rfc-editor.org/rfc/rfc6455\\n  * 'kubernetes.io/wss' - WebSocket over TLS as described in https://www.rfc-editor.org/rfc/rfc6455\\n\\n* Other protocols should use implementation-defined prefixed names such as mycompany.com/my-custom-protocol.\"", args=[d.arg(name='appProtocol', type=d.T.string)]),
  withAppProtocol(appProtocol): { appProtocol: appProtocol },
  '#withName':: d.fn(help="\"The name of this port.  This must match the 'name' field in the corresponding ServicePort. Must be a DNS_LABEL. Optional only if one port is defined.\"", args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withPort':: d.fn(help='"The port number of the endpoint."', args=[d.arg(name='port', type=d.T.integer)]),
  withPort(port): { port: port },
  '#withProtocol':: d.fn(help='"The IP protocol for this port. Must be UDP, TCP, or SCTP. Default is TCP."', args=[d.arg(name='protocol', type=d.T.string)]),
  withProtocol(protocol): { protocol: protocol },
  '#mixin': 'ignore',
  mixin: self,
}
