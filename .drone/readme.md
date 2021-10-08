# Drone

Drone is used for building our official dockerhub images. It is broken into 3 
pipelines. Note that none of the pipelines include testing so it's important that 
the codebase is otherwise tested when it begins this process. Currently we use GitHub
Actions for testing every PR and only build the main branch, tags and weekly release
branches (`r**`).

# Pipelines

The pipelines are `docker-amd64`, `docker-arm64`, and `manifest`. The two docker pipelines
run concurrently and create images tagged like `tempo:<tag>-<arch>` or `tempo:<branch>-<sha>-<arch>.
E.g. `tempo:1.1.0-arm64` or `tempo:main-e2a314-amd64`. The manifest step then creates a manifest 
that combines the mentioned images into one multiarch image named as you would expect: 
`tempo:1.1.0` or `tempo:main-e2a314`.  

The documentation on the manifest step is basically non-existent. There's some very
weak documentation in the Drone docs, but it's not even worth looking at. To understand
how to use the manifest step I'd recommend looking at the code itself:

https://github.com/drone-plugins/drone-manifest

It is a very simple wrapper that takes the configuration options and runs the following 
cli tool:

https://github.com/estesp/manifest-tool

[`docker-manifest.tmpl`](./docker-manifest.tmpl) is pushed through the standard go templating library with access
to these objects: https://github.com/drone-plugins/drone-manifest/blob/master/plugin.go#L23

# Updating drone.yml

`drone.yml` is generated based upon `drone.jsonnet`. To change the Drone pipelines edit
`drone.jsonnet` and run:

```
make drone
```

# Signature

`drone.yml` contains a signature that can only be generated with an access token from the Grafana
Drone server. If you do not have an access token the last step of `make drone` will fail. Feel free
to still submit a PR, a Tempo maintainer can update the signature before merging the PR. To regenerate
the signature run:

```
make drone-signature
```