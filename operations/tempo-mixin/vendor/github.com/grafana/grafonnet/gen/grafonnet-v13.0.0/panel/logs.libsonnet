// This file is generated, do not manually edit.
(import '../panel.libsonnet')
+ {
  '#': { help: 'grafonnet.panel.logs', name: 'logs' },
  panelOptions+:
    {
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'logs',
      },
    },
  options+:
    {
      '#withDedupStrategy': { 'function': { args: [{ default: null, enums: ['none', 'exact', 'numbers', 'signature'], name: 'value', type: ['string'] }], help: '' } },
      withDedupStrategy(value): {
        options+: {
          dedupStrategy: value,
        },
      },
      '#withDetailsMode': { 'function': { args: [{ default: null, enums: ['inline', 'sidebar'], name: 'value', type: ['string'] }], help: '' } },
      withDetailsMode(value): {
        options+: {
          detailsMode: value,
        },
      },
      '#withEnableInfiniteScrolling': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withEnableInfiniteScrolling(value=true): {
        options+: {
          enableInfiniteScrolling: value,
        },
      },
      '#withEnableLogDetails': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withEnableLogDetails(value=true): {
        options+: {
          enableLogDetails: value,
        },
      },
      '#withFontSize': { 'function': { args: [{ default: null, enums: ['default', 'small'], name: 'value', type: ['string'] }], help: '' } },
      withFontSize(value): {
        options+: {
          fontSize: value,
        },
      },
      '#withPrettifyLogMessage': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withPrettifyLogMessage(value=true): {
        options+: {
          prettifyLogMessage: value,
        },
      },
      '#withShowCommonLabels': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withShowCommonLabels(value=true): {
        options+: {
          showCommonLabels: value,
        },
      },
      '#withShowControls': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Display controls to jump to the last or first log line, and filters by log level.' } },
      withShowControls(value=true): {
        options+: {
          showControls: value,
        },
      },
      '#withShowFieldSelector': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Show a component to manage the displayed fields from the logs.' } },
      withShowFieldSelector(value=true): {
        options+: {
          showFieldSelector: value,
        },
      },
      '#withShowLabels': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withShowLabels(value=true): {
        options+: {
          showLabels: value,
        },
      },
      '#withShowLogContextToggle': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withShowLogContextToggle(value=true): {
        options+: {
          showLogContextToggle: value,
        },
      },
      '#withShowTime': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withShowTime(value=true): {
        options+: {
          showTime: value,
        },
      },
      '#withSortOrder': { 'function': { args: [{ default: null, enums: ['Descending', 'Ascending'], name: 'value', type: ['string'] }], help: '' } },
      withSortOrder(value): {
        options+: {
          sortOrder: value,
        },
      },
      '#withSyntaxHighlighting': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Use a predefined coloring scheme to highlight relevant parts of the log lines.' } },
      withSyntaxHighlighting(value=true): {
        options+: {
          syntaxHighlighting: value,
        },
      },
      '#withWrapLogMessage': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withWrapLogMessage(value=true): {
        options+: {
          wrapLogMessage: value,
        },
      },
    },
}
+ {
  panelOptions+: {
    '#withType':: {
      ignore: true,
    },
  },
}
