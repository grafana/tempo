{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='statefulSetOrdinals', url='', help='"StatefulSetOrdinals describes the policy used for replica ordinal assignment in this StatefulSet."'),
  '#withStart':: d.fn(help="\"start is the number representing the first replica's index. It may be used to number replicas from an alternate index (eg: 1-indexed) over the default 0-indexed names, or to orchestrate progressive movement of replicas from one StatefulSet to another. If set, replica indices will be in the range:\\n  [.spec.ordinals.start, .spec.ordinals.start + .spec.replicas).\\nIf unset, defaults to 0. Replica indices will be in the range:\\n  [0, .spec.replicas).\"", args=[d.arg(name='start', type=d.T.integer)]),
  withStart(start): { start: start },
  '#mixin': 'ignore',
  mixin: self,
}
