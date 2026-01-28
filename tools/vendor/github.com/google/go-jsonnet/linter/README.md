# jsonnet-lint

This is a linter for Jsonnet. It is alpha stage, but it should be already useful.

## Features

The linter detect the following kinds of issues:
* "Type" problems, such as:
    * Accessing nonexistent fields
    * Calling a function with a wrong number of arguments or named arguments
    which do not match the parameters
    * Trying to call a value which is not a function
    * Trying to index a value which is not an object, array or a string
* Unused variables
* Endlessly looping constructs, which are always invalid, but often appear  as a result of confusion about language semantics (e.g. local x = x + 1)
* Anything that is statically detected during normal execution, such as syntax errors and undeclared variables.

## Usage

`jsonnet-lint [options] <filename>`

## Design

### Goals

The purpose of the linter is to aid development by providing quick and clear feedback about simple problems. With that in mind I defined the following goals:
- It should find common problems, especially the kinds resulting from typos, trivial omissions and issues resulting from misunderstanding of the semantics.
- It should find problems in the parts of code which are not reached by the tests (especially
important due to the lazy evaluation).
- It must be practical to use with the existing Jsonnet code, without any need for modification.
- It must be fast enough so it is practical to always run the linter before execution during development. The overhead required to run the linter prior to running the program in real world conditions should be comparable with parsing and desugaring.
- It must be conservative regarding the reported problems. False negatives are preferable to false positives. False positives are allowed as long as they relate to code which is going to be confusing for humans to read or if they can be worked around easily while preserving readability.
- Its results must be stable, i.e. trivial changes such as changing the order of variables in local expressions should not change the result nontrivially.
- It must preserve the abstractions. Validity of the definitions should not depend on their use. In particular calling functions with specific arguments or accessing fields of objects should not cause errors in their definitions.
- It should be possible to explicitly silence individual errors, so that occasional acknowledged false positives do not distract the users. This is espcially important if a clean pass is enforced in  Continuous Integration.

### Rules

The above goals naturally lead to the some more specific code-level rules which all analyses must obey:

- All expressions should be checked, even the provably dead code.
- Always consider both branches of the `if` expression possible (even if the condition is trivially always true or always false).
- Correctness of a function definition should not depend on how it is used. In particular
when analyzing the definition assume that function arguments can take arbitrary values.
- Correctness of an object definition should not depend on how it is used. In particular when analyzing the definition assume that the object may be part of an arbitrary inheritance chain
