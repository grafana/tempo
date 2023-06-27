{

  local k = import 'k.libsonnet',
  local kausal = import 'ksonnet-util/kausal.libsonnet',

  local configMap = k.core.v1.configMap,
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local deployment = k.apps.v1.deployment,
  local pvc = k.core.v1.persistentVolumeClaim,
  local service = k.core.v1.service,
  local servicePort = k.core.v1.servicePort,
  local statefulset = k.apps.v1.statefulSet,
  local volume = k.core.v1.volume,
  local volumeMount = k.core.v1.volumeMount,
  local envVar = k.core.v1.envVar,

  newTempoComponent(target, image='grafana/tempo:latest', port=3200):: {
    local this = self,

    configVolumeName:: 'tempo-conf',
    overridesVolumeName:: 'overrides',
    dataVolumeName:: '%s-data' % target,
    ephemeralDataVolumeName:: '%s-ephemeral-data' % target,

    containerArgs:: {
      target: target,
      'config.file': '/conf/tempo.yaml',
      'mem-ballast-size-mbs': this.config.ballast_size_mbs,
    },

    // Use withResources()
    containerResources:: {},

    workload: {},

    appLabels:: {
      app: target,
    },

    config:: {},
    configData:: {},
    configmap:
      configMap.new('tempo-%s' % target) +
      configMap.withData({
        'tempo.yaml': kausal.util.manifestYaml(this.configData),
      }),

    readinessProbe::
      container.mixin.readinessProbe.httpGet.withPath('/ready') +
      container.mixin.readinessProbe.httpGet.withPort(this.config.port) +
      container.mixin.readinessProbe.withInitialDelaySeconds(15) +
      container.mixin.readinessProbe.withTimeoutSeconds(1),

    service: {},
    svc::
      kausal.util.serviceFor(this.workload),

    local podDisruptionBudget = k.policy.v1.podDisruptionBudget,
    podDisruptionBudget: {},
    pdb::
      podDisruptionBudget.new(target)
      + podDisruptionBudget.mixin.metadata.withLabels({ name: target })
      + podDisruptionBudget.mixin.spec.selector.withMatchLabels({ name: target })
      + podDisruptionBudget.mixin.spec.withMaxUnavailable(1),

    discoveryService: {},
    discoverySvc::
      kausal.util.serviceFor(this.workload)
      + service.mixin.spec.withPortsMixin([
        servicePort.withName('grpc')
        + servicePort.withPort(9095)
        + servicePort.withTargetPort(9095),
      ])
      + service.spec.withPublishNotReadyAddresses(true)
      + service.spec.withClusterIP('None')
      + service.metadata.withName('%s-discovery' % target),

    dataPVC::
      pvc.new(this.dataVolumeName)
      + pvc.mixin.spec.withAccessModes(['ReadWriteOnce'])
      + pvc.mixin.metadata.withLabels(this.appLabels)
      + pvc.mixin.metadata.withNamespace(this.config.namespace),

    container::
      container.new(target, image)
      + container.withPorts(
        containerPort.newNamed(port, 'prom-metrics')
      )
      + container.withArgs(kausal.util.mapToFlags(this.containerArgs)) +
      (if this.config.variables_expansion then container.withEnvMixin(this.config.variables_expansion_env_mixin) else {}) +
      container.withVolumeMounts([
        volumeMount.new(this.configVolumeName, '/conf'),
        volumeMount.new(this.overridesVolumeName, '/overrides'),
      ]) +
      // $.util.withResources($._config.distributor.resources) +
      this.readinessProbe +
      (if this.config.variables_expansion then container.withArgsMixin(['-config.expand-env=true']) else {}),

    deployment::
      deployment.new(
        target,
        1,
        this.container,
        this.appLabels,
      ) +
      deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge('50%') +
      deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable('100%') +
      deployment.mixin.spec.template.spec.withTerminationGracePeriodSeconds(30) +
      deployment.mixin.spec.template.metadata.withAnnotations({
        config_hash: std.md5(std.toString(this.configmap.data['tempo.yaml'])),
      }) +
      deployment.mixin.spec.template.spec.withVolumes([
        volume.fromConfigMap(this.configVolumeName, this.configmap.metadata.name),
        volume.fromConfigMap(this.overridesVolumeName, this.config.overrides_configmap_name),
      ]),

    statefulset::
      statefulset.new(
        target,
        3,
        this.container,
        this.dataPVC,
        this.appLabels,
      )
      + kausal.util.antiAffinityStatefulSet
      + statefulset.mixin.spec.withServiceName(target)
      + statefulset.mixin.spec.template.metadata.withAnnotations({
        config_hash: std.md5(std.toString(this.configmap.data['tempo.yaml'])),
      })
      + statefulset.mixin.spec.template.spec.withVolumes([
        volume.fromConfigMap(this.configVolumeName, this.configmap.metadata.name),
        volume.fromConfigMap(this.overridesVolumeName, this.config.overrides_configmap_name),
      ]) +
      statefulset.mixin.spec.withPodManagementPolicy('Parallel') +
      statefulset.mixin.spec.template.spec.withTerminationGracePeriodSeconds(1200) +
      kausal.util.podPriority('high'),
  },

  withSlowRollout():: {
    deployment+:
      deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge(3) +
      deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable(1),
  },

  withPorts(ports=[]):: {
    container+:
      container.withPorts(ports),
  },

  withDeployment():: {
    local this = self,

    workload: this.deployment,
    service: this.svc,
  },

  withStatefulset():: {
    local this = self,

    workload: this.statefulset,
    service: this.svc,
  },

  withDiscoveryService():: {
    local this = self,

    discoveryService: this.discoverySvc,
  },

  withReplicas(replicas):: {
    deployment+:
      deployment.spec.withReplicas(replicas),

    statefulset+:
      statefulset.spec.withReplicas(replicas),
  },

  withResources(resources):: {
    containerResources: resources,

    container+:
      kausal.util.resourcesRequests(resources.requests.cpu, resources.requests.memory)
      + kausal.util.resourcesLimits(resources.limits.cpu, resources.limits.memory),
  },

  withGlobalConfig(config={}):: {
    config: config,
  },

  withConfigData(data={}):: {
    configData: data,
  },

  withGossipSelector():: {
    local this = self,

    local gossipMemberLabel = {
      [this.config.gossip_member_label]: 'true',
    },

    deployment+:
      deployment.spec.selector.withMatchLabelsMixin(gossipMemberLabel)
      + deployment.spec.template.metadata.withLabelsMixin(gossipMemberLabel),

    statefulset+:
      statefulset.spec.selector.withMatchLabelsMixin(gossipMemberLabel)
      + statefulset.spec.template.metadata.withLabelsMixin(gossipMemberLabel),
  },

  withPVC(size, storage_class):: {
    local this = self,

    dataPVC+:
      pvc.mixin.spec.resources.withRequests({ storage: size })
      + pvc.mixin.spec.withStorageClassName(storage_class),

    container+:
      container.withVolumeMountsMixin([
        volumeMount.new(this.dataVolumeName, '/var/tempo'),
      ]),

    deployment+:
      deployment.spec.template.spec.withVolumesMixin([
        volume.fromPersistentVolumeClaim(this.dataVolumeName, this.dataPVC.metadata.name),
      ]),

    // The PVC is already included on the statefulset.new() call, and so we don't need to handle it here.
    // statefulset+:
    //   statefulset.spec.withVolumeClaimTemplatesMixin(
    //     this.dataPVC
    //   ),

    // + statefulset.spec.template.spec.withVolumesMixin([
    //   volume.fromPersistentVolumeClaim(this.dataVolumeName, this.dataPVC.metadata.name),
    // ]),
  },

  withEphemeralStorage(request, limit, mountpoint):: {
    local this = self,

    container+:
      container.mixin.resources.withRequestsMixin({ 'ephemeral-storage': request })
      + container.mixin.resources.withLimitsMixin({ 'ephemeral-storage': limit })
      + container.withVolumeMountsMixin([
        volumeMount.new(this.ephemeralDataVolumeName, mountpoint),
      ]),

    deployment+:
      deployment.spec.template.spec.withVolumesMixin([
        volume.fromEmptyDir(this.ephemeralDataVolumeName),
      ]),
  },

  withPDB(max_unavailable=1):: {
    local this = self,
    local podDisruptionBudget = k.policy.v1.podDisruptionBudget,

    pdb+:
      podDisruptionBudget.mixin.spec.withMaxUnavailable(1),

    podDisruptionBudget: this.pdb,
  },

  withGoMemLimit():: {
    local this = self,

    container+:
      if this.containerResources.limits.memory != null then
        container.withEnvMixin([envVar.new('GOMEMLIMIT', this.containerResources.limits.memory + 'B')])
      else
        {},
  },
}
