// This file is generated, do not manually edit.
(import '../panel.libsonnet')
+ {
  '#': { help: 'grafonnet.panel.barGauge', name: 'barGauge' },
  panelOptions+:
    {
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'bargauge',
      },
    },
  options+:
    {
      '#withDisplayMode': { 'function': { args: [{ default: 'gradient', enums: ['basic', 'lcd', 'gradient'], name: 'value', type: ['string'] }], help: 'Enum expressing the possible display modes\nfor the bar gauge component of Grafana UI' } },
      withDisplayMode(value='gradient'): {
        options+: {
          displayMode: value,
        },
      },
      '#withLegend': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
      withLegend(value): {
        options+: {
          legend: value,
        },
      },
      '#withLegendMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
      withLegendMixin(value): {
        options+: {
          legend+: value,
        },
      },
      legend+:
        {
          '#withAsTable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withAsTable(value=true): {
            options+: {
              legend+: {
                asTable: value,
              },
            },
          },
          '#withCalcs': { 'function': { args: [{ default: [], enums: null, name: 'value', type: ['array'] }], help: '' } },
          withCalcs(value): {
            options+: {
              legend+: {
                calcs:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withCalcsMixin': { 'function': { args: [{ default: [], enums: null, name: 'value', type: ['array'] }], help: '' } },
          withCalcsMixin(value): {
            options+: {
              legend+: {
                calcs+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withDisplayMode': { 'function': { args: [{ default: 'list', enums: ['list', 'table', 'hidden'], name: 'value', type: ['string'] }], help: 'TODO docs\nNote: "hidden" needs to remain as an option for plugins compatibility' } },
          withDisplayMode(value='list'): {
            options+: {
              legend+: {
                displayMode: value,
              },
            },
          },
          '#withIsVisible': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withIsVisible(value=true): {
            options+: {
              legend+: {
                isVisible: value,
              },
            },
          },
          '#withPlacement': { 'function': { args: [{ default: 'bottom', enums: ['bottom', 'right'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
          withPlacement(value='bottom'): {
            options+: {
              legend+: {
                placement: value,
              },
            },
          },
          '#withShowLegend': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withShowLegend(value=true): {
            options+: {
              legend+: {
                showLegend: value,
              },
            },
          },
          '#withSortBy': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withSortBy(value): {
            options+: {
              legend+: {
                sortBy: value,
              },
            },
          },
          '#withSortDesc': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withSortDesc(value=true): {
            options+: {
              legend+: {
                sortDesc: value,
              },
            },
          },
          '#withWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
          withWidth(value): {
            options+: {
              legend+: {
                width: value,
              },
            },
          },
        },
      '#withMaxVizHeight': { 'function': { args: [{ default: 300, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withMaxVizHeight(value=300): {
        options+: {
          maxVizHeight: value,
        },
      },
      '#withMinVizHeight': { 'function': { args: [{ default: 16, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withMinVizHeight(value=16): {
        options+: {
          minVizHeight: value,
        },
      },
      '#withMinVizWidth': { 'function': { args: [{ default: 8, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withMinVizWidth(value=8): {
        options+: {
          minVizWidth: value,
        },
      },
      '#withNamePlacement': { 'function': { args: [{ default: 'auto', enums: ['auto', 'top', 'left', 'hidden'], name: 'value', type: ['string'] }], help: 'Allows for the bar gauge name to be placed explicitly' } },
      withNamePlacement(value='auto'): {
        options+: {
          namePlacement: value,
        },
      },
      '#withOrientation': { 'function': { args: [{ default: null, enums: ['auto', 'vertical', 'horizontal'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
      withOrientation(value): {
        options+: {
          orientation: value,
        },
      },
      '#withReduceOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
      withReduceOptions(value): {
        options+: {
          reduceOptions: value,
        },
      },
      '#withReduceOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
      withReduceOptionsMixin(value): {
        options+: {
          reduceOptions+: value,
        },
      },
      reduceOptions+:
        {
          '#withCalcs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'When !values, pick one value for the whole field' } },
          withCalcs(value): {
            options+: {
              reduceOptions+: {
                calcs:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withCalcsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'When !values, pick one value for the whole field' } },
          withCalcsMixin(value): {
            options+: {
              reduceOptions+: {
                calcs+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withFields': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Which fields to show.  By default this is only numeric fields' } },
          withFields(value): {
            options+: {
              reduceOptions+: {
                fields: value,
              },
            },
          },
          '#withLimit': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'if showing all values limit' } },
          withLimit(value): {
            options+: {
              reduceOptions+: {
                limit: value,
              },
            },
          },
          '#withValues': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If true show each row value' } },
          withValues(value=true): {
            options+: {
              reduceOptions+: {
                values: value,
              },
            },
          },
        },
      '#withShowUnfilled': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withShowUnfilled(value=true): {
        options+: {
          showUnfilled: value,
        },
      },
      '#withSizing': { 'function': { args: [{ default: 'auto', enums: ['auto', 'manual'], name: 'value', type: ['string'] }], help: 'Allows for the bar gauge size to be set explicitly' } },
      withSizing(value='auto'): {
        options+: {
          sizing: value,
        },
      },
      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
      withText(value): {
        options+: {
          text: value,
        },
      },
      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
      withTextMixin(value): {
        options+: {
          text+: value,
        },
      },
      text+:
        {
          '#withPercentSize': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Explicit percent text size' } },
          withPercentSize(value): {
            options+: {
              text+: {
                percentSize: value,
              },
            },
          },
          '#withTitleSize': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Explicit title text size' } },
          withTitleSize(value): {
            options+: {
              text+: {
                titleSize: value,
              },
            },
          },
          '#withValueSize': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Explicit value text size' } },
          withValueSize(value): {
            options+: {
              text+: {
                valueSize: value,
              },
            },
          },
        },
      '#withValueMode': { 'function': { args: [{ default: 'color', enums: ['color', 'text', 'hidden'], name: 'value', type: ['string'] }], help: 'Allows for the table cell gauge display type to set the gauge mode.' } },
      withValueMode(value='color'): {
        options+: {
          valueMode: value,
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
