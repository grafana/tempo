# Contributing

This guide is specific to the OpenTelemetry Transformation Language.  All guidelines in [Collector Contrib's CONTRIBUTING.MD](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md) must also be followed.

## General Guidelines

- Changes to the OpenTelemetry Transformation Language should be made independent of any component that depend on the package.  Whenever possible, try not to submit PRs that change both the OTTL and a dependent component.  Instead, submit a PR that updates the OTTL and then, once merged, update the component as needed.

## Adding New Editors/Converters

Before raising a PR with a new Editor or Converter, raise an issue to verify its acceptance. While acceptance is strongly specific to a specific use case, consider these guidelines for early assessment.

Your proposal likely will be accepted if:

- The proposed functionality is missing,
- The proposed solution significantly improves user experience and readability for very common use cases,
- The proposed solution is more performant in cases where it is possible to achieve the same result with existing options.
- The proposed solution makes use of packages from the Go standard library to offer functionality possible through an existing option in a more standard or reliable manner.

It will be up for discussion if:

- Your proposal solves an issue that can be achieved in another way but does not improve user experience or performance.
- The proposed functionality is not obviously applicable to the needs of a significant number of OTTL users.
- Your proposal extracts data into a structure with enumerable keys or values and OpenTelemetry semantic conventions do not cover the shape or values for this data.

Your proposal likely won't be accepted if:

- User experience is worse and assumes a highly technical user,
- The performance of your proposal very negatively affects the processing pipeline.

As with code, OTTL aims for readability first. This means:

- Using short, meaningful, and descriptive names,
- Ensuring naming consistency across Editors and Converters,
- Avoiding deep nesting to achieve desired transformations,
- Ensuring Editors and Converters have a single responsibility.

### Implementation guidelines

All new functions are experimental. They are not covered by OTTL's stability guarantees until they are promoted (see [Promoting Experimental Functions](#promoting-experimental-functions)).

All new functions must be added via a new file.  Function files must start with `func_`.  Functions must be placed in `xottl/ottlfuncs`.  A new function must:

- Be marked experimental by passing `ottl.WithExperimental[K]()` to `ottl.NewFactory`. The parser rejects experimental functions unless the `pkg.ottl.functions.enableExperimental` feature gate is enabled.
- Be added to `ExperimentalConverters` in `xottl/ottlfuncs/functions.go` if it is a Converter.
- Be documented in the [experimental functions README](./xottl/ottlfuncs/README.md), including a note that it is experimental and requires the [`pkg.ottl.functions.enableExperimental`](./documentation.md#feature-gates) feature gate.

Unit tests must be added for all new functions.  Unit test files must start with `func_` and end in `_test`.  Unit tests must be placed in the same directory as the function. Functions that are not specific to a context should be tested independently of any specific context. Functions that are specific to a context, such as `IsRootSpan`, should be tested against that context. End-to-end tests must be added in the `xottl/e2e` directory and must enable the `pkg.ottl.functions.enableExperimental` feature gate, for example with `testutil.SetFeatureGateForTest(t, metadata.PkgOttlFunctionsEnableExperimentalFeatureGate, true)`.

#### Naming and Parameter Guidelines

Functions should be named and formatted according to the following standards.

- Function names MUST start with a verb unless it is a Factory that creates a new type.
- Converters MUST be UpperCamelCase.
- Function names that contain multiple words MUST separate those words with `_`.
- Functions that interact with multiple items MUST have plurality in the name. Ex: `truncate_all`, `keep_keys`, `replace_all_matches`.
- Functions that interact with a single item MUST NOT have plurality in the name. If a function would interact with multiple items due to a condition, like `where`, it is still considered singular. Ex: `set`, `delete`, `replace_match`.
- Functions that change a specific target MUST set the target as the first parameter.

### Promoting Experimental Functions

An experimental function may be promoted once it meets the [promotion criteria](./ottlfuncs/README.md#promotion-to-standard). Promotion freezes the function's signature for the life of OTTL `1.x`, so any signature changes must be made before it is promoted. To promote a function:

1. Move its `func_` file and unit tests from `xottl/ottlfuncs` to `ottlfuncs`, and its end-to-end tests from `xottl/e2e` to `e2e`.
2. Remove `ottl.WithExperimental[K]()` from its factory.
3. Remove it from `ExperimentalConverters` and add it to `ottlfuncs/functions.go`: Editors go in `StandardFuncs` and Converters go in `converters`.
4. Move its documentation to the [standard functions README](./ottlfuncs/README.md) and remove the experimental note.
5. Run `make update-ottl-signatures` from `pkg/ottl` and commit the updated golden file.

## New Values

When adding new values to the grammar you must:

1. Update the `Value` struct with the new value.  This may also mean adding new token(s) to the lexer.
2. Update `NewFunctionCall` to be able to handle calling functions with this new value.
3. Update `NewGetter` to be able to handle the new value.
4. Add new unit tests.
