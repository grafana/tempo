{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='endpointConditions', url='', help='EndpointConditions represents the current condition of an endpoint.'),
  '#withReady':: d.fn(help='ready indicates that this endpoint is prepared to receive traffic, according to whatever system is managing the endpoint. A nil value indicates an unknown state. In most cases consumers should interpret this unknown state as ready.', args=[d.arg(name='ready', type=d.T.boolean)]),
  withReady(ready): { ready: ready },
  '#mixin': 'ignore',
  mixin: self,
}
