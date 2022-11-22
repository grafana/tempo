local k = import 'ksonnet-util/kausal.libsonnet';

{
  local deployment = k.apps.v1.deployment,
  local statefulSet = k.apps.v1.statefulSet,

  deployment+:: {
    withMetamonitoring()::
      deployment.spec.template.metadata.withAnnotationsMixin({ 'prometheus.io.metamon': 'true' }),
  },
  statefulSet+:: {
    withMetamonitoring()::
      statefulSet.spec.template.metadata.withAnnotationsMixin({ 'prometheus.io.metamon': 'true' }),
  },
}
