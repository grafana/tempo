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

All new functions must be added via a new file.  Function files must start with `func_`.  Functions must be placed in `ottlfuncs`.

Unit tests must be added for all new functions.  Unit test files must start with `func_` and end in `_test`.  Unit tests must be placed in the same directory as the function.  Functions that are not specific to a pipeline should be tested independently of any specific pipeline. Functions that are specific to a pipeline should be tests against that pipeline. End-to-end tests must be added in the `e2e` directory.

#### Naming and Parameter Guidelines

Functions should be named and formatted according to the following standards.

- Function names MUST start with a verb unless it is a Factory that creates a new type.
- Converters MUST be UpperCamelCase.
- Function names that contain multiple words MUST separate those words with `_`.
- Functions that interact with multiple items MUST have plurality in the name. Ex: `truncate_all`, `keep_keys`, `replace_all_matches`.
- Functions that interact with a single item MUST NOT have plurality in the name. If a function would interact with multiple items due to a condition, like `where`, it is still considered singular. Ex: `set`, `delete`, `replace_match`.
- Functions that change a specific target MUST set the target as the first parameter.

## New Values

When adding new values to the grammar you must:

1. Update the `Value` struct with the new value.  This may also mean adding new token(s) to the lexer.
2. Update `NewFunctionCall` to be able to handle calling functions with this new value.
3. Update `NewGetter` to be able to handle the new value.
4. Add new unit tests.
