# Drone

Drone is used for our building our official dockerhub images. It is broken into 3 
pipelines.  Note that none of the pipelines include testing so it's important that 
the codebase is otherwise tested when it begins this process. Currently we use GHA
for testing on PR and only build the master branch.

# Pipelines

The pipelines are docker-amd64, docker-arm64, and manifest.  The two docker pipelines 
run concurrently and create images tagged like this: `tempo:<tag>-<arch>`.  e.g.
`tempo-2.3.0-arm64` or `tempo-e2a314-amd64`.  The manifest step then creates a manifest 
that combines the mentioned images into one multiarch image named as you would expect: 
`tempo-2.3.0` or `tempo-e2a314`.  

The documentation on the manifest step is basically non-existent. There's some very
weak documentation in the drone docs, but it's not even worth looking at. To understand
how to use the manifest step I'd recommend looking at the code itself:

https://github.com/drone-plugins/drone-manifest

It is a very simple wrapper that takes the configuration options and runs the following 
cli tool:

https://github.com/estesp/manifest-tool

`docker-manifest.tmpl` is pushed through the standard go templating library with access
to these objects: https://github.com/drone-plugins/drone-manifest/blob/master/plugin.go#L23

# Future work

If we extend this any further we should probably generate drone.yml. It has a lot of 
repeated lines that will become difficult to maintain.  Currently loki uses a jsonnet file
and we should perhaps follow suit: https://github.com/grafana/loki/blob/master/.drone/drone.jsonnet