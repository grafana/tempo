# threshold.step



## Index

* [`fn withColor(value)`](#fn-withcolor)
* [`fn withValue(value)`](#fn-withvalue)

## Fields

### fn withColor

```jsonnet
withColor(value)
```

PARAMETERS:

* **value** (`string`)

Color represents the color of the visual change that will occur in the dashboard when the threshold value is met or exceeded.
### fn withValue

```jsonnet
withValue(value)
```

PARAMETERS:

* **value** (`number`)

Value represents a specified metric for the threshold, which triggers a visual change in the dashboard when this value is met or exceeded.
Nulls currently appear here when serializing -Infinity to JSON.