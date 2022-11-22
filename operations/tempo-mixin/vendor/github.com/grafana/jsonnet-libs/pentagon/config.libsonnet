{
  local this = self,

  _config+: {
    namespace: error 'must define namespace',
    cluster_name: error 'must define cluster_name',
    pentagon+: {
      name: 'pentagon',
      refresh: '15m',
      vault_address: error 'must provide vault_address',
      vault_auth_path: 'auth/kubernetes/%s' % this._config.cluster_name,
      vault_role: this._config.cluster_name,
    },
  },

  _images+:: {
    pentagon: 'grafana/pentagon:59',
  },

  pentagonKVMapping(path, secret, type='kv-v2', keepLabels=false, keepAnnotations=false):: {
    vaultPath: path,
    secretName: secret,
    vaultEngineType: type,
    // We don't need to add the options if they aren't true, they default to false
    [if keepLabels then 'keepLabels']: true,
    [if keepAnnotations then 'keepAnnotations']: true,
  },

  addPentagonMapping(path, secret, type='kv-v2', keepLabels=false, keepAnnotations=false):: {
    pentagon_mappings_map+: {
      [secret]+: this.pentagonKVMapping(path, secret, type, keepLabels, keepAnnotations),
    },
  },

  pentagon_mappings+:: [],
  local pentagon_mappings_with_index = std.mapWithIndex(function(i, x) x { idx:: i }, this.pentagon_mappings),
  pentagon_mappings_map+::
    // a list like this.pentagon_mappings could have duplicates, using a map we can
    // dedupe them base on a unique key (secretName in this case)
    std.foldl(
      function(obj, mapping)
        obj {
          [mapping.secretName]+: mapping,
        },
      pentagon_mappings_with_index,
      {}
    ),

  pentagonConfig:: {
    vault: {
      url: this._config.pentagon.vault_address,
      authType: 'kubernetes',
      authPath: this._config.pentagon.vault_auth_path,
      defaultEngineType: 'kv',
      role: this._config.pentagon.vault_role,
    },
    namespace: this._config.namespace,
    label: this._config.pentagon.vault_role,
    daemon: true,
    refresh: this._config.pentagon.refresh,
    mappings: std.sort([
      this.pentagon_mappings_map[m]
      for m in std.objectFields(this.pentagon_mappings_map)
    ], function(e) if std.objectHasAll(e, 'idx') then e.idx else 0),
  },
}
