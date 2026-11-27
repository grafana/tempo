# prometheus

grafonnet.query.prometheus

## Subpackages

* [adhocFilters](adhocFilters.md)
* [scopes](scopes/index.md)

## Index

* [`fn new(datasource, expr)`](#fn-new)
* [`fn withAdhocFilters(value)`](#fn-withadhocfilters)
* [`fn withAdhocFiltersMixin(value)`](#fn-withadhocfiltersmixin)
* [`fn withDatasource(value)`](#fn-withdatasource)
* [`fn withEditorMode(value)`](#fn-witheditormode)
* [`fn withExemplar(value=true)`](#fn-withexemplar)
* [`fn withExpr(value)`](#fn-withexpr)
* [`fn withFormat(value)`](#fn-withformat)
* [`fn withGroupByKeys(value)`](#fn-withgroupbykeys)
* [`fn withGroupByKeysMixin(value)`](#fn-withgroupbykeysmixin)
* [`fn withHide(value=true)`](#fn-withhide)
* [`fn withInstant(value=true)`](#fn-withinstant)
* [`fn withInterval(value)`](#fn-withinterval)
* [`fn withIntervalFactor(value)`](#fn-withintervalfactor)
* [`fn withIntervalMs(value)`](#fn-withintervalms)
* [`fn withLegendFormat(value)`](#fn-withlegendformat)
* [`fn withMaxDataPoints(value)`](#fn-withmaxdatapoints)
* [`fn withQueryType(value)`](#fn-withquerytype)
* [`fn withRange(value=true)`](#fn-withrange)
* [`fn withRefId(value)`](#fn-withrefid)
* [`fn withResultAssertions(value)`](#fn-withresultassertions)
* [`fn withResultAssertionsMixin(value)`](#fn-withresultassertionsmixin)
* [`fn withScopes(value)`](#fn-withscopes)
* [`fn withScopesMixin(value)`](#fn-withscopesmixin)
* [`fn withTimeRange(value)`](#fn-withtimerange)
* [`fn withTimeRangeMixin(value)`](#fn-withtimerangemixin)
* [`obj datasource`](#obj-datasource)
  * [`fn withType(value)`](#fn-datasourcewithtype)
  * [`fn withUid(value)`](#fn-datasourcewithuid)
* [`obj resultAssertions`](#obj-resultassertions)
  * [`fn withMaxFrames(value)`](#fn-resultassertionswithmaxframes)
  * [`fn withType(value)`](#fn-resultassertionswithtype)
  * [`fn withTypeVersion(value)`](#fn-resultassertionswithtypeversion)
  * [`fn withTypeVersionMixin(value)`](#fn-resultassertionswithtypeversionmixin)
* [`obj timeRange`](#obj-timerange)
  * [`fn withFrom(value="now-6h")`](#fn-timerangewithfrom)
  * [`fn withTo(value="now")`](#fn-timerangewithto)

## Fields

### fn new

```jsonnet
new(datasource, expr)
```

PARAMETERS:

* **datasource** (`string`)
* **expr** (`string`)

Creates a new prometheus query target for panels.
### fn withAdhocFilters

```jsonnet
withAdhocFilters(value)
```

PARAMETERS:

* **value** (`array`)

Additional Ad-hoc filters that take precedence over Scope on conflict.
### fn withAdhocFiltersMixin

```jsonnet
withAdhocFiltersMixin(value)
```

PARAMETERS:

* **value** (`array`)

Additional Ad-hoc filters that take precedence over Scope on conflict.
### fn withDatasource

```jsonnet
withDatasource(value)
```

PARAMETERS:

* **value** (`string`)

Set the datasource for this query.
### fn withEditorMode

```jsonnet
withEditorMode(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"builder"`, `"code"`

what we should show in the editor
Possible enum values:
 - `"builder"` 
 - `"code"` 
### fn withExemplar

```jsonnet
withExemplar(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Execute an additional query to identify interesting raw samples relevant for the given expr
### fn withExpr

```jsonnet
withExpr(value)
```

PARAMETERS:

* **value** (`string`)

The actual expression/query that will be evaluated by Prometheus
### fn withFormat

```jsonnet
withFormat(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `"time_series"`, `"table"`, `"heatmap"`

The response format
Possible enum values:
 - `"time_series"` 
 - `"table"` 
 - `"heatmap"` 
### fn withGroupByKeys

```jsonnet
withGroupByKeys(value)
```

PARAMETERS:

* **value** (`array`)

Group By parameters to apply to aggregate expressions in the query
### fn withGroupByKeysMixin

```jsonnet
withGroupByKeysMixin(value)
```

PARAMETERS:

* **value** (`array`)

Group By parameters to apply to aggregate expressions in the query
### fn withHide

```jsonnet
withHide(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

true if query is disabled (ie should not be returned to the dashboard)
NOTE: this does not always imply that the query should not be executed since
the results from a hidden query may be used as the input to other queries (SSE etc)
### fn withInstant

```jsonnet
withInstant(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Returns only the latest value that Prometheus has scraped for the requested time series
### fn withInterval

```jsonnet
withInterval(value)
```

PARAMETERS:

* **value** (`string`)

An additional lower limit for the step parameter of the Prometheus query and for the
`$__interval` and `$__rate_interval` variables.
### fn withIntervalFactor

```jsonnet
withIntervalFactor(value)
```

PARAMETERS:

* **value** (`string`)

Set the interval factor for this query.
### fn withIntervalMs

```jsonnet
withIntervalMs(value)
```

PARAMETERS:

* **value** (`number`)

Interval is the suggested duration between time points in a time series query.
NOTE: the values for intervalMs is not saved in the query model.  It is typically calculated
from the interval required to fill a pixels in the visualization
### fn withLegendFormat

```jsonnet
withLegendFormat(value)
```

PARAMETERS:

* **value** (`string`)

Set the legend format for this query.
### fn withMaxDataPoints

```jsonnet
withMaxDataPoints(value)
```

PARAMETERS:

* **value** (`integer`)

MaxDataPoints is the maximum number of data points that should be returned from a time series query.
NOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated
from the number of pixels visible in a visualization
### fn withQueryType

```jsonnet
withQueryType(value)
```

PARAMETERS:

* **value** (`string`)

QueryType is an optional identifier for the type of query.
It can be used to distinguish different types of queries.
### fn withRange

```jsonnet
withRange(value=true)
```

PARAMETERS:

* **value** (`boolean`)
   - default value: `true`

Returns a Range vector, comprised of a set of time series containing a range of data points over time for each time series
### fn withRefId

```jsonnet
withRefId(value)
```

PARAMETERS:

* **value** (`string`)

RefID is the unique identifier of the query, set by the frontend call.
### fn withResultAssertions

```jsonnet
withResultAssertions(value)
```

PARAMETERS:

* **value** (`object`)

Optionally define expected query result behavior
### fn withResultAssertionsMixin

```jsonnet
withResultAssertionsMixin(value)
```

PARAMETERS:

* **value** (`object`)

Optionally define expected query result behavior
### fn withScopes

```jsonnet
withScopes(value)
```

PARAMETERS:

* **value** (`array`)

A set of filters applied to apply to the query
### fn withScopesMixin

```jsonnet
withScopesMixin(value)
```

PARAMETERS:

* **value** (`array`)

A set of filters applied to apply to the query
### fn withTimeRange

```jsonnet
withTimeRange(value)
```

PARAMETERS:

* **value** (`object`)

TimeRange represents the query range
NOTE: unlike generic /ds/query, we can now send explicit time values in each query
NOTE: the values for timeRange are not saved in a dashboard, they are constructed on the fly
### fn withTimeRangeMixin

```jsonnet
withTimeRangeMixin(value)
```

PARAMETERS:

* **value** (`object`)

TimeRange represents the query range
NOTE: unlike generic /ds/query, we can now send explicit time values in each query
NOTE: the values for timeRange are not saved in a dashboard, they are constructed on the fly
### obj datasource


#### fn datasource.withType

```jsonnet
datasource.withType(value)
```

PARAMETERS:

* **value** (`string`)

The plugin type-id
#### fn datasource.withUid

```jsonnet
datasource.withUid(value)
```

PARAMETERS:

* **value** (`string`)

Specific datasource instance
### obj resultAssertions


#### fn resultAssertions.withMaxFrames

```jsonnet
resultAssertions.withMaxFrames(value)
```

PARAMETERS:

* **value** (`integer`)

Maximum frame count
#### fn resultAssertions.withType

```jsonnet
resultAssertions.withType(value)
```

PARAMETERS:

* **value** (`string`)
   - valid values: `""`, `"timeseries-wide"`, `"timeseries-long"`, `"timeseries-many"`, `"timeseries-multi"`, `"directory-listing"`, `"table"`, `"numeric-wide"`, `"numeric-multi"`, `"numeric-long"`, `"log-lines"`

Type asserts that the frame matches a known type structure.
Possible enum values:
 - `""` 
 - `"timeseries-wide"` 
 - `"timeseries-long"` 
 - `"timeseries-many"` 
 - `"timeseries-multi"` 
 - `"directory-listing"` 
 - `"table"` 
 - `"numeric-wide"` 
 - `"numeric-multi"` 
 - `"numeric-long"` 
 - `"log-lines"` 
#### fn resultAssertions.withTypeVersion

```jsonnet
resultAssertions.withTypeVersion(value)
```

PARAMETERS:

* **value** (`array`)

TypeVersion is the version of the Type property. Versions greater than 0.0 correspond to the dataplane
contract documentation https://grafana.github.io/dataplane/contract/.
#### fn resultAssertions.withTypeVersionMixin

```jsonnet
resultAssertions.withTypeVersionMixin(value)
```

PARAMETERS:

* **value** (`array`)

TypeVersion is the version of the Type property. Versions greater than 0.0 correspond to the dataplane
contract documentation https://grafana.github.io/dataplane/contract/.
### obj timeRange


#### fn timeRange.withFrom

```jsonnet
timeRange.withFrom(value="now-6h")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"now-6h"`

From is the start time of the query.
#### fn timeRange.withTo

```jsonnet
timeRange.withTo(value="now")
```

PARAMETERS:

* **value** (`string`)
   - default value: `"now"`

To is the end time of the query.