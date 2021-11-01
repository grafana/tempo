local d = import 'doc-util/main.libsonnet';

{
  local container = $.core.v1.container,
  local volumeMount = $.core.v1.volumeMount,
  local volume = $.core.v1.volume,

  local patch = {
    local volumeMountDescription =
      |||
        This helper function can be augmented with a `volumeMountsMixin. For example,
        passing "k.core.v1.volumeMount.withSubPath(subpath)" will result in a subpath
        mixin.
      |||,


    '#configVolumeMount': d.fn(
      '`configVolumeMount` mounts a ConfigMap by `name` into all container on `path`.'
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('volumeMountMixin', d.T.object),
      ]
    ),
    configVolumeMount(name, path, volumeMountMixin={})::
      local addMount(c) = c + container.withVolumeMountsMixin(
        volumeMount.new(name, path) +
        volumeMountMixin,
      );

      super.mapContainers(addMount) +
      super.spec.template.spec.withVolumesMixin([
        volume.fromConfigMap(name, name),
      ]),


    '#configMapVolumeMount': d.fn(
      |||
        `configMapVolumeMount` mounts a `configMap` into all container on `path`. It will
        also add an annotation hash to ensure the pods are re-deployed when the config map
        changes.
      |||
      + volumeMountDescription,
      [
        d.arg('configMap', d.T.object),
        d.arg('path', d.T.string),
        d.arg('volumeMountMixin', d.T.object),
      ]
    ),
    configMapVolumeMount(configMap, path, volumeMountMixin={})::
      local name = configMap.metadata.name,
            hash = std.md5(std.toString(configMap)),
            addMount(c) = c + container.withVolumeMountsMixin(
        volumeMount.new(name, path) +
        volumeMountMixin,
      );

      super.mapContainers(addMount) +
      super.spec.template.spec.withVolumesMixin([
        volume.fromConfigMap(name, name),
      ]) +
      super.spec.template.metadata.withAnnotationsMixin({
        ['%s-hash' % name]: hash,
      }),


    '#hostVolumeMount': d.fn(
      '`hostVolumeMount` mounts a `hostPath` into all container on `path`.'
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('hostPath', d.T.string),
        d.arg('path', d.T.string),
        d.arg('readOnly', d.T.bool),
        d.arg('volumeMountMixin', d.T.object),
      ]
    ),
    hostVolumeMount(name, hostPath, path, readOnly=false, volumeMountMixin={})::
      local addMount(c) = c + container.withVolumeMountsMixin(
        volumeMount.new(name, path, readOnly=readOnly) +
        volumeMountMixin,
      );

      super.mapContainers(addMount) +
      super.spec.template.spec.withVolumesMixin([
        volume.fromHostPath(name, hostPath),
      ]),


    '#pvcVolumeMount': d.fn(
      '`hostVolumeMount` mounts a PersistentVolumeClaim by `name` into all container on `path`.'
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('readOnly', d.T.bool),
        d.arg('volumeMountMixin', d.T.object),
      ]
    ),
    pvcVolumeMount(name, path, readOnly=false, volumeMountMixin={})::
      local addMount(c) = c + container.withVolumeMountsMixin(
        volumeMount.new(name, path, readOnly=readOnly) +
        volumeMountMixin,
      );

      super.mapContainers(addMount) +
      super.spec.template.spec.withVolumesMixin([
        volume.fromPersistentVolumeClaim(name, name),
      ]),


    '#secretVolumeMount': d.fn(
      '`secretVolumeMount` mounts a Secret by `name` into all container on `path`.'
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('defaultMode', d.T.string),
        d.arg('volumeMountMixin', d.T.object),
      ]
    ),
    secretVolumeMount(name, path, defaultMode=256, volumeMountMixin={})::
      local addMount(c) = c + container.withVolumeMountsMixin(
        volumeMount.new(name, path) +
        volumeMountMixin,
      );

      super.mapContainers(addMount) +
      super.spec.template.spec.withVolumesMixin([
        volume.fromSecret(name, secretName=name) +
        volume.secret.withDefaultMode(defaultMode),
      ]),


    '#emptyVolumeMount': d.fn(
      '`emptyVolumeMount` mounts empty volume by `name` into all container on `path`.'
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('volumeMountMixin', d.T.object),
        d.arg('volumeMixin', d.T.object),
      ]
    ),
    emptyVolumeMount(name, path, volumeMountMixin={}, volumeMixin={})::
      local addMount(c) = c + container.withVolumeMountsMixin(
        volumeMount.new(name, path) +
        volumeMountMixin,
      );

      super.mapContainers(addMount) +
      super.spec.template.spec.withVolumesMixin([
        volume.fromEmptyDir(name) + volumeMixin,
      ]),
  },

  batch+: {
    v1+: {
      job+: patch,
    },
  },
  apps+: { v1+: {
    daemonSet+: patch,
    deployment+: patch,
    replicaSet+: patch,
    statefulSet+: patch,
  } },
}
