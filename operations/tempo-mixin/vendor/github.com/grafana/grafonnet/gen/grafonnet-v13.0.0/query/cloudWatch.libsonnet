// This file is generated, do not manually edit.
{
  '#': { help: 'grafonnet.query.cloudWatch', name: 'cloudWatch' },
  AnnotationQuery+:
    {
      '#withAccountId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The ID of the AWS account to query for the metric, specifying `all` will query all accounts that the monitoring account is permitted to query.' } },
      withAccountId(value): {
        accountId: value,
      },
      '#withActionPrefix': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Use this parameter to filter the results of the operation to only those alarms\nthat use a certain alarm action. For example, you could specify the ARN of\nan SNS topic to find all alarms that send notifications to that topic.\ne.g. `arn:aws:sns:us-east-1:123456789012:my-app-` would match `arn:aws:sns:us-east-1:123456789012:my-app-action`\nbut not match `arn:aws:sns:us-east-1:123456789012:your-app-action`' } },
      withActionPrefix(value): {
        actionPrefix: value,
      },
      '#withAlarmNamePrefix': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'An alarm name prefix. If you specify this parameter, you receive information\nabout all alarms that have names that start with this prefix.\ne.g. `my-team-service-` would match `my-team-service-high-cpu` but not match `your-team-service-high-cpu`' } },
      withAlarmNamePrefix(value): {
        alarmNamePrefix: value,
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
      '#withDimensions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'A name/value pair that is part of the identity of a metric. For example, you can get statistics for a specific EC2 instance by specifying the InstanceId dimension when you search for metrics.' } },
      withDimensions(value): {
        dimensions: value,
      },
      '#withDimensionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'A name/value pair that is part of the identity of a metric. For example, you can get statistics for a specific EC2 instance by specifying the InstanceId dimension when you search for metrics.' } },
      withDimensionsMixin(value): {
        dimensions+: value,
      },
      '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.' } },
      withHide(value=true): {
        hide: value,
      },
      '#withMatchExact': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Only show metrics that exactly match all defined dimension names.' } },
      withMatchExact(value=true): {
        matchExact: value,
      },
      '#withMetricName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Name of the metric' } },
      withMetricName(value): {
        metricName: value,
      },
      '#withNamespace': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'A namespace is a container for CloudWatch metrics. Metrics in different namespaces are isolated from each other, so that metrics from different applications are not mistakenly aggregated into the same statistics. For example, Amazon EC2 uses the AWS/EC2 namespace.' } },
      withNamespace(value): {
        namespace: value,
      },
      '#withPeriod': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: "The length of time associated with a specific Amazon CloudWatch statistic. Can be specified by a number of seconds, 'auto', or as a duration string e.g. '15m' being 15 minutes" } },
      withPeriod(value): {
        period: value,
      },
      '#withPrefixMatching': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Enable matching on the prefix of the action name or alarm name, specify the prefixes with actionPrefix and/or alarmNamePrefix' } },
      withPrefixMatching(value=true): {
        prefixMatching: value,
      },
      '#withQueryMode': { 'function': { args: [{ default: 'Annotations', enums: ['Metrics', 'Logs', 'Annotations'], name: 'value', type: ['string'] }], help: 'Whether a query is a Metrics, Logs, or Annotations query' } },
      withQueryMode(value='Annotations'): {
        queryMode: value,
      },
      '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specify the query flavor\nTODO make this required and give it a default' } },
      withQueryType(value): {
        queryType: value,
      },
      '#withRefId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'A unique identifier for the query within the list of targets.\nIn server side expressions, the refId is used as a variable name to identify results.\nBy default, the UI will assign A->Z; however setting meaningful names may be useful.' } },
      withRefId(value): {
        refId: value,
      },
      '#withRegion': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'AWS region to query for the metric' } },
      withRegion(value): {
        region: value,
      },
      '#withStatistic': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Metric data aggregations over specified periods of time. For detailed definitions of the statistics supported by CloudWatch, see https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/Statistics-definitions.html.' } },
      withStatistic(value): {
        statistic: value,
      },
      '#withStatistics': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '@deprecated use statistic' } },
      withStatistics(value): {
        statistics:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withStatisticsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '@deprecated use statistic' } },
      withStatisticsMixin(value): {
        statistics+:
          (if std.isArray(value)
           then value
           else [value]),
      },
    },
  LogsQuery+:
    {
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
      '#withExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The CloudWatch Logs Insights query to execute' } },
      withExpression(value): {
        expression: value,
      },
      '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.' } },
      withHide(value=true): {
        hide: value,
      },
      '#withId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
      withId(value): {
        id: value,
      },
      '#withLogGroupNames': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '@deprecated use logGroups' } },
      withLogGroupNames(value): {
        logGroupNames:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withLogGroupNamesMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '@deprecated use logGroups' } },
      withLogGroupNamesMixin(value): {
        logGroupNames+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withLogGroups': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Log groups to query' } },
      withLogGroups(value): {
        logGroups:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withLogGroupsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Log groups to query' } },
      withLogGroupsMixin(value): {
        logGroups+:
          (if std.isArray(value)
           then value
           else [value]),
      },
      logGroups+:
        {
          '#': { help: '', name: 'logGroups' },
          '#withAccountId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'AccountId of the log group' } },
          withAccountId(value): {
            accountId: value,
          },
          '#withAccountLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Label of the log group' } },
          withAccountLabel(value): {
            accountLabel: value,
          },
          '#withArn': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'ARN of the log group' } },
          withArn(value): {
            arn: value,
          },
          '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Name of the log group' } },
          withName(value): {
            name: value,
          },
        },
      '#withQueryLanguage': { 'function': { args: [{ default: null, enums: ['CWLI', 'SQL', 'PPL'], name: 'value', type: ['string'] }], help: 'Language used for querying logs, can be CWLI, SQL, or PPL. If empty, the default language is CWLI.' } },
      withQueryLanguage(value): {
        queryLanguage: value,
      },
      '#withQueryMode': { 'function': { args: [{ default: 'Logs', enums: ['Metrics', 'Logs', 'Annotations'], name: 'value', type: ['string'] }], help: 'Whether a query is a Metrics, Logs, or Annotations query' } },
      withQueryMode(value='Logs'): {
        queryMode: value,
      },
      '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specify the query flavor\nTODO make this required and give it a default' } },
      withQueryType(value): {
        queryType: value,
      },
      '#withRefId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'A unique identifier for the query within the list of targets.\nIn server side expressions, the refId is used as a variable name to identify results.\nBy default, the UI will assign A->Z; however setting meaningful names may be useful.' } },
      withRefId(value): {
        refId: value,
      },
      '#withRegion': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'AWS region to query for the logs' } },
      withRegion(value): {
        region: value,
      },
      '#withStatsGroups': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Fields to group the results by, this field is automatically populated whenever the query is updated' } },
      withStatsGroups(value): {
        statsGroups:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withStatsGroupsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: 'Fields to group the results by, this field is automatically populated whenever the query is updated' } },
      withStatsGroupsMixin(value): {
        statsGroups+:
          (if std.isArray(value)
           then value
           else [value]),
      },
    },
  MetricsQuery+:
    {
      '#withAccountId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The ID of the AWS account to query for the metric, specifying `all` will query all accounts that the monitoring account is permitted to query.' } },
      withAccountId(value): {
        accountId: value,
      },
      '#withAlias': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Deprecated: use label\n@deprecated use label' } },
      withAlias(value): {
        alias: value,
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
      '#withDimensions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'A name/value pair that is part of the identity of a metric. For example, you can get statistics for a specific EC2 instance by specifying the InstanceId dimension when you search for metrics.' } },
      withDimensions(value): {
        dimensions: value,
      },
      '#withDimensionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'A name/value pair that is part of the identity of a metric. For example, you can get statistics for a specific EC2 instance by specifying the InstanceId dimension when you search for metrics.' } },
      withDimensionsMixin(value): {
        dimensions+: value,
      },
      '#withExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Math expression query' } },
      withExpression(value): {
        expression: value,
      },
      '#withHide': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.' } },
      withHide(value=true): {
        hide: value,
      },
      '#withId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'ID can be used to reference other queries in math expressions. The ID can include numbers, letters, and underscore, and must start with a lowercase letter.' } },
      withId(value): {
        id: value,
      },
      '#withLabel': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Change the time series legend names using dynamic labels. See https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/graph-dynamic-labels.html for more details.' } },
      withLabel(value): {
        label: value,
      },
      '#withMatchExact': { 'function': { args: [{ default: true, enums: null, name: 'value', type: ['boolean'] }], help: 'Only show metrics that exactly match all defined dimension names.' } },
      withMatchExact(value=true): {
        matchExact: value,
      },
      '#withMetricEditorMode': { 'function': { args: [{ default: null, enums: [0, 1], name: 'value', type: ['integer'] }], help: 'Whether to use the query builder or code editor to create the query' } },
      withMetricEditorMode(value): {
        metricEditorMode: value,
      },
      '#withMetricName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Name of the metric' } },
      withMetricName(value): {
        metricName: value,
      },
      '#withMetricQueryType': { 'function': { args: [{ default: null, enums: [0, 1], name: 'value', type: ['integer'] }], help: 'Whether to use a metric search or metric insights query' } },
      withMetricQueryType(value): {
        metricQueryType: value,
      },
      '#withNamespace': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'A namespace is a container for CloudWatch metrics. Metrics in different namespaces are isolated from each other, so that metrics from different applications are not mistakenly aggregated into the same statistics. For example, Amazon EC2 uses the AWS/EC2 namespace.' } },
      withNamespace(value): {
        namespace: value,
      },
      '#withPeriod': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: "The length of time associated with a specific Amazon CloudWatch statistic. Can be specified by a number of seconds, 'auto', or as a duration string e.g. '15m' being 15 minutes" } },
      withPeriod(value): {
        period: value,
      },
      '#withQueryMode': { 'function': { args: [{ default: 'Metrics', enums: ['Metrics', 'Logs', 'Annotations'], name: 'value', type: ['string'] }], help: 'Whether a query is a Metrics, Logs, or Annotations query' } },
      withQueryMode(value='Metrics'): {
        queryMode: value,
      },
      '#withQueryType': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Specify the query flavor\nTODO make this required and give it a default' } },
      withQueryType(value): {
        queryType: value,
      },
      '#withRefId': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'A unique identifier for the query within the list of targets.\nIn server side expressions, the refId is used as a variable name to identify results.\nBy default, the UI will assign A->Z; however setting meaningful names may be useful.' } },
      withRefId(value): {
        refId: value,
      },
      '#withRegion': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'AWS region to query for the metric' } },
      withRegion(value): {
        region: value,
      },
      '#withSql': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'When the metric query type is set to `Insights` and the `metricEditorMode` is set to `Builder`, this field is used to build up an object representation of a SQL query.' } },
      withSql(value): {
        sql: value,
      },
      '#withSqlMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'When the metric query type is set to `Insights` and the `metricEditorMode` is set to `Builder`, this field is used to build up an object representation of a SQL query.' } },
      withSqlMixin(value): {
        sql+: value,
      },
      sql+:
        {
          '#withFrom': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object'] }], help: 'FROM part of the SQL expression' } },
          withFrom(value): {
            sql+: {
              from: value,
            },
          },
          '#withFromMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object', 'object'] }], help: 'FROM part of the SQL expression' } },
          withFromMixin(value): {
            sql+: {
              from+: value,
            },
          },
          from+:
            {
              '#withQueryEditorPropertyExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withQueryEditorPropertyExpression(value): {
                sql+: {
                  from+: {
                    QueryEditorPropertyExpression: value,
                  },
                },
              },
              '#withQueryEditorPropertyExpressionMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withQueryEditorPropertyExpressionMixin(value): {
                sql+: {
                  from+: {
                    QueryEditorPropertyExpression+: value,
                  },
                },
              },
              QueryEditorPropertyExpression+:
                {
                  '#withProperty': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withProperty(value): {
                    sql+: {
                      from+: {
                        property: value,
                      },
                    },
                  },
                  '#withPropertyMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
                  withPropertyMixin(value): {
                    sql+: {
                      from+: {
                        property+: value,
                      },
                    },
                  },
                  property+:
                    {
                      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withName(value): {
                        sql+: {
                          from+: {
                            property+: {
                              name: value,
                            },
                          },
                        },
                      },
                      '#withType': { 'function': { args: [{ default: 'string', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='string'): {
                        sql+: {
                          from+: {
                            property+: {
                              type: value,
                            },
                          },
                        },
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'string', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        sql+: {
                          from+: {
                            property+: {
                              type+: value,
                            },
                          },
                        },
                      },
                      type+:
                        {
                          '#withQueryEditorPropertyType': { 'function': { args: [{ default: null, enums: ['string'], name: 'value', type: ['string'] }], help: '' } },
                          withQueryEditorPropertyType(value): {
                            sql+: {
                              from+: {
                                property+: {
                                  type+: {
                                    QueryEditorPropertyType: value,
                                  },
                                },
                              },
                            },
                          },
                        },
                    },
                  '#withType': { 'function': { args: [{ default: 'property', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withType(value='property'): {
                    sql+: {
                      from+: {
                        type: value,
                      },
                    },
                  },
                  '#withTypeMixin': { 'function': { args: [{ default: 'property', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withTypeMixin(value): {
                    sql+: {
                      from+: {
                        type+: value,
                      },
                    },
                  },
                  type+:
                    {
                      '#withQueryEditorExpressionType': { 'function': { args: [{ default: null, enums: ['property', 'operator', 'or', 'and', 'groupBy', 'function', 'functionParameter'], name: 'value', type: ['string'] }], help: '' } },
                      withQueryEditorExpressionType(value): {
                        sql+: {
                          from+: {
                            type+: {
                              QueryEditorExpressionType: value,
                            },
                          },
                        },
                      },
                    },
                },
              '#withQueryEditorFunctionExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withQueryEditorFunctionExpression(value): {
                sql+: {
                  from+: {
                    QueryEditorFunctionExpression: value,
                  },
                },
              },
              '#withQueryEditorFunctionExpressionMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: '' } },
              withQueryEditorFunctionExpressionMixin(value): {
                sql+: {
                  from+: {
                    QueryEditorFunctionExpression+: value,
                  },
                },
              },
              QueryEditorFunctionExpression+:
                {
                  '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value): {
                    sql+: {
                      from+: {
                        name: value,
                      },
                    },
                  },
                  '#withParameters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withParameters(value): {
                    sql+: {
                      from+: {
                        parameters:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                    },
                  },
                  '#withParametersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
                  withParametersMixin(value): {
                    sql+: {
                      from+: {
                        parameters+:
                          (if std.isArray(value)
                           then value
                           else [value]),
                      },
                    },
                  },
                  parameters+:
                    {
                      '#': { help: '', name: 'parameters' },
                      '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withName(value): {
                        name: value,
                      },
                      '#withType': { 'function': { args: [{ default: 'functionParameter', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withType(value='functionParameter'): {
                        type: value,
                      },
                      '#withTypeMixin': { 'function': { args: [{ default: 'functionParameter', enums: null, name: 'value', type: ['string'] }], help: '' } },
                      withTypeMixin(value): {
                        type+: value,
                      },
                      type+:
                        {
                          '#withQueryEditorExpressionType': { 'function': { args: [{ default: null, enums: ['property', 'operator', 'or', 'and', 'groupBy', 'function', 'functionParameter'], name: 'value', type: ['string'] }], help: '' } },
                          withQueryEditorExpressionType(value): {
                            type+: {
                              QueryEditorExpressionType: value,
                            },
                          },
                        },
                    },
                  '#withType': { 'function': { args: [{ default: 'function', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withType(value='function'): {
                    sql+: {
                      from+: {
                        type: value,
                      },
                    },
                  },
                  '#withTypeMixin': { 'function': { args: [{ default: 'function', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withTypeMixin(value): {
                    sql+: {
                      from+: {
                        type+: value,
                      },
                    },
                  },
                  type+:
                    {
                      '#withQueryEditorExpressionType': { 'function': { args: [{ default: null, enums: ['property', 'operator', 'or', 'and', 'groupBy', 'function', 'functionParameter'], name: 'value', type: ['string'] }], help: '' } },
                      withQueryEditorExpressionType(value): {
                        sql+: {
                          from+: {
                            type+: {
                              QueryEditorExpressionType: value,
                            },
                          },
                        },
                      },
                    },
                },
            },
          '#withGroupBy': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'GROUP BY part of the SQL expression' } },
          withGroupBy(value): {
            sql+: {
              groupBy: value,
            },
          },
          '#withGroupByMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'GROUP BY part of the SQL expression' } },
          withGroupByMixin(value): {
            sql+: {
              groupBy+: value,
            },
          },
          groupBy+:
            {
              '#withExpressions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withExpressions(value): {
                sql+: {
                  groupBy+: {
                    expressions:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                },
              },
              '#withExpressionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withExpressionsMixin(value): {
                sql+: {
                  groupBy+: {
                    expressions+:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                },
              },
              '#withType': { 'function': { args: [{ default: null, enums: ['and', 'or'], name: 'value', type: ['string'] }], help: '' } },
              withType(value): {
                sql+: {
                  groupBy+: {
                    type: value,
                  },
                },
              },
            },
          '#withLimit': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['integer'] }], help: 'LIMIT part of the SQL expression' } },
          withLimit(value): {
            sql+: {
              limit: value,
            },
          },
          '#withOrderBy': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'ORDER BY part of the SQL expression' } },
          withOrderBy(value): {
            sql+: {
              orderBy: value,
            },
          },
          '#withOrderByMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'ORDER BY part of the SQL expression' } },
          withOrderByMixin(value): {
            sql+: {
              orderBy+: value,
            },
          },
          orderBy+:
            {
              '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withName(value): {
                sql+: {
                  orderBy+: {
                    name: value,
                  },
                },
              },
              '#withParameters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParameters(value): {
                sql+: {
                  orderBy+: {
                    parameters:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                },
              },
              '#withParametersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParametersMixin(value): {
                sql+: {
                  orderBy+: {
                    parameters+:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                },
              },
              parameters+:
                {
                  '#': { help: '', name: 'parameters' },
                  '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value): {
                    name: value,
                  },
                  '#withType': { 'function': { args: [{ default: 'functionParameter', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withType(value='functionParameter'): {
                    type: value,
                  },
                  '#withTypeMixin': { 'function': { args: [{ default: 'functionParameter', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withTypeMixin(value): {
                    type+: value,
                  },
                  type+:
                    {
                      '#withQueryEditorExpressionType': { 'function': { args: [{ default: null, enums: ['property', 'operator', 'or', 'and', 'groupBy', 'function', 'functionParameter'], name: 'value', type: ['string'] }], help: '' } },
                      withQueryEditorExpressionType(value): {
                        type+: {
                          QueryEditorExpressionType: value,
                        },
                      },
                    },
                },
              '#withType': { 'function': { args: [{ default: 'function', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withType(value='function'): {
                sql+: {
                  orderBy+: {
                    type: value,
                  },
                },
              },
              '#withTypeMixin': { 'function': { args: [{ default: 'function', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withTypeMixin(value): {
                sql+: {
                  orderBy+: {
                    type+: value,
                  },
                },
              },
              type+:
                {
                  '#withQueryEditorExpressionType': { 'function': { args: [{ default: null, enums: ['property', 'operator', 'or', 'and', 'groupBy', 'function', 'functionParameter'], name: 'value', type: ['string'] }], help: '' } },
                  withQueryEditorExpressionType(value): {
                    sql+: {
                      orderBy+: {
                        type+: {
                          QueryEditorExpressionType: value,
                        },
                      },
                    },
                  },
                },
            },
          '#withOrderByDirection': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'The sort order of the SQL expression, `ASC` or `DESC`' } },
          withOrderByDirection(value): {
            sql+: {
              orderByDirection: value,
            },
          },
          '#withSelect': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'SELECT part of the SQL expression' } },
          withSelect(value): {
            sql+: {
              select: value,
            },
          },
          '#withSelectMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'SELECT part of the SQL expression' } },
          withSelectMixin(value): {
            sql+: {
              select+: value,
            },
          },
          select+:
            {
              '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
              withName(value): {
                sql+: {
                  select+: {
                    name: value,
                  },
                },
              },
              '#withParameters': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParameters(value): {
                sql+: {
                  select+: {
                    parameters:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                },
              },
              '#withParametersMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withParametersMixin(value): {
                sql+: {
                  select+: {
                    parameters+:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                },
              },
              parameters+:
                {
                  '#': { help: '', name: 'parameters' },
                  '#withName': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withName(value): {
                    name: value,
                  },
                  '#withType': { 'function': { args: [{ default: 'functionParameter', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withType(value='functionParameter'): {
                    type: value,
                  },
                  '#withTypeMixin': { 'function': { args: [{ default: 'functionParameter', enums: null, name: 'value', type: ['string'] }], help: '' } },
                  withTypeMixin(value): {
                    type+: value,
                  },
                  type+:
                    {
                      '#withQueryEditorExpressionType': { 'function': { args: [{ default: null, enums: ['property', 'operator', 'or', 'and', 'groupBy', 'function', 'functionParameter'], name: 'value', type: ['string'] }], help: '' } },
                      withQueryEditorExpressionType(value): {
                        type+: {
                          QueryEditorExpressionType: value,
                        },
                      },
                    },
                },
              '#withType': { 'function': { args: [{ default: 'function', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withType(value='function'): {
                sql+: {
                  select+: {
                    type: value,
                  },
                },
              },
              '#withTypeMixin': { 'function': { args: [{ default: 'function', enums: null, name: 'value', type: ['string'] }], help: '' } },
              withTypeMixin(value): {
                sql+: {
                  select+: {
                    type+: value,
                  },
                },
              },
              type+:
                {
                  '#withQueryEditorExpressionType': { 'function': { args: [{ default: null, enums: ['property', 'operator', 'or', 'and', 'groupBy', 'function', 'functionParameter'], name: 'value', type: ['string'] }], help: '' } },
                  withQueryEditorExpressionType(value): {
                    sql+: {
                      select+: {
                        type+: {
                          QueryEditorExpressionType: value,
                        },
                      },
                    },
                  },
                },
            },
          '#withWhere': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'WHERE part of the SQL expression' } },
          withWhere(value): {
            sql+: {
              where: value,
            },
          },
          '#withWhereMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['object'] }], help: 'WHERE part of the SQL expression' } },
          withWhereMixin(value): {
            sql+: {
              where+: value,
            },
          },
          where+:
            {
              '#withExpressions': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withExpressions(value): {
                sql+: {
                  where+: {
                    expressions:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                },
              },
              '#withExpressionsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '' } },
              withExpressionsMixin(value): {
                sql+: {
                  where+: {
                    expressions+:
                      (if std.isArray(value)
                       then value
                       else [value]),
                  },
                },
              },
              '#withType': { 'function': { args: [{ default: null, enums: ['and', 'or'], name: 'value', type: ['string'] }], help: '' } },
              withType(value): {
                sql+: {
                  where+: {
                    type: value,
                  },
                },
              },
            },
        },
      '#withSqlExpression': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'When the metric query type is set to `Insights`, this field is used to specify the query string.' } },
      withSqlExpression(value): {
        sqlExpression: value,
      },
      '#withStatistic': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['string'] }], help: 'Metric data aggregations over specified periods of time. For detailed definitions of the statistics supported by CloudWatch, see https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/Statistics-definitions.html.' } },
      withStatistic(value): {
        statistic: value,
      },
      '#withStatistics': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '@deprecated use statistic' } },
      withStatistics(value): {
        statistics:
          (if std.isArray(value)
           then value
           else [value]),
      },
      '#withStatisticsMixin': { 'function': { args: [{ default: null, enums: null, name: 'value', type: ['array'] }], help: '@deprecated use statistic' } },
      withStatisticsMixin(value): {
        statistics+:
          (if std.isArray(value)
           then value
           else [value]),
      },
    },
}
+ (import '../custom/query/cloudWatch.libsonnet')
