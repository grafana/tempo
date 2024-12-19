{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='statefulSetUpdateStrategy', url='', help='"StatefulSetUpdateStrategy indicates the strategy that the StatefulSet controller will use to perform updates. It includes any additional parameters necessary to perform the update for the indicated strategy."'),
  '#rollingUpdate':: d.obj(help='"RollingUpdateStatefulSetStrategy is used to communicate parameter for RollingUpdateStatefulSetStrategyType."'),
  rollingUpdate: {
    '#withMaxUnavailable':: d.fn(help='"IntOrString is a type that can hold an int32 or a string.  When used in JSON or YAML marshalling and unmarshalling, it produces or consumes the inner type.  This allows you to have, for example, a JSON field that can accept a name or number."', args=[d.arg(name='maxUnavailable', type=d.T.string)]),
    withMaxUnavailable(maxUnavailable): { rollingUpdate+: { maxUnavailable: maxUnavailable } },
    '#withPartition':: d.fn(help='"Partition indicates the ordinal at which the StatefulSet should be partitioned for updates. During a rolling update, all pods from ordinal Replicas-1 to Partition are updated. All pods from ordinal Partition-1 to 0 remain untouched. This is helpful in being able to do a canary based deployment. The default value is 0."', args=[d.arg(name='partition', type=d.T.integer)]),
    withPartition(partition): { rollingUpdate+: { partition: partition } },
  },
  '#withType':: d.fn(help='"Type indicates the type of the StatefulSetUpdateStrategy. Default is RollingUpdate."', args=[d.arg(name='type', type=d.T.string)]),
  withType(type): { type: type },
  '#mixin': 'ignore',
  mixin: self,
}
