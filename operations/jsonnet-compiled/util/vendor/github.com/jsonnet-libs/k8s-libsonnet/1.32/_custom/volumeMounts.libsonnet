local d = import 'doc-util/main.libsonnet';

{
  local container = $.core.v1.container,
  local volumeMount = $.core.v1.volumeMount,
  local volume = $.core.v1.volume,

  local patch = {
    local volumeMountDescription =
      |||
        This helper function can be augmented with a `volumeMountsMixin`. For example,
        passing "k.core.v1.volumeMount.withSubPath(subpath)" will result in a subpath
        mixin.
      |||,


    '#configVolumeMount': d.fn(
      |||
        `configVolumeMount` mounts a ConfigMap by `name` on `path`.

        If `containers` is specified as an array of container names it will only be mounted
        to those containers, otherwise it will be mounted on all containers.

        This helper function can be augmented with a `volumeMixin`. For example,
        passing "k.core.v1.volume.configMap.withDefaultMode(420)" will result in a
        default mode mixin.
      |||
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('volumeMountMixin', d.T.object),
        d.arg('volumeMixin', d.T.object),
        d.arg('containers', d.T.array),
      ]
    ),
    configVolumeMount(name, path, volumeMountMixin={}, volumeMixin={}, containers=null, includeInitContainers=false)::
      local addMount(c) = c + (
        if containers == null || std.member(containers, c.name)
        then container.withVolumeMountsMixin(
          volumeMount.new(name, path) +
          volumeMountMixin,
        )
        else {}
      );
      local volumeMixins = [volume.fromConfigMap(name, name) + volumeMixin];

      super.mapContainers(addMount, includeInitContainers=includeInitContainers) +
      if std.objectHas(super.spec, 'template')
      then super.spec.template.spec.withVolumesMixin(volumeMixins)
      else super.spec.jobTemplate.spec.template.spec.withVolumesMixin(volumeMixins),


    '#configMapVolumeMount': d.fn(
      |||
        `configMapVolumeMount` mounts a `configMap` on `path`. It will
        also add an annotation hash to ensure the pods are re-deployed when the config map
        changes.

        If `containers` is specified as an array of container names it will only be mounted
        to those containers, otherwise it will be mounted on all containers.

        This helper function can be augmented with a `volumeMixin`. For example,
        passing "k.core.v1.volume.configMap.withDefaultMode(420)" will result in a
        default mode mixin.
      |||
      + volumeMountDescription,
      [
        d.arg('configMap', d.T.object),
        d.arg('path', d.T.string),
        d.arg('volumeMountMixin', d.T.object),
        d.arg('volumeMixin', d.T.object),
        d.arg('containers', d.T.array),
      ]
    ),
    configMapVolumeMount(configMap, path, volumeMountMixin={}, volumeMixin={}, containers=null, includeInitContainers=false)::
      local name = configMap.metadata.name,
            hash = std.md5(std.toString(configMap));
      local addMount(c) = c + (
        if containers == null || std.member(containers, c.name)
        then container.withVolumeMountsMixin(
          volumeMount.new(name, path) +
          volumeMountMixin,
        )
        else {}
      );
      local volumeMixins = [volume.fromConfigMap(name, name) + volumeMixin];
      local annotations = { ['%s-hash' % name]: hash };

      super.mapContainers(addMount, includeInitContainers=includeInitContainers) +
      if std.objectHas(super.spec, 'template')
      then
        super.spec.template.spec.withVolumesMixin(volumeMixins) +
        super.spec.template.metadata.withAnnotationsMixin(annotations)
      else
        super.spec.jobTemplate.spec.template.spec.withVolumesMixin(volumeMixins) +
        super.spec.jobTemplate.spec.template.metadata.withAnnotationsMixin(annotations),


    '#hostVolumeMount': d.fn(
      |||
        `hostVolumeMount` mounts a `hostPath` on `path`.

        If `containers` is specified as an array of container names it will only be mounted
        to those containers, otherwise it will be mounted on all containers.

        This helper function can be augmented with a `volumeMixin`. For example,
        passing "k.core.v1.volume.hostPath.withType('Socket')" will result in a
        socket type mixin.
      |||
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('hostPath', d.T.string),
        d.arg('path', d.T.string),
        d.arg('readOnly', d.T.bool),
        d.arg('volumeMountMixin', d.T.object),
        d.arg('volumeMixin', d.T.object),
        d.arg('containers', d.T.array),
      ]
    ),
    hostVolumeMount(name, hostPath, path, readOnly=false, volumeMountMixin={}, volumeMixin={}, containers=null, includeInitContainers=false)::
      local addMount(c) = c + (
        if containers == null || std.member(containers, c.name)
        then container.withVolumeMountsMixin(
          volumeMount.new(name, path, readOnly=readOnly) +
          volumeMountMixin,
        )
        else {}
      );
      local volumeMixins = [volume.fromHostPath(name, hostPath) + volumeMixin];

      super.mapContainers(addMount, includeInitContainers=includeInitContainers) +
      if std.objectHas(super.spec, 'template')
      then super.spec.template.spec.withVolumesMixin(volumeMixins)
      else super.spec.jobTemplate.spec.template.spec.withVolumesMixin(volumeMixins),


    '#pvcVolumeMount': d.fn(
      |||
        `hostVolumeMount` mounts a PersistentVolumeClaim by `name` on `path`.

        If `containers` is specified as an array of container names it will only be mounted
        to those containers, otherwise it will be mounted on all containers.

        This helper function can be augmented with a `volumeMixin`. For example,
        passing "k.core.v1.volume.persistentVolumeClaim.withReadOnly(true)" will result in a
        mixin that forces all container mounts to be read-only.
      |||
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('readOnly', d.T.bool),
        d.arg('volumeMountMixin', d.T.object),
        d.arg('volumeMixin', d.T.object),
        d.arg('containers', d.T.array),
      ]
    ),
    pvcVolumeMount(name, path, readOnly=false, volumeMountMixin={}, volumeMixin={}, containers=null, includeInitContainers=false)::
      local addMount(c) = c + (
        if containers == null || std.member(containers, c.name)
        then container.withVolumeMountsMixin(
          volumeMount.new(name, path, readOnly=readOnly) +
          volumeMountMixin,
        )
        else {}
      );
      local volumeMixins = [volume.fromPersistentVolumeClaim(name, name) + volumeMixin];

      super.mapContainers(addMount, includeInitContainers=includeInitContainers) +
      if std.objectHas(super.spec, 'template')
      then super.spec.template.spec.withVolumesMixin(volumeMixins)
      else super.spec.jobTemplate.spec.template.spec.withVolumesMixin(volumeMixins),


    '#secretVolumeMount': d.fn(
      |||
        `secretVolumeMount` mounts a Secret by `name` into all container on `path`.'

        If `containers` is specified as an array of container names it will only be mounted
        to those containers, otherwise it will be mounted on all containers.

        This helper function can be augmented with a `volumeMixin`. For example,
        passing "k.core.v1.volume.secret.withOptional(true)" will result in a
        mixin that allows the secret to be optional.
      |||
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('defaultMode', d.T.string),
        d.arg('volumeMountMixin', d.T.object),
        d.arg('volumeMixin', d.T.object),
        d.arg('containers', d.T.array),
      ]
    ),
    secretVolumeMount(name, path, defaultMode=256, volumeMountMixin={}, volumeMixin={}, containers=null, includeInitContainers=false)::
      local addMount(c) = c + (
        if containers == null || std.member(containers, c.name)
        then container.withVolumeMountsMixin(
          volumeMount.new(name, path) +
          volumeMountMixin,
        )
        else {}
      );
      local volumeMixins = [
        volume.fromSecret(name, secretName=name) +
        volume.secret.withDefaultMode(defaultMode) +
        volumeMixin,
      ];

      super.mapContainers(addMount, includeInitContainers=includeInitContainers) +
      if std.objectHas(super.spec, 'template')
      then super.spec.template.spec.withVolumesMixin(volumeMixins)
      else super.spec.jobTemplate.spec.template.spec.withVolumesMixin(volumeMixins),

    '#secretVolumeMountAnnotated': d.fn(
      'same as `secretVolumeMount`, adding an annotation to force redeploy on change.'
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('defaultMode', d.T.string),
        d.arg('volumeMountMixin', d.T.object),
        d.arg('volumeMixin', d.T.object),
        d.arg('containers', d.T.array),
      ]
    ),
    secretVolumeMountAnnotated(name, path, defaultMode=256, volumeMountMixin={}, volumeMixin={}, containers=null, includeInitContainers=false)::
      local annotations = { ['%s-secret-hash' % name]: std.md5(std.toString(name)) };

      self.secretVolumeMount(name, path, defaultMode, volumeMountMixin, volumeMixin, containers)
      + super.spec.template.metadata.withAnnotationsMixin(annotations),

    '#emptyVolumeMount': d.fn(
      |||
        `emptyVolumeMount` mounts empty volume by `name` into all container on `path`.

        If `containers` is specified as an array of container names it will only be mounted
        to those containers, otherwise it will be mounted on all containers.

        This helper function can be augmented with a `volumeMixin`. For example,
        passing "k.core.v1.volume.emptyDir.withSizeLimit('100Mi')" will result in a
        mixin that limits the size of the volume to 100Mi.
      |||
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('volumeMountMixin', d.T.object),
        d.arg('volumeMixin', d.T.object),
        d.arg('containers', d.T.array),
      ]
    ),
    emptyVolumeMount(name, path, volumeMountMixin={}, volumeMixin={}, containers=null, includeInitContainers=false)::
      local addMount(c) = c + (
        if containers == null || std.member(containers, c.name)
        then container.withVolumeMountsMixin(
          volumeMount.new(name, path) +
          volumeMountMixin,
        )
        else {}
      );
      local volumeMixins = [volume.fromEmptyDir(name) + volumeMixin];

      super.mapContainers(addMount, includeInitContainers=includeInitContainers) +
      if std.objectHas(super.spec, 'template')
      then super.spec.template.spec.withVolumesMixin(volumeMixins)
      else super.spec.jobTemplate.spec.template.spec.withVolumesMixin(volumeMixins),

    '#csiVolumeMount': d.fn(
      |||
        `csiVolumeMount` mounts CSI volume by `name` into all container on `path`.
        If `containers` is specified as an array of container names it will only be mounted
        to those containers, otherwise it will be mounted on all containers.
        This helper function can be augmented with a `volumeMixin`. For example,
        passing "k.core.v1.volume.csi.withReadOnly(false)" will result in a
        mixin that makes the volume writeable.
      |||
      + volumeMountDescription,
      [
        d.arg('name', d.T.string),
        d.arg('path', d.T.string),
        d.arg('driver', d.T.string),
        d.arg('volumeAttributes', d.T.object, {}),
        d.arg('volumeMountMixin', d.T.object),
        d.arg('volumeMixin', d.T.object),
        d.arg('containers', d.T.array),
      ]
    ),
    csiVolumeMount(name, path, driver, volumeAttributes, volumeMountMixin={}, volumeMixin={}, containers=null, includeInitContainers=false)::
      local addMount(c) = c + (
        if containers == null || std.member(containers, c.name)
        then container.withVolumeMountsMixin(
          volumeMount.new(name, path) +
          volumeMountMixin,
        )
        else {}
      );
      local volumeMixins = [volume.fromCsi(name, driver, volumeAttributes) + volumeMixin];

      super.mapContainers(addMount, includeInitContainers=includeInitContainers) +
      if std.objectHas(super.spec, 'template')
      then super.spec.template.spec.withVolumesMixin(volumeMixins)
      else super.spec.jobTemplate.spec.template.spec.withVolumesMixin(volumeMixins),
  },

  batch+: {
    v1+: {
      job+: patch,
      cronJob+: patch,
    },
  },
  apps+: {
    v1+: {
      daemonSet+: patch,
      deployment+: patch,
      replicaSet+: patch,
      statefulSet+: patch,
    },
  },
}
