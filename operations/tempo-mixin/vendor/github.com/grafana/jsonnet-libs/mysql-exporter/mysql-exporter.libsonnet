local k = import 'ksonnet-util/kausal.libsonnet';
local container = k.core.v1.container;
local mysqld_exporter = import 'github.com/grafana/jsonnet-libs/mysqld-exporter/main.libsonnet';
local exporter = std.trace('Deprecated: please consider switching to github.com/grafana/jsonnet-libs/mysqld-exporter', mysqld_exporter);

{
  local this = self,
  image:: 'prom/mysqld-exporter:v0.13.0-rc.0',
  mysql_fqdn:: '%s.%s.svc.cluster.local.' % [
    this._config.deployment_name,
    this._config.namespace,
  ],

  _config:: {
    mysql_user: error 'must specify mysql user',
    deployment_name: error 'must specify deployment name',
    namespace: error 'must specify namespace',
  },

  mysql_additional_env_list:: [],

  mysql_exporter::
    exporter.new(
      '%s-mysql-exporter' % this._config.deployment_name,
      this._config.mysql_user,
      this.mysql_fqdn,
      3306,
      this.image,
    )
    + (
      if 'mysql_password' in this._config
      then exporter.withPassword(this._config.mysql_password)
      else {}
    )
    + {
      container+::
        container.withEnvMixin(this.mysql_additional_env_list) +
        this.mysqld_exporter_container,

      deployment+: this.mysql_exporter_deployment,
      service+: this.mysql_exporter_deployment_service,
    },

  // Hidden now, covered by this.mysql_exporter
  mysqld_exporter_container::
    if 'mysqld_exporter_container' in super
    then super.mysqld_exporter_container
    else {},
  mysql_exporter_deployment::
    if 'mysql_exporter_deployment' in super
    then super.mysql_exporter_deployment
    else {},
  mysql_exporter_deployment_service::
    if 'mysql_exporter_deployment_service' in super
    then super.mysql_exporter_deployment_service
    else {},

  withSecretPassword(name, key='password'):: {
    mysql_exporter+: exporter.withPasswordSecretRef(name, key),
  },
}
