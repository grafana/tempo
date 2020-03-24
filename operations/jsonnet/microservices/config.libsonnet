{
  _images+:: {
    tempo: 'joeelliott/canary-frigg:84a516fd',
    tempo_query: 'joeelliott/canary-frigg-query:84a516fd',
  },

  _config+:: {
    gossip_member_label: 'tempo-gossip-member',
    compactor: {
      pvc_size: error 'Must specify a compactor pvc size',
      pvc_storage_class: error 'Must specify a compactor pvc storage class',
      replicas: 1,
    },
    querier: {
      pvc_size: error 'Must specify a querier pvc size',
      pvc_storage_class: error 'Must specify a querier pvc storage class',
      replicas: 1,
    },
    ingester: {
      pvc_size: error 'Must specify an ingester pvc size',
      pvc_storage_class: error 'Must specify an ingester pvc storage class',
      replicas: 3,
    },
    distributor: {
      receivers: error 'Must specify receivers',
      replicas: 1,
    },
    jaeger_ui: {
      base_path: '/',
    },
    ballast_size_mbs: '1024',
    port: 3100,
    gossip_ring_port: 7946,
    gcs_bucket: error 'Must specify a bucket',
  },

  tempo_compactor_container+::
    $.util.resourcesRequests('500m', '3Gi') +
    $.util.resourcesLimits('1', '5Gi'),

  tempo_distributor_container+::
    $.util.resourcesRequests('3', '3Gi') +
    $.util.resourcesLimits('5', '5Gi'),

  tempo_ingester_container+::
    $.util.resourcesRequests('3', '3Gi') +
    $.util.resourcesLimits('5', '5Gi'),

  tempo_querier_container+::
    $.util.resourcesRequests('500m', '1Gi') +
    $.util.resourcesLimits('1', '2Gi'),
}
