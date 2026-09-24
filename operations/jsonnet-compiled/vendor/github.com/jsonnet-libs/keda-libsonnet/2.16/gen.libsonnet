{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='keda', url='github.com/jsonnet-libs/keda-libsonnet/2.16/main.libsonnet', help=''),
  eventing:: (import '_gen/eventing/main.libsonnet'),
  keda:: (import '_gen/keda/main.libsonnet'),
}
