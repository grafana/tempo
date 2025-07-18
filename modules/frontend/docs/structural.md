# TraceQL Structural Operators Documentation

## Overview
Structural operators in TraceQL examine the hierarchical relationships between spans within a trace. These operators help you find traces based on parent-child relationships and span ordering. **Important** For all structural operators (except union) only the spans matching the right hand side conditions are returned.

## Structural Operators

### Descendant Operator (`>>`)
Finds spans matching the right condition that are descendants of spans matching the left condition.
```
{condA} >> {condB}
```

Example - Find database calls downstream from frontend service:
```
{ resource.service.name = "frontend" } >> { span.db.system = "postgresql" }
```

### Ancestor Operator (`<<`)
Finds spans matching the right condition that are ancestors of spans matching the left condition.
```
{condA} << {condB}
```

Example - Find root spans that have error descendants:
```
{ span:status = error } << { span:kind = server }
```

### Child Operator (`>`)
Finds spans matching the right condition that are direct children of spans matching the left condition.
```
{condA} > {condB}
```

Example - Find direct child spans of HTTP requests:
```
{ span.http.method = "GET" } > { span.db.system != nil }
```

### Parent Operator (`<`)
Finds spans matching the right condition that are direct parents of spans matching the left condition.
```
{condA} < {condB}
```

Example - Find parent spans of database operations:
```
{ span.db.system = "postgresql" } < { span.http.method != nil }
```

### Sibling Operator (`~`)
Finds spans matching the right condition that have siblings matching the left condition.
```
{condA} ~ {condB}
```

Example - Find spans that are siblings of error spans:
```
{ span:status = error } ~ { span:status = ok }
```

## Union Structural Operators
These operators return spans that match on both sides of the operator:

### Union Descendant (`&>>`)
Returns both the ancestor and descendant spans.
```
{ span.http.url = "/api/orders" && span:status = error } &>> { span:status = error }
```

### Union Ancestor (`&<<`)
Returns both the descendant and ancestor spans.

### Union Child (`&>`)
Returns both the parent and child spans.

### Union Parent (`&<`)
Returns both the child and parent spans.

### Union Sibling (`&~`)
Returns both sibling spans.

## Experimental Structural Operators (Not Operators)
These operators can have false positives but are useful for specific queries:

### Not-Descendant (`!>>`)
Finds spans that are NOT descendants of the left condition.
```
{ } !>> { resource.service.name = "database" }
```

### Not-Child (`!>`)
Finds spans that are NOT direct children of the left condition.

### Not-Parent (`!<`)
Finds spans that are NOT direct parents of the left condition.

Example - Find leaf spans in a service:
```
{ } !< { resource.service.name = "productcatalog" }
```

### Not-Sibling (`!~`)
Finds spans that are NOT siblings of the left condition.

## Practical Examples

### Find cascading errors
Find the last error in a series of cascading errors:
```
{ span:status = error } !< { span:status = error }
```

### Find service interaction patterns
Find frontend calls that lead to database operations:
```
{ resource.service.name = "frontend" && span.http.method = "POST" } >> { span.db.system = "postgresql" }
```

### Find cross-environment issues
Find traces that go through both production and staging:
```
{ resource.deployment.environment = "production" } && { resource.deployment.environment = "staging" }
```

### Find service dependencies
Find which services call the user service:
```
{ resource.service.name = "user" } << { resource.service.name != "user" }
```

## Performance Considerations
- Structural operators can be expensive on large traces
- Consider using logical operators (`&&`, `||`) instead when span relationships aren't required
- Combine with other filters to improve performance