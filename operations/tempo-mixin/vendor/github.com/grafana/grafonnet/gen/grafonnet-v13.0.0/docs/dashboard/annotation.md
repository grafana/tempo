# annotation



## Index

* [`fn withBuiltIn(value=0)`](#fn-withbuiltin)
* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withDatasourceMixin(value)`](#fn-withdatasourcemixin)
* [`fn withEnable(value=true)`](#fn-withenable)
* [`fn withExpr(value)`](#fn-withexpr)
* [`fn withFilter(value)`](#fn-withfilter)
* [`fn withFilterMixin(value)`](#fn-withfiltermixin)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withIconColor(value)`](#fn-withiconcolor)
* [`fn withMappings(value)`](#fn-withmappings)
* [`fn withMappingsMixin(value)`](#fn-withmappingsmixin)
* [`fn withName(value)`](#fn-withname)
* [`fn withPlacement(value="inControlsMenu")`](#fn-withplacement)
* [`fn withPlacementMixin(value="inControlsMenu")`](#fn-withplacementmixin)
* [`fn withStep(value)`](#fn-withstep)
* [`fn withTagKeys(value)`](#fn-withtagkeys)
* [`fn withTarget(value)`](#fn-withtarget)
* [`fn withTargetMixin(value)`](#fn-withtargetmixin)
* [`fn withTextFormat(value)`](#fn-withtextformat)
* [`fn withTitleFormat(value)`](#fn-withtitleformat)
* [`fn withType(value)`](#fn-withtype)
* [`fn withUseValueForTime(value=true)`](#fn-withusevaluefortime)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)
* [`obj filter`](#obj-filter)
  * [`fn withExclude(value=true)`](#fn-filterwithexclude)
  * [`fn withIds(value)`](#fn-filterwithids)
  * [`fn withIdsMixin(value)`](#fn-filterwithidsmixin)
* [`obj placement`](#obj-placement)
  * [`fn withAnnotationQueryPlacement(value)`](#fn-placementwithannotationqueryplacement)

## Fields

### fn withBuiltIn

```jsonnet
withBuiltIn(value=0)
```

PARAMETERS:

* **value** (`number`)
   - default value: `0`

Set to 1 for the standard annotation query all dashboards have by default.
### fn withDatasource

```jsonnet
withDatasource(value)
```

PARAMETERS:

* **value** (`object`)

Datasource where the annotations data is
### fn withDatasourceMixin

```jsonnet
withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)

Datasource where the annotations data is
### fn withEnable

```jsonnet
withEnable(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

When enabled the annotation query is issued with every dashboard refresh
### fn withExpr

```jsonnet
withExpr(value)
```

PARAMETERS:

* **value** (`string`)


### fn withFilter

```jsonnet
withFilter(value)
```

PARAMETERS:

* **value** (`object`)

Filters to apply when fetching annotations
### fn withFilterMixin

```jsonnet
withFilterMixin(value)
```

PARAMETERS:

* **value** (`object`)

Filters to apply when fetching annotations
### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Annotation queries can be toggled on or off at the top of the dashboard.
When hide is true, the toggle is not shown in the dashboard.
### fn withIconColor

```jsonnet
withIconColor(value)
```

PARAMETERS:

* **value** (`string`)

Color to use for the annotation event markers
### fn withMappings

```jsonnet
withMappings(value)
```

PARAMETERS:

* **value** (`object`)

Mappings define how to convert data frame fields to annotation event fields.
### fn withMappingsMixin

```jsonnet
withMappingsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Mappings define how to convert data frame fields to annotation event fields.
### fn withName

```jsonnet
withName(value)
```

PARAMETERS:

* **value** (`string`)

Name of annotation.
### fn withPlacement

```jsonnet
withPlacement(value="inControlsMenu")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"inControlsMenu"`

Placement can be used to display the annotation query somewhere else on the dashboard other than the default location.
### fn withPlacementMixin

```jsonnet
withPlacementMixin(value="inControlsMenu")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"inControlsMenu"`

Placement can be used to display the annotation query somewhere else on the dashboard other than the default location.
### fn withStep

```jsonnet
withStep(value)
```

PARAMETERS:

* **value** (`string`)

Legacy Prometheus annotation query step interval.
### fn withTagKeys

```jsonnet
withTagKeys(value)
```

PARAMETERS:

* **value** (`string`)

Comma-separated label keys used as annotation tags.
### fn withTarget

```jsonnet
withTarget(value)
```

PARAMETERS:

* **value** (`object`)

TODO.. this should just be a normal query target
### fn withTargetMixin

```jsonnet
withTargetMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO.. this should just be a normal query target
### fn withTextFormat

```jsonnet
withTextFormat(value)
```

PARAMETERS:

* **value** (`string`)

Format for Prometheus annotation text. Label values can be interpolated with templates like {{instance}}.
### fn withTitleFormat

```jsonnet
withTitleFormat(value)
```

PARAMETERS:

* **value** (`string`)

Format for Prometheus and Loki annotation titles. Label values can be interpolated with templates like {{instance}}.
### fn withType

```jsonnet
withType(value)
```

PARAMETERS:

* **value** (`string`)

TODO -- this should not exist here, it is based on the --grafana-- datasource
### fn withUseValueForTime

```jsonnet
withUseValueForTime(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Use the Prometheus series value as the annotation timestamp.
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
### obj filter


#### fn filter.withExclude

```jsonnet
filter.withExclude(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Should the specified panels be included or excluded
#### fn filter.withIds

```jsonnet
filter.withIds(value)
```

PARAMETERS:

* **value** (`array`)

Panel IDs that should be included or excluded
#### fn filter.withIdsMixin

```jsonnet
filter.withIdsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Panel IDs that should be included or excluded
### obj placement


#### fn placement.withAnnotationQueryPlacement

```jsonnet
placement.withAnnotationQueryPlacement(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"inControlsMenu"`

Annotation Query placement. Defines where the annotation query should be displayed.
- "inControlsMenu" renders the annotation query in the dashboard controls dropdown menu