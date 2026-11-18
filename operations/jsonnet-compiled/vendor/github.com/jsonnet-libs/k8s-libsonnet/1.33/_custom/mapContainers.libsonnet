local d = import 'doc-util/main.libsonnet';

local patch = {
  '#mapContainers': d.fn(
    |||
      `mapContainers` applies the function f to each container.
      It works exactly as `std.map`, but on the containers of this object.

      **Signature of `f`**:
      ```ts
      f(container: Object) Object
      ```
    |||,
    [d.arg('f', d.T.func)]
  ),
  mapContainers(f, includeInitContainers=false):: {
    local podContainers = super.spec.template.spec.containers,
    local podInitContainers = super.spec.template.spec.initContainers,
    spec+: {
      template+: {
        spec+: {
          containers: std.map(f, podContainers),
          [if includeInitContainers then 'initContainers']: std.map(f, podInitContainers),
        },
      },
    },
  },

  '#mapContainersWithName': d.fn('`mapContainersWithName` is like `mapContainers`, but only applies to those containers in the `names` array',
                                 [d.arg('names', d.T.array), d.arg('f', d.T.func)]),
  mapContainersWithName(names, f, includeInitContainers=false)::
    local nameSet = if std.type(names) == 'array' then std.set(names) else std.set([names]);
    local inNameSet(name) = std.length(std.setInter(nameSet, std.set([name]))) > 0;

    self.mapContainers(function(c) if std.objectHas(c, 'name') && inNameSet(c.name) then f(c) else c, includeInitContainers),
};

// batch.job and batch.cronJob have the podSpec at a different location
local cronPatch = patch {
  mapContainers(f, includeInitContainers=false):: {
    local podContainers = super.spec.jobTemplate.spec.template.spec.containers,
    local podInitContainers = super.spec.jobTemplate.spec.template.spec.initContainers,
    spec+: {
      jobTemplate+: {
        spec+: {
          template+: {
            spec+: {
              containers: std.map(f, podContainers),
              [if includeInitContainers then 'initContainers']: std.map(f, podInitContainers),
            },
          },
        },
      },
    },
  },
};

{
  core+: {
    v1+: {
      pod+: patch,
      podTemplate+: patch,
      replicationController+: patch,
    },
  },
  batch+: {
    v1+: {
      job+: patch,
      cronJob+: cronPatch,
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
