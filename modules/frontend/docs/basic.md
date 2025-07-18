# TraceQL Basic Query Documentation

## Overview
TraceQL is a query language for searching and filtering traces. Queries are expressions evaluated on one trace at a time, structured as a set of chained expressions called a pipeline. TraceQL selection can not answer broad questions about trace data. It can only find specific traces. Use metrics for broader questions.

## Query Structure
TraceQL queries use curly brackets `{}` to select spans from traces. Each query expression filters spans based on conditions.

### Basic Syntax
```
{ <conditions> }
```

### Pipeline Structure
```
{ <span_selection> } | <aggregator> | <condition>
```

## Scopes and Attributes
TraceQL differentiates between intrinsics (fundamental to spans) and attributes (customizable key-value pairs).

### Intrinsic Fields
Intrinsic fields use colon notation `<scope>:`:
- `span:status` - Status: error, ok, or unset
- `span:statusMessage` - Optional text accompanying span status
- `span:duration` - End - start time of the span
- `span:name` - Operation or span name
- `span:kind` - Kind: server, client, producer, consumer, internal, unspecified
- `span:id` - Span ID using hex string
- `span:parentID` - Parent span ID using hex string
- `trace:duration` - Max(end) - min(start) time of spans in trace
- `trace:rootName` - Name of the root span (if it exists)
- `trace:rootService` - Service name of the root span (if it exists)
- `trace:id` - Trace ID using hex string
- `event:name` - Name of event
- `event:timeSinceStart` - Time of event relative to span start
- `link:spanID` - Link span ID using hex string
- `link:traceID` - Link trace ID using hex string
- `instrumentation:name` - Instrumentation scope name
- `instrumentation:version` - Instrumentation scope version

### Attribute Fields
Attributes use dot notation `<scope>.`:
- `span.` - Span attributes
- `resource.` - Resource attributes
- `event.` - Event attributes
- `link.` - Link attributes
- `instrumentation.` - Instrumentation scope attributes

## Comparison Operators
- `=` - Equality
- `!=` - Inequality
- `>` - Greater than
- `>=` - Greater than or equal
- `<` - Less than
- `<=` - Less than or equal
- `=~` - Regular expression match (fully anchored)
- `!~` - Negated regular expression match

### Regular Expression Notes
- All regex patterns are fully anchored (treated as `^pattern$`)
- Use `.*pattern.*` for unanchored matching
- TraceQL uses Golang regular expressions

## Logical Operators
- `&&` - AND (all conditions must be true on the same span)
- `||` - OR (either condition can be true)

## Spanset Operators
Spanset operators allow conditions to be asserted on different spans within a trace. Also see strutural operators.

- `{condA} && {condB}` - Both conditions found matches
- `{condA} || {condB}` - Either condition found matches

## Common Examples

### Find traces of a specific operation
```
{ resource.service.name = "frontend" && span:name = "POST /api/orders" }
```

### Find traces with environment and service specification
```
{
  resource.service.namespace = "ecommerce" &&
  resource.service.name = "frontend" &&
  resource.deployment.environment = "production" &&
  span:name = "POST /api/orders"
}
```

### Find traces with errors
```
{ resource.service.name = "frontend" && span:name = "POST /api/orders" && span:status = error }
```

### Find traces with HTTP 5xx errors
```
{ resource.service.name = "frontend" && span:name = "POST /api/orders" && span.http.status_code >= 500 }
```

### Find traces with specific behavior patterns
```
{ resource.service.name = "frontend" && span:name = "GET /api/products/{id}" } && { span.db.system = "postgresql" }
```

### Find traces going through multiple environments
```
{ resource.deployment.environment = "production" } && { resource.deployment.environment = "staging" }
```

### Find traces with arrays
```
{ span.foo = "bar" }
{ span.http.request.header.Accept =~ "application.*" }
```

### Field expressions with combined conditions
```
{ span.http.status_code >= 200 && span.http.status_code < 300 }
{ span.http.method = "DELETE" && span:status != ok }
```

### Check attribute existence
```
{ span.any_attribute != nil }
```

### Using quoted attribute names
```
{ span."attribute name with space" = "value" }
{ span.attribute."attribute name with space" = "value" }
```

## Performance Tips
- Use trace-level intrinsics (`trace:duration`, `trace:rootName`, `trace:rootService`) when possible for better performance
- Use appropriate scopes to limit data scanning
- Use specific span selections to reduce data processing