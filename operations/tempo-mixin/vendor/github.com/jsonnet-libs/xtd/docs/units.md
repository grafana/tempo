---
permalink: /units/
---

# units

```jsonnet
local units = import "github.com/jsonnet-libs/xtd/units.libsonnet"
```

`units` implements helper functions for converting units.

## Index

* [`fn formatDuration(seconds)`](#fn-formatduration)
* [`fn parseDuration(duration)`](#fn-parseduration)
* [`fn parseKubernetesCPU(input)`](#fn-parsekubernetescpu)
* [`fn siToBytes(str)`](#fn-sitobytes)

## Fields

### fn formatDuration

```ts
formatDuration(seconds)
```

`formatDuration` formats a number of seconds into a human-readable duration string.
Returns the duration in the smallest appropriate unit (s, m, h, or combined formats like "4m30s").


### fn parseDuration

```ts
parseDuration(duration)
```

`parseDuration` parses a duration string and returns the number of seconds.
Handles milliseconds (ms), seconds (s), minutes (m), hours (h), and combined formats like "4m30s" or "1h30m".


### fn parseKubernetesCPU

```ts
parseKubernetesCPU(input)
```

`parseKubernetesCPU` parses a Kubernetes CPU string/number into a number of cores.
 The function assumes the input is in a correct Kubernetes format, i.e., an integer, a float,
 a string representation of an integer or a float, or a string containing a number ending with 'm'
 representing a number of millicores.


### fn siToBytes

```ts
siToBytes(str)
```

`siToBytes` converts Kubernetes byte units to bytes.
Only works for limited set of SI prefixes: Ki, Mi, Gi, Ti.
