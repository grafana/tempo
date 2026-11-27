# bucketAggs



## Subpackages

* [Filters.settings.filters](Filters/settings/filters.md)

## Index

* [`obj DateHistogram`](#obj-datehistogram)
  * [`fn withField(value)`](#fn-datehistogramwithfield)
  * [`fn withId(value)`](#fn-datehistogramwithid)
  * [`fn withSettings(value)`](#fn-datehistogramwithsettings)
  * [`fn withSettingsMixin(value)`](#fn-datehistogramwithsettingsmixin)
  * [`fn withType(value="date_histogram")`](#fn-datehistogramwithtype)
  * [`fn withTypeMixin(value="date_histogram")`](#fn-datehistogramwithtypemixin)
  * [`obj settings`](#obj-datehistogramsettings)
    * [`fn withInterval(value)`](#fn-datehistogramsettingswithinterval)
    * [`fn withMinDocCount(value)`](#fn-datehistogramsettingswithmindoccount)
    * [`fn withOffset(value)`](#fn-datehistogramsettingswithoffset)
    * [`fn withTimeZone(value)`](#fn-datehistogramsettingswithtimezone)
    * [`fn withTrimEdges(value)`](#fn-datehistogramsettingswithtrimedges)
  * [`obj type`](#obj-datehistogramtype)
    * [`fn withBucketAggregationType(value)`](#fn-datehistogramtypewithbucketaggregationtype)
* [`obj Filters`](#obj-filters)
  * [`fn withId(value)`](#fn-filterswithid)
  * [`fn withSettings(value)`](#fn-filterswithsettings)
  * [`fn withSettingsMixin(value)`](#fn-filterswithsettingsmixin)
  * [`fn withType(value="filters")`](#fn-filterswithtype)
  * [`fn withTypeMixin(value="filters")`](#fn-filterswithtypemixin)
  * [`obj settings`](#obj-filterssettings)
    * [`fn withFilters(value)`](#fn-filterssettingswithfilters)
    * [`fn withFiltersMixin(value)`](#fn-filterssettingswithfiltersmixin)
  * [`obj type`](#obj-filterstype)
    * [`fn withBucketAggregationType(value)`](#fn-filterstypewithbucketaggregationtype)
* [`obj GeoHashGrid`](#obj-geohashgrid)
  * [`fn withField(value)`](#fn-geohashgridwithfield)
  * [`fn withId(value)`](#fn-geohashgridwithid)
  * [`fn withSettings(value)`](#fn-geohashgridwithsettings)
  * [`fn withSettingsMixin(value)`](#fn-geohashgridwithsettingsmixin)
  * [`fn withType(value="geohash_grid")`](#fn-geohashgridwithtype)
  * [`fn withTypeMixin(value="geohash_grid")`](#fn-geohashgridwithtypemixin)
  * [`obj settings`](#obj-geohashgridsettings)
    * [`fn withPrecision(value)`](#fn-geohashgridsettingswithprecision)
  * [`obj type`](#obj-geohashgridtype)
    * [`fn withBucketAggregationType(value)`](#fn-geohashgridtypewithbucketaggregationtype)
* [`obj Histogram`](#obj-histogram)
  * [`fn withField(value)`](#fn-histogramwithfield)
  * [`fn withId(value)`](#fn-histogramwithid)
  * [`fn withSettings(value)`](#fn-histogramwithsettings)
  * [`fn withSettingsMixin(value)`](#fn-histogramwithsettingsmixin)
  * [`fn withType(value="histogram")`](#fn-histogramwithtype)
  * [`fn withTypeMixin(value="histogram")`](#fn-histogramwithtypemixin)
  * [`obj settings`](#obj-histogramsettings)
    * [`fn withInterval(value)`](#fn-histogramsettingswithinterval)
    * [`fn withMinDocCount(value)`](#fn-histogramsettingswithmindoccount)
  * [`obj type`](#obj-histogramtype)
    * [`fn withBucketAggregationType(value)`](#fn-histogramtypewithbucketaggregationtype)
* [`obj Nested`](#obj-nested)
  * [`fn withField(value)`](#fn-nestedwithfield)
  * [`fn withId(value)`](#fn-nestedwithid)
  * [`fn withSettings(value)`](#fn-nestedwithsettings)
  * [`fn withSettingsMixin(value)`](#fn-nestedwithsettingsmixin)
  * [`fn withType(value="nested")`](#fn-nestedwithtype)
  * [`fn withTypeMixin(value="nested")`](#fn-nestedwithtypemixin)
  * [`obj type`](#obj-nestedtype)
    * [`fn withBucketAggregationType(value)`](#fn-nestedtypewithbucketaggregationtype)
* [`obj Terms`](#obj-terms)
  * [`fn withField(value)`](#fn-termswithfield)
  * [`fn withId(value)`](#fn-termswithid)
  * [`fn withSettings(value)`](#fn-termswithsettings)
  * [`fn withSettingsMixin(value)`](#fn-termswithsettingsmixin)
  * [`fn withType(value="terms")`](#fn-termswithtype)
  * [`fn withTypeMixin(value="terms")`](#fn-termswithtypemixin)
  * [`obj settings`](#obj-termssettings)
    * [`fn withMinDocCount(value)`](#fn-termssettingswithmindoccount)
    * [`fn withMissing(value)`](#fn-termssettingswithmissing)
    * [`fn withOrder(value)`](#fn-termssettingswithorder)
    * [`fn withOrderBy(value)`](#fn-termssettingswithorderby)
    * [`fn withSize(value)`](#fn-termssettingswithsize)
  * [`obj type`](#obj-termstype)
    * [`fn withBucketAggregationType(value)`](#fn-termstypewithbucketaggregationtype)

## Fields

### obj DateHistogram


#### fn DateHistogram.withField

```jsonnet
DateHistogram.withField(value)
```

PARAMETERS:

* **value** (`string`)


#### fn DateHistogram.withId

```jsonnet
DateHistogram.withId(value)
```

PARAMETERS:

* **value** (`string`)


#### fn DateHistogram.withSettings

```jsonnet
DateHistogram.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


#### fn DateHistogram.withSettingsMixin

```jsonnet
DateHistogram.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn DateHistogram.withType

```jsonnet
DateHistogram.withType(value="date_histogram")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"date_histogram"`


#### fn DateHistogram.withTypeMixin

```jsonnet
DateHistogram.withTypeMixin(value="date_histogram")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"date_histogram"`


#### obj DateHistogram.settings


##### fn DateHistogram.settings.withInterval

```jsonnet
DateHistogram.settings.withInterval(value)
```

PARAMETERS:

* **value** (`string`)


##### fn DateHistogram.settings.withMinDocCount

```jsonnet
DateHistogram.settings.withMinDocCount(value)
```

PARAMETERS:

* **value** (`string`)


##### fn DateHistogram.settings.withOffset

```jsonnet
DateHistogram.settings.withOffset(value)
```

PARAMETERS:

* **value** (`string`)


##### fn DateHistogram.settings.withTimeZone

```jsonnet
DateHistogram.settings.withTimeZone(value)
```

PARAMETERS:

* **value** (`string`)


##### fn DateHistogram.settings.withTrimEdges

```jsonnet
DateHistogram.settings.withTrimEdges(value)
```

PARAMETERS:

* **value** (`string`)


#### obj DateHistogram.type


##### fn DateHistogram.type.withBucketAggregationType

```jsonnet
DateHistogram.type.withBucketAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"terms"`, `"filters"`, `"geohash_grid"`, `"date_histogram"`, `"histogram"`, `"nested"`


### obj Filters


#### fn Filters.withId

```jsonnet
Filters.withId(value)
```

PARAMETERS:

* **value** (`string`)


#### fn Filters.withSettings

```jsonnet
Filters.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


#### fn Filters.withSettingsMixin

```jsonnet
Filters.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn Filters.withType

```jsonnet
Filters.withType(value="filters")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"filters"`


#### fn Filters.withTypeMixin

```jsonnet
Filters.withTypeMixin(value="filters")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"filters"`


#### obj Filters.settings


##### fn Filters.settings.withFilters

```jsonnet
Filters.settings.withFilters(value)
```

PARAMETERS:

* **value** (`array`)


##### fn Filters.settings.withFiltersMixin

```jsonnet
Filters.settings.withFiltersMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### obj Filters.type


##### fn Filters.type.withBucketAggregationType

```jsonnet
Filters.type.withBucketAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"terms"`, `"filters"`, `"geohash_grid"`, `"date_histogram"`, `"histogram"`, `"nested"`


### obj GeoHashGrid


#### fn GeoHashGrid.withField

```jsonnet
GeoHashGrid.withField(value)
```

PARAMETERS:

* **value** (`string`)


#### fn GeoHashGrid.withId

```jsonnet
GeoHashGrid.withId(value)
```

PARAMETERS:

* **value** (`string`)


#### fn GeoHashGrid.withSettings

```jsonnet
GeoHashGrid.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


#### fn GeoHashGrid.withSettingsMixin

```jsonnet
GeoHashGrid.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn GeoHashGrid.withType

```jsonnet
GeoHashGrid.withType(value="geohash_grid")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"geohash_grid"`


#### fn GeoHashGrid.withTypeMixin

```jsonnet
GeoHashGrid.withTypeMixin(value="geohash_grid")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"geohash_grid"`


#### obj GeoHashGrid.settings


##### fn GeoHashGrid.settings.withPrecision

```jsonnet
GeoHashGrid.settings.withPrecision(value)
```

PARAMETERS:

* **value** (`string`)


#### obj GeoHashGrid.type


##### fn GeoHashGrid.type.withBucketAggregationType

```jsonnet
GeoHashGrid.type.withBucketAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"terms"`, `"filters"`, `"geohash_grid"`, `"date_histogram"`, `"histogram"`, `"nested"`


### obj Histogram


#### fn Histogram.withField

```jsonnet
Histogram.withField(value)
```

PARAMETERS:

* **value** (`string`)


#### fn Histogram.withId

```jsonnet
Histogram.withId(value)
```

PARAMETERS:

* **value** (`string`)


#### fn Histogram.withSettings

```jsonnet
Histogram.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


#### fn Histogram.withSettingsMixin

```jsonnet
Histogram.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn Histogram.withType

```jsonnet
Histogram.withType(value="histogram")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"histogram"`


#### fn Histogram.withTypeMixin

```jsonnet
Histogram.withTypeMixin(value="histogram")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"histogram"`


#### obj Histogram.settings


##### fn Histogram.settings.withInterval

```jsonnet
Histogram.settings.withInterval(value)
```

PARAMETERS:

* **value** (`string`)


##### fn Histogram.settings.withMinDocCount

```jsonnet
Histogram.settings.withMinDocCount(value)
```

PARAMETERS:

* **value** (`string`)


#### obj Histogram.type


##### fn Histogram.type.withBucketAggregationType

```jsonnet
Histogram.type.withBucketAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"terms"`, `"filters"`, `"geohash_grid"`, `"date_histogram"`, `"histogram"`, `"nested"`


### obj Nested


#### fn Nested.withField

```jsonnet
Nested.withField(value)
```

PARAMETERS:

* **value** (`string`)


#### fn Nested.withId

```jsonnet
Nested.withId(value)
```

PARAMETERS:

* **value** (`string`)


#### fn Nested.withSettings

```jsonnet
Nested.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


#### fn Nested.withSettingsMixin

```jsonnet
Nested.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn Nested.withType

```jsonnet
Nested.withType(value="nested")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"nested"`


#### fn Nested.withTypeMixin

```jsonnet
Nested.withTypeMixin(value="nested")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"nested"`


#### obj Nested.type


##### fn Nested.type.withBucketAggregationType

```jsonnet
Nested.type.withBucketAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"terms"`, `"filters"`, `"geohash_grid"`, `"date_histogram"`, `"histogram"`, `"nested"`


### obj Terms


#### fn Terms.withField

```jsonnet
Terms.withField(value)
```

PARAMETERS:

* **value** (`string`)


#### fn Terms.withId

```jsonnet
Terms.withId(value)
```

PARAMETERS:

* **value** (`string`)


#### fn Terms.withSettings

```jsonnet
Terms.withSettings(value)
```

PARAMETERS:

* **value** (`object`)


#### fn Terms.withSettingsMixin

```jsonnet
Terms.withSettingsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn Terms.withType

```jsonnet
Terms.withType(value="terms")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"terms"`


#### fn Terms.withTypeMixin

```jsonnet
Terms.withTypeMixin(value="terms")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"terms"`


#### obj Terms.settings


##### fn Terms.settings.withMinDocCount

```jsonnet
Terms.settings.withMinDocCount(value)
```

PARAMETERS:

* **value** (`string`)


##### fn Terms.settings.withMissing

```jsonnet
Terms.settings.withMissing(value)
```

PARAMETERS:

* **value** (`string`)


##### fn Terms.settings.withOrder

```jsonnet
Terms.settings.withOrder(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"desc"`, `"asc"`


##### fn Terms.settings.withOrderBy

```jsonnet
Terms.settings.withOrderBy(value)
```

PARAMETERS:

* **value** (`string`)


##### fn Terms.settings.withSize

```jsonnet
Terms.settings.withSize(value)
```

PARAMETERS:

* **value** (`string`)


#### obj Terms.type


##### fn Terms.type.withBucketAggregationType

```jsonnet
Terms.type.withBucketAggregationType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"terms"`, `"filters"`, `"geohash_grid"`, `"date_histogram"`, `"histogram"`, `"nested"`

