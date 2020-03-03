# Example

## Docker Compose

```
cd docker-compose
docker-compose up
```
- Frigg
  - http://localhost:3100
- Frigg-Query
  - http://localhost:16686
- Grafana
  - http://localhost:3000
- Prometheus
  - http://localhost:9090

## Jsonnet/Tanka

The Jsonnet is meant to be applied to with (tanka)[https://github.com/grafana/tanka]

### Local Testing

To test the jsonnet locally requires

- k3d > v1.6.0
- tanka > v0.8.0

```
cd tk
k3d create --name frigg \
           --publish 16686:16686

export KUBECONFIG="$(k3d get-kubeconfig --name='frigg')"

# double check you're applying to your local k3d before running this!
tk apply frigg

k3d delete --name frigg
```