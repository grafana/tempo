# grafanaPyroscope

grafonnet.query.grafanaPyroscope

## Index

* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withGroupBy(value)`](#fn-withgroupby)
* [`fn withGroupByMixin(value)`](#fn-withgroupbymixin)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withLabelSelector(value="{}")`](#fn-withlabelselector)
* [`fn withLimit(value)`](#fn-withlimit)
* [`fn withMaxNodes(value)`](#fn-withmaxnodes)
* [`fn withProfileTypeId(value)`](#fn-withprofiletypeid)
* [`fn withQueryType(value)`](#fn-withquerytype)
* [`fn withRefId(value)`](#fn-withrefid)
* [`fn withSpanSelector(value)`](#fn-withspanselector)
* [`fn withSpanSelectorMixin(value)`](#fn-withspanselectormixin)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)

## Fields

### fn withDatasource

```jsonnet
withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### fn withGroupBy

```jsonnet
withGroupBy(value)
```

PARAMETERS:

* **value** (`array`)

Allows to group the results.
### fn withGroupByMixin

```jsonnet
withGroupByMixin(value)
```

PARAMETERS:

* **value** (`array`)

Allows to group the results.
### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
### fn withLabelSelector

```jsonnet
withLabelSelector(value="{}")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"{}"`

Specifies the query label selectors.
### fn withLimit

```jsonnet
withLimit(value)
```

PARAMETERS:

* **value** (`integer`)

Sets the maximum number of time series.
### fn withMaxNodes

```jsonnet
withMaxNodes(value)
```

PARAMETERS:

* **value** (`integer`)

Sets the maximum number of nodes in the flamegraph.
### fn withProfileTypeId

```jsonnet
withProfileTypeId(value)
```

PARAMETERS:

* **value** (`string`)

Specifies the type of profile to query.
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
### fn withSpanSelector

```jsonnet
withSpanSelector(value)
```

PARAMETERS:

* **value** (`array`)

Specifies the query span selectors.
### fn withSpanSelectorMixin

```jsonnet
withSpanSelectorMixin(value)
```

PARAMETERS:

* **value** (`array`)

Specifies the query span selectors.
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