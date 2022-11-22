local mimir =
  (import 'github.com/grafana/mimir/operations/mimir/mimir.libsonnet')
  + {
    _config+:: {
      // Remove the alertmanagerStorageConfig from Mimir as this library provides its own config interface.
      alertmanagerStorageClientConfig:: {},
      alertmanager_enabled: true,
      // No resources should have a pre-configured namespace. These should be removed with removeNamespaceReferences.
      // TODO: Remove upstream.
      namespace:: 'namespace',
      // Remove the rulerClientConfig from Mimir as this library provides its own config interface.
      rulerClientConfig: {},
      ruler_enabled: true,
      // Enable memberlist ring
      memberlist_ring_enabled: true,
      // Enable mimir query sharding feature by default.
      query_sharding_enabled: true,
      // Enable query scheduler by default.
      query_scheduler_enabled: true,
    },
    alertmanager_args+:: {
      // Memberlist gossip is used instead of consul for the ring.
      'alertmanager.sharding-ring.consul.hostname':: null,
      'alertmanager.sharding-ring.store': 'memberlist',
    },
    compactor_args+:: {
      // Use default ring prefix
      'compactor.ring.prefix':: null,
    },
    distributor_args+:: {
      // Disable the ha-tracker by default.
      'distributor.ha-tracker.enable':: null,
      'distributor.ha-tracker.enable-for-all-users':: null,
      'distributor.ha-tracker.store':: null,
      'distributor.ha-tracker.etcd.endpoints':: null,
      'distributor.ha-tracker.prefix':: null,
    },
    ingester_args+:: {},
    querier_args+:: {
      // Use default ring prefix
      'store-gateway.sharding-ring.prefix':: null,
    },
    ruler_args+:: {
      // Remove ruler limits configuration.
      'ruler.max-rule-groups-per-tenant':: null,
      'ruler.max-rules-per-rule-group':: null,
      // Use default ring prefix
      'store-gateway.sharding-ring.prefix':: null,
    },
    store_gateway_args+:: {
      // Use default ring prefix
      'store-gateway.sharding-ring.prefix':: null,
    },
  };
local d = import 'github.com/jsonnet-libs/docsonnet/doc-util/main.libsonnet';
local k = import 'github.com/grafana/jsonnet-libs/ksonnet-util/kausal.libsonnet',
      configMap = k.core.v1.configMap,
      container = k.core.v1.container,
      deployment = k.apps.v1.deployment,
      job = k.batch.v1.job,
      policyRule = k.rbac.v1.policyRule,
      persistentVolumeClaim = k.core.v1.persistentVolumeClaim,
      role = k.rbac.v1.role,
      roleBinding = k.rbac.v1.roleBinding,
      subject = k.rbac.v1.subject,
      service = k.core.v1.service,
      serviceAccount = k.core.v1.serviceAccount,
      statefulSet = k.apps.v1.statefulSet;
local util = (import 'github.com/grafana/jsonnet-libs/ksonnet-util/util.libsonnet').withK(k);

// removeNamespaceReferences removes the cluster domain and namespace from container arguments.
local removeNamespaceReferences(args) = std.map(function(arg) std.strReplace(arg, '.namespace.svc.cluster.local', ''), args);


{
  '#':: d.pkg(
    name='enterprise-metrics',
    url='github.com/grafana/jsonnet-libs/enterprise-metrics/main.libsonnet',
    help='`enterprise-metrics` produces Kubernetes manifests for a Grafana Enterprise Metrics cluster.',
  ),
  local this = self,

  '#_config':: d.obj('`_config` is used for consumer overrides and configuration. Similar to a Helm values.yaml file'),
  _config:: {
    '#commonArgs':: d.obj('`commonArgs` is a convenience field that can be used to modify the container arguments of all modules as key-value pairs.'),
    commonArgs:: {
      'admin.client.backend-type': error 'you must set the `admin.client.backend-type` flag to an object storage backend ("gcs"|"s3")',
      'admin.client.gcs.bucket-name': 'admin',
      'admin.client.s3.bucket-name': 'admin',
      '#auth.multitenancy-enabled':: d.val(default=self['auth.multitenancy-enabled'], help='`auth.multitenancy-enabled` enables multitenancy', type=d.T.bool),
      'auth.multitenancy-enabled': true,
      '#auth.type':: d.val(
        default=self['auth.type'], help=|||
          `auth.type` configures the type of authentication in use.
          `enterprise` uses Grafana Enterprise token authentication.
          `default` uses Mimir authentication.
        |||, type=d.T.bool
      ),
      'auth.type': 'enterprise',
      'blocks-storage.backend': error 'you must set the `blocks-storage.backend` flag to an object storage backend ("gcs"|"s3")',
      'blocks-storage.gcs.bucket-name': 'tsdb',
      'blocks-storage.s3.bucket-name': 'tsdb',
      '#cluster-name':: d.val(
        default=self['cluster-name'],
        help='`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.',
        type=d.T.string
      ),
      'cluster-name': error 'you must set the `cluster-name` flag to the cluster name associated with your Grafana Enterprise Metrics license',
      '#memberlist.join':: d.val(
        default=self['memberlist.join'],
        help='`memberlist.join` is an address used to find memberlist peers for ring gossip',
        type=d.T.string
      ),
      'memberlist.join': 'gossip-ring',
      // Remove 'limits-per-user-override-config' flag and use 'runtime-config.file' instead.
      // TODO: remove this when upstream uses 'runtime-config.file'.
      'limits.per-user-override-config':: null,
      '#runtime-config.file':: d.val(
        help='`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.',
        type=d.T.string
      ),
      'runtime-config.file': '/etc/enterprise-metrics/runtime.yml',
      '#instrumentation.enabled':: d.val(
        default=self['instrumentation.enabled'],
        help='`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance',
        type=d.T.string
      ),
      'instrumentation.enabled': true,
      '#instrumentation.distributor-client.address':: d.val(
        default=self['instrumentation.distributor-client.address'],
        help='`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.',
        type=d.T.string
      ),
      'instrumentation.distributor-client.address': 'dns:///distributor:9095',
      '#license.path':: d.val(
        default=self['license.path'],
        help='`license.path` configures where this component expects to find a Grafana Enterprise Metrics License.',
        type=d.T.string
      ),
      'license.path': '/etc/gem-license/license.jwt',
    },

    '#licenseSecretName':: d.val(
      default=self.licenseSecretName,
      help=|||
        The admin-api expects a Grafana Enterprise Metrics license configured as 'license.jwt' in the
        Kubernetes Secret with `licenseSecretName`.
        To create the Kubernetes Secret from a local 'license.jwt' file:
        $ kubectl create secret generic gem-license -from-file=license.jwt
      |||,
      type=d.T.string,
    ),
    licenseSecretName: 'gem-license',
    '#adminTokenSecretName':: d.val(
      default=self.adminTokenSecretName,
      help=|||
        When generating an admin token using the `tokengen` target, the result is written to the Kubernetes
        Secret `adminTokenSecretName`.
        There are two versions of the token in the Secret: `token` is the raw token obtained from the `tokengen`,
        and `grafana-token` is a base64 encoded version that can be used directly when provisioning Grafana
        via configuration file.
        To retrieve these tokens from Kubernetes using `kubectl`:
        $ kubectl get secret gem-admin-token -o jsonpath="{.data.token}" | base64 --decode ; echo
        $ kubectl get secret gem-admin-token -o jsonpath="{.data.grafana-token}" | base64 --decode ; echo
      |||,
      type=d.T.string,
    ),
    adminTokenSecretName: 'gem-admin-token',
  },
  '#_images':: d.obj('`_images` contains fields for container images.'),
  _images:: {
    '#gem':: d.val(default=self.gem, help='`gem` is the Grafana Enterprise Metrics container image.', type=d.T.string),
    gem: 'grafana/metrics-enterprise:v2.0.1',
    '#kubectl':: d.val(default=self.kubectl, help='`kubectl` is the image used for kubectl containers.', type=d.T.string),
    kubectl: 'bitnami/kubectl',
  },

  '#adminApi':: d.obj('`adminApi` has configuration for the admin-api.'),
  adminApi: {
    '#args':: d.obj('`args` is a convenience field that can be used to modify the admin-api container arguments as key value pairs.'),
    args:: this._config.commonArgs {
      '#admin-api.leader-election.enabled':: d.val(
        default=self['admin-api.leader-election.enabled'],
        help='`admin-api.leader-election.enabled` enables leader election for to avoid inconsistent state with parallel writes when multiple replicas of the admin-api are running.',
        type=d.T.bool
      ),
      'admin-api.leader-election.enabled': true,
      '#admin-api.leader-election.ring.store':: d.val(
        default=self['admin-api.leader-election.ring.store'],
        help='`admin-api.leader-election.ring.store` is the type of key-value store to use for admin-api leader election.',
        type=d.T.string,
      ),
      'admin-api.leader-election.ring.store': 'memberlist',
      target: 'admin-api',
    },
    '#container':: d.obj('`container` is a convenience field that can be used to modify the admin-api container.'),
    container::
      container.new('admin-api', image=this._images.gem)
      + container.withArgs(util.mapToFlags(self.args))
      + container.withPorts(mimir.util.defaultPorts)
      + container.resources.withLimits({ cpu: '2', memory: '4Gi' })
      + container.resources.withRequests({ cpu: '500m', memory: '1Gi' })
      + mimir.util.readinessProbe,
    '#deployment':: d.obj('`deployment` is the Kubernetes Deployment for the admin-api.'),
    deployment:
      deployment.new(name='admin-api', replicas=1, containers=[self.container])
      + deployment.spec.selector.withMatchLabelsMixin({ name: 'admin-api' })
      + deployment.spec.template.metadata.withLabelsMixin({ name: 'admin-api', gossip_ring_member: 'true' })
      + deployment.spec.template.spec.withTerminationGracePeriodSeconds(15)
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the admin-api.'),
    service:
      util.serviceFor(self.deployment),
  },

  '#alertmanager':: d.obj('`alertmanager` has configuration for the alertmanager. To disable the alertmanager, ensure the alertmanager object field is hidden'),
  alertmanager: {
    local alertmanager = self,
    '#args':: d.obj('`args` is a convenience field that can be used to modify the alertmanager container arguments as key-value pairs.'),
    args:: mimir.alertmanager_args + this._config.commonArgs + {
      'alertmanager-storage.backend':
        if 'alertmanager-storage.backend' in super then
          super['alertmanager-storage.backend']
        else error 'you must set the `alertmanager-storage.backend` flag to an object storage backend ("azure"|"local"|"gcs"|"s3")',
      '#alertmanager-storage.s3.bucket-name':: d.val(
        default=self['alertmanager-storage.s3.bucket-name'],
        help='`alertmanager-storage.s3.bucket-name` is the name of the bucket in which the alertmanager data will be stored.',
        type=d.T.string
      ),
      'alertmanager-storage.s3.bucket-name': 'alertmanager',
      'alertmanager.web.external-url': error 'you must set the `alertmanager.web.external-url` flag in order to run an Alertmanager.',
    },
    '#container':: d.obj('`container` is a convenience field that can be used to modify the alertmanager container.'),
    container::
      mimir.alertmanager_container
      + container.withArgs(removeNamespaceReferences(util.mapToFlags(alertmanager.args)))
      + container.withImage(this._images.gem),
    '#persistentVolumeClaim':: d.obj('`persistentVolumeClaim` is a convenience field that can be used to modify the alertmanager PersistentVolumeClaim.'),
    persistentVolumeClaim:: mimir.alertmanager_pvc,
    '#statefulSet':: d.obj('`statefulSet` is the Kubernetes StatefulSet for the alertmanager.'),
    statefulSet:
      mimir.alertmanager_statefulset { metadata+: { namespace:: null } }  // Hide the metadata.namespace field as Tanka provides that.
      + statefulSet.spec.selector.withMatchLabelsMixin({ name: 'alertmanager' })
      + statefulSet.spec.withVolumeClaimTemplates([self.persistentVolumeClaim])
      + statefulSet.spec.template.metadata.withLabelsMixin({ name: 'alertmanager', gossip_ring_member: 'true' })
      + statefulSet.spec.template.spec.withContainers([alertmanager.container])
      // Remove Mimir volumes.
      + statefulSet.spec.template.spec.withVolumes([])
      // Replace with GEM volumes.
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the alertmanager.'),
    service: util.serviceFor(self.statefulSet),
  },

  '#compactor':: d.obj('`compactor` has configuration for the compactor.'),
  compactor: {
    local compactor = self,
    '#args':: d.obj('`args` is a convenience field that can be used to modify the compactor container arguments as key-value pairs.'),
    args:: mimir.compactor_args + this._config.commonArgs,
    '#container':: d.obj('`container` is a convenience field that can be used to modify the compactor container.'),
    container::
      mimir.compactor_container
      + container.withArgs(removeNamespaceReferences(util.mapToFlags(compactor.args)))
      + container.withImage(this._images.gem),
    '#persistentVolumeClaim':: d.obj('`persistentVolumeClaim` is a convenience field that can be used to modify the compactor PersistentVolumeClaim.'),
    persistentVolumeClaim::
      persistentVolumeClaim.new()
      + persistentVolumeClaim.mixin.spec.resources.withRequests({ storage: '250Gi' })
      + persistentVolumeClaim.mixin.spec.withAccessModes(['ReadWriteOnce'])
      + persistentVolumeClaim.mixin.metadata.withName('compactor-data'),
    '#statefulSet':: d.obj('`statefulSet` is the Kubernetes StatefulSet for the compactor.'),
    statefulSet:
      mimir.compactor_statefulset { metadata+: { namespace:: null } }  // Hide the metadata.namespace field as Tanka provides that.
      + statefulSet.spec.withVolumeClaimTemplates([self.persistentVolumeClaim])
      + statefulSet.spec.selector.withMatchLabelsMixin({ name: 'compactor' })
      + statefulSet.spec.template.metadata.withLabelsMixin({ name: 'compactor', gossip_ring_member: 'true' })
      + statefulSet.spec.template.spec.withContainers([compactor.container])
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the compactor.'),
    service: util.serviceFor(self.statefulSet),
  },

  '#distributor':: d.obj('`distributor` has configuration for the distributor.'),
  distributor: {
    local distributor = self,
    '#args':: d.obj('`args` is a convenience field that can be used to modify the distributor container arguments as key-value pairs.'),
    args:: mimir.distributor_args + this._config.commonArgs,
    '#container':: d.obj('`container` is a convenience field that can be used to modify the distributor container.'),
    container::
      mimir.distributor_container
      + container.withArgs(removeNamespaceReferences(util.mapToFlags(distributor.args)))
      + container.withImage(this._images.gem),
    '#statefulSet':: d.obj('`deployment` is the Kubernetes Deployment for the distributor.'),
    deployment:
      mimir.distributor_deployment
      + deployment.spec.selector.withMatchLabelsMixin({ name: 'distributor', gossip_ring_member: 'true' })
      + deployment.spec.template.metadata.withLabelsMixin({ name: 'distributor', gossip_ring_member: 'true' })
      + deployment.spec.template.spec.withContainers([distributor.container])
      // Remove Mimir volumes.
      + deployment.spec.template.spec.withVolumes([])
      // Replace with GEM volumes.
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the distributor.'),
    service:
      util.serviceFor(self.deployment)
      + service.spec.withClusterIp('None'),
  },

  '#gateway':: d.obj('`gateway` has configuration for the gateway.'),
  gateway: {
    '#args':: d.obj('`args` is a convenience field that can be used to modify the gateway container arguments as key-value pairs.'),
    args:: this._config.commonArgs {
      '#gateway.proxy.admin-api.url':: d.val(
        default=self['gateway.proxy.admin-api.url'], help='`gateway.proxy.admin-api.url is the upstream URL of the admin-api.', type=d.T.string
      ),
      'gateway.proxy.admin-api.url': 'http://admin-api:8080',
      '#gateway.proxy.compactor.url':: d.val(
        default=self['gateway.proxy.compactor.url'], help='`gateway.proxy.compactor.url is the upstream URL of the compactor.', type=d.T.string
      ),
      'gateway.proxy.compactor.url': 'http://compactor:8080',
      '#gateway.proxy.distributor.url':: d.val(
        default=self['gateway.proxy.distributor.url'], help='`gateway.proxy.distributor.url is the upstream URL of the distributor.', type=d.T.string
      ),
      '#gateway.proxy.alertmanager.url':: d.val(
        default=self['gateway.proxy.alertmanager.url'], help='`gateway.proxy.alertmanager.url is the upstream URL of the alertmanager.', type=d.T.string
      ),
      'gateway.proxy.alertmanager.url': 'http://alertmanager:8080',
      'gateway.proxy.distributor.url': 'dns:///distributor:9095',
      '#gateway.proxy.ingester.url':: d.val(
        default=self['gateway.proxy.ingester.url'], help='`gateway.proxy.ingester.url is the upstream URL of the ingester.', type=d.T.string
      ),
      'gateway.proxy.ingester.url': 'http://ingester:8080',
      '#gateway.proxy.ruler.url':: d.val(
        default=self['gateway.proxy.ruler.url'], help='`gateway.proxy.ruler.url is the upstream URL of the ruler.', type=d.T.string
      ),
      'gateway.proxy.ruler.url': 'http://ruler:8080',
      '#gateway.proxy.query-frontend.url':: d.val(
        default=self['gateway.proxy.query-frontend.url'], help='`gateway.proxy.query-frontend.url is the upstream URL of the query-frontend.', type=d.T.string
      ),
      'gateway.proxy.query-frontend.url': 'http://query-frontend:8080',
      '#gateway.proxy.store-gateway.url':: d.val(
        default=self['gateway.proxy.store-gateway.url'], help='`gateway.proxy.store-gateway.url is the upstream URL of the store-gateway.', type=d.T.string
      ),
      'gateway.proxy.store-gateway.url': 'http://store-gateway:8080',
      target: 'gateway',
    },

    '#container':: d.obj('`container` is a convenience field that can be used to modify the gateway container.'),
    container::
      container.new('gateway', this._images.gem)
      + container.withArgs(util.mapToFlags(self.args))
      + container.withPorts(mimir.util.defaultPorts)
      + container.resources.withLimits({ cpu: '2', memory: '4Gi' })
      + container.resources.withRequests({ cpu: '500m', memory: '1Gi' })
      + mimir.util.readinessProbe,
    '#deployment':: d.obj('`deployment` is the Kubernetes Deployment for the gateway.'),
    deployment:
      deployment.new('gateway', 3, [self.container]) +
      deployment.spec.template.spec.withTerminationGracePeriodSeconds(15)
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the gateway.'),
    service:
      util.serviceFor(self.deployment),
  },

  '#gossipRing':: d.obj('`gossipRing` is used by microservices to discover other memberlist members.'),
  gossipRing: {
    '#service':: d.obj('`service` is the Kubernetes Service for the gossip ring.'),
    service:
      mimir.gossip_ring_service
      // Publish not ready addresses for initial memberlist start up.
      + service.spec.withPublishNotReadyAddresses(true),
  },

  '#ingester':: d.obj('`ingester` has configuration for the ingester.'),
  ingester: {
    local ingester = self,
    '#args':: d.obj('`args` is a convenience field that can be used to modify the ingester container arguments as key-value pairs.'),
    args:: mimir.ingester_args + this._config.commonArgs,
    '#container':: d.obj('`container` is a convenience field that can be used to modify the ingester container.'),
    container::
      mimir.ingester_container
      + container.withArgs(removeNamespaceReferences(util.mapToFlags(ingester.args)))
      + container.withImage(this._images.gem)
      + container.withVolumeMounts([{ name: 'ingester-data', mountPath: '/data' }]),
    '#persistentVolumeClaim':: d.obj('`persistentVolumeClaim` is a convenience field that can be used to modify the ingester PersistentVolumeClaim. It is recommended to use a fast storage class.'),
    persistentVolumeClaim::
      persistentVolumeClaim.new()
      + persistentVolumeClaim.mixin.spec.resources.withRequests({ storage: '100Gi' })
      + persistentVolumeClaim.mixin.spec.withAccessModes(['ReadWriteOnce'])
      + persistentVolumeClaim.mixin.metadata.withName('ingester-data'),
    '#podDisruptionBudget':: d.obj('`podDisruptionBudget` is the Kubernetes PodDisruptionBudget for the ingester.'),
    podDisruptionBudget: mimir.ingester_pdb,
    '#statefulSet':: d.obj('`statefulSet` is the Kubernetes StatefulSet for the ingester.'),
    statefulSet:
      mimir.ingester_statefulset { metadata+: { namespace:: null } }  // Hide the metadata.namespace field as Tanka provides that.
      + statefulSet.spec.withVolumeClaimTemplates([self.persistentVolumeClaim])
      + statefulSet.spec.selector.withMatchLabelsMixin({ name: 'ingester' })
      + statefulSet.spec.template.metadata.withLabelsMixin({ name: 'ingester', gossip_ring_member: 'true' })
      + statefulSet.spec.template.spec.withContainers([ingester.container])
      // Remove Mimir volumes.
      + statefulSet.spec.template.spec.withVolumes([])
      // Replace with GEM volumes.
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the ingester.'),
    service: util.serviceFor(self.statefulSet),
  },

  '#memcached':: d.obj('`memcached` has configuration for GEM caches.'),
  memcached: {
    '#frontend':: d.obj('`frontend` is a cache for query-frontend query results.'),
    frontend: mimir.memcached_frontend,
    '#chunks':: d.obj('`chunks` is a cache for time series chunks.'),
    chunks: mimir.memcached_chunks,
    '#metadata':: d.obj('`metadata` is cache for object store metadata used by the queriers and store-gateways.'),
    metadata: mimir.memcached_metadata,
    '#queries':: d.obj('`queries` is a cache for index queries used by the store-gateways.'),
    queries: mimir.memcached_index_queries,
  },

  '#querier':: d.obj('`querier` has configuration for the querier.'),
  querier: {
    local querier = self,
    '#args':: d.obj('`args` is a convenience field that can be used to modify the querier container arguments as key-value pairs.'),
    args:: mimir.querier_args + this._config.commonArgs,
    '#container':: d.obj('`container` is a convenience field that can be used to modify the querier container.'),
    container::
      mimir.querier_container
      + container.withArgs(removeNamespaceReferences(util.mapToFlags(querier.args)))
      + container.withImage(this._images.gem),
    '#deployment':: d.obj('`deployment` is the Kubernetes Deployment for the querier.'),
    deployment:
      mimir.querier_deployment
      + deployment.spec.selector.withMatchLabelsMixin({ name: 'querier', gossip_ring_member: 'true' })
      + deployment.spec.template.metadata.withLabelsMixin({ name: 'querier', gossip_ring_member: 'true' })
      + deployment.spec.template.spec.withContainers([querier.container])
      // Remove Mimir volumes.
      + deployment.spec.template.spec.withVolumes([])
      // Replace with GEM volumes.
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the querier.'),
    service: util.serviceFor(self.deployment),
  },

  '#overridesExporter':: d.obj('`overridesExporter` has configuration for the overrides-exporter.'),
  overridesExporter: {
    '#args':: d.obj('`args` is a convenience field that can be used to modify the overrides-exporter container arguments as key value pairs.'),
    args:: this._config.commonArgs {
      target: 'overrides-exporter',
    },
    '#container':: d.obj('`container` is a convenience field that can be used to modify the overrides-exporter container.'),
    container::
      container.new('overrides-exporter', image=this._images.gem)
      + container.withArgs(util.mapToFlags(self.args))
      + container.withPorts(mimir.util.defaultPorts)
      + container.resources.withLimits({ cpu: '1', memory: '1Gi' })
      + container.resources.withRequests({ cpu: '100m', memory: '1Gi' })
      + mimir.util.readinessProbe,
    '#deployment':: d.obj('`deployment` is the Kubernetes Deployment for the overrides-exporter.'),
    deployment:
      deployment.new(name='overrides-exporter', replicas=1, containers=[self.container])
      + deployment.spec.selector.withMatchLabelsMixin({ name: 'overrides-exporter' })
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the overrides-exporter.'),
    service:
      util.serviceFor(self.deployment),
  },

  '#queryFrontend':: d.obj('`queryFrontend` has configuration for the query-frontend.'),
  queryFrontend: {
    local queryFrontend = self,
    '#args':: d.obj('`args` is a convenience field that can be used to modify the query-frontend container arguments as key-value pairs.'),
    args:: mimir.query_frontend_args + this._config.commonArgs,
    '#container':: d.obj('`container` is a convenience field that can be used to modify the query-frontend container.'),
    container::
      mimir.query_frontend_container
      + container.withArgs(removeNamespaceReferences(util.mapToFlags(queryFrontend.args)))
      + container.withImage(this._images.gem),
    '#deployment':: d.obj('`deployment` is the Kubernetes Deployment for the query-frontend.'),
    deployment:
      mimir.query_frontend_deployment
      + deployment.spec.template.spec.withContainers([queryFrontend.container])
      // Remove Mimir volumes.
      + deployment.spec.template.spec.withVolumes([])
      // Replace with GEM volumes.
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the query-frontend.'),
    service: util.serviceFor(self.deployment),
    '#discoveryService':: d.obj('`discoveryService` is a headless Kubernetes Service used by queriers to discover query-frontend addresses.'),
    discoveryService:
      mimir.query_frontend_discovery_service,
  },

  '#queryScheduler':: d.obj('`queryScheduler` has configuration for the query-scheduler.'),
  queryScheduler: {
    local queryScheduler = self,
    '#args':: d.obj('`args` is a convenience field that can be used to modify the query-scheduler container arguments as key-value pairs.'),
    args:: mimir.query_scheduler_args + this._config.commonArgs,
    '#container':: d.obj('`container` is a convenience field that can be used to modify the query-scheduler container.'),
    container::
      mimir.query_scheduler_container
      + container.withArgs(removeNamespaceReferences(util.mapToFlags(queryScheduler.args)))
      + container.withImage(this._images.gem),
    '#deployment':: d.obj('`deployment` is the Kubernetes Deployment for the query-scheduler.'),
    deployment:
      mimir.query_scheduler_deployment
      + deployment.spec.template.spec.withContainers([queryScheduler.container])
      // Remove Mimir volumes.
      + deployment.spec.template.spec.withVolumes([])
      // Replace with GEM volumes.
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the query-scheduler.'),
    service: util.serviceFor(self.deployment),
    '#discoveryService':: d.obj('`discoveryService` is a headless Kubernetes Service used by queriers to discover query-scheduler addresses.'),
    discoveryService:
      mimir.query_scheduler_discovery_service,
  },

  '#ruler':: d.obj('`ruler` has configuration for the ruler.'),
  ruler: {
    local ruler = self,
    '#args':: d.obj('`args` is a convenience field that can be used to modify the ruler container arguments as key-value pairs.'),
    args:: mimir.ruler_args + this._config.commonArgs + {
      'ruler-storage.backend':
        if 'ruler-storage.backend' in super then
          super['ruler.storage.backend'] else
          error 'you must set the `ruler-storage.backend` flag to an object storage backend ("azure"|"local"|"gcs"|"s3")',
      '#ruler-storage.s3.bucket-name':: d.val(
        default=self['ruler-storage.s3.bucket-name'],
        help='`ruler-storage.s3.bucket-name` is the name of the bucket in which the ruler data will be stored.',
        type=d.T.string
      ),
      'ruler-storage.s3.bucket-name': 'ruler',
    },
    '#container':: d.obj('`container` is a convenience field that can be used to modify the ruler container.'),
    container::
      mimir.ruler_container
      + container.withArgs(removeNamespaceReferences(util.mapToFlags(ruler.args)))
      + container.withImage(this._images.gem),
    '#deployment':: d.obj('`deployment` is the Kubernetes Deployment for the ruler.'),
    deployment:
      mimir.ruler_deployment
      + deployment.spec.selector.withMatchLabelsMixin({ name: 'ruler' })
      + deployment.spec.template.metadata.withLabelsMixin({ name: 'ruler', gossip_ring_member: 'true' })
      + deployment.spec.template.spec.withContainers([ruler.container])
      // Remove Mimir volumes.
      + deployment.spec.template.spec.withVolumes([])
      // Replace with GEM volumes.
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the ruler.'),
    service: util.serviceFor(self.deployment),
  },

  '#runtime':: d.obj('`runtime` has configuration for runtime overrides.'),
  runtime: {
    local runtime = self,
    '#config':: d.obj('`config` is a convenience field for modifying the runtime configuration.'),
    configuration:: {
      '#overrides':: d.obj(|||
        `overrides` are per tenant runtime limits overrides.
        Each field should be keyed by tenant ID and have an object value containing the specific overrides.
        For example:
        {
          tenantId: {
            max_global_series_per_user: 150000,
            max_global_series_per_metric: 20000,
            ingestion_rate: 10000,
            ingestion_burst_size: 200000,
            ruler_max_rules_per_rule_group: 20,
            ruler_max_rule_groups_per_tenant: 35,
            compactor_blocks_retention_period: '0',
          },
        }
      |||),
      overrides: mimir._config.overrides,
    },
    '#configMap':: d.obj('`configMap` is the Kubernetes ConfigMap containing the runtime configuration.'),
    configMap: configMap.new('runtime', { 'runtime.yml': std.manifestYamlDoc(runtime.configuration) }),
  },

  '#storeGateway':: d.obj('`storeGateway` has configuration for the store-gateway.'),
  storeGateway: {
    local storeGateway = self,
    '#args':: d.obj('`args` is a convenience field that can be used to modify the store-gateway container arguments as key-value pairs.'),
    args:: mimir.store_gateway_args + this._config.commonArgs,
    '#container':: d.obj('`container` is a convenience field that can be used to modify the store-gateway container.'),
    container::
      mimir.store_gateway_container
      + container.withArgs(removeNamespaceReferences(util.mapToFlags(storeGateway.args)))
      + container.withImage(this._images.gem),
    '#persistentVolumeClaim':: d.obj('`persistentVolumeClaim` is a convenience field that can be used to modify the store-gateway PersistentVolumeClaim.'),
    persistentVolumeClaim::
      persistentVolumeClaim.new()
      + persistentVolumeClaim.mixin.spec.resources.withRequests({ storage: '100Gi' })
      + persistentVolumeClaim.mixin.spec.withAccessModes(['ReadWriteOnce'])
      + persistentVolumeClaim.mixin.metadata.withName('store-gateway-data'),
    '#podDisruptionBudget':: d.obj('`podDisruptionBudget` is the Kubernetes PodDisruptionBudget for the store-gateway.'),
    podDisruptionBudget: mimir.store_gateway_pdb,
    '#statefulSet':: d.obj('`statefulSet` is the Kubernetes StatefulSet for the store-gateway.'),
    statefulSet:
      mimir.store_gateway_statefulset { metadata+: { namespace:: null } }  // Hide the metadata.namespace field as Tanka provides that.
      + statefulSet.spec.withVolumeClaimTemplates([self.persistentVolumeClaim])
      + statefulSet.spec.selector.withMatchLabelsMixin({ name: 'store-gateway' })
      + statefulSet.spec.template.metadata.withLabelsMixin({ name: 'store-gateway', gossip_ring_member: 'true' })
      + statefulSet.spec.template.spec.withContainers([storeGateway.container])
      // Remove Cortex volumes.
      + statefulSet.spec.template.spec.withVolumes([])
      // Replace with GEM volumes.
      + util.configVolumeMount('runtime', '/etc/enterprise-metrics')
      + if this._config.licenseSecretName != null then util.secretVolumeMount(this._config.licenseSecretName, '/etc/gem-license/') else {},
    '#service':: d.obj('`service` is the Kubernetes Service for the store-gateway.'),
    service: util.serviceFor(self.statefulSet),
  },

  '#tokengen':: d.obj(|||
    `tokengen` has configuration for tokengen.
    By default the tokengen object is hidden as it is a one-off task. To deploy the tokengen job, unhide the tokengen object field.
  |||),
  tokengen:: {
    local target = 'tokengen',
    '#args':: d.obj('`args` is convenience field for modifying the tokegen container arguments as key-value pairs.'),
    args:: this._config.commonArgs {
      target: target,
      'tokengen.token-file': '/shared/admin-token',
    },
    '#container':: d.obj(|||
      `container` is a convenience field for modifying the tokengen container.
      By default, the container runs GEM with the tokengen target and writes the token to a file.
    |||),
    container::
      container.new(target, this._images.gem)
      + container.withPorts(mimir.util.defaultPorts)
      + container.withArgs(util.mapToFlags(self.args))
      + container.withVolumeMounts([{ mountPath: '/shared', name: 'shared' }])
      + container.resources.withLimits({ memory: '4Gi' })
      + container.resources.withRequests({ cpu: '500m', memory: '500Mi' }),

    '#createSecretContainer':: d.obj('`createSecretContainer` creates a Kubernetes Secret from a token file.'),
    createSecretContainer::
      container.new('create-secret', this._images.kubectl)
      + container.withCommand([
        '/bin/bash',
        '-euc',
        'kubectl create secret generic %s --from-file=token=/shared/admin-token --from-literal=grafana-token="$(base64 <(echo :$(cat /shared/admin-token)))"' % this._config.adminTokenSecretName,
      ])
      + container.withVolumeMounts([{ mountPath: '/shared', name: 'shared' }])
      // Need to run as root because the GEM container does.
      + container.securityContext.withRunAsUser(0),

    '#job':: d.obj('`job` is the Kubernetes Job for tokengen'),
    job:
      job.new(target)
      + job.spec.withCompletions(1)
      + job.spec.withParallelism(1)
      + job.spec.template.spec.withContainers([self.createSecretContainer])
      + job.spec.template.spec.withInitContainers([self.container])
      + job.spec.template.spec.withRestartPolicy('OnFailure')
      + job.spec.template.spec.withServiceAccount(target)
      + job.spec.template.spec.withServiceAccountName(target)
      + job.spec.template.spec.withVolumes([{ name: 'shared', emptyDir: {} }]),

    '#serviceAccount':: d.obj('`serviceAccount` is the Kubernetes ServiceAccount for tokengen'),
    serviceAccount:
      serviceAccount.new(target),

    '#role':: d.obj('`role` is the Kubernetes Role for tokengen'),
    role:
      role.new(target)
      + role.withRules([
        policyRule.withApiGroups([''])
        + policyRule.withResources(['secrets'])
        + policyRule.withVerbs(['create']),
      ]),

    '#roleBinding':: d.obj('`roleBinding` is the Kubernetes RoleBinding for tokengen'),
    roleBinding:
      roleBinding.new()
      + roleBinding.metadata.withName(target)
      + roleBinding.roleRef.withApiGroup('rbac.authorization.k8s.io')
      + roleBinding.roleRef.withKind('Role')
      + roleBinding.roleRef.withName(target)
      + roleBinding.withSubjects([
        subject.new()
        + subject.withName(target)
        + subject.withKind('ServiceAccount'),
      ]),
  },

  '#util':: d.obj('`util` contains utility functions for working with the GEM Jsonnet library'),
  util:: {
    '#util':: d.val(d.T.array, '`modules` is an array of the names of all modules in the cluster'),
    modules:: ['adminApi', 'alertmanager', 'compactor', 'distributor', 'gateway', 'ingester', 'querier', 'queryFrontend', 'queryScheduler', 'ruler', 'storeGateway'],
    '#mapModules': d.fn('`mapModules` applies the function fn to each module in the GEM cluster', [d.arg('fn', d.T.func)]),
    mapModules(fn=function(module) module)::
      local activeModules = std.filter(function(field) std.member(self.modules, field), std.objectFields($));
      std.foldl(function(acc, module) acc { [module]: fn(super[module]) }, activeModules, {}),
  },
}
