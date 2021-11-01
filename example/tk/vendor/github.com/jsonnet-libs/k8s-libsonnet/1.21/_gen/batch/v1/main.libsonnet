{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='v1', url='', help=''),
  cronJob: (import 'cronJob.libsonnet'),
  cronJobSpec: (import 'cronJobSpec.libsonnet'),
  cronJobStatus: (import 'cronJobStatus.libsonnet'),
  job: (import 'job.libsonnet'),
  jobCondition: (import 'jobCondition.libsonnet'),
  jobSpec: (import 'jobSpec.libsonnet'),
  jobStatus: (import 'jobStatus.libsonnet'),
  jobTemplateSpec: (import 'jobTemplateSpec.libsonnet'),
}
