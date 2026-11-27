# bigquery

grafonnet.query.bigquery

## Subpackages

* [sql.columns](sql/columns/index.md)
* [sql.groupBy](sql/groupBy.md)

## Index

* [`fn withConvertToUTC(value=true)`](#fn-withconverttoutc)
* [`fn withDataset(value)`](#fn-withdataset)
* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withDatasourceMixin(value)`](#fn-withdatasourcemixin)
* [`fn withEditorMode(value)`](#fn-witheditormode)
* [`fn withFormat(value)`](#fn-withformat)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withLocation(value)`](#fn-withlocation)
* [`fn withPartitioned(value=true)`](#fn-withpartitioned)
* [`fn withPartitionedField(value)`](#fn-withpartitionedfield)
* [`fn withProject(value)`](#fn-withproject)
* [`fn withQueryPriority(value)`](#fn-withquerypriority)
* [`fn withQueryType(value)`](#fn-withquerytype)
* [`fn withRawQuery(value=true)`](#fn-withrawquery)
* [`fn withRawSql(value)`](#fn-withrawsql)
* [`fn withRefId(value)`](#fn-withrefid)
* [`fn withSharded(value=true)`](#fn-withsharded)
* [`fn withSql(value)`](#fn-withsql)
* [`fn withSqlMixin(value)`](#fn-withsqlmixin)
* [`fn withTable(value)`](#fn-withtable)
* [`fn withTimeShift(value)`](#fn-withtimeshift)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)
* [`obj sql`](#obj-sql)
  * [`fn withColumns(value)`](#fn-sqlwithcolumns)
  * [`fn withColumnsMixin(value)`](#fn-sqlwithcolumnsmixin)
  * [`fn withFrom(value)`](#fn-sqlwithfrom)
  * [`fn withGroupBy(value)`](#fn-sqlwithgroupby)
  * [`fn withGroupByMixin(value)`](#fn-sqlwithgroupbymixin)
  * [`fn withLimit(value)`](#fn-sqlwithlimit)
  * [`fn withOffset(value)`](#fn-sqlwithoffset)
  * [`fn withOrderBy(value)`](#fn-sqlwithorderby)
  * [`fn withOrderByDirection(value)`](#fn-sqlwithorderbydirection)
  * [`fn withOrderByMixin(value)`](#fn-sqlwithorderbymixin)
  * [`fn withWhereString(value)`](#fn-sqlwithwherestring)
  * [`obj orderBy`](#obj-sqlorderby)
    * [`fn withProperty(value)`](#fn-sqlorderbywithproperty)
    * [`fn withPropertyMixin(value)`](#fn-sqlorderbywithpropertymixin)
    * [`fn withType(value="property")`](#fn-sqlorderbywithtype)
    * [`fn withTypeMixin(value="property")`](#fn-sqlorderbywithtypemixin)
    * [`obj property`](#obj-sqlorderbyproperty)
      * [`fn withName(value)`](#fn-sqlorderbypropertywithname)
      * [`fn withType(value="string")`](#fn-sqlorderbypropertywithtype)
      * [`fn withTypeMixin(value="string")`](#fn-sqlorderbypropertywithtypemixin)
      * [`obj type`](#obj-sqlorderbypropertytype)
        * [`fn withQueryEditorPropertyType(value)`](#fn-sqlorderbypropertytypewithqueryeditorpropertytype)
    * [`obj type`](#obj-sqlorderbytype)
      * [`fn withQueryEditorExpressionType(value)`](#fn-sqlorderbytypewithqueryeditorexpressiontype)

## Fields

### fn withConvertToUTC

```jsonnet
withConvertToUTC(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


### fn withDataset

```jsonnet
withDataset(value)
```

PARAMETERS:

* **value** (`string`)


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
### fn withEditorMode

```jsonnet
withEditorMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"code"`, `"builder"`


### fn withFormat

```jsonnet
withFormat(value)
```

PARAMETERS:

* **value** (`integer`)
   - valid values: `0`, `1`


### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
### fn withLocation

```jsonnet
withLocation(value)
```

PARAMETERS:

* **value** (`string`)


### fn withPartitioned

```jsonnet
withPartitioned(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


### fn withPartitionedField

```jsonnet
withPartitionedField(value)
```

PARAMETERS:

* **value** (`string`)


### fn withProject

```jsonnet
withProject(value)
```

PARAMETERS:

* **value** (`string`)


### fn withQueryPriority

```jsonnet
withQueryPriority(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"INTERACTIVE"`, `"BATCH"`


### fn withQueryType

```jsonnet
withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

Specify the query flavor
TODO make this required and give it a default
### fn withRawQuery

```jsonnet
withRawQuery(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


### fn withRawSql

```jsonnet
withRawSql(value)
```

PARAMETERS:

* **value** (`string`)


### fn withRefId

```jsonnet
withRefId(value)
```

PARAMETERS:

* **value** (`string`)

A unique identifier for the query within the list of targets.
In server side expressions, the refId is used as a variable name to identify results.
By default, the UI will assign A->Z; however setting meaningful names may be useful.
### fn withSharded

```jsonnet
withSharded(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


### fn withSql

```jsonnet
withSql(value)
```

PARAMETERS:

* **value** (`object`)


### fn withSqlMixin

```jsonnet
withSqlMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withTable

```jsonnet
withTable(value)
```

PARAMETERS:

* **value** (`string`)


### fn withTimeShift

```jsonnet
withTimeShift(value)
```

PARAMETERS:

* **value** (`string`)


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
### obj sql


#### fn sql.withColumns

```jsonnet
sql.withColumns(value)
```

PARAMETERS:

* **value** (`array`)


#### fn sql.withColumnsMixin

```jsonnet
sql.withColumnsMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### fn sql.withFrom

```jsonnet
sql.withFrom(value)
```

PARAMETERS:

* **value** (`string`)


#### fn sql.withGroupBy

```jsonnet
sql.withGroupBy(value)
```

PARAMETERS:

* **value** (`array`)


#### fn sql.withGroupByMixin

```jsonnet
sql.withGroupByMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### fn sql.withLimit

```jsonnet
sql.withLimit(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn sql.withOffset

```jsonnet
sql.withOffset(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn sql.withOrderBy

```jsonnet
sql.withOrderBy(value)
```

PARAMETERS:

* **value** (`object`)


#### fn sql.withOrderByDirection

```jsonnet
sql.withOrderByDirection(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"ASC"`, `"DESC"`


#### fn sql.withOrderByMixin

```jsonnet
sql.withOrderByMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn sql.withWhereString

```jsonnet
sql.withWhereString(value)
```

PARAMETERS:

* **value** (`string`)

whereJsonTree?: _
#### obj sql.orderBy


##### fn sql.orderBy.withProperty

```jsonnet
sql.orderBy.withProperty(value)
```

PARAMETERS:

* **value** (`object`)


##### fn sql.orderBy.withPropertyMixin

```jsonnet
sql.orderBy.withPropertyMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn sql.orderBy.withType

```jsonnet
sql.orderBy.withType(value="property")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"property"`


##### fn sql.orderBy.withTypeMixin

```jsonnet
sql.orderBy.withTypeMixin(value="property")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"property"`


##### obj sql.orderBy.property


###### fn sql.orderBy.property.withName

```jsonnet
sql.orderBy.property.withName(value)
```

PARAMETERS:

* **value** (`string`)


###### fn sql.orderBy.property.withType

```jsonnet
sql.orderBy.property.withType(value="string")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"string"`


###### fn sql.orderBy.property.withTypeMixin

```jsonnet
sql.orderBy.property.withTypeMixin(value="string")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"string"`


###### obj sql.orderBy.property.type


####### fn sql.orderBy.property.type.withQueryEditorPropertyType

```jsonnet
sql.orderBy.property.type.withQueryEditorPropertyType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"string"`


##### obj sql.orderBy.type


###### fn sql.orderBy.type.withQueryEditorExpressionType

```jsonnet
sql.orderBy.type.withQueryEditorExpressionType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"property"`, `"operator"`, `"or"`, `"and"`, `"groupBy"`, `"function"`, `"functionParameter"`

