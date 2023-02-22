{

  local k = import 'k.libsonnet',
  local kausal = import 'ksonnet-util/kausal.libsonnet',

  // local k = import 'ksonnet-util/kausal.libsonnet',
  local deployment = k.apps.v1.deployment,
  local container = k.core.v1.container,
  local volumeMount = k.core.v1.volumeMount,
  local containerPort = k.core.v1.containerPort,
  local volume = k.core.v1.volume,
  local statefulset = k.apps.v1.statefulSet,

  new(target, image='grafana/tempo:latest', port=3200): {
    local this = self,

    configVolumeName:: 'tempo-conf',
    overridesVolumeName:: 'overrides',

    containerArgs:: {
      target: target,
      'config.file': '/conf/tempo.yaml',
      'mem-ballast-size-mbs': $._config.ballast_size_mbs,
    },

    workload: {},

    svc::
      kausal.util.serviceFor(this.workload),

    container::
      container.new(target, image)
      + container.withPorts(
        containerPort.new('prom-metrics', port)
      )
      + container.withArgs($.util.mapToFlags(this.containerArgs)) +
      (if $._config.variables_expansion then container.withEnvMixin($._config.variables_expansion_env_mixin) else {}) +
      container.withVolumeMounts([
        volumeMount.new(this.configVolumeName, '/conf'),
        volumeMount.new(this.overridesVolumeName, '/overrides'),
      ]) +
      $.util.withResources($._config.distributor.resources) +
      $.util.readinessProbe +
      (if $._config.variables_expansion then container.withArgsMixin(['-config.expand-env=true']) else {}),

    deployment::
      deployment.new(target,
                     1,
                     this.container,
                     {
                       app: target,
                       [$._config.gossip_member_label]: 'true',
                     }) +
      deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge(3) +
      deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable(1) +
      deployment.mixin.spec.template.spec.withTerminationGracePeriodSeconds(60) +
      deployment.mixin.spec.template.metadata.withAnnotations({
        config_hash: std.md5(std.toString($.tempo_distributor_configmap.data['tempo.yaml'])),
      }) +
      deployment.mixin.spec.template.spec.withVolumes([
        volume.fromConfigMap(this.configVolumeName, $.tempo_distributor_configmap.metadata.name),
        volume.fromConfigMap(this.overridesVolumeName, $._config.overrides_configmap_name),
      ]),

  },

  withPorts(ports=[]): {
    container+:
      container.withPorts(ports),
  },

  withDeployment(): {
    workload: self.deployment,
    service: self.svc,
  },

  withReplicas(replicas): {
    deployment+:
      deployment.spec.withReplicas(replicas),

    statefulset+:
      statefulset.spec.withReplicas(replicas),
  },

}
