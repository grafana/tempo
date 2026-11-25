// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.apps.metadata', name: 'metadata' },
  '#withMetadata': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withMetadata(value): {
    metadata: value,
  },
  '#withMetadataMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withMetadataMixin(value): {
    metadata+: value,
  },
  metadata+:
    {
      '#withAnnotations': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withAnnotations(value): {
        metadata+: {
          annotations: value,
        },
      },
      '#withAnnotationsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withAnnotationsMixin(value): {
        metadata+: {
          annotations+: value,
        },
      },
      '#withCreationTimestamp': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withCreationTimestamp(value): {
        metadata+: {
          creationTimestamp: value,
        },
      },
      '#withDeletionTimestamp': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withDeletionTimestamp(value): {
        metadata+: {
          deletionTimestamp: value,
        },
      },
      '#withGeneration': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withGeneration(value): {
        metadata+: {
          generation: value,
        },
      },
      '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withLabels(value): {
        metadata+: {
          labels: value,
        },
      },
      '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withLabelsMixin(value): {
        metadata+: {
          labels+: value,
        },
      },
      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withName(value): {
        metadata+: {
          name: value,
        },
      },
      '#withNamespace': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withNamespace(value): {
        metadata+: {
          namespace: value,
        },
      },
      '#withResourceVersion': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withResourceVersion(value): {
        metadata+: {
          resourceVersion: value,
        },
      },
      '#withUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withUid(value): {
        metadata+: {
          uid: value,
        },
      },
      '#withUpdateTimestamp': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withUpdateTimestamp(value): {
        metadata+: {
          updateTimestamp: value,
        },
      },
    },
}
