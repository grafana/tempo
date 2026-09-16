{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='endpointConditions', url='', help='"EndpointConditions represents the current condition of an endpoint."'),
  '#withReady':: d.fn(help='"ready indicates that this endpoint is ready to receive traffic, according to whatever system is managing the endpoint. A nil value should be interpreted as \\"true\\". In general, an endpoint should be marked ready if it is serving and not terminating, though this can be overridden in some cases, such as when the associated Service has set the publishNotReadyAddresses flag."', args=[d.arg(name='ready', type=d.T.boolean)]),
  withReady(ready): { ready: ready },
  '#withServing':: d.fn(help="\"serving indicates that this endpoint is able to receive traffic, according to whatever system is managing the endpoint. For endpoints backed by pods, the EndpointSlice controller will mark the endpoint as serving if the pod's Ready condition is True. A nil value should be interpreted as \\\"true\\\".\"", args=[d.arg(name='serving', type=d.T.boolean)]),
  withServing(serving): { serving: serving },
  '#withTerminating':: d.fn(help='"terminating indicates that this endpoint is terminating. A nil value should be interpreted as \\"false\\"."', args=[d.arg(name='terminating', type=d.T.boolean)]),
  withTerminating(terminating): { terminating: terminating },
  '#mixin': 'ignore',
  mixin: self,
}
