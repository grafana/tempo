// This file is generated, do not manually edit.
(import '../metadata.libsonnet')
+ {
  '#': { help: 'grafonnet.apps.dashboard.v2', name: 'v2' },
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
      '#withAnnotations': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withAnnotations(value): {
        spec+: {
          annotations:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withAnnotationsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withAnnotationsMixin(value): {
        spec+: {
          annotations+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      annotations+:
        {
          '#': { help: '', name: 'annotations' },
          '#withKind': { 'function': { args: [], help: '' } },
          withKind(): {
            kind: 'AnnotationQuery',
          },
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
              '#withBuiltIn': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
              withBuiltIn(value=true): {
                spec+: {
                  builtIn: value,
                },
              },
              '#withEnable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
              withEnable(value=true): {
                spec+: {
                  enable: value,
                },
              },
              '#withFilter': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withFilter(value): {
                spec+: {
                  filter: value,
                },
              },
              '#withFilterMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withFilterMixin(value): {
                spec+: {
                  filter+: value,
                },
              },
              filter+:
                {
                  '#withExclude': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Should the specified panels be included or excluded' } },
                  withExclude(value=true): {
                    spec+: {
                      filter+: {
                        exclude: value,
                      },
                    },
                  },
                  '#withIds': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Panel IDs that should be included or excluded' } },
                  withIds(value): {
                    spec+: {
                      filter+: {
                        ids:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                    },
                  },
                  '#withIdsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Panel IDs that should be included or excluded' } },
                  withIdsMixin(value): {
                    spec+: {
                      filter+: {
                        ids+:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                    },
                  },
                },
              '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
              withHide(value=true): {
                spec+: {
                  hide: value,
                },
              },
              '#withIconColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withIconColor(value): {
                spec+: {
                  iconColor: value,
                },
              },
              '#withLegacyOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Catch-all field for datasource-specific properties. Should not be available in as code tooling.' } },
              withLegacyOptions(value): {
                spec+: {
                  legacyOptions: value,
                },
              },
              '#withLegacyOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Catch-all field for datasource-specific properties. Should not be available in as code tooling.' } },
              withLegacyOptionsMixin(value): {
                spec+: {
                  legacyOptions+: value,
                },
              },
              '#withMappings': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Mappings define how to convert data frame fields to annotation event fields.' } },
              withMappings(value): {
                spec+: {
                  mappings: value,
                },
              },
              '#withMappingsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Mappings define how to convert data frame fields to annotation event fields.' } },
              withMappingsMixin(value): {
                spec+: {
                  mappings+: value,
                },
              },
              '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withName(value): {
                spec+: {
                  name: value,
                },
              },
              '#withPlacement': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Placement can be used to display the annotation query somewhere else on the dashboard other than the default location.' } },
              withPlacement(value): {
                spec+: {
                  placement: value,
                },
              },
              '#withPlacementMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Placement can be used to display the annotation query somewhere else on the dashboard other than the default location.' } },
              withPlacementMixin(value): {
                spec+: {
                  placement+: value,
                },
              },
              placement+:
                {
                  '#withAnnotationQueryPlacement': { 'function': { args: [{ default: null, enums: ['inControlsMenu'], name: 'value', type: ['string'] }], help: 'Annotation Query placement. Defines where the annotation query should be displayed.\n- "inControlsMenu" renders the annotation query in the dashboard controls dropdown menu' } },
                  withAnnotationQueryPlacement(value): {
                    spec+: {
                      placement+: {
                        AnnotationQueryPlacement: value,
                      },
                    },
                  },
                },
              '#withQuery': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withQuery(value): {
                spec+: {
                  query: value,
                },
              },
              '#withQueryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withQueryMixin(value): {
                spec+: {
                  query+: value,
                },
              },
              query+:
                {
                  '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'New type for datasource reference\nNot creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.' } },
                  withDatasource(value): {
                    spec+: {
                      query+: {
                        datasource: value,
                      },
                    },
                  },
                  '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'New type for datasource reference\nNot creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.' } },
                  withDatasourceMixin(value): {
                    spec+: {
                      query+: {
                        datasource+: value,
                      },
                    },
                  },
                  datasource+:
                    {
                      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withName(value): {
                        spec+: {
                          query+: {
                            datasource+: {
                              name: value,
                            },
                          },
                        },
                      },
                    },
                  '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withGroup(value): {
                    spec+: {
                      query+: {
                        group: value,
                      },
                    },
                  },
                  '#withKind': { 'function': { args: [], help: '' } },
                  withKind(): {
                    spec+: {
                      query+: {
                        kind: 'DataQuery',
                      },
                    },
                  },
                  '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withLabels(value): {
                    spec+: {
                      query+: {
                        labels: value,
                      },
                    },
                  },
                  '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withLabelsMixin(value): {
                    spec+: {
                      query+: {
                        labels+: value,
                      },
                    },
                  },
                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSpec(value): {
                    query+: {
                      spec: value,
                    },
                  },
                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSpecMixin(value): {
                    query+: {
                      spec+: value,
                    },
                  },
                  '#withVersion': { 'function': { args: [{ default: 'v0', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withVersion(value='v0'): {
                    spec+: {
                      query+: {
                        version: value,
                      },
                    },
                  },
                },
            },
        },
      '#withCursorSync': { 'function': { args: [{ default: 'Off', enums: ['Crosshair', 'Tooltip', 'Off'], name: 'value', type: ['string'] }], help: '"Off" for no shared crosshair or tooltip (default).\n"Crosshair" for shared crosshair.\n"Tooltip" for shared crosshair AND shared tooltip.' } },
      withCursorSync(value='Off'): {
        spec+: {
          cursorSync: value,
        },
      },
      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Description of dashboard.' } },
      withDescription(value): {
        spec+: {
          description: value,
        },
      },
      '#withEditable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether a dashboard is editable or not.' } },
      withEditable(value=true): {
        spec+: {
          editable: value,
        },
      },
      '#withElements': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withElements(value): {
        spec+: {
          elements: value,
        },
      },
      '#withElementsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withElementsMixin(value): {
        spec+: {
          elements+: value,
        },
      },
      '#withLayout': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object', 'object', 'object'] }], help: '' } },
      withLayout(value): {
        spec+: {
          layout: value,
        },
      },
      '#withLayoutMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object', 'object', 'object'] }], help: '' } },
      withLayoutMixin(value): {
        spec+: {
          layout+: value,
        },
      },
      layout+:
        {
          '#withGridLayoutKind': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withGridLayoutKind(value): {
            spec+: {
              layout+: {
                GridLayoutKind: value,
              },
            },
          },
          '#withGridLayoutKindMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withGridLayoutKindMixin(value): {
            spec+: {
              layout+: {
                GridLayoutKind+: value,
              },
            },
          },
          GridLayoutKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                spec+: {
                  layout+: {
                    kind: 'GridLayout',
                  },
                },
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withSpec(value): {
                layout+: {
                  spec: value,
                },
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withSpecMixin(value): {
                layout+: {
                  spec+: value,
                },
              },
              spec+:
                {
                  '#withItems': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withItems(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          items:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                    },
                  },
                  '#withItemsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withItemsMixin(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          items+:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                    },
                  },
                  items+:
                    {
                      '#': { help: '', name: 'items' },
                      '#withKind': { 'function': { args: [], help: '' } },
                      withKind(): {
                        kind: 'GridLayoutItem',
                      },
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
                          '#withElement': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'reference to a PanelKind from dashboard.spec.elements Expressed as JSON Schema reference' } },
                          withElement(value): {
                            spec+: {
                              element: value,
                            },
                          },
                          '#withElementMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'reference to a PanelKind from dashboard.spec.elements Expressed as JSON Schema reference' } },
                          withElementMixin(value): {
                            spec+: {
                              element+: value,
                            },
                          },
                          element+:
                            {
                              '#withKind': { 'function': { args: [], help: '' } },
                              withKind(): {
                                spec+: {
                                  element+: {
                                    kind: 'ElementReference',
                                  },
                                },
                              },
                              '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withName(value): {
                                spec+: {
                                  element+: {
                                    name: value,
                                  },
                                },
                              },
                            },
                          '#withHeight': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                          withHeight(value): {
                            spec+: {
                              height: value,
                            },
                          },
                          '#withRepeat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withRepeat(value): {
                            spec+: {
                              repeat: value,
                            },
                          },
                          '#withRepeatMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withRepeatMixin(value): {
                            spec+: {
                              repeat+: value,
                            },
                          },
                          repeat+:
                            {
                              '#withDirection': { 'function': { args: [{ default: null, enums: ['h', 'v'], name: 'value', type: ['string'] }], help: '' } },
                              withDirection(value): {
                                spec+: {
                                  repeat+: {
                                    direction: value,
                                  },
                                },
                              },
                              '#withMaxPerRow': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                              withMaxPerRow(value): {
                                spec+: {
                                  repeat+: {
                                    maxPerRow: value,
                                  },
                                },
                              },
                              '#withMode': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withMode(value='variable'): {
                                spec+: {
                                  repeat+: {
                                    mode: value,
                                  },
                                },
                              },
                              '#withModeMixin': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withModeMixin(value): {
                                spec+: {
                                  repeat+: {
                                    mode+: value,
                                  },
                                },
                              },
                              mode+:
                                {
                                  '#withRepeatMode': { 'function': { args: [{ default: null, enums: ['variable'], name: 'value', type: ['string'] }], help: 'other repeat modes will be added in the future: label, frame' } },
                                  withRepeatMode(value): {
                                    spec+: {
                                      repeat+: {
                                        mode+: {
                                          RepeatMode: value,
                                        },
                                      },
                                    },
                                  },
                                },
                              '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withValue(value): {
                                spec+: {
                                  repeat+: {
                                    value: value,
                                  },
                                },
                              },
                            },
                          '#withWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                          withWidth(value): {
                            spec+: {
                              width: value,
                            },
                          },
                          '#withX': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                          withX(value): {
                            spec+: {
                              x: value,
                            },
                          },
                          '#withY': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                          withY(value): {
                            spec+: {
                              y: value,
                            },
                          },
                        },
                    },
                },
            },
          '#withRowsLayoutKind': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withRowsLayoutKind(value): {
            spec+: {
              layout+: {
                RowsLayoutKind: value,
              },
            },
          },
          '#withRowsLayoutKindMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withRowsLayoutKindMixin(value): {
            spec+: {
              layout+: {
                RowsLayoutKind+: value,
              },
            },
          },
          RowsLayoutKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                spec+: {
                  layout+: {
                    kind: 'RowsLayout',
                  },
                },
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withSpec(value): {
                layout+: {
                  spec: value,
                },
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withSpecMixin(value): {
                layout+: {
                  spec+: value,
                },
              },
              spec+:
                {
                  '#withRows': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withRows(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          rows:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                    },
                  },
                  '#withRowsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withRowsMixin(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          rows+:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                    },
                  },
                  rows+:
                    {
                      '#': { help: '', name: 'rows' },
                      '#withKind': { 'function': { args: [], help: '' } },
                      withKind(): {
                        kind: 'RowsLayoutRow',
                      },
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
                          '#withCollapse': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                          withCollapse(value=true): {
                            spec+: {
                              collapse: value,
                            },
                          },
                          '#withConditionalRendering': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withConditionalRendering(value): {
                            spec+: {
                              conditionalRendering: value,
                            },
                          },
                          '#withConditionalRenderingMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withConditionalRenderingMixin(value): {
                            spec+: {
                              conditionalRendering+: value,
                            },
                          },
                          conditionalRendering+:
                            {
                              '#withKind': { 'function': { args: [], help: '' } },
                              withKind(): {
                                spec+: {
                                  conditionalRendering+: {
                                    kind: 'ConditionalRenderingGroup',
                                  },
                                },
                              },
                              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withSpec(value): {
                                conditionalRendering+: {
                                  spec: value,
                                },
                              },
                              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withSpecMixin(value): {
                                conditionalRendering+: {
                                  spec+: value,
                                },
                              },
                              spec+:
                                {
                                  '#withCondition': { 'function': { args: [{ default: null, enums: ['and', 'or'], name: 'value', type: ['string'] }], help: '' } },
                                  withCondition(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          condition: value,
                                        },
                                      },
                                    },
                                  },
                                  '#withItems': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                  withItems(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          items:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                    },
                                  },
                                  '#withItemsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                  withItemsMixin(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          items+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                    },
                                  },
                                  '#withVisibility': { 'function': { args: [{ default: null, enums: ['show', 'hide'], name: 'value', type: ['string'] }], help: '' } },
                                  withVisibility(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          visibility: value,
                                        },
                                      },
                                    },
                                  },
                                },
                            },
                          '#withFillScreen': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                          withFillScreen(value=true): {
                            spec+: {
                              fillScreen: value,
                            },
                          },
                          '#withHideHeader': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                          withHideHeader(value=true): {
                            spec+: {
                              hideHeader: value,
                            },
                          },
                          '#withLayout': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withLayout(value): {
                            spec+: {
                              layout: value,
                            },
                          },
                          '#withLayoutMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withLayoutMixin(value): {
                            spec+: {
                              layout+: value,
                            },
                          },
                          '#withRepeat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withRepeat(value): {
                            spec+: {
                              repeat: value,
                            },
                          },
                          '#withRepeatMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withRepeatMixin(value): {
                            spec+: {
                              repeat+: value,
                            },
                          },
                          repeat+:
                            {
                              '#withMode': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withMode(value='variable'): {
                                spec+: {
                                  repeat+: {
                                    mode: value,
                                  },
                                },
                              },
                              '#withModeMixin': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withModeMixin(value): {
                                spec+: {
                                  repeat+: {
                                    mode+: value,
                                  },
                                },
                              },
                              mode+:
                                {
                                  '#withRepeatMode': { 'function': { args: [{ default: null, enums: ['variable'], name: 'value', type: ['string'] }], help: 'other repeat modes will be added in the future: label, frame' } },
                                  withRepeatMode(value): {
                                    spec+: {
                                      repeat+: {
                                        mode+: {
                                          RepeatMode: value,
                                        },
                                      },
                                    },
                                  },
                                },
                              '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withValue(value): {
                                spec+: {
                                  repeat+: {
                                    value: value,
                                  },
                                },
                              },
                            },
                          '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                          withTitle(value): {
                            spec+: {
                              title: value,
                            },
                          },
                          '#withVariables': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                          withVariables(value): {
                            spec+: {
                              variables:
                                (if std.isArray(value)
                                 then value
                                 else [value]),
                            },
                          },
                          '#withVariablesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                          withVariablesMixin(value): {
                            spec+: {
                              variables+:
                                (if std.isArray(value)
                                 then value
                                 else [value]),
                            },
                          },
                          variables+:
                            {
                              QueryVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'QueryVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Query variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Query variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAllValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withAllValue(value): {
                                        spec+: {
                                          allValue: value,
                                        },
                                      },
                                      '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAllowCustomValue(value=true): {
                                        spec+: {
                                          allowCustomValue: value,
                                        },
                                      },
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDefinition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDefinition(value): {
                                        spec+: {
                                          definition: value,
                                        },
                                      },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withIncludeAll': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withIncludeAll(value=true): {
                                        spec+: {
                                          includeAll: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withMulti(value=true): {
                                        spec+: {
                                          multi: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withPlaceholder': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withPlaceholder(value): {
                                        spec+: {
                                          placeholder: value,
                                        },
                                      },
                                      '#withQuery': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                      withQuery(value): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withQueryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                      withQueryMixin(value): {
                                        spec+: {
                                          query+: value,
                                        },
                                      },
                                      query+:
                                        {
                                          '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'New type for datasource reference\nNot creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.' } },
                                          withDatasource(value): {
                                            spec+: {
                                              query+: {
                                                datasource: value,
                                              },
                                            },
                                          },
                                          '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'New type for datasource reference\nNot creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.' } },
                                          withDatasourceMixin(value): {
                                            spec+: {
                                              query+: {
                                                datasource+: value,
                                              },
                                            },
                                          },
                                          datasource+:
                                            {
                                              '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                              withName(value): {
                                                spec+: {
                                                  query+: {
                                                    datasource+: {
                                                      name: value,
                                                    },
                                                  },
                                                },
                                              },
                                            },
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withGroup(value): {
                                            spec+: {
                                              query+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withKind': { 'function': { args: [], help: '' } },
                                          withKind(): {
                                            spec+: {
                                              query+: {
                                                kind: 'DataQuery',
                                              },
                                            },
                                          },
                                          '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                          withLabels(value): {
                                            spec+: {
                                              query+: {
                                                labels: value,
                                              },
                                            },
                                          },
                                          '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                          withLabelsMixin(value): {
                                            spec+: {
                                              query+: {
                                                labels+: value,
                                              },
                                            },
                                          },
                                          '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                          withSpec(value): {
                                            query+: {
                                              spec: value,
                                            },
                                          },
                                          '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                          withSpecMixin(value): {
                                            query+: {
                                              spec+: value,
                                            },
                                          },
                                          '#withVersion': { 'function': { args: [{ default: 'v0', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withVersion(value='v0'): {
                                            spec+: {
                                              query+: {
                                                version: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withRefresh': { 'function': { args: [{ default: 'never', enums: ['never', 'onDashboardLoad', 'onTimeRangeChanged'], name: 'value', type: ['string'] }], help: 'Options to config when to refresh a variable\n`never`: Never refresh the variable\n`onDashboardLoad`: Queries the data source every time the dashboard loads.\n`onTimeRangeChanged`: Queries the data source when the dashboard time range changes.' } },
                                      withRefresh(value='never'): {
                                        spec+: {
                                          refresh: value,
                                        },
                                      },
                                      '#withRegex': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withRegex(value=''): {
                                        spec+: {
                                          regex: value,
                                        },
                                      },
                                      '#withRegexApplyTo': { 'function': { args: [{ default: 'value', enums: ['value', 'text'], name: 'value', type: ['string'] }], help: 'Determine whether regex applies to variable value or display text\nAccepted values are `value` (apply to value used in queries) or `text` (apply to display text shown to users)' } },
                                      withRegexApplyTo(value='value'): {
                                        spec+: {
                                          regexApplyTo: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                      '#withSort': { 'function': { args: [{ default: null, enums: ['disabled', 'alphabeticalAsc', 'alphabeticalDesc', 'numericalAsc', 'numericalDesc', 'alphabeticalCaseInsensitiveAsc', 'alphabeticalCaseInsensitiveDesc', 'naturalAsc', 'naturalDesc'], name: 'value', type: ['string'] }], help: 'Sort variable options\nAccepted values are:\n`disabled`: No sorting\n`alphabeticalAsc`: Alphabetical ASC\n`alphabeticalDesc`: Alphabetical DESC\n`numericalAsc`: Numerical ASC\n`numericalDesc`: Numerical DESC\n`alphabeticalCaseInsensitiveAsc`: Alphabetical Case Insensitive ASC\n`alphabeticalCaseInsensitiveDesc`: Alphabetical Case Insensitive DESC\n`naturalAsc`: Natural ASC\n`naturalDesc`: Natural DESC\nVariableSort enum with default value' } },
                                      withSort(value): {
                                        spec+: {
                                          sort: value,
                                        },
                                      },
                                      '#withStaticOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withStaticOptions(value): {
                                        spec+: {
                                          staticOptions:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withStaticOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withStaticOptionsMixin(value): {
                                        spec+: {
                                          staticOptions+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      staticOptions+:
                                        {
                                          '#': { help: '', name: 'staticOptions' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withStaticOptionsOrder': { 'function': { args: [{ default: null, enums: ['before', 'after', 'sorted'], name: 'value', type: ['string'] }], help: '' } },
                                      withStaticOptionsOrder(value): {
                                        spec+: {
                                          staticOptionsOrder: value,
                                        },
                                      },
                                    },
                                },
                              TextVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'TextVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Text variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Text variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withQuery(value=''): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              ConstantVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'ConstantVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Constant variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Constant variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withQuery(value=''): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              DatasourceVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'DatasourceVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Datasource variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Datasource variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAllValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withAllValue(value): {
                                        spec+: {
                                          allValue: value,
                                        },
                                      },
                                      '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAllowCustomValue(value=true): {
                                        spec+: {
                                          allowCustomValue: value,
                                        },
                                      },
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withIncludeAll': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withIncludeAll(value=true): {
                                        spec+: {
                                          includeAll: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withMulti(value=true): {
                                        spec+: {
                                          multi: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withPluginId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withPluginId(value=''): {
                                        spec+: {
                                          pluginId: value,
                                        },
                                      },
                                      '#withRefresh': { 'function': { args: [{ default: 'never', enums: ['never', 'onDashboardLoad', 'onTimeRangeChanged'], name: 'value', type: ['string'] }], help: 'Options to config when to refresh a variable\n`never`: Never refresh the variable\n`onDashboardLoad`: Queries the data source every time the dashboard loads.\n`onTimeRangeChanged`: Queries the data source when the dashboard time range changes.' } },
                                      withRefresh(value='never'): {
                                        spec+: {
                                          refresh: value,
                                        },
                                      },
                                      '#withRegex': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withRegex(value=''): {
                                        spec+: {
                                          regex: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              IntervalVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'IntervalVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Interval variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Interval variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAuto': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAuto(value=true): {
                                        spec+: {
                                          auto: value,
                                        },
                                      },
                                      '#withAutoCount': { 'function': { args: [{ default: 0, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                                      withAutoCount(value=0): {
                                        spec+: {
                                          auto_count: value,
                                        },
                                      },
                                      '#withAutoMin': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withAutoMin(value=''): {
                                        spec+: {
                                          auto_min: value,
                                        },
                                      },
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withQuery(value=''): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withRefresh': { 'function': { args: [], help: '' } },
                                      withRefresh(): {
                                        spec+: {
                                          refresh: 'onTimeRangeChanged',
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              CustomVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'CustomVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Custom variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Custom variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAllValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withAllValue(value): {
                                        spec+: {
                                          allValue: value,
                                        },
                                      },
                                      '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAllowCustomValue(value=true): {
                                        spec+: {
                                          allowCustomValue: value,
                                        },
                                      },
                                      '#withCurrent': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withIncludeAll': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withIncludeAll(value=true): {
                                        spec+: {
                                          includeAll: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withMulti(value=true): {
                                        spec+: {
                                          multi: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withQuery(value=''): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                      '#withValuesFormat': { 'function': { args: [{ default: null, enums: ['csv', 'json'], name: 'value', type: ['string'] }], help: '' } },
                                      withValuesFormat(value): {
                                        spec+: {
                                          valuesFormat: value,
                                        },
                                      },
                                    },
                                },
                              GroupByVariableKind+:
                                {
                                  '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withDatasource(value): {
                                    datasource: value,
                                  },
                                  '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withDatasourceMixin(value): {
                                    datasource+: value,
                                  },
                                  datasource+:
                                    {
                                      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value): {
                                        datasource+: {
                                          name: value,
                                        },
                                      },
                                    },
                                  '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withGroup(value): {
                                    group: value,
                                  },
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'GroupByVariable',
                                  },
                                  '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withLabels(value): {
                                    labels: value,
                                  },
                                  '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withLabelsMixin(value): {
                                    labels+: value,
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'GroupBy variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'GroupBy variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDefaultValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withDefaultValue(value): {
                                        spec+: {
                                          defaultValue: value,
                                        },
                                      },
                                      '#withDefaultValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withDefaultValueMixin(value): {
                                        spec+: {
                                          defaultValue+: value,
                                        },
                                      },
                                      defaultValue+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              defaultValue+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              defaultValue+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              defaultValue+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              defaultValue+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              defaultValue+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              defaultValue+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              defaultValue+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withMulti(value=true): {
                                        spec+: {
                                          multi: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              AdhocVariableKind+:
                                {
                                  '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withDatasource(value): {
                                    datasource: value,
                                  },
                                  '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withDatasourceMixin(value): {
                                    datasource+: value,
                                  },
                                  datasource+:
                                    {
                                      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value): {
                                        datasource+: {
                                          name: value,
                                        },
                                      },
                                    },
                                  '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withGroup(value): {
                                    group: value,
                                  },
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'AdhocVariable',
                                  },
                                  '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withLabels(value): {
                                    labels: value,
                                  },
                                  '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withLabelsMixin(value): {
                                    labels+: value,
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Adhoc variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Adhoc variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAllowCustomValue(value=true): {
                                        spec+: {
                                          allowCustomValue: value,
                                        },
                                      },
                                      '#withBaseFilters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withBaseFilters(value): {
                                        spec+: {
                                          baseFilters:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withBaseFiltersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withBaseFiltersMixin(value): {
                                        spec+: {
                                          baseFilters+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      baseFilters+:
                                        {
                                          '#': { help: '', name: 'baseFilters' },
                                          '#withCondition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '@deprecated' } },
                                          withCondition(value): {
                                            condition: value,
                                          },
                                          '#withForceEdit': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                          withForceEdit(value=true): {
                                            forceEdit: value,
                                          },
                                          '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withKey(value): {
                                            key: value,
                                          },
                                          '#withKeyLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withKeyLabel(value): {
                                            keyLabel: value,
                                          },
                                          '#withOperator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withOperator(value): {
                                            operator: value,
                                          },
                                          '#withOrigin': { 'function': { args: [], help: 'Determine the origin of the adhoc variable filter' } },
                                          withOrigin(): {
                                            origin: 'dashboard',
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValueLabels(value): {
                                            valueLabels:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValueLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValueLabelsMixin(value): {
                                            valueLabels+:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValues': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValues(value): {
                                            values:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValuesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValuesMixin(value): {
                                            values+:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                        },
                                      '#withDefaultKeys': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withDefaultKeys(value): {
                                        spec+: {
                                          defaultKeys:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withDefaultKeysMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withDefaultKeysMixin(value): {
                                        spec+: {
                                          defaultKeys+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      defaultKeys+:
                                        {
                                          '#': { help: '', name: 'defaultKeys' },
                                          '#withExpandable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                          withExpandable(value=true): {
                                            expandable: value,
                                          },
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withGroup(value): {
                                            group: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'number'] }], help: '' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'number'] }], help: '' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withEnableGroupBy': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the group-by operator is enabled in the ad hoc filter combobox.' } },
                                      withEnableGroupBy(value=true): {
                                        spec+: {
                                          enableGroupBy: value,
                                        },
                                      },
                                      '#withFilters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withFilters(value): {
                                        spec+: {
                                          filters:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withFiltersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withFiltersMixin(value): {
                                        spec+: {
                                          filters+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      filters+:
                                        {
                                          '#': { help: '', name: 'filters' },
                                          '#withCondition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '@deprecated' } },
                                          withCondition(value): {
                                            condition: value,
                                          },
                                          '#withForceEdit': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                          withForceEdit(value=true): {
                                            forceEdit: value,
                                          },
                                          '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withKey(value): {
                                            key: value,
                                          },
                                          '#withKeyLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withKeyLabel(value): {
                                            keyLabel: value,
                                          },
                                          '#withOperator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withOperator(value): {
                                            operator: value,
                                          },
                                          '#withOrigin': { 'function': { args: [], help: 'Determine the origin of the adhoc variable filter' } },
                                          withOrigin(): {
                                            origin: 'dashboard',
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValueLabels(value): {
                                            valueLabels:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValueLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValueLabelsMixin(value): {
                                            valueLabels+:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValues': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValues(value): {
                                            values:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValuesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValuesMixin(value): {
                                            values+:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                        },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              SwitchVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'SwitchVariable',
                                  },
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
                                      '#withCurrent': { 'function': { args: [{ default: 'false', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withCurrent(value='false'): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withDisabledValue': { 'function': { args: [{ default: 'false', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDisabledValue(value='false'): {
                                        spec+: {
                                          disabledValue: value,
                                        },
                                      },
                                      '#withEnabledValue': { 'function': { args: [{ default: 'true', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withEnabledValue(value='true'): {
                                        spec+: {
                                          enabledValue: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                            },
                        },
                    },
                },
            },
          '#withAutoGridLayoutKind': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withAutoGridLayoutKind(value): {
            spec+: {
              layout+: {
                AutoGridLayoutKind: value,
              },
            },
          },
          '#withAutoGridLayoutKindMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withAutoGridLayoutKindMixin(value): {
            spec+: {
              layout+: {
                AutoGridLayoutKind+: value,
              },
            },
          },
          AutoGridLayoutKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                spec+: {
                  layout+: {
                    kind: 'AutoGridLayout',
                  },
                },
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withSpec(value): {
                layout+: {
                  spec: value,
                },
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withSpecMixin(value): {
                layout+: {
                  spec+: value,
                },
              },
              spec+:
                {
                  '#withColumnWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withColumnWidth(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          columnWidth: value,
                        },
                      },
                    },
                  },
                  '#withColumnWidthMode': { 'function': { args: [{ default: 'standard', enums: ['narrow', 'standard', 'wide', 'custom'], name: 'value', type: ['string'] }], help: '' } },
                  withColumnWidthMode(value='standard'): {
                    spec+: {
                      layout+: {
                        spec+: {
                          columnWidthMode: value,
                        },
                      },
                    },
                  },
                  '#withFillScreen': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withFillScreen(value=true): {
                    spec+: {
                      layout+: {
                        spec+: {
                          fillScreen: value,
                        },
                      },
                    },
                  },
                  '#withItems': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withItems(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          items:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                    },
                  },
                  '#withItemsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withItemsMixin(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          items+:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                    },
                  },
                  items+:
                    {
                      '#': { help: '', name: 'items' },
                      '#withKind': { 'function': { args: [], help: '' } },
                      withKind(): {
                        kind: 'AutoGridLayoutItem',
                      },
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
                          '#withConditionalRendering': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withConditionalRendering(value): {
                            spec+: {
                              conditionalRendering: value,
                            },
                          },
                          '#withConditionalRenderingMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withConditionalRenderingMixin(value): {
                            spec+: {
                              conditionalRendering+: value,
                            },
                          },
                          conditionalRendering+:
                            {
                              '#withKind': { 'function': { args: [], help: '' } },
                              withKind(): {
                                spec+: {
                                  conditionalRendering+: {
                                    kind: 'ConditionalRenderingGroup',
                                  },
                                },
                              },
                              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withSpec(value): {
                                conditionalRendering+: {
                                  spec: value,
                                },
                              },
                              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withSpecMixin(value): {
                                conditionalRendering+: {
                                  spec+: value,
                                },
                              },
                              spec+:
                                {
                                  '#withCondition': { 'function': { args: [{ default: null, enums: ['and', 'or'], name: 'value', type: ['string'] }], help: '' } },
                                  withCondition(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          condition: value,
                                        },
                                      },
                                    },
                                  },
                                  '#withItems': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                  withItems(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          items:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                    },
                                  },
                                  '#withItemsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                  withItemsMixin(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          items+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                    },
                                  },
                                  '#withVisibility': { 'function': { args: [{ default: null, enums: ['show', 'hide'], name: 'value', type: ['string'] }], help: '' } },
                                  withVisibility(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          visibility: value,
                                        },
                                      },
                                    },
                                  },
                                },
                            },
                          '#withElement': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withElement(value): {
                            spec+: {
                              element: value,
                            },
                          },
                          '#withElementMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withElementMixin(value): {
                            spec+: {
                              element+: value,
                            },
                          },
                          element+:
                            {
                              '#withKind': { 'function': { args: [], help: '' } },
                              withKind(): {
                                spec+: {
                                  element+: {
                                    kind: 'ElementReference',
                                  },
                                },
                              },
                              '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withName(value): {
                                spec+: {
                                  element+: {
                                    name: value,
                                  },
                                },
                              },
                            },
                          '#withRepeat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withRepeat(value): {
                            spec+: {
                              repeat: value,
                            },
                          },
                          '#withRepeatMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withRepeatMixin(value): {
                            spec+: {
                              repeat+: value,
                            },
                          },
                          repeat+:
                            {
                              '#withMode': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withMode(value='variable'): {
                                spec+: {
                                  repeat+: {
                                    mode: value,
                                  },
                                },
                              },
                              '#withModeMixin': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withModeMixin(value): {
                                spec+: {
                                  repeat+: {
                                    mode+: value,
                                  },
                                },
                              },
                              mode+:
                                {
                                  '#withRepeatMode': { 'function': { args: [{ default: null, enums: ['variable'], name: 'value', type: ['string'] }], help: 'other repeat modes will be added in the future: label, frame' } },
                                  withRepeatMode(value): {
                                    spec+: {
                                      repeat+: {
                                        mode+: {
                                          RepeatMode: value,
                                        },
                                      },
                                    },
                                  },
                                },
                              '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withValue(value): {
                                spec+: {
                                  repeat+: {
                                    value: value,
                                  },
                                },
                              },
                            },
                        },
                    },
                  '#withMaxColumnCount': { 'function': { args: [{ default: 3, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withMaxColumnCount(value=3): {
                    spec+: {
                      layout+: {
                        spec+: {
                          maxColumnCount: value,
                        },
                      },
                    },
                  },
                  '#withRowHeight': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                  withRowHeight(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          rowHeight: value,
                        },
                      },
                    },
                  },
                  '#withRowHeightMode': { 'function': { args: [{ default: 'standard', enums: ['short', 'standard', 'tall', 'custom'], name: 'value', type: ['string'] }], help: '' } },
                  withRowHeightMode(value='standard'): {
                    spec+: {
                      layout+: {
                        spec+: {
                          rowHeightMode: value,
                        },
                      },
                    },
                  },
                },
            },
          '#withTabsLayoutKind': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withTabsLayoutKind(value): {
            spec+: {
              layout+: {
                TabsLayoutKind: value,
              },
            },
          },
          '#withTabsLayoutKindMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
          withTabsLayoutKindMixin(value): {
            spec+: {
              layout+: {
                TabsLayoutKind+: value,
              },
            },
          },
          TabsLayoutKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                spec+: {
                  layout+: {
                    kind: 'TabsLayout',
                  },
                },
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withSpec(value): {
                layout+: {
                  spec: value,
                },
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withSpecMixin(value): {
                layout+: {
                  spec+: value,
                },
              },
              spec+:
                {
                  '#withTabs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withTabs(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          tabs:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                    },
                  },
                  '#withTabsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withTabsMixin(value): {
                    spec+: {
                      layout+: {
                        spec+: {
                          tabs+:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                    },
                  },
                  tabs+:
                    {
                      '#': { help: '', name: 'tabs' },
                      '#withKind': { 'function': { args: [], help: '' } },
                      withKind(): {
                        kind: 'TabsLayoutTab',
                      },
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
                          '#withConditionalRendering': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withConditionalRendering(value): {
                            spec+: {
                              conditionalRendering: value,
                            },
                          },
                          '#withConditionalRenderingMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withConditionalRenderingMixin(value): {
                            spec+: {
                              conditionalRendering+: value,
                            },
                          },
                          conditionalRendering+:
                            {
                              '#withKind': { 'function': { args: [], help: '' } },
                              withKind(): {
                                spec+: {
                                  conditionalRendering+: {
                                    kind: 'ConditionalRenderingGroup',
                                  },
                                },
                              },
                              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withSpec(value): {
                                conditionalRendering+: {
                                  spec: value,
                                },
                              },
                              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withSpecMixin(value): {
                                conditionalRendering+: {
                                  spec+: value,
                                },
                              },
                              spec+:
                                {
                                  '#withCondition': { 'function': { args: [{ default: null, enums: ['and', 'or'], name: 'value', type: ['string'] }], help: '' } },
                                  withCondition(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          condition: value,
                                        },
                                      },
                                    },
                                  },
                                  '#withItems': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                  withItems(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          items:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                    },
                                  },
                                  '#withItemsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                  withItemsMixin(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          items+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                    },
                                  },
                                  '#withVisibility': { 'function': { args: [{ default: null, enums: ['show', 'hide'], name: 'value', type: ['string'] }], help: '' } },
                                  withVisibility(value): {
                                    spec+: {
                                      conditionalRendering+: {
                                        spec+: {
                                          visibility: value,
                                        },
                                      },
                                    },
                                  },
                                },
                            },
                          '#withLayout': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withLayout(value): {
                            spec+: {
                              layout: value,
                            },
                          },
                          '#withLayoutMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withLayoutMixin(value): {
                            spec+: {
                              layout+: value,
                            },
                          },
                          '#withRepeat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withRepeat(value): {
                            spec+: {
                              repeat: value,
                            },
                          },
                          '#withRepeatMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                          withRepeatMixin(value): {
                            spec+: {
                              repeat+: value,
                            },
                          },
                          repeat+:
                            {
                              '#withMode': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withMode(value='variable'): {
                                spec+: {
                                  repeat+: {
                                    mode: value,
                                  },
                                },
                              },
                              '#withModeMixin': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withModeMixin(value): {
                                spec+: {
                                  repeat+: {
                                    mode+: value,
                                  },
                                },
                              },
                              mode+:
                                {
                                  '#withRepeatMode': { 'function': { args: [{ default: null, enums: ['variable'], name: 'value', type: ['string'] }], help: 'other repeat modes will be added in the future: label, frame' } },
                                  withRepeatMode(value): {
                                    spec+: {
                                      repeat+: {
                                        mode+: {
                                          RepeatMode: value,
                                        },
                                      },
                                    },
                                  },
                                },
                              '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                              withValue(value): {
                                spec+: {
                                  repeat+: {
                                    value: value,
                                  },
                                },
                              },
                            },
                          '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                          withTitle(value): {
                            spec+: {
                              title: value,
                            },
                          },
                          '#withVariables': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                          withVariables(value): {
                            spec+: {
                              variables:
                                (if std.isArray(value)
                                 then value
                                 else [value]),
                            },
                          },
                          '#withVariablesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                          withVariablesMixin(value): {
                            spec+: {
                              variables+:
                                (if std.isArray(value)
                                 then value
                                 else [value]),
                            },
                          },
                          variables+:
                            {
                              QueryVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'QueryVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Query variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Query variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAllValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withAllValue(value): {
                                        spec+: {
                                          allValue: value,
                                        },
                                      },
                                      '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAllowCustomValue(value=true): {
                                        spec+: {
                                          allowCustomValue: value,
                                        },
                                      },
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDefinition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDefinition(value): {
                                        spec+: {
                                          definition: value,
                                        },
                                      },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withIncludeAll': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withIncludeAll(value=true): {
                                        spec+: {
                                          includeAll: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withMulti(value=true): {
                                        spec+: {
                                          multi: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withPlaceholder': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withPlaceholder(value): {
                                        spec+: {
                                          placeholder: value,
                                        },
                                      },
                                      '#withQuery': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                      withQuery(value): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withQueryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                      withQueryMixin(value): {
                                        spec+: {
                                          query+: value,
                                        },
                                      },
                                      query+:
                                        {
                                          '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'New type for datasource reference\nNot creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.' } },
                                          withDatasource(value): {
                                            spec+: {
                                              query+: {
                                                datasource: value,
                                              },
                                            },
                                          },
                                          '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'New type for datasource reference\nNot creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.' } },
                                          withDatasourceMixin(value): {
                                            spec+: {
                                              query+: {
                                                datasource+: value,
                                              },
                                            },
                                          },
                                          datasource+:
                                            {
                                              '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                              withName(value): {
                                                spec+: {
                                                  query+: {
                                                    datasource+: {
                                                      name: value,
                                                    },
                                                  },
                                                },
                                              },
                                            },
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withGroup(value): {
                                            spec+: {
                                              query+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withKind': { 'function': { args: [], help: '' } },
                                          withKind(): {
                                            spec+: {
                                              query+: {
                                                kind: 'DataQuery',
                                              },
                                            },
                                          },
                                          '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                          withLabels(value): {
                                            spec+: {
                                              query+: {
                                                labels: value,
                                              },
                                            },
                                          },
                                          '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                          withLabelsMixin(value): {
                                            spec+: {
                                              query+: {
                                                labels+: value,
                                              },
                                            },
                                          },
                                          '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                          withSpec(value): {
                                            query+: {
                                              spec: value,
                                            },
                                          },
                                          '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                          withSpecMixin(value): {
                                            query+: {
                                              spec+: value,
                                            },
                                          },
                                          '#withVersion': { 'function': { args: [{ default: 'v0', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withVersion(value='v0'): {
                                            spec+: {
                                              query+: {
                                                version: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withRefresh': { 'function': { args: [{ default: 'never', enums: ['never', 'onDashboardLoad', 'onTimeRangeChanged'], name: 'value', type: ['string'] }], help: 'Options to config when to refresh a variable\n`never`: Never refresh the variable\n`onDashboardLoad`: Queries the data source every time the dashboard loads.\n`onTimeRangeChanged`: Queries the data source when the dashboard time range changes.' } },
                                      withRefresh(value='never'): {
                                        spec+: {
                                          refresh: value,
                                        },
                                      },
                                      '#withRegex': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withRegex(value=''): {
                                        spec+: {
                                          regex: value,
                                        },
                                      },
                                      '#withRegexApplyTo': { 'function': { args: [{ default: 'value', enums: ['value', 'text'], name: 'value', type: ['string'] }], help: 'Determine whether regex applies to variable value or display text\nAccepted values are `value` (apply to value used in queries) or `text` (apply to display text shown to users)' } },
                                      withRegexApplyTo(value='value'): {
                                        spec+: {
                                          regexApplyTo: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                      '#withSort': { 'function': { args: [{ default: null, enums: ['disabled', 'alphabeticalAsc', 'alphabeticalDesc', 'numericalAsc', 'numericalDesc', 'alphabeticalCaseInsensitiveAsc', 'alphabeticalCaseInsensitiveDesc', 'naturalAsc', 'naturalDesc'], name: 'value', type: ['string'] }], help: 'Sort variable options\nAccepted values are:\n`disabled`: No sorting\n`alphabeticalAsc`: Alphabetical ASC\n`alphabeticalDesc`: Alphabetical DESC\n`numericalAsc`: Numerical ASC\n`numericalDesc`: Numerical DESC\n`alphabeticalCaseInsensitiveAsc`: Alphabetical Case Insensitive ASC\n`alphabeticalCaseInsensitiveDesc`: Alphabetical Case Insensitive DESC\n`naturalAsc`: Natural ASC\n`naturalDesc`: Natural DESC\nVariableSort enum with default value' } },
                                      withSort(value): {
                                        spec+: {
                                          sort: value,
                                        },
                                      },
                                      '#withStaticOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withStaticOptions(value): {
                                        spec+: {
                                          staticOptions:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withStaticOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withStaticOptionsMixin(value): {
                                        spec+: {
                                          staticOptions+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      staticOptions+:
                                        {
                                          '#': { help: '', name: 'staticOptions' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withStaticOptionsOrder': { 'function': { args: [{ default: null, enums: ['before', 'after', 'sorted'], name: 'value', type: ['string'] }], help: '' } },
                                      withStaticOptionsOrder(value): {
                                        spec+: {
                                          staticOptionsOrder: value,
                                        },
                                      },
                                    },
                                },
                              TextVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'TextVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Text variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Text variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withQuery(value=''): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              ConstantVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'ConstantVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Constant variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Constant variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withQuery(value=''): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              DatasourceVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'DatasourceVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Datasource variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Datasource variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAllValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withAllValue(value): {
                                        spec+: {
                                          allValue: value,
                                        },
                                      },
                                      '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAllowCustomValue(value=true): {
                                        spec+: {
                                          allowCustomValue: value,
                                        },
                                      },
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withIncludeAll': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withIncludeAll(value=true): {
                                        spec+: {
                                          includeAll: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withMulti(value=true): {
                                        spec+: {
                                          multi: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withPluginId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withPluginId(value=''): {
                                        spec+: {
                                          pluginId: value,
                                        },
                                      },
                                      '#withRefresh': { 'function': { args: [{ default: 'never', enums: ['never', 'onDashboardLoad', 'onTimeRangeChanged'], name: 'value', type: ['string'] }], help: 'Options to config when to refresh a variable\n`never`: Never refresh the variable\n`onDashboardLoad`: Queries the data source every time the dashboard loads.\n`onTimeRangeChanged`: Queries the data source when the dashboard time range changes.' } },
                                      withRefresh(value='never'): {
                                        spec+: {
                                          refresh: value,
                                        },
                                      },
                                      '#withRegex': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withRegex(value=''): {
                                        spec+: {
                                          regex: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              IntervalVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'IntervalVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Interval variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Interval variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAuto': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAuto(value=true): {
                                        spec+: {
                                          auto: value,
                                        },
                                      },
                                      '#withAutoCount': { 'function': { args: [{ default: 0, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                                      withAutoCount(value=0): {
                                        spec+: {
                                          auto_count: value,
                                        },
                                      },
                                      '#withAutoMin': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withAutoMin(value=''): {
                                        spec+: {
                                          auto_min: value,
                                        },
                                      },
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withQuery(value=''): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withRefresh': { 'function': { args: [], help: '' } },
                                      withRefresh(): {
                                        spec+: {
                                          refresh: 'onTimeRangeChanged',
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              CustomVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'CustomVariable',
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Custom variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Custom variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAllValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withAllValue(value): {
                                        spec+: {
                                          allValue: value,
                                        },
                                      },
                                      '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAllowCustomValue(value=true): {
                                        spec+: {
                                          allowCustomValue: value,
                                        },
                                      },
                                      '#withCurrent': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withIncludeAll': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withIncludeAll(value=true): {
                                        spec+: {
                                          includeAll: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withMulti(value=true): {
                                        spec+: {
                                          multi: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withQuery(value=''): {
                                        spec+: {
                                          query: value,
                                        },
                                      },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                      '#withValuesFormat': { 'function': { args: [{ default: null, enums: ['csv', 'json'], name: 'value', type: ['string'] }], help: '' } },
                                      withValuesFormat(value): {
                                        spec+: {
                                          valuesFormat: value,
                                        },
                                      },
                                    },
                                },
                              GroupByVariableKind+:
                                {
                                  '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withDatasource(value): {
                                    datasource: value,
                                  },
                                  '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withDatasourceMixin(value): {
                                    datasource+: value,
                                  },
                                  datasource+:
                                    {
                                      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value): {
                                        datasource+: {
                                          name: value,
                                        },
                                      },
                                    },
                                  '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withGroup(value): {
                                    group: value,
                                  },
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'GroupByVariable',
                                  },
                                  '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withLabels(value): {
                                    labels: value,
                                  },
                                  '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withLabelsMixin(value): {
                                    labels+: value,
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'GroupBy variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'GroupBy variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrent(value={ text: '', value: '' }): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withCurrentMixin(value): {
                                        spec+: {
                                          current+: value,
                                        },
                                      },
                                      current+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              current+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              current+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              current+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              current+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              current+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              current+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              current+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDefaultValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withDefaultValue(value): {
                                        spec+: {
                                          defaultValue: value,
                                        },
                                      },
                                      '#withDefaultValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                                      withDefaultValueMixin(value): {
                                        spec+: {
                                          defaultValue+: value,
                                        },
                                      },
                                      defaultValue+:
                                        {
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            spec+: {
                                              defaultValue+: {
                                                properties: value,
                                              },
                                            },
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            spec+: {
                                              defaultValue+: {
                                                properties+: value,
                                              },
                                            },
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            spec+: {
                                              defaultValue+: {
                                                selected: value,
                                              },
                                            },
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            spec+: {
                                              defaultValue+: {
                                                text: value,
                                              },
                                            },
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            spec+: {
                                              defaultValue+: {
                                                text+: value,
                                              },
                                            },
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            spec+: {
                                              defaultValue+: {
                                                value: value,
                                              },
                                            },
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            spec+: {
                                              defaultValue+: {
                                                value+: value,
                                              },
                                            },
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withMulti(value=true): {
                                        spec+: {
                                          multi: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptions(value): {
                                        spec+: {
                                          options:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withOptionsMixin(value): {
                                        spec+: {
                                          options+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      options+:
                                        {
                                          '#': { help: '', name: 'options' },
                                          '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withProperties(value): {
                                            properties: value,
                                          },
                                          '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                                          withPropertiesMixin(value): {
                                            properties+: value,
                                          },
                                          '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                                          withSelected(value=true): {
                                            selected: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                                          withTextMixin(value): {
                                            text+: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              AdhocVariableKind+:
                                {
                                  '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withDatasource(value): {
                                    datasource: value,
                                  },
                                  '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withDatasourceMixin(value): {
                                    datasource+: value,
                                  },
                                  datasource+:
                                    {
                                      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value): {
                                        datasource+: {
                                          name: value,
                                        },
                                      },
                                    },
                                  '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withGroup(value): {
                                    group: value,
                                  },
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'AdhocVariable',
                                  },
                                  '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withLabels(value): {
                                    labels: value,
                                  },
                                  '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withLabelsMixin(value): {
                                    labels+: value,
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Adhoc variable specification' } },
                                  withSpec(value): {
                                    spec: value,
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Adhoc variable specification' } },
                                  withSpecMixin(value): {
                                    spec+: value,
                                  },
                                  spec+:
                                    {
                                      '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withAllowCustomValue(value=true): {
                                        spec+: {
                                          allowCustomValue: value,
                                        },
                                      },
                                      '#withBaseFilters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withBaseFilters(value): {
                                        spec+: {
                                          baseFilters:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withBaseFiltersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withBaseFiltersMixin(value): {
                                        spec+: {
                                          baseFilters+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      baseFilters+:
                                        {
                                          '#': { help: '', name: 'baseFilters' },
                                          '#withCondition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '@deprecated' } },
                                          withCondition(value): {
                                            condition: value,
                                          },
                                          '#withForceEdit': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                          withForceEdit(value=true): {
                                            forceEdit: value,
                                          },
                                          '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withKey(value): {
                                            key: value,
                                          },
                                          '#withKeyLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withKeyLabel(value): {
                                            keyLabel: value,
                                          },
                                          '#withOperator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withOperator(value): {
                                            operator: value,
                                          },
                                          '#withOrigin': { 'function': { args: [], help: 'Determine the origin of the adhoc variable filter' } },
                                          withOrigin(): {
                                            origin: 'dashboard',
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValueLabels(value): {
                                            valueLabels:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValueLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValueLabelsMixin(value): {
                                            valueLabels+:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValues': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValues(value): {
                                            values:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValuesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValuesMixin(value): {
                                            values+:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                        },
                                      '#withDefaultKeys': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withDefaultKeys(value): {
                                        spec+: {
                                          defaultKeys:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withDefaultKeysMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withDefaultKeysMixin(value): {
                                        spec+: {
                                          defaultKeys+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      defaultKeys+:
                                        {
                                          '#': { help: '', name: 'defaultKeys' },
                                          '#withExpandable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                          withExpandable(value=true): {
                                            expandable: value,
                                          },
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withGroup(value): {
                                            group: value,
                                          },
                                          '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withText(value): {
                                            text: value,
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'number'] }], help: '' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'number'] }], help: '' } },
                                          withValueMixin(value): {
                                            value+: value,
                                          },
                                        },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withEnableGroupBy': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the group-by operator is enabled in the ad hoc filter combobox.' } },
                                      withEnableGroupBy(value=true): {
                                        spec+: {
                                          enableGroupBy: value,
                                        },
                                      },
                                      '#withFilters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withFilters(value): {
                                        spec+: {
                                          filters:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      '#withFiltersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withFiltersMixin(value): {
                                        spec+: {
                                          filters+:
                                            (if std.isArray(value)
                                             then value
                                             else [value]),
                                        },
                                      },
                                      filters+:
                                        {
                                          '#': { help: '', name: 'filters' },
                                          '#withCondition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '@deprecated' } },
                                          withCondition(value): {
                                            condition: value,
                                          },
                                          '#withForceEdit': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                          withForceEdit(value=true): {
                                            forceEdit: value,
                                          },
                                          '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withKey(value): {
                                            key: value,
                                          },
                                          '#withKeyLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withKeyLabel(value): {
                                            keyLabel: value,
                                          },
                                          '#withOperator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withOperator(value): {
                                            operator: value,
                                          },
                                          '#withOrigin': { 'function': { args: [], help: 'Determine the origin of the adhoc variable filter' } },
                                          withOrigin(): {
                                            origin: 'dashboard',
                                          },
                                          '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                          withValue(value): {
                                            value: value,
                                          },
                                          '#withValueLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValueLabels(value): {
                                            valueLabels:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValueLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValueLabelsMixin(value): {
                                            valueLabels+:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValues': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValues(value): {
                                            values:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                          '#withValuesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                          withValuesMixin(value): {
                                            values+:
                                              (if std.isArray(value)
                                               then value
                                               else [value]),
                                          },
                                        },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
                                        },
                                      },
                                    },
                                },
                              SwitchVariableKind+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    kind: 'SwitchVariable',
                                  },
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
                                      '#withCurrent': { 'function': { args: [{ default: 'false', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withCurrent(value='false'): {
                                        spec+: {
                                          current: value,
                                        },
                                      },
                                      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDescription(value): {
                                        spec+: {
                                          description: value,
                                        },
                                      },
                                      '#withDisabledValue': { 'function': { args: [{ default: 'false', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withDisabledValue(value='false'): {
                                        spec+: {
                                          disabledValue: value,
                                        },
                                      },
                                      '#withEnabledValue': { 'function': { args: [{ default: 'true', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withEnabledValue(value='true'): {
                                        spec+: {
                                          enabledValue: value,
                                        },
                                      },
                                      '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                                      withHide(value='dontHide'): {
                                        spec+: {
                                          hide: value,
                                        },
                                      },
                                      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withLabel(value): {
                                        spec+: {
                                          label: value,
                                        },
                                      },
                                      '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                      withName(value=''): {
                                        spec+: {
                                          name: value,
                                        },
                                      },
                                      '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOrigin(value): {
                                        spec+: {
                                          origin: value,
                                        },
                                      },
                                      '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                                      withOriginMixin(value): {
                                        spec+: {
                                          origin+: value,
                                        },
                                      },
                                      origin+:
                                        {
                                          '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                                          withGroup(value): {
                                            spec+: {
                                              origin+: {
                                                group: value,
                                              },
                                            },
                                          },
                                          '#withType': { 'function': { args: [], help: '' } },
                                          withType(): {
                                            spec+: {
                                              origin+: {
                                                type: 'datasource',
                                              },
                                            },
                                          },
                                        },
                                      '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                                      withSkipUrlSync(value=true): {
                                        spec+: {
                                          skipUrlSync: value,
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
      '#withLinks': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Links with references to other dashboards or external websites.' } },
      withLinks(value): {
        spec+: {
          links:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withLinksMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Links with references to other dashboards or external websites.' } },
      withLinksMixin(value): {
        spec+: {
          links+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      links+:
        {
          '#': { help: '', name: 'links' },
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
          '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
          withOrigin(value): {
            origin: value,
          },
          '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
          withOriginMixin(value): {
            origin+: value,
          },
          origin+:
            {
              '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
              withGroup(value): {
                origin+: {
                  group: value,
                },
              },
              '#withType': { 'function': { args: [], help: '' } },
              withType(): {
                origin+: {
                  type: 'datasource',
                },
              },
            },
          '#withPlacement': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Placement can be used to display the link somewhere else on the dashboard other than above the visualisations.' } },
          withPlacement(value): {
            placement: value,
          },
          '#withPlacementMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Placement can be used to display the link somewhere else on the dashboard other than above the visualisations.' } },
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
      '#withLiveNow': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'When set to true, the dashboard will redraw panels at an interval matching the pixel width.\nThis will keep data "moving left" regardless of the query refresh rate. This setting helps\navoid dashboards presenting stale live data.' } },
      withLiveNow(value=true): {
        spec+: {
          liveNow: value,
        },
      },
      '#withPreferences': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Dashboard specific preferences (applied per dashboard = all users using the dashboard)' } },
      withPreferences(value): {
        spec+: {
          preferences: value,
        },
      },
      '#withPreferencesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Dashboard specific preferences (applied per dashboard = all users using the dashboard)' } },
      withPreferencesMixin(value): {
        spec+: {
          preferences+: value,
        },
      },
      preferences+:
        {
          '#withLayout': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object'] }], help: 'default layout template to be used when new containers are created' } },
          withLayout(value): {
            spec+: {
              preferences+: {
                layout: value,
              },
            },
          },
          '#withLayoutMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object'] }], help: 'default layout template to be used when new containers are created' } },
          withLayoutMixin(value): {
            spec+: {
              preferences+: {
                layout+: value,
              },
            },
          },
          layout+:
            {
              '#withAutoGridLayoutKind': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withAutoGridLayoutKind(value): {
                spec+: {
                  preferences+: {
                    layout+: {
                      AutoGridLayoutKind: value,
                    },
                  },
                },
              },
              '#withAutoGridLayoutKindMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withAutoGridLayoutKindMixin(value): {
                spec+: {
                  preferences+: {
                    layout+: {
                      AutoGridLayoutKind+: value,
                    },
                  },
                },
              },
              AutoGridLayoutKind+:
                {
                  '#withKind': { 'function': { args: [], help: '' } },
                  withKind(): {
                    spec+: {
                      preferences+: {
                        layout+: {
                          kind: 'AutoGridLayout',
                        },
                      },
                    },
                  },
                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSpec(value): {
                    preferences+: {
                      layout+: {
                        spec: value,
                      },
                    },
                  },
                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSpecMixin(value): {
                    preferences+: {
                      layout+: {
                        spec+: value,
                      },
                    },
                  },
                  spec+:
                    {
                      '#withColumnWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withColumnWidth(value): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                columnWidth: value,
                              },
                            },
                          },
                        },
                      },
                      '#withColumnWidthMode': { 'function': { args: [{ default: 'standard', enums: ['narrow', 'standard', 'wide', 'custom'], name: 'value', type: ['string'] }], help: '' } },
                      withColumnWidthMode(value='standard'): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                columnWidthMode: value,
                              },
                            },
                          },
                        },
                      },
                      '#withFillScreen': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withFillScreen(value=true): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                fillScreen: value,
                              },
                            },
                          },
                        },
                      },
                      '#withItems': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withItems(value): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                items:
                                  (if std.isArray(value)
                                   then value
                                   else [value]),
                              },
                            },
                          },
                        },
                      },
                      '#withItemsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withItemsMixin(value): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                items+:
                                  (if std.isArray(value)
                                   then value
                                   else [value]),
                              },
                            },
                          },
                        },
                      },
                      items+:
                        {
                          '#': { help: '', name: 'items' },
                          '#withKind': { 'function': { args: [], help: '' } },
                          withKind(): {
                            kind: 'AutoGridLayoutItem',
                          },
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
                              '#withConditionalRendering': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withConditionalRendering(value): {
                                spec+: {
                                  conditionalRendering: value,
                                },
                              },
                              '#withConditionalRenderingMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withConditionalRenderingMixin(value): {
                                spec+: {
                                  conditionalRendering+: value,
                                },
                              },
                              conditionalRendering+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    spec+: {
                                      conditionalRendering+: {
                                        kind: 'ConditionalRenderingGroup',
                                      },
                                    },
                                  },
                                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withSpec(value): {
                                    conditionalRendering+: {
                                      spec: value,
                                    },
                                  },
                                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                                  withSpecMixin(value): {
                                    conditionalRendering+: {
                                      spec+: value,
                                    },
                                  },
                                  spec+:
                                    {
                                      '#withCondition': { 'function': { args: [{ default: null, enums: ['and', 'or'], name: 'value', type: ['string'] }], help: '' } },
                                      withCondition(value): {
                                        spec+: {
                                          conditionalRendering+: {
                                            spec+: {
                                              condition: value,
                                            },
                                          },
                                        },
                                      },
                                      '#withItems': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withItems(value): {
                                        spec+: {
                                          conditionalRendering+: {
                                            spec+: {
                                              items:
                                                (if std.isArray(value)
                                                 then value
                                                 else [value]),
                                            },
                                          },
                                        },
                                      },
                                      '#withItemsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                                      withItemsMixin(value): {
                                        spec+: {
                                          conditionalRendering+: {
                                            spec+: {
                                              items+:
                                                (if std.isArray(value)
                                                 then value
                                                 else [value]),
                                            },
                                          },
                                        },
                                      },
                                      '#withVisibility': { 'function': { args: [{ default: null, enums: ['show', 'hide'], name: 'value', type: ['string'] }], help: '' } },
                                      withVisibility(value): {
                                        spec+: {
                                          conditionalRendering+: {
                                            spec+: {
                                              visibility: value,
                                            },
                                          },
                                        },
                                      },
                                    },
                                },
                              '#withElement': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withElement(value): {
                                spec+: {
                                  element: value,
                                },
                              },
                              '#withElementMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withElementMixin(value): {
                                spec+: {
                                  element+: value,
                                },
                              },
                              element+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    spec+: {
                                      element+: {
                                        kind: 'ElementReference',
                                      },
                                    },
                                  },
                                  '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withName(value): {
                                    spec+: {
                                      element+: {
                                        name: value,
                                      },
                                    },
                                  },
                                },
                              '#withRepeat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withRepeat(value): {
                                spec+: {
                                  repeat: value,
                                },
                              },
                              '#withRepeatMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withRepeatMixin(value): {
                                spec+: {
                                  repeat+: value,
                                },
                              },
                              repeat+:
                                {
                                  '#withMode': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withMode(value='variable'): {
                                    spec+: {
                                      repeat+: {
                                        mode: value,
                                      },
                                    },
                                  },
                                  '#withModeMixin': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withModeMixin(value): {
                                    spec+: {
                                      repeat+: {
                                        mode+: value,
                                      },
                                    },
                                  },
                                  mode+:
                                    {
                                      '#withRepeatMode': { 'function': { args: [{ default: null, enums: ['variable'], name: 'value', type: ['string'] }], help: 'other repeat modes will be added in the future: label, frame' } },
                                      withRepeatMode(value): {
                                        spec+: {
                                          repeat+: {
                                            mode+: {
                                              RepeatMode: value,
                                            },
                                          },
                                        },
                                      },
                                    },
                                  '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withValue(value): {
                                    spec+: {
                                      repeat+: {
                                        value: value,
                                      },
                                    },
                                  },
                                },
                            },
                        },
                      '#withMaxColumnCount': { 'function': { args: [{ default: 3, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withMaxColumnCount(value=3): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                maxColumnCount: value,
                              },
                            },
                          },
                        },
                      },
                      '#withRowHeight': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: '' } },
                      withRowHeight(value): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                rowHeight: value,
                              },
                            },
                          },
                        },
                      },
                      '#withRowHeightMode': { 'function': { args: [{ default: 'standard', enums: ['short', 'standard', 'tall', 'custom'], name: 'value', type: ['string'] }], help: '' } },
                      withRowHeightMode(value='standard'): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                rowHeightMode: value,
                              },
                            },
                          },
                        },
                      },
                    },
                },
              '#withGridLayoutKind': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withGridLayoutKind(value): {
                spec+: {
                  preferences+: {
                    layout+: {
                      GridLayoutKind: value,
                    },
                  },
                },
              },
              '#withGridLayoutKindMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withGridLayoutKindMixin(value): {
                spec+: {
                  preferences+: {
                    layout+: {
                      GridLayoutKind+: value,
                    },
                  },
                },
              },
              GridLayoutKind+:
                {
                  '#withKind': { 'function': { args: [], help: '' } },
                  withKind(): {
                    spec+: {
                      preferences+: {
                        layout+: {
                          kind: 'GridLayout',
                        },
                      },
                    },
                  },
                  '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSpec(value): {
                    preferences+: {
                      layout+: {
                        spec: value,
                      },
                    },
                  },
                  '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withSpecMixin(value): {
                    preferences+: {
                      layout+: {
                        spec+: value,
                      },
                    },
                  },
                  spec+:
                    {
                      '#withItems': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withItems(value): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                items:
                                  (if std.isArray(value)
                                   then value
                                   else [value]),
                              },
                            },
                          },
                        },
                      },
                      '#withItemsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withItemsMixin(value): {
                        spec+: {
                          preferences+: {
                            layout+: {
                              spec+: {
                                items+:
                                  (if std.isArray(value)
                                   then value
                                   else [value]),
                              },
                            },
                          },
                        },
                      },
                      items+:
                        {
                          '#': { help: '', name: 'items' },
                          '#withKind': { 'function': { args: [], help: '' } },
                          withKind(): {
                            kind: 'GridLayoutItem',
                          },
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
                              '#withElement': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'reference to a PanelKind from dashboard.spec.elements Expressed as JSON Schema reference' } },
                              withElement(value): {
                                spec+: {
                                  element: value,
                                },
                              },
                              '#withElementMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'reference to a PanelKind from dashboard.spec.elements Expressed as JSON Schema reference' } },
                              withElementMixin(value): {
                                spec+: {
                                  element+: value,
                                },
                              },
                              element+:
                                {
                                  '#withKind': { 'function': { args: [], help: '' } },
                                  withKind(): {
                                    spec+: {
                                      element+: {
                                        kind: 'ElementReference',
                                      },
                                    },
                                  },
                                  '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withName(value): {
                                    spec+: {
                                      element+: {
                                        name: value,
                                      },
                                    },
                                  },
                                },
                              '#withHeight': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                              withHeight(value): {
                                spec+: {
                                  height: value,
                                },
                              },
                              '#withRepeat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withRepeat(value): {
                                spec+: {
                                  repeat: value,
                                },
                              },
                              '#withRepeatMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                              withRepeatMixin(value): {
                                spec+: {
                                  repeat+: value,
                                },
                              },
                              repeat+:
                                {
                                  '#withDirection': { 'function': { args: [{ default: null, enums: ['h', 'v'], name: 'value', type: ['string'] }], help: '' } },
                                  withDirection(value): {
                                    spec+: {
                                      repeat+: {
                                        direction: value,
                                      },
                                    },
                                  },
                                  '#withMaxPerRow': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                                  withMaxPerRow(value): {
                                    spec+: {
                                      repeat+: {
                                        maxPerRow: value,
                                      },
                                    },
                                  },
                                  '#withMode': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withMode(value='variable'): {
                                    spec+: {
                                      repeat+: {
                                        mode: value,
                                      },
                                    },
                                  },
                                  '#withModeMixin': { 'function': { args: [{ default: 'variable', enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withModeMixin(value): {
                                    spec+: {
                                      repeat+: {
                                        mode+: value,
                                      },
                                    },
                                  },
                                  mode+:
                                    {
                                      '#withRepeatMode': { 'function': { args: [{ default: null, enums: ['variable'], name: 'value', type: ['string'] }], help: 'other repeat modes will be added in the future: label, frame' } },
                                      withRepeatMode(value): {
                                        spec+: {
                                          repeat+: {
                                            mode+: {
                                              RepeatMode: value,
                                            },
                                          },
                                        },
                                      },
                                    },
                                  '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                                  withValue(value): {
                                    spec+: {
                                      repeat+: {
                                        value: value,
                                      },
                                    },
                                  },
                                },
                              '#withWidth': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                              withWidth(value): {
                                spec+: {
                                  width: value,
                                },
                              },
                              '#withX': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                              withX(value): {
                                spec+: {
                                  x: value,
                                },
                              },
                              '#withY': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                              withY(value): {
                                spec+: {
                                  y: value,
                                },
                              },
                            },
                        },
                    },
                },
            },
        },
      '#withPreload': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: "When set to true, the dashboard will load all panels in the dashboard when it's loaded." } },
      withPreload(value=true): {
        spec+: {
          preload: value,
        },
      },
      '#withRevision': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Plugins only. The version of the dashboard installed together with the plugin.\nThis is used to determine if the dashboard should be updated when the plugin is updated.' } },
      withRevision(value): {
        spec+: {
          revision: value,
        },
      },
      '#withTags': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Tags associated with dashboard.' } },
      withTags(value): {
        spec+: {
          tags:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withTagsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Tags associated with dashboard.' } },
      withTagsMixin(value): {
        spec+: {
          tags+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withTimeSettings': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Time configuration\nIt defines the default time config for the time picker, the refresh picker for the specific dashboard.' } },
      withTimeSettings(value): {
        spec+: {
          timeSettings: value,
        },
      },
      '#withTimeSettingsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Time configuration\nIt defines the default time config for the time picker, the refresh picker for the specific dashboard.' } },
      withTimeSettingsMixin(value): {
        spec+: {
          timeSettings+: value,
        },
      },
      timeSettings+:
        {
          '#withAutoRefresh': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: 'Refresh rate of dashboard. Represented via interval string, e.g. "5s", "1m", "1h", "1d".\nv1: refresh' } },
          withAutoRefresh(value=''): {
            spec+: {
              timeSettings+: {
                autoRefresh: value,
              },
            },
          },
          '#withAutoRefreshIntervals': { 'function': { args: [{ default: ['5s', '10s', '30s', '1m', '5m', '15m', '30m', '1h', '2h', '1d'], enums: null, name: 'value', type: ['array'] }], help: 'Interval options available in the refresh picker dropdown.\nv1: timepicker.refresh_intervals' } },
          withAutoRefreshIntervals(value): {
            spec+: {
              timeSettings+: {
                autoRefreshIntervals:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withAutoRefreshIntervalsMixin': { 'function': { args: [{ default: ['5s', '10s', '30s', '1m', '5m', '15m', '30m', '1h', '2h', '1d'], enums: null, name: 'value', type: ['array'] }], help: 'Interval options available in the refresh picker dropdown.\nv1: timepicker.refresh_intervals' } },
          withAutoRefreshIntervalsMixin(value): {
            spec+: {
              timeSettings+: {
                autoRefreshIntervals+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withFiscalYearStartMonth': { 'function': { args: [{ default: 0, enums: null, name: 'value', type: ['integer'] }], help: 'The month that the fiscal year starts on. 0 = January, 11 = December' } },
          withFiscalYearStartMonth(value=0): {
            spec+: {
              timeSettings+: {
                fiscalYearStartMonth: value,
              },
            },
          },
          '#withFrom': { 'function': { args: [{ default: 'now-6h', enums: null, name: 'value', type: ['string'] }], help: 'Start time range for dashboard.\nAccepted values are relative time strings like "now-6h" or absolute time strings like "2020-07-10T08:00:00.000Z".' } },
          withFrom(value='now-6h'): {
            spec+: {
              timeSettings+: {
                from: value,
              },
            },
          },
          '#withHideTimepicker': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether timepicker is visible or not.\nv1: timepicker.hidden' } },
          withHideTimepicker(value=true): {
            spec+: {
              timeSettings+: {
                hideTimepicker: value,
              },
            },
          },
          '#withNowDelay': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Override the now time by entering a time delay. Use this option to accommodate known delays in data aggregation to avoid null values.\nv1: timepicker.nowDelay' } },
          withNowDelay(value): {
            spec+: {
              timeSettings+: {
                nowDelay: value,
              },
            },
          },
          '#withQuickRanges': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Selectable options available in the time picker dropdown. Has no effect on provisioned dashboard.\nv1: timepicker.quick_ranges , not exposed in the UI' } },
          withQuickRanges(value): {
            spec+: {
              timeSettings+: {
                quickRanges:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withQuickRangesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Selectable options available in the time picker dropdown. Has no effect on provisioned dashboard.\nv1: timepicker.quick_ranges , not exposed in the UI' } },
          withQuickRangesMixin(value): {
            spec+: {
              timeSettings+: {
                quickRanges+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          quickRanges+:
            {
              '#': { help: '', name: 'quickRanges' },
              '#withDisplay': { 'function': { args: [{ default: 'Last 6 hours', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withDisplay(value='Last 6 hours'): {
                display: value,
              },
              '#withFrom': { 'function': { args: [{ default: 'now-6h', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withFrom(value='now-6h'): {
                from: value,
              },
              '#withTo': { 'function': { args: [{ default: 'now', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withTo(value='now'): {
                to: value,
              },
            },
          '#withTimezone': { 'function': { args: [{ default: 'browser', enums: null, name: 'value', type: ['string'] }], help: 'Timezone of dashboard. Accepted values are IANA TZDB zone ID or "browser" or "utc".' } },
          withTimezone(value='browser'): {
            spec+: {
              timeSettings+: {
                timezone: value,
              },
            },
          },
          '#withTo': { 'function': { args: [{ default: 'now', enums: null, name: 'value', type: ['string'] }], help: 'End time range for dashboard.\nAccepted values are relative time strings like "now-6h" or absolute time strings like "2020-07-10T08:00:00.000Z".' } },
          withTo(value='now'): {
            spec+: {
              timeSettings+: {
                to: value,
              },
            },
          },
          '#withWeekStart': { 'function': { args: [{ default: null, enums: ['saturday', 'monday', 'sunday'], name: 'value', type: ['string'] }], help: 'Day when the week starts. Expressed by the name of the day in lowercase, e.g. "monday".' } },
          withWeekStart(value): {
            spec+: {
              timeSettings+: {
                weekStart: value,
              },
            },
          },
        },
      '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Title of dashboard.' } },
      withTitle(value): {
        spec+: {
          title: value,
        },
      },
      '#withVariables': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Configured template variables.' } },
      withVariables(value): {
        spec+: {
          variables:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withVariablesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Configured template variables.' } },
      withVariablesMixin(value): {
        spec+: {
          variables+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      variables+:
        {
          QueryVariableKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                kind: 'QueryVariable',
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Query variable specification' } },
              withSpec(value): {
                spec: value,
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Query variable specification' } },
              withSpecMixin(value): {
                spec+: value,
              },
              spec+:
                {
                  '#withAllValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withAllValue(value): {
                    spec+: {
                      allValue: value,
                    },
                  },
                  '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withAllowCustomValue(value=true): {
                    spec+: {
                      allowCustomValue: value,
                    },
                  },
                  '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrent(value={ text: '', value: '' }): {
                    spec+: {
                      current: value,
                    },
                  },
                  '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrentMixin(value): {
                    spec+: {
                      current+: value,
                    },
                  },
                  current+:
                    {
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        spec+: {
                          current+: {
                            properties: value,
                          },
                        },
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        spec+: {
                          current+: {
                            properties+: value,
                          },
                        },
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        spec+: {
                          current+: {
                            selected: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        spec+: {
                          current+: {
                            text: value,
                          },
                        },
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        spec+: {
                          current+: {
                            text+: value,
                          },
                        },
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        spec+: {
                          current+: {
                            value: value,
                          },
                        },
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        spec+: {
                          current+: {
                            value+: value,
                          },
                        },
                      },
                    },
                  '#withDefinition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDefinition(value): {
                    spec+: {
                      definition: value,
                    },
                  },
                  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDescription(value): {
                    spec+: {
                      description: value,
                    },
                  },
                  '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                  withHide(value='dontHide'): {
                    spec+: {
                      hide: value,
                    },
                  },
                  '#withIncludeAll': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withIncludeAll(value=true): {
                    spec+: {
                      includeAll: value,
                    },
                  },
                  '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withLabel(value): {
                    spec+: {
                      label: value,
                    },
                  },
                  '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withMulti(value=true): {
                    spec+: {
                      multi: value,
                    },
                  },
                  '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value=''): {
                    spec+: {
                      name: value,
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptions(value): {
                    spec+: {
                      options:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptionsMixin(value): {
                    spec+: {
                      options+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  options+:
                    {
                      '#': { help: '', name: 'options' },
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        properties: value,
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        properties+: value,
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        selected: value,
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        text: value,
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        text+: value,
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        value: value,
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        value+: value,
                      },
                    },
                  '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOrigin(value): {
                    spec+: {
                      origin: value,
                    },
                  },
                  '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOriginMixin(value): {
                    spec+: {
                      origin+: value,
                    },
                  },
                  origin+:
                    {
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                      withGroup(value): {
                        spec+: {
                          origin+: {
                            group: value,
                          },
                        },
                      },
                      '#withType': { 'function': { args: [], help: '' } },
                      withType(): {
                        spec+: {
                          origin+: {
                            type: 'datasource',
                          },
                        },
                      },
                    },
                  '#withPlaceholder': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withPlaceholder(value): {
                    spec+: {
                      placeholder: value,
                    },
                  },
                  '#withQuery': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withQuery(value): {
                    spec+: {
                      query: value,
                    },
                  },
                  '#withQueryMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withQueryMixin(value): {
                    spec+: {
                      query+: value,
                    },
                  },
                  query+:
                    {
                      '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'New type for datasource reference\nNot creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.' } },
                      withDatasource(value): {
                        spec+: {
                          query+: {
                            datasource: value,
                          },
                        },
                      },
                      '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'New type for datasource reference\nNot creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.' } },
                      withDatasourceMixin(value): {
                        spec+: {
                          query+: {
                            datasource+: value,
                          },
                        },
                      },
                      datasource+:
                        {
                          '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                          withName(value): {
                            spec+: {
                              query+: {
                                datasource+: {
                                  name: value,
                                },
                              },
                            },
                          },
                        },
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withGroup(value): {
                        spec+: {
                          query+: {
                            group: value,
                          },
                        },
                      },
                      '#withKind': { 'function': { args: [], help: '' } },
                      withKind(): {
                        spec+: {
                          query+: {
                            kind: 'DataQuery',
                          },
                        },
                      },
                      '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                      withLabels(value): {
                        spec+: {
                          query+: {
                            labels: value,
                          },
                        },
                      },
                      '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                      withLabelsMixin(value): {
                        spec+: {
                          query+: {
                            labels+: value,
                          },
                        },
                      },
                      '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                      withSpec(value): {
                        query+: {
                          spec: value,
                        },
                      },
                      '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                      withSpecMixin(value): {
                        query+: {
                          spec+: value,
                        },
                      },
                      '#withVersion': { 'function': { args: [{ default: 'v0', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withVersion(value='v0'): {
                        spec+: {
                          query+: {
                            version: value,
                          },
                        },
                      },
                    },
                  '#withRefresh': { 'function': { args: [{ default: 'never', enums: ['never', 'onDashboardLoad', 'onTimeRangeChanged'], name: 'value', type: ['string'] }], help: 'Options to config when to refresh a variable\n`never`: Never refresh the variable\n`onDashboardLoad`: Queries the data source every time the dashboard loads.\n`onTimeRangeChanged`: Queries the data source when the dashboard time range changes.' } },
                  withRefresh(value='never'): {
                    spec+: {
                      refresh: value,
                    },
                  },
                  '#withRegex': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withRegex(value=''): {
                    spec+: {
                      regex: value,
                    },
                  },
                  '#withRegexApplyTo': { 'function': { args: [{ default: 'value', enums: ['value', 'text'], name: 'value', type: ['string'] }], help: 'Determine whether regex applies to variable value or display text\nAccepted values are `value` (apply to value used in queries) or `text` (apply to display text shown to users)' } },
                  withRegexApplyTo(value='value'): {
                    spec+: {
                      regexApplyTo: value,
                    },
                  },
                  '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withSkipUrlSync(value=true): {
                    spec+: {
                      skipUrlSync: value,
                    },
                  },
                  '#withSort': { 'function': { args: [{ default: null, enums: ['disabled', 'alphabeticalAsc', 'alphabeticalDesc', 'numericalAsc', 'numericalDesc', 'alphabeticalCaseInsensitiveAsc', 'alphabeticalCaseInsensitiveDesc', 'naturalAsc', 'naturalDesc'], name: 'value', type: ['string'] }], help: 'Sort variable options\nAccepted values are:\n`disabled`: No sorting\n`alphabeticalAsc`: Alphabetical ASC\n`alphabeticalDesc`: Alphabetical DESC\n`numericalAsc`: Numerical ASC\n`numericalDesc`: Numerical DESC\n`alphabeticalCaseInsensitiveAsc`: Alphabetical Case Insensitive ASC\n`alphabeticalCaseInsensitiveDesc`: Alphabetical Case Insensitive DESC\n`naturalAsc`: Natural ASC\n`naturalDesc`: Natural DESC\nVariableSort enum with default value' } },
                  withSort(value): {
                    spec+: {
                      sort: value,
                    },
                  },
                  '#withStaticOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withStaticOptions(value): {
                    spec+: {
                      staticOptions:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  '#withStaticOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withStaticOptionsMixin(value): {
                    spec+: {
                      staticOptions+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  staticOptions+:
                    {
                      '#': { help: '', name: 'staticOptions' },
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        properties: value,
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        properties+: value,
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        selected: value,
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        text: value,
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        text+: value,
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        value: value,
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        value+: value,
                      },
                    },
                  '#withStaticOptionsOrder': { 'function': { args: [{ default: null, enums: ['before', 'after', 'sorted'], name: 'value', type: ['string'] }], help: '' } },
                  withStaticOptionsOrder(value): {
                    spec+: {
                      staticOptionsOrder: value,
                    },
                  },
                },
            },
          TextVariableKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                kind: 'TextVariable',
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Text variable specification' } },
              withSpec(value): {
                spec: value,
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Text variable specification' } },
              withSpecMixin(value): {
                spec+: value,
              },
              spec+:
                {
                  '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrent(value={ text: '', value: '' }): {
                    spec+: {
                      current: value,
                    },
                  },
                  '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrentMixin(value): {
                    spec+: {
                      current+: value,
                    },
                  },
                  current+:
                    {
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        spec+: {
                          current+: {
                            properties: value,
                          },
                        },
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        spec+: {
                          current+: {
                            properties+: value,
                          },
                        },
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        spec+: {
                          current+: {
                            selected: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        spec+: {
                          current+: {
                            text: value,
                          },
                        },
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        spec+: {
                          current+: {
                            text+: value,
                          },
                        },
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        spec+: {
                          current+: {
                            value: value,
                          },
                        },
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        spec+: {
                          current+: {
                            value+: value,
                          },
                        },
                      },
                    },
                  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDescription(value): {
                    spec+: {
                      description: value,
                    },
                  },
                  '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                  withHide(value='dontHide'): {
                    spec+: {
                      hide: value,
                    },
                  },
                  '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withLabel(value): {
                    spec+: {
                      label: value,
                    },
                  },
                  '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value=''): {
                    spec+: {
                      name: value,
                    },
                  },
                  '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOrigin(value): {
                    spec+: {
                      origin: value,
                    },
                  },
                  '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOriginMixin(value): {
                    spec+: {
                      origin+: value,
                    },
                  },
                  origin+:
                    {
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                      withGroup(value): {
                        spec+: {
                          origin+: {
                            group: value,
                          },
                        },
                      },
                      '#withType': { 'function': { args: [], help: '' } },
                      withType(): {
                        spec+: {
                          origin+: {
                            type: 'datasource',
                          },
                        },
                      },
                    },
                  '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withQuery(value=''): {
                    spec+: {
                      query: value,
                    },
                  },
                  '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withSkipUrlSync(value=true): {
                    spec+: {
                      skipUrlSync: value,
                    },
                  },
                },
            },
          ConstantVariableKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                kind: 'ConstantVariable',
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Constant variable specification' } },
              withSpec(value): {
                spec: value,
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Constant variable specification' } },
              withSpecMixin(value): {
                spec+: value,
              },
              spec+:
                {
                  '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrent(value={ text: '', value: '' }): {
                    spec+: {
                      current: value,
                    },
                  },
                  '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrentMixin(value): {
                    spec+: {
                      current+: value,
                    },
                  },
                  current+:
                    {
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        spec+: {
                          current+: {
                            properties: value,
                          },
                        },
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        spec+: {
                          current+: {
                            properties+: value,
                          },
                        },
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        spec+: {
                          current+: {
                            selected: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        spec+: {
                          current+: {
                            text: value,
                          },
                        },
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        spec+: {
                          current+: {
                            text+: value,
                          },
                        },
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        spec+: {
                          current+: {
                            value: value,
                          },
                        },
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        spec+: {
                          current+: {
                            value+: value,
                          },
                        },
                      },
                    },
                  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDescription(value): {
                    spec+: {
                      description: value,
                    },
                  },
                  '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                  withHide(value='dontHide'): {
                    spec+: {
                      hide: value,
                    },
                  },
                  '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withLabel(value): {
                    spec+: {
                      label: value,
                    },
                  },
                  '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value=''): {
                    spec+: {
                      name: value,
                    },
                  },
                  '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOrigin(value): {
                    spec+: {
                      origin: value,
                    },
                  },
                  '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOriginMixin(value): {
                    spec+: {
                      origin+: value,
                    },
                  },
                  origin+:
                    {
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                      withGroup(value): {
                        spec+: {
                          origin+: {
                            group: value,
                          },
                        },
                      },
                      '#withType': { 'function': { args: [], help: '' } },
                      withType(): {
                        spec+: {
                          origin+: {
                            type: 'datasource',
                          },
                        },
                      },
                    },
                  '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withQuery(value=''): {
                    spec+: {
                      query: value,
                    },
                  },
                  '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withSkipUrlSync(value=true): {
                    spec+: {
                      skipUrlSync: value,
                    },
                  },
                },
            },
          DatasourceVariableKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                kind: 'DatasourceVariable',
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Datasource variable specification' } },
              withSpec(value): {
                spec: value,
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Datasource variable specification' } },
              withSpecMixin(value): {
                spec+: value,
              },
              spec+:
                {
                  '#withAllValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withAllValue(value): {
                    spec+: {
                      allValue: value,
                    },
                  },
                  '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withAllowCustomValue(value=true): {
                    spec+: {
                      allowCustomValue: value,
                    },
                  },
                  '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrent(value={ text: '', value: '' }): {
                    spec+: {
                      current: value,
                    },
                  },
                  '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrentMixin(value): {
                    spec+: {
                      current+: value,
                    },
                  },
                  current+:
                    {
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        spec+: {
                          current+: {
                            properties: value,
                          },
                        },
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        spec+: {
                          current+: {
                            properties+: value,
                          },
                        },
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        spec+: {
                          current+: {
                            selected: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        spec+: {
                          current+: {
                            text: value,
                          },
                        },
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        spec+: {
                          current+: {
                            text+: value,
                          },
                        },
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        spec+: {
                          current+: {
                            value: value,
                          },
                        },
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        spec+: {
                          current+: {
                            value+: value,
                          },
                        },
                      },
                    },
                  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDescription(value): {
                    spec+: {
                      description: value,
                    },
                  },
                  '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                  withHide(value='dontHide'): {
                    spec+: {
                      hide: value,
                    },
                  },
                  '#withIncludeAll': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withIncludeAll(value=true): {
                    spec+: {
                      includeAll: value,
                    },
                  },
                  '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withLabel(value): {
                    spec+: {
                      label: value,
                    },
                  },
                  '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withMulti(value=true): {
                    spec+: {
                      multi: value,
                    },
                  },
                  '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value=''): {
                    spec+: {
                      name: value,
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptions(value): {
                    spec+: {
                      options:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptionsMixin(value): {
                    spec+: {
                      options+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  options+:
                    {
                      '#': { help: '', name: 'options' },
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        properties: value,
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        properties+: value,
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        selected: value,
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        text: value,
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        text+: value,
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        value: value,
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        value+: value,
                      },
                    },
                  '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOrigin(value): {
                    spec+: {
                      origin: value,
                    },
                  },
                  '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOriginMixin(value): {
                    spec+: {
                      origin+: value,
                    },
                  },
                  origin+:
                    {
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                      withGroup(value): {
                        spec+: {
                          origin+: {
                            group: value,
                          },
                        },
                      },
                      '#withType': { 'function': { args: [], help: '' } },
                      withType(): {
                        spec+: {
                          origin+: {
                            type: 'datasource',
                          },
                        },
                      },
                    },
                  '#withPluginId': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withPluginId(value=''): {
                    spec+: {
                      pluginId: value,
                    },
                  },
                  '#withRefresh': { 'function': { args: [{ default: 'never', enums: ['never', 'onDashboardLoad', 'onTimeRangeChanged'], name: 'value', type: ['string'] }], help: 'Options to config when to refresh a variable\n`never`: Never refresh the variable\n`onDashboardLoad`: Queries the data source every time the dashboard loads.\n`onTimeRangeChanged`: Queries the data source when the dashboard time range changes.' } },
                  withRefresh(value='never'): {
                    spec+: {
                      refresh: value,
                    },
                  },
                  '#withRegex': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withRegex(value=''): {
                    spec+: {
                      regex: value,
                    },
                  },
                  '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withSkipUrlSync(value=true): {
                    spec+: {
                      skipUrlSync: value,
                    },
                  },
                },
            },
          IntervalVariableKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                kind: 'IntervalVariable',
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Interval variable specification' } },
              withSpec(value): {
                spec: value,
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Interval variable specification' } },
              withSpecMixin(value): {
                spec+: value,
              },
              spec+:
                {
                  '#withAuto': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withAuto(value=true): {
                    spec+: {
                      auto: value,
                    },
                  },
                  '#withAutoCount': { 'function': { args: [{ default: 0, enums: null, name: 'value', type: ['integer'] }], help: '' } },
                  withAutoCount(value=0): {
                    spec+: {
                      auto_count: value,
                    },
                  },
                  '#withAutoMin': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withAutoMin(value=''): {
                    spec+: {
                      auto_min: value,
                    },
                  },
                  '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrent(value={ text: '', value: '' }): {
                    spec+: {
                      current: value,
                    },
                  },
                  '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrentMixin(value): {
                    spec+: {
                      current+: value,
                    },
                  },
                  current+:
                    {
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        spec+: {
                          current+: {
                            properties: value,
                          },
                        },
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        spec+: {
                          current+: {
                            properties+: value,
                          },
                        },
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        spec+: {
                          current+: {
                            selected: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        spec+: {
                          current+: {
                            text: value,
                          },
                        },
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        spec+: {
                          current+: {
                            text+: value,
                          },
                        },
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        spec+: {
                          current+: {
                            value: value,
                          },
                        },
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        spec+: {
                          current+: {
                            value+: value,
                          },
                        },
                      },
                    },
                  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDescription(value): {
                    spec+: {
                      description: value,
                    },
                  },
                  '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                  withHide(value='dontHide'): {
                    spec+: {
                      hide: value,
                    },
                  },
                  '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withLabel(value): {
                    spec+: {
                      label: value,
                    },
                  },
                  '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value=''): {
                    spec+: {
                      name: value,
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptions(value): {
                    spec+: {
                      options:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptionsMixin(value): {
                    spec+: {
                      options+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  options+:
                    {
                      '#': { help: '', name: 'options' },
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        properties: value,
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        properties+: value,
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        selected: value,
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        text: value,
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        text+: value,
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        value: value,
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        value+: value,
                      },
                    },
                  '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOrigin(value): {
                    spec+: {
                      origin: value,
                    },
                  },
                  '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOriginMixin(value): {
                    spec+: {
                      origin+: value,
                    },
                  },
                  origin+:
                    {
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                      withGroup(value): {
                        spec+: {
                          origin+: {
                            group: value,
                          },
                        },
                      },
                      '#withType': { 'function': { args: [], help: '' } },
                      withType(): {
                        spec+: {
                          origin+: {
                            type: 'datasource',
                          },
                        },
                      },
                    },
                  '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withQuery(value=''): {
                    spec+: {
                      query: value,
                    },
                  },
                  '#withRefresh': { 'function': { args: [], help: '' } },
                  withRefresh(): {
                    spec+: {
                      refresh: 'onTimeRangeChanged',
                    },
                  },
                  '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withSkipUrlSync(value=true): {
                    spec+: {
                      skipUrlSync: value,
                    },
                  },
                },
            },
          CustomVariableKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                kind: 'CustomVariable',
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Custom variable specification' } },
              withSpec(value): {
                spec: value,
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Custom variable specification' } },
              withSpecMixin(value): {
                spec+: value,
              },
              spec+:
                {
                  '#withAllValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withAllValue(value): {
                    spec+: {
                      allValue: value,
                    },
                  },
                  '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withAllowCustomValue(value=true): {
                    spec+: {
                      allowCustomValue: value,
                    },
                  },
                  '#withCurrent': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrent(value): {
                    spec+: {
                      current: value,
                    },
                  },
                  '#withCurrentMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrentMixin(value): {
                    spec+: {
                      current+: value,
                    },
                  },
                  current+:
                    {
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        spec+: {
                          current+: {
                            properties: value,
                          },
                        },
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        spec+: {
                          current+: {
                            properties+: value,
                          },
                        },
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        spec+: {
                          current+: {
                            selected: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        spec+: {
                          current+: {
                            text: value,
                          },
                        },
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        spec+: {
                          current+: {
                            text+: value,
                          },
                        },
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        spec+: {
                          current+: {
                            value: value,
                          },
                        },
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        spec+: {
                          current+: {
                            value+: value,
                          },
                        },
                      },
                    },
                  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDescription(value): {
                    spec+: {
                      description: value,
                    },
                  },
                  '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                  withHide(value='dontHide'): {
                    spec+: {
                      hide: value,
                    },
                  },
                  '#withIncludeAll': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withIncludeAll(value=true): {
                    spec+: {
                      includeAll: value,
                    },
                  },
                  '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withLabel(value): {
                    spec+: {
                      label: value,
                    },
                  },
                  '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withMulti(value=true): {
                    spec+: {
                      multi: value,
                    },
                  },
                  '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value=''): {
                    spec+: {
                      name: value,
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptions(value): {
                    spec+: {
                      options:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptionsMixin(value): {
                    spec+: {
                      options+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  options+:
                    {
                      '#': { help: '', name: 'options' },
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        properties: value,
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        properties+: value,
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        selected: value,
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        text: value,
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        text+: value,
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        value: value,
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        value+: value,
                      },
                    },
                  '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOrigin(value): {
                    spec+: {
                      origin: value,
                    },
                  },
                  '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOriginMixin(value): {
                    spec+: {
                      origin+: value,
                    },
                  },
                  origin+:
                    {
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                      withGroup(value): {
                        spec+: {
                          origin+: {
                            group: value,
                          },
                        },
                      },
                      '#withType': { 'function': { args: [], help: '' } },
                      withType(): {
                        spec+: {
                          origin+: {
                            type: 'datasource',
                          },
                        },
                      },
                    },
                  '#withQuery': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withQuery(value=''): {
                    spec+: {
                      query: value,
                    },
                  },
                  '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withSkipUrlSync(value=true): {
                    spec+: {
                      skipUrlSync: value,
                    },
                  },
                  '#withValuesFormat': { 'function': { args: [{ default: null, enums: ['csv', 'json'], name: 'value', type: ['string'] }], help: '' } },
                  withValuesFormat(value): {
                    spec+: {
                      valuesFormat: value,
                    },
                  },
                },
            },
          GroupByVariableKind+:
            {
              '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withDatasource(value): {
                datasource: value,
              },
              '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withDatasourceMixin(value): {
                datasource+: value,
              },
              datasource+:
                {
                  '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value): {
                    datasource+: {
                      name: value,
                    },
                  },
                },
              '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withGroup(value): {
                group: value,
              },
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                kind: 'GroupByVariable',
              },
              '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withLabels(value): {
                labels: value,
              },
              '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withLabelsMixin(value): {
                labels+: value,
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'GroupBy variable specification' } },
              withSpec(value): {
                spec: value,
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'GroupBy variable specification' } },
              withSpecMixin(value): {
                spec+: value,
              },
              spec+:
                {
                  '#withCurrent': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrent(value={ text: '', value: '' }): {
                    spec+: {
                      current: value,
                    },
                  },
                  '#withCurrentMixin': { 'function': { args: [{ default: { text: '', value: '' }, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withCurrentMixin(value): {
                    spec+: {
                      current+: value,
                    },
                  },
                  current+:
                    {
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        spec+: {
                          current+: {
                            properties: value,
                          },
                        },
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        spec+: {
                          current+: {
                            properties+: value,
                          },
                        },
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        spec+: {
                          current+: {
                            selected: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        spec+: {
                          current+: {
                            text: value,
                          },
                        },
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        spec+: {
                          current+: {
                            text+: value,
                          },
                        },
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        spec+: {
                          current+: {
                            value: value,
                          },
                        },
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        spec+: {
                          current+: {
                            value+: value,
                          },
                        },
                      },
                    },
                  '#withDefaultValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withDefaultValue(value): {
                    spec+: {
                      defaultValue: value,
                    },
                  },
                  '#withDefaultValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Variable option specification' } },
                  withDefaultValueMixin(value): {
                    spec+: {
                      defaultValue+: value,
                    },
                  },
                  defaultValue+:
                    {
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        spec+: {
                          defaultValue+: {
                            properties: value,
                          },
                        },
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        spec+: {
                          defaultValue+: {
                            properties+: value,
                          },
                        },
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        spec+: {
                          defaultValue+: {
                            selected: value,
                          },
                        },
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        spec+: {
                          defaultValue+: {
                            text: value,
                          },
                        },
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        spec+: {
                          defaultValue+: {
                            text+: value,
                          },
                        },
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        spec+: {
                          defaultValue+: {
                            value: value,
                          },
                        },
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        spec+: {
                          defaultValue+: {
                            value+: value,
                          },
                        },
                      },
                    },
                  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDescription(value): {
                    spec+: {
                      description: value,
                    },
                  },
                  '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                  withHide(value='dontHide'): {
                    spec+: {
                      hide: value,
                    },
                  },
                  '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withLabel(value): {
                    spec+: {
                      label: value,
                    },
                  },
                  '#withMulti': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withMulti(value=true): {
                    spec+: {
                      multi: value,
                    },
                  },
                  '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value=''): {
                    spec+: {
                      name: value,
                    },
                  },
                  '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptions(value): {
                    spec+: {
                      options:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withOptionsMixin(value): {
                    spec+: {
                      options+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  options+:
                    {
                      '#': { help: '', name: 'options' },
                      '#withProperties': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withProperties(value): {
                        properties: value,
                      },
                      '#withPropertiesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Additional properties for multi-props variables' } },
                      withPropertiesMixin(value): {
                        properties+: value,
                      },
                      '#withSelected': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the option is selected or not' } },
                      withSelected(value=true): {
                        selected: value,
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withText(value): {
                        text: value,
                      },
                      '#withTextMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Text to be displayed for the option' } },
                      withTextMixin(value): {
                        text+: value,
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValue(value): {
                        value: value,
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'array'] }], help: 'Value of the option' } },
                      withValueMixin(value): {
                        value+: value,
                      },
                    },
                  '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOrigin(value): {
                    spec+: {
                      origin: value,
                    },
                  },
                  '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOriginMixin(value): {
                    spec+: {
                      origin+: value,
                    },
                  },
                  origin+:
                    {
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                      withGroup(value): {
                        spec+: {
                          origin+: {
                            group: value,
                          },
                        },
                      },
                      '#withType': { 'function': { args: [], help: '' } },
                      withType(): {
                        spec+: {
                          origin+: {
                            type: 'datasource',
                          },
                        },
                      },
                    },
                  '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withSkipUrlSync(value=true): {
                    spec+: {
                      skipUrlSync: value,
                    },
                  },
                },
            },
          AdhocVariableKind+:
            {
              '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withDatasource(value): {
                datasource: value,
              },
              '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withDatasourceMixin(value): {
                datasource+: value,
              },
              datasource+:
                {
                  '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value): {
                    datasource+: {
                      name: value,
                    },
                  },
                },
              '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withGroup(value): {
                group: value,
              },
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                kind: 'AdhocVariable',
              },
              '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withLabels(value): {
                labels: value,
              },
              '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withLabelsMixin(value): {
                labels+: value,
              },
              '#withSpec': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Adhoc variable specification' } },
              withSpec(value): {
                spec: value,
              },
              '#withSpecMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Adhoc variable specification' } },
              withSpecMixin(value): {
                spec+: value,
              },
              spec+:
                {
                  '#withAllowCustomValue': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withAllowCustomValue(value=true): {
                    spec+: {
                      allowCustomValue: value,
                    },
                  },
                  '#withBaseFilters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withBaseFilters(value): {
                    spec+: {
                      baseFilters:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  '#withBaseFiltersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withBaseFiltersMixin(value): {
                    spec+: {
                      baseFilters+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  baseFilters+:
                    {
                      '#': { help: '', name: 'baseFilters' },
                      '#withCondition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '@deprecated' } },
                      withCondition(value): {
                        condition: value,
                      },
                      '#withForceEdit': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withForceEdit(value=true): {
                        forceEdit: value,
                      },
                      '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withKey(value): {
                        key: value,
                      },
                      '#withKeyLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withKeyLabel(value): {
                        keyLabel: value,
                      },
                      '#withOperator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withOperator(value): {
                        operator: value,
                      },
                      '#withOrigin': { 'function': { args: [], help: 'Determine the origin of the adhoc variable filter' } },
                      withOrigin(): {
                        origin: 'dashboard',
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withValue(value): {
                        value: value,
                      },
                      '#withValueLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withValueLabels(value): {
                        valueLabels:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                      '#withValueLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withValueLabelsMixin(value): {
                        valueLabels+:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                      '#withValues': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withValues(value): {
                        values:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                      '#withValuesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withValuesMixin(value): {
                        values+:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                    },
                  '#withDefaultKeys': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withDefaultKeys(value): {
                    spec+: {
                      defaultKeys:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  '#withDefaultKeysMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withDefaultKeysMixin(value): {
                    spec+: {
                      defaultKeys+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  defaultKeys+:
                    {
                      '#': { help: '', name: 'defaultKeys' },
                      '#withExpandable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withExpandable(value=true): {
                        expandable: value,
                      },
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withGroup(value): {
                        group: value,
                      },
                      '#withText': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withText(value): {
                        text: value,
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'number'] }], help: '' } },
                      withValue(value): {
                        value: value,
                      },
                      '#withValueMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string', 'number'] }], help: '' } },
                      withValueMixin(value): {
                        value+: value,
                      },
                    },
                  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDescription(value): {
                    spec+: {
                      description: value,
                    },
                  },
                  '#withEnableGroupBy': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether the group-by operator is enabled in the ad hoc filter combobox.' } },
                  withEnableGroupBy(value=true): {
                    spec+: {
                      enableGroupBy: value,
                    },
                  },
                  '#withFilters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withFilters(value): {
                    spec+: {
                      filters:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  '#withFiltersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withFiltersMixin(value): {
                    spec+: {
                      filters+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                  filters+:
                    {
                      '#': { help: '', name: 'filters' },
                      '#withCondition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '@deprecated' } },
                      withCondition(value): {
                        condition: value,
                      },
                      '#withForceEdit': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                      withForceEdit(value=true): {
                        forceEdit: value,
                      },
                      '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withKey(value): {
                        key: value,
                      },
                      '#withKeyLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withKeyLabel(value): {
                        keyLabel: value,
                      },
                      '#withOperator': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withOperator(value): {
                        operator: value,
                      },
                      '#withOrigin': { 'function': { args: [], help: 'Determine the origin of the adhoc variable filter' } },
                      withOrigin(): {
                        origin: 'dashboard',
                      },
                      '#withValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withValue(value): {
                        value: value,
                      },
                      '#withValueLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withValueLabels(value): {
                        valueLabels:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                      '#withValueLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withValueLabelsMixin(value): {
                        valueLabels+:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                      '#withValues': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withValues(value): {
                        values:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                      '#withValuesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withValuesMixin(value): {
                        values+:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                    },
                  '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                  withHide(value='dontHide'): {
                    spec+: {
                      hide: value,
                    },
                  },
                  '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withLabel(value): {
                    spec+: {
                      label: value,
                    },
                  },
                  '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value=''): {
                    spec+: {
                      name: value,
                    },
                  },
                  '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOrigin(value): {
                    spec+: {
                      origin: value,
                    },
                  },
                  '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOriginMixin(value): {
                    spec+: {
                      origin+: value,
                    },
                  },
                  origin+:
                    {
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                      withGroup(value): {
                        spec+: {
                          origin+: {
                            group: value,
                          },
                        },
                      },
                      '#withType': { 'function': { args: [], help: '' } },
                      withType(): {
                        spec+: {
                          origin+: {
                            type: 'datasource',
                          },
                        },
                      },
                    },
                  '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withSkipUrlSync(value=true): {
                    spec+: {
                      skipUrlSync: value,
                    },
                  },
                },
            },
          SwitchVariableKind+:
            {
              '#withKind': { 'function': { args: [], help: '' } },
              withKind(): {
                kind: 'SwitchVariable',
              },
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
                  '#withCurrent': { 'function': { args: [{ default: 'false', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withCurrent(value='false'): {
                    spec+: {
                      current: value,
                    },
                  },
                  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDescription(value): {
                    spec+: {
                      description: value,
                    },
                  },
                  '#withDisabledValue': { 'function': { args: [{ default: 'false', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withDisabledValue(value='false'): {
                    spec+: {
                      disabledValue: value,
                    },
                  },
                  '#withEnabledValue': { 'function': { args: [{ default: 'true', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withEnabledValue(value='true'): {
                    spec+: {
                      enabledValue: value,
                    },
                  },
                  '#withHide': { 'function': { args: [{ default: 'dontHide', enums: ['dontHide', 'hideLabel', 'hideVariable', 'inControlsMenu'], name: 'value', type: ['string'] }], help: 'Determine if the variable shows on dashboard\nAccepted values are `dontHide` (show label and value), `hideLabel` (show value only), `hideVariable` (show nothing), `inControlsMenu` (show in a drop-down menu).' } },
                  withHide(value='dontHide'): {
                    spec+: {
                      hide: value,
                    },
                  },
                  '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withLabel(value): {
                    spec+: {
                      label: value,
                    },
                  },
                  '#withName': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value=''): {
                    spec+: {
                      name: value,
                    },
                  },
                  '#withOrigin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOrigin(value): {
                    spec+: {
                      origin: value,
                    },
                  },
                  '#withOriginMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Source information for controls (e.g. variables or links)' } },
                  withOriginMixin(value): {
                    spec+: {
                      origin+: value,
                    },
                  },
                  origin+:
                    {
                      '#withGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
                      withGroup(value): {
                        spec+: {
                          origin+: {
                            group: value,
                          },
                        },
                      },
                      '#withType': { 'function': { args: [], help: '' } },
                      withType(): {
                        spec+: {
                          origin+: {
                            type: 'datasource',
                          },
                        },
                      },
                    },
                  '#withSkipUrlSync': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withSkipUrlSync(value=true): {
                    spec+: {
                      skipUrlSync: value,
                    },
                  },
                },
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
      help: 'Creates a new dashboard.Dashboard resource.',
    },
  },
  new: function(name, title) {
    apiVersion: 'dashboard.grafana.app/v2',
    kind: 'Dashboard',
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
      apiVersion: 'dashboard.grafana.app/v2',
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
      kind: 'Dashboard',
    },
}
