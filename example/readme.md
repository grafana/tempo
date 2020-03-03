# Operations


## Jsonnet

The Jsonnet is meant to be applied to with (tanka)[https://github.com/grafana/tanka]

### Local Testing

To test the jsonnet locally 

- k3d > v1.6.0
- tanka > v0.8.0

```
k3d create --name frigg \
           --publish 16686:16686

export KUBECONFIG="$(k3d get-kubeconfig --name='frigg')"

k3d delete --name frigg
```