# v1

grafonnet.apps.playlist.v1

## Subpackages

* [spec.items](spec/items.md)

## Index

* [`fn new(name, title)`](#fn-new)
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
  * [`fn withInterval(value)`](#fn-specwithinterval)
  * [`fn withItems(value)`](#fn-specwithitems)
  * [`fn withItemsMixin(value)`](#fn-specwithitemsmixin)
  * [`fn withTitle(value)`](#fn-specwithtitle)

## Fields

### fn new

```jsonnet
new(name, title)
```

PARAMETERS:

* **name** (`string`)
* **title** (`string`)

Creates a new playlist.Playlist resource.
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


#### fn spec.withInterval

```jsonnet
spec.withInterval(value)
```

PARAMETERS:

* **value** (`string`)


#### fn spec.withItems

```jsonnet
spec.withItems(value)
```

PARAMETERS:

* **value** (`array`)


#### fn spec.withItemsMixin

```jsonnet
spec.withItemsMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### fn spec.withTitle

```jsonnet
spec.withTitle(value)
```

PARAMETERS:

* **value** (`string`)

