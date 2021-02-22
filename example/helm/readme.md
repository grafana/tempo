## Helm

Congratulations!  You have successfully found the Helm examples.  These examples are meant for
advanced users looking to deploy Tempo in a microservices pattern.  If you are just getting started
might I recommend the [docker-compose examples](../docker-compose).  The docker-compose examples also are much
better at demonstrating trace discovery flows using Loki and other tools.

If you're convinced this is the place for you then keep reading!

### Initial Steps

To test the Helm example locally requires:

- k3d > v3.2.0
- helm > v3.0.0

Create a cluster

```console
k3d cluster create tempo --api-port 6443 --port "16686:80@loadbalancer"
```

Next either deploy the microservices or the single binary.

### Microservices
The microservices deploy of Tempo is fault tolerant, high volume, independently scalable.  This jsonnet is in use by
Grafana to run our hosted Tempo offering.

```console
# double check you're applying to your local k3d before running this!
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm upgrade --install tempo grafana/tempo-distributed
kubectl create -f microservices-extras.yaml
```

### Single Binary
The Tempo single binary configuration is currently setup to store traces locally on disk, but can easily be configured to
store them in an S3 or GCS bucket.  See configuration docs or some of the other examples for help.

```console
# double check you're applying to your local k3d before running this!
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm upgrade --install tempo grafana/tempo
kubectl create -f single-binary-extras.yaml
```

### View a trace
After the applications are running check the load generators logs
```console
kc logs synthetic-load-generator-???
...
20/03/03 21:30:01 INFO ScheduledTraceGenerator: Emitted traceId e9f4add3ac7c7115 for service frontend route /product
20/03/03 21:30:01 INFO ScheduledTraceGenerator: Emitted traceId 3890ea9c4d7fab00 for service frontend route /cart
20/03/03 21:30:01 INFO ScheduledTraceGenerator: Emitted traceId c36fc5169bf0693d for service frontend route /cart
20/03/03 21:30:01 INFO ScheduledTraceGenerator: Emitted traceId ebaf7d02b96b30fc for service frontend route /cart
20/03/03 21:30:02 INFO ScheduledTraceGenerator: Emitted traceId 23a09a0efd0d1ef0 for service frontend route /cart
```

Extract a trace id and view it in your browser at `http://localhost:16686/trace/<traceid>`

### Clean up
```console
k3d cluster delete tempo
```