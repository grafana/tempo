---
Authors: Martin Disibio (@mdisibio), Joe Elliott (@joe-elliott), Jennie Pham (@ie-pham), Adrian Stoewer (@stoewer)
Created: 2023 July - 2023 November
Last updated: 2023-11-28
---

# TraceQL Extensions

## Summary

This design document describes extensions to the existing TraceQL language that allow for searching new fields and data types.

This document is **not** meant to be a complete language specification. Rather it is meant to invite the community to comment on the basic structure and concepts of the language extensions.

### Table of Contents

- [Escaping Attribute Names](#escaping-attribute-names)
- [Additional Scopes](#additional-scopes)
  - [Trace](#trace)
  - [Instrumentation Scope](#instrumentation-scope)
  - [Event](#event)
  - [Link](#link)
- [Scoped Intrinsics](#scoped-intrinsics)
- [Additional Data Types](#additional-data-types)
  - [Arrays](#arrays)
  - [IDs](#ids)
- [Future Considerations](#future-considerations)

## Escaping Attribute Names

OpenTelemetry supports [any valid Unicode sequence](https://opentelemetry.io/docs/specs/otel/common/attribute-naming/) for attribute names. This includes characters problematic for a query language such as whitespace, mathematical symbols, various braces, etc. Previously TraceQL did not support [these problematic characters](https://github.com/grafana/tempo/blob/c00e7ef1103a8e7804b6f71fe8e90d0a2d9a9b02/pkg/traceql/lexer.go#L344) in attributes names.

TraceQL will be extended to support these characters by allowing attribute names wrapped in `"` to be unparsed.

`{ span."attribute with spaces" = "foo"}`

TraceQL will also support the escape sequences `\\` and `\"` in an attribute name.

`{ span."this is \"bad\"" = "foo"}`

## Additional Scopes

We have added four additional scopes to the TraceQL language: [trace](#trace), [scope](#instrumentation-scope), [event](#event), and [link](#link). See below for details.

### Trace

The trace scope allows a user to access intrinsic fields that exist at the trace level. There are no attributes at the trace scope. See [intrinsics](#scoped-intrinsics) for a complete list of allowed trace level intrinsics.

```
{ trace:duration > 100ms }
```

### Instrumentation Scope

The [Instrumentation Scope](https://github.com/open-telemetry/opentelemetry-proto/blob/ea449ae0e9b282f96ec12a09e796dbb3d390ed4f/opentelemetry/proto/common/v1/common.proto#L71) object contains information about the instrumentation used to generate telemetry data. This scope contains new intrinsics as well as attributes. It will be referenced using the `scope` keyword.

```
{ scope.foo = "bar" }
{ scope:name ~= ".*Java.*" }
```

### Event

Span [events](https://github.com/open-telemetry/opentelemetry-proto/blob/ea449ae0e9b282f96ec12a09e796dbb3d390ed4f/opentelemetry/proto/trace/v1/trace.proto#L213) contain attributes as well as intrinsic properties that describe discrete events that occur during a span's lifetime. Note that both events and [links](#link) are different than existing scopes because there can be many events to one span. Due to this we are being purposefully conservative when defining event scopes and intend to extend functionality as we better understand use cases. See [future considerations](#future-considerations) for additional discussion.

```
{ event.exception.message =~ ".*Division by zero.*" }
{ event:name = "exception" }
```

In the above cases the operators will return true if there are any events on the span that meets the specified condition.

### Link

Span [links](https://github.com/open-telemetry/opentelemetry-proto/blob/ea449ae0e9b282f96ec12a09e796dbb3d390ed4f/opentelemetry/proto/trace/v1/trace.proto#L242) contain attributes as well as intrinsic properties that can create connections to other spans. Note that both [events](#event) and links are different than existing scopes because there can be many events to one span. Due to this we are being purposefully conservative when defining event scopes and intend to extend functionality as we better understand use cases. See [future considerations](#future-considerations) for additional discussion.

```
{ link.foo = "bar" }
{ link:traceID = "<hex string>" }
```

In the above cases the operators will return true if there are any links on the span that meets the specified condition.

## Scoped Intrinsics

TraceQL supports a number of intrinsics that will continue to exist in the language. These will be referred to as "Legacy Intrinsics" and there are currently no plans to remove them. However, due to the increasing number of [scopes](#additional-scopes) we will provide scoped names for all legacy intrinsics and all future intrinsics will be scoped only.

Scoped intrinsics will always use a `:` to separate the scope from the intrinsic to differentiate between an attribute of the same name. i.e. `span.name` accesses an attribute "name" on the span vs `span:name` which accesses the intrinsic span name.

Legacy intrinsics will be kept for the foreseeable future to preserve backwards compatibility, but we will not be adding any new unscoped intrinsics.

| Scope  | Intrinsic      | Legacy | Description |
| ------ | -------------- | ------ | ----------- |
| trace  | :duration      | traceDuration   | Max end time of any span minus the min start time of any span in the trace. |
|        | :id            |                 | Trace id |
|        | :rootName      | rootName | `span:name` of the root span |
|        | :rootService   | rootServiceName        | `resource.service.name` of the root span |
| scope  | :name          |                 | Instrumentation scope name |
|        | :version       |                 | Instrumentation scope version |
| span   | :id            |                 | Span id |
|        | :name          | name            | Span name |
|        | :duration      | duration        | Span duration |
|        | :status        | status          | Span status | 
|        | :statusMessage | statusMessage   | Span status | 
|        | :kind          | kind            | Span kind |
| event  | :name          |                 | Event name |
| link   | :spanID        |                 | Link's span id |
|        | :traceID       |                 | Link's trace id |
| parent | :id            |                 | Parent's `span:id` |

## Additional Data Types

### Arrays

TraceQL will support [array](https://github.com/open-telemetry/opentelemetry-proto/blob/ea449ae0e9b282f96ec12a09e796dbb3d390ed4f/opentelemetry/proto/common/v1/common.proto#L44) attribute types with the `[]` syntax.

To access a specific element of an array use a 0-based array syntax.

```
{ span.http.response.header.content-type[0] = "application/json" }
```

To test all elements of an array use empty square brackets. This will evaluate to true if any of the elements in the array `http.response.header.content-type` are equal to the value `"application/json"`.

```
{ span.http.response.header.content-type[] = "application/json" }
```

Note that currently there is no array literal or operations defined on the array data type itself. For now we will limit ourselves to existing operations on the elements of the array using the above syntax.

### IDs

The new [scoped intrinsics](#scoped-intrinsics) section above defines a few fields with an "id" data type such as `span:id`. The id type is only comparable to hex strings and only supports the operations `=` and `!=`. 

```
{ span.id = "8bf5306cb6a28" }
```

Span IDs and trace IDs are always 64 bit or 128 bit. However, for the purposes of comparison, leading 0s will be ignored. The following are equivalent.

```
{ trace.id = "0007f2b8d1c69375e0d46a9cf8072bc4" }
{ trace.id = "7f2b8d1c69375e0d46a9cf8072bc4" }
```

## Future Considerations

[Links](#link) and [events](#event) have been introduced with purposefully limited syntax. We think it wise to not commit to language features that we will regret later and want the community to experiment with basic access to these fields and communicate their needs so we can continue to move the language forward in a way that makes sense for everyone. It would be very easy to unnecessarily over complicate the language and we'd prefer taking a slower approach.

Note that currently you can only assert whether there are any links or events with specific field values on a span. We believe this covers a huge range of use cases, but also leaves some obvious gaps. As you are exploring links and events consider how you would like to express more complex relationships such as asserting two conditions on the same event or comparing fields across links.

Thanks for your patience! Hopefully this functionality will jump start links and events in TraceQL and we can quickly follow up with more advanced functionality.