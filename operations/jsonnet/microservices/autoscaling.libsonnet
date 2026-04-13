{
  local keda = (import 'github.com/jsonnet-libs/keda-libsonnet/2.15/main.libsonnet').keda.v1alpha1,
  local scaledObject = keda.scaledObject,
  local scaleUpBehavior = scaledObject.spec.advanced.horizontalPodAutoscalerConfig.behavior.scaleUp,
  local scaleDownBehavior = scaledObject.spec.advanced.horizontalPodAutoscalerConfig.behavior.scaleDown,

  _config+:: {
    autoscaling: {
      // Prometheus server address for Prometheus-based autoscaling triggers.
      // Required when enabling backend_worker autoscaling.
      prometheus_address: '',

      distributor: {
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
        scale_down_pods: 0,
        scale_down_period_seconds: 60 * 5,
      },

      metrics_generator: {
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

      backend_worker: {
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

      live_store: {
        enabled: false,
        min_replicas: 1,
        max_replicas: 200,
        paused_replicas: 0,
        target_cpu: '1',
        scale_up_stabilization_window_seconds: 120,
        scale_up_pods: 5,
        scale_up_period_seconds: 120,
        // Scale down slowly: each scale-down takes time for the live-store to drain.
        scale_down_stabilization_window_seconds: 60 * 40,
        scale_down_pods: 5,
        scale_down_period_seconds: 60 * 15,
      },

      // Block builder scales to match live-store zone-a pods via KEDA kubernetes-workload scaler.
      // See: https://github.com/grafana/tempo/issues/6933
      block_builder: {
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
      if $._config.autoscaling.block_builder.enabled then '100%'
      else $.tempo_block_builder_statefulset.spec.replicas,
  },

  // Create a KEDA ScaledObject for a target controller with configurable scaling behavior.
  scaledObjectForController(target, config)::
    assert config.min_replicas >= 0 : 'min_replicas must be >= 0';
    assert config.min_replicas <= config.max_replicas : 'min_replicas must be <= max_replicas';

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
    if $._config.autoscaling.distributor.enabled then
      $.scaledObjectForController($.tempo_distributor_deployment, $._config.autoscaling.distributor)
      + scaledObject.spec.withTriggersMixin([{
        type: 'cpu',
        metricType: 'AverageValue',
        metadata: {
          value: $._config.autoscaling.distributor.target_cpu,
        },
      }])
      + scaledObject.spec.withInitialCooldownPeriod(90)
    else {},

  tempo_distributor_deployment+:
    if $._config.autoscaling.distributor.enabled then $.removeReplicasFromSpec else {},

  //
  // Metrics Generator: CPU-based autoscaling.
  //
  tempo_metrics_generator_scaled_object:
    if $._config.autoscaling.metrics_generator.enabled then
      $.scaledObjectForController($.tempo_metrics_generator_statefulset, $._config.autoscaling.metrics_generator)
      + scaledObject.spec.withTriggersMixin([{
        type: 'cpu',
        metricType: 'AverageValue',
        metadata: {
          value: $._config.autoscaling.metrics_generator.target_cpu,
        },
      }])
    else {},

  tempo_metrics_generator_statefulset+:
    if $._config.autoscaling.metrics_generator.enabled then $.removeReplicasFromSpec else {},

  //
  // Backend Worker: Prometheus-based autoscaling on outstanding blocks.
  //
  tempo_backend_worker_scaled_object:
    if $._config.autoscaling.backend_worker.enabled then
      assert $._config.autoscaling.prometheus_address != '' : 'autoscaling.prometheus_address is required for backend_worker autoscaling';
      $.scaledObjectForController($.tempo_backend_worker_statefulset, $._config.autoscaling.backend_worker)
      + scaledObject.spec.withTriggersMixin([{
        type: 'prometheus',
        metadata: {
          serverAddress: $._config.autoscaling.prometheus_address,
          metricName: 'tempodb_compaction_outstanding_blocks',
          query: $._config.autoscaling.backend_worker.query,
          threshold: '%d' % $._config.autoscaling.backend_worker.threshold,
        },
      }])
    else {},

  tempo_backend_worker_statefulset+:
    if $._config.autoscaling.backend_worker.enabled then $.removeReplicasFromSpec else {},

  //
  // Live Store: CPU-based autoscaling targeting the ReplicaTemplate for zone-aware scaling.
  //
  tempo_live_store_scaled_object:
    if $._config.autoscaling.live_store.enabled then
      $.scaledObjectForController($.tempo_live_store_replica_template, $._config.autoscaling.live_store)
      + scaledObject.spec.withTriggersMixin([{
        type: 'cpu',
        metricType: 'AverageValue',
        metadata: {
          value: '%s' % $._config.autoscaling.live_store.target_cpu,
        },
      }])
    else {},

  tempo_live_store_zone_a_statefulset+:
    if $._config.autoscaling.live_store.enabled then $.removeReplicasFromSpec else {},

  tempo_live_store_zone_b_statefulset+:
    if $._config.autoscaling.live_store.enabled then $.removeReplicasFromSpec else {},

  tempo_live_store_replica_template+:
    if $._config.autoscaling.live_store.enabled then $.removeReplicasFromSpec else {},

  //
  // Block Builder: KEDA kubernetes-workload scaler matching live-store zone-a pod count.
  //
  tempo_block_builder_scaled_object:
    if $._config.autoscaling.block_builder.enabled then
      $.scaledObjectForController($.tempo_block_builder_statefulset, $._config.autoscaling.block_builder)
      + scaledObject.spec.withTriggersMixin([{
        type: 'kubernetes-workload',
        metadata: {
          podSelector: $._config.autoscaling.block_builder.pod_selector,
          value: '%d' % $._config.autoscaling.block_builder.partitions_per_instance,
        },
      }])
    else {},

  tempo_block_builder_statefulset+:
    if $._config.autoscaling.block_builder.enabled then $.removeReplicasFromSpec else {},
}
