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

    live_store+: {
      keda: {
        enabled: false,
        min_replicas: 1,
        max_replicas: 200,
        paused_replicas: 0,
        // Time window (seconds) over which bytes written to Kafka are expected
        // to be held in the live-store. Should be >= complete_block_timeout (20m
        // default) plus any query_backend_after delay. Default: 30 minutes.
        window_seconds: 30 * 60,
        // Maximum bytes a single live-store pod is expected to hold over window_seconds.
        // Expressed as ingest_rate_per_pod × window_seconds so the two knobs stay coupled:
        //   default: 10 MB/s × 1800s ≈ 16 GiB per pod (observed production baseline)
        // To tune for your environment, change the rate multiplier:
        //   1 MB/s per pod  → 1 * 1000 * 1000 * self.window_seconds  (~1.7 GiB at 30m)
        //   5 MB/s per pod  → 5 * 1000 * 1000 * self.window_seconds  (~8.4 GiB at 30m)
        // Alternatively, set a literal byte value if you prefer to reason purely about
        // total bytes held per pod (e.g. bytes_per_replica: 17 * 1024 * 1024 * 1024).
        bytes_per_replica: 10 * 1000 * 1000 * self.window_seconds,
        // Prometheus query returning total bytes expected to be held across all
        // live-store pods. Measures the distributor (Kafka producer) side so that
        // draining pods — which have abandoned their partitions but are still up
        // for query — do not dilute the signal.
        // When empty (default) the ScaledObject uses:
        //   sum(rate(tempo_distributor_kafka_write_bytes_total{namespace="<ns>"}[<window_seconds>s])) * <window_seconds>
        // The rate window matches window_seconds so the averaged rate and the accumulation
        // window are always consistent. Set to a non-empty string to use a fully custom query.
        query: '',
        scale_up_stabilization_window_seconds: 60,
        scale_up_pods: 2,
        scale_up_period_seconds: 60,
        // Minimum duration of sustained low load before KEDA issues a scale-down.
        // Defaults to the live-store drain window (35m) so a load dip shorter than
        // one full drain cycle cannot trigger a scale-down. Note: the drain itself
        // runs after the signal, so total time from load drop to pod termination
        // is roughly 2x this value.
        scale_down_stabilization_window_seconds: 60 * 35,
        scale_down_pods: 1,
        scale_down_period_seconds: 60 * 5,
        // Extra KEDA trigger objects appended after the default Prometheus trigger.
        // Use this to inject environment-specific triggers (e.g. Kafka lag, CPU)
        // from private config without requiring an upstream Jsonnet change.
        // Each entry is a raw KEDA trigger object passed directly to withTriggersMixin.
        additional_triggers: [],
      },
    },

    // Number of Kafka partitions each block-builder pod is responsible for.
    // When block-builder KEDA is enabled, replicas are dynamic; this value must
    // stay coupled so the partition assignment math stays correct.
    // Default: 1 (each block-builder pod mirrors one live-store pod 1:1).
    block_builder+: {
      partitions_per_instance: 1,
      keda: {
        enabled: false,
        // 'rollout-operator' (default): rollout-operator mirrors live-store zone-a replicas
        //   directly to block-builder. Faster on both scale-up and scale-down.
        //   Requires live_store.keda.enabled=true.
        // 'keda': a dedicated KEDA ScaledObject uses a kubernetes-workload trigger counting
        //   live-store zone-a pods. Works with or without live-store KEDA.
        scaling: 'rollout-operator',
        min_replicas: 1,
        max_replicas: 200,
        paused_replicas: 0,
        // Pod selector to count live-store zone-a pods.
        // KEDA sets block-builder replicas = ceil(matching pods / partitions_per_instance).
        pod_selector: 'name=live-store-zone-a',
        // Block-builder must scale up as fast as live-store: new live-store pods create new
        // Kafka partitions that block-builders must immediately begin reading. Any delay on
        // scale-up means partition backlogs accumulate. Use 0s stabilization so KEDA acts on
        // the first observation (KEDA's ~30s polling interval is the practical floor).
        scale_up_stabilization_window_seconds: 0,
        scale_up_pods: 10,
        scale_up_period_seconds: 30,
        // Scale down quickly once live-store pods are gone: those partitions are reassigned
        // and block-builder no longer needs capacity for them.
        scale_down_stabilization_window_seconds: 60,
        scale_down_pods: 5,
        scale_down_period_seconds: 60,
      },
    },

    // Auto-derive: whenever live-store KEDA is enabled, rollout_operator_replica_template_access_enabled
    // must be true — rollout-operator uses the ReplicaTemplate to sync both zone StatefulSets regardless
    // of which block-builder scaling approach is active. The extra statefulsets/scale RBAC grant is
    // still gated to block_builder_scaling == 'rollout-operator' (see rollout_operator_role+ below).
    rollout_operator_replica_template_access_enabled:
      super.rollout_operator_replica_template_access_enabled || self.live_store.keda.enabled,

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

  // When block-builder KEDA is enabled, remove spec.replicas so the chosen scaling
  // mechanism (rollout-operator or KEDA ScaledObject) exclusively owns the replica count.
  tempo_block_builder_statefulset+:
    if $._config.block_builder.keda.enabled then
      assert std.member(['rollout-operator', 'keda'], $._config.block_builder.keda.scaling) :
             'block_builder.keda.scaling must be "rollout-operator" or "keda", got: ' + $._config.block_builder.keda.scaling;
      assert $._config.block_builder.keda.scaling != 'rollout-operator' || $._config.live_store.keda.enabled :
             'block_builder.keda.scaling="rollout-operator" requires live_store.keda.enabled=true';
      $.removeReplicasFromSpec
    else {},

  //
  // Block Builder: KEDA kubernetes-workload scaler counting live-store zone-a pods.
  // Used when block_builder.keda.enabled and block_builder.keda.scaling == 'keda'.
  // Works independently of live_store.keda.enabled.
  //
  tempo_block_builder_scaled_object:
    if $._config.block_builder.keda.enabled && $._config.block_builder.keda.scaling == 'keda' then
      assert $._config.block_builder.partitions_per_instance > 0 : 'block_builder.partitions_per_instance must be > 0';
      $.scaledObjectForController($.tempo_block_builder_statefulset, 'block_builder')
      + scaledObject.spec.withTriggersMixin([{
        type: 'kubernetes-workload',
        metadata: {
          podSelector: $._config.block_builder.keda.pod_selector,
          value: '%d' % $._config.block_builder.partitions_per_instance,
        },
      }])
    else {},

  //
  // Live Store: Prometheus-based autoscaling on expected bytes held.
  // Targets the ReplicaTemplate; the rollout-operator mirrors the replica count
  // to both zone StatefulSets, which must not carry their own spec.replicas.
  //
  tempo_live_store_scaled_object:
    if $._config.live_store.keda.enabled then
      assert $._config.autoscaling_prometheus_url != '' : 'autoscaling_prometheus_url is required for live_store autoscaling';
      local config = $._config.live_store.keda;
      local query = if config.query != '' then config.query else
        |||
          sum(rate(tempo_distributor_kafka_write_bytes_total{namespace="%(namespace)s"}[%(window_seconds)ds])) * %(window_seconds)d
        ||| % { namespace: $._config.namespace, window_seconds: config.window_seconds };
      $.scaledObjectForController($.tempo_live_store_replica_template, 'live_store')
      + scaledObject.spec.withTriggersMixin(
        [
          // metricType: Value → desiredReplicas = ceil(queryResult / threshold).
          // The query returns total expected bytes across all pods, so we must use
          // Value rather than the default AverageValue (which would multiply by
          // currentReplicas and cause runaway scaling).
          $.prometheusTrigger(
            query=query,
            metricName='tempo_live_store_expected_bytes',
            threshold='%d' % config.bytes_per_replica,
          ) + { metricType: 'Value' },
        ] + config.additional_triggers
      )
    else {},

  // When KEDA manages the live-store, drop spec.replicas from the ReplicaTemplate and
  // both zone StatefulSets so the ScaledObject exclusively owns the replica count and
  // Tanka apply does not fight it.
  tempo_live_store_replica_template+:
    if $._config.live_store.keda.enabled then $.removeReplicasFromSpec else {},

  tempo_live_store_zone_a_statefulset+:
    if $._config.live_store.keda.enabled then $.removeReplicasFromSpec else {},

  tempo_live_store_zone_b_statefulset+:
    if $._config.live_store.keda.enabled then $.removeReplicasFromSpec else {},

  // The rollout-operator mirror-replicas feature reads the source StatefulSet's scale
  // subresource to obtain the current replica count. The default rollout-operator role
  // grants apps/statefulsets (list/get/watch/patch) but not apps/statefulsets/scale.
  rollout_operator_role+:
    if $._config.block_builder.keda.enabled && $._config.block_builder.keda.scaling == 'rollout-operator' then
      local role = $.rbac.v1.role;
      local policyRule = $.rbac.v1.policyRule;
      role.withRulesMixin([
        policyRule.withApiGroups(['apps']) +
        policyRule.withResources(['statefulsets/scale']) +
        policyRule.withVerbs(['get']),
      ])
    else {},

}
