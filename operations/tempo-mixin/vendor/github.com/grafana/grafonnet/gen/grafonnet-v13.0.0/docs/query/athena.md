# athena

grafonnet.query.athena

## Index

* [`fn withColumn(value)`](#fn-withcolumn)
* [`fn withConnectionArgs(value)`](#fn-withconnectionargs)
* [`fn withConnectionArgsMixin(value)`](#fn-withconnectionargsmixin)
* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withDatasourceMixin(value)`](#fn-withdatasourcemixin)
* [`fn withFormat(value)`](#fn-withformat)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withQueryID(value)`](#fn-withqueryid)
* [`fn withQueryType(value)`](#fn-withquerytype)
* [`fn withRawSQL(value="")`](#fn-withrawsql)
* [`fn withRefId(value)`](#fn-withrefid)
* [`fn withTable(value)`](#fn-withtable)
* [`obj connectionArgs`](#obj-connectionargs)
  * [`fn withCatalog(value="__default")`](#fn-connectionargswithcatalog)
  * [`fn withDatabase(value="__default")`](#fn-connectionargswithdatabase)
  * [`fn withRegion(value="__default")`](#fn-connectionargswithregion)
  * [`fn withResultReuseEnabled(value=true)`](#fn-connectionargswithresultreuseenabled)
  * [`fn withResultReuseMaxAgeInMinutes(value=60)`](#fn-connectionargswithresultreusemaxageinminutes)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)

## Fields

### fn withColumn

```jsonnet
withColumn(value)
```

PARAMETERS:

* **value** (`string`)


### fn withConnectionArgs

```jsonnet
withConnectionArgs(value)
```

PARAMETERS:

* **value** (`object`)


### fn withConnectionArgsMixin

```jsonnet
withConnectionArgsMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withDatasource

```jsonnet
withDatasource(value)
```

PARAMETERS:

* **value** (`object`)

For mixed data sources the selected datasource is on the query level.
For non mixed scenarios this is undefined.
TODO find a better way to do this ^ that's friendly to schema
TODO this shouldn't be unknown but DataSourceRef | null
### fn withDatasourceMixin

```jsonnet
withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)

For mixed data sources the selected datasource is on the query level.
For non mixed scenarios this is undefined.
TODO find a better way to do this ^ that's friendly to schema
TODO this shouldn't be unknown but DataSourceRef | null
### fn withFormat

```jsonnet
withFormat(value)
```

PARAMETERS:

* **value** (`integer`)
   - valid values: `0`, `1`, `2`


### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
### fn withQueryID

```jsonnet
withQueryID(value)
```

PARAMETERS:

* **value** (`string`)


### fn withQueryType

```jsonnet
withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

Specify the query flavor
TODO make this required and give it a default
### fn withRawSQL

```jsonnet
withRawSQL(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`


### fn withRefId

```jsonnet
withRefId(value)
```

PARAMETERS:

* **value** (`string`)

A unique identifier for the query within the list of targets.
In server side expressions, the refId is used as a variable name to identify results.
By default, the UI will assign A->Z; however setting meaningful names may be useful.
### fn withTable

```jsonnet
withTable(value)
```

PARAMETERS:

* **value** (`string`)


### obj connectionArgs


#### fn connectionArgs.withCatalog

```jsonnet
connectionArgs.withCatalog(value="__default")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"__default"`


#### fn connectionArgs.withDatabase

```jsonnet
connectionArgs.withDatabase(value="__default")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"__default"`


#### fn connectionArgs.withRegion

```jsonnet
connectionArgs.withRegion(value="__default")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"__default"`


#### fn connectionArgs.withResultReuseEnabled

```jsonnet
connectionArgs.withResultReuseEnabled(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn connectionArgs.withResultReuseMaxAgeInMinutes

```jsonnet
connectionArgs.withResultReuseMaxAgeInMinutes(value=60)
```

PARAMETERS:

* **value** (`number`)
   - default value: `60`


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