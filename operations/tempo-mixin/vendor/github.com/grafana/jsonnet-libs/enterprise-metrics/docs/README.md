---
permalink: /
---

# enterprise-metrics

```jsonnet
local enterprise-metrics = import "github.com/grafana/jsonnet-libs/enterprise-metrics/main.libsonnet"
```

`enterprise-metrics` produces Kubernetes manifests for a Grafana Enterprise Metrics cluster.

## Index

* [`obj _config`](#obj-_config)
  * [`string _config.adminTokenSecretName`](#string-_configadmintokensecretname)
  * [`obj _config.commonArgs`](#obj-_configcommonargs)
    * [`bool _config.commonArgs.auth.multitenancy-enabled`](#bool-_configcommonargsauthmultitenancy-enabled)
    * [`bool _config.commonArgs.auth.type`](#bool-_configcommonargsauthtype)
    * [`string _config.commonArgs.cluster-name`](#string-_configcommonargscluster-name)
    * [`string _config.commonArgs.instrumentation.distributor-client.address`](#string-_configcommonargsinstrumentationdistributor-clientaddress)
    * [`string _config.commonArgs.instrumentation.enabled`](#string-_configcommonargsinstrumentationenabled)
    * [`string _config.commonArgs.memberlist.join`](#string-_configcommonargsmemberlistjoin)
    * [`string _config.commonArgs.runtime-config.file`](#string-_configcommonargsruntime-configfile)
  * [`string _config.license.path`](#string-_configlicensepath)
  * [`string _config.licenseSecretName`](#string-_configlicensesecretname)
* [`obj _images`](#obj-_images)
  * [`string _images.gem`](#string-_imagesgem)
  * [`string _images.kubectl`](#string-_imageskubectl)
* [`obj adminApi`](#obj-adminapi)
  * [`obj adminApi.args`](#obj-adminapiargs)
    * [`bool adminApi.args.admin-api.leader-election.enabled`](#bool-adminapiargsadmin-apileader-electionenabled)
    * [`string adminApi.args.admin-api.leader-election.ring.store`](#string-adminapiargsadmin-apileader-electionringstore)
    * [`bool adminApi.args.auth.multitenancy-enabled`](#bool-adminapiargsauthmultitenancy-enabled)
    * [`bool adminApi.args.auth.type`](#bool-adminapiargsauthtype)
    * [`string adminApi.args.cluster-name`](#string-adminapiargscluster-name)
    * [`string adminApi.args.instrumentation.distributor-client.address`](#string-adminapiargsinstrumentationdistributor-clientaddress)
    * [`string adminApi.args.instrumentation.enabled`](#string-adminapiargsinstrumentationenabled)
    * [`string adminApi.args.memberlist.join`](#string-adminapiargsmemberlistjoin)
    * [`string adminApi.args.runtime-config.file`](#string-adminapiargsruntime-configfile)
  * [`obj adminApi.container`](#obj-adminapicontainer)
    
  * [`obj adminApi.deployment`](#obj-adminapideployment)
    
  * [`obj adminApi.service`](#obj-adminapiservice)
    
* [`obj alertmanager`](#obj-alertmanager)
  * [`obj alertmanager.args`](#obj-alertmanagerargs)
    * [`string alertmanager.args.alertmanager-storage.s3.bucket-name`](#string-alertmanagerargsalertmanager-storages3bucket-name)
    * [`bool alertmanager.args.auth.multitenancy-enabled`](#bool-alertmanagerargsauthmultitenancy-enabled)
    * [`bool alertmanager.args.auth.type`](#bool-alertmanagerargsauthtype)
    * [`string alertmanager.args.cluster-name`](#string-alertmanagerargscluster-name)
    * [`string alertmanager.args.instrumentation.distributor-client.address`](#string-alertmanagerargsinstrumentationdistributor-clientaddress)
    * [`string alertmanager.args.instrumentation.enabled`](#string-alertmanagerargsinstrumentationenabled)
    * [`string alertmanager.args.memberlist.join`](#string-alertmanagerargsmemberlistjoin)
    * [`string alertmanager.args.runtime-config.file`](#string-alertmanagerargsruntime-configfile)
  * [`obj alertmanager.container`](#obj-alertmanagercontainer)
    
  * [`obj alertmanager.persistentVolumeClaim`](#obj-alertmanagerpersistentvolumeclaim)
    
  * [`obj alertmanager.service`](#obj-alertmanagerservice)
    
  * [`obj alertmanager.statefulSet`](#obj-alertmanagerstatefulset)
    
* [`obj compactor`](#obj-compactor)
  * [`obj compactor.args`](#obj-compactorargs)
    * [`bool compactor.args.auth.multitenancy-enabled`](#bool-compactorargsauthmultitenancy-enabled)
    * [`bool compactor.args.auth.type`](#bool-compactorargsauthtype)
    * [`string compactor.args.cluster-name`](#string-compactorargscluster-name)
    * [`string compactor.args.instrumentation.distributor-client.address`](#string-compactorargsinstrumentationdistributor-clientaddress)
    * [`string compactor.args.instrumentation.enabled`](#string-compactorargsinstrumentationenabled)
    * [`string compactor.args.memberlist.join`](#string-compactorargsmemberlistjoin)
    * [`string compactor.args.runtime-config.file`](#string-compactorargsruntime-configfile)
  * [`obj compactor.container`](#obj-compactorcontainer)
    
  * [`obj compactor.persistentVolumeClaim`](#obj-compactorpersistentvolumeclaim)
    
  * [`obj compactor.service`](#obj-compactorservice)
    
  * [`obj compactor.statefulSet`](#obj-compactorstatefulset)
    
* [`obj distributor`](#obj-distributor)
  * [`obj distributor.args`](#obj-distributorargs)
    * [`bool distributor.args.auth.multitenancy-enabled`](#bool-distributorargsauthmultitenancy-enabled)
    * [`bool distributor.args.auth.type`](#bool-distributorargsauthtype)
    * [`string distributor.args.cluster-name`](#string-distributorargscluster-name)
    * [`string distributor.args.instrumentation.distributor-client.address`](#string-distributorargsinstrumentationdistributor-clientaddress)
    * [`string distributor.args.instrumentation.enabled`](#string-distributorargsinstrumentationenabled)
    * [`string distributor.args.memberlist.join`](#string-distributorargsmemberlistjoin)
    * [`string distributor.args.runtime-config.file`](#string-distributorargsruntime-configfile)
  * [`obj distributor.container`](#obj-distributorcontainer)
    
  * [`obj distributor.service`](#obj-distributorservice)
    
  * [`obj distributor.statefulSet`](#obj-distributorstatefulset)
    
* [`obj gateway`](#obj-gateway)
  * [`obj gateway.args`](#obj-gatewayargs)
    * [`bool gateway.args.auth.multitenancy-enabled`](#bool-gatewayargsauthmultitenancy-enabled)
    * [`bool gateway.args.auth.type`](#bool-gatewayargsauthtype)
    * [`string gateway.args.cluster-name`](#string-gatewayargscluster-name)
    * [`string gateway.args.gateway.proxy.admin-api.url`](#string-gatewayargsgatewayproxyadmin-apiurl)
    * [`string gateway.args.gateway.proxy.alertmanager.url`](#string-gatewayargsgatewayproxyalertmanagerurl)
    * [`string gateway.args.gateway.proxy.compactor.url`](#string-gatewayargsgatewayproxycompactorurl)
    * [`string gateway.args.gateway.proxy.distributor.url`](#string-gatewayargsgatewayproxydistributorurl)
    * [`string gateway.args.gateway.proxy.ingester.url`](#string-gatewayargsgatewayproxyingesterurl)
    * [`string gateway.args.gateway.proxy.query-frontend.url`](#string-gatewayargsgatewayproxyquery-frontendurl)
    * [`string gateway.args.gateway.proxy.ruler.url`](#string-gatewayargsgatewayproxyrulerurl)
    * [`string gateway.args.gateway.proxy.store-gateway.url`](#string-gatewayargsgatewayproxystore-gatewayurl)
    * [`string gateway.args.instrumentation.distributor-client.address`](#string-gatewayargsinstrumentationdistributor-clientaddress)
    * [`string gateway.args.instrumentation.enabled`](#string-gatewayargsinstrumentationenabled)
    * [`string gateway.args.memberlist.join`](#string-gatewayargsmemberlistjoin)
    * [`string gateway.args.runtime-config.file`](#string-gatewayargsruntime-configfile)
  * [`obj gateway.container`](#obj-gatewaycontainer)
    
  * [`obj gateway.deployment`](#obj-gatewaydeployment)
    
  * [`obj gateway.service`](#obj-gatewayservice)
    
* [`obj gossipRing`](#obj-gossipring)
  * [`obj gossipRing.service`](#obj-gossipringservice)
    
* [`obj ingester`](#obj-ingester)
  * [`obj ingester.args`](#obj-ingesterargs)
    * [`bool ingester.args.auth.multitenancy-enabled`](#bool-ingesterargsauthmultitenancy-enabled)
    * [`bool ingester.args.auth.type`](#bool-ingesterargsauthtype)
    * [`string ingester.args.cluster-name`](#string-ingesterargscluster-name)
    * [`string ingester.args.instrumentation.distributor-client.address`](#string-ingesterargsinstrumentationdistributor-clientaddress)
    * [`string ingester.args.instrumentation.enabled`](#string-ingesterargsinstrumentationenabled)
    * [`string ingester.args.memberlist.join`](#string-ingesterargsmemberlistjoin)
    * [`string ingester.args.runtime-config.file`](#string-ingesterargsruntime-configfile)
  * [`obj ingester.container`](#obj-ingestercontainer)
    
  * [`obj ingester.persistentVolumeClaim`](#obj-ingesterpersistentvolumeclaim)
    
  * [`obj ingester.podDisruptionBudget`](#obj-ingesterpoddisruptionbudget)
    
  * [`obj ingester.service`](#obj-ingesterservice)
    
  * [`obj ingester.statefulSet`](#obj-ingesterstatefulset)
    
* [`obj memcached`](#obj-memcached)
  * [`obj memcached.chunks`](#obj-memcachedchunks)
    
  * [`obj memcached.frontend`](#obj-memcachedfrontend)
    
  * [`obj memcached.metadata`](#obj-memcachedmetadata)
    
  * [`obj memcached.queries`](#obj-memcachedqueries)
    
* [`obj overridesExporter`](#obj-overridesexporter)
  * [`obj overridesExporter.args`](#obj-overridesexporterargs)
    * [`bool overridesExporter.args.auth.multitenancy-enabled`](#bool-overridesexporterargsauthmultitenancy-enabled)
    * [`bool overridesExporter.args.auth.type`](#bool-overridesexporterargsauthtype)
    * [`string overridesExporter.args.cluster-name`](#string-overridesexporterargscluster-name)
    * [`string overridesExporter.args.instrumentation.distributor-client.address`](#string-overridesexporterargsinstrumentationdistributor-clientaddress)
    * [`string overridesExporter.args.instrumentation.enabled`](#string-overridesexporterargsinstrumentationenabled)
    * [`string overridesExporter.args.memberlist.join`](#string-overridesexporterargsmemberlistjoin)
    * [`string overridesExporter.args.runtime-config.file`](#string-overridesexporterargsruntime-configfile)
  * [`obj overridesExporter.container`](#obj-overridesexportercontainer)
    
  * [`obj overridesExporter.deployment`](#obj-overridesexporterdeployment)
    
  * [`obj overridesExporter.service`](#obj-overridesexporterservice)
    
* [`obj querier`](#obj-querier)
  * [`obj querier.args`](#obj-querierargs)
    * [`bool querier.args.auth.multitenancy-enabled`](#bool-querierargsauthmultitenancy-enabled)
    * [`bool querier.args.auth.type`](#bool-querierargsauthtype)
    * [`string querier.args.cluster-name`](#string-querierargscluster-name)
    * [`string querier.args.instrumentation.distributor-client.address`](#string-querierargsinstrumentationdistributor-clientaddress)
    * [`string querier.args.instrumentation.enabled`](#string-querierargsinstrumentationenabled)
    * [`string querier.args.memberlist.join`](#string-querierargsmemberlistjoin)
    * [`string querier.args.runtime-config.file`](#string-querierargsruntime-configfile)
  * [`obj querier.container`](#obj-queriercontainer)
    
  * [`obj querier.deployment`](#obj-querierdeployment)
    
  * [`obj querier.service`](#obj-querierservice)
    
* [`obj queryFrontend`](#obj-queryfrontend)
  * [`obj queryFrontend.args`](#obj-queryfrontendargs)
    * [`bool queryFrontend.args.auth.multitenancy-enabled`](#bool-queryfrontendargsauthmultitenancy-enabled)
    * [`bool queryFrontend.args.auth.type`](#bool-queryfrontendargsauthtype)
    * [`string queryFrontend.args.cluster-name`](#string-queryfrontendargscluster-name)
    * [`string queryFrontend.args.instrumentation.distributor-client.address`](#string-queryfrontendargsinstrumentationdistributor-clientaddress)
    * [`string queryFrontend.args.instrumentation.enabled`](#string-queryfrontendargsinstrumentationenabled)
    * [`string queryFrontend.args.memberlist.join`](#string-queryfrontendargsmemberlistjoin)
    * [`string queryFrontend.args.runtime-config.file`](#string-queryfrontendargsruntime-configfile)
  * [`obj queryFrontend.container`](#obj-queryfrontendcontainer)
    
  * [`obj queryFrontend.deployment`](#obj-queryfrontenddeployment)
    
  * [`obj queryFrontend.discoveryService`](#obj-queryfrontenddiscoveryservice)
    
  * [`obj queryFrontend.service`](#obj-queryfrontendservice)
    
* [`obj queryScheduler`](#obj-queryscheduler)
  * [`obj queryScheduler.args`](#obj-queryschedulerargs)
    * [`bool queryScheduler.args.auth.multitenancy-enabled`](#bool-queryschedulerargsauthmultitenancy-enabled)
    * [`bool queryScheduler.args.auth.type`](#bool-queryschedulerargsauthtype)
    * [`string queryScheduler.args.cluster-name`](#string-queryschedulerargscluster-name)
    * [`string queryScheduler.args.instrumentation.distributor-client.address`](#string-queryschedulerargsinstrumentationdistributor-clientaddress)
    * [`string queryScheduler.args.instrumentation.enabled`](#string-queryschedulerargsinstrumentationenabled)
    * [`string queryScheduler.args.memberlist.join`](#string-queryschedulerargsmemberlistjoin)
    * [`string queryScheduler.args.runtime-config.file`](#string-queryschedulerargsruntime-configfile)
  * [`obj queryScheduler.container`](#obj-queryschedulercontainer)
    
  * [`obj queryScheduler.deployment`](#obj-queryschedulerdeployment)
    
  * [`obj queryScheduler.discoveryService`](#obj-queryschedulerdiscoveryservice)
    
  * [`obj queryScheduler.service`](#obj-queryschedulerservice)
    
* [`obj ruler`](#obj-ruler)
  * [`obj ruler.args`](#obj-rulerargs)
    * [`bool ruler.args.auth.multitenancy-enabled`](#bool-rulerargsauthmultitenancy-enabled)
    * [`bool ruler.args.auth.type`](#bool-rulerargsauthtype)
    * [`string ruler.args.cluster-name`](#string-rulerargscluster-name)
    * [`string ruler.args.instrumentation.distributor-client.address`](#string-rulerargsinstrumentationdistributor-clientaddress)
    * [`string ruler.args.instrumentation.enabled`](#string-rulerargsinstrumentationenabled)
    * [`string ruler.args.memberlist.join`](#string-rulerargsmemberlistjoin)
    * [`string ruler.args.ruler-storage.s3.bucket-name`](#string-rulerargsruler-storages3bucket-name)
    * [`string ruler.args.runtime-config.file`](#string-rulerargsruntime-configfile)
  * [`obj ruler.container`](#obj-rulercontainer)
    
  * [`obj ruler.deployment`](#obj-rulerdeployment)
    
  * [`obj ruler.service`](#obj-rulerservice)
    
* [`obj runtime`](#obj-runtime)
  * [`obj runtime.config`](#obj-runtimeconfig)
    
  * [`obj runtime.configMap`](#obj-runtimeconfigmap)
    
  * [`obj runtime.configuration`](#obj-runtimeconfiguration)
    * [`obj runtime.configuration.overrides`](#obj-runtimeconfigurationoverrides)
      
* [`obj storeGateway`](#obj-storegateway)
  * [`obj storeGateway.args`](#obj-storegatewayargs)
    * [`bool storeGateway.args.auth.multitenancy-enabled`](#bool-storegatewayargsauthmultitenancy-enabled)
    * [`bool storeGateway.args.auth.type`](#bool-storegatewayargsauthtype)
    * [`string storeGateway.args.cluster-name`](#string-storegatewayargscluster-name)
    * [`string storeGateway.args.instrumentation.distributor-client.address`](#string-storegatewayargsinstrumentationdistributor-clientaddress)
    * [`string storeGateway.args.instrumentation.enabled`](#string-storegatewayargsinstrumentationenabled)
    * [`string storeGateway.args.memberlist.join`](#string-storegatewayargsmemberlistjoin)
    * [`string storeGateway.args.runtime-config.file`](#string-storegatewayargsruntime-configfile)
  * [`obj storeGateway.container`](#obj-storegatewaycontainer)
    
  * [`obj storeGateway.persistentVolumeClaim`](#obj-storegatewaypersistentvolumeclaim)
    
  * [`obj storeGateway.podDisruptionBudget`](#obj-storegatewaypoddisruptionbudget)
    
  * [`obj storeGateway.service`](#obj-storegatewayservice)
    
  * [`obj storeGateway.statefulSet`](#obj-storegatewaystatefulset)
    
* [`obj tokengen`](#obj-tokengen)
  * [`obj tokengen.args`](#obj-tokengenargs)
    * [`bool tokengen.args.auth.multitenancy-enabled`](#bool-tokengenargsauthmultitenancy-enabled)
    * [`bool tokengen.args.auth.type`](#bool-tokengenargsauthtype)
    * [`string tokengen.args.cluster-name`](#string-tokengenargscluster-name)
    * [`string tokengen.args.instrumentation.distributor-client.address`](#string-tokengenargsinstrumentationdistributor-clientaddress)
    * [`string tokengen.args.instrumentation.enabled`](#string-tokengenargsinstrumentationenabled)
    * [`string tokengen.args.memberlist.join`](#string-tokengenargsmemberlistjoin)
    * [`string tokengen.args.runtime-config.file`](#string-tokengenargsruntime-configfile)
  * [`obj tokengen.container`](#obj-tokengencontainer)
    
  * [`obj tokengen.createSecretContainer`](#obj-tokengencreatesecretcontainer)
    
  * [`obj tokengen.job`](#obj-tokengenjob)
    
  * [`obj tokengen.role`](#obj-tokengenrole)
    
  * [`obj tokengen.roleBinding`](#obj-tokengenrolebinding)
    
  * [`obj tokengen.serviceAccount`](#obj-tokengenserviceaccount)
    
* [`obj util`](#obj-util)
  * [`fn mapModules(fn)`](#fn-utilmapmodules)
  * [`array util.util`](#array-utilutil)

## Fields

## obj _config

`_config` is used for consumer overrides and configuration. Similar to a Helm values.yaml file

### string _config.adminTokenSecretName

*Default value: * `gem-admin-token`

When generating an admin token using the `tokengen` target, the result is written to the Kubernetes
Secret `adminTokenSecretName`.
There are two versions of the token in the Secret: `token` is the raw token obtained from the `tokengen`,
and `grafana-token` is a base64 encoded version that can be used directly when provisioning Grafana
via configuration file.
To retrieve these tokens from Kubernetes using `kubectl`:
$ kubectl get secret gem-admin-token -o jsonpath="{.data.token}" | base64 --decode ; echo
$ kubectl get secret gem-admin-token -o jsonpath="{.data.grafana-token}" | base64 --decode ; echo


## obj _config.commonArgs

`commonArgs` is a convenience field that can be used to modify the container arguments of all modules as key-value pairs.

### bool _config.commonArgs.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool _config.commonArgs.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string _config.commonArgs.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string _config.commonArgs.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string _config.commonArgs.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string _config.commonArgs.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string _config.commonArgs.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

### string _config.license.path

*Default value: * `/etc/gem-license/license.jwt`

`license.path` configures where this component expects to find a Grafana Enterprise Metrics License.

### string _config.licenseSecretName

*Default value: * `gem-license`

The admin-api expects a Grafana Enterprise Metrics license configured as 'license.jwt' in the
Kubernetes Secret with `licenseSecretName`.
To create the Kubernetes Secret from a local 'license.jwt' file:
$ kubectl create secret generic gem-license -from-file=license.jwt


## obj _images

`_images` contains fields for container images.

### string _images.gem

*Default value: * `grafana/metrics-enterprise:v2.0.1`

`gem` is the Grafana Enterprise Metrics container image.

### string _images.kubectl

*Default value: * `bitnami/kubectl`

`kubectl` is the image used for kubectl containers.

## obj adminApi

`adminApi` has configuration for the admin-api.

## obj adminApi.args

`args` is a convenience field that can be used to modify the admin-api container arguments as key value pairs.

### bool adminApi.args.admin-api.leader-election.enabled

*Default value: * `true`

`admin-api.leader-election.enabled` enables leader election for to avoid inconsistent state with parallel writes when multiple replicas of the admin-api are running.

### string adminApi.args.admin-api.leader-election.ring.store

*Default value: * `memberlist`

`admin-api.leader-election.ring.store` is the type of key-value store to use for admin-api leader election.

### bool adminApi.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool adminApi.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string adminApi.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string adminApi.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string adminApi.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string adminApi.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string adminApi.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj adminApi.container

`container` is a convenience field that can be used to modify the admin-api container.

## obj adminApi.deployment

`deployment` is the Kubernetes Deployment for the admin-api.

## obj adminApi.service

`service` is the Kubernetes Service for the admin-api.

## obj alertmanager

`alertmanager` has configuration for the alertmanager. To disable the alertmanager, ensure the alertmanager object field is hidden

## obj alertmanager.args

`args` is a convenience field that can be used to modify the alertmanager container arguments as key-value pairs.

### string alertmanager.args.alertmanager-storage.s3.bucket-name

*Default value: * `alertmanager`

`alertmanager-storage.s3.bucket-name` is the name of the bucket in which the alertmanager data will be stored.

### bool alertmanager.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool alertmanager.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string alertmanager.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string alertmanager.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string alertmanager.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string alertmanager.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string alertmanager.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj alertmanager.container

`container` is a convenience field that can be used to modify the alertmanager container.

## obj alertmanager.persistentVolumeClaim

`persistentVolumeClaim` is a convenience field that can be used to modify the alertmanager PersistentVolumeClaim.

## obj alertmanager.service

`service` is the Kubernetes Service for the alertmanager.

## obj alertmanager.statefulSet

`statefulSet` is the Kubernetes StatefulSet for the alertmanager.

## obj compactor

`compactor` has configuration for the compactor.

## obj compactor.args

`args` is a convenience field that can be used to modify the compactor container arguments as key-value pairs.

### bool compactor.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool compactor.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string compactor.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string compactor.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string compactor.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string compactor.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string compactor.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj compactor.container

`container` is a convenience field that can be used to modify the compactor container.

## obj compactor.persistentVolumeClaim

`persistentVolumeClaim` is a convenience field that can be used to modify the compactor PersistentVolumeClaim.

## obj compactor.service

`service` is the Kubernetes Service for the compactor.

## obj compactor.statefulSet

`statefulSet` is the Kubernetes StatefulSet for the compactor.

## obj distributor

`distributor` has configuration for the distributor.

## obj distributor.args

`args` is a convenience field that can be used to modify the distributor container arguments as key-value pairs.

### bool distributor.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool distributor.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string distributor.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string distributor.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string distributor.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string distributor.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string distributor.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj distributor.container

`container` is a convenience field that can be used to modify the distributor container.

## obj distributor.service

`service` is the Kubernetes Service for the distributor.

## obj distributor.statefulSet

`deployment` is the Kubernetes Deployment for the distributor.

## obj gateway

`gateway` has configuration for the gateway.

## obj gateway.args

`args` is a convenience field that can be used to modify the gateway container arguments as key-value pairs.

### bool gateway.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool gateway.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string gateway.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string gateway.args.gateway.proxy.admin-api.url

*Default value: * `http://admin-api`

`gateway.proxy.admin-api.url is the upstream URL of the admin-api.

### string gateway.args.gateway.proxy.alertmanager.url

*Default value: * `http://alertmanager`

`gateway.proxy.alertmanager.url is the upstream URL of the alertmanager.

### string gateway.args.gateway.proxy.compactor.url

*Default value: * `http://compactor`

`gateway.proxy.compactor.url is the upstream URL of the compactor.

### string gateway.args.gateway.proxy.distributor.url

*Default value: * `dns:///distributor:9095`

`gateway.proxy.distributor.url is the upstream URL of the distributor.

### string gateway.args.gateway.proxy.ingester.url

*Default value: * `http://ingester`

`gateway.proxy.ingester.url is the upstream URL of the ingester.

### string gateway.args.gateway.proxy.query-frontend.url

*Default value: * `http://query-frontend`

`gateway.proxy.query-frontend.url is the upstream URL of the query-frontend.

### string gateway.args.gateway.proxy.ruler.url

*Default value: * `http://ruler`

`gateway.proxy.ruler.url is the upstream URL of the ruler.

### string gateway.args.gateway.proxy.store-gateway.url

*Default value: * `http://store-gateway`

`gateway.proxy.store-gateway.url is the upstream URL of the store-gateway.

### string gateway.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string gateway.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string gateway.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string gateway.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj gateway.container

`container` is a convenience field that can be used to modify the gateway container.

## obj gateway.deployment

`deployment` is the Kubernetes Deployment for the gateway.

## obj gateway.service

`service` is the Kubernetes Service for the gateway.

## obj gossipRing

`gossipRing` is used by microservices to discover other memberlist members.

## obj gossipRing.service

`service` is the Kubernetes Service for the gossip ring.

## obj ingester

`ingester` has configuration for the ingester.

## obj ingester.args

`args` is a convenience field that can be used to modify the ingester container arguments as key-value pairs.

### bool ingester.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool ingester.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string ingester.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string ingester.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string ingester.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string ingester.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string ingester.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj ingester.container

`container` is a convenience field that can be used to modify the ingester container.

## obj ingester.persistentVolumeClaim

`persistentVolumeClaim` is a convenience field that can be used to modify the ingester PersistentVolumeClaim. It is recommended to use a fast storage class.

## obj ingester.podDisruptionBudget

`podDisruptionBudget` is the Kubernetes PodDisruptionBudget for the ingester.

## obj ingester.service

`service` is the Kubernetes Service for the ingester.

## obj ingester.statefulSet

`statefulSet` is the Kubernetes StatefulSet for the ingester.

## obj memcached

`memcached` has configuration for GEM caches.

## obj memcached.chunks

`chunks` is a cache for time series chunks.

## obj memcached.frontend

`frontend` is a cache for query-frontend query results.

## obj memcached.metadata

`metadata` is cache for object store metadata used by the queriers and store-gateways.

## obj memcached.queries

`queries` is a cache for index queries used by the store-gateways.

## obj overridesExporter

`overridesExporter` has configuration for the overrides-exporter.

## obj overridesExporter.args

`args` is a convenience field that can be used to modify the overrides-exporter container arguments as key value pairs.

### bool overridesExporter.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool overridesExporter.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string overridesExporter.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string overridesExporter.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string overridesExporter.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string overridesExporter.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string overridesExporter.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj overridesExporter.container

`container` is a convenience field that can be used to modify the overrides-exporter container.

## obj overridesExporter.deployment

`deployment` is the Kubernetes Deployment for the overrides-exporter.

## obj overridesExporter.service

`service` is the Kubernetes Service for the overrides-exporter.

## obj querier

`querier` has configuration for the querier.

## obj querier.args

`args` is a convenience field that can be used to modify the querier container arguments as key-value pairs.

### bool querier.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool querier.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string querier.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string querier.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string querier.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string querier.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string querier.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj querier.container

`container` is a convenience field that can be used to modify the querier container.

## obj querier.deployment

`deployment` is the Kubernetes Deployment for the querier.

## obj querier.service

`service` is the Kubernetes Service for the querier.

## obj queryFrontend

`queryFrontend` has configuration for the query-frontend.

## obj queryFrontend.args

`args` is a convenience field that can be used to modify the query-frontend container arguments as key-value pairs.

### bool queryFrontend.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool queryFrontend.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string queryFrontend.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string queryFrontend.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string queryFrontend.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string queryFrontend.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string queryFrontend.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj queryFrontend.container

`container` is a convenience field that can be used to modify the query-frontend container.

## obj queryFrontend.deployment

`deployment` is the Kubernetes Deployment for the query-frontend.

## obj queryFrontend.discoveryService

`discoveryService` is a headless Kubernetes Service used by queriers to discover query-frontend addresses.

## obj queryFrontend.service

`service` is the Kubernetes Service for the query-frontend.

## obj queryScheduler

`queryScheduler` has configuration for the query-scheduler.

## obj queryScheduler.args

`args` is a convenience field that can be used to modify the query-scheduler container arguments as key-value pairs.

### bool queryScheduler.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool queryScheduler.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string queryScheduler.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string queryScheduler.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string queryScheduler.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string queryScheduler.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string queryScheduler.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj queryScheduler.container

`container` is a convenience field that can be used to modify the query-scheduler container.

## obj queryScheduler.deployment

`deployment` is the Kubernetes Deployment for the query-scheduler.

## obj queryScheduler.discoveryService

`discoveryService` is a headless Kubernetes Service used by queriers to discover query-scheduler addresses.

## obj queryScheduler.service

`service` is the Kubernetes Service for the query-scheduler.

## obj ruler

`ruler` has configuration for the ruler.

## obj ruler.args

`args` is a convenience field that can be used to modify the ruler container arguments as key-value pairs.

### bool ruler.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool ruler.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string ruler.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string ruler.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string ruler.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string ruler.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string ruler.args.ruler-storage.s3.bucket-name

*Default value: * `ruler`

`ruler-storage.s3.bucket-name` is the name of the bucket in which the ruler data will be stored.

### string ruler.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj ruler.container

`container` is a convenience field that can be used to modify the ruler container.

## obj ruler.deployment

`deployment` is the Kubernetes Deployment for the ruler.

## obj ruler.service

`service` is the Kubernetes Service for the ruler.

## obj runtime

`runtime` has configuration for runtime overrides.

## obj runtime.config

`config` is a convenience field for modifying the runtime configuration.

## obj runtime.configMap

`configMap` is the Kubernetes ConfigMap containing the runtime configuration.

## obj runtime.configuration



## obj runtime.configuration.overrides

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
  },
}


## obj storeGateway

`storeGateway` has configuration for the store-gateway.

## obj storeGateway.args

`args` is a convenience field that can be used to modify the store-gateway container arguments as key-value pairs.

### bool storeGateway.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool storeGateway.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string storeGateway.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string storeGateway.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string storeGateway.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string storeGateway.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string storeGateway.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj storeGateway.container

`container` is a convenience field that can be used to modify the store-gateway container.

## obj storeGateway.persistentVolumeClaim

`persistentVolumeClaim` is a convenience field that can be used to modify the store-gateway PersistentVolumeClaim.

## obj storeGateway.podDisruptionBudget

`podDisruptionBudget` is the Kubernetes PodDisruptionBudget for the store-gateway.

## obj storeGateway.service

`service` is the Kubernetes Service for the store-gateway.

## obj storeGateway.statefulSet

`statefulSet` is the Kubernetes StatefulSet for the store-gateway.

## obj tokengen

`tokengen` has configuration for tokengen.
By default the tokengen object is hidden as it is a one-off task. To deploy the tokengen job, unhide the tokengen object field.


## obj tokengen.args

`args` is convenience field for modifying the tokegen container arguments as key-value pairs.

### bool tokengen.args.auth.multitenancy-enabled

*Default value: * `true`

`auth.multitenancy-enabled` enables multitenancy

### bool tokengen.args.auth.type

*Default value: * `enterprise`

`auth.type` configures the type of authentication in use.
`enterprise` uses Grafana Enterprise token authentication.
`default` uses Cortex authentication.


### string tokengen.args.cluster-name

`cluster-name` is the cluster name associated with your Grafana Enterprise Metrics license.

### string tokengen.args.instrumentation.distributor-client.address

*Default value: * `dns:///distributor:9095`

`instrumentation.distributor-client.address` specifies the gRPGC listen address of the distributor service to which the self-monitoring metrics are pushed. Must be a DNS address (`dns:///`) to enable client side load balancing.

### string tokengen.args.instrumentation.enabled

*Default value: * `true`

`instrumentation.enabled` enables self-monitoring metrics recorded under the system instance

### string tokengen.args.memberlist.join

*Default value: * `gossip-ring`

`memberlist.join` is an address used to find memberlist peers for ring gossip

### string tokengen.args.runtime-config.file

`runtime-config.file` provides a reloadable runtime configuration file for some specific configuration.

## obj tokengen.container

`container` is a convenience field for modifying the tokengen container.
By default, the container runs GEM with the tokengen target and writes the token to a file.


## obj tokengen.createSecretContainer

`createSecretContainer` creates a Kubernetes Secret from a token file.

## obj tokengen.job

`job` is the Kubernetes Job for tokengen

## obj tokengen.role

`role` is the Kubernetes Role for tokengen

## obj tokengen.roleBinding

`roleBinding` is the Kubernetes RoleBinding for tokengen

## obj tokengen.serviceAccount

`serviceAccount` is the Kubernetes ServiceAccount for tokengen

## obj util

`util` contains utility functions for working with the GEM Jsonnet library

### fn util.mapModules

```ts
mapModules(fn)
```

`mapModules` applies the function fn to each module in the GEM cluster

### array util.util

`modules` is an array of the names of all modules in the cluster