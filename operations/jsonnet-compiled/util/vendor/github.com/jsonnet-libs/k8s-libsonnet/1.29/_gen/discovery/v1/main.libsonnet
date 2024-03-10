{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='v1', url='', help=''),
  endpoint: (import 'endpoint.libsonnet'),
  endpointConditions: (import 'endpointConditions.libsonnet'),
  endpointHints: (import 'endpointHints.libsonnet'),
  endpointPort: (import 'endpointPort.libsonnet'),
  endpointSlice: (import 'endpointSlice.libsonnet'),
  forZone: (import 'forZone.libsonnet'),
}
