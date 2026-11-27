# rule



## Subpackages

* [data](data.md)

## Index

* [`fn withAnnotations(value)`](#fn-withannotations)
* [`fn withAnnotationsMixin(value)`](#fn-withannotationsmixin)
* [`fn withCondition(value)`](#fn-withcondition)
* [`fn withData(value)`](#fn-withdata)
* [`fn withDataMixin(value)`](#fn-withdatamixin)
* [`fn withExecErrState(value)`](#fn-withexecerrstate)
* [`fn withFolderUID(value)`](#fn-withfolderuid)
* [`fn withFor(value)`](#fn-withfor)
* [`fn withIsPaused(value=true)`](#fn-withispaused)
* [`fn withLabels(value)`](#fn-withlabels)
* [`fn withLabelsMixin(value)`](#fn-withlabelsmixin)
* [`fn withName(value)`](#fn-withname)
* [`fn withNoDataState(value)`](#fn-withnodatastate)
* [`fn withOrgID(value)`](#fn-withorgid)
* [`fn withRuleGroup(value)`](#fn-withrulegroup)

## Fields

### fn withAnnotations

```jsonnet
withAnnotations(value)
```

PARAMETERS:

* **value** (`object`)


### fn withAnnotationsMixin

```jsonnet
withAnnotationsMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withCondition

```jsonnet
withCondition(value)
```

PARAMETERS:

* **value** (`string`)


### fn withData

```jsonnet
withData(value)
```

PARAMETERS:

* **value** (`array`)


### fn withDataMixin

```jsonnet
withDataMixin(value)
```

PARAMETERS:

* **value** (`array`)


### fn withExecErrState

```jsonnet
withExecErrState(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"Alerting"`, `"Error"`, `"OK"`, `"KeepLast"`


### fn withFolderUID

```jsonnet
withFolderUID(value)
```

PARAMETERS:

* **value** (`string`)


### fn withFor

```jsonnet
withFor(value)
```

PARAMETERS:

* **value** (`string`)

The amount of time, in seconds, for which the rule must be breached for the rule to be considered to be Firing.
Before this time has elapsed, the rule is only considered to be Pending.
### fn withIsPaused

```jsonnet
withIsPaused(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`


### fn withLabels

```jsonnet
withLabels(value)
```

PARAMETERS:

* **value** (`object`)


### fn withLabelsMixin

```jsonnet
withLabelsMixin(value)
```

PARAMETERS:

* **value** (`object`)


### fn withName

```jsonnet
withName(value)
```

PARAMETERS:

* **value** (`string`)


### fn withNoDataState

```jsonnet
withNoDataState(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"OK"`, `"Alerting"`, `"NoData"`, `"KeepLast"`


### fn withOrgID

```jsonnet
withOrgID(value)
```

PARAMETERS:

* **value** (`integer`)


### fn withRuleGroup

```jsonnet
withRuleGroup(value)
```

PARAMETERS:

* **value** (`string`)

