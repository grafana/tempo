{
  local this = self,
  _config+:: {
    vault: {
      port: 8200,
      clusterPort: 8201,
      logLevel: 'info',
      replicas: 3,
      config: {
        // A YAML representation of a final vault config.json file.
        // See https://www.vaultproject.io/docs/configuration/ for more information.
        default_listener:: {
          tcp: {
            address: '[::]:%s' % this._config.vault.port,
            cluster_address: '[::]:%s' % this._config.vault.clusterPort,
          },
        },
        listener+: [this._config.vault.config.default_listener],
        disable_mlock: true,
        ui: true,
      },
    },
  },

  _images+:: {
    vault: 'vault:1.6.3',
  },
}
