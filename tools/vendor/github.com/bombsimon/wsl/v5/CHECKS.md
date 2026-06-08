# Checks

This document describes all the checks done by `wsl` with examples of what's not
allowed and what's allowed.

## `assign`

Assign (`foo := bar`) or re-assignments (`foo = bar`) should only be cuddled
with other assignments or increment/decrement.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
if true {
    fmt.Println("hello")
}
a := 1 // 1

defer func() {
    fmt.Println("hello")
}()
a := 1 // 2
```

</td><td valign="top">

```go
if true {
    fmt.Println("hello")
}

a := 1

defer func() {
    fmt.Println("hello")
}()

a := 1

a := 1
b := 2
c := 3
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> Not an assign statement above

<sup>2</sup> Not an assign statement above

</td><td valign="top">

</td></tr>
</tbody></table>

## `branch`

> Configurable via `branch-max-lines`

Branch statement (`break`, `continue`, `fallthrough`, `goto`) should only be
cuddled if the block is less than `n` lines where `n` is the value of
`branch-max-statements`.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
for {
    a, err : = SomeFn()
    if err != nil {
        return err
    }

    fmt.Println(a)
    break // 1
}
```

</td><td valign="top">

```go
for {
    a, err : = SomeFn()
    if err != nil {
        return err
    }

    fmt.Println(a)

    break
}

for {
    fmt.Println("hello")
    break
}
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> Block is more than 2 lines so should be a blank line above

</td><td valign="top">

</td></tr>
</tbody></table>

## `decl`

Declarations should never be cuddled. When grouping multiple declarations
together they should be declared in the same group with parenthesis into a
single statement. The benefit of this is that it also aligns the declaration or
assignment increasing readability.

> **NOTE** The fixer can't do smart adjustments if there are comments on the
> same line as the declaration.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
var a string
var b int // 1

const a = 1
const b = 2 // 2

a := 1
var b string // 3

fmt.Println("hello")
var a string // 4
```

</td><td valign="top">

```go
var (
    a string
    b int
)

const (
    a = 1
    b = 2
)

a := 1

var b string

fmt.Println("hello")

var a string
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> Multiple declarations should be grouped to one

<sup>2</sup> Multiple declarations should be grouped to one

<sup>3</sup> Declaration should always have a whitespace above

<sup>4</sup> Declaration should always have a whitespace above

</td><td valign="top">

</td></tr>
</tbody></table>

## `defer`

Deferring execution should only be used directly in the context of what's being
deferred and there should only be one statement above.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
val, closeFn := SomeFn()
val2 := fmt.Sprintf("v-%s", val)
fmt.Println(val)
defer closeFn() // 1

defer fn1()
a := 1
defer fn3() // 2

f, err := os.Open("/path/to/f.txt")
if err != nil {
   return err
}

lines := ReadFile(f)
trimLines(lines)
defer f.Close() // 3
```

</td><td valign="top">

```go
val, closeFn := SomeFn()
defer closeFn()

defer fn1()
defer fn2()
defer fn3()

f, err := os.Open("/path/to/f.txt")
if err != nil {
   return err
}
defer f.Close()

m.Lock()
defer m.Unlock()
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> More than a single statement between `defer` and `closeFn`

<sup>2</sup> `a` is not used in expression

<sup>3</sup> More than a single statement between `defer` and `f.Close`

</td><td valign="top">

</td></tr>
</tbody></table>

## `expr`

Expressions can be multiple things and a big part of them are not handled by
`wsl`. However all function calls are expressions which can be verified.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
a := 1
b := 2
fmt.Println("not b") // 1
```

</td><td valign="top">

```go
a := 1
b := 2

fmt.Println("not b")

a := 1
fmt.Println(a)
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `b` is not used in expression

</td><td valign="top">

</td></tr>
</tbody></table>

## `for`

> Configurable via `allow-first-in-block` to allow cuddling if the variable is
> used _first_ in the block (enabled by default).
>
> Configurable via `allow-whole-block` to allow cuddling if the variable is used
> _anywhere_ in the following block (disabled by default).
>
> See [Configuration](#configuration) for details.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
i := 0
for j := 0; j < 3; j++ { // 1
    fmt.Println(j)
}

a := 0
i := 3
for j := 0; j < i; j++ { // 2
    fmt.Println(j)
}

x := 1
for { // 3
    fmt.Println("hello")
    break
}
```

</td><td valign="top">

```go
i := 0
for j := 0; j < i; j++ {
    fmt.Println(j)
}

a := 0

i := 3
for j := 0; j < i; j++ {
    fmt.Println(j)
}

// Allowed with `allow-first-in-block`
x := 1
for {
    x++
    break
}

// Allowed with `allow-whole-block`
x := 1
for {
    fmt.Println("hello")

    if shouldIncrement() {
        x++
    }
}
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `i` is not used in expression

<sup>2</sup> More than one variable above statement

<sup>3</sup> No variable in expression

</td><td valign="top">

</td></tr>
</tbody></table>

## `go`

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
someFunc := func() {}
go anotherFunc() // 1

x := 1
go func () // 2
    fmt.Println(y)
}()

someArg := 1
go Fn(notArg) // 3
```

</td><td valign="top">

```go
someFunc := func() {}
go someFunc()

x := 1
go func (s string) {
    fmt.Println(s)
}(x)

someArg := 1
go Fn(someArg)
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `someFunc` is not used in expression

<sup>2</sup> `x` is not used in expression

<sup>3</sup> `someArg` is not used in expression

</td><td valign="top">

</td></tr>
</tbody></table>

## `if`

> Configurable via `allow-first-in-block` to allow cuddling if the variable is
> used _first_ in the block (enabled by default).
>
> Configurable via `allow-whole-block` to allow cuddling if the variable is used
> _anywhere_ in the following block (disabled by default).
>
> See [Configuration](#configuration) for details.

`if` statements are one of several block statements (a statement with a block)
that can have some form of expression or condition. To make block context more
readable, only one variable is allowed immediately above the `if` statement and
the variable must be used in the condition (unless configured otherwise).

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
x := 1
if y > 1 { // 1
    fmt.Println("y > 1")
}

a := 1
b := 2
if b > 1 { // 2
    fmt.Println("a > 1")
}

a := 1
b := 2
if a > 1 { // 3
    fmt.Println("a > 1")
}

a := 1
b := 2
if notEvenAOrB() { // 4
    fmt.Println("not a or b")
}

a := 1
x, err := SomeFn() // 5
if err != nil {
    return err
}
```

</td><td valign="top">

```go
x := 1

if y > 1 {
    fmt.Println("y > 1")
}

a := 1

b := 2
if b > 1 {
    fmt.Println("a > 1")
}

b := 2

a := 1
if a > 1 {
    fmt.Println("a > b")
}

a := 1
b := 2

if notEvenAOrB() {
    fmt.Println("not a or b")
}

a := 1

x, err := SomeFn()
if err != nil {
    return err
}

// Allowed with `allow-first-in-block`
x := 1
if xUsedFirstInBlock() {
    x = 2
}

// Allowed with `allow-whole-block`
x := 1
if xUsedLaterInBlock() {
    fmt.Println("will use x later")

    if orEvenNestedWouldWork() {
        x = 3
    }
}
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `x` is not used in expression

<sup>2</sup> More than one variable above statement

<sup>3</sup> `b` is not used in expression and too many statements

<sup>4</sup> No variable in expression

<sup>5</sup> More than one variable above statement

</td><td valign="top">

</td></tr>
</tbody></table>

## `inc-dec`

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
i := 1

if true {
    fmt.Println("hello")
}
i++ // 1

defer func() {
    fmt.Println("hello")
}()
i++ // 2
```

</td><td valign="top">

```go
i := 1
i++

i--
j := i
j++
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> Not an assign or inc/dec statement above

<sup>2</sup> Not an assign or inc/dec statement above

</td><td valign="top">

</td></tr>
</tbody></table>

## `label`

Labels should never be cuddled. Labels in itself is often a symptom of big scope
and split context and because of that should always have an empty line above.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
L1:
    if true {
        _ = 1
    }
L2: // 1
    if true {
        _ = 1
    }
```

</td><td valign="top">

```go
L1:
    if true {
        _ = 1
    }

L2:
    if true {
        _ = 1
    }
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> Labels should always have a whitespabe above

</td><td valign="top">

</td></tr>
</tbody></table>

## `range`

> Configurable via `allow-first-in-block` to allow cuddling if the variable is
> used _first_ in the block (enabled by default).
>
> Configurable via `allow-whole-block` to allow cuddling if the variable is used
> _anywhere_ in the following block (disabled by default).
>
> See [Configuration](#configuration) for details.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
someRange := []int{1, 2, 3}
for _, i := range thisIsNotSomeRange { // 1
    fmt.Println(i)
}

x := 1
for i := range make([]int, 3) { // 2
    fmt.Println("hello")
    break
}

s1 := []int{1, 2, 3}
s2 := []int{3, 2, 1}
for _, v := range s2 { // 3
    fmt.Println(v)
}
```

</td><td valign="top">

```go
someRange := []int{1, 2, 3}

for _, i := range thisIsNotSomeRange {
    fmt.Println(i)
}

someRange := []int{1, 2, 3}
for _, i := range someRange {
    fmt.Println(i)
}

notARange := 1
for i := range returnsRange(notARange) {
    fmt.Println(i)
}

s1 := []int{1, 2, 3}

s2 := []int{3, 2, 1}
for _, v := range s2 {
    fmt.Println(v)
}
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `someRange` is not used in expression

<sup>2</sup> `x` is not used in expression

<sup>3</sup> More than one variable above statement

</td><td valign="top">

</td></tr>
</tbody></table>

## `return`

> Configurable via `branch-max-lines`

Return statements is an important statement that is easiy to miss in larger code
blocks. To better visualize the `return` statement and that the method is
returning it should always be followed by a blank line unless the scope is as
small as `branch-max-lines`.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
func Fn() int {
    x, err := someFn()
    if err != nil {
        panic(err)
    }

    fmt.Println(x)
    return // 1
}
```

</td><td valign="top">

```go
func Fn() int {
    x, err := someFn()
    if err != nil {
        panic(err)
    }

    fmt.Println(x)

    return
}
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> Block is more than 2 lines so should be a blank line above

</td><td valign="top">

</td></tr>
</tbody></table>

## `select`

Identifiers used in case arms of select statements are allowed to be cuddled.

> Configurable via `allow-first-in-block` to allow cuddling if the variable is
> used _first_ in the block (enabled by default).
>
> Configurable via `allow-whole-block` to allow cuddling if the variable is used
> _anywhere_ in the following block (disabled by default).
>
> See [Configuration](#configuration) for details.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
x := 0
select { // 1
case <-time.After(time.Second):
    // ...
case <-stop:
    // ...
}
```

</td><td valign="top">

```go
stop := make(chan struct{})
select {
case <-time.After(time.Second):
    // ...
case <-stop:
    // ...
}

x := 0

select {
case <-time.After(time.Second):
    // ...
case <-stop:
    // ...
}

// Allowed with `allow-whole-block`
x := 1
select {
case <-time.After(time.Second):
    // ...
case <-stop:
    Fn(x)
}
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `x` is not used in expression

</td><td valign="top">

</td></tr>
</tbody></table>

## `send`

Send statements should only be cuddled with a single variable that is used on
the line above.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
a := 1
ch <- 1 // 1

b := 2
<-ch // 2
```

</td><td valign="top">

```go
a := 1
ch <- a

b := 1

<-ch
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `a` is not used in expression

<sup>2</sup> `b` is not used in expression

</td><td valign="top">

</td></tr>
</tbody></table>

## `switch`

In addition to checking the switch condition, switch statements also checks
identifiers in all case arms. If a variable is used in one or more of the case
arms it's allowed to be cuddled.

> Configurable via `allow-first-in-block` to allow cuddling if the variable is
> used _first_ in the block (enabled by default).
>
> Configurable via `allow-whole-block` to allow cuddling if the variable is used
> _anywhere_ in the following block (disabled by default).
>
> See [Configuration](#configuration) for details.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
x := 0
switch y { // 1
case 1:
    // ...
case 2:
    // ...
}


x := 0
y := 1
switch y { // 2
case 1:
    // ...
case 2:
    // ...
}
```

</td><td valign="top">

```go
n := 1
switch n {
case 1:
    // ...
case 2:
    // ...
}

n := 1
switch {
case n < 1:
    // ...
case n > 1:
    // ...
}

x := 0

switch y {
case 1:
    // ...
case 2:
    // ...
}


x := 0

y := 1
switch y {
case 1:
    // ...
case 2:
    // ...
}

// Allowed with `allow-whole-block`
x := 1
switch y {
case 1:
    // ...
case 2:
    fmt.Println(x)
}
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `x` is not used in expression

<sup>2</sup> More than one variable above statement

</td><td valign="top">

</td></tr>
</tbody></table>

## `type-switch`

> Configurable via `allow-first-in-block` to allow cuddling if the variable is
> used _first_ in the block (enabled by default).
>
> Configurable via `allow-whole-block` to allow cuddling if the variable is used
> _anywhere_ in the following block (disabled by default).
>
> See [Configuration](#configuration) for details.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
x := someType()
switch y.(type) { // 1
case int32:
    // ...
case int64:
    // ...
}


x := 0
y := someType()
switch y.(type) {
case int32:
    // ...
case int64:
    // ...
}
```

</td><td valign="top">

```go
x := someType()

switch y.(type) {
case int32:
    // ...
case int64:
    // ...
}


x := 0

y := someType()
switch y.(type) {
case int32:
    // ...
case int64:
    // ...
}

// Allowed with `allow-whole-block`
x := 1
switch y.(type) {
case int32:
    // ...
case int64:
    fmt.Println(x)
}
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `x` is not used in expression

</td><td valign="top">

</td></tr>
</tbody></table>

## `append`

Append enables strict `append` checking where assignments that are
re-assignments with `append` (e.g. `x = append(x, y)`) is only allowed to be
cuddled with other assignments if the `append` uses the variable on the line
above.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
s := []string{}

a := 1
s = append(s, 2) // 1
b := 3
s = append(s, a) // 2
```

</td><td valign="top">

```go
s := []string{}

a := 1
s = append(s, a)

b := 3

s = append(s, 2)
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `a` is not used in append

<sup>2</sup> `b` is not used in append

</td><td valign="top">

</td></tr>
</tbody></table>

## `assign-exclusive`

Assign exclusive does not allow mixing new assignments (`:=`) with
re-assignments (`=`).

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
a := 1
b = 2  // 1
c := 3 // 2
d = 4  // 3
```

</td><td valign="top">

```go
a := 1
c := 3

b = 2
d = 4
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> `a` is not a re-assignment

<sup>2</sup> `b` is not a new assignment

<sup>3</sup> `c` is not a re-assignment

</td><td valign="top">

</td></tr>
</tbody></table>

## `assign-expr`

Assignments are allowed to be cuddled with expressions, primarily to support
mixing assignments and function calls which can often make sense in shorter
flows. By enabling this check `wsl` will ensure assignments are not cuddled with
expressions.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
t1.Fn1()
x := t1.Fn2() // 1
t1.Fn3()
```

</td><td valign="top">

```go
t1.Fn1()

x := t1.Fn2()
t1.Fn3()
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> Line above is not an assignment

</td><td valign="top">

</td></tr>
</tbody></table>

## `err`

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
_, err := SomeFn()

if err != nil { // 1
    return fmt.Errorf("failed to fn: %w", err)
}
```

</td><td valign="top">

```go
_, err := SomeFn()
if err != nil {
    return fmt.Errorf("failed to fn: %w", err)
}
```

</td></tr>

<tr><td valign="top">

<sup>1</sup> Whitespace between error assignment and error checking

</td><td valign="top">

</td></tr>
</tbody></table>

## `leading-whitespace`

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
if true {

    fmt.Println("hello")
}
```

</td><td valign="top">

```go
if true {
    fmt.Println("hello")
}
```

</td></tr>

</tbody></table>

## `trailing-whitespace`

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td valign="top">

```go
if true {
    fmt.Println("hello")

}
```

</td><td valign="top">

```go
if true {
    fmt.Println("hello")
}
```

</td></tr>

</tbody></table>

## Configuration

One shared logic across different checks is the logic around statements
containing a block, i.e. a statement with a following `{}` (e.g. `if`, `for`,
`switch` etc).

`wsl` only allows one statement immediately above and that statement must also
be referenced in the expression in the statement with the block. E.g.

```go
someVariable := true
if someVariable {
    // Here `someVariable` used in the `if` expression is the only variable
    // immediately above the statement.
}
```

This can be configured to be more "laxed" by also allowing a single statement
immediately above if it's used either first in the following block or anywhere
inside the following block.

### `allow-first-in-block`

By setting this to true (default), the variable doesn't have to be used in the
expression itself but is also allowed if it's the first statement in the block
body.

```go
someVariable := 1
if anotherVariable {
    someVariable++
}
```

### `allow-whole-block`

This is similar to `allow-first-in-block` but now allows the lack of whitespace
if it's used anywhere in the following block.

```go
someVariable := 1
if anotherVariable {
    someFn(yetAnotherVariable)

    if stillNotSomeVariable {
        someVariable++
    }
}
```
