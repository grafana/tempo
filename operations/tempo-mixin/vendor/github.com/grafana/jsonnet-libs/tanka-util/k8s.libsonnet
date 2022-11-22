local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';
{
  local this = self,

  '#patchKubernetesObjects':: d.fn(
    '`patchKubernetesObjects` applies `patch` to all Kubernetes objects it finds in `object`.',
    [
      d.arg('object', d.T.object),
      d.arg('patch', d.T.object),
    ]
  ),
  patchKubernetesObjects(object, patch, kind=null, name=null)::
    if std.isObject(object)
    then
      // a Kubernetes object is characterized by having an apiVersion and Kind
      if std.objectHas(object, 'apiVersion') && std.objectHas(object, 'kind')
         && (kind == null || object.kind == kind) && (name == null || object.metadata.name == name)
      then object + patch
      else
        std.mapWithKey(
          function(key, obj)
            this.patchKubernetesObjects(obj, patch, kind, name),
          object
        )
    else if std.isArray(object)
    then
      std.map(
        function(obj)
          this.patchKubernetesObjects(obj, patch, kind, name),
        object
      )
    else object,

  '#patchLabels':: d.fn(
    '`patchLabels` finds all Kubernetes objects and adds labels to them.',
    [
      d.arg('object', d.T.object),
      d.arg('labels', d.T.object),
    ]
  ),
  patchLabels(object, labels={})::
    this.patchKubernetesObjects(
      object,
      {
        metadata+: {
          labels+: labels,
        },
      }
    ),
}
