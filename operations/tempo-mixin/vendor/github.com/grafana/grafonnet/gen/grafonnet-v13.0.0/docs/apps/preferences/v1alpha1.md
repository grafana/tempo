# v1alpha1

grafonnet.apps.preferences.v1alpha1

## Index

* [`fn new(name)`](#fn-new)
* [`fn withApiVersion()`](#fn-withapiversion)
* [`fn withKind()`](#fn-withkind)
* [`fn withMetadata(value)`](#fn-withmetadata)
* [`fn withMetadataMixin(value)`](#fn-withmetadatamixin)
* [`fn withSpec(value)`](#fn-withspec)
* [`fn withSpecMixin(value)`](#fn-withspecmixin)
* [`obj metadata`](#obj-metadata)
  * [`fn withAnnotations(value)`](#fn-metadatawithannotations)
  * [`fn withAnnotationsMixin(value)`](#fn-metadatawithannotationsmixin)
  * [`fn withCreationTimestamp(value)`](#fn-metadatawithcreationtimestamp)
  * [`fn withDeletionTimestamp(value)`](#fn-metadatawithdeletiontimestamp)
  * [`fn withGeneration(value)`](#fn-metadatawithgeneration)
  * [`fn withLabels(value)`](#fn-metadatawithlabels)
  * [`fn withLabelsMixin(value)`](#fn-metadatawithlabelsmixin)
  * [`fn withName(value)`](#fn-metadatawithname)
  * [`fn withNamespace(value)`](#fn-metadatawithnamespace)
  * [`fn withResourceVersion(value)`](#fn-metadatawithresourceversion)
  * [`fn withUid(value)`](#fn-metadatawithuid)
  * [`fn withUpdateTimestamp(value)`](#fn-metadatawithupdatetimestamp)
* [`obj spec`](#obj-spec)
  * [`fn withCookiePreferences(value)`](#fn-specwithcookiepreferences)
  * [`fn withCookiePreferencesMixin(value)`](#fn-specwithcookiepreferencesmixin)
  * [`fn withHomeDashboardUID(value)`](#fn-specwithhomedashboarduid)
  * [`fn withLanguage(value)`](#fn-specwithlanguage)
  * [`fn withNavbar(value)`](#fn-specwithnavbar)
  * [`fn withNavbarMixin(value)`](#fn-specwithnavbarmixin)
  * [`fn withQueryHistory(value)`](#fn-specwithqueryhistory)
  * [`fn withQueryHistoryMixin(value)`](#fn-specwithqueryhistorymixin)
  * [`fn withRegionalFormat(value)`](#fn-specwithregionalformat)
  * [`fn withTheme(value)`](#fn-specwiththeme)
  * [`fn withTimezone(value)`](#fn-specwithtimezone)
  * [`fn withWeekStart(value)`](#fn-specwithweekstart)
  * [`obj cookiePreferences`](#obj-speccookiepreferences)
    * [`fn withAnalytics(value)`](#fn-speccookiepreferenceswithanalytics)
    * [`fn withAnalyticsMixin(value)`](#fn-speccookiepreferenceswithanalyticsmixin)
    * [`fn withFunctional(value)`](#fn-speccookiepreferenceswithfunctional)
    * [`fn withFunctionalMixin(value)`](#fn-speccookiepreferenceswithfunctionalmixin)
    * [`fn withPerformance(value)`](#fn-speccookiepreferenceswithperformance)
    * [`fn withPerformanceMixin(value)`](#fn-speccookiepreferenceswithperformancemixin)
  * [`obj navbar`](#obj-specnavbar)
    * [`fn withBookmarkUrls(value)`](#fn-specnavbarwithbookmarkurls)
    * [`fn withBookmarkUrlsMixin(value)`](#fn-specnavbarwithbookmarkurlsmixin)
  * [`obj queryHistory`](#obj-specqueryhistory)
    * [`fn withHomeTab(value)`](#fn-specqueryhistorywithhometab)

## Fields

### fn new

```jsonnet
new(name)
```

PARAMETERS:

* **name** (`string`)

Creates a new preferences.Preferences resource.
### fn withApiVersion

```jsonnet
withApiVersion()
```


set the resource's apiVersion
### fn withKind

```jsonnet
withKind()
```


set the resource's kind
### fn withMetadata

```jsonnet
withMetadata(value)
```

PARAMETERS:

* **value** (`object`)


### fn withMetadataMixin

```jsonnet
withMetadataMixin(value)
```

PARAMETERS:

* **value** (`object`)


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


### obj metadata


#### fn metadata.withAnnotations

```jsonnet
metadata.withAnnotations(value)
```

PARAMETERS:

* **value** (`object`)


#### fn metadata.withAnnotationsMixin

```jsonnet
metadata.withAnnotationsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn metadata.withCreationTimestamp

```jsonnet
metadata.withCreationTimestamp(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withDeletionTimestamp

```jsonnet
metadata.withDeletionTimestamp(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withGeneration

```jsonnet
metadata.withGeneration(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn metadata.withLabels

```jsonnet
metadata.withLabels(value)
```

PARAMETERS:

* **value** (`object`)


#### fn metadata.withLabelsMixin

```jsonnet
metadata.withLabelsMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn metadata.withName

```jsonnet
metadata.withName(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withNamespace

```jsonnet
metadata.withNamespace(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withResourceVersion

```jsonnet
metadata.withResourceVersion(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withUid

```jsonnet
metadata.withUid(value)
```

PARAMETERS:

* **value** (`string`)


#### fn metadata.withUpdateTimestamp

```jsonnet
metadata.withUpdateTimestamp(value)
```

PARAMETERS:

* **value** (`string`)


### obj spec


#### fn spec.withCookiePreferences

```jsonnet
spec.withCookiePreferences(value)
```

PARAMETERS:

* **value** (`object`)

Cookie preferences
#### fn spec.withCookiePreferencesMixin

```jsonnet
spec.withCookiePreferencesMixin(value)
```

PARAMETERS:

* **value** (`object`)

Cookie preferences
#### fn spec.withHomeDashboardUID

```jsonnet
spec.withHomeDashboardUID(value)
```

PARAMETERS:

* **value** (`string`)

UID for the home dashboard
#### fn spec.withLanguage

```jsonnet
spec.withLanguage(value)
```

PARAMETERS:

* **value** (`string`)

Selected language (beta)
#### fn spec.withNavbar

```jsonnet
spec.withNavbar(value)
```

PARAMETERS:

* **value** (`object`)

Navigation preferences
#### fn spec.withNavbarMixin

```jsonnet
spec.withNavbarMixin(value)
```

PARAMETERS:

* **value** (`object`)

Navigation preferences
#### fn spec.withQueryHistory

```jsonnet
spec.withQueryHistory(value)
```

PARAMETERS:

* **value** (`object`)

Explore query history preferences
#### fn spec.withQueryHistoryMixin

```jsonnet
spec.withQueryHistoryMixin(value)
```

PARAMETERS:

* **value** (`object`)

Explore query history preferences
#### fn spec.withRegionalFormat

```jsonnet
spec.withRegionalFormat(value)
```

PARAMETERS:

* **value** (`string`)

Selected locale (beta)
#### fn spec.withTheme

```jsonnet
spec.withTheme(value)
```

PARAMETERS:

* **value** (`string`)

light, dark, empty is default
#### fn spec.withTimezone

```jsonnet
spec.withTimezone(value)
```

PARAMETERS:

* **value** (`string`)

The timezone selection
TODO: this should use the timezone defined in common
#### fn spec.withWeekStart

```jsonnet
spec.withWeekStart(value)
```

PARAMETERS:

* **value** (`string`)

day of the week (sunday, monday, etc)
#### obj spec.cookiePreferences


##### fn spec.cookiePreferences.withAnalytics

```jsonnet
spec.cookiePreferences.withAnalytics(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.cookiePreferences.withAnalyticsMixin

```jsonnet
spec.cookiePreferences.withAnalyticsMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.cookiePreferences.withFunctional

```jsonnet
spec.cookiePreferences.withFunctional(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.cookiePreferences.withFunctionalMixin

```jsonnet
spec.cookiePreferences.withFunctionalMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.cookiePreferences.withPerformance

```jsonnet
spec.cookiePreferences.withPerformance(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.cookiePreferences.withPerformanceMixin

```jsonnet
spec.cookiePreferences.withPerformanceMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### obj spec.navbar


##### fn spec.navbar.withBookmarkUrls

```jsonnet
spec.navbar.withBookmarkUrls(value)
```

PARAMETERS:

* **value** (`array`)


##### fn spec.navbar.withBookmarkUrlsMixin

```jsonnet
spec.navbar.withBookmarkUrlsMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### obj spec.queryHistory


##### fn spec.queryHistory.withHomeTab

```jsonnet
spec.queryHistory.withHomeTab(value)
```

PARAMETERS:

* **value** (`string`)

one of: '' | 'query' | 'starred';