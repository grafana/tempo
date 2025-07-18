# TraceQL Aggregates Documentation

## Overview
TraceQL aggregates allow you to perform calculations on sets of spans within traces. These functions help select individual traces based on aggregate data in the spans.

## Available Aggregate Functions
- `count` - Count of spans in the spanset
- `avg` - Average of a numeric attribute or intrinsic
- `max` - Maximum value of a numeric attribute or intrinsic
- `min` - Minimum value of a numeric attribute or intrinsic
- `sum` - Sum of a numeric attribute or intrinsic

## Basic Usage
Aggregates are applied to spansets using the pipe operator `|`:
```
{ <span_selection> } | <aggregate_function>
```

## Examples

### Count Aggregates
Count total spans in a trace:
```
{ } | count()
```

Count spans with specific conditions:
```
{ span.http.status_code = 200 } | count()
```

Find traces with more than 10 spans:
```
{ } | count() > 10
```

Find traces with more than 3 error spans:
```
{ span:status = error } | count() > 3
```

### Duration Aggregates
Find traces with average span duration > 20ms:
```
{ } | avg(span:duration) > 20ms
```

Find traces with maximum span duration > 1s:
```
{ } | max(span:duration) > 1s
```

Find traces with minimum span duration < 1ms:
```
{ } | min(span:duration) < 1ms
```

### Custom Attribute Aggregates
Sum up custom numeric attributes:
```
{ } | sum(span.bytes_processed) > 1000000
```

Average response size:
```
{ span.http.response_size != nil } | avg(span.http.response_size) > 1024
```

### Combining with Conditions
Find slow frontend operations:
```
{ resource.service.name = "frontend" } | avg(span:duration) > 100ms
```

Find services with high error rates:
```
{ span:status = error } | count() > 5
```

## Grouping with Aggregates
You can group spans before applying aggregates using the `by()` operator:
```
{ span:status = error } | by(resource.service.name) | count() > 1
```

This finds services with more than 1 error span per trace.

## Performance Notes
- Aggregates are computed per trace
