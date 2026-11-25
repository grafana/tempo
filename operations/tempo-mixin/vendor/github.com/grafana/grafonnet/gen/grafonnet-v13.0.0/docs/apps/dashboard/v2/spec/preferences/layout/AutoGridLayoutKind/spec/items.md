# items



## Index

* [`fn withKind()`](#fn-withkind)
* [`fn withSpec(value)`](#fn-withspec)
* [`fn withSpecMixin(value)`](#fn-withspecmixin)
* [`obj spec`](#obj-spec)
  * [`fn withConditionalRendering(value)`](#fn-specwithconditionalrendering)
  * [`fn withConditionalRenderingMixin(value)`](#fn-specwithconditionalrenderingmixin)
  * [`fn withElement(value)`](#fn-specwithelement)
  * [`fn withElementMixin(value)`](#fn-specwithelementmixin)
  * [`fn withRepeat(value)`](#fn-specwithrepeat)
  * [`fn withRepeatMixin(value)`](#fn-specwithrepeatmixin)
  * [`obj conditionalRendering`](#obj-specconditionalrendering)
    * [`fn withKind()`](#fn-specconditionalrenderingwithkind)
    * [`fn withSpec(value)`](#fn-specconditionalrenderingwithspec)
    * [`fn withSpecMixin(value)`](#fn-specconditionalrenderingwithspecmixin)
    * [`obj spec`](#obj-specconditionalrenderingspec)
      * [`fn withCondition(value)`](#fn-specconditionalrenderingspecwithcondition)
      * [`fn withItems(value)`](#fn-specconditionalrenderingspecwithitems)
      * [`fn withItemsMixin(value)`](#fn-specconditionalrenderingspecwithitemsmixin)
      * [`fn withVisibility(value)`](#fn-specconditionalrenderingspecwithvisibility)
  * [`obj element`](#obj-specelement)
    * [`fn withKind()`](#fn-specelementwithkind)
    * [`fn withName(value)`](#fn-specelementwithname)
  * [`obj repeat`](#obj-specrepeat)
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


#### fn spec.withConditionalRendering

```jsonnet
spec.withConditionalRendering(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withConditionalRenderingMixin

```jsonnet
spec.withConditionalRenderingMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withElement

```jsonnet
spec.withElement(value)
```

PARAMETERS:

* **value** (`object`)


#### fn spec.withElementMixin

```jsonnet
spec.withElementMixin(value)
```

PARAMETERS:

* **value** (`object`)


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


#### obj spec.conditionalRendering


##### fn spec.conditionalRendering.withKind

```jsonnet
spec.conditionalRendering.withKind()
```



##### fn spec.conditionalRendering.withSpec

```jsonnet
spec.conditionalRendering.withSpec(value)
```

PARAMETERS:

* **value** (`object`)


##### fn spec.conditionalRendering.withSpecMixin

```jsonnet
spec.conditionalRendering.withSpecMixin(value)
```

PARAMETERS:

* **value** (`object`)


##### obj spec.conditionalRendering.spec


###### fn spec.conditionalRendering.spec.withCondition

```jsonnet
spec.conditionalRendering.spec.withCondition(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"and"`, `"or"`


###### fn spec.conditionalRendering.spec.withItems

```jsonnet
spec.conditionalRendering.spec.withItems(value)
```

PARAMETERS:

* **value** (`array`)


###### fn spec.conditionalRendering.spec.withItemsMixin

```jsonnet
spec.conditionalRendering.spec.withItemsMixin(value)
```

PARAMETERS:

* **value** (`array`)


###### fn spec.conditionalRendering.spec.withVisibility

```jsonnet
spec.conditionalRendering.spec.withVisibility(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"show"`, `"hide"`


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