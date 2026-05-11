local default = import 'environments/default/main.jsonnet';
local base = import 'outputs/base.json';
local test = import 'testonnet/main.libsonnet';

// Helper: default env with backend_worker KEDA enabled and a given Prometheus URL/tenant.
local withBackendWorkerKeda(url, tenant='') = default {
  _config+:: {
    autoscaling_prometheus_url: url,
    autoscaling_prometheus_tenant: tenant,
    backend_worker+: { keda+: { enabled: true } },
  },
};

// Helper: default env with live_store KEDA enabled and a given Prometheus URL/tenant.
local withLiveStoreKeda(url, tenant='') = default {
  _config+:: {
    autoscaling_prometheus_url: url,
    autoscaling_prometheus_tenant: tenant,
    live_store+: { replicas: 1, keda+: { enabled: true } },
    rollout_operator_replica_template_access_enabled: true,
  },
};

test.new(std.thisFile)
+ test.case.new(
  'Basic',
  test.expect.eq(
    default,
    base
  )
)
+ test.case.new(
  'backend_worker KEDA Prometheus trigger uses autoscaling_prometheus_url',
  test.expect.eq(
    withBackendWorkerKeda('http://prometheus:9090').tempo_backend_worker_scaled_object.spec.triggers[0].metadata.serverAddress,
    'http://prometheus:9090'
  )
)
+ test.case.new(
  'backend_worker KEDA Prometheus trigger includes X-Scope-OrgID header when tenant is set',
  test.expect.eq(
    withBackendWorkerKeda('http://prometheus:9090', 'my-tenant').tempo_backend_worker_scaled_object.spec.triggers[0].metadata.customHeaders,
    'X-Scope-OrgID=my-tenant'
  )
)
+ test.case.new(
  'backend_worker KEDA Prometheus trigger omits customHeaders when tenant is empty',
  test.expect.eq(
    std.objectHas(
      withBackendWorkerKeda('http://prometheus:9090').tempo_backend_worker_scaled_object.spec.triggers[0].metadata,
      'customHeaders'
    ),
    false
  )
)
+ test.case.new(
  'live_store KEDA ScaledObject targets the ReplicaTemplate',
  test.expect.eq(
    withLiveStoreKeda('http://prometheus:9090').tempo_live_store_scaled_object.spec.scaleTargetRef.name,
    'live-store'
  )
)
+ test.case.new(
  'live_store KEDA ScaledObject scaleTargetRef kind is ReplicaTemplate',
  test.expect.eq(
    withLiveStoreKeda('http://prometheus:9090').tempo_live_store_scaled_object.spec.scaleTargetRef.kind,
    'ReplicaTemplate'
  )
)
+ test.case.new(
  'live_store KEDA Prometheus trigger uses autoscaling_prometheus_url',
  test.expect.eq(
    withLiveStoreKeda('http://prometheus:9090').tempo_live_store_scaled_object.spec.triggers[0].metadata.serverAddress,
    'http://prometheus:9090'
  )
)
+ test.case.new(
  'live_store KEDA Prometheus trigger includes X-Scope-OrgID header when tenant is set',
  test.expect.eq(
    withLiveStoreKeda('http://prometheus:9090', 'my-tenant').tempo_live_store_scaled_object.spec.triggers[0].metadata.customHeaders,
    'X-Scope-OrgID=my-tenant'
  )
)
+ test.case.new(
  'live_store KEDA Prometheus trigger omits customHeaders when tenant is empty',
  test.expect.eq(
    std.objectHas(
      withLiveStoreKeda('http://prometheus:9090').tempo_live_store_scaled_object.spec.triggers[0].metadata,
      'customHeaders'
    ),
    false
  )
)
+ test.case.new(
  'live_store KEDA query embeds window_seconds from config',
  test.expect.eq(
    std.length(std.findSubstr(
      '1800',
      withLiveStoreKeda('http://prometheus:9090').tempo_live_store_scaled_object.spec.triggers[0].metadata.query,
    )) > 0,
    true
  )
)
+ test.case.new(
  'live_store KEDA default query uses 10m rate window',
  test.expect.eq(
    std.length(std.findSubstr(
      '[10m]',
      withLiveStoreKeda('http://prometheus:9090').tempo_live_store_scaled_object.spec.triggers[0].metadata.query,
    )) > 0,
    true
  )
)
+ test.case.new(
  'live_store KEDA additional_triggers are appended after the default trigger',
  test.expect.eq(
    (default {
       _config+:: {
         autoscaling_prometheus_url: 'http://prometheus:9090',
         rollout_operator_replica_template_access_enabled: true,
         live_store+: {
           replicas: 1,
           keda+: {
             enabled: true,
             additional_triggers: [{ type: 'cpu', metricType: 'Utilization', metadata: { value: '80' } }],
           },
         },
       },
     }).tempo_live_store_scaled_object.spec.triggers[1],
    { type: 'cpu', metricType: 'Utilization', metadata: { value: '80' } }
  )
)
+ test.case.new(
  'live_store KEDA enabled removes spec.replicas from block-builder',
  test.expect.eq(
    std.objectHas(
      withLiveStoreKeda('http://prometheus:9090').tempo_block_builder_statefulset.spec,
      'replicas'
    ),
    false
  )
)
+ test.case.new(
  'block_builder_max_unavailable defaults to 100%',
  test.expect.eq(
    default._config.block_builder_max_unavailable,
    '100%'
  )
)
+ test.case.new(
  'live_store KEDA enabled block_builder follows live-store zone-a',
  test.expect.eq(
    withLiveStoreKeda('http://prometheus:9090').tempo_block_builder_statefulset.spec.template.metadata.annotations['grafana.com/rollout-mirror-replicas-from-resource-name'],
    'live-store-zone-a'
  )
)
+ test.case.new(
  'live_store KEDA enabled injects partitions_per_instance into block-builder config',
  test.expect.eq(
    withLiveStoreKeda('http://prometheus:9090').tempo_block_builder_config.block_builder.partitions_per_instance,
    1
  )
)
+ test.case.new(
  'live_store KEDA disabled does not inject partitions_per_instance into block-builder config',
  test.expect.eq(
    std.get(
      std.get(default.tempo_block_builder_config, 'block_builder', {}),
      'partitions_per_instance',
      null
    ),
    null
  )
)
