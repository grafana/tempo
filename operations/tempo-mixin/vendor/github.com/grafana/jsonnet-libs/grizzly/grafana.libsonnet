local resource = import 'resource.libsonnet';
local util = import 'util.libsonnet';

{
  getFolder(main):: util.get(main, 'grafanaDashboardFolder', 'General'),

  fromMap(dashboards, folder):: {
    [k]: util.makeResource(
      'Dashboard',
      std.strReplace(std.strReplace(std.strReplace(k, '.json', ''), '.yaml', ''), '.yml', ''),
      dashboards[k],
      { folder: folder }
    )
    for k in std.objectFields(dashboards)
  },

  fromMixins(mixins):: {
    [key]:
      local mixin = mixins[key];
      {
        local folder = util.get(mixin, 'grafanaDashboardFolder', 'General'),
        [k]: util.makeResource(
          'Dashboard',
          std.strReplace(std.strReplace(std.strReplace(k, '.json', ''), '.yaml', ''), '.yml', ''),
          mixin.grafanaDashboards[k],
          { folder: folder }
        )
        for k in std.objectFieldsAll(util.get(mixin, 'grafanaDashboards', {}))
      }
    for key in std.objectFields(mixins)
  },

  dashboard: {
    new(name, dashboard_json)::
      resource.new('Dashboard', name)
      + resource.withSpec(dashboard_json),
  },

  folder: {
    new(name, title)::
      resource.new('DashboardFolder', name)
      + resource.withSpec({
        title: title,
      }),
  },

  datasource: {
    new(name, datasource_json)::
      resource.new('Datasource', name)
      + resource.withSpec(datasource_json),
  },
}
