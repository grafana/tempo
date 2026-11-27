// This file is generated, do not manually edit.
(import '../metadata.libsonnet')
+ {
  '#': { help: 'grafonnet.apps.folder.v1', name: 'v1' },
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
      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withDescription(value): {
        spec+: {
          description: value,
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
      help: 'Creates a new folder.Folder resource.',
    },
  },
  new: function(name, title) {
    apiVersion: 'folder.grafana.app/v1',
    kind: 'Folder',
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
      apiVersion: 'folder.grafana.app/v1',
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
      kind: 'Folder',
    },
}
