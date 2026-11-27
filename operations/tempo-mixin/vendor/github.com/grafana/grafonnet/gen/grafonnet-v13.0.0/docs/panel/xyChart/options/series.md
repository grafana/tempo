# series



## Index

* [`fn withColor(value)`](#fn-withcolor)
* [`fn withColorMixin(value)`](#fn-withcolormixin)
* [`fn withFrame(value)`](#fn-withframe)
* [`fn withFrameMixin(value)`](#fn-withframemixin)
* [`fn withName(value)`](#fn-withname)
* [`fn withNameMixin(value)`](#fn-withnamemixin)
* [`fn withSize(value)`](#fn-withsize)
* [`fn withSizeMixin(value)`](#fn-withsizemixin)
* [`fn withX(value)`](#fn-withx)
* [`fn withXMixin(value)`](#fn-withxmixin)
* [`fn withY(value)`](#fn-withy)
* [`fn withYMixin(value)`](#fn-withymixin)
* [`obj color`](#obj-color)
  * [`fn withMatcher(value)`](#fn-colorwithmatcher)
  * [`fn withMatcherMixin(value)`](#fn-colorwithmatchermixin)
  * [`obj matcher`](#obj-colormatcher)
    * [`fn withId(value="")`](#fn-colormatcherwithid)
    * [`fn withOptions(value)`](#fn-colormatcherwithoptions)
    * [`fn withOptionsMixin(value)`](#fn-colormatcherwithoptionsmixin)
* [`obj frame`](#obj-frame)
  * [`fn withMatcher(value)`](#fn-framewithmatcher)
  * [`fn withMatcherMixin(value)`](#fn-framewithmatchermixin)
  * [`obj matcher`](#obj-framematcher)
    * [`fn withId(value="")`](#fn-framematcherwithid)
    * [`fn withOptions(value)`](#fn-framematcherwithoptions)
    * [`fn withOptionsMixin(value)`](#fn-framematcherwithoptionsmixin)
* [`obj name`](#obj-name)
  * [`fn withFixed(value)`](#fn-namewithfixed)
* [`obj size`](#obj-size)
  * [`fn withMatcher(value)`](#fn-sizewithmatcher)
  * [`fn withMatcherMixin(value)`](#fn-sizewithmatchermixin)
  * [`obj matcher`](#obj-sizematcher)
    * [`fn withId(value="")`](#fn-sizematcherwithid)
    * [`fn withOptions(value)`](#fn-sizematcherwithoptions)
    * [`fn withOptionsMixin(value)`](#fn-sizematcherwithoptionsmixin)
* [`obj x`](#obj-x)
  * [`fn withMatcher(value)`](#fn-xwithmatcher)
  * [`fn withMatcherMixin(value)`](#fn-xwithmatchermixin)
  * [`obj matcher`](#obj-xmatcher)
    * [`fn withId(value="")`](#fn-xmatcherwithid)
    * [`fn withOptions(value)`](#fn-xmatcherwithoptions)
    * [`fn withOptionsMixin(value)`](#fn-xmatcherwithoptionsmixin)
* [`obj y`](#obj-y)
  * [`fn withMatcher(value)`](#fn-ywithmatcher)
  * [`fn withMatcherMixin(value)`](#fn-ywithmatchermixin)
  * [`obj matcher`](#obj-ymatcher)
    * [`fn withId(value="")`](#fn-ymatcherwithid)
    * [`fn withOptions(value)`](#fn-ymatcherwithoptions)
    * [`fn withOptionsMixin(value)`](#fn-ymatcherwithoptionsmixin)

## Fields

### fn withColor

```jsonnet
withColor(value)
```

PARAMETERS:

* **value** (`object`)


### fn withColorMixin

```jsonnet
withColorMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withFrame

```jsonnet
withFrame(value)
```

PARAMETERS:

* **value** (`object`)


### fn withFrameMixin

```jsonnet
withFrameMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withName

```jsonnet
withName(value)
```

PARAMETERS:

* **value** (`object`)


### fn withNameMixin

```jsonnet
withNameMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withSize

```jsonnet
withSize(value)
```

PARAMETERS:

* **value** (`object`)


### fn withSizeMixin

```jsonnet
withSizeMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withX

```jsonnet
withX(value)
```

PARAMETERS:

* **value** (`object`)


### fn withXMixin

```jsonnet
withXMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withY

```jsonnet
withY(value)
```

PARAMETERS:

* **value** (`object`)


### fn withYMixin

```jsonnet
withYMixin(value)
```

PARAMETERS:

* **value** (`object`)


### obj color


#### fn color.withMatcher

```jsonnet
color.withMatcher(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### fn color.withMatcherMixin

```jsonnet
color.withMatcherMixin(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### obj color.matcher


##### fn color.matcher.withId

```jsonnet
color.matcher.withId(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`

The matcher id. This is used to find the matcher implementation from registry.
##### fn color.matcher.withOptions

```jsonnet
color.matcher.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
##### fn color.matcher.withOptionsMixin

```jsonnet
color.matcher.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
### obj frame


#### fn frame.withMatcher

```jsonnet
frame.withMatcher(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### fn frame.withMatcherMixin

```jsonnet
frame.withMatcherMixin(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### obj frame.matcher


##### fn frame.matcher.withId

```jsonnet
frame.matcher.withId(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`

The matcher id. This is used to find the matcher implementation from registry.
##### fn frame.matcher.withOptions

```jsonnet
frame.matcher.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
##### fn frame.matcher.withOptionsMixin

```jsonnet
frame.matcher.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
### obj name


#### fn name.withFixed

```jsonnet
name.withFixed(value)
```

PARAMETERS:

* **value** (`string`)


### obj size


#### fn size.withMatcher

```jsonnet
size.withMatcher(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### fn size.withMatcherMixin

```jsonnet
size.withMatcherMixin(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### obj size.matcher


##### fn size.matcher.withId

```jsonnet
size.matcher.withId(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`

The matcher id. This is used to find the matcher implementation from registry.
##### fn size.matcher.withOptions

```jsonnet
size.matcher.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
##### fn size.matcher.withOptionsMixin

```jsonnet
size.matcher.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
### obj x


#### fn x.withMatcher

```jsonnet
x.withMatcher(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### fn x.withMatcherMixin

```jsonnet
x.withMatcherMixin(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### obj x.matcher


##### fn x.matcher.withId

```jsonnet
x.matcher.withId(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`

The matcher id. This is used to find the matcher implementation from registry.
##### fn x.matcher.withOptions

```jsonnet
x.matcher.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
##### fn x.matcher.withOptionsMixin

```jsonnet
x.matcher.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
### obj y


#### fn y.withMatcher

```jsonnet
y.withMatcher(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### fn y.withMatcherMixin

```jsonnet
y.withMatcherMixin(value)
```

PARAMETERS:

* **value** (`object`)

NOTE: (copied from dashboard_kind.cue, since not exported)
Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
#### obj y.matcher


##### fn y.matcher.withId

```jsonnet
y.matcher.withId(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`

The matcher id. This is used to find the matcher implementation from registry.
##### fn y.matcher.withOptions

```jsonnet
y.matcher.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
##### fn y.matcher.withOptionsMixin

```jsonnet
y.matcher.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.