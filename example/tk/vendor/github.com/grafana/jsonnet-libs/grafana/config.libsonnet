{
  _images+:: {
    grafana: 'grafana/grafana:8.2.5',
  },

  _config+:: {
    replicas: 1,
    rootUrl: '',
    provisioningDir: '/etc/grafana/provisioning',
    port: 80,
    containerPort: 3000,

    // Split configmaps into multiple files
    // 100000 is a good default, because a `kubectl` client-side apply cannot exceed 256kB,
    //   and the size of the request is doubled when setting the last-applied configuration
    //   https://github.com/kubernetes/kubectl/issues/712
    // For serverside applies, it can be increased, but keep in mind the 1MB limit of etcd
    //   https://github.com/kubernetes/kubernetes/issues/19781
    //
    // The meaning of `configmap_shard_size` depends on `configmap_binpack`:
    //   * binpack = false (default, "balanced"): a *target average* shard size. The
    //     number of shards is `ceil(total / configmap_shard_size)` and dashboards are
    //     split across them by count, so an individual shard MAY exceed this value.
    //   * binpack = true: a *hard per-ConfigMap byte cap* enforced by First-Fit
    //     Decreasing bin packing (see configmaps.libsonnet).
    configmap_shard_size: 100000,

    // Opt in to First-Fit Decreasing (FFD) bin packing for dashboard ConfigMaps.
    // When true, `configmap_shard_size` becomes a hard per-ConfigMap byte budget:
    // dashboards are packed largest-first into the fewest shards that keep every
    // shard at or below the budget, so a shard can never silently exceed
    // Kubernetes' 1 MiB ConfigMap limit. Leave headroom below 1 MiB for ConfigMap
    // keys/metadata/encoding when choosing the budget (~900000 is a safe cap).
    //
    // Caveats:
    //   * `configmap_shard_size` changes meaning (target average -> hard cap), see above.
    //   * Enabling this (or later changing the budget) reassigns dashboards across
    //     shards, which bumps the `grafana-dashboards-hash` annotation and triggers a
    //     one-time Grafana reload of the affected folders. This is expected and harmless.
    //   * A single dashboard larger than the budget cannot be split across ConfigMaps
    //     and is placed in its own (over-budget) shard as a best effort.
    // Defaults to false to preserve the existing balanced sharding behaviour for
    // current consumers.
    configmap_binpack: false,

    labels+: {
      dashboards: {},
      datasources: {},
      notificationChannels: {},
    },
    grafana_ini+: {
      sections+: {
        server: {
          http_port: $._config.containerPort,
          router_logging: true,
          root_url: $._config.rootUrl,
        },
        analytics: {
          reporting_enabled: false,
        },
        users: {
          default_theme: 'light',
        },
        'log.frontend': {
          enabled: true,
        },
      },
    },
  },

  withImage(image):: {
    _images+:: {
      grafana: image,
    },
  },

  withGrafanaIniConfig(config):: {
    _config+:: {
      grafana_ini+: config,
    },
  },

  withTheme(theme):: self.withGrafanaIniConfig({
    sections+: {
      users+: {
        default_theme: theme,
      },
    },
  }),

  // NOTE: This method will be enforced to only have `org_role` set 'Viewer'
  // Deprecated: Deprecated from G12
  withAnonymous():: self.withGrafanaIniConfig({
    assert std.trace('DEPRECATED: Anonymous users will be enforced to Viewer from Grafana v12. Please use withAnonymousAuth', true),
    sections+: {
      'auth.anonymous': {
        enabled: true,
        org_role: 'Admin',
      },
    },
  }),

  // NOTE: new method
  withAnonymousAuth():: self.withGrafanaIniConfig({
    sections+: {
      'auth.anonymous': {
        enabled: true,
        org_role: 'Viewer',
      },
    },
  }),

  withEnterpriseLicenseText(text):: self.withGrafanaIniConfig({
    sections+: {
      enterprise+: {
        license_text: text,
      },
    },
  }),

  withEnterpriseLicensePath(path):: self.withGrafanaIniConfig({
    sections+: {
      enterprise+: {
        license_path: path,
      },
    },
  }),

  withRootUrl(url):: {
    _config+:: {
      rootUrl: url,
    },
  },

  withReplicas(replicas):: {
    _config+:: {
      replicas: replicas,
    },
  },
}
