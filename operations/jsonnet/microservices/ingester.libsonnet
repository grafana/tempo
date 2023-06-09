{
  local k = import 'k.libsonnet',
  local kausal = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local containerPort = kausal.core.v1.containerPort,
  local volumeMount = k.core.v1.volumeMount,
  local pvc = k.core.v1.persistentVolumeClaim,
  local statefulset = k.apps.v1.statefulSet,
  local volume = k.core.v1.volume,
  local service = k.core.v1.service,
  local servicePort = k.core.v1.servicePort,

  local target_name = 'ingester',
  local tempo_config_volume = 'tempo-conf',
  local tempo_data_volume = 'ingester-data',
  local tempo_overrides_config_volume = 'overrides',


  local component = import 'component.libsonnet',
  local target = component.newTempoComponent(target_name)
                 + component.withConfigData($.tempo_ingester_config)
                 + component.withGlobalConfig($._config)
                 + component.withGossipSelector()
                 + component.withPDB(max_unavailable=1)
                 + component.withPVC(size=$._config.ingester.pvc_size, storage_class=$._config.ingester.pvc_storage_class)
                 + component.withReplicas($._config.ingester.replicas)
                 + component.withResources($._config.ingester.resources)
                 + component.withStatefulset()
  ,

  ingester:
    target
    {
      // Backwards compatibility for user overrides
      container+: $.tempo_ingester_container,
      statefulset+: $.tempo_ingester_statefulset,
      service+: $.tempo_ingester_service,
      podDisruptionBudget+: $.ingester_pdb,
    },

  tempo_ingester_container:: {},
  tempo_ingester_statefulset:: {},
  tempo_ingester_service:: {},
  ingester_pdb:: {},
}
