local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';
local k8s = import 'k8s.libsonnet';
{
  local this = self,

  '#_config':: 'ignore',
  _config: {
    calledFrom:: error 'new(std.thisFile) was not called',
  },

  '#new': d.fn(
    |||
      `new` initiates the `helm` object. It must be called before any `helm.template` call:
       > ```jsonnet
       > // std.thisFile required to correctly resolve local Helm Charts
       > helm.new(std.thisFile)
       > ```
    |||,
    [d.arg('calledFrom', d.T.string)]
  ),
  new(calledFrom):: self {
    _config+: { calledFrom: calledFrom },
  },

  // This common label is usually set to 'Helm', this is not true anymore.
  // You can override this with any value you choose.
  // https://helm.sh/docs/chart_best_practices/labels/#standard-labels
  defaultLabels:: { 'app.kubernetes.io/managed-by': 'Helmraiser' },

  '#template':: d.fn(
    |||
      `template` expands the Helm Chart to its underlying resources and returns them in an `Object`,
      so they can be consumed and modified from within Jsonnet.

      This functionality requires Helmraiser support in Jsonnet (e.g. using Grafana Tanka) and also
      the `helm` binary installed on your `$PATH`.
    |||,
    [
      d.arg('name', d.T.string),
      d.arg('chart', d.T.string),
      d.arg('conf', d.T.object),
    ]
  ),
  template(name, chart, conf={})::
    local cfg = conf { calledFrom: this._config.calledFrom };
    local chartData = std.native('helmTemplate')(name, chart, cfg);

    k8s.patchLabels(chartData, this.defaultLabels),
}
