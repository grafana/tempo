---
permalink: /
---

# package d

```jsonnet
local d = import "github.com/jsonnet-libs/docsonnet/doc-util"
```

`doc-util` provides a Jsonnet interface for `docsonnet`,
 a Jsonnet API doc generator that uses structured data instead of comments.


## Index

* [`fn arg(name, type, default)`](#fn-arg)
* [`fn fn(help, args)`](#fn-fn)
* [`fn obj(help, fields)`](#fn-obj)
* [`fn pkg(name, url, help)`](#fn-pkg)
* [`fn val(type, help, default)`](#fn-val)
* [`obj argument`](#obj-argument)
  * [`fn new(name, type, default)`](#fn-argumentnew)
* [`obj func`](#obj-func)
  * [`fn new(help, args)`](#fn-funcnew)
  * [`fn withArgs(args)`](#fn-funcwithargs)
  * [`fn withHelp(help)`](#fn-funcwithhelp)
* [`obj object`](#obj-object)
  * [`fn new(help, fields)`](#fn-objectnew)
  * [`fn withFields(fields)`](#fn-objectwithfields)
* [`obj package`](#obj-package)
  * [`fn new(name, url, help)`](#fn-packagenew)
* [`obj value`](#obj-value)
  * [`fn new(type, help, default)`](#fn-valuenew)

## Fields

### fn arg

```ts
arg(name, type, default)
```

`arg` is a shorthand for `argument.new`

### fn fn

```ts
fn(help, args)
```

`fn` is a shorthand for `func.new`

### fn obj

```ts
obj(help, fields)
```

`obj` is a shorthand for `object.new`

### fn pkg

```ts
pkg(name, url, help)
```

`new` is a shorthand for `package.new`

### fn val

```ts
val(type, help, default)
```

`val` is a shorthand for `value.new`

## obj argument

Utilities for creating function arguments

### fn argument.new

```ts
new(name, type, default)
```

new creates a new function argument, taking the name, the type and optionally a default value

## obj func

Utilities for documenting Jsonnet methods (functions of objects)

### fn func.new

```ts
new(help, args)
```

new creates a new function, optionally with description and arguments

### fn func.withArgs

```ts
withArgs(args)
```

The `withArgs` modifier overrides the arguments of that function

### fn func.withHelp

```ts
withHelp(help)
```

The `withHelp` modifier overrides the help text of that function

## obj object

Utilities for documenting Jsonnet objects (`{ }`).

### fn object.new

```ts
new(help, fields)
```

new creates a new object, optionally with description and fields

### fn object.withFields

```ts
withFields(fields)
```

The `withFields` modifier overrides the fields property of an already created object

## obj package



### fn package.new

```ts
new(name, url, help)
```

new creates a new package with given `name`, `import` URL and `help` text

## obj value

Utilities for documenting plain Jsonnet values (primitives)

### fn value.new

```ts
new(type, help, default)
```

new creates a new object of given type, optionally with description and default value