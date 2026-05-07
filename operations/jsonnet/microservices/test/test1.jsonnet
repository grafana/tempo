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
