# elasticsearch

grafonnet.query.elasticsearch

## Subpackages

* [bucketAggs](bucketAggs/index.md)
* [metrics](metrics/index.md)

## Index

* [`fn withAlias(value)`](#fn-withalias)
* [`fn withBucketAggs(value)`](#fn-withbucketaggs)
* [`fn withBucketAggsMixin(value)`](#fn-withbucketaggsmixin)
* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withMetrics(value)`](#fn-withmetrics)
* [`fn withMetricsMixin(value)`](#fn-withmetricsmixin)
* [`fn withQuery(value)`](#fn-withquery)
* [`fn withQueryType(value)`](#fn-withquerytype)
* [`fn withRefId(value)`](#fn-withrefid)
* [`fn withTimeField(value)`](#fn-withtimefield)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)

## Fields

### fn withAlias

```jsonnet
withAlias(value)
```

PARAMETERS:

* **value** (`string`)

Alias pattern
### fn withBucketAggs

```jsonnet
withBucketAggs(value)
```

PARAMETERS:

* **value** (`array`)

List of bucket aggregations
### fn withBucketAggsMixin

```jsonnet
withBucketAggsMixin(value)
```

PARAMETERS:

* **value** (`array`)

List of bucket aggregations
### fn withDatasource

```jsonnet
withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
### fn withMetrics

```jsonnet
withMetrics(value)
```

PARAMETERS:

* **value** (`array`)

List of metric aggregations
### fn withMetricsMixin

```jsonnet
withMetricsMixin(value)
```

PARAMETERS:

* **value** (`array`)

List of metric aggregations
### fn withQuery

```jsonnet
withQuery(value)
```

PARAMETERS:

* **value** (`string`)

Lucene query
### fn withQueryType

```jsonnet
withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

Specify the query flavor
TODO make this required and give it a default
### fn withRefId

```jsonnet
withRefId(value)
```

PARAMETERS:

* **value** (`string`)

A unique identifier for the query within the list of targets.
In server side expressions, the refId is used as a variable name to identify results.
By default, the UI will assign A->Z; however setting meaningful names may be useful.
### fn withTimeField

```jsonnet
withTimeField(value)
```

PARAMETERS:

* **value** (`string`)

Name of time field
### obj datasource


#### fn datasource.withType

```jsonnet
datasource.withType(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
#### fn datasource.withUid

```jsonnet
datasource.withUid(value)
```

PARAMETERS:

* **value** (`string`)

Specific datasource instance