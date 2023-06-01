# Deploy Grafana Tempo with Jsonnet and Tanka
You can use Tanka and jsonnet-bundler to generate Kubernetes YAML manifests from the jsonnet files.

1. Install tanka and jb:

   Follow the steps at https://tanka.dev/install. If you have go installed locally you can also use:

```shell
# make sure to be outside of GOPATH or a go.mod project
go install github.com/grafana/tanka/cmd/tk@latest
go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@latest
```

2. Set up a Jsonnet project, based on the example that follows:

  * Initialize Tanka 
  * Install Grafana Tempo and Kubernetes Jsonnet libraries 
  * Set up an environment

   ```shell
   #!/usr/bin/env bash
   # SPDX-License-Identifier: AGPL-3.0-only
   
   set -e
   
   # Initialise the Tanka.
   mkdir jsonnet-example && cd jsonnet-example
   tk init --k8s=1.26
   
   # Install Tempo jsonnet.
   jb install github.com/grafana/tempo/operations/jsonnet/microservices@main
   jb install github.com/grafana/jsonnet-libs/memcached
   
   # Use the provided example. In tempo repository in operations/jsonnet-compiled
   cp operations/jsonnet-compliled/util/example/main.jsonnet environments/default/main.jsonnet
   
   # Generate the YAML manifests.
   export PAGER=cat
   tk show environments/default
   ```

3. Generate the Kubernetes YAML manifests and store them in the ./manifests directory:

   ```shell
   # Generate the YAML manifests:
   export PAGER=cat
   tk show environments/default
   tk export manifests environments/default
   ```
4. Configure the environment specification file at environments/default/spec.json.

To learn about how to use Tanka and to configure the spec.json file, see Using Jsonnet: Creating a new project.

5. Deploy the manifests to a Kubernetes cluster, in one of two ways:
   * Use the tk apply command.

   Tanka supports commands to show the diff and apply changes to a Kubernetes cluster:

   ```shell
   # Show the difference between your Jsonnet definition and your Kubernetes cluster:
   tk diff environments/default
   
   # Apply changes to your Kubernetes cluster:
   tk apply environments/default
   ```
  
   * Use the kubectl apply command.

   You generated the Kubernetes manifests and stored them in the ./manifests directory in the previous step.

   You can run the following command to directly apply these manifests to your Kubernetes cluster:

   ```shell
   # Review the changes that will apply to your Kubernetes cluster:
   kubectl apply --dry-run=client -k manifests/
   
   # Apply the changes to your Kubernetes cluster:
   kubectl apply -k manifests/
   ```
6. Multizone ingesters
   To use multizone ingesters use following config fields
   ```
    _config+: {
        multi_zone_ingester_enabled: false,
        multi_zone_ingester_migration_enabled: false,
        multi_zone_ingester_replicas: 0,
        multi_zone_ingester_max_unavailable: 25,
   }
   ```
