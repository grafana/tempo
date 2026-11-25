// This file is generated, do not manually edit.
(import '../panel.libsonnet')
+ {
  '#': { help: 'grafonnet.panel.stateTimeline', name: 'stateTimeline' },
  panelOptions+:
    {
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'state-timeline',
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
          '#withFillOpacity': { 'function': { args: [{ default: 70, enums: null, name: 'value', type: ['integer'] }], help: '' } },
          withFillOpacity(value=70): {
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
          '#withInsertNulls': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['boolean', 'integer'] }], help: '' } },
          withInsertNulls(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  insertNulls: value,
                },
              },
            },
          },
          '#withInsertNullsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['boolean', 'integer'] }], help: '' } },
          withInsertNullsMixin(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  insertNulls+: value,
                },
              },
            },
          },
          '#withLineWidth': { 'function': { args: [{ default: 0, enums: null, name: 'value', type: ['integer'] }], help: '' } },
          withLineWidth(value=0): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  lineWidth: value,
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
          '#withSpanNulls': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['boolean', 'integer'] }], help: '' } },
          withSpanNulls(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  spanNulls: value,
                },
              },
            },
          },
          '#withSpanNullsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['boolean', 'integer'] }], help: '' } },
          withSpanNullsMixin(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  spanNulls+: value,
                },
              },
            },
          },
        },
    },
  },
  options+:
    {
      '#withAlignValue': { 'function': { args: [{ default: 'left', enums: ['center', 'left', 'right'], name: 'value', type: ['string'] }], help: 'Controls the value alignment in the TimelineChart component' } },
      withAlignValue(value='left'): {
        options+: {
          alignValue: value,
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
      '#withMergeValues': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Merge equal consecutive values' } },
      withMergeValues(value=true): {
        options+: {
          mergeValues: value,
        },
      },
      '#withPerPage': { 'function': { args: [{ default: 20, enums: null, name: 'value', type: ['number'] }], help: 'Enables pagination when > 0' } },
      withPerPage(value=20): {
        options+: {
          perPage: value,
        },
      },
      '#withRowHeight': { 'function': { args: [{ default: 0.9, enums: null, name: 'value', type: ['number'] }], help: 'Controls the row height' } },
      withRowHeight(value=0.9): {
        options+: {
          rowHeight: value,
        },
      },
      '#withShowValue': { 'function': { args: [{ default: 'auto', enums: ['auto', 'never', 'always'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
      withShowValue(value='auto'): {
        options+: {
          showValue: value,
        },
      },
      '#withTimezone': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withTimezone(value): {
        options+: {
          timezone:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withTimezoneMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withTimezoneMixin(value): {
        options+: {
          timezone+:
            (if std.isArray(value)
             then value
             else [value]),
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
