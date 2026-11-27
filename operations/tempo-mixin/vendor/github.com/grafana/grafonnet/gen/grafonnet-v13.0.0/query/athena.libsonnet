// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.query.athena', name: 'athena' },
  '#withColumn': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withColumn(value): {
    column: value,
  },
  '#withConnectionArgs': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withConnectionArgs(value): {
    connectionArgs: value,
  },
  '#withConnectionArgsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
  withConnectionArgsMixin(value): {
    connectionArgs+: value,
  },
  connectionArgs+:
    {
      '#withCatalog': { 'function': { args: [{ default: '__default', enums: null, name: 'value', type: ['string'] }], help: '' } },
      withCatalog(value='__default'): {
        connectionArgs+: {
          catalog: value,
        },
      },
      '#withDatabase': { 'function': { args: [{ default: '__default', enums: null, name: 'value', type: ['string'] }], help: '' } },
      withDatabase(value='__default'): {
        connectionArgs+: {
          database: value,
        },
      },
      '#withRegion': { 'function': { args: [{ default: '__default', enums: null, name: 'value', type: ['string'] }], help: '' } },
      withRegion(value='__default'): {
        connectionArgs+: {
          region: value,
        },
      },
      '#withResultReuseEnabled': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withResultReuseEnabled(value=true): {
        connectionArgs+: {
          resultReuseEnabled: value,
        },
      },
      '#withResultReuseMaxAgeInMinutes': { 'function': { args: [{ default: 60, enums: null, name: 'value', type: ['number'] }], help: '' } },
      withResultReuseMaxAgeInMinutes(value=60): {
        connectionArgs+: {
          resultReuseMaxAgeInMinutes: value,
        },
      },
    },
  '#withDatasource': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: "For mixed data sources the selected datasource is on the query level.\nFor non mixed scenarios this is undefined.\nTODO find a better way to do this ^ that's friendly to schema\nTODO this shouldn't be unknown but DataSourceRef | null" } },
  withDatasource(value): {
    datasource: value,
  },
  '#withDatasourceMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: "For mixed data sources the selected datasource is on the query level.\nFor non mixed scenarios this is undefined.\nTODO find a better way to do this ^ that's friendly to schema\nTODO this shouldn't be unknown but DataSourceRef | null" } },
  withDatasourceMixin(value): {
    datasource+: value,
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
  '#withFormat': { 'function': { args: [{ default: null, enums: [0, 1, 2], name: 'value', type: ['integer'] }], help: '' } },
  withFormat(value): {
    format: value,
  },
  '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.' } },
  withHide(value=true): {
    hide: value,
  },
  '#withQueryID': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withQueryID(value): {
    queryID: value,
  },
  '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specify the query flavor\nTODO make this required and give it a default' } },
  withQueryType(value): {
    queryType: value,
  },
  '#withRawSQL': { 'function': { args: [{ default: '', enums: null, name: 'value', type: ['string'] }], help: '' } },
  withRawSQL(value=''): {
    rawSQL: value,
  },
  '#withRefId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'A unique identifier for the query within the list of targets.\nIn server side expressions, the refId is used as a variable name to identify results.\nBy default, the UI will assign A->Z; however setting meaningful names may be useful.' } },
  withRefId(value): {
    refId: value,
  },
  '#withTable': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withTable(value): {
    table: value,
  },
}
