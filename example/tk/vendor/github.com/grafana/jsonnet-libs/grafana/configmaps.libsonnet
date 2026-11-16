local k = import 'ksonnet-util/kausal.libsonnet';
local configMap = k.core.v1.configMap;
local deployment = k.apps.v1.deployment;

{
  // grafana_ini configmap
  grafana_ini_config_map:
    configMap.new('grafana-config') +
    configMap.withData({ 'grafana.ini': std.manifestIni($._config.grafana_ini) }),

  // datasource provisioning configmap
  grafana_datasource_config_map:
    configMap.new('grafana-datasources') +
    configMap.withDataMixin({
      ['%s.yml' % name]: k.util.manifestYaml({
        apiVersion: 1,
        datasources: [$.grafanaDatasources[name]],
      })
      for name in std.objectFields($.grafanaDatasources)
    })
    + configMap.mixin.metadata.withLabels($._config.labels.datasources),

  // notification channel provisioning configmap
  notification_channel_config_map:
    configMap.new('grafana-notification-channels') +
    configMap.withDataMixin({
      [name]: k.util.manifestYaml({
        notifiers: [
          $.grafanaNotificationChannels[name],
        ],
      })
      for name in std.objectFields($.grafanaNotificationChannels)
    }) +
    configMap.mixin.metadata.withLabels($._config.labels.notificationChannels),

  local prefix(name) = if name == '' then 'dashboards' else 'dashboards-%s' % name,

  // dashboard provisioning configmaps
  dashboard_provisioning_config_map:
    configMap.new('grafana-dashboard-provisioning') +
    configMap.withData({
      'dashboards.yml': k.util.manifestYaml({
        apiVersion: 1,
        providers: [
          {
            name: prefix($.grafanaDashboardFolders[name].id),
            orgId: 1,
            folder: $.grafanaDashboardFolders[name].name,
            type: 'file',
            disableDeletion: true,
            editable: false,
            options: {
              path: '/grafana/%s' % prefix($.grafanaDashboardFolders[name].id),
            },
          }
          for name in std.objectFields($.grafanaDashboardFolders)
        ],
      }),
    }),

  // Dashboard JSON configmaps.
  //
  // Two sharding strategies are available, selected by `$._config.configmap_binpack`:
  //   * false (default): the balanced strategy — `configmap_shard_size` is a target
  //     average and dashboards are split across `ceil(total / size)` shards by count.
  //     An individual shard may exceed `configmap_shard_size`.
  //   * true: First-Fit Decreasing (FFD) bin packing — `configmap_shard_size` is a
  //     hard per-ConfigMap byte budget; no shard exceeds it (barring a single
  //     dashboard that alone is larger than the budget).
  // See config.libsonnet for the full description and caveats.
  //
  // Both strategies return the same shape:
  //   { '<prefix>-<index>': { '<dashboard>.json': <content>, ... }, ... }

  // Balanced strategy: fix the shard count from the target average size, then
  // balance shards by pairing the smallest dashboards with the biggest ones.
  // `configmap_shard_size` is a target average, not a hard cap.
  local calculateShardsBalanced(folder) =
    // Sort dashboards descending by size
    local dashboards = std.sort([
      {
        name: if std.endsWith(name, '.json') then name else '%s.json' % name,
        content: std.toString(folder.dashboards[name]),
      }
      for name in std.objectFields(folder.dashboards)
    ], function(d) -std.length(d.content));
    local count = std.length(dashboards);

    // Shard configmaps at around 100kB per shard
    local totalCharacters = std.foldl(function(x, y) x + y, [std.length(d.content) for d in dashboards], 0);
    local shardCount = std.min(count, std.ceil(totalCharacters / $._config.configmap_shard_size));
    {
      // Calculate the number of dashboards per shard
      // This is skewed towards tail dashboards (smallest ones)
      // For example, if we need 3 per shard, it will be 1 big and 2 smalls
      local perShard = std.floor(count / shardCount),
      local perShardHead = std.floor(perShard / 2),
      local perShardTail = std.ceil(perShard / 2),

      // perShard is a floor, so we can have a remainder
      // It is taken from the end of the array (smallest dashboards)
      // At the end of the loop, we add the remainder to the last shard
      local maxTail = shardCount * perShard,
      local remainder = count - maxTail,
      ['%s-%d' % [prefix(folder.id), shard]]+:
        local head = shard * perShardHead;
        local nextHead = head + perShardHead;
        local tail = maxTail - (shard * perShardTail);
        local nextTail = tail - perShardTail;

        {
          [dashboard.name]: dashboard.content
          for dashboard in
            // Dashboards from beginning + from end + remainder for last shard
            std.slice(dashboards, head, nextHead, 1)
            + std.slice(dashboards, nextTail, tail, 1)
            + if shard == shardCount - 1 && remainder > 0 then std.slice(dashboards, maxTail, maxTail + remainder, 1) else []
        }
      for shard in std.range(0, shardCount - 1)
      if count > 0
    },

  // FFD strategy: `configmap_shard_size` is a hard per-ConfigMap byte budget.
  // Dashboards are sorted largest-first and each is placed into the first shard
  // that still has room, opening a new shard only when none fits. This keeps
  // every shard at or below the budget while filling shards as densely as
  // possible, so we emit close to the minimum number of ConfigMaps and never
  // blow past Kubernetes' hard 1 MiB ConfigMap limit. A single dashboard larger
  // than the budget cannot be split, so it is placed in its own (over-budget)
  // shard as a best effort.
  local calculateShardsBinpack(folder) =
    local capacity = $._config.configmap_shard_size;

    // Serialize each dashboard once and sort descending by size
    // (the "Decreasing" in First-Fit Decreasing).
    local mkItem(name) =
      local content = std.toString(folder.dashboards[name]);
      {
        name: if std.endsWith(name, '.json') then name else '%s.json' % name,
        content: content,
        size: std.length(content),
      };
    local dashboards = std.sort(
      [mkItem(name) for name in std.objectFields(folder.dashboards)],
      function(d) -d.size,
    );

    // Fold each dashboard into the first shard that still has room.
    // A shard (bin) is { size: <bytes used>, items: [ <dashboard>, ... ] }.
    local bins = std.foldl(
      function(acc, d)
        local fits = std.filter(
          function(i) acc[i].size + d.size <= capacity,
          std.range(0, std.length(acc) - 1),
        );
        if std.length(fits) == 0 then
          // No existing shard fits: open a new one.
          acc + [{ size: d.size, items: [d] }]
        else
          // Place into the first fitting shard (lowest index).
          local target = fits[0];
          [
            if i == target
            then { size: acc[i].size + d.size, items: acc[i].items + [d] }
            else acc[i]
            for i in std.range(0, std.length(acc) - 1)
          ],
      dashboards,
      [],
    );

    // Emit shards as `<prefix>-<index>` -> { <dashboard name>: <content> }.
    // An empty folder yields no shards.
    {
      ['%s-%d' % [prefix(folder.id), i]]: {
        [item.name]: item.content
        for item in bins[i].items
      }
      for i in std.range(0, std.length(bins) - 1)
    },

  // Dispatch on the configured strategy. Kept as a single entry point so the
  // ConfigMap emission and the volume mounts (both call this) always agree on
  // shard names.
  local calculateShards(folder) =
    if $._config.configmap_binpack
    then calculateShardsBinpack(folder)
    else calculateShardsBalanced(folder),


  local shardedConfigMaps(folder) =
    local shards = calculateShards(folder);
    {
      [shardName]+:
        configMap.new(shardName) +
        configMap.withDataMixin(shards[shardName])
        + configMap.mixin.metadata.withLabels($._config.labels.dashboards)
      for shardName in std.objectFields(shards)
    },

  dashboard_folders_config_maps: std.foldl(
    function(acc, name)
      acc + shardedConfigMaps($.grafanaDashboardFolders[name]),
    std.objectFields($.grafanaDashboardFolders),
    {},
  ),

  // Helper to mount a variable number of sharded config maps.
  local shardedMounts(folder) =
    local shards = calculateShards(folder);
    [
      k.util.volumeMountItem(shard, '/grafana/%s/%s' % [prefix(folder.id), shard])
      for shard in std.objectFields(shards)
    ],

  // configmap mounts for use within statefulset/deployment
  configmap_mounts::
    local mounts =
      [
        k.util.configMapVolumeMountItem($.grafana_ini_config_map, '/etc/grafana-config'),
        k.util.configMapVolumeMountItem($.dashboard_provisioning_config_map, '%(provisioningDir)s/dashboards' % $._config),
        k.util.configMapVolumeMountItem($.grafana_datasource_config_map, '%(provisioningDir)s/datasources' % $._config),
        k.util.configMapVolumeMountItem($.notification_channel_config_map, '%(provisioningDir)s/notifiers' % $._config),
      ]
      + std.flattenArrays([
        shardedMounts($.grafanaDashboardFolders[folder])
        for folder in std.objectFields($.grafanaDashboardFolders)
      ]);

    k.util.volumeMounts(mounts)
    + deployment.mixin.spec.template.metadata.withAnnotationsMixin({
      'grafana-dashboards-hash': std.md5(std.toString($.grafanaDashboardFolders)),
    }),
}
