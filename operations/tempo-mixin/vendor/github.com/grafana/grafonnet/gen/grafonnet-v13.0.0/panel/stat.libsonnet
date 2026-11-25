// This file is generated, do not manually edit.
(import '../panel.libsonnet')
+ {
  '#': { help: 'grafonnet.panel.stat', name: 'stat' },
  panelOptions+:
    {
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'stat',
      },
    },
  options+:
    {
      '#withColorMode': { 'function': { args: [{ default: 'value', enums: ['value', 'background', 'background_solid', 'none'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
      withColorMode(value='value'): {
        options+: {
          colorMode: value,
        },
      },
      '#withGraphMode': { 'function': { args: [{ default: 'area', enums: ['none', 'line', 'area'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
      withGraphMode(value='area'): {
        options+: {
          graphMode: value,
        },
      },
      '#withJustifyMode': { 'function': { args: [{ default: 'auto', enums: ['auto', 'center'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
      withJustifyMode(value='auto'): {
        options+: {
          justifyMode: value,
        },
      },
      '#withOrientation': { 'function': { args: [{ default: null, enums: ['auto', 'vertical', 'horizontal'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
      withOrientation(value): {
        options+: {
          orientation: value,
        },
      },
      '#withPercentChangeColorMode': { 'function': { args: [{ default: 'standard', enums: ['standard', 'inverted', 'same_as_value'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
      withPercentChangeColorMode(value='standard'): {
        options+: {
          percentChangeColorMode: value,
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
      '#withShowPercentChange': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withShowPercentChange(value=true): {
        options+: {
          showPercentChange: value,
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
      '#withTextMode': { 'function': { args: [{ default: 'auto', enums: ['auto', 'value', 'value_and_name', 'name', 'none'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
      withTextMode(value='auto'): {
        options+: {
          textMode: value,
        },
      },
      '#withWideLayout': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withWideLayout(value=true): {
        options+: {
          wideLayout: value,
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
