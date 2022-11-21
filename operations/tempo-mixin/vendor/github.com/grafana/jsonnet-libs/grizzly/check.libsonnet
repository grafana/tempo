local resource = import 'resource.libsonnet';
{
  new(type, name, check)::
    resource.new('SyntheticMonitoringCheck', name)
    + resource.addMetadata('type', type)
    + resource.withSpec(check),
}
