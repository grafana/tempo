# overrides



## Subpackages

* [properties](properties.md)

## Index

* [`fn withMatcher(value)`](#fn-withmatcher)
* [`fn withMatcherMixin(value)`](#fn-withmatchermixin)
* [`fn withProperties(value)`](#fn-withproperties)
* [`fn withPropertiesMixin(value)`](#fn-withpropertiesmixin)
* [`fn withSystemRef(value)`](#fn-withsystemref)
* [`obj matcher`](#obj-matcher)
  * [`fn withId(value="")`](#fn-matcherwithid)
  * [`fn withOptions(value)`](#fn-matcherwithoptions)
  * [`fn withOptionsMixin(value)`](#fn-matcherwithoptionsmixin)

## Fields

### fn withMatcher

```jsonnet
withMatcher(value)
```

PARAMETERS:

* **value** (`object`)

Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
### fn withMatcherMixin

```jsonnet
withMatcherMixin(value)
```

PARAMETERS:

* **value** (`object`)

Matcher is a predicate configuration. Based on the config a set of field(s) or values is filtered in order to apply override / transformation.
It comes with in id ( to resolve implementation from registry) and a configuration that’s specific to a particular matcher type.
### fn withProperties

```jsonnet
withProperties(value)
```

PARAMETERS:

* **value** (`array`)


### fn withPropertiesMixin

```jsonnet
withPropertiesMixin(value)
```

PARAMETERS:

* **value** (`array`)


### fn withSystemRef

```jsonnet
withSystemRef(value)
```

PARAMETERS:

* **value** (`string`)


### obj matcher


#### fn matcher.withId

```jsonnet
matcher.withId(value="")
```

PARAMETERS:

* **value** (`string`)
   - default value: `""`

The matcher id. This is used to find the matcher implementation from registry.
#### fn matcher.withOptions

```jsonnet
matcher.withOptions(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.
#### fn matcher.withOptionsMixin

```jsonnet
matcher.withOptionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

The matcher options. This is specific to the matcher implementation.