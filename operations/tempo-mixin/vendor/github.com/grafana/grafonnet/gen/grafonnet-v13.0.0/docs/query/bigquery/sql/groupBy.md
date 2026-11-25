# groupBy



## Index

* [`fn withProperty(value)`](#fn-withproperty)
* [`fn withPropertyMixin(value)`](#fn-withpropertymixin)
* [`fn withType(value="groupBy")`](#fn-withtype)
* [`fn withTypeMixin(value="groupBy")`](#fn-withtypemixin)
* [`obj property`](#obj-property)
  * [`fn withName(value)`](#fn-propertywithname)
  * [`fn withType(value="string")`](#fn-propertywithtype)
  * [`fn withTypeMixin(value="string")`](#fn-propertywithtypemixin)
  * [`obj type`](#obj-propertytype)
    * [`fn withQueryEditorPropertyType(value)`](#fn-propertytypewithqueryeditorpropertytype)
* [`obj type`](#obj-type)
  * [`fn withQueryEditorExpressionType(value)`](#fn-typewithqueryeditorexpressiontype)

## Fields

### fn withProperty

```jsonnet
withProperty(value)
```

PARAMETERS:

* **value** (`object`)


### fn withPropertyMixin

```jsonnet
withPropertyMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withType

```jsonnet
withType(value="groupBy")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"groupBy"`


### fn withTypeMixin

```jsonnet
withTypeMixin(value="groupBy")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"groupBy"`


### obj property


#### fn property.withName

```jsonnet
property.withName(value)
```

PARAMETERS:

* **value** (`string`)


#### fn property.withType

```jsonnet
property.withType(value="string")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"string"`


#### fn property.withTypeMixin

```jsonnet
property.withTypeMixin(value="string")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"string"`


#### obj property.type


##### fn property.type.withQueryEditorPropertyType

```jsonnet
property.type.withQueryEditorPropertyType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"string"`


### obj type


#### fn type.withQueryEditorExpressionType

```jsonnet
type.withQueryEditorExpressionType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"property"`, `"operator"`, `"or"`, `"and"`, `"groupBy"`, `"function"`, `"functionParameter"`

