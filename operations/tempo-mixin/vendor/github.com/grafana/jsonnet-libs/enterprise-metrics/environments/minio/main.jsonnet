local gem = import '../../main.libsonnet';
local k = import 'github.com/grafana/jsonnet-libs/ksonnet-util/kausal.libsonnet',
      container = k.core.v1.container,
      containerPort = k.core.v1.containerPort,
      persistentVolumeClaim = k.core.v1.persistentVolumeClaim,
      service = k.core.v1.service,
      statefulSet = k.apps.v1.statefulSet;
local util = (import 'github.com/grafana/jsonnet-libs/ksonnet-util/util.libsonnet').withK(k);


function(kubeconfig) {
  apiVersion: 'tanka.dev/v1alpha1',
  kind: 'Environment',
  metadata: {
    name: 'environments/minio',
  },
  spec: {
    apiServer: std.filter(function(cluster) cluster.name == 'k3d-enterprise-metrics', kubeconfig.clusters)[0].cluster.server,
    namespace: 'default',
  },
  data: {
    local data = self,
    local adminBucket = 'admin',
    local alertmanagerBucket = 'alertmanager',
    local rulerBucket = 'ruler',
    local tsdbBucket = 'tsdb',
    local minioAccessKey = 'gem',
    local minioPort = 9000,
    local minioSecretKey = 'supersecret',

    minio: {
      local buckets = [adminBucket, rulerBucket, tsdbBucket, alertmanagerBucket],

      container::
        container.new('minio', 'minio/minio')
        + container.withCommand(
          ['/bin/sh', '-euc', std.join(' ', ['mkdir', '-p'] + ['/data/' + bucket for bucket in buckets] + ['&&', 'minio', 'server', '/data'])]
        )
        + container.withEnv([
          { name: 'MINIO_ACCESS_KEY', value: minioAccessKey },
          { name: 'MINIO_PROMETHEUS_AUTH_TYPE', value: 'public' },
          { name: 'MINIO_SECRET_KEY', value: minioSecretKey },
        ])
        + container.withImagePullPolicy('Always')
        + container.withPorts(containerPort.new('http-metrics', minioPort))
        + container.withVolumeMounts([{ name: 'minio-data', mountPath: '/data' }]),


      persistentVolumeClaim::
        persistentVolumeClaim.new('minio-data')
        + persistentVolumeClaim.spec.resources.withRequests({ storage: '1Gi' })
        + persistentVolumeClaim.spec.withAccessModes(['ReadWriteOnce']),

      // TODO: even though this is a StatefulSet, minio is not configured to run in a distributed mode.
      // You cannot yet scale the minio StatefulSet to increase storage or compute.
      statefulSet:
        statefulSet.new('minio', 1, [self.container], self.persistentVolumeClaim)
        + statefulSet.spec.withServiceName(self.service.metadata.name)
        + statefulSet.spec.template.metadata.withAnnotationsMixin({ 'prometheus.io/path': '/minio/v2/metrics/cluster' }),

      service:
        util.serviceFor(self.statefulSet),
    },
    gem: gem {
           _config+:: {
             commonArgs+: {
               'admin.client.backend-type': 's3',
               'admin.client.s3.access-key-id': minioAccessKey,
               'admin.client.s3.bucket-name': adminBucket,
               'admin.client.s3.endpoint': data.minio.service.metadata.name + ':' + minioPort,
               'admin.client.s3.insecure': 'true',
               'admin.client.s3.secret-access-key': minioSecretKey,
               'auth.type': 'trust',
               'blocks-storage.backend': 's3',
               'blocks-storage.s3.access-key-id': minioAccessKey,
               'blocks-storage.s3.bucket-name': tsdbBucket,
               'blocks-storage.s3.endpoint': data.minio.service.metadata.name + ':' + minioPort,
               'blocks-storage.s3.insecure': 'true',
               'blocks-storage.s3.secret-access-key': minioSecretKey,
               'cluster-name': 'minio',
             },
             licenseSecretName:: null,
           },
           alertmanager+: {
             args+:: {
               'alertmanager-storage.backend': 's3',
               'alertmanager-storage.s3.access-key-id': minioAccessKey,
               'alertmanager-storage.s3.bucket-name': alertmanagerBucket,
               'alertmanager-storage.s3.endpoint': data.minio.service.metadata.name + ':' + minioPort,
               'alertmanager-storage.s3.insecure': 'true',
               'alertmanager-storage.s3.secret-access-key': minioSecretKey,
               'alertmanager.web.external-url': 'test',
             },
           },

           memcached+: {
             frontend+: { statefulSet+: { spec+: { replicas: 1 } } },
             chunks+: { statefulSet+: { spec+: { replicas: 1 } } },
             metadata+: { statefulSet+: { spec+: { replicas: 1 } } },
             queries+: { statefulSet+: { spec+: { replicas: 1 } } },
           },

           ruler+: {
             args+:: {
               'ruler-storage.backend': 's3',
               'ruler-storage.s3.access-key-id': minioAccessKey,
               'ruler-storage.s3.bucket-name': rulerBucket,
               'ruler-storage.s3.endpoint': data.minio.service.metadata.name + ':' + minioPort,
               'ruler-storage.s3.insecure': 'true',
               'ruler-storage.s3.secret-access-key': minioSecretKey,
             },
           },
         }
         + gem.util.mapModules(
           function(module)
             local app = if std.objectHas(module, 'deployment') then 'deployment'
             else if std.objectHas(module, 'statefulSet') then 'statefulSet'
             else null;
             module {
               [app]+: { spec+: { replicas: 1 } },
               container+:: {
                 resources+: { requests: { cpu: '200m', memory: '500Mi' } },
               },
               persistentVolumeClaim+:: { spec+: { resources+: { requests+: { storage: '1Gi' } } } },
             }
         ),
  },
}
