local tanka = (import 'github.com/grafana/jsonnet-libs/tanka-util/main.libsonnet');
local helm = tanka.helm.new(std.thisFile);

{
  chartVersion:: '1.1.0',
  values:: {
    installCRDs: $._config.install_crds,
    global: {
      podSecurityPolicy: {
        enabled: true,
        useAppArmor: false,
      },
    },
  },

  local generated = helm.template('cert-manager', './charts/' + $.chartVersion, {
    values: $.values,
    namespace: $._config.namespace,
  }),

  // manual generated lib used different labels as selectors
  // keeping these minimizes impact on production
  local patch_labels(o, app, name) =
    local labels = {
      app: app,
      name: name,
    };

    if std.isObject(o)
    then
      if std.objectHas(o, 'kind')
      then
        local t = std.asciiLower(o.kind);
        if t == 'deployment' then
          o {
            metadata+: {
              labels+: labels,
            },
            spec+: {
              selector: {
                matchLabels: labels,
              },
              template+: {
                metadata+: {
                  labels: labels,
                },
              },
            },
          }
        else if t == 'service' then
          o {
            metadata+: {
              labels+: labels,
            },
            spec+: {
              selector: labels,
            },
          }
        else
          o {
            metadata+: {
              labels+: labels,
            },
          }
      else
        std.mapWithKey(
          function(key, obj)
            patch_labels(obj, app, name),
          o
        )
    else if std.isArray(o)
    then
      std.map(
        function(obj)
          patch_labels(obj, app, name),
        o
      )
    else o
  ,

  labeled: std.mapWithKey(
    function(key, obj)
      if std.length(std.findSubstr('cainjector', key)) > 0
      then patch_labels(obj, 'cainjector', 'cert-manager-cainjector')
      else
        if std.length(std.findSubstr('webhook', key)) > 0
        then patch_labels(obj, 'webhook', 'cert-manager-webhook')
        else patch_labels(obj, 'controller', 'cert-manager')
    ,
    generated
  ),

  crds:
    if $._config.custom_crds
    then std.native('parseYaml')(importstr 'files/00-crds.yaml')
    else {},
} + {
  local _containers = super.labeled.deployment_cert_manager.spec.template.spec.containers,
  labeled+: {
    deployment_cert_manager+: {
      spec+: {
        template+: {
          spec+: {
            containers: [
              _container {
                ports: [
                  {
                    // Convention in grafana/jsonnet-libs is to name the prometheus
                    // scraping port 'http-metrics' for service discovery
                    containerPort: 9402,
                    protocol: 'TCP',
                    name: 'http-metrics',
                  },
                ],
              }
              for _container in _containers
            ],
          },
        },
      },
    },
  },
}
