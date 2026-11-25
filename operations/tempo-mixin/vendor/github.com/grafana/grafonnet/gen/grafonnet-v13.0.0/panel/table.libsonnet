// This file is generated, do not manually edit.
(import '../panel.libsonnet')
+ {
  '#': { help: 'grafonnet.panel.table', name: 'table' },
  panelOptions+:
    {
      '#withType': { 'function': { args: [], help: '' } },
      withType(): {
        type: 'table',
      },
    },
  fieldConfig+: {
    defaults+: {
      custom+:
        {
          '#withAlign': { 'function': { args: [{ default: 'auto', enums: ['auto', 'left', 'right', 'center'], name: 'value', type: ['string'] }], help: 'TODO -- should not be table specific!\nTODO docs' } },
          withAlign(value='auto'): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  align: value,
                },
              },
            },
          },
          '#withCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object', 'object', 'object', 'object', 'object', 'object', 'object', 'object'] }], help: '' } },
          withCellOptions(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  cellOptions: value,
                },
              },
            },
          },
          '#withCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object', 'object', 'object', 'object', 'object', 'object', 'object', 'object'] }], help: '' } },
          withCellOptionsMixin(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  cellOptions+: value,
                },
              },
            },
          },
          cellOptions+:
            {
              '#withTableCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object', 'object', 'object', 'object', 'object', 'object', 'object', 'object'] }], help: 'Table cell options. Each cell has a display mode\nand other potential options for that display.' } },
              withTableCellOptions(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      cellOptions+: {
                        TableCellOptions: value,
                      },
                    },
                  },
                },
              },
              '#withTableCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object', 'object', 'object', 'object', 'object', 'object', 'object', 'object'] }], help: 'Table cell options. Each cell has a display mode\nand other potential options for that display.' } },
              withTableCellOptionsMixin(value): {
                fieldConfig+: {
                  defaults+: {
                    custom+: {
                      cellOptions+: {
                        TableCellOptions+: value,
                      },
                    },
                  },
                },
              },
              TableCellOptions+:
                {
                  '#withTableAutoCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Auto mode table cell options' } },
                  withTableAutoCellOptions(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableAutoCellOptions: value,
                          },
                        },
                      },
                    },
                  },
                  '#withTableAutoCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Auto mode table cell options' } },
                  withTableAutoCellOptionsMixin(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableAutoCellOptions+: value,
                          },
                        },
                      },
                    },
                  },
                  TableAutoCellOptions+:
                    {
                      '#withType': { 'function': { args: [{ default: 'auto', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='auto'): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'auto', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type+: value,
                              },
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withTableCellDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
                          withTableCellDisplayMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    type+: {
                                      TableCellDisplayMode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                      '#withWrapText': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withWrapText(value=true): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                wrapText: value,
                              },
                            },
                          },
                        },
                      },
                    },
                  '#withTableSparklineCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Sparkline cell options' } },
                  withTableSparklineCellOptions(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableSparklineCellOptions: value,
                          },
                        },
                      },
                    },
                  },
                  '#withTableSparklineCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Sparkline cell options' } },
                  withTableSparklineCellOptionsMixin(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableSparklineCellOptions+: value,
                          },
                        },
                      },
                    },
                  },
                  TableSparklineCellOptions+:
                    {
                      '#withAxisBorderShow': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withAxisBorderShow(value=true): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                axisBorderShow: value,
                              },
                            },
                          },
                        },
                      },
                      '#withAxisCenteredZero': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withAxisCenteredZero(value=true): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                axisCenteredZero: value,
                              },
                            },
                          },
                        },
                      },
                      '#withAxisColorMode': { 'function': { args: [{ default: null, enums: ['text', 'series'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
                      withAxisColorMode(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                axisColorMode: value,
                              },
                            },
                          },
                        },
                      },
                      '#withAxisGridShow': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withAxisGridShow(value=true): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                axisGridShow: value,
                              },
                            },
                          },
                        },
                      },
                      '#withAxisLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withAxisLabel(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                axisLabel: value,
                              },
                            },
                          },
                        },
                      },
                      '#withAxisPlacement': { 'function': { args: [{ default: null, enums: ['auto', 'top', 'right', 'bottom', 'left', 'hidden'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
                      withAxisPlacement(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                axisPlacement: value,
                              },
                            },
                          },
                        },
                      },
                      '#withAxisSoftMax': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withAxisSoftMax(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                axisSoftMax: value,
                              },
                            },
                          },
                        },
                      },
                      '#withAxisSoftMin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withAxisSoftMin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                axisSoftMin: value,
                              },
                            },
                          },
                        },
                      },
                      '#withAxisWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withAxisWidth(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                axisWidth: value,
                              },
                            },
                          },
                        },
                      },
                      '#withBarAlignment': { 'function': { args: [{ default: null, enums: [-1, 0, 1], name: 'value', type: ['integer'] }], help: 'TODO docs' } },
                      withBarAlignment(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                barAlignment: value,
                              },
                            },
                          },
                        },
                      },
                      '#withBarMaxWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withBarMaxWidth(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                barMaxWidth: value,
                              },
                            },
                          },
                        },
                      },
                      '#withBarWidthFactor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withBarWidthFactor(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                barWidthFactor: value,
                              },
                            },
                          },
                        },
                      },
                      '#withDrawStyle': { 'function': { args: [{ default: null, enums: ['line', 'bars', 'points'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
                      withDrawStyle(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                drawStyle: value,
                              },
                            },
                          },
                        },
                      },
                      '#withFillBelowTo': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withFillBelowTo(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                fillBelowTo: value,
                              },
                            },
                          },
                        },
                      },
                      '#withFillColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withFillColor(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                fillColor: value,
                              },
                            },
                          },
                        },
                      },
                      '#withFillOpacity': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withFillOpacity(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                fillOpacity: value,
                              },
                            },
                          },
                        },
                      },
                      '#withGradientMode': { 'function': { args: [{ default: null, enums: ['none', 'opacity', 'hue', 'scheme'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
                      withGradientMode(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                gradientMode: value,
                              },
                            },
                          },
                        },
                      },
                      '#withHideFrom': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
                      withHideFrom(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                hideFrom: value,
                              },
                            },
                          },
                        },
                      },
                      '#withHideFromMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
                      withHideFromMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                hideFrom+: value,
                              },
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
                                  cellOptions+: {
                                    hideFrom+: {
                                      legend: value,
                                    },
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
                                  cellOptions+: {
                                    hideFrom+: {
                                      tooltip: value,
                                    },
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
                                  cellOptions+: {
                                    hideFrom+: {
                                      viz: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                      '#withHideValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withHideValue(value=true): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                hideValue: value,
                              },
                            },
                          },
                        },
                      },
                      '#withInsertNulls': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['boolean', 'number'] }], help: '' } },
                      withInsertNulls(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                insertNulls: value,
                              },
                            },
                          },
                        },
                      },
                      '#withInsertNullsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['boolean', 'number'] }], help: '' } },
                      withInsertNullsMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                insertNulls+: value,
                              },
                            },
                          },
                        },
                      },
                      '#withLineColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withLineColor(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                lineColor: value,
                              },
                            },
                          },
                        },
                      },
                      '#withLineInterpolation': { 'function': { args: [{ default: null, enums: ['linear', 'smooth', 'stepBefore', 'stepAfter'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
                      withLineInterpolation(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                lineInterpolation: value,
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
                              cellOptions+: {
                                lineStyle: value,
                              },
                            },
                          },
                        },
                      },
                      '#withLineStyleMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
                      withLineStyleMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                lineStyle+: value,
                              },
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
                                  cellOptions+: {
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
                          },
                          '#withDashMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                          withDashMixin(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
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
                          },
                          '#withFill': { 'function': { args: [{ default: null, enums: ['solid', 'dash', 'dot', 'square'], name: 'value', type: ['string'] }], help: '' } },
                          withFill(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    lineStyle+: {
                                      fill: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                      '#withLineWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withLineWidth(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                lineWidth: value,
                              },
                            },
                          },
                        },
                      },
                      '#withPointColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withPointColor(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                pointColor: value,
                              },
                            },
                          },
                        },
                      },
                      '#withPointSize': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withPointSize(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                pointSize: value,
                              },
                            },
                          },
                        },
                      },
                      '#withPointSymbol': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withPointSymbol(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                pointSymbol: value,
                              },
                            },
                          },
                        },
                      },
                      '#withScaleDistribution': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
                      withScaleDistribution(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                scaleDistribution: value,
                              },
                            },
                          },
                        },
                      },
                      '#withScaleDistributionMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
                      withScaleDistributionMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                scaleDistribution+: value,
                              },
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
                                  cellOptions+: {
                                    scaleDistribution+: {
                                      linearThreshold: value,
                                    },
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
                                  cellOptions+: {
                                    scaleDistribution+: {
                                      log: value,
                                    },
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
                                  cellOptions+: {
                                    scaleDistribution+: {
                                      type: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                      '#withShowPoints': { 'function': { args: [{ default: null, enums: ['auto', 'never', 'always'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
                      withShowPoints(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                showPoints: value,
                              },
                            },
                          },
                        },
                      },
                      '#withSpanNulls': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['boolean', 'number'] }], help: 'Indicate if null values should be treated as gaps or connected.\nWhen the value is a number, it represents the maximum delta in the\nX axis that should be considered connected.  For timeseries, this is milliseconds' } },
                      withSpanNulls(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                spanNulls: value,
                              },
                            },
                          },
                        },
                      },
                      '#withSpanNullsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['boolean', 'number'] }], help: 'Indicate if null values should be treated as gaps or connected.\nWhen the value is a number, it represents the maximum delta in the\nX axis that should be considered connected.  For timeseries, this is milliseconds' } },
                      withSpanNullsMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                spanNulls+: value,
                              },
                            },
                          },
                        },
                      },
                      '#withStacking': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
                      withStacking(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                stacking: value,
                              },
                            },
                          },
                        },
                      },
                      '#withStackingMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
                      withStackingMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                stacking+: value,
                              },
                            },
                          },
                        },
                      },
                      stacking+:
                        {
                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                          withGroup(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    stacking+: {
                                      group: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                          '#withMode': { 'function': { args: [{ default: null, enums: ['none', 'normal', 'percent'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
                          withMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    stacking+: {
                                      mode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                      '#withThresholdsStyle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
                      withThresholdsStyle(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                thresholdsStyle: value,
                              },
                            },
                          },
                        },
                      },
                      '#withThresholdsStyleMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'TODO docs' } },
                      withThresholdsStyleMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                thresholdsStyle+: value,
                              },
                            },
                          },
                        },
                      },
                      thresholdsStyle+:
                        {
                          '#withMode': { 'function': { args: [{ default: null, enums: ['off', 'line', 'dashed', 'area', 'line+area', 'dashed+area', 'series'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
                          withMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    thresholdsStyle+: {
                                      mode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                      '#withTransform': { 'function': { args: [{ default: null, enums: ['constant', 'negative-Y'], name: 'value', type: ['string'] }], help: 'TODO docs' } },
                      withTransform(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                transform: value,
                              },
                            },
                          },
                        },
                      },
                      '#withType': { 'function': { args: [{ default: 'sparkline', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='sparkline'): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'sparkline', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type+: value,
                              },
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withTableCellDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
                          withTableCellDisplayMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    type+: {
                                      TableCellDisplayMode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                    },
                  '#withTableBarGaugeCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Gauge cell options' } },
                  withTableBarGaugeCellOptions(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableBarGaugeCellOptions: value,
                          },
                        },
                      },
                    },
                  },
                  '#withTableBarGaugeCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Gauge cell options' } },
                  withTableBarGaugeCellOptionsMixin(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableBarGaugeCellOptions+: value,
                          },
                        },
                      },
                    },
                  },
                  TableBarGaugeCellOptions+:
                    {
                      '#withMode': { 'function': { args: [{ default: null, enums: ['basic', 'lcd', 'gradient'], name: 'value', type: ['string'] }], help: 'Enum expressing the possible display modes\nfor the bar gauge component of Grafana UI' } },
                      withMode(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                mode: value,
                              },
                            },
                          },
                        },
                      },
                      '#withType': { 'function': { args: [{ default: 'gauge', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='gauge'): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'gauge', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type+: value,
                              },
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withTableCellDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
                          withTableCellDisplayMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    type+: {
                                      TableCellDisplayMode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                      '#withValueDisplayMode': { 'function': { args: [{ default: null, enums: ['color', 'text', 'hidden'], name: 'value', type: ['string'] }], help: 'Allows for the table cell gauge display type to set the gauge mode.' } },
                      withValueDisplayMode(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                valueDisplayMode: value,
                              },
                            },
                          },
                        },
                      },
                    },
                  '#withTableColoredBackgroundCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Colored background cell options' } },
                  withTableColoredBackgroundCellOptions(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableColoredBackgroundCellOptions: value,
                          },
                        },
                      },
                    },
                  },
                  '#withTableColoredBackgroundCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Colored background cell options' } },
                  withTableColoredBackgroundCellOptionsMixin(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableColoredBackgroundCellOptions+: value,
                          },
                        },
                      },
                    },
                  },
                  TableColoredBackgroundCellOptions+:
                    {
                      '#withApplyToRow': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withApplyToRow(value=true): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                applyToRow: value,
                              },
                            },
                          },
                        },
                      },
                      '#withMode': { 'function': { args: [{ default: null, enums: ['basic', 'gradient'], name: 'value', type: ['string'] }], help: 'Display mode to the "Colored Background" display\nmode for table cells. Either displays a solid color (basic mode)\nor a gradient.' } },
                      withMode(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                mode: value,
                              },
                            },
                          },
                        },
                      },
                      '#withType': { 'function': { args: [{ default: 'color-background', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='color-background'): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'color-background', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type+: value,
                              },
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withTableCellDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
                          withTableCellDisplayMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    type+: {
                                      TableCellDisplayMode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                      '#withWrapText': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withWrapText(value=true): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                wrapText: value,
                              },
                            },
                          },
                        },
                      },
                    },
                  '#withTableColorTextCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Colored text cell options' } },
                  withTableColorTextCellOptions(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableColorTextCellOptions: value,
                          },
                        },
                      },
                    },
                  },
                  '#withTableColorTextCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Colored text cell options' } },
                  withTableColorTextCellOptionsMixin(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableColorTextCellOptions+: value,
                          },
                        },
                      },
                    },
                  },
                  TableColorTextCellOptions+:
                    {
                      '#withType': { 'function': { args: [{ default: 'color-text', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='color-text'): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'color-text', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type+: value,
                              },
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withTableCellDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
                          withTableCellDisplayMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    type+: {
                                      TableCellDisplayMode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                      '#withWrapText': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withWrapText(value=true): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                wrapText: value,
                              },
                            },
                          },
                        },
                      },
                    },
                  '#withTableImageCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Json view cell options' } },
                  withTableImageCellOptions(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableImageCellOptions: value,
                          },
                        },
                      },
                    },
                  },
                  '#withTableImageCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Json view cell options' } },
                  withTableImageCellOptionsMixin(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableImageCellOptions+: value,
                          },
                        },
                      },
                    },
                  },
                  TableImageCellOptions+:
                    {
                      '#withAlt': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withAlt(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                alt: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTitle(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                title: value,
                              },
                            },
                          },
                        },
                      },
                      '#withType': { 'function': { args: [{ default: 'image', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='image'): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'image', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type+: value,
                              },
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withTableCellDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
                          withTableCellDisplayMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    type+: {
                                      TableCellDisplayMode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                    },
                  '#withTableDataLinksCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Show data links in the cell' } },
                  withTableDataLinksCellOptions(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableDataLinksCellOptions: value,
                          },
                        },
                      },
                    },
                  },
                  '#withTableDataLinksCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Show data links in the cell' } },
                  withTableDataLinksCellOptionsMixin(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableDataLinksCellOptions+: value,
                          },
                        },
                      },
                    },
                  },
                  TableDataLinksCellOptions+:
                    {
                      '#withType': { 'function': { args: [{ default: 'data-links', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='data-links'): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'data-links', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type+: value,
                              },
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withTableCellDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
                          withTableCellDisplayMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    type+: {
                                      TableCellDisplayMode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                    },
                  '#withTableActionsCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Show actions in the cell' } },
                  withTableActionsCellOptions(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableActionsCellOptions: value,
                          },
                        },
                      },
                    },
                  },
                  '#withTableActionsCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Show actions in the cell' } },
                  withTableActionsCellOptionsMixin(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableActionsCellOptions+: value,
                          },
                        },
                      },
                    },
                  },
                  TableActionsCellOptions+:
                    {
                      '#withType': { 'function': { args: [{ default: 'actions', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='actions'): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'actions', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type+: value,
                              },
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withTableCellDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
                          withTableCellDisplayMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    type+: {
                                      TableCellDisplayMode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                    },
                  '#withTableJsonViewCellOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Json view cell options' } },
                  withTableJsonViewCellOptions(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableJsonViewCellOptions: value,
                          },
                        },
                      },
                    },
                  },
                  '#withTableJsonViewCellOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Json view cell options' } },
                  withTableJsonViewCellOptionsMixin(value): {
                    fieldConfig+: {
                      defaults+: {
                        custom+: {
                          cellOptions+: {
                            TableJsonViewCellOptions+: value,
                          },
                        },
                      },
                    },
                  },
                  TableJsonViewCellOptions+:
                    {
                      '#withType': { 'function': { args: [{ default: 'json-view', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='json-view'): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type: value,
                              },
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'json-view', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        fieldConfig+: {
                          defaults+: {
                            custom+: {
                              cellOptions+: {
                                type+: value,
                              },
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withTableCellDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
                          withTableCellDisplayMode(value): {
                            fieldConfig+: {
                              defaults+: {
                                custom+: {
                                  cellOptions+: {
                                    type+: {
                                      TableCellDisplayMode: value,
                                    },
                                  },
                                },
                              },
                            },
                          },
                        },
                    },
                },
            },
          '#withDisplayMode': { 'function': { args: [{ default: null, enums: ['auto', 'color-text', 'color-background', 'color-background-solid', 'gradient-gauge', 'lcd-gauge', 'json-view', 'basic', 'image', 'gauge', 'sparkline', 'data-links', 'custom', 'actions'], name: 'value', type: ['string'] }], help: "Internally, this is the \"type\" of cell that's being displayed\nin the table such as colored text, JSON, gauge, etc.\nThe color-background-solid, gradient-gauge, and lcd-gauge\nmodes are deprecated in favor of new cell subOptions" } },
          withDisplayMode(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  displayMode: value,
                },
              },
            },
          },
          '#withFilterable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withFilterable(value=true): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  filterable: value,
                },
              },
            },
          },
          '#withHidden': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '?? default is missing or false ??' } },
          withHidden(value=true): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  hidden: value,
                },
              },
            },
          },
          '#withHideHeader': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Hides any header for a column, useful for columns that show some static content or buttons.' } },
          withHideHeader(value=true): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  hideHeader: value,
                },
              },
            },
          },
          '#withInspect': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withInspect(value=true): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  inspect: value,
                },
              },
            },
          },
          '#withMinWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
          withMinWidth(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  minWidth: value,
                },
              },
            },
          },
          '#withWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
          withWidth(value): {
            fieldConfig+: {
              defaults+: {
                custom+: {
                  width: value,
                },
              },
            },
          },
        },
    },
  },
  options+:
    {
      '#withCellHeight': { 'function': { args: [{ default: 'sm', enums: ['sm', 'md', 'lg', 'auto'], name: 'value', type: ['string'] }], help: 'Height of a table cell' } },
      withCellHeight(value='sm'): {
        options+: {
          cellHeight: value,
        },
      },
      '#withFooter': { 'function': { args: [{ default: { countRows: false, reducer: null, show: false }, enums: null, name: 'value', type: ['object'] }], help: 'Footer options' } },
      withFooter(value={ countRows: false, reducer: null, show: false }): {
        options+: {
          footer: value,
        },
      },
      '#withFooterMixin': { 'function': { args: [{ default: { countRows: false, reducer: null, show: false }, enums: null, name: 'value', type: ['object'] }], help: 'Footer options' } },
      withFooterMixin(value): {
        options+: {
          footer+: value,
        },
      },
      footer+:
        {
          '#withCountRows': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withCountRows(value=true): {
            options+: {
              footer+: {
                countRows: value,
              },
            },
          },
          '#withEnablePagination': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withEnablePagination(value=true): {
            options+: {
              footer+: {
                enablePagination: value,
              },
            },
          },
          '#withFields': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
          withFields(value): {
            options+: {
              footer+: {
                fields:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withFieldsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
          withFieldsMixin(value): {
            options+: {
              footer+: {
                fields+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withReducer': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'actually 1 value' } },
          withReducer(value): {
            options+: {
              footer+: {
                reducer:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withReducerMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'actually 1 value' } },
          withReducerMixin(value): {
            options+: {
              footer+: {
                reducer+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withShow': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
          withShow(value=true): {
            options+: {
              footer+: {
                show: value,
              },
            },
          },
        },
      '#withFrameIndex': { 'function': { args: [{ default: 0, enums: null, name: 'value', type: ['number'] }], help: 'Represents the index of the selected frame' } },
      withFrameIndex(value=0): {
        options+: {
          frameIndex: value,
        },
      },
      '#withShowHeader': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Controls whether the panel should show the header' } },
      withShowHeader(value=true): {
        options+: {
          showHeader: value,
        },
      },
      '#withShowTypeIcons': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Controls whether the header should show icons for the column types' } },
      withShowTypeIcons(value=true): {
        options+: {
          showTypeIcons: value,
        },
      },
      '#withSortBy': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Used to control row sorting' } },
      withSortBy(value): {
        options+: {
          sortBy:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withSortByMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Used to control row sorting' } },
      withSortByMixin(value): {
        options+: {
          sortBy+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      sortBy+:
        {
          '#': { help: '', name: 'sortBy' },
          '#withDesc': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Flag used to indicate descending sort order' } },
          withDesc(value=true): {
            desc: value,
          },
          '#withDisplayName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Sets the display name of the field to sort by' } },
          withDisplayName(value): {
            displayName: value,
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
