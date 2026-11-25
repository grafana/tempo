# elements



## Subpackages

* [connections](connections/index.md)

## Index

* [`fn withBackground(value)`](#fn-withbackground)
* [`fn withBackgroundMixin(value)`](#fn-withbackgroundmixin)
* [`fn withBorder(value)`](#fn-withborder)
* [`fn withBorderMixin(value)`](#fn-withbordermixin)
* [`fn withConfig(value)`](#fn-withconfig)
* [`fn withConfigMixin(value)`](#fn-withconfigmixin)
* [`fn withConnections(value)`](#fn-withconnections)
* [`fn withConnectionsMixin(value)`](#fn-withconnectionsmixin)
* [`fn withConstraint(value)`](#fn-withconstraint)
* [`fn withConstraintMixin(value)`](#fn-withconstraintmixin)
* [`fn withName(value)`](#fn-withname)
* [`fn withPlacement(value)`](#fn-withplacement)
* [`fn withPlacementMixin(value)`](#fn-withplacementmixin)
* [`fn withType(value)`](#fn-withtype)
* [`obj background`](#obj-background)
  * [`fn withColor(value)`](#fn-backgroundwithcolor)
  * [`fn withColorMixin(value)`](#fn-backgroundwithcolormixin)
  * [`fn withImage(value)`](#fn-backgroundwithimage)
  * [`fn withImageMixin(value)`](#fn-backgroundwithimagemixin)
  * [`fn withSize(value)`](#fn-backgroundwithsize)
  * [`obj color`](#obj-backgroundcolor)
    * [`fn withField(value)`](#fn-backgroundcolorwithfield)
    * [`fn withFixed(value)`](#fn-backgroundcolorwithfixed)
  * [`obj image`](#obj-backgroundimage)
    * [`fn withField(value)`](#fn-backgroundimagewithfield)
    * [`fn withFixed(value)`](#fn-backgroundimagewithfixed)
    * [`fn withMode(value)`](#fn-backgroundimagewithmode)
* [`obj border`](#obj-border)
  * [`fn withColor(value)`](#fn-borderwithcolor)
  * [`fn withColorMixin(value)`](#fn-borderwithcolormixin)
  * [`fn withRadius(value)`](#fn-borderwithradius)
  * [`fn withWidth(value)`](#fn-borderwithwidth)
  * [`obj color`](#obj-bordercolor)
    * [`fn withField(value)`](#fn-bordercolorwithfield)
    * [`fn withFixed(value)`](#fn-bordercolorwithfixed)
* [`obj constraint`](#obj-constraint)
  * [`fn withHorizontal(value)`](#fn-constraintwithhorizontal)
  * [`fn withVertical(value)`](#fn-constraintwithvertical)
* [`obj placement`](#obj-placement)
  * [`fn withBottom(value)`](#fn-placementwithbottom)
  * [`fn withHeight(value)`](#fn-placementwithheight)
  * [`fn withLeft(value)`](#fn-placementwithleft)
  * [`fn withRight(value)`](#fn-placementwithright)
  * [`fn withRotation(value)`](#fn-placementwithrotation)
  * [`fn withTop(value)`](#fn-placementwithtop)
  * [`fn withWidth(value)`](#fn-placementwithwidth)

## Fields

### fn withBackground

```jsonnet
withBackground(value)
```

PARAMETERS:

* **value** (`object`)


### fn withBackgroundMixin

```jsonnet
withBackgroundMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withBorder

```jsonnet
withBorder(value)
```

PARAMETERS:

* **value** (`object`)


### fn withBorderMixin

```jsonnet
withBorderMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withConfig

```jsonnet
withConfig(value)
```

PARAMETERS:

* **value** (`object`)

TODO: figure out how to define this (element config(s))
### fn withConfigMixin

```jsonnet
withConfigMixin(value)
```

PARAMETERS:

* **value** (`object`)

TODO: figure out how to define this (element config(s))
### fn withConnections

```jsonnet
withConnections(value)
```

PARAMETERS:

* **value** (`array`)


### fn withConnectionsMixin

```jsonnet
withConnectionsMixin(value)
```

PARAMETERS:

* **value** (`array`)


### fn withConstraint

```jsonnet
withConstraint(value)
```

PARAMETERS:

* **value** (`object`)


### fn withConstraintMixin

```jsonnet
withConstraintMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withName

```jsonnet
withName(value)
```

PARAMETERS:

* **value** (`string`)


### fn withPlacement

```jsonnet
withPlacement(value)
```

PARAMETERS:

* **value** (`object`)


### fn withPlacementMixin

```jsonnet
withPlacementMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withType

```jsonnet
withType(value)
```

PARAMETERS:

* **value** (`string`)


### obj background


#### fn background.withColor

```jsonnet
background.withColor(value)
```

PARAMETERS:

* **value** (`object`)


#### fn background.withColorMixin

```jsonnet
background.withColorMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn background.withImage

```jsonnet
background.withImage(value)
```

PARAMETERS:

* **value** (`object`)

Links to a resource (image/svg path)
#### fn background.withImageMixin

```jsonnet
background.withImageMixin(value)
```

PARAMETERS:

* **value** (`object`)

Links to a resource (image/svg path)
#### fn background.withSize

```jsonnet
background.withSize(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"original"`, `"contain"`, `"cover"`, `"fill"`, `"tile"`


#### obj background.color


##### fn background.color.withField

```jsonnet
background.color.withField(value)
```

PARAMETERS:

* **value** (`string`)

fixed: T -- will be added by each element
##### fn background.color.withFixed

```jsonnet
background.color.withFixed(value)
```

PARAMETERS:

* **value** (`string`)

color value
#### obj background.image


##### fn background.image.withField

```jsonnet
background.image.withField(value)
```

PARAMETERS:

* **value** (`string`)

fixed: T -- will be added by each element
##### fn background.image.withFixed

```jsonnet
background.image.withFixed(value)
```

PARAMETERS:

* **value** (`string`)


##### fn background.image.withMode

```jsonnet
background.image.withMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"fixed"`, `"field"`, `"mapping"`


### obj border


#### fn border.withColor

```jsonnet
border.withColor(value)
```

PARAMETERS:

* **value** (`object`)


#### fn border.withColorMixin

```jsonnet
border.withColorMixin(value)
```

PARAMETERS:

* **value** (`object`)


#### fn border.withRadius

```jsonnet
border.withRadius(value)
```

PARAMETERS:

* **value** (`number`)


#### fn border.withWidth

```jsonnet
border.withWidth(value)
```

PARAMETERS:

* **value** (`number`)


#### obj border.color


##### fn border.color.withField

```jsonnet
border.color.withField(value)
```

PARAMETERS:

* **value** (`string`)

fixed: T -- will be added by each element
##### fn border.color.withFixed

```jsonnet
border.color.withFixed(value)
```

PARAMETERS:

* **value** (`string`)

color value
### obj constraint


#### fn constraint.withHorizontal

```jsonnet
constraint.withHorizontal(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"left"`, `"right"`, `"leftright"`, `"center"`, `"scale"`


#### fn constraint.withVertical

```jsonnet
constraint.withVertical(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"top"`, `"bottom"`, `"topbottom"`, `"center"`, `"scale"`


### obj placement


#### fn placement.withBottom

```jsonnet
placement.withBottom(value)
```

PARAMETERS:

* **value** (`number`)


#### fn placement.withHeight

```jsonnet
placement.withHeight(value)
```

PARAMETERS:

* **value** (`number`)


#### fn placement.withLeft

```jsonnet
placement.withLeft(value)
```

PARAMETERS:

* **value** (`number`)


#### fn placement.withRight

```jsonnet
placement.withRight(value)
```

PARAMETERS:

* **value** (`number`)


#### fn placement.withRotation

```jsonnet
placement.withRotation(value)
```

PARAMETERS:

* **value** (`number`)


#### fn placement.withTop

```jsonnet
placement.withTop(value)
```

PARAMETERS:

* **value** (`number`)


#### fn placement.withWidth

```jsonnet
placement.withWidth(value)
```

PARAMETERS:

* **value** (`number`)

