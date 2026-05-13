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

// Helper: live-store KEDA only (no block-builder autoscaling).
local withLiveStoreKeda(url, tenant='') = default {
  _config+:: {
    autoscaling_prometheus_url: url,
    autoscaling_prometheus_tenant: tenant,
    live_store+: { replicas: 1, keda+: { enabled: true } },
  },
};

// Helper: live-store + block-builder KEDA, rollout-operator approach (default).
// Requires live_store.keda.enabled=true; rollout_operator_replica_template_access_enabled is auto-derived.
local withBlockBuilderKedaRolloutOp(url, tenant='') = default {
  _config+:: {
    autoscaling_prometheus_url: url,
    autoscaling_prometheus_tenant: tenant,
    live_store+: { replicas: 1, keda+: { enabled: true } },
    block_builder+: { keda+: { enabled: true } },
  },
};

// Helper: block-builder KEDA, keda approach (live-store KEDA also enabled here but not required).
local withBlockBuilderKedaKeda(url) = default {
  _config+:: {
    autoscaling_prometheus_url: url,
    live_store+: { replicas: 1, keda+: { enabled: true } },
    block_builder+: { keda+: { enabled: true, scaling: 'keda' } },
  },
};

// Helper: block-builder KEDA (keda approach) without live-store KEDA — proves decoupling.
local withBlockBuilderKedaKedaOnly(url) = default {
  _config+:: {
    autoscaling_prometheus_url: url,
    block_builder+: { keda+: { enabled: true, scaling: 'keda' } },
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
  'live_store KEDA default query rate window matches window_seconds',
  test.expect.eq(
    std.length(std.findSubstr(
      '[1800s]',
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
  'block_builder KEDA enabled removes spec.replicas from block-builder',
  test.expect.eq(
    std.objectHas(
      withBlockBuilderKedaRolloutOp('http://prometheus:9090').tempo_block_builder_statefulset.spec,
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
  'block_builder KEDA rollout-operator approach block_builder follows live-store zone-a',
  // rollout-operator reads grafana.com/rollout-mirror-replicas-from-resource-* from StatefulSet
  // metadata.annotations, NOT from the pod template spec.template.metadata.annotations.
  test.expect.eq(
    withBlockBuilderKedaRolloutOp('http://prometheus:9090').tempo_block_builder_statefulset.metadata.annotations['grafana.com/rollout-mirror-replicas-from-resource-name'],
    'live-store-zone-a'
  )
)
+ test.case.new(
  'block_builder KEDA enabled injects partitions_per_instance into block-builder config',
  test.expect.eq(
    withBlockBuilderKedaRolloutOp('http://prometheus:9090').tempo_block_builder_config.block_builder.partitions_per_instance,
    1
  )
)
+ test.case.new(
  'block_builder KEDA disabled does not inject partitions_per_instance into block-builder config',
  test.expect.eq(
    std.get(
      std.get(default.tempo_block_builder_config, 'block_builder', {}),
      'partitions_per_instance',
      null
    ),
    null
  )
)
+ test.case.new(
  'block_builder KEDA default scaling is rollout-operator',
  test.expect.eq(
    (default { _config+:: { block_builder+: { keda+: { enabled: true } } } })._config.block_builder.keda.scaling,
    'rollout-operator'
  )
)
+ test.case.new(
  'block_builder KEDA rollout-operator approach omits block-builder ScaledObject',
  test.expect.eq(
    withBlockBuilderKedaRolloutOp('http://prometheus:9090').tempo_block_builder_scaled_object,
    {}
  )
)
+ test.case.new(
  'live_store KEDA auto-enables rollout_operator_replica_template_access_enabled',
  test.expect.eq(
    withLiveStoreKeda('http://prometheus:9090')._config.rollout_operator_replica_template_access_enabled,
    true
  )
)
+ test.case.new(
  'block_builder KEDA rollout-operator approach auto-enables rollout_operator_replica_template_access_enabled',
  test.expect.eq(
    withBlockBuilderKedaRolloutOp('http://prometheus:9090')._config.rollout_operator_replica_template_access_enabled,
    true
  )
)
+ test.case.new(
  'block_builder KEDA keda approach auto-enables rollout_operator_replica_template_access_enabled via live-store',
  test.expect.eq(
    withBlockBuilderKedaKeda('http://prometheus:9090')._config.rollout_operator_replica_template_access_enabled,
    true
  )
)
+ test.case.new(
  'block_builder KEDA keda approach without live-store KEDA does not auto-enable rollout_operator_replica_template_access_enabled',
  test.expect.eq(
    withBlockBuilderKedaKedaOnly('http://prometheus:9090')._config.rollout_operator_replica_template_access_enabled,
    false
  )
)
+ test.case.new(
  'block_builder KEDA disabled omits rollout-mirror annotations from block-builder metadata',
  test.expect.eq(
    std.objectHas(
      std.get(default.tempo_block_builder_statefulset.metadata, 'annotations', {}),
      'grafana.com/rollout-mirror-replicas-from-resource-name'
    ),
    false
  )
)
+ test.case.new(
  'block_builder KEDA rollout-operator grants statefulsets/scale to rollout-operator role',
  test.expect.eq(
    std.filter(
      function(r) std.member(r.resources, 'statefulsets/scale'),
      withBlockBuilderKedaRolloutOp('http://prometheus:9090').rollout_operator_role.rules
    )[0].verbs,
    ['get']
  )
)
+ test.case.new(
  'block_builder KEDA keda approach creates block-builder ScaledObject',
  test.expect.eq(
    withBlockBuilderKedaKeda('http://prometheus:9090').tempo_block_builder_scaled_object.spec.scaleTargetRef.name,
    'block-builder'
  )
)
+ test.case.new(
  'block_builder KEDA keda approach block-builder ScaledObject uses kubernetes-workload trigger',
  test.expect.eq(
    withBlockBuilderKedaKeda('http://prometheus:9090').tempo_block_builder_scaled_object.spec.triggers[0].type,
    'kubernetes-workload'
  )
)
+ test.case.new(
  'block_builder KEDA keda approach omits rollout-mirror annotations from block-builder metadata',
  test.expect.eq(
    std.objectHas(
      std.get(withBlockBuilderKedaKeda('http://prometheus:9090').tempo_block_builder_statefulset.metadata, 'annotations', {}),
      'grafana.com/rollout-mirror-replicas-from-resource-name'
    ),
    false
  )
)
+ test.case.new(
  'block_builder KEDA keda approach uses pod_selector from block_builder.keda config',
  test.expect.eq(
    withBlockBuilderKedaKeda('http://prometheus:9090').tempo_block_builder_scaled_object.spec.triggers[0].metadata.podSelector,
    'name=live-store-zone-a'
  )
)
+ test.case.new(
  'block_builder KEDA keda approach works without live-store KEDA',
  test.expect.eq(
    withBlockBuilderKedaKedaOnly('http://prometheus:9090').tempo_block_builder_scaled_object.spec.scaleTargetRef.name,
    'block-builder'
  )
)
