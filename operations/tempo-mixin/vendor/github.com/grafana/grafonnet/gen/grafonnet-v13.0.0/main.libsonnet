// This file is generated, do not manually edit.
{
  '#': {
    filename: 'main.libsonnet',
    help: 'Jsonnet library for rendering Grafana resources\n## Install\n\n```\njb install github.com/grafana/grafonnet/gen/grafonnet-v13.0.0@main\n```\n\n## Usage\n\n```jsonnet\nlocal grafonnet = import "github.com/grafana/grafonnet/gen/grafonnet-v13.0.0/main.libsonnet"\n```\n',
    'import': 'github.com/grafana/grafonnet/gen/grafonnet-v13.0.0/main.libsonnet',
    installTemplate: '\n## Install\n\n```\njb install %(url)s@%(version)s\n```\n',
    name: 'grafonnet',
    url: 'github.com/grafana/grafonnet/gen/grafonnet-v13.0.0',
    usageTemplate: '\n## Usage\n\n```jsonnet\nlocal %(name)s = import "%(import)s"\n```\n',
    version: 'main',
  },
  team: import 'team.libsonnet',
  publicdashboard: import 'publicdashboard.libsonnet',
  preferences: import 'preferences.libsonnet',
  librarypanel: import 'librarypanel.libsonnet',
  dashboard: import 'dashboard.libsonnet',
  folder: import 'folder.libsonnet',
  panel: import 'panelindex.libsonnet',
  query: import 'query.libsonnet',
  util: import 'custom/util/main.libsonnet',
  alerting: import 'alerting.libsonnet',
  apps: import 'apps.libsonnet',
}
