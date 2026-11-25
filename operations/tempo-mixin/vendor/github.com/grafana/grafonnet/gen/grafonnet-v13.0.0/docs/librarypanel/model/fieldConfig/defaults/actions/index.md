# actions



## Subpackages

* [variables](variables.md)

## Index

* [`fn withConfirmation(value)`](#fn-withconfirmation)
* [`fn withFetch(value)`](#fn-withfetch)
* [`fn withFetchMixin(value)`](#fn-withfetchmixin)
* [`fn withInfinity(value)`](#fn-withinfinity)
* [`fn withInfinityMixin(value)`](#fn-withinfinitymixin)
* [`fn withOneClick(value=true)`](#fn-withoneclick)
* [`fn withStyle(value)`](#fn-withstyle)
* [`fn withStyleMixin(value)`](#fn-withstylemixin)
* [`fn withTitle(value)`](#fn-withtitle)
* [`fn withType(value)`](#fn-withtype)
* [`fn withVariables(value)`](#fn-withvariables)
* [`fn withVariablesMixin(value)`](#fn-withvariablesmixin)
* [`obj fetch`](#obj-fetch)
  * [`fn withBody(value)`](#fn-fetchwithbody)
  * [`fn withHeaders(value)`](#fn-fetchwithheaders)
  * [`fn withHeadersMixin(value)`](#fn-fetchwithheadersmixin)
  * [`fn withMethod(value)`](#fn-fetchwithmethod)
  * [`fn withQueryParams(value)`](#fn-fetchwithqueryparams)
  * [`fn withQueryParamsMixin(value)`](#fn-fetchwithqueryparamsmixin)
  * [`fn withUrl(value)`](#fn-fetchwithurl)
* [`obj infinity`](#obj-infinity)
  * [`fn withBody(value)`](#fn-infinitywithbody)
  * [`fn withDatasourceUid(value)`](#fn-infinitywithdatasourceuid)
  * [`fn withHeaders(value)`](#fn-infinitywithheaders)
  * [`fn withHeadersMixin(value)`](#fn-infinitywithheadersmixin)
  * [`fn withMethod(value)`](#fn-infinitywithmethod)
  * [`fn withQueryParams(value)`](#fn-infinitywithqueryparams)
  * [`fn withQueryParamsMixin(value)`](#fn-infinitywithqueryparamsmixin)
  * [`fn withUrl(value)`](#fn-infinitywithurl)
* [`obj style`](#obj-style)
  * [`fn withBackgroundColor(value)`](#fn-stylewithbackgroundcolor)

## Fields

### fn withConfirmation

```jsonnet
withConfirmation(value)
```

PARAMETERS:

* **value** (`string`)


### fn withFetch

```jsonnet
withFetch(value)
```

PARAMETERS:

* **value** (`object`)

Fetch options
### fn withFetchMixin

```jsonnet
withFetchMixin(value)
```

PARAMETERS:

* **value** (`object`)

Fetch options
### fn withInfinity

```jsonnet
withInfinity(value)
```

PARAMETERS:

* **value** (`object`)

Infinity options
### fn withInfinityMixin

```jsonnet
withInfinityMixin(value)
```

PARAMETERS:

* **value** (`object`)

Infinity options
### fn withOneClick

```jsonnet
withOneClick(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


### fn withStyle

```jsonnet
withStyle(value)
```

PARAMETERS:

* **value** (`object`)


### fn withStyleMixin

```jsonnet
withStyleMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withTitle

```jsonnet
withTitle(value)
```

PARAMETERS:

* **value** (`string`)


### fn withType

```jsonnet
withType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"fetch"`, `"infinity"`

Dashboard action type
### fn withVariables

```jsonnet
withVariables(value)
```

PARAMETERS:

* **value** (`array`)


### fn withVariablesMixin

```jsonnet
withVariablesMixin(value)
```

PARAMETERS:

* **value** (`array`)


### obj fetch


#### fn fetch.withBody

```jsonnet
fetch.withBody(value)
```

PARAMETERS:

* **value** (`string`)


#### fn fetch.withHeaders

```jsonnet
fetch.withHeaders(value)
```

PARAMETERS:

* **value** (`array`)


#### fn fetch.withHeadersMixin

```jsonnet
fetch.withHeadersMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### fn fetch.withMethod

```jsonnet
fetch.withMethod(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"GET"`, `"PUT"`, `"POST"`, `"DELETE"`, `"PATCH"`


#### fn fetch.withQueryParams

```jsonnet
fetch.withQueryParams(value)
```

PARAMETERS:

* **value** (`array`)

These are 2D arrays of strings, each representing a key-value pair
We are defining this way because we can't generate a go struct that
that would have exactly two strings in each sub-array
#### fn fetch.withQueryParamsMixin

```jsonnet
fetch.withQueryParamsMixin(value)
```

PARAMETERS:

* **value** (`array`)

These are 2D arrays of strings, each representing a key-value pair
We are defining this way because we can't generate a go struct that
that would have exactly two strings in each sub-array
#### fn fetch.withUrl

```jsonnet
fetch.withUrl(value)
```

PARAMETERS:

* **value** (`string`)


### obj infinity


#### fn infinity.withBody

```jsonnet
infinity.withBody(value)
```

PARAMETERS:

* **value** (`string`)


#### fn infinity.withDatasourceUid

```jsonnet
infinity.withDatasourceUid(value)
```

PARAMETERS:

* **value** (`string`)


#### fn infinity.withHeaders

```jsonnet
infinity.withHeaders(value)
```

PARAMETERS:

* **value** (`array`)


#### fn infinity.withHeadersMixin

```jsonnet
infinity.withHeadersMixin(value)
```

PARAMETERS:

* **value** (`array`)


#### fn infinity.withMethod

```jsonnet
infinity.withMethod(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"GET"`, `"PUT"`, `"POST"`, `"DELETE"`, `"PATCH"`


#### fn infinity.withQueryParams

```jsonnet
infinity.withQueryParams(value)
```

PARAMETERS:

* **value** (`array`)

These are 2D arrays of strings, each representing a key-value pair
We are defining them this way because we can't generate a go struct that
that would have exactly two strings in each sub-array
#### fn infinity.withQueryParamsMixin

```jsonnet
infinity.withQueryParamsMixin(value)
```

PARAMETERS:

* **value** (`array`)

These are 2D arrays of strings, each representing a key-value pair
We are defining them this way because we can't generate a go struct that
that would have exactly two strings in each sub-array
#### fn infinity.withUrl

```jsonnet
infinity.withUrl(value)
```

PARAMETERS:

* **value** (`string`)


### obj style


#### fn style.withBackgroundColor

```jsonnet
style.withBackgroundColor(value)
```

PARAMETERS:

* **value** (`string`)

