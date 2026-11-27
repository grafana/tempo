// This file is generated, do not manually edit.
(import '../panel.libsonnet')
+ {
  '#': { help: 'grafonnet.panel.xyChart', name: 'xyChart' },
  panelOptions+:
    {
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'xychart',
      },
    },
  fieldConfig+: {
    defaults+: {
      custom+:
        {
          '#withAxisBorderShow': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withAxisBorderShow(value=true): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  axisBorderShow: value,
                },
              },
            },
          },
          '#withAxisCenteredZero': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withAxisCenteredZero(value=true): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  axisCenteredZero: value,
                },
              },
            },
          },
          '#withAxisColorMode': { 'function': { args: [{ default: null, enums: ['text', 'series'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
          withAxisColorMode(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  axisColorMode: value,
                },
              },
            },
          },
          '#withAxisGridShow': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withAxisGridShow(value=true): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  axisGridShow: value,
                },
              },
            },
          },
          '#withAxisLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withAxisLabel(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  axisLabel: value,
                },
              },
            },
          },
          '#withAxisPlacement': { 'function': { args: [{ default: null, enums: ['auto', 'top', 'right', 'bottom', 'left', 'hidden'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
          withAxisPlacement(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  axisPlacement: value,
                },
              },
            },
          },
          '#withAxisSoftMax': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
          withAxisSoftMax(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  axisSoftMax: value,
                },
              },
            },
          },
          '#withAxisSoftMin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
          withAxisSoftMin(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  axisSoftMin: value,
                },
              },
            },
          },
          '#withAxisWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
          withAxisWidth(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  axisWidth: value,
                },
              },
            },
          },
          '#withFillOpacity': { 'function': { args: [{ default: 50, enums: null, name: 'value', type: ['integer'] }], help: '' } },
          withFillOpacity(value=50): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  fillOpacity: value,
                },
              },
            },
          },
          '#withHideFrom': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
          withHideFrom(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  hideFrom: value,
                },
              },
            },
          },
          '#withHideFromMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
          withHideFromMixin(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  hideFrom+: value,
                },
              },
            },
          },
          hideFrom+:
            {
              '#withLegend': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
              withLegend(value=true): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      hideFrom+: {
                        legend: value,
                      },
                    },
                  },
                },
              },
              '#withTooltip': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
              withTooltip(value=true): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      hideFrom+: {
                        tooltip: value,
                      },
                    },
                  },
                },
              },
              '#withViz': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
              withViz(value=true): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      hideFrom+: {
                        viz: value,
                      },
                    },
                  },
                },
              },
            },
          '#withLineStyle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
          withLineStyle(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  lineStyle: value,
                },
              },
            },
          },
          '#withLineStyleMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
          withLineStyleMixin(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  lineStyle+: value,
                },
              },
            },
          },
          lineStyle+:
            {
              '#withDash': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withDash(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      lineStyle+: {
                        dash:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                    },
                  },
                },
              },
              '#withDashMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withDashMixin(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      lineStyle+: {
                        dash+:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                    },
                  },
                },
              },
              '#withFill': { 'function': { args: [{ default: null, enums: ['solid', 'dash', 'dot', 'square'], name: 'value', type: ['string'] }], help: '' } },
              withFill(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      lineStyle+: {
                        fill: value,
                      },
                    },
                  },
                },
              },
            },
          '#withLineWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
          withLineWidth(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  lineWidth: value,
                },
              },
            },
          },
          '#withPointShape': { 'function': { args: [{ default: null, enums: ['circle', 'square'], name: 'value', type: ['string'] }], help: '' } },
          withPointShape(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  pointShape: value,
                },
              },
            },
          },
          '#withPointSize': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withPointSize(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  pointSize: value,
                },
              },
            },
          },
          '#withPointSizeMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withPointSizeMixin(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  pointSize+: value,
                },
              },
            },
          },
          pointSize+:
            {
              '#withFixed': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
              withFixed(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      pointSize+: {
                        fixed: value,
                      },
                    },
                  },
                },
              },
              '#withMax': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
              withMax(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      pointSize+: {
                        max: value,
                      },
                    },
                  },
                },
              },
              '#withMin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
              withMin(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      pointSize+: {
                        min: value,
                      },
                    },
                  },
                },
              },
            },
          '#withPointStrokeWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
          withPointStrokeWidth(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  pointStrokeWidth: value,
                },
              },
            },
          },
          '#withScaleDistribution': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
          withScaleDistribution(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  scaleDistribution: value,
                },
              },
            },
          },
          '#withScaleDistributionMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
          withScaleDistributionMixin(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  scaleDistribution+: value,
                },
              },
            },
          },
          scaleDistribution+:
            {
              '#withLinearThreshold': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
              withLinearThreshold(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      scaleDistribution+: {
                        linearThreshold: value,
                      },
                    },
                  },
                },
              },
              '#withLog': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
              withLog(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      scaleDistribution+: {
                        log: value,
                      },
                    },
                  },
                },
              },
              '#withType': { 'function': { args: [{ default: null, enums: ['linear', 'log', 'ordinal', 'symlog'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
              withType(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      scaleDistribution+: {
                        type: value,
                      },
                    },
                  },
                },
              },
            },
          '#withShow': { 'function': { args: [{ default: 'points', enums: ['points', 'lines', 'points+lines'], name: 'value', type: ['string'] }], help: '' } },
          withShow(value='points'): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  show: value,
                },
              },
            },
          },
        },
    },
  },
  options+:
    {
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
      '#withMapping': { 'function': { args: [{ default: null, enums: ['auto', 'manual'], name: 'value', type: ['string'] }], help: '' } },
      withMapping(value): {
        options+: {
          mapping: value,
        },
      },
      '#withSeries': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withSeries(value): {
        options+: {
          series:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withSeriesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withSeriesMixin(value): {
        options+: {
          series+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      series+:
        {
          '#': { help: '', name: 'series' },
          '#withColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withColor(value): {
            color: value,
          },
          '#withColorMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withColorMixin(value): {
            color+: value,
          },
          color+:
            {
              '#withMatcher': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcher(value): {
                color+: {
                  matcher: value,
                },
              },
              '#withMatcherMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcherMixin(value): {
                color+: {
                  matcher+: value,
                },
              },
              matcher+:
                {
                  '#withId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: 'The matcher id. This is used to find the matcher implementation from registry.' } },
                  withId(value=''): {
                    color+: {
                      matcher+: {
                        id: value,
                      },
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptions(value): {
                    color+: {
                      matcher+: {
                        options: value,
                      },
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptionsMixin(value): {
                    color+: {
                      matcher+: {
                        options+: value,
                      },
                    },
                  },
                },
            },
          '#withFrame': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withFrame(value): {
            frame: value,
          },
          '#withFrameMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withFrameMixin(value): {
            frame+: value,
          },
          frame+:
            {
              '#withMatcher': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcher(value): {
                frame+: {
                  matcher: value,
                },
              },
              '#withMatcherMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcherMixin(value): {
                frame+: {
                  matcher+: value,
                },
              },
              matcher+:
                {
                  '#withId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: 'The matcher id. This is used to find the matcher implementation from registry.' } },
                  withId(value=''): {
                    frame+: {
                      matcher+: {
                        id: value,
                      },
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptions(value): {
                    frame+: {
                      matcher+: {
                        options: value,
                      },
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptionsMixin(value): {
                    frame+: {
                      matcher+: {
                        options+: value,
                      },
                    },
                  },
                },
            },
          '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withName(value): {
            name: value,
          },
          '#withNameMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withNameMixin(value): {
            name+: value,
          },
          name+:
            {
              '#withFixed': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withFixed(value): {
                name+: {
                  fixed: value,
                },
              },
            },
          '#withSize': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withSize(value): {
            size: value,
          },
          '#withSizeMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withSizeMixin(value): {
            size+: value,
          },
          size+:
            {
              '#withMatcher': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcher(value): {
                size+: {
                  matcher: value,
                },
              },
              '#withMatcherMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcherMixin(value): {
                size+: {
                  matcher+: value,
                },
              },
              matcher+:
                {
                  '#withId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: 'The matcher id. This is used to find the matcher implementation from registry.' } },
                  withId(value=''): {
                    size+: {
                      matcher+: {
                        id: value,
                      },
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptions(value): {
                    size+: {
                      matcher+: {
                        options: value,
                      },
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptionsMixin(value): {
                    size+: {
                      matcher+: {
                        options+: value,
                      },
                    },
                  },
                },
            },
          '#withX': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withX(value): {
            x: value,
          },
          '#withXMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withXMixin(value): {
            x+: value,
          },
          x+:
            {
              '#withMatcher': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcher(value): {
                x+: {
                  matcher: value,
                },
              },
              '#withMatcherMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcherMixin(value): {
                x+: {
                  matcher+: value,
                },
              },
              matcher+:
                {
                  '#withId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: 'The matcher id. This is used to find the matcher implementation from registry.' } },
                  withId(value=''): {
                    x+: {
                      matcher+: {
                        id: value,
                      },
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptions(value): {
                    x+: {
                      matcher+: {
                        options: value,
                      },
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptionsMixin(value): {
                    x+: {
                      matcher+: {
                        options+: value,
                      },
                    },
                  },
                },
            },
          '#withY': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withY(value): {
            y: value,
          },
          '#withYMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withYMixin(value): {
            y+: value,
          },
          y+:
            {
              '#withMatcher': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcher(value): {
                y+: {
                  matcher: value,
                },
              },
              '#withMatcherMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'NOTE: (copied from dashboard_kind.cue, since not exported)\nMatcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.\nIt comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.' } },
              withMatcherMixin(value): {
                y+: {
                  matcher+: value,
                },
              },
              matcher+:
                {
                  '#withId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: 'The matcher id. This is used to find the matcher implementation from registry.' } },
                  withId(value=''): {
                    y+: {
                      matcher+: {
                        id: value,
                      },
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptions(value): {
                    y+: {
                      matcher+: {
                        options: value,
                      },
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The matcher options. This is specific to the matcher implementation.' } },
                  withOptionsMixin(value): {
                    y+: {
                      matcher+: {
                        options+: value,
                      },
                    },
                  },
                },
            },
        },
      '#withTooltip': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
      withTooltip(value): {
        options+: {
          tooltip: value,
        },
      },
      '#withTooltipMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
      withTooltipMixin(value): {
        options+: {
          tooltip+: value,
        },
      },
      tooltip+:
        {
          '#withHideZeros': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withHideZeros(value=true): {
            options+: {
              tooltip+: {
                hideZeros: value,
              },
            },
          },
          '#withMaxHeight': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
          withMaxHeight(value): {
            options+: {
              tooltip+: {
                maxHeight: value,
              },
            },
          },
          '#withMaxWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
          withMaxWidth(value): {
            options+: {
              tooltip+: {
                maxWidth: value,
              },
            },
          },
          '#withMode': { 'function': { args: [{ default: null, enums: ['single', 'multi', 'none'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
          withMode(value): {
            options+: {
              tooltip+: {
                mode: value,
              },
            },
          },
          '#withSort': { 'function': { args: [{ default: null, enums: ['asc', 'desc', 'none'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
          withSort(value): {
            options+: {
              tooltip+: {
                sort: value,
              },
            },
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
