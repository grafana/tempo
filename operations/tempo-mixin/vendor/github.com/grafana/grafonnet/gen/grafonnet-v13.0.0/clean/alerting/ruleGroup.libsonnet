// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.alerting.ruleGroup', name: 'ruleGroup' },
  '#withFolderUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withFolderUid(value): {
    folderUid: value,
  },
  '#withInterval': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Duration in seconds.' } },
  withInterval(value): {
    interval: value,
  },
  '#withRules': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
  withRules(value): {
    rules:
      (if std.isArray(value)
       then value
       else [value]),
  },
  '#withRulesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
  withRulesMixin(value): {
    rules+:
      (if std.isArray(value)
       then value
       else [value]),
  },
  '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
  withTitle(value): {
    title: value,
  },
  rule+:
    {
      '#withAnnotations': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withAnnotations(value): {
        annotations: value,
      },
      '#withAnnotationsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withAnnotationsMixin(value): {
        annotations+: value,
      },
      '#withCondition': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withCondition(value): {
        condition: value,
      },
      '#withData': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withData(value): {
        data:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withDataMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
      withDataMixin(value): {
        data+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withExecErrState': { 'function': { args: [{ default: null, enums: ['Alerting', 'Error', 'OK', 'KeepLast'], name: 'value', type: ['string'] }], help: '' } },
      withExecErrState(value): {
        execErrState: value,
      },
      '#withFolderUID': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withFolderUID(value): {
        folderUID: value,
      },
      '#withFor': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The amount of time, in seconds, for which the rule must be breached for the rule to be considered to be Firing.\nBefore this time has elapsed, the rule is only considered to be Pending.' } },
      withFor(value): {
        'for': value,
      },
      '#withIsPaused': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: '' } },
      withIsPaused(value=true): {
        isPaused: value,
      },
      '#withLabels': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withLabels(value): {
        labels: value,
      },
      '#withLabelsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
      withLabelsMixin(value): {
        labels+: value,
      },
      '#withNoDataState': { 'function': { args: [{ default: null, enums: ['OK', 'Alerting', 'NoData', 'KeepLast'], name: 'value', type: ['string'] }], help: '' } },
      withNoDataState(value): {
        noDataState: value,
      },
      '#withOrgID': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: '' } },
      withOrgID(value): {
        orgID: value,
      },
      '#withRuleGroup': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withRuleGroup(value): {
        ruleGroup: value,
      },
      '#withTitle': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withTitle(value): {
        title: value,
      },
      data+:
        {
          '#': { help: '', name: 'data' },
          '#withDatasourceUid': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: "Grafana data source unique identifier; it should be '__expr__' for a Server Side Expression operation." } },
          withDatasourceUid(value): {
            datasourceUid: value,
          },
          '#withModel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'JSON is the raw JSON query and includes the above properties as well as custom properties.' } },
          withModel(value): {
            model: value,
          },
          '#withModelMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'JSON is the raw JSON query and includes the above properties as well as custom properties.' } },
          withModelMixin(value): {
            model+: value,
          },
          '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'QueryType is an optional identifier for the type of query.\nIt can be used to distinguish different types of queries.' } },
          withQueryType(value): {
            queryType: value,
          },
          '#withRefId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'RefID is the unique identifier of the query, set by the frontend call.' } },
          withRefId(value): {
            refId: value,
          },
          '#withRelativeTimeRange': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'RelativeTimeRange is the per query start and end time\nfor requests.' } },
          withRelativeTimeRange(value): {
            relativeTimeRange: value,
          },
          '#withRelativeTimeRangeMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'RelativeTimeRange is the per query start and end time\nfor requests.' } },
          withRelativeTimeRangeMixin(value): {
            relativeTimeRange+: value,
          },
          relativeTimeRange+:
            {
              '#withFrom': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Duration in seconds.' } },
              withFrom(value): {
                relativeTimeRange+: {
                  from: value,
                },
              },
              '#withTo': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'Duration in seconds.' } },
              withTo(value): {
                relativeTimeRange+: {
                  to: value,
                },
              },
            },
        },
    },
}
+ (import '../../custom/alerting/ruleGroup.libsonnet')
