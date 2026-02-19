// The jsonnet file used to generate the Kubernetes manifests.

local tempo = import 'microservices/tempo.libsonnet';

tempo
+ tempo.util.withInet6()
  {
  _images+:: {
    tempo: 'grafana/tempo:latest',
    tempo_vulture: 'grafana/tempo-vulture:latest',
    tempo_query: 'grafana/tempo-query:latest',
  },

  // generate with `tempo_query.enabled: true` to include tempo-query manifests
  _config+:: {
    namespace: 'tracing',

    backend_scheduler+: {
      pvc_size: '200Mi',
      pvc_storage_class: 'fast',
      vpa+: {
        enabled: true,
      },
      pdb+: {
        enabled: true,
      },
    },

    backend_worker+: {
      vpa+: {
        enabled: true,
      },
      pdb+: {
        enabled: true,
      },
    },

    block_builder+: {
      vpa+: {
        enabled: true,
      },
      pdb+: {
        enabled: true,
      },
    },

    compactor+: {
      replicas: 5,
    },

    live_store+: {
      pvc_size: '10Gi',
      pvc_storage_class: 'fast',

      vpa+: {
        enabled: true,
      },
    },


    metrics_generator+: {
      pvc_size: '10Gi',
      pvc_storage_class: 'fast',
      ephemeral_storage_request_size: '10Gi',
      ephemeral_storage_limit_size: '11Gi',
      vpa+: {
        enabled: true,
      },
      pdb+: {
        enabled: true,
      },
    },

    query_frontend+: {
      replicas: 2,
      pdb+: {
        enabled: true,
      },
    },

    querier+: {
      replicas: 5,
      vpa+: {
        enabled: true,
      },
      pdb+: {
        enabled: true,
      },
    },
    distributor+: {
      replicas: 5,
      receivers: {
        jaeger: {
          protocols: {
            grpc: {
              endpoint: '0.0.0.0:14250',
            },
          },
        },
        otlp: {
          protocols: {
            grpc: {
              endpoint: '0.0.0.0:4317',
            },
          },
        },
      },
      pdb+: {
        enabled: true,
      },
    },
    memcached+: {
      replicas: 5,
      pdb+: {
        enabled: true,
      },
    },
    vulture+: {
      replicas: 1,
      tempoOrgId: '1',
      tempoPushUrl: 'http://distributor',
      tempoQueryUrl: 'http://query-frontend:3200/tempo',
    },
    jaeger_ui: {
      base_path: '/tempo',
    },
    backend: 'gcs',
    bucket: 'tempo',

    // NOTE: Enable the ReplicaTemplate role if intend to use the rollout-operator.
    // rollout_operator_replica_template_access_enabled: true,

    // NOTE: Disable the custom resource installation for the rollout-operator
    // if you have already installed the custom resources in your cluster, or
    // if you want to manage them separately.
    // zpdb_custom_resource_definition_enabled: false,
    // replica_template_custom_resource_definition_enabled: false,
  },

}
