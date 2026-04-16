{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='v1', url='', help=''),
  verticalPodAutoscaler: (import 'verticalPodAutoscaler.libsonnet'),
  verticalPodAutoscalerCheckpoint: (import 'verticalPodAutoscalerCheckpoint.libsonnet'),
}
