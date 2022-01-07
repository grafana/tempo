local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';
{
  local this = self,

  '#_config':: 'ignore',
  _config: {
    calledFrom:: error 'new(std.thisFile) was not called',
  },

  '#new': d.fn(
    |||
      `new` initiates the `kustomize` object. It must be called before any `kustomize.build` call:
       > ```jsonnet
       > // std.thisFile required to correctly resolve local Kustomize objects
       > kustomize.new(std.thisFile)
       > ```
    |||,
    [d.arg('calledFrom', d.T.string)]
  ),
  new(calledFrom):: self {
    _config+: { calledFrom: calledFrom },
  },

  '#build':: d.fn(
    |||
      `build` expands the Kustomize object to its underlying resources and returns them in an `Object`,
      so they can be consumed and modified from within Jsonnet.

      This functionality requires Kustomize support in Jsonnet (e.g. using Grafana Tanka) and also
      the `kustomize` binary installed on your `$PATH`.

      `path` is relative to the file calling this function.
    |||,
    [
      d.arg('path', d.T.string),
      d.arg('conf', d.T.object),
    ]
  ),
  build(path, conf={})::
    local cfg = conf { calledFrom: this._config.calledFrom };
    std.native('kustomizeBuild')(path, cfg),
}
