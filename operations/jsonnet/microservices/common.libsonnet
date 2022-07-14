{
  util+:: {
    local k = import 'ksonnet-util/kausal.libsonnet',
    local container = k.core.v1.container,

    withResources(resources)::
      k.util.resourcesRequests(resources.requests.cpu, resources.requests.memory) +
      k.util.resourcesLimits(resources.limits.cpu, resources.limits.memory),

    readinessProbe::
      container.mixin.readinessProbe.httpGet.withPath('/ready') +
      container.mixin.readinessProbe.httpGet.withPort($._config.port) +
      container.mixin.readinessProbe.withInitialDelaySeconds(15) +
      container.mixin.readinessProbe.withTimeoutSeconds(1),
  },
}
