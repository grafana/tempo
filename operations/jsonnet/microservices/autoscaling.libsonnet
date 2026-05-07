// KEDA-based horizontal pod autoscaling for Tempo microservices.
// Requires KEDA operator and CRDs installed in the cluster.
// All scalers are disabled by default; enable via _config.<component>.keda.enabled.
{
  local keda = (import 'github.com/jsonnet-libs/keda-libsonnet/2.15/main.libsonnet').keda.v1alpha1,
  local scaledObject = keda.scaledObject,
  local scaleUpBehavior = scaledObject.spec.advanced.horizontalPodAutoscalerConfig.behavior.scaleUp,
  local scaleDownBehavior = scaledObject.spec.advanced.horizontalPodAutoscalerConfig.behavior.scaleDown,

  _config+:: {
    // Prometheus server address used by all KEDA Prometheus triggers.
    // Must be set explicitly when enabling any Prometheus-based autoscaler (e.g. backend_worker).
    autoscaling_prometheus_url: '',
    // Optional tenant ID sent as the X-Scope-OrgID header on every KEDA Prometheus
    // scrape request. Required when the backend is a multi-tenant system such as
    // Grafana Mimir. Leave empty (default) for single-tenant setups.
    autoscaling_prometheus_tenant: '',

    distributor+: {
      keda: {
        enabled: false,
        min_replicas: 2,
        max_replicas: 200,
        paused_replicas: 0,
        // Target CPU per pod. 330m was observed to correlate to ~10MB/s throughput per distributor pod.
        target_cpu: '330m',
        // Scale up aggressively: add up to 5 pods every 15s.
        scale_up_stabilization_window_seconds: 15,
        scale_up_pods: 5,
        scale_up_period_seconds: 15,
        // Keep highest replica count from last 30 minutes to prevent flapping.
        scale_down_stabilization_window_seconds: 60 * 30,
        // 0 means use a default 100% Percent policy (i.e. no pod-count limit on scale-down,
        // relying on the stabilization window to prevent flapping).
        scale_down_pods: 0,
        scale_down_period_seconds: 60 * 5,
      },
    },

    metrics_generator+: {
      keda: {
        enabled: false,
        min_replicas: 1,
        max_replicas: 200,
        paused_replicas: 0,
        target_cpu: '500m',
        scale_up_stabilization_window_seconds: 60,
        scale_up_pods: 5,
        scale_up_period_seconds: 60,
        scale_down_stabilization_window_seconds: 60 * 5,
        scale_down_pods: 1,
        scale_down_period_seconds: 60 * 5,
      },
    },

    backend_worker+: {
      keda: {
        enabled: false,
        min_replicas: 3,
        max_replicas: 200,
        paused_replicas: 0,
        // Average outstanding blocks per pod threshold.
        threshold: 200,
        // Prometheus query returning the average outstanding blocks per backend-worker.
        query: |||
          sum(avg_over_time(tempodb_compaction_outstanding_blocks{namespace="%(namespace)s", container="backend-scheduler"}[10m]))
          /
          sum(avg_over_time(kube_statefulset_status_replicas{namespace="%(namespace)s", statefulset="backend-worker"}[10m]))
        ||| % { namespace: $._config.namespace },
        // Scale up in large chunks to catch up on work, but limit frequency to avoid ring churn.
        scale_up_stabilization_window_seconds: 60 * 15,
        scale_up_pods: 20,
        scale_up_period_seconds: 60 * 30,
        // Scale down gradually.
        scale_down_stabilization_window_seconds: 60 * 15,
        scale_down_pods: 5,
        scale_down_period_seconds: 60 * 15,
      },
    },

    // Block builder scales to match live-store zone-a pods via KEDA kubernetes-workload scaler.
    // See: https://github.com/grafana/tempo/issues/6933
    block_builder+: {
      keda: {
        enabled: false,
        min_replicas: 1,
        max_replicas: 200,
        paused_replicas: 0,
        // Number of partitions each block-builder pod handles.
        // KEDA sets replicas = ceil(live-store-zone-a pods / partitions_per_instance).
        partitions_per_instance: 1,
        // Pod selector to count live-store zone-a pods.
        pod_selector: 'name=live-store-zone-a',
        scale_up_stabilization_window_seconds: 60,
        scale_up_pods: 5,
        scale_up_period_seconds: 60,
        scale_down_stabilization_window_seconds: 60 * 5,
        scale_down_pods: 5,
        scale_down_period_seconds: 60 * 5,
      },
    },

    // Override block_builder_max_unavailable when block_builder autoscaling is enabled
    // since the replica count is dynamic.
    block_builder_max_unavailable:
      if $._config.block_builder.keda.enabled then '100%'
      else $.tempo_block_builder_statefulset.spec.replicas,
  },

  // Returns a KEDA Prometheus trigger using the shared autoscaling_prometheus_url and
  // autoscaling_prometheus_tenant config. All Prometheus-based scalers should use this
  // so that the URL and X-Scope-OrgID header are configured in one place.
  prometheusTrigger(query, metricName, threshold):: {
    type: 'prometheus',
    metadata: {
      serverAddress: $._config.autoscaling_prometheus_url,
      [if $._config.autoscaling_prometheus_tenant != '' then 'customHeaders']:
        'X-Scope-OrgID=%s' % $._config.autoscaling_prometheus_tenant,
      metricName: metricName,
      query: query,
      threshold: threshold,
    },
  },

  // Create a KEDA ScaledObject for a target controller with configurable scaling behavior.
  scaledObjectForController(target, configKey)::
    assert std.objectHas($._config, configKey) : '$._config must have key ' + configKey;
    assert std.objectHas($._config[configKey], 'keda') : '$._config.%s must have key "keda"' % configKey;

    local config = $._config[configKey].keda;
    assert config.min_replicas >= 0 : 'min_replicas must be >= 0';
    assert config.min_replicas <= config.max_replicas : 'min_replicas must be <= max_replicas';
    assert config.scale_up_pods > 0 : 'scale_up_pods must be > 0';
    assert config.scale_up_period_seconds > 0 : 'scale_up_period_seconds must be > 0';
    assert config.scale_down_pods >= 0 : 'scale_down_pods must be >= 0';
    assert config.scale_down_period_seconds > 0 : 'scale_down_period_seconds must be > 0';

    scaledObject.new(target.metadata.name)
    + scaledObject.spec.scaleTargetRef.withKind(target.kind)
    + scaledObject.spec.scaleTargetRef.withName(target.metadata.name)
    + (if std.objectHas(target, 'apiVersion') then scaledObject.spec.scaleTargetRef.withApiVersion(target.apiVersion) else {})
    + scaledObject.spec.withMinReplicaCount(config.min_replicas)
    + scaledObject.spec.withMaxReplicaCount(config.max_replicas)
    + scaledObject.spec.withPollingInterval(60)
    + (
      if std.objectHas(config, 'paused_replicas') && config.paused_replicas > 0 then
        scaledObject.metadata.withAnnotationsMixin({
          'autoscaling.keda.sh/paused': 'true',
          'autoscaling.keda.sh/paused-replicas': '%s' % config.paused_replicas,
        })
      else {}
    )
    + scaleUpBehavior.withStabilizationWindowSeconds(config.scale_up_stabilization_window_seconds)
    + scaleUpBehavior.withPoliciesMixin({
      value: config.scale_up_pods,
      type: 'Pods',
      periodSeconds: config.scale_up_period_seconds,
    })
    + scaleDownBehavior.withStabilizationWindowSeconds(config.scale_down_stabilization_window_seconds)
    + (
      if config.scale_down_pods > 0 then
        scaleDownBehavior.withPoliciesMixin({
          value: config.scale_down_pods,
          type: 'Pods',
          periodSeconds: config.scale_down_period_seconds,
        })
      else
        // Explicit default policy so the stabilization window isn't ignored
        // due to an empty policy list.
        scaleDownBehavior.withPoliciesMixin({
          value: 100,
          type: 'Percent',
          periodSeconds: config.scale_down_period_seconds,
        })
    ),

  //
  // Distributor: CPU-based autoscaling.
  //
  tempo_distributor_scaled_object:
    if $._config.distributor.keda.enabled then
      $.scaledObjectForController($.tempo_distributor_deployment, 'distributor')
      + scaledObject.spec.withTriggersMixin([{
        type: 'cpu',
        metricType: 'AverageValue',
        metadata: {
          value: $._config.distributor.keda.target_cpu,
        },
      }])
      + scaledObject.spec.withInitialCooldownPeriod(90)
    else {},

  tempo_distributor_deployment+:
    if $._config.distributor.keda.enabled then $.removeReplicasFromSpec else {},

  //
  // Metrics Generator: CPU-based autoscaling.
  //
  tempo_metrics_generator_scaled_object:
    if $._config.metrics_generator.keda.enabled then
      $.scaledObjectForController($.tempo_metrics_generator_statefulset, 'metrics_generator')
      + scaledObject.spec.withTriggersMixin([{
        type: 'cpu',
        metricType: 'AverageValue',
        metadata: {
          value: $._config.metrics_generator.keda.target_cpu,
        },
      }])
    else {},

  tempo_metrics_generator_statefulset+:
    if $._config.metrics_generator.keda.enabled then $.removeReplicasFromSpec else {},

  //
  // Backend Worker: Prometheus-based autoscaling on outstanding blocks.
  //
  tempo_backend_worker_scaled_object:
    if $._config.backend_worker.keda.enabled then
      assert $._config.autoscaling_prometheus_url != '' : 'autoscaling_prometheus_url is required for backend_worker autoscaling';
      $.scaledObjectForController($.tempo_backend_worker_statefulset, 'backend_worker')
      + scaledObject.spec.withTriggersMixin([
        $.prometheusTrigger(
          query=$._config.backend_worker.keda.query,
          metricName='tempodb_compaction_outstanding_blocks',
          threshold='%d' % $._config.backend_worker.keda.threshold,
        ),
      ])
    else {},

  tempo_backend_worker_statefulset+:
    if $._config.backend_worker.keda.enabled then $.removeReplicasFromSpec else {},

  //
  // Block Builder: KEDA kubernetes-workload scaler matching live-store zone-a pod count.
  //
  tempo_block_builder_scaled_object:
    if $._config.block_builder.keda.enabled then
      assert $._config.block_builder.keda.partitions_per_instance > 0 : 'block_builder.keda.partitions_per_instance must be > 0';
      $.scaledObjectForController($.tempo_block_builder_statefulset, 'block_builder')
      + scaledObject.spec.withTriggersMixin([{
        type: 'kubernetes-workload',
        metadata: {
          podSelector: $._config.block_builder.keda.pod_selector,
          value: '%d' % $._config.block_builder.keda.partitions_per_instance,
        },
      }])
    else {},

  tempo_block_builder_statefulset+:
    if $._config.block_builder.keda.enabled then $.removeReplicasFromSpec else {},
}
