local default = import 'environments/default/main.jsonnet';
local test = import 'testonnet/main.libsonnet';

test.new(std.thisFile)

// Distributor ScaledObject is created with correct spec (enabled in default env).
+ test.case.new(
  'Distributor ScaledObject',
  test.expect.eq(
    default.tempo_distributor_scaled_object,
    {
      apiVersion: 'keda.sh/v1alpha1',
      kind: 'ScaledObject',
      metadata: {
        name: 'distributor',
      },
      spec: {
        advanced: {
          horizontalPodAutoscalerConfig: {
            behavior: {
              scaleDown: {
                stabilizationWindowSeconds: 1800,
              },
              scaleUp: {
                policies: [{
                  periodSeconds: 15,
                  type: 'Pods',
                  value: 5,
                }],
                stabilizationWindowSeconds: 15,
              },
            },
          },
        },
        initialCooldownPeriod: 90,
        maxReplicaCount: 200,
        minReplicaCount: 2,
        pollingInterval: 60,
        scaleTargetRef: {
          apiVersion: 'apps/v1',
          kind: 'Deployment',
          name: 'distributor',
        },
        triggers: [{
          metadata: {
            value: '330m',
          },
          metricType: 'AverageValue',
          type: 'cpu',
        }],
      },
    },
  )
)

// Distributor deployment has replicas removed when autoscaling is enabled.
+ test.case.new(
  'Distributor replicas removed',
  test.expect.eq(
    std.objectHas(default.tempo_distributor_deployment.spec, 'replicas'),
    false,
  )
)

// ScaledObjects are empty when autoscaling is disabled.
+ test.case.new(
  'ScaledObjects empty when disabled',
  test.expect.eq(
    default.tempo_backend_worker_scaled_object,
    {},
  )
)
