# items



## Index

* [`fn withKind()`](#fn-withkind)
* [`fn withSpec(value)`](#fn-withspec)
* [`fn withSpecMixin(value)`](#fn-withspecmixin)
* [`obj spec`](#obj-spec)
  * [`fn withElement(value)`](#fn-specwithelement)
  * [`fn withElementMixin(value)`](#fn-specwithelementmixin)
  * [`fn withHeight(value)`](#fn-specwithheight)
  * [`fn withRepeat(value)`](#fn-specwithrepeat)
  * [`fn withRepeatMixin(value)`](#fn-specwithrepeatmixin)
  * [`fn withWidth(value)`](#fn-specwithwidth)
  * [`fn withX(value)`](#fn-specwithx)
  * [`fn withY(value)`](#fn-specwithy)
  * [`obj element`](#obj-specelement)
    * [`fn withKind()`](#fn-specelementwithkind)
    * [`fn withName(value)`](#fn-specelementwithname)
  * [`obj repeat`](#obj-specrepeat)
    * [`fn withDirection(value)`](#fn-specrepeatwithdirection)
    * [`fn withMaxPerRow(value)`](#fn-specrepeatwithmaxperrow)
    * [`fn withMode(value="variable")`](#fn-specrepeatwithmode)
    * [`fn withModeMixin(value="variable")`](#fn-specrepeatwithmodemixin)
    * [`fn withValue(value)`](#fn-specrepeatwithvalue)
    * [`obj mode`](#obj-specrepeatmode)
      * [`fn withRepeatMode(value)`](#fn-specrepeatmodewithrepeatmode)

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


#### fn spec.withElement

```jsonnet
spec.withElement(value)
```

PARAMETERS:

* **value** (`object`)

reference to a PanelKind from dashboard.spec.elements Expressed as JSON Schema reference
#### fn spec.withElementMixin

```jsonnet
spec.withElementMixin(value)
```

PARAMETERS:

* **value** (`object`)

reference to a PanelKind from dashboard.spec.elements Expressed as JSON Schema reference
#### fn spec.withHeight

```jsonnet
spec.withHeight(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn spec.withRepeat

```jsonnet
spec.withRepeat(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withRepeatMixin

```jsonnet
spec.withRepeatMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withWidth

```jsonnet
spec.withWidth(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn spec.withX

```jsonnet
spec.withX(value)
```

PARAMETERS:

* **value** (`integer`)


#### fn spec.withY

```jsonnet
spec.withY(value)
```

PARAMETERS:

* **value** (`integer`)


#### obj spec.element


##### fn spec.element.withKind

```jsonnet
spec.element.withKind()
```



##### fn spec.element.withName

```jsonnet
spec.element.withName(value)
```

PARAMETERS:

* **value** (`string`)


#### obj spec.repeat


##### fn spec.repeat.withDirection

```jsonnet
spec.repeat.withDirection(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"h"`, `"v"`


##### fn spec.repeat.withMaxPerRow

```jsonnet
spec.repeat.withMaxPerRow(value)
```

PARAMETERS:

* **value** (`integer`)


##### fn spec.repeat.withMode

```jsonnet
spec.repeat.withMode(value="variable")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"variable"`


##### fn spec.repeat.withModeMixin

```jsonnet
spec.repeat.withModeMixin(value="variable")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"variable"`


##### fn spec.repeat.withValue

```jsonnet
spec.repeat.withValue(value)
```

PARAMETERS:

* **value** (`string`)


##### obj spec.repeat.mode


###### fn spec.repeat.mode.withRepeatMode

```jsonnet
spec.repeat.mode.withRepeatMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"variable"`

other repeat modes will be added in the future: label, frame