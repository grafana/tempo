// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.librarypanel', name: 'librarypanel' },
  '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Panel description' } },
  withDescription(value): {
    description: value,
  },
  '#withFolderUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Folder UID' } },
  withFolderUid(value): {
    folderUid: value,
  },
  '#withMeta': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Object storage metadata' } },
  withMeta(value): {
    meta: value,
  },
  '#withMetaMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Object storage metadata' } },
  withMetaMixin(value): {
    meta+: value,
  },
  meta+:
    {
      '#withConnectedDashboards': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withConnectedDashboards(value): {
        meta+: {
          connectedDashboards: value,
        },
      },
      '#withCreated': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withCreated(value): {
        meta+: {
          created: value,
        },
      },
      '#withCreatedBy': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withCreatedBy(value): {
        meta+: {
          createdBy: value,
        },
      },
      '#withCreatedByMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withCreatedByMixin(value): {
        meta+: {
          createdBy+: value,
        },
      },
      createdBy+:
        {
          '#withAvatarUrl': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withAvatarUrl(value): {
            meta+: {
              createdBy+: {
                avatarUrl: value,
              },
            },
          },
          '#withId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
          withId(value): {
            meta+: {
              createdBy+: {
                id: value,
              },
            },
          },
          '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withName(value): {
            meta+: {
              createdBy+: {
                name: value,
              },
            },
          },
        },
      '#withFolderName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withFolderName(value): {
        meta+: {
          folderName: value,
        },
      },
      '#withFolderUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withFolderUid(value): {
        meta+: {
          folderUid: value,
        },
      },
      '#withUpdated': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withUpdated(value): {
        meta+: {
          updated: value,
        },
      },
      '#withUpdatedBy': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withUpdatedBy(value): {
        meta+: {
          updatedBy: value,
        },
      },
      '#withUpdatedByMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withUpdatedByMixin(value): {
        meta+: {
          updatedBy+: value,
        },
      },
      updatedBy+:
        {
          '#withAvatarUrl': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withAvatarUrl(value): {
            meta+: {
              updatedBy+: {
                avatarUrl: value,
              },
            },
          },
          '#withId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
          withId(value): {
            meta+: {
              updatedBy+: {
                id: value,
              },
            },
          },
          '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
          withName(value): {
            meta+: {
              updatedBy+: {
                name: value,
              },
            },
          },
        },
    },
  '#withModel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Dashboard panels are the basic visualization building blocks.' } },
  withModel(value): {
    model: value,
  },
  '#withModelMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Dashboard panels are the basic visualization building blocks.' } },
  withModelMixin(value): {
    model+: value,
  },
  model+:
    {
      '#withCacheTimeout': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Sets panel queries cache timeout.' } },
      withCacheTimeout(value): {
        model+: {
          cacheTimeout: value,
        },
      },
      '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The datasource used in all targets.' } },
      withDatasource(value): {
        model+: {
          datasource: value,
        },
      },
      '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The datasource used in all targets.' } },
      withDatasourceMixin(value): {
        model+: {
          datasource+: value,
        },
      },
      datasource+:
        {
          '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The plugin type-id' } },
          withType(value): {
            model+: {
              datasource+: {
                type: value,
              },
            },
          },
          '#withUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specific datasource instance' } },
          withUid(value): {
            model+: {
              datasource+: {
                uid: value,
              },
            },
          },
        },
      '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Panel description.' } },
      withDescription(value): {
        model+: {
          description: value,
        },
      },
      '#withFieldConfig': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The data model used in Grafana, namely the data frame, is a columnar-oriented table structure that unifies both time series and table query results.\nEach column within this structure is called a field. A field can represent a single time series or table column.\nField options allow you to change how the data is displayed in your visualizations.' } },
      withFieldConfig(value): {
        model+: {
          fieldConfig: value,
        },
      },
      '#withFieldConfigMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The data model used in Grafana, namely the data frame, is a columnar-oriented table structure that unifies both time series and table query results.\nEach column within this structure is called a field. A field can represent a single time series or table column.\nField options allow you to change how the data is displayed in your visualizations.' } },
      withFieldConfigMixin(value): {
        model+: {
          fieldConfig+: value,
        },
      },
      fieldConfig+:
        {
          '#withDefaults': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The data model used in Grafana, namely the data frame, is a columnar-oriented table structure that unifies both time series and table query results.\nEach column within this structure is called a field. A field can represent a single time series or table column.\nField options allow you to change how the data is displayed in your visualizations.' } },
          withDefaults(value): {
            model+: {
              fieldConfig+: {
                defaults: value,
              },
            },
          },
          '#withDefaultsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'The data model used in Grafana, namely the data frame, is a columnar-oriented table structure that unifies both time series and table query results.\nEach column within this structure is called a field. A field can represent a single time series or table column.\nField options allow you to change how the data is displayed in your visualizations.' } },
          withDefaultsMixin(value): {
            model+: {
              fieldConfig+: {
                defaults+: value,
              },
            },
          },
          defaults+:
            {
              '#withActions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Define interactive HTTP requests that can be triggered from data visualizations.' } },
              withActions(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      actions:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                },
              },
              '#withActionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Define interactive HTTP requests that can be triggered from data visualizations.' } },
              withActionsMixin(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      actions+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                },
              },
              actions+:
                {
                  '#': { help: '', name: 'actions' },
                  '#withConfirmation': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withConfirmation(value): {
                    confirmation: value,
                  },
                  '#withFetch': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Fetch options' } },
                  withFetch(value): {
                    fetch: value,
                  },
                  '#withFetchMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Fetch options' } },
                  withFetchMixin(value): {
                    fetch+: value,
                  },
                  fetch+:
                    {
                      '#withBody': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withBody(value): {
                        fetch+: {
                          body: value,
                        },
                      },
                      '#withHeaders': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withHeaders(value): {
                        fetch+: {
                          headers:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                      '#withHeadersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withHeadersMixin(value): {
                        fetch+: {
                          headers+:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                      '#withMethod': { 'function': { args: [{ default: null, enums: ['GET', 'PUT', 'POST', 'DELETE', 'PATCH'], name: 'value', type: ['string'] }], help: '' } },
                      withMethod(value): {
                        fetch+: {
                          method: value,
                        },
                      },
                      '#withQueryParams': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: "These are 2D arrays of strings, each representing a key-value pair\nWe are defining this way because we can't generate a go struct that\nthat would have exactly two strings in each sub-array" } },
                      withQueryParams(value): {
                        fetch+: {
                          queryParams:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                      '#withQueryParamsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: "These are 2D arrays of strings, each representing a key-value pair\nWe are defining this way because we can't generate a go struct that\nthat would have exactly two strings in each sub-array" } },
                      withQueryParamsMixin(value): {
                        fetch+: {
                          queryParams+:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                      '#withUrl': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withUrl(value): {
                        fetch+: {
                          url: value,
                        },
                      },
                    },
                  '#withInfinity': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Infinity options' } },
                  withInfinity(value): {
                    infinity: value,
                  },
                  '#withInfinityMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Infinity options' } },
                  withInfinityMixin(value): {
                    infinity+: value,
                  },
                  infinity+:
                    {
                      '#withBody': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withBody(value): {
                        infinity+: {
                          body: value,
                        },
                      },
                      '#withDatasourceUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withDatasourceUid(value): {
                        infinity+: {
                          datasourceUid: value,
                        },
                      },
                      '#withHeaders': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withHeaders(value): {
                        infinity+: {
                          headers:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                      '#withHeadersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                      withHeadersMixin(value): {
                        infinity+: {
                          headers+:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                      '#withMethod': { 'function': { args: [{ default: null, enums: ['GET', 'PUT', 'POST', 'DELETE', 'PATCH'], name: 'value', type: ['string'] }], help: '' } },
                      withMethod(value): {
                        infinity+: {
                          method: value,
                        },
                      },
                      '#withQueryParams': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: "These are 2D arrays of strings, each representing a key-value pair\nWe are defining them this way because we can't generate a go struct that\nthat would have exactly two strings in each sub-array" } },
                      withQueryParams(value): {
                        infinity+: {
                          queryParams:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                      '#withQueryParamsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: "These are 2D arrays of strings, each representing a key-value pair\nWe are defining them this way because we can't generate a go struct that\nthat would have exactly two strings in each sub-array" } },
                      withQueryParamsMixin(value): {
                        infinity+: {
                          queryParams+:
                            (if std.isArray(value)
                             then value
                             else [value]),
                        },
                      },
                      '#withUrl': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withUrl(value): {
                        infinity+: {
                          url: value,
                        },
                      },
                    },
                  '#withOneClick': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
                  withOneClick(value=true): {
                    oneClick: value,
                  },
                  '#withStyle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withStyle(value): {
                    style: value,
                  },
                  '#withStyleMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withStyleMixin(value): {
                    style+: value,
                  },
                  style+:
                    {
                      '#withBackgroundColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withBackgroundColor(value): {
                        style+: {
                          backgroundColor: value,
                        },
                      },
                    },
                  '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withTitle(value): {
                    title: value,
                  },
                  '#withType': { 'function': { args: [{ default: null, enums: ['fetch', 'infinity'], name: 'value', type: ['string'] }], help: 'Dashboard action type' } },
                  withType(value): {
                    type: value,
                  },
                  '#withVariables': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withVariables(value): {
                    variables:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                  '#withVariablesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withVariablesMixin(value): {
                    variables+:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                  variables+:
                    {
                      '#': { help: '', name: 'variables' },
                      '#withKey': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withKey(value): {
                        key: value,
                      },
                      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withName(value): {
                        name: value,
                      },
                      '#withType': { 'function': { args: [{ default: 'string', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='string'): {
                        type: value,
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'string', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        type+: value,
                      },
                      type+:
                        {
                          '#withActionVariableType': { 'function': { args: [{ default: null, enums: ['string'], name: 'value', type: ['string'] }], help: 'Action variable type' } },
                          withActionVariableType(value): {
                            type+: {
                              ActionVariableType: value,
                            },
                          },
                        },
                    },
                },
              '#withColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Map a field to a color.' } },
              withColor(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      color: value,
                    },
                  },
                },
              },
              '#withColorMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Map a field to a color.' } },
              withColorMixin(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      color+: value,
                    },
                  },
                },
              },
              color+:
                {
                  '#withFixedColor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The fixed color value for fixed or shades color modes.' } },
                  withFixedColor(value): {
                    model+: {
                      fieldConfig+: {
                        defaults+: {
                          color+: {
                            fixedColor: value,
                          },
                        },
                      },
                    },
                  },
                  '#withMode': { 'function': { args: [{ default: null, enums: ['thresholds', 'palette-classic', 'palette-classic-by-name', 'continuous-GrYlRd', 'continuous-RdYlGr', 'continuous-BlYlRd', 'continuous-YlRd', 'continuous-BlPu', 'continuous-YlBl', 'continuous-blues', 'continuous-reds', 'continuous-greens', 'continuous-purples', 'fixed', 'shades'], name: 'value', type: ['string'] }], help: 'Color mode for a field. You can specify a single color, or select a continuous (gradient) color schemes, based on a value.\nContinuous color interpolates a color using the percentage of a value relative to min and max.\nAccepted values are:\n`thresholds`: From thresholds. Informs Grafana to take the color from the matching threshold\n`palette-classic`: Classic palette. Grafana will assign color by looking up a color in a palette by series index. Useful for Graphs and pie charts and other categorical data visualizations\n`palette-classic-by-name`: Classic palette (by name). Grafana will assign color by looking up a color in a palette by series name. Useful for Graphs and pie charts and other categorical data visualizations\n`continuous-GrYlRd`: ontinuous Green-Yellow-Red palette mode\n`continuous-RdYlGr`: Continuous Red-Yellow-Green palette mode\n`continuous-BlYlRd`: Continuous Blue-Yellow-Red palette mode\n`continuous-YlRd`: Continuous Yellow-Red palette mode\n`continuous-BlPu`: Continuous Blue-Purple palette mode\n`continuous-YlBl`: Continuous Yellow-Blue palette mode\n`continuous-blues`: Continuous Blue palette mode\n`continuous-reds`: Continuous Red palette mode\n`continuous-greens`: Continuous Green palette mode\n`continuous-purples`: Continuous Purple palette mode\n`shades`: Shades of a single color. Specify a single color, useful in an override rule.\n`fixed`: Fixed color mode. Specify a single color, useful in an override rule.' } },
                  withMode(value): {
                    model+: {
                      fieldConfig+: {
                        defaults+: {
                          color+: {
                            mode: value,
                          },
                        },
                      },
                    },
                  },
                  '#withSeriesBy': { 'function': { args: [{ default: null, enums: ['min', 'max', 'last'], name: 'value', type: ['string'] }], help: 'Defines how to assign a series color from "by value" color schemes. For example for an aggregated data points like a timeseries, the color can be assigned by the min, max or last value.' } },
                  withSeriesBy(value): {
                    model+: {
                      fieldConfig+: {
                        defaults+: {
                          color+: {
                            seriesBy: value,
                          },
                        },
                      },
                    },
                  },
                },
              '#withCustom': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'custom is specified by the FieldConfig field\nin panel plugin schemas.' } },
              withCustom(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      custom: value,
                    },
                  },
                },
              },
              '#withCustomMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'custom is specified by the FieldConfig field\nin panel plugin schemas.' } },
              withCustomMixin(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      custom+: value,
                    },
                  },
                },
              },
              '#withDecimals': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Specify the number of decimals Grafana includes in the rendered value.\nIf you leave this field blank, Grafana automatically truncates the number of decimals based on the value.\nFor example 1.1234 will display as 1.12 and 100.456 will display as 100.\nTo display all decimals, set the unit to `String`.' } },
              withDecimals(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      decimals: value,
                    },
                  },
                },
              },
              '#withDescription': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Human readable field metadata' } },
              withDescription(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      description: value,
                    },
                  },
                },
              },
              '#withDisplayName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The display value for this field.  This supports template variables blank is auto' } },
              withDisplayName(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      displayName: value,
                    },
                  },
                },
              },
              '#withDisplayNameFromDS': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'This can be used by data sources that return and explicit naming structure for values and labels\nWhen this property is configured, this value is used rather than the default naming strategy.' } },
              withDisplayNameFromDS(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      displayNameFromDS: value,
                    },
                  },
                },
              },
              '#withFieldMinMax': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
              withFieldMinMax(value=true): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      fieldMinMax: value,
                    },
                  },
                },
              },
              '#withFilterable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'True if data source field supports ad-hoc filters' } },
              withFilterable(value=true): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      filterable: value,
                    },
                  },
                },
              },
              '#withLinks': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'The behavior when clicking on a result' } },
              withLinks(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      links:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                },
              },
              '#withLinksMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'The behavior when clicking on a result' } },
              withLinksMixin(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      links+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
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
              '#withMappings': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Convert input values into a display string' } },
              withMappings(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      mappings:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                },
              },
              '#withMappingsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Convert input values into a display string' } },
              withMappingsMixin(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      mappings+:
                        (if std.isArray(value)
                         then value
                         else [value]),
                    },
                  },
                },
              },
              mappings+:
                {
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
              '#withMax': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'The maximum value used in percentage threshold calculations. Leave blank for auto calculation based on all series and fields.' } },
              withMax(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      max: value,
                    },
                  },
                },
              },
              '#withMin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'The minimum value used in percentage threshold calculations. Leave blank for auto calculation based on all series and fields.' } },
              withMin(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      min: value,
                    },
                  },
                },
              },
              '#withNoValue': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Alternative to empty string' } },
              withNoValue(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      noValue: value,
                    },
                  },
                },
              },
              '#withPath': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'An explicit path to the field in the datasource.  When the frame meta includes a path,\nThis will default to `${frame.meta.path}/${field.name}\n\nWhen defined, this value can be used as an identifier within the datasource scope, and\nmay be used to update the results' } },
              withPath(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      path: value,
                    },
                  },
                },
              },
              '#withThresholds': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Thresholds configuration for the panel' } },
              withThresholds(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      thresholds: value,
                    },
                  },
                },
              },
              '#withThresholdsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'Thresholds configuration for the panel' } },
              withThresholdsMixin(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      thresholds+: value,
                    },
                  },
                },
              },
              thresholds+:
                {
                  '#withMode': { 'function': { args: [{ default: null, enums: ['absolute', 'percentage'], name: 'value', type: ['string'] }], help: 'Thresholds can either be `absolute` (specific number) or `percentage` (relative to min or max, it will be values between 0 and 1).' } },
                  withMode(value): {
                    model+: {
                      fieldConfig+: {
                        defaults+: {
                          thresholds+: {
                            mode: value,
                          },
                        },
                      },
                    },
                  },
                  '#withSteps': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: "Must be sorted by 'value', first value is always -Infinity" } },
                  withSteps(value): {
                    model+: {
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
                  },
                  '#withStepsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: "Must be sorted by 'value', first value is always -Infinity" } },
                  withStepsMixin(value): {
                    model+: {
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
                  steps+:
                    {
                      '#': { help: '', name: 'steps' },
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
              '#withUnit': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Unit a field should use. The unit you select is applied to all fields except time.\nYou can use the units ID availables in Grafana or a custom unit.\nAvailable units in Grafana: https://github.com/grafana/grafana/blob/main/packages/grafana-data/src/valueFormats/categories.ts\nAs custom unit, you can use the following formats:\n`suffix:<suffix>` for custom unit that should go after value.\n`prefix:<prefix>` for custom unit that should go before value.\n`time:<format>` For custom date time formats type for example `time:YYYY-MM-DD`.\n`si:<base scale><unit characters>` for custom SI units. For example: `si: mF`. This one is a bit more advanced as you can specify both a unit and the source data scale. So if your source data is represented as milli (thousands of) something prefix the unit with that SI scale character.\n`count:<unit>` for a custom count unit.\n`currency:<unit>` for custom a currency unit.' } },
              withUnit(value): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      unit: value,
                    },
                  },
                },
              },
              '#withWriteable': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'True if data source can write a value to the path. Auth/authz are supported separately' } },
              withWriteable(value=true): {
                model+: {
                  fieldConfig+: {
                    defaults+: {
                      writeable: value,
                    },
                  },
                },
              },
            },
          '#withOverrides': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Overrides are the options applied to specific fields overriding the defaults.' } },
          withOverrides(value): {
            model+: {
              fieldConfig+: {
                overrides:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          '#withOverridesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Overrides are the options applied to specific fields overriding the defaults.' } },
          withOverridesMixin(value): {
            model+: {
              fieldConfig+: {
                overrides+:
                  (if std.isArray(value)
                   then value
                   else [value]),
              },
            },
          },
          overrides+:
            {
              '#': { help: '', name: 'overrides' },
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
      '#withHideTimeOverride': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Controls if the timeFrom or timeShift overrides are shown in the panel header' } },
      withHideTimeOverride(value=true): {
        model+: {
          hideTimeOverride: value,
        },
      },
      '#withInterval': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The min time interval setting defines a lower limit for the $__interval and $__interval_ms variables.\nThis value must be formatted as a number followed by a valid time\nidentifier like: "40s", "3d", etc.\nSee: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options' } },
      withInterval(value): {
        model+: {
          interval: value,
        },
      },
      '#withLinks': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Panel links.' } },
      withLinks(value): {
        model+: {
          links:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withLinksMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Panel links.' } },
      withLinksMixin(value): {
        model+: {
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
      '#withMaxDataPoints': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'The maximum number of data points that the panel queries are retrieving.' } },
      withMaxDataPoints(value): {
        model+: {
          maxDataPoints: value,
        },
      },
      '#withMaxPerRow': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Option for repeated panels that controls max items per row\nOnly relevant for horizontally repeated panels' } },
      withMaxPerRow(value): {
        model+: {
          maxPerRow: value,
        },
      },
      '#withOptions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'It depends on the panel plugin. They are specified by the Options field in panel plugin schemas.' } },
      withOptions(value): {
        model+: {
          options: value,
        },
      },
      '#withOptionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'It depends on the panel plugin. They are specified by the Options field in panel plugin schemas.' } },
      withOptionsMixin(value): {
        model+: {
          options+: value,
        },
      },
      '#withPluginVersion': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The version of the plugin that is used for this panel. This is used to find the plugin to display the panel and to migrate old panel configs.' } },
      withPluginVersion(value): {
        model+: {
          pluginVersion: value,
        },
      },
      '#withQueryCachingTTL': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['number'] }], help: 'Overrides the data source configured time-to-live for a query cache item in milliseconds' } },
      withQueryCachingTTL(value): {
        model+: {
          queryCachingTTL: value,
        },
      },
      '#withRepeat': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Name of template variable to repeat for.' } },
      withRepeat(value): {
        model+: {
          repeat: value,
        },
      },
      '#withRepeatDirection': { 'function': { args: [{ default: 'h', enums: ['h', 'v'], name: 'value', type: ['string'] }], help: "Direction to repeat in if 'repeat' is set.\n`h` for horizontal, `v` for vertical." } },
      withRepeatDirection(value='h'): {
        model+: {
          repeatDirection: value,
        },
      },
      '#withTargets': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Depends on the panel plugin. See the plugin documentation for details.' } },
      withTargets(value): {
        model+: {
          targets:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withTargetsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Depends on the panel plugin. See the plugin documentation for details.' } },
      withTargetsMixin(value): {
        model+: {
          targets+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withTimeCompare': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Compare the current time range with a previous period\nFor example "1d" to compare current period but shifted back 1 day' } },
      withTimeCompare(value): {
        model+: {
          timeCompare: value,
        },
      },
      '#withTimeFrom': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Overrides the relative time range for individual panels,\nwhich causes them to be different than what is selected in\nthe dashboard time picker in the top-right corner of the dashboard. You can use this to show metrics from different\ntime periods or days on the same dashboard.\nThe value is formatted as time operation like: `now-5m` (Last 5 minutes), `now/d` (the day so far),\n`now-5d/d`(Last 5 days), `now/w` (This week so far), `now-2y/y` (Last 2 years).\nNote: Panel time overrides have no effect when the dashboard’s time range is absolute.\nSee: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options' } },
      withTimeFrom(value): {
        model+: {
          timeFrom: value,
        },
      },
      '#withTimeShift': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Overrides the time range for individual panels by shifting its start and end relative to the time picker.\nFor example, you can shift the time range for the panel to be two hours earlier than the dashboard time picker setting `2h`.\nNote: Panel time overrides have no effect when the dashboard’s time range is absolute.\nSee: https://grafana.com/docs/grafana/latest/panels-visualizations/query-transform-data/#query-options' } },
      withTimeShift(value): {
        model+: {
          timeShift: value,
        },
      },
      '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Panel title.' } },
      withTitle(value): {
        model+: {
          title: value,
        },
      },
      '#withTransformations': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'List of transformations that are applied to the panel data before rendering.\nWhen there are multiple transformations, Grafana applies them in the order they are listed.\nEach transformation creates a result set that then passes on to the next transformation in the processing pipeline.' } },
      withTransformations(value): {
        model+: {
          transformations:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      '#withTransformationsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'List of transformations that are applied to the panel data before rendering.\nWhen there are multiple transformations, Grafana applies them in the order they are listed.\nEach transformation creates a result set that then passes on to the next transformation in the processing pipeline.' } },
      withTransformationsMixin(value): {
        model+: {
          transformations+:
            (if std.isArray(value)
             then value
             else [value]),
        },
      },
      transformations+:
        {
          '#': { help: '', name: 'transformations' },
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
      '#withTransparent': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Whether to display the panel without a background.' } },
      withTransparent(value=true): {
        model+: {
          transparent: value,
        },
      },
      '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The panel plugin type id. This is used to find the plugin to display the panel.' } },
      withType(value): {
        model+: {
          type: value,
        },
      },
    },
  '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Panel name (also saved in the model)' } },
  withName(value): {
    name: value,
  },
  '#withSchemaVersion': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Dashboard version when this was saved (zero if unknown)' } },
  withSchemaVersion(value): {
    schemaVersion: value,
  },
  '#withType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The panel type (from inside the model)' } },
  withType(value): {
    type: value,
  },
  '#withUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Library element UID' } },
  withUid(value): {
    uid: value,
  },
  '#withVersion': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'panel version, incremented each time the dashboard is updated.' } },
  withVersion(value): {
    version: value,
  },
}
