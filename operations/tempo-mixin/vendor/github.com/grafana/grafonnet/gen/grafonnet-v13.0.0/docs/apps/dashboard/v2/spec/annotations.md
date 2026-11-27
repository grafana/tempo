# annotations



## Index

* [`fn withKind()`](#fn-withkind)
* [`fn withSpec(value)`](#fn-withspec)
* [`fn withSpecMixin(value)`](#fn-withspecmixin)
* [`obj spec`](#obj-spec)
  * [`fn withBuiltIn(value=true)`](#fn-specwithbuiltin)
  * [`fn withEnable(value=true)`](#fn-specwithenable)
  * [`fn withFilter(value)`](#fn-specwithfilter)
  * [`fn withFilterMixin(value)`](#fn-specwithfiltermixin)
  * [`fn withHide(value=true)`](#fn-specwithhide)
  * [`fn withIconColor(value)`](#fn-specwithiconcolor)
  * [`fn withLegacyOptions(value)`](#fn-specwithlegacyoptions)
  * [`fn withLegacyOptionsMixin(value)`](#fn-specwithlegacyoptionsmixin)
  * [`fn withMappings(value)`](#fn-specwithmappings)
  * [`fn withMappingsMixin(value)`](#fn-specwithmappingsmixin)
  * [`fn withName(value)`](#fn-specwithname)
  * [`fn withPlacement(value)`](#fn-specwithplacement)
  * [`fn withPlacementMixin(value)`](#fn-specwithplacementmixin)
  * [`fn withQuery(value)`](#fn-specwithquery)
  * [`fn withQueryMixin(value)`](#fn-specwithquerymixin)
  * [`obj filter`](#obj-specfilter)
    * [`fn withExclude(value=true)`](#fn-specfilterwithexclude)
    * [`fn withIds(value)`](#fn-specfilterwithids)
    * [`fn withIdsMixin(value)`](#fn-specfilterwithidsmixin)
  * [`obj placement`](#obj-specplacement)
    * [`fn withAnnotationQueryPlacement(value)`](#fn-specplacementwithannotationqueryplacement)
  * [`obj query`](#obj-specquery)
    * [`fn withDatasource(value)`](#fn-specquerywithdatasource)
    * [`fn withDatasourceMixin(value)`](#fn-specquerywithdatasourcemixin)
    * [`fn withGroup(value)`](#fn-specquerywithgroup)
    * [`fn withKind()`](#fn-specquerywithkind)
    * [`fn withLabels(value)`](#fn-specquerywithlabels)
    * [`fn withLabelsMixin(value)`](#fn-specquerywithlabelsmixin)
    * [`fn withSpec(value)`](#fn-specquerywithspec)
    * [`fn withSpecMixin(value)`](#fn-specquerywithspecmixin)
    * [`fn withVersion(value="v0")`](#fn-specquerywithversion)
    * [`obj datasource`](#obj-specquerydatasource)
      * [`fn withName(value)`](#fn-specquerydatasourcewithname)

## Fields

### fn withKind

```jsonnet
withKind()
```



### fn withSpec

```jsonnet
withSpec(value)
```

PARAMETERS:

* **value** (`object`)


### fn withSpecMixin

```jsonnet
withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


### obj spec


#### fn spec.withBuiltIn

```jsonnet
spec.withBuiltIn(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn spec.withEnable

```jsonnet
spec.withEnable(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn spec.withFilter

```jsonnet
spec.withFilter(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withFilterMixin

```jsonnet
spec.withFilterMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withHide

```jsonnet
spec.withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


#### fn spec.withIconColor

```jsonnet
spec.withIconColor(value)
```

PARAMETERS:

* **value** (`string`)


#### fn spec.withLegacyOptions

```jsonnet
spec.withLegacyOptions(value)
```

PARAMETERS:

* **value** (`object`)

Catch-all field for datasource-specific properties. Should not be available in as code tooling.
#### fn spec.withLegacyOptionsMixin

```jsonnet
spec.withLegacyOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Catch-all field for datasource-specific properties. Should not be available in as code tooling.
#### fn spec.withMappings

```jsonnet
spec.withMappings(value)
```

PARAMETERS:

* **value** (`object`)

Mappings define how to convert data frame fields to annotation event fields.
#### fn spec.withMappingsMixin

```jsonnet
spec.withMappingsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Mappings define how to convert data frame fields to annotation event fields.
#### fn spec.withName

```jsonnet
spec.withName(value)
```

PARAMETERS:

* **value** (`string`)


#### fn spec.withPlacement

```jsonnet
spec.withPlacement(value)
```

PARAMETERS:

* **value** (`string`)

Placement can be used to display the annotation query somewhere else on the dashboard other than the default location.
#### fn spec.withPlacementMixin

```jsonnet
spec.withPlacementMixin(value)
```

PARAMETERS:

* **value** (`string`)

Placement can be used to display the annotation query somewhere else on the dashboard other than the default location.
#### fn spec.withQuery

```jsonnet
spec.withQuery(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withQueryMixin

```jsonnet
spec.withQueryMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### obj spec.filter


##### fn spec.filter.withExclude

```jsonnet
spec.filter.withExclude(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Should the specified panels be included or excluded
##### fn spec.filter.withIds

```jsonnet
spec.filter.withIds(value)
```

PARAMETERS:

* **value** (`array`)

Panel IDs that should be included or excluded
##### fn spec.filter.withIdsMixin

```jsonnet
spec.filter.withIdsMixin(value)
```

PARAMETERS:

* **value** (`array`)

Panel IDs that should be included or excluded
#### obj spec.placement


##### fn spec.placement.withAnnotationQueryPlacement

```jsonnet
spec.placement.withAnnotationQueryPlacement(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"inControlsMenu"`

Annotation Query placement. Defines where the annotation query should be displayed.
- "inControlsMenu" renders the annotation query in the dashboard controls dropdown menu
#### obj spec.query


##### fn spec.query.withDatasource

```jsonnet
spec.query.withDatasource(value)
```

PARAMETERS:

* **value** (`object`)

New type for datasource reference
Not creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.
##### fn spec.query.withDatasourceMixin

```jsonnet
spec.query.withDatasourceMixin(value)
```

PARAMETERS:

* **value** (`object`)

New type for datasource reference
Not creating a new type until we figure out how to handle DS refs for group by, adhoc, and every place that uses DataSourceRef in TS.
##### fn spec.query.withGroup

```jsonnet
spec.query.withGroup(value)
```

PARAMETERS:

* **value** (`string`)


##### fn spec.query.withKind

```jsonnet
spec.query.withKind()
```



##### fn spec.query.withLabels

```jsonnet
spec.query.withLabels(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.query.withLabelsMixin

```jsonnet
spec.query.withLabelsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.query.withSpec

```jsonnet
spec.query.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.query.withSpecMixin

```jsonnet
spec.query.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.query.withVersion

```jsonnet
spec.query.withVersion(value="v0")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"v0"`


##### obj spec.query.datasource


###### fn spec.query.datasource.withName

```jsonnet
spec.query.datasource.withName(value)
```

PARAMETERS:

* **value** (`string`)

