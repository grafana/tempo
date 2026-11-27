// This file is generated, do not manually edit.
(import '../metadata.libsonnet')
+ {
  '#': { help: 'grafonnet.apps.playlist.v0alpha1', name: 'v0alpha1' },
  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withSpec(value): {
    spec: value,
  },
  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withSpecMixin(value): {
    spec+: value,
  },
  spec+:
    {
      '#withInterval': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withInterval(value): {
        spec+: {
          interval: value,
        },
      },
      '#withItems': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withItems(value): {
        spec+: {
          items:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withItemsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withItemsMixin(value): {
        spec+: {
          items+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      items+:
        {
          '#': { help: '', name: 'items' },
          '#withType': { 'function': { args: [{ default: null, enums: ['dashboard_by_tag', 'dashboard_by_uid', 'dashboard_by_id'], name: 'value', type: ['string'] }], help: 'type of the item.' } },
          withType(value): {
            type: value,
          },
          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Value depends on type and describes the playlist item.\n - dashboard_by_id: The value is an internal numerical identifier set by Grafana. This\n is not portable as the numerical identifier is non-deterministic between different instances.\n Will be replaced by dashboard_by_uid in the future. (deprecated)\n - dashboard_by_tag: The value is a tag which is set on any number of dashboards. All\n dashboards behind the tag will be added to the playlist.\n - dashboard_by_uid: The value is the dashboard UID' } },
          withValue(value): {
            value: value,
          },
        },
      '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withTitle(value): {
        spec+: {
          title: value,
        },
      },
    },
}
+ {
  '#new': {
    'function': {
      args: [
        {
          default: null,
          enums: null,
          name: 'name',
          type: 'string',
        },
        {
          default: null,
          enums: null,
          name: 'title',
          type: 'string',
        },
      ],
      help: 'Creates a new playlist.Playlist resource.',
    },
  },
  new: function(name, title) {
    apiVersion: 'playlist.grafana.app/playlistv0alpha1',
    kind: 'Playlist',
    metadata+: { name: name },
    spec+: { title: title },
  },
  '#withApiVersion': {
    'function': {
      args: [

      ],
      help: "set the resource's apiVersion",
    },
  },
  withApiVersion: function()
    {
      apiVersion: 'playlist.grafana.app/playlistv0alpha1',
    },
  '#withKind': {
    'function': {
      args: [

      ],
      help: "set the resource's kind",
    },
  },
  withKind: function()
    {
      kind: 'Playlist',
    },
}
