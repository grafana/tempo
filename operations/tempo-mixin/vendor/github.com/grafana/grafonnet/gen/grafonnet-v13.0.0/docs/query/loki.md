# loki

grafonnet.query.loki

## Index

* [`fn new(datasource, expr)`](#fn-new)
* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withDirection(value)`](#fn-withdirection)
* [`fn withEditorMode(value)`](#fn-witheditormode)
* [`fn withExpr(value)`](#fn-withexpr)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withInstant(value=true)`](#fn-withinstant)
* [`fn withLegendFormat(value)`](#fn-withlegendformat)
* [`fn withMaxLines(value)`](#fn-withmaxlines)
* [`fn withQueryType(value)`](#fn-withquerytype)
* [`fn withRange(value=true)`](#fn-withrange)
* [`fn withRefId(value)`](#fn-withrefid)
* [`fn withResolution(value)`](#fn-withresolution)
* [`fn withStep(value)`](#fn-withstep)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)

## Fields

### fn new

```jsonnet
new(datasource, expr)
```

PARAMETERS:

* **datasource** (`string`)
* **expr** (`string`)

Creates a new loki query target for panels.
### fn withDatasource

```jsonnet
withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### fn withDirection

```jsonnet
withDirection(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"forward"`, `"backward"`, `"scan"`


### fn withEditorMode

```jsonnet
withEditorMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"code"`, `"builder"`


### fn withExpr

```jsonnet
withExpr(value)
```

PARAMETERS:

* **value** (`string`)

The LogQL query.
### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

If hide is set to true, Grafana will filter out the response(s) associated with this query before returning it to the panel.
### fn withInstant

```jsonnet
withInstant(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

@deprecated, now use queryType.
### fn withLegendFormat

```jsonnet
withLegendFormat(value)
```

PARAMETERS:

* **value** (`string`)

Used to override the name of the series.
### fn withMaxLines

```jsonnet
withMaxLines(value)
```

PARAMETERS:

* **value** (`integer`)

Used to limit the number of log rows returned.
### fn withQueryType

```jsonnet
withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

Specify the query flavor
TODO make this required and give it a default
### fn withRange

```jsonnet
withRange(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

@deprecated, now use queryType.
### fn withRefId

```jsonnet
withRefId(value)
```

PARAMETERS:

* **value** (`string`)

A unique identifier for the query within the list of targets.
In server side expressions, the refId is used as a variable name to identify results.
By default, the UI will assign A->Z; however setting meaningful names may be useful.
### fn withResolution

```jsonnet
withResolution(value)
```

PARAMETERS:

* **value** (`integer`)

@deprecated, now use step.
### fn withStep

```jsonnet
withStep(value)
```

PARAMETERS:

* **value** (`string`)

Used to set step value for range queries.
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