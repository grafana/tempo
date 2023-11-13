{
  util+:: {
    local k = import 'ksonnet-util/kausal.libsonnet',
    local container = k.core.v1.container,
    local service = k.core.v1.service,

    withResources(resources)::
      k.util.resourcesRequests(resources.requests.cpu, resources.requests.memory) +
      k.util.resourcesLimits(resources.limits.cpu, resources.limits.memory),

    readinessProbe::
      container.mixin.readinessProbe.httpGet.withPath('/ready') +
      container.mixin.readinessProbe.httpGet.withPort($._config.port) +
      container.mixin.readinessProbe.withInitialDelaySeconds(15) +
      container.mixin.readinessProbe.withTimeoutSeconds(1),

    withInet6():: {
      tempo_compactor_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      tempo_distributor_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      tempo_ingester_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      tempo_querier_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      tempo_query_frontend_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      tempo_query_frontend_discovery_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      tempo_metrics_generator_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      gossip_ring_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      ingest_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      memcached+: {
        service+:
          service.mixin.spec.withIpFamilies(['IPv6']),
      },
      tempo_config+:: {
        server+: {
          http_listen_address: '::0',
          grpc_listen_address: '::0',
        },
        ingester+: {
          lifecycler+: {
            enable_inet6: true,
          },
        },
        memberlist+: {
          bind_addr: ['::'],
        },
        compactor+: {
          ring+: {
            enable_inet6: true,
          },
        },
        metrics_generator+: {
          ring+: {
            enable_inet6: true,
          },
        },
      },
    },
  },
}
