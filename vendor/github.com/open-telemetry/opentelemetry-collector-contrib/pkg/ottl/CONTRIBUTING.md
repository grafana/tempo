# Contributing

This guide is specific to the OpenTelemetry Transformation Language.  All guidelines in [Collector Contrib's CONTRIBUTING.MD](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md) must also be followed.

## General Guidelines

- Changes to the OpenTelemetry Transformation Language should be made independent of any component that depend on the package.  Whenever possible, try not to submit PRs that change both the OTTL and a dependent component.  Instead, submit a PR that updates the OTTL and then, once merged, update the component as needed.

## New Values

When adding new values to the grammar you must:

1. Update the `Value` struct with the new value.  This may also mean adding new token(s) to the lexer.
2. Update `NewFunctionCall` to be able to handle calling functions with this new value.
3. Update `NewGetter` to be able to handle the new value.
4. Add new unit tests.

## New Functions

All new functions must be added via a new file.  Function files must start with `func_`.  Functions must be placed in `ottlfuncs`.

Unit tests must be added for all new functions.  Unit test files must start with `func_` and end in `_test`.  Unit tests must be placed in the same directory as the function.  Functions that are not specific to a pipeline should be tested independently of any specific pipeline. Functions that are specific to a pipeline should be tests against that pipeline. End-to-end tests must be added in the `e2e` directory.

Function names should follow the [Function Syntax Guidelines](ottlfuncs/README.md#function-syntax)

