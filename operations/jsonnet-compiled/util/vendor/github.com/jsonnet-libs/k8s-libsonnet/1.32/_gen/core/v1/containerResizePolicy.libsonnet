{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='containerResizePolicy', url='', help='"ContainerResizePolicy represents resource resize policy for the container."'),
  '#withResourceName':: d.fn(help='"Name of the resource to which this resource resize policy applies. Supported values: cpu, memory."', args=[d.arg(name='resourceName', type=d.T.string)]),
  withResourceName(resourceName): { resourceName: resourceName },
  '#withRestartPolicy':: d.fn(help='"Restart policy to apply when specified resource is resized. If not specified, it defaults to NotRequired."', args=[d.arg(name='restartPolicy', type=d.T.string)]),
  withRestartPolicy(restartPolicy): { restartPolicy: restartPolicy },
  '#mixin': 'ignore',
  mixin: self,
}
