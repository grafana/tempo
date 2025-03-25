{

  local k = import 'k.libsonnet',
  local kausal = import 'ksonnet-util/kausal.libsonnet',

  local container = k.core.v1.container,
  local volumeMount = k.core.v1.volumeMount,
  local statefulset = k.apps.v1.statefulSet,

  tempo_chown_container(data_volume, uid='10001', gid='10001', dir='/var/tempo')::
    container.new('chown-' + data_volume, $._images.tempo) +
    container.withCommand('chown') +
    container.withArgs([
      '-R',
      '%s:%s' % [uid, gid],
      dir,
    ]) +
    container.withVolumeMounts([
      volumeMount.new(data_volume, dir),
    ]) +
    container.securityContext.withRunAsUser(0) +
    container.securityContext.withRunAsGroup(0) +
    {},

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
      tempo_block_builder_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      tempo_backend_scheduler_service+:
        service.mixin.spec.withIpFamilies(['IPv6']),
      tempo_backend_worker_service+:
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
        backend_worker+: {
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

    withInitChown():: {
      tempo_metrics_generator_statefulset+:
        statefulset.spec.template.spec.withInitContainers([
          $.tempo_chown_container('metrics-generator-data'),
        ]),

      tempo_ingester_statefulset+:
        statefulset.spec.template.spec.withInitContainers([
          $.tempo_chown_container('ingester-data'),
        ]),
    },
  },
}
