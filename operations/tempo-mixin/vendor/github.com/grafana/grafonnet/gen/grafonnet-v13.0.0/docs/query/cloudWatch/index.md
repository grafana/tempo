# cloudWatch

grafonnet.query.cloudWatch

## Subpackages

* [LogsQuery.logGroups](LogsQuery/logGroups.md)
* [MetricsQuery.sql.from.QueryEditorFunctionExpression.parameters](MetricsQuery/sql/from/QueryEditorFunctionExpression/parameters.md)
* [MetricsQuery.sql.orderBy.parameters](MetricsQuery/sql/orderBy/parameters.md)
* [MetricsQuery.sql.select.parameters](MetricsQuery/sql/select/parameters.md)

## Index

* [`obj AnnotationQuery`](#obj-annotationquery)
  * [`fn withAccountId(value)`](#fn-annotationquerywithaccountid)
  * [`fn withActionPrefix(value)`](#fn-annotationquerywithactionprefix)
  * [`fn withAlarmNamePrefix(value)`](#fn-annotationquerywithalarmnameprefix)
  * [`fn withDatasource(value)`](#fn-annotationquerywithdatasource)
  * [`fn withDatasourceMixin(value)`](#fn-annotationquerywithdatasourcemixin)
  * [`fn withDimensions(value)`](#fn-annotationquerywithdimensions)
  * [`fn withDimensionsMixin(value)`](#fn-annotationquerywithdimensionsmixin)
  * [`fn withHide(value=true)`](#fn-annotationquerywithhide)
  * [`fn withMatchExact(value=true)`](#fn-annotationquerywithmatchexact)
  * [`fn withMetricName(value)`](#fn-annotationquerywithmetricname)
  * [`fn withNamespace(value)`](#fn-annotationquerywithnamespace)
  * [`fn withPeriod(value)`](#fn-annotationquerywithperiod)
  * [`fn withPrefixMatching(value=true)`](#fn-annotationquerywithprefixmatching)
  * [`fn withQueryMode(value="Annotations")`](#fn-annotationquerywithquerymode)
  * [`fn withQueryType(value)`](#fn-annotationquerywithquerytype)
  * [`fn withRefId(value)`](#fn-annotationquerywithrefid)
  * [`fn withRegion(value)`](#fn-annotationquerywithregion)
  * [`fn withStatistic(value)`](#fn-annotationquerywithstatistic)
  * [`fn withStatistics(value)`](#fn-annotationquerywithstatistics)
  * [`fn withStatisticsMixin(value)`](#fn-annotationquerywithstatisticsmixin)
  * [`obj datasource`](#obj-annotationquerydatasource)
    * [`fn withType(value)`](#fn-annotationquerydatasourcewithtype)
    * [`fn withUid(value)`](#fn-annotationquerydatasourcewithuid)
* [`obj CloudWatchAnnotationQuery`](#obj-cloudwatchannotationquery)
  * [`fn withDatasource(value)`](#fn-cloudwatchannotationquerywithdatasource)
* [`obj CloudWatchLogsQuery`](#obj-cloudwatchlogsquery)
  * [`fn withDatasource(value)`](#fn-cloudwatchlogsquerywithdatasource)
* [`obj CloudWatchMetricsQuery`](#obj-cloudwatchmetricsquery)
  * [`fn withDatasource(value)`](#fn-cloudwatchmetricsquerywithdatasource)
* [`obj LogsQuery`](#obj-logsquery)
  * [`fn withDatasource(value)`](#fn-logsquerywithdatasource)
  * [`fn withDatasourceMixin(value)`](#fn-logsquerywithdatasourcemixin)
  * [`fn withExpression(value)`](#fn-logsquerywithexpression)
  * [`fn withHide(value=true)`](#fn-logsquerywithhide)
  * [`fn withId(value)`](#fn-logsquerywithid)
  * [`fn withLogGroupNames(value)`](#fn-logsquerywithloggroupnames)
  * [`fn withLogGroupNamesMixin(value)`](#fn-logsquerywithloggroupnamesmixin)
  * [`fn withLogGroups(value)`](#fn-logsquerywithloggroups)
  * [`fn withLogGroupsMixin(value)`](#fn-logsquerywithloggroupsmixin)
  * [`fn withQueryLanguage(value)`](#fn-logsquerywithquerylanguage)
  * [`fn withQueryMode(value="Logs")`](#fn-logsquerywithquerymode)
  * [`fn withQueryType(value)`](#fn-logsquerywithquerytype)
  * [`fn withRefId(value)`](#fn-logsquerywithrefid)
  * [`fn withRegion(value)`](#fn-logsquerywithregion)
  * [`fn withStatsGroups(value)`](#fn-logsquerywithstatsgroups)
  * [`fn withStatsGroupsMixin(value)`](#fn-logsquerywithstatsgroupsmixin)
  * [`obj datasource`](#obj-logsquerydatasource)
    * [`fn withType(value)`](#fn-logsquerydatasourcewithtype)
    * [`fn withUid(value)`](#fn-logsquerydatasourcewithuid)
* [`obj MetricsQuery`](#obj-metricsquery)
  * [`fn withAccountId(value)`](#fn-metricsquerywithaccountid)
  * [`fn withAlias(value)`](#fn-metricsquerywithalias)
  * [`fn withDatasource(value)`](#fn-metricsquerywithdatasource)
  * [`fn withDatasourceMixin(value)`](#fn-metricsquerywithdatasourcemixin)
  * [`fn withDimensions(value)`](#fn-metricsquerywithdimensions)
  * [`fn withDimensionsMixin(value)`](#fn-metricsquerywithdimensionsmixin)
  * [`fn withExpression(value)`](#fn-metricsquerywithexpression)
  * [`fn withHide(value=true)`](#fn-metricsquerywithhide)
  * [`fn withId(value)`](#fn-metricsquerywithid)
  * [`fn withLabel(value)`](#fn-metricsquerywithlabel)
  * [`fn withMatchExact(value=true)`](#fn-metricsquerywithmatchexact)
  * [`fn withMetricEditorMode(value)`](#fn-metricsquerywithmetriceditormode)
  * [`fn withMetricName(value)`](#fn-metricsquerywithmetricname)
  * [`fn withMetricQueryType(value)`](#fn-metricsquerywithmetricquerytype)
  * [`fn withNamespace(value)`](#fn-metricsquerywithnamespace)
  * [`fn withPeriod(value)`](#fn-metricsquerywithperiod)
  * [`fn withQueryMode(value="Metrics")`](#fn-metricsquerywithquerymode)
  * [`fn withQueryType(value)`](#fn-metricsquerywithquerytype)
  * [`fn withRefId(value)`](#fn-metricsquerywithrefid)
  * [`fn withRegion(value)`](#fn-metricsquerywithregion)
  * [`fn withSql(value)`](#fn-metricsquerywithsql)
  * [`fn withSqlExpression(value)`](#fn-metricsquerywithsqlexpression)
  * [`fn withSqlMixin(value)`](#fn-metricsquerywithsqlmixin)
  * [`fn withStatistic(value)`](#fn-metricsquerywithstatistic)
  * [`fn withStatistics(value)`](#fn-metricsquerywithstatistics)
  * [`fn withStatisticsMixin(value)`](#fn-metricsquerywithstatisticsmixin)
  * [`obj datasource`](#obj-metricsquerydatasource)
    * [`fn withType(value)`](#fn-metricsquerydatasourcewithtype)
    * [`fn withUid(value)`](#fn-metricsquerydatasourcewithuid)
  * [`obj sql`](#obj-metricsquerysql)
    * [`fn withFrom(value)`](#fn-metricsquerysqlwithfrom)
    * [`fn withFromMixin(value)`](#fn-metricsquerysqlwithfrommixin)
    * [`fn withGroupBy(value)`](#fn-metricsquerysqlwithgroupby)
    * [`fn withGroupByMixin(value)`](#fn-metricsquerysqlwithgroupbymixin)
    * [`fn withLimit(value)`](#fn-metricsquerysqlwithlimit)
    * [`fn withOrderBy(value)`](#fn-metricsquerysqlwithorderby)
    * [`fn withOrderByDirection(value)`](#fn-metricsquerysqlwithorderbydirection)
    * [`fn withOrderByMixin(value)`](#fn-metricsquerysqlwithorderbymixin)
    * [`fn withSelect(value)`](#fn-metricsquerysqlwithselect)
    * [`fn withSelectMixin(value)`](#fn-metricsquerysqlwithselectmixin)
    * [`fn withWhere(value)`](#fn-metricsquerysqlwithwhere)
    * [`fn withWhereMixin(value)`](#fn-metricsquerysqlwithwheremixin)
    * [`obj from`](#obj-metricsquerysqlfrom)
      * [`fn withQueryEditorFunctionExpression(value)`](#fn-metricsquerysqlfromwithqueryeditorfunctionexpression)
      * [`fn withQueryEditorFunctionExpressionMixin(value)`](#fn-metricsquerysqlfromwithqueryeditorfunctionexpressionmixin)
      * [`fn withQueryEditorPropertyExpression(value)`](#fn-metricsquerysqlfromwithqueryeditorpropertyexpression)
      * [`fn withQueryEditorPropertyExpressionMixin(value)`](#fn-metricsquerysqlfromwithqueryeditorpropertyexpressionmixin)
      * [`obj QueryEditorFunctionExpression`](#obj-metricsquerysqlfromqueryeditorfunctionexpression)
        * [`fn withName(value)`](#fn-metricsquerysqlfromqueryeditorfunctionexpressionwithname)
        * [`fn withParameters(value)`](#fn-metricsquerysqlfromqueryeditorfunctionexpressionwithparameters)
        * [`fn withParametersMixin(value)`](#fn-metricsquerysqlfromqueryeditorfunctionexpressionwithparametersmixin)
        * [`fn withType(value="function")`](#fn-metricsquerysqlfromqueryeditorfunctionexpressionwithtype)
        * [`fn withTypeMixin(value="function")`](#fn-metricsquerysqlfromqueryeditorfunctionexpressionwithtypemixin)
        * [`obj type`](#obj-metricsquerysqlfromqueryeditorfunctionexpressiontype)
          * [`fn withQueryEditorExpressionType(value)`](#fn-metricsquerysqlfromqueryeditorfunctionexpressiontypewithqueryeditorexpressiontype)
      * [`obj QueryEditorPropertyExpression`](#obj-metricsquerysqlfromqueryeditorpropertyexpression)
        * [`fn withProperty(value)`](#fn-metricsquerysqlfromqueryeditorpropertyexpressionwithproperty)
        * [`fn withPropertyMixin(value)`](#fn-metricsquerysqlfromqueryeditorpropertyexpressionwithpropertymixin)
        * [`fn withType(value="property")`](#fn-metricsquerysqlfromqueryeditorpropertyexpressionwithtype)
        * [`fn withTypeMixin(value="property")`](#fn-metricsquerysqlfromqueryeditorpropertyexpressionwithtypemixin)
        * [`obj property`](#obj-metricsquerysqlfromqueryeditorpropertyexpressionproperty)
          * [`fn withName(value)`](#fn-metricsquerysqlfromqueryeditorpropertyexpressionpropertywithname)
          * [`fn withType(value="string")`](#fn-metricsquerysqlfromqueryeditorpropertyexpressionpropertywithtype)
          * [`fn withTypeMixin(value="string")`](#fn-metricsquerysqlfromqueryeditorpropertyexpressionpropertywithtypemixin)
          * [`obj type`](#obj-metricsquerysqlfromqueryeditorpropertyexpressionpropertytype)
            * [`fn withQueryEditorPropertyType(value)`](#fn-metricsquerysqlfromqueryeditorpropertyexpressionpropertytypewithqueryeditorpropertytype)
        * [`obj type`](#obj-metricsquerysqlfromqueryeditorpropertyexpressiontype)
          * [`fn withQueryEditorExpressionType(value)`](#fn-metricsquerysqlfromqueryeditorpropertyexpressiontypewithqueryeditorexpressiontype)
    * [`obj groupBy`](#obj-metricsquerysqlgroupby)
      * [`fn withExpressions(value)`](#fn-metricsquerysqlgroupbywithexpressions)
      * [`fn withExpressionsMixin(value)`](#fn-metricsquerysqlgroupbywithexpressionsmixin)
      * [`fn withType(value)`](#fn-metricsquerysqlgroupbywithtype)
    * [`obj orderBy`](#obj-metricsquerysqlorderby)
      * [`fn withName(value)`](#fn-metricsquerysqlorderbywithname)
      * [`fn withParameters(value)`](#fn-metricsquerysqlorderbywithparameters)
      * [`fn withParametersMixin(value)`](#fn-metricsquerysqlorderbywithparametersmixin)
      * [`fn withType(value="function")`](#fn-metricsquerysqlorderbywithtype)
      * [`fn withTypeMixin(value="function")`](#fn-metricsquerysqlorderbywithtypemixin)
      * [`obj type`](#obj-metricsquerysqlorderbytype)
        * [`fn withQueryEditorExpressionType(value)`](#fn-metricsquerysqlorderbytypewithqueryeditorexpressiontype)
    * [`obj select`](#obj-metricsquerysqlselect)
      * [`fn withName(value)`](#fn-metricsquerysqlselectwithname)
      * [`fn withParameters(value)`](#fn-metricsquerysqlselectwithparameters)
      * [`fn withParametersMixin(value)`](#fn-metricsquerysqlselectwithparametersmixin)
      * [`fn withType(value="function")`](#fn-metricsquerysqlselectwithtype)
      * [`fn withTypeMixin(value="function")`](#fn-metricsquerysqlselectwithtypemixin)
      * [`obj type`](#obj-metricsquerysqlselecttype)
        * [`fn withQueryEditorExpressionType(value)`](#fn-metricsquerysqlselecttypewithqueryeditorexpressiontype)
    * [`obj where`](#obj-metricsquerysqlwhere)
      * [`fn withExpressions(value)`](#fn-metricsquerysqlwherewithexpressions)
      * [`fn withExpressionsMixin(value)`](#fn-metricsquerysqlwherewithexpressionsmixin)
      * [`fn withType(value)`](#fn-metricsquerysqlwherewithtype)

## Fields

### obj AnnotationQuery


#### fn AnnotationQuery.withAccountId

```jsonnet
AnnotationQuery.withAccountId(value)
```

PARAMETERS:

* **value** (`string`)

The ID of the AWS account to query for the metric, specifying `all` will query all accounts that the monitoring account is permitted to query.
#### fn AnnotationQuery.withActionPrefix

```jsonnet
AnnotationQuery.withActionPrefix(value)
```

PARAMETERS:

* **value** (`string`)

Use this parameter to filter the results of the operation to only those alarms
that use a certain alarm action. For example, you could specify the ARN of
an SNS topic to find all alarms that send notifications to that topic.
e.g. `arn:aws:sns:us-east-1:123456789012:my-app-` would match `arn:aws:sns:us-east-1:123456789012:my-app-action`
but not match `arn:aws:sns:us-east-1:123456789012:your-app-action`
#### fn AnnotationQuery.withAlarmNamePrefix

```jsonnet
AnnotationQuery.withAlarmNamePrefix(value)
```

PARAMETERS:

* **value** (`string`)

An alarm name prefix. If you specify this parameter, you receive information
about all alarms that have names that start with this prefix.
e.g. `my-team-service-` would match `my-team-service-high-cpu` but not match `your-team-service-high-cpu`
#### fn AnnotationQuery.withDatasource

```jsonnet
AnnotationQuery.withDatasource(value)
```

PARAMETERS:

* **value** (`object`)

For mixed data sources the selected datasource is on the query level.
For non mixed scenarios this is undefined.
TODO find a better way to do this ^ that's friendly to schema
TODO this shouldn't be unknown but DataSourceRef | null
#### fn AnnotationQuery.withDatasourceMixin

```jsonnet
AnnotationQuery.withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)

For mixed data sources the selected datasource is on the query level.
For non mixed scenarios this is undefined.
TODO find a better way to do this ^ that's friendly to schema
TODO this shouldn't be unknown but DataSourceRef | null
#### fn AnnotationQuery.withDimensions

```jsonnet
AnnotationQuery.withDimensions(value)
```

PARAMETERS:

* **value** (`object`)

A name/value pair that is part of the identity of a metric. For example, you can get statistics for a specific EC2 instance by specifying the InstanceId dimension when you search for metrics.
#### fn AnnotationQuery.withDimensionsMixin

```jsonnet
AnnotationQuery.withDimensionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

A name/value pair that is part of the identity of a metric. For example, you can get statistics for a specific EC2 instance by specifying the InstanceId dimension when you search for metrics.
#### fn AnnotationQuery.withHide

```jsonnet
AnnotationQuery.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
#### fn AnnotationQuery.withMatchExact

```jsonnet
AnnotationQuery.withMatchExact(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Only show metrics that exactly match all defined dimension names.
#### fn AnnotationQuery.withMetricName

```jsonnet
AnnotationQuery.withMetricName(value)
```

PARAMETERS:

* **value** (`string`)

Name of the metric
#### fn AnnotationQuery.withNamespace

```jsonnet
AnnotationQuery.withNamespace(value)
```

PARAMETERS:

* **value** (`string`)

A namespace is a container for CloudWatch metrics. Metrics in different namespaces are isolated from each other, so that metrics from different applications are not mistakenly aggregated into the same statistics. For example, Amazon EC2 uses the AWS/EC2 namespace.
#### fn AnnotationQuery.withPeriod

```jsonnet
AnnotationQuery.withPeriod(value)
```

PARAMETERS:

* **value** (`string`)

The length of time associated with a specific Amazon CloudWatch statistic. Can be specified by a number of seconds, 'auto', or as a duration string e.g. '15m' being 15 minutes
#### fn AnnotationQuery.withPrefixMatching

```jsonnet
AnnotationQuery.withPrefixMatching(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Enable matching on the prefix of the action name or alarm name, specify the prefixes with actionPrefix and/or alarmNamePrefix
#### fn AnnotationQuery.withQueryMode

```jsonnet
AnnotationQuery.withQueryMode(value="Annotations")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"Annotations"`
   - valid values: `"Metrics"`, `"Logs"`, `"Annotations"`

Whether a query is a Metrics, Logs, or Annotations query
#### fn AnnotationQuery.withQueryType

```jsonnet
AnnotationQuery.withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

Specify the query flavor
TODO make this required and give it a default
#### fn AnnotationQuery.withRefId

```jsonnet
AnnotationQuery.withRefId(value)
```

PARAMETERS:

* **value** (`string`)

A unique identifier for the query within the list of targets.
In server side expressions, the refId is used as a variable name to identify results.
By default, the UI will assign A->Z; however setting meaningful names may be useful.
#### fn AnnotationQuery.withRegion

```jsonnet
AnnotationQuery.withRegion(value)
```

PARAMETERS:

* **value** (`string`)

AWS region to query for the metric
#### fn AnnotationQuery.withStatistic

```jsonnet
AnnotationQuery.withStatistic(value)
```

PARAMETERS:

* **value** (`string`)

Metric data aggregations over specified periods of time. For detailed definitions of the statistics supported by CloudWatch, see https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/Statistics-definitions.html.
#### fn AnnotationQuery.withStatistics

```jsonnet
AnnotationQuery.withStatistics(value)
```

PARAMETERS:

* **value** (`array`)

@deprecated use statistic
#### fn AnnotationQuery.withStatisticsMixin

```jsonnet
AnnotationQuery.withStatisticsMixin(value)
```

PARAMETERS:

* **value** (`array`)

@deprecated use statistic
#### obj AnnotationQuery.datasource


##### fn AnnotationQuery.datasource.withType

```jsonnet
AnnotationQuery.datasource.withType(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
##### fn AnnotationQuery.datasource.withUid

```jsonnet
AnnotationQuery.datasource.withUid(value)
```

PARAMETERS:

* **value** (`string`)

Specific datasource instance
### obj CloudWatchAnnotationQuery


#### fn CloudWatchAnnotationQuery.withDatasource

```jsonnet
CloudWatchAnnotationQuery.withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### obj CloudWatchLogsQuery


#### fn CloudWatchLogsQuery.withDatasource

```jsonnet
CloudWatchLogsQuery.withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### obj CloudWatchMetricsQuery


#### fn CloudWatchMetricsQuery.withDatasource

```jsonnet
CloudWatchMetricsQuery.withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### obj LogsQuery


#### fn LogsQuery.withDatasource

```jsonnet
LogsQuery.withDatasource(value)
```

PARAMETERS:

* **value** (`object`)

For mixed data sources the selected datasource is on the query level.
For non mixed scenarios this is undefined.
TODO find a better way to do this ^ that's friendly to schema
TODO this shouldn't be unknown but DataSourceRef | null
#### fn LogsQuery.withDatasourceMixin

```jsonnet
LogsQuery.withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)

For mixed data sources the selected datasource is on the query level.
For non mixed scenarios this is undefined.
TODO find a better way to do this ^ that's friendly to schema
TODO this shouldn't be unknown but DataSourceRef | null
#### fn LogsQuery.withExpression

```jsonnet
LogsQuery.withExpression(value)
```

PARAMETERS:

* **value** (`string`)

The CloudWatch Logs Insights query to execute
#### fn LogsQuery.withHide

```jsonnet
LogsQuery.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
#### fn LogsQuery.withId

```jsonnet
LogsQuery.withId(value)
```

PARAMETERS:

* **value** (`string`)


#### fn LogsQuery.withLogGroupNames

```jsonnet
LogsQuery.withLogGroupNames(value)
```

PARAMETERS:

* **value** (`array`)

@deprecated use logGroups
#### fn LogsQuery.withLogGroupNamesMixin

```jsonnet
LogsQuery.withLogGroupNamesMixin(value)
```

PARAMETERS:

* **value** (`array`)

@deprecated use logGroups
#### fn LogsQuery.withLogGroups

```jsonnet
LogsQuery.withLogGroups(value)
```

PARAMETERS:

* **value** (`array`)

Log groups to query
#### fn LogsQuery.withLogGroupsMixin

```jsonnet
LogsQuery.withLogGroupsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Log groups to query
#### fn LogsQuery.withQueryLanguage

```jsonnet
LogsQuery.withQueryLanguage(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"CWLI"`, `"SQL"`, `"PPL"`

Language used for querying logs, can be CWLI, SQL, or PPL. If empty, the default language is CWLI.
#### fn LogsQuery.withQueryMode

```jsonnet
LogsQuery.withQueryMode(value="Logs")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"Logs"`
   - valid values: `"Metrics"`, `"Logs"`, `"Annotations"`

Whether a query is a Metrics, Logs, or Annotations query
#### fn LogsQuery.withQueryType

```jsonnet
LogsQuery.withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

Specify the query flavor
TODO make this required and give it a default
#### fn LogsQuery.withRefId

```jsonnet
LogsQuery.withRefId(value)
```

PARAMETERS:

* **value** (`string`)

A unique identifier for the query within the list of targets.
In server side expressions, the refId is used as a variable name to identify results.
By default, the UI will assign A->Z; however setting meaningful names may be useful.
#### fn LogsQuery.withRegion

```jsonnet
LogsQuery.withRegion(value)
```

PARAMETERS:

* **value** (`string`)

AWS region to query for the logs
#### fn LogsQuery.withStatsGroups

```jsonnet
LogsQuery.withStatsGroups(value)
```

PARAMETERS:

* **value** (`array`)

Fields to group the results by, this field is automatically populated whenever the query is updated
#### fn LogsQuery.withStatsGroupsMixin

```jsonnet
LogsQuery.withStatsGroupsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Fields to group the results by, this field is automatically populated whenever the query is updated
#### obj LogsQuery.datasource


##### fn LogsQuery.datasource.withType

```jsonnet
LogsQuery.datasource.withType(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
##### fn LogsQuery.datasource.withUid

```jsonnet
LogsQuery.datasource.withUid(value)
```

PARAMETERS:

* **value** (`string`)

Specific datasource instance
### obj MetricsQuery


#### fn MetricsQuery.withAccountId

```jsonnet
MetricsQuery.withAccountId(value)
```

PARAMETERS:

* **value** (`string`)

The ID of the AWS account to query for the metric, specifying `all` will query all accounts that the monitoring account is permitted to query.
#### fn MetricsQuery.withAlias

```jsonnet
MetricsQuery.withAlias(value)
```

PARAMETERS:

* **value** (`string`)

Deprecated: use label
@deprecated use label
#### fn MetricsQuery.withDatasource

```jsonnet
MetricsQuery.withDatasource(value)
```

PARAMETERS:

* **value** (`object`)

For mixed data sources the selected datasource is on the query level.
For non mixed scenarios this is undefined.
TODO find a better way to do this ^ that's friendly to schema
TODO this shouldn't be unknown but DataSourceRef | null
#### fn MetricsQuery.withDatasourceMixin

```jsonnet
MetricsQuery.withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)

For mixed data sources the selected datasource is on the query level.
For non mixed scenarios this is undefined.
TODO find a better way to do this ^ that's friendly to schema
TODO this shouldn't be unknown but DataSourceRef | null
#### fn MetricsQuery.withDimensions

```jsonnet
MetricsQuery.withDimensions(value)
```

PARAMETERS:

* **value** (`object`)

A name/value pair that is part of the identity of a metric. For example, you can get statistics for a specific EC2 instance by specifying the InstanceId dimension when you search for metrics.
#### fn MetricsQuery.withDimensionsMixin

```jsonnet
MetricsQuery.withDimensionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

A name/value pair that is part of the identity of a metric. For example, you can get statistics for a specific EC2 instance by specifying the InstanceId dimension when you search for metrics.
#### fn MetricsQuery.withExpression

```jsonnet
MetricsQuery.withExpression(value)
```

PARAMETERS:

* **value** (`string`)

Math expression query
#### fn MetricsQuery.withHide

```jsonnet
MetricsQuery.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
#### fn MetricsQuery.withId

```jsonnet
MetricsQuery.withId(value)
```

PARAMETERS:

* **value** (`string`)

ID can be used to reference other queries in math expressions. The ID can include numbers, letters, and underscore, and must start with a lowercase letter.
#### fn MetricsQuery.withLabel

```jsonnet
MetricsQuery.withLabel(value)
```

PARAMETERS:

* **value** (`string`)

Change the time series legend names using dynamic labels. See https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/graph-dynamic-labels.html for more details.
#### fn MetricsQuery.withMatchExact

```jsonnet
MetricsQuery.withMatchExact(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Only show metrics that exactly match all defined dimension names.
#### fn MetricsQuery.withMetricEditorMode

```jsonnet
MetricsQuery.withMetricEditorMode(value)
```

PARAMETERS:

* **value** (`integer`)
   - valid values: `0`, `1`

Whether to use the query builder or code editor to create the query
#### fn MetricsQuery.withMetricName

```jsonnet
MetricsQuery.withMetricName(value)
```

PARAMETERS:

* **value** (`string`)

Name of the metric
#### fn MetricsQuery.withMetricQueryType

```jsonnet
MetricsQuery.withMetricQueryType(value)
```

PARAMETERS:

* **value** (`integer`)
   - valid values: `0`, `1`

Whether to use a metric search or metric insights query
#### fn MetricsQuery.withNamespace

```jsonnet
MetricsQuery.withNamespace(value)
```

PARAMETERS:

* **value** (`string`)

A namespace is a container for CloudWatch metrics. Metrics in different namespaces are isolated from each other, so that metrics from different applications are not mistakenly aggregated into the same statistics. For example, Amazon EC2 uses the AWS/EC2 namespace.
#### fn MetricsQuery.withPeriod

```jsonnet
MetricsQuery.withPeriod(value)
```

PARAMETERS:

* **value** (`string`)

The length of time associated with a specific Amazon CloudWatch statistic. Can be specified by a number of seconds, 'auto', or as a duration string e.g. '15m' being 15 minutes
#### fn MetricsQuery.withQueryMode

```jsonnet
MetricsQuery.withQueryMode(value="Metrics")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"Metrics"`
   - valid values: `"Metrics"`, `"Logs"`, `"Annotations"`

Whether a query is a Metrics, Logs, or Annotations query
#### fn MetricsQuery.withQueryType

```jsonnet
MetricsQuery.withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

Specify the query flavor
TODO make this required and give it a default
#### fn MetricsQuery.withRefId

```jsonnet
MetricsQuery.withRefId(value)
```

PARAMETERS:

* **value** (`string`)

A unique identifier for the query within the list of targets.
In server side expressions, the refId is used as a variable name to identify results.
By default, the UI will assign A->Z; however setting meaningful names may be useful.
#### fn MetricsQuery.withRegion

```jsonnet
MetricsQuery.withRegion(value)
```

PARAMETERS:

* **value** (`string`)

AWS region to query for the metric
#### fn MetricsQuery.withSql

```jsonnet
MetricsQuery.withSql(value)
```

PARAMETERS:

* **value** (`object`)

When the metric query type is set to `Insights` and the `metricEditorMode` is set to `Builder`, this field is used to build up an object representation of a SQL query.
#### fn MetricsQuery.withSqlExpression

```jsonnet
MetricsQuery.withSqlExpression(value)
```

PARAMETERS:

* **value** (`string`)

When the metric query type is set to `Insights`, this field is used to specify the query string.
#### fn MetricsQuery.withSqlMixin

```jsonnet
MetricsQuery.withSqlMixin(value)
```

PARAMETERS:

* **value** (`object`)

When the metric query type is set to `Insights` and the `metricEditorMode` is set to `Builder`, this field is used to build up an object representation of a SQL query.
#### fn MetricsQuery.withStatistic

```jsonnet
MetricsQuery.withStatistic(value)
```

PARAMETERS:

* **value** (`string`)

Metric data aggregations over specified periods of time. For detailed definitions of the statistics supported by CloudWatch, see https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/Statistics-definitions.html.
#### fn MetricsQuery.withStatistics

```jsonnet
MetricsQuery.withStatistics(value)
```

PARAMETERS:

* **value** (`array`)

@deprecated use statistic
#### fn MetricsQuery.withStatisticsMixin

```jsonnet
MetricsQuery.withStatisticsMixin(value)
```

PARAMETERS:

* **value** (`array`)

@deprecated use statistic
#### obj MetricsQuery.datasource


##### fn MetricsQuery.datasource.withType

```jsonnet
MetricsQuery.datasource.withType(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
##### fn MetricsQuery.datasource.withUid

```jsonnet
MetricsQuery.datasource.withUid(value)
```

PARAMETERS:

* **value** (`string`)

Specific datasource instance
#### obj MetricsQuery.sql


##### fn MetricsQuery.sql.withFrom

```jsonnet
MetricsQuery.sql.withFrom(value)
```

PARAMETERS:

* **value** (`object`)

FROM part of the SQL expression
##### fn MetricsQuery.sql.withFromMixin

```jsonnet
MetricsQuery.sql.withFromMixin(value)
```

PARAMETERS:

* **value** (`object`)

FROM part of the SQL expression
##### fn MetricsQuery.sql.withGroupBy

```jsonnet
MetricsQuery.sql.withGroupBy(value)
```

PARAMETERS:

* **value** (`object`)

GROUP BY part of the SQL expression
##### fn MetricsQuery.sql.withGroupByMixin

```jsonnet
MetricsQuery.sql.withGroupByMixin(value)
```

PARAMETERS:

* **value** (`object`)

GROUP BY part of the SQL expression
##### fn MetricsQuery.sql.withLimit

```jsonnet
MetricsQuery.sql.withLimit(value)
```

PARAMETERS:

* **value** (`integer`)

LIMIT part of the SQL expression
##### fn MetricsQuery.sql.withOrderBy

```jsonnet
MetricsQuery.sql.withOrderBy(value)
```

PARAMETERS:

* **value** (`object`)

ORDER BY part of the SQL expression
##### fn MetricsQuery.sql.withOrderByDirection

```jsonnet
MetricsQuery.sql.withOrderByDirection(value)
```

PARAMETERS:

* **value** (`string`)

The sort order of the SQL expression, `ASC` or `DESC`
##### fn MetricsQuery.sql.withOrderByMixin

```jsonnet
MetricsQuery.sql.withOrderByMixin(value)
```

PARAMETERS:

* **value** (`object`)

ORDER BY part of the SQL expression
##### fn MetricsQuery.sql.withSelect

```jsonnet
MetricsQuery.sql.withSelect(value)
```

PARAMETERS:

* **value** (`object`)

SELECT part of the SQL expression
##### fn MetricsQuery.sql.withSelectMixin

```jsonnet
MetricsQuery.sql.withSelectMixin(value)
```

PARAMETERS:

* **value** (`object`)

SELECT part of the SQL expression
##### fn MetricsQuery.sql.withWhere

```jsonnet
MetricsQuery.sql.withWhere(value)
```

PARAMETERS:

* **value** (`object`)

WHERE part of the SQL expression
##### fn MetricsQuery.sql.withWhereMixin

```jsonnet
MetricsQuery.sql.withWhereMixin(value)
```

PARAMETERS:

* **value** (`object`)

WHERE part of the SQL expression
##### obj MetricsQuery.sql.from


###### fn MetricsQuery.sql.from.withQueryEditorFunctionExpression

```jsonnet
MetricsQuery.sql.from.withQueryEditorFunctionExpression(value)
```

PARAMETERS:

* **value** (`object`)


###### fn MetricsQuery.sql.from.withQueryEditorFunctionExpressionMixin

```jsonnet
MetricsQuery.sql.from.withQueryEditorFunctionExpressionMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### fn MetricsQuery.sql.from.withQueryEditorPropertyExpression

```jsonnet
MetricsQuery.sql.from.withQueryEditorPropertyExpression(value)
```

PARAMETERS:

* **value** (`object`)


###### fn MetricsQuery.sql.from.withQueryEditorPropertyExpressionMixin

```jsonnet
MetricsQuery.sql.from.withQueryEditorPropertyExpressionMixin(value)
```

PARAMETERS:

* **value** (`object`)


###### obj MetricsQuery.sql.from.QueryEditorFunctionExpression


####### fn MetricsQuery.sql.from.QueryEditorFunctionExpression.withName

```jsonnet
MetricsQuery.sql.from.QueryEditorFunctionExpression.withName(value)
```

PARAMETERS:

* **value** (`string`)


####### fn MetricsQuery.sql.from.QueryEditorFunctionExpression.withParameters

```jsonnet
MetricsQuery.sql.from.QueryEditorFunctionExpression.withParameters(value)
```

PARAMETERS:

* **value** (`array`)


####### fn MetricsQuery.sql.from.QueryEditorFunctionExpression.withParametersMixin

```jsonnet
MetricsQuery.sql.from.QueryEditorFunctionExpression.withParametersMixin(value)
```

PARAMETERS:

* **value** (`array`)


####### fn MetricsQuery.sql.from.QueryEditorFunctionExpression.withType

```jsonnet
MetricsQuery.sql.from.QueryEditorFunctionExpression.withType(value="function")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"function"`


####### fn MetricsQuery.sql.from.QueryEditorFunctionExpression.withTypeMixin

```jsonnet
MetricsQuery.sql.from.QueryEditorFunctionExpression.withTypeMixin(value="function")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"function"`


####### obj MetricsQuery.sql.from.QueryEditorFunctionExpression.type


######## fn MetricsQuery.sql.from.QueryEditorFunctionExpression.type.withQueryEditorExpressionType

```jsonnet
MetricsQuery.sql.from.QueryEditorFunctionExpression.type.withQueryEditorExpressionType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"property"`, `"operator"`, `"or"`, `"and"`, `"groupBy"`, `"function"`, `"functionParameter"`


###### obj MetricsQuery.sql.from.QueryEditorPropertyExpression


####### fn MetricsQuery.sql.from.QueryEditorPropertyExpression.withProperty

```jsonnet
MetricsQuery.sql.from.QueryEditorPropertyExpression.withProperty(value)
```

PARAMETERS:

* **value** (`object`)


####### fn MetricsQuery.sql.from.QueryEditorPropertyExpression.withPropertyMixin

```jsonnet
MetricsQuery.sql.from.QueryEditorPropertyExpression.withPropertyMixin(value)
```

PARAMETERS:

* **value** (`object`)


####### fn MetricsQuery.sql.from.QueryEditorPropertyExpression.withType

```jsonnet
MetricsQuery.sql.from.QueryEditorPropertyExpression.withType(value="property")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"property"`


####### fn MetricsQuery.sql.from.QueryEditorPropertyExpression.withTypeMixin

```jsonnet
MetricsQuery.sql.from.QueryEditorPropertyExpression.withTypeMixin(value="property")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"property"`


####### obj MetricsQuery.sql.from.QueryEditorPropertyExpression.property


######## fn MetricsQuery.sql.from.QueryEditorPropertyExpression.property.withName

```jsonnet
MetricsQuery.sql.from.QueryEditorPropertyExpression.property.withName(value)
```

PARAMETERS:

* **value** (`string`)


######## fn MetricsQuery.sql.from.QueryEditorPropertyExpression.property.withType

```jsonnet
MetricsQuery.sql.from.QueryEditorPropertyExpression.property.withType(value="string")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"string"`


######## fn MetricsQuery.sql.from.QueryEditorPropertyExpression.property.withTypeMixin

```jsonnet
MetricsQuery.sql.from.QueryEditorPropertyExpression.property.withTypeMixin(value="string")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"string"`


######## obj MetricsQuery.sql.from.QueryEditorPropertyExpression.property.type


######### fn MetricsQuery.sql.from.QueryEditorPropertyExpression.property.type.withQueryEditorPropertyType

```jsonnet
MetricsQuery.sql.from.QueryEditorPropertyExpression.property.type.withQueryEditorPropertyType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"string"`


####### obj MetricsQuery.sql.from.QueryEditorPropertyExpression.type


######## fn MetricsQuery.sql.from.QueryEditorPropertyExpression.type.withQueryEditorExpressionType

```jsonnet
MetricsQuery.sql.from.QueryEditorPropertyExpression.type.withQueryEditorExpressionType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"property"`, `"operator"`, `"or"`, `"and"`, `"groupBy"`, `"function"`, `"functionParameter"`


##### obj MetricsQuery.sql.groupBy


###### fn MetricsQuery.sql.groupBy.withExpressions

```jsonnet
MetricsQuery.sql.groupBy.withExpressions(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricsQuery.sql.groupBy.withExpressionsMixin

```jsonnet
MetricsQuery.sql.groupBy.withExpressionsMixin(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricsQuery.sql.groupBy.withType

```jsonnet
MetricsQuery.sql.groupBy.withType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"and"`, `"or"`


##### obj MetricsQuery.sql.orderBy


###### fn MetricsQuery.sql.orderBy.withName

```jsonnet
MetricsQuery.sql.orderBy.withName(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricsQuery.sql.orderBy.withParameters

```jsonnet
MetricsQuery.sql.orderBy.withParameters(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricsQuery.sql.orderBy.withParametersMixin

```jsonnet
MetricsQuery.sql.orderBy.withParametersMixin(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricsQuery.sql.orderBy.withType

```jsonnet
MetricsQuery.sql.orderBy.withType(value="function")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"function"`


###### fn MetricsQuery.sql.orderBy.withTypeMixin

```jsonnet
MetricsQuery.sql.orderBy.withTypeMixin(value="function")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"function"`


###### obj MetricsQuery.sql.orderBy.type


####### fn MetricsQuery.sql.orderBy.type.withQueryEditorExpressionType

```jsonnet
MetricsQuery.sql.orderBy.type.withQueryEditorExpressionType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"property"`, `"operator"`, `"or"`, `"and"`, `"groupBy"`, `"function"`, `"functionParameter"`


##### obj MetricsQuery.sql.select


###### fn MetricsQuery.sql.select.withName

```jsonnet
MetricsQuery.sql.select.withName(value)
```

PARAMETERS:

* **value** (`string`)


###### fn MetricsQuery.sql.select.withParameters

```jsonnet
MetricsQuery.sql.select.withParameters(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricsQuery.sql.select.withParametersMixin

```jsonnet
MetricsQuery.sql.select.withParametersMixin(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricsQuery.sql.select.withType

```jsonnet
MetricsQuery.sql.select.withType(value="function")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"function"`


###### fn MetricsQuery.sql.select.withTypeMixin

```jsonnet
MetricsQuery.sql.select.withTypeMixin(value="function")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"function"`


###### obj MetricsQuery.sql.select.type


####### fn MetricsQuery.sql.select.type.withQueryEditorExpressionType

```jsonnet
MetricsQuery.sql.select.type.withQueryEditorExpressionType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"property"`, `"operator"`, `"or"`, `"and"`, `"groupBy"`, `"function"`, `"functionParameter"`


##### obj MetricsQuery.sql.where


###### fn MetricsQuery.sql.where.withExpressions

```jsonnet
MetricsQuery.sql.where.withExpressions(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricsQuery.sql.where.withExpressionsMixin

```jsonnet
MetricsQuery.sql.where.withExpressionsMixin(value)
```

PARAMETERS:

* **value** (`array`)


###### fn MetricsQuery.sql.where.withType

```jsonnet
MetricsQuery.sql.where.withType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"and"`, `"or"`

