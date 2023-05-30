{
  _images+:: {
    tempo: 'grafana/tempo:latest',
    tempo_query: 'grafana/tempo-query:latest',
    tempo_vulture: 'grafana/tempo-vulture:latest',
  },

  _config+:: {
    tempo: {
      port: 3200,
      replicas: 1,
      headless_service_name: 'tempo-members',
    },
    // disable tempo-query by default
    tempo_query: {
      enabled: false,
    },
    pvc_size: error 'Must specify a pvc size',
    pvc_storage_class: error 'Must specify a pvc storage class',
    receivers: error 'Must specify receivers',
    ballast_size_mbs: '1024',
    jaeger_ui: {
      base_path: '/',
    },
  },
}
