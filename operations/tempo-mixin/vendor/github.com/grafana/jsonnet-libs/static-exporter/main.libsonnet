local k = import 'ksonnet-util/kausal.libsonnet';

{
  new(name, image='httpd:2.4-alpine'):: {
    data:: { metrics: '' },

    local configMap = k.core.v1.configMap,
    configmap:
      configMap.new(name, self.data),

    local container = k.core.v1.container,
    container::
      container.new('static-exporter', image)
      + container.withPorts([
        k.core.v1.containerPort.newNamed(name='http-metrics', containerPort=80),
      ])
      + k.util.resourcesRequests('10m', '10Mi')
    ,

    local deployment = k.apps.v1.deployment,
    deployment:
      deployment.new(name, replicas=1, containers=[self.container])
      + k.util.configMapVolumeMount(self.configmap, '/usr/local/apache2/htdocs'),
  },

  withData(data):: { data: data },

  withDataMixin(data):: { data+: data },

  withMetrics(metrics)::
    self.withDataMixin({
      metrics:
        std.lines(
          std.foldl(
            function(acc, metric)
              acc + [
                '# HELP %(name)s %(description)s' % metric,
                '# TYPE %(name)s counter' % metric,
              ] + [
                metric.name + value
                for value in metric.values
              ],
            metrics,
            []
          )
        ),
    }),

  metric:: {
    new(name, description)::
      self.withName(name)
      + self.withDescription(description),

    withName(name): { name: name },

    withDescription(description): { description: description },

    local generateValues(labelMap, value=1) =
      local labels = [
        key + '="' + labelMap[key] + '"'
        for key in std.objectFields(labelMap)
      ];
      [
        '{%s} %d' % [std.join(',', labels), value],
      ],

    // withValue adds a labeled metric with a value
    // labelMap = { key: value }
    withValue(labelMap, value=1): {
      values+: generateValues(labelMap, value),
    },

    // withLabelMapList adds multiple labeled metrics with the same value
    // labelMapList = [labelMap1, labelMap2]
    withLabelMapList(labelMapList, value=1):: {
      values+: std.foldr(
        function(data, acc)
          acc + generateValues(data, value),
        labelMapList,
        []
      ),
    },
  },
}
