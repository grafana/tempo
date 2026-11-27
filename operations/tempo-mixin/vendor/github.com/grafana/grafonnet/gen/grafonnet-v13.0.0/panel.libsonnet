// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.panel', name: 'panel' },
  panelOptions+:
    {
      '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Panel title.' } },
      withTitle(value): {
        title: value,
      },
      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Panel description.' } },
      withDescription(value): {
        description: value,
      },
      '#withTransparent': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether to display the panel without a background.' } },
      withTransparent(value=true): {
        transparent: value,
      },
      '#withLinks': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Panel links.' } },
      withLinks(value): {
        links:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withLinksMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Panel links.' } },
      withLinksMixin(value): {
        links+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withMaxPerRow': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Option for repeated panels that controls max items per row\nOnly relevant for horizontally repeated panels' } },
      withMaxPerRow(value): {
        maxPerRow: value,
      },
      '#withRepeat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Name of template variable to repeat for.' } },
      withRepeat(value): {
        repeat: value,
      },
      '#withRepeatDirection': { 'function': { args: [{ default: 'h', enums: ['h', 'v'], name: 'value', type: ['string'] }], help: "Direction to repeat in if 'repeat' is set.\n`h` for horizontal, `v` for vertical." } },
      withRepeatDirection(value='h'): {
        repeatDirection: value,
      },
      '#withPluginVersion': { 'function': { args: [], help: '' } },
      withPluginVersion(): {
        pluginVersion: 'v13.0.0',
      },
      '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The panel plugin type id. This is used to find the plugin to display the panel.' } },
      withType(value): {
        type: value,
      },
      link+:
        {
          '#': { help: '', name: 'link' },
          '#withAsDropdown': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If true, all dashboards links will be displayed in a dropdown. If false, all dashboards links will be displayed side by side. Only valid if the type is dashboards' } },
          withAsDropdown(value=true): {
            asDropdown: value,
          },
          '#withIcon': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Icon name to be displayed with the link' } },
          withIcon(value): {
            icon: value,
          },
          '#withIncludeVars': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If true, includes current template variables values in the link as query params' } },
          withIncludeVars(value=true): {
            includeVars: value,
          },
          '#withKeepTime': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If true, includes current time range in the link as query params' } },
          withKeepTime(value=true): {
            keepTime: value,
          },
          '#withPlacement': { 'function': { args: [{ default: 'inControlsMenu', enums: null, name: 'value', type: ['string'] }], help: 'Placement can be used to display the link somewhere else on the dashboard other than above the visualisations.' } },
          withPlacement(value='inControlsMenu'): {
            placement: value,
          },
          '#withPlacementMixin': { 'function': { args: [{ default: 'inControlsMenu', enums: null, name: 'value', type: ['string'] }], help: 'Placement can be used to display the link somewhere else on the dashboard other than above the visualisations.' } },
          withPlacementMixin(value): {
            placement+: value,
          },
          placement+:
            {
              '#withDashboardLinkPlacement': { 'function': { args: [{ default: null, enums: ['inControlsMenu'], name: 'value', type: ['string'] }], help: 'Dashboard Link placement. Defines where the link should be displayed.\n- "inControlsMenu" renders the link in bottom part of the dashboard controls dropdown menu' } },
              withDashboardLinkPlacement(value): {
                placement+: {
                  DashboardLinkPlacement: value,
                },
              },
            },
          '#withTags': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'List of tags to limit the linked dashboards. If empty, all dashboards will be displayed. Only valid if the type is dashboards' } },
          withTags(value): {
            tags:
              (if std.isArray(value)
               then value
               else [value]),
          },
          '#withTagsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'List of tags to limit the linked dashboards. If empty, all dashboards will be displayed. Only valid if the type is dashboards' } },
          withTagsMixin(value): {
            tags+:
              (if std.isArray(value)
               then value
               else [value]),
          },
          '#withTargetBlank': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If true, the link will be opened in a new tab' } },
          withTargetBlank(value=true): {
            targetBlank: value,
          },
          '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Title to display with the link' } },
          withTitle(value): {
            title: value,
          },
          '#withTooltip': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Tooltip to display when the user hovers their mouse over it' } },
          withTooltip(value): {
            tooltip: value,
          },
          '#withType': { 'function': { args: [{ default: null, enums: ['link', 'dashboards'], name: 'value', type: ['string'] }], help: 'Dashboard Link type. Accepted values are dashboards (to refer to another dashboard) and link (to refer to an external resource)' } },
          withType(value): {
            type: value,
          },
          '#withUrl': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Link URL. Only required/valid if the type is link' } },
          withUrl(value): {
            url: value,
          },
        },
    },
  queryOptions+:
    {
      '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The datasource used in all targets.' } },
      withDatasource(value): {
        datasource: value,
      },
      '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The datasource used in all targets.' } },
      withDatasourceMixin(value): {
        datasource+: value,
      },
      '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'The maximum number of data points that the panel queries are retrieving.' } },
      withMaxDataPoints(value): {
        maxDataPoints: value,
      },
      '#withInterval': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The min time interval setting defines a lower limit for the $__interval and $__interval_ms variables.\nThis value must be formatted as a number followed by a valid time\nidentifier like: "40s", "3d", etc.\nSee: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options' } },
      withInterval(value): {
        interval: value,
      },
      '#withQueryCachingTTL': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Overrides the data source configured time-to-live for a query cache item in milliseconds' } },
      withQueryCachingTTL(value): {
        queryCachingTTL: value,
      },
      '#withTimeFrom': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Overrides the relative time range for individual panels,\nwhich causes them to be different than what is selected in\nthe dashboard time picker in the top-right corner of the dashboard. You can use this to show metrics from different\ntime periods or days on the same dashboard.\nThe value is formatted as time operation like: `now-5m` (Last 5 minutes), `now/d` (the day so far),\n`now-5d/d`(Last 5 days), `now/w` (This week so far), `now-2y/y` (Last 2 years).\nNote: Panel time overrides have no effect when the dashboard’s time range is absolute.\nSee: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options' } },
      withTimeFrom(value): {
        timeFrom: value,
      },
      '#withTimeShift': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Overrides the time range for individual panels by shifting its start and end relative to the time picker.\nFor example, you can shift the time range for the panel to be two hours earlier than the dashboard time picker setting `2h`.\nNote: Panel time overrides have no effect when the dashboard’s time range is absolute.\nSee: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options' } },
      withTimeShift(value): {
        timeShift: value,
      },
      '#withHideTimeOverride': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Controls if the timeFrom or timeShift overrides are shown in the panel header' } },
      withHideTimeOverride(value=true): {
        hideTimeOverride: value,
      },
      '#withTargets': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Depends on the panel plugin. See the plugin documentation for details.' } },
      withTargets(value): {
        targets:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withTargetsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Depends on the panel plugin. See the plugin documentation for details.' } },
      withTargetsMixin(value): {
        targets+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withTransformations': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'List of transformations that are applied to the panel data before rendering.\nWhen there are multiple transformations, Grafana applies them in the order they are listed.\nEach transformation creates a result set that then passes on to the next transformation in the processing pipeline.' } },
      withTransformations(value): {
        transformations:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withTransformationsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'List of transformations that are applied to the panel data before rendering.\nWhen there are multiple transformations, Grafana applies them in the order they are listed.\nEach transformation creates a result set that then passes on to the next transformation in the processing pipeline.' } },
      withTransformationsMixin(value): {
        transformations+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      transformation+:
        {
          '#': { help: '', name: 'transformation' },
          '#withDisabled': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Disabled transformations are skipped' } },
          withDisabled(value=true): {
            disabled: value,
          },
          '#withFilter': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
          withFilter(value): {
            filter: value,
          },
          '#withFilterMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
          withFilterMixin(value): {
            filter+: value,
          },
          filter+:
            {
              '#withId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: 'The matcher id. This is used to find the matcher implementation from registry.' } },
              withId(value=''): {
                filter+: {
                  id: value,
                },
              },
              '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
              withOptions(value): {
                filter+: {
                  options: value,
                },
              },
              '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
              withOptionsMixin(value): {
                filter+: {
                  options+: value,
                },
              },
            },
          '#withId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Unique identifier of transformer' } },
          withId(value): {
            id: value,
          },
          '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Options to be passed to the transformer\nValid options depend on the transformer id' } },
          withOptions(value): {
            options: value,
          },
          '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Options to be passed to the transformer\nValid options depend on the transformer id' } },
          withOptionsMixin(value): {
            options+: value,
          },
          '#withTopic': { 'function': { args: [{ default: null, enums: ['series', 'annotations', 'alertStates'], name: 'value', type: ['string'] }], help: 'Where to pull DataFrames from as input to transformation\nreplaced with common.DataTopic' } },
          withTopic(value): {
            topic: value,
          },
        },
    },
  standardOptions+:
    {
      '#withUnit': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Unit a field should use. The unit you select is applied to all fields except time.\nYou can use the units ID availables in Grafana or a custom unit.\nAvailable units in Grafana: https://github.com/grafana/grafana/blob/main/packages/grafana-data/src/valueFormats/categories.ts\nAs custom unit, you can use the following formats:\n`suffix:<suffix>` for custom unit that should go after value.\n`prefix:<prefix>` for custom unit that should go before value.\n`time:<format>` For custom date time formats type for example `time:YYYY-MM-DD`.\n`si:<base scale><unit characters>` for custom SI units. For example: `si: mF`. This one is a bit more advanced as you can specify both a unit and the source data scale. So if your source data is represented as milli (thousands of) something prefix the unit with that SI scale character.\n`count:<unit>` for a custom count unit.\n`currency:<unit>` for custom a currency unit.' } },
      withUnit(value): {
        fieldConfig+: {
          defaults+: {
            unit: value,
          },
        },
      },
      '#withMin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'The minimum value used in percentage threshold calculations. Leave blank for auto calculation based on all series and fields.' } },
      withMin(value): {
        fieldConfig+: {
          defaults+: {
            min: value,
          },
        },
      },
      '#withMax': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'The maximum value used in percentage threshold calculations. Leave blank for auto calculation based on all series and fields.' } },
      withMax(value): {
        fieldConfig+: {
          defaults+: {
            max: value,
          },
        },
      },
      '#withDecimals': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Specify the number of decimals Grafana includes in the rendered value.\nIf you leave this field blank, Grafana automatically truncates the number of decimals based on the value.\nFor example 1.1234 will display as 1.12 and 100.456 will display as 100.\nTo display all decimals, set the unit to `String`.' } },
      withDecimals(value): {
        fieldConfig+: {
          defaults+: {
            decimals: value,
          },
        },
      },
      '#withDisplayName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The display value for this field.  This supports template variables blank is auto' } },
      withDisplayName(value): {
        fieldConfig+: {
          defaults+: {
            displayName: value,
          },
        },
      },
      color+:
        {
          '#withFixedColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The fixed color value for fixed or shades color modes.' } },
          withFixedColor(value): {
            fieldConfig+: {
              defaults+: {
                color+: {
                  fixedColor: value,
                },
              },
            },
          },
          '#withMode': { 'function': { args: [{ default: null, enums: ['thresholds', 'palette-classic', 'palette-classic-by-name', 'continuous-GrYlRd', 'continuous-RdYlGr', 'continuous-BlYlRd', 'continuous-YlRd', 'continuous-BlPu', 'continuous-YlBl', 'continuous-blues', 'continuous-reds', 'continuous-greens', 'continuous-purples', 'fixed', 'shades'], name: 'value', type: ['string'] }], help: 'Color mode for a field. You can specify a single color, or select a continuous (gradient) color schemes, based on a value.\nContinuous color interpolates a color using the percentage of a value relative to min and max.\nAccepted values are:\n`thresholds`: From thresholds. Informs Grafana to take the color from the matching threshold\n`palette-classic`: Classic palette. Grafana will assign color by looking up a color in a palette by series index. Useful for Graphs and pie charts and other categorical data visualizations\n`palette-classic-by-name`: Classic palette (by name). Grafana will assign color by looking up a color in a palette by series name. Useful for Graphs and pie charts and other categorical data visualizations\n`continuous-GrYlRd`: ontinuous Green-Yellow-Red palette mode\n`continuous-RdYlGr`: Continuous Red-Yellow-Green palette mode\n`continuous-BlYlRd`: Continuous Blue-Yellow-Red palette mode\n`continuous-YlRd`: Continuous Yellow-Red palette mode\n`continuous-BlPu`: Continuous Blue-Purple palette mode\n`continuous-YlBl`: Continuous Yellow-Blue palette mode\n`continuous-blues`: Continuous Blue palette mode\n`continuous-reds`: Continuous Red palette mode\n`continuous-greens`: Continuous Green palette mode\n`continuous-purples`: Continuous Purple palette mode\n`shades`: Shades of a single color. Specify a single color, useful in an override rule.\n`fixed`: Fixed color mode. Specify a single color, useful in an override rule.' } },
          withMode(value): {
            fieldConfig+: {
              defaults+: {
                color+: {
                  mode: value,
                },
              },
            },
          },
          '#withSeriesBy': { 'function': { args: [{ default: null, enums: ['min', 'max', 'last'], name: 'value', type: ['string'] }], help: 'Defines how to assign a series color from "by value" color schemes. For example for an aggregated data points like a timeseries, the color can be assigned by the min, max or last value.' } },
          withSeriesBy(value): {
            fieldConfig+: {
              defaults+: {
                color+: {
                  seriesBy: value,
                },
              },
            },
          },
        },
      '#withNoValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Alternative to empty string' } },
      withNoValue(value): {
        fieldConfig+: {
          defaults+: {
            noValue: value,
          },
        },
      },
      '#withLinks': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'The behavior when clicking on a result' } },
      withLinks(value): {
        fieldConfig+: {
          defaults+: {
            links:
              (if std.isArray(value)
               then value
               else [value]),
          },
        },
      },
      '#withLinksMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'The behavior when clicking on a result' } },
      withLinksMixin(value): {
        fieldConfig+: {
          defaults+: {
            links+:
              (if std.isArray(value)
               then value
               else [value]),
          },
        },
      },
      '#withMappings': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Convert input values into a display string' } },
      withMappings(value): {
        fieldConfig+: {
          defaults+: {
            mappings:
              (if std.isArray(value)
               then value
               else [value]),
          },
        },
      },
      '#withMappingsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Convert input values into a display string' } },
      withMappingsMixin(value): {
        fieldConfig+: {
          defaults+: {
            mappings+:
              (if std.isArray(value)
               then value
               else [value]),
          },
        },
      },
      '#withOverrides': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Overrides are the options applied to specific fields overriding the defaults.' } },
      withOverrides(value): {
        fieldConfig+: {
          overrides:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withOverridesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Overrides are the options applied to specific fields overriding the defaults.' } },
      withOverridesMixin(value): {
        fieldConfig+: {
          overrides+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withFilterable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'True if data source field supports ad-hoc filters' } },
      withFilterable(value=true): {
        fieldConfig+: {
          defaults+: {
            filterable: value,
          },
        },
      },
      '#withPath': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'An explicit path to the field in the datasource.  When the frame meta includes a path,\nThis will default to `${frame.meta.path}/${field.name}\n\nWhen defined, this value can be used as an identifier within the datasource scope, and\nmay be used to update the results' } },
      withPath(value): {
        fieldConfig+: {
          defaults+: {
            path: value,
          },
        },
      },
      thresholds+:
        {
          '#withMode': { 'function': { args: [{ default: null, enums: ['absolute', 'percentage'], name: 'value', type: ['string'] }], help: 'Thresholds can either be `absolute` (specific number) or `percentage` (relative to min or max, it will be values between 0 and 1).' } },
          withMode(value): {
            fieldConfig+: {
              defaults+: {
                thresholds+: {
                  mode: value,
                },
              },
            },
          },
          '#withSteps': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: "Must be sorted by 'value', first value is always -Infinity" } },
          withSteps(value): {
            fieldConfig+: {
              defaults+: {
                thresholds+: {
                  steps:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
            },
          },
          '#withStepsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: "Must be sorted by 'value', first value is always -Infinity" } },
          withStepsMixin(value): {
            fieldConfig+: {
              defaults+: {
                thresholds+: {
                  steps+:
                    (if std.isArray(value)
                     then value
                     else [value]),
                },
              },
            },
          },
        },
      mapping+:
        {
          '#': { help: '', name: 'mapping' },
          ValueMap+:
            {
              '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Map with <value_to_match>: ValueMappingResult. For example: { "10": { text: "Perfection!", color: "green" } }' } },
              withOptions(value): {
                options: value,
              },
              '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Map with <value_to_match>: ValueMappingResult. For example: { "10": { text: "Perfection!", color: "green" } }' } },
              withOptionsMixin(value): {
                options+: value,
              },
              '#withType': { 'function': { args: [{ default: 'value', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withType(value='value'): {
                type: value,
              },
              '#withTypeMixin': { 'function': { args: [{ default: 'value', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withTypeMixin(value): {
                type+: value,
              },
              type+:
                {
                  '#withMappingType': { 'function': { args: [{ default: null, enums: ['value', 'range', 'regex', 'special'], name: 'value', type: ['string'] }], help: 'Supported value mapping types\n`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.\n`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.\n`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.\n`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.' } },
                  withMappingType(value): {
                    type+: {
                      MappingType: value,
                    },
                  },
                },
            },
          RangeMap+:
            {
              '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Range to match against and the result to apply when the value is within the range' } },
              withOptions(value): {
                options: value,
              },
              '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Range to match against and the result to apply when the value is within the range' } },
              withOptionsMixin(value): {
                options+: value,
              },
              options+:
                {
                  '#withFrom': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Min value of the range. It can be null which means -Infinity' } },
                  withFrom(value): {
                    options+: {
                      from: value,
                    },
                  },
                  '#withResult': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Result used as replacement with text and color when the value matches' } },
                  withResult(value): {
                    options+: {
                      result: value,
                    },
                  },
                  '#withResultMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Result used as replacement with text and color when the value matches' } },
                  withResultMixin(value): {
                    options+: {
                      result+: value,
                    },
                  },
                  result+:
                    {
                      '#withColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Text to use when the value matches' } },
                      withColor(value): {
                        options+: {
                          result+: {
                            color: value,
                          },
                        },
                      },
                      '#withIcon': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Icon to display when the value matches. Only specific visualizations.' } },
                      withIcon(value): {
                        options+: {
                          result+: {
                            icon: value,
                          },
                        },
                      },
                      '#withIndex': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Position in the mapping array. Only used internally.' } },
                      withIndex(value): {
                        options+: {
                          result+: {
                            index: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Text to display when the value matches' } },
                      withText(value): {
                        options+: {
                          result+: {
                            text: value,
                          },
                        },
                      },
                    },
                  '#withTo': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Max value of the range. It can be null which means +Infinity' } },
                  withTo(value): {
                    options+: {
                      to: value,
                    },
                  },
                },
              '#withType': { 'function': { args: [{ default: 'range', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withType(value='range'): {
                type: value,
              },
              '#withTypeMixin': { 'function': { args: [{ default: 'range', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withTypeMixin(value): {
                type+: value,
              },
              type+:
                {
                  '#withMappingType': { 'function': { args: [{ default: null, enums: ['value', 'range', 'regex', 'special'], name: 'value', type: ['string'] }], help: 'Supported value mapping types\n`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.\n`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.\n`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.\n`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.' } },
                  withMappingType(value): {
                    type+: {
                      MappingType: value,
                    },
                  },
                },
            },
          RegexMap+:
            {
              '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Regular expression to match against and the result to apply when the value matches the regex' } },
              withOptions(value): {
                options: value,
              },
              '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Regular expression to match against and the result to apply when the value matches the regex' } },
              withOptionsMixin(value): {
                options+: value,
              },
              options+:
                {
                  '#withPattern': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Regular expression to match against' } },
                  withPattern(value): {
                    options+: {
                      pattern: value,
                    },
                  },
                  '#withResult': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Result used as replacement with text and color when the value matches' } },
                  withResult(value): {
                    options+: {
                      result: value,
                    },
                  },
                  '#withResultMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Result used as replacement with text and color when the value matches' } },
                  withResultMixin(value): {
                    options+: {
                      result+: value,
                    },
                  },
                  result+:
                    {
                      '#withColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Text to use when the value matches' } },
                      withColor(value): {
                        options+: {
                          result+: {
                            color: value,
                          },
                        },
                      },
                      '#withIcon': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Icon to display when the value matches. Only specific visualizations.' } },
                      withIcon(value): {
                        options+: {
                          result+: {
                            icon: value,
                          },
                        },
                      },
                      '#withIndex': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Position in the mapping array. Only used internally.' } },
                      withIndex(value): {
                        options+: {
                          result+: {
                            index: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Text to display when the value matches' } },
                      withText(value): {
                        options+: {
                          result+: {
                            text: value,
                          },
                        },
                      },
                    },
                },
              '#withType': { 'function': { args: [{ default: 'regex', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withType(value='regex'): {
                type: value,
              },
              '#withTypeMixin': { 'function': { args: [{ default: 'regex', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withTypeMixin(value): {
                type+: value,
              },
              type+:
                {
                  '#withMappingType': { 'function': { args: [{ default: null, enums: ['value', 'range', 'regex', 'special'], name: 'value', type: ['string'] }], help: 'Supported value mapping types\n`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.\n`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.\n`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.\n`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.' } },
                  withMappingType(value): {
                    type+: {
                      MappingType: value,
                    },
                  },
                },
            },
          SpecialValueMap+:
            {
              '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withOptions(value): {
                options: value,
              },
              '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withOptionsMixin(value): {
                options+: value,
              },
              options+:
                {
                  '#withMatch': { 'function': { args: [{ default: null, enums: ['true', 'false', 'null', 'nan', 'null+nan', 'empty'], name: 'value', type: ['string'] }], help: 'Special value types supported by the `SpecialValueMap`' } },
                  withMatch(value): {
                    options+: {
                      match: value,
                    },
                  },
                  '#withResult': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Result used as replacement with text and color when the value matches' } },
                  withResult(value): {
                    options+: {
                      result: value,
                    },
                  },
                  '#withResultMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Result used as replacement with text and color when the value matches' } },
                  withResultMixin(value): {
                    options+: {
                      result+: value,
                    },
                  },
                  result+:
                    {
                      '#withColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Text to use when the value matches' } },
                      withColor(value): {
                        options+: {
                          result+: {
                            color: value,
                          },
                        },
                      },
                      '#withIcon': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Icon to display when the value matches. Only specific visualizations.' } },
                      withIcon(value): {
                        options+: {
                          result+: {
                            icon: value,
                          },
                        },
                      },
                      '#withIndex': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Position in the mapping array. Only used internally.' } },
                      withIndex(value): {
                        options+: {
                          result+: {
                            index: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Text to display when the value matches' } },
                      withText(value): {
                        options+: {
                          result+: {
                            text: value,
                          },
                        },
                      },
                    },
                },
              '#withType': { 'function': { args: [{ default: 'special', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withType(value='special'): {
                type: value,
              },
              '#withTypeMixin': { 'function': { args: [{ default: 'special', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withTypeMixin(value): {
                type+: value,
              },
              type+:
                {
                  '#withMappingType': { 'function': { args: [{ default: null, enums: ['value', 'range', 'regex', 'special'], name: 'value', type: ['string'] }], help: 'Supported value mapping types\n`value`: Maps text values to a color or different display text and color. For example, you can configure a value mapping so that all instances of the value 10 appear as Perfection! rather than the number.\n`range`: Maps numerical ranges to a display text and color. For example, if a value is within a certain range, you can configure a range value mapping to display Low or High rather than the number.\n`regex`: Maps regular expressions to replacement text and a color. For example, if a value is www.example.com, you can configure a regex value mapping so that Grafana displays www and truncates the domain.\n`special`: Maps special values like Null, NaN (not a number), and boolean values like true and false to a display text and color. See SpecialValueMatch to see the list of special values. For example, you can configure a special value mapping so that null values appear as N/A.' } },
                  withMappingType(value): {
                    type+: {
                      MappingType: value,
                    },
                  },
                },
            },
        },
      threshold+: {
        step+:
          {
            '#': { help: '', name: 'step' },
            '#withColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Color represents the color of the visual change that will occur in the dashboard when the threshold value is met or exceeded.' } },
            withColor(value): {
              color: value,
            },
            '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Value represents a specified metric for the threshold, which triggers a visual change in the dashboard when this value is met or exceeded.\nNulls currently appear here when serializing -Infinity to JSON.' } },
            withValue(value): {
              value: value,
            },
          },
      },
      override+:
        {
          '#': { help: '', name: 'override' },
          '#withSystemRef': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withSystemRef(value): {
            __systemRef: value,
          },
          '#withMatcher': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
          withMatcher(value): {
            matcher: value,
          },
          '#withMatcherMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
          withMatcherMixin(value): {
            matcher+: value,
          },
          matcher+:
            {
              '#withId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: 'The matcher id. This is used to find the matcher implementation from registry.' } },
              withId(value=''): {
                matcher+: {
                  id: value,
                },
              },
              '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
              withOptions(value): {
                matcher+: {
                  options: value,
                },
              },
              '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
              withOptionsMixin(value): {
                matcher+: {
                  options+: value,
                },
              },
            },
          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
          withProperties(value): {
            properties:
              (if std.isArray(value)
               then value
               else [value]),
          },
          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
          withPropertiesMixin(value): {
            properties+:
              (if std.isArray(value)
               then value
               else [value]),
          },
          properties+:
            {
              '#': { help: '', name: 'properties' },
              '#withId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withId(value=''): {
                id: value,
              },
              '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withValue(value): {
                value: value,
              },
              '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withValueMixin(value): {
                value+: value,
              },
            },
        },
    },
  datasource+:
    {
      '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
      withType(value): {
        datasource+: {
          type: value,
        },
      },
      '#withUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specific datasource instance' } },
      withUid(value): {
        datasource+: {
          uid: value,
        },
      },
    },
  libraryPanel+:
    {
      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Library panel name' } },
      withName(value): {
        libraryPanel+: {
          name: value,
        },
      },
      '#withUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Library panel uid' } },
      withUid(value): {
        libraryPanel+: {
          uid: value,
        },
      },
    },
  gridPos+:
    {
      '#withH': { 'function': { args: [{ default: 9, enums: null, name: 'value', type: ['integer'] }], help: 'Panel height. The height is the number of rows from the top edge of the panel.' } },
      withH(value=9): {
        gridPos+: {
          h: value,
        },
      },
      '#withStatic': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: "Whether the panel is fixed within the grid. If true, the panel will not be affected by other panels' interactions" } },
      withStatic(value=true): {
        gridPos+: {
          static: value,
        },
      },
      '#withW': { 'function': { args: [{ default: 12, enums: null, name: 'value', type: ['integer'] }], help: 'Panel width. The width is the number of columns from the left edge of the panel.' } },
      withW(value=12): {
        gridPos+: {
          w: value,
        },
      },
      '#withX': { 'function': { args: [{ default: 0, enums: null, name: 'value', type: ['integer'] }], help: 'Panel x. The x coordinate is the number of columns from the left edge of the grid' } },
      withX(value=0): {
        gridPos+: {
          x: value,
        },
      },
      '#withY': { 'function': { args: [{ default: 0, enums: null, name: 'value', type: ['integer'] }], help: 'Panel y. The y coordinate is the number of rows from the top edge of the grid' } },
      withY(value=0): {
        gridPos+: {
          y: value,
        },
      },
    },
}
+ (import 'custom/panel.libsonnet')
