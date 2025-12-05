---
Authors: Mariell Hoversholm (@Proximyst)
Created: 2025 November
Last updated: 2025-11-25
---

# Usage of `gogo/protobuf`

## Summary

This is a retroactive document, intended to document a decision that has already
been made.

We use a fork of Protobuf, [`gogo/protobuf`], rather than the official
`google.golang.org/protobuf` or `github.com/golang/protobuf` libraries.
This is fuelled by one major reason: the official libraries don't behave well
enough in terms of performance. This is a decision that we inherited from
OpenTelemetry, see also [this issue](https://github.com/open-telemetry/opentelemetry-collector/issues/7095).

The OpenTelemetry issue linked cites the large amount of memory allocations by
the generated code from the official libraries as a particularly important
reason for this decision. This is seen in generated code as use of `[]*Value`
pointer-values instead of `[]Value` value-types.

## Alternatives

As of writing, there are many alternatives that exist, to varying degrees of
sufficiency. When we look at migration, read up on the following projects, as
well as anything that is relevant at that future time:

- [vtprotobuf](https://github.com/planetscale/vtprotobuf)
- [csproto](https://github.com/CrowdStrike/csproto)
- [hyperpb](https://github.com/bufbuild/hyperpb-go)
- [pdatagen](https://github.com/open-telemetry/opentelemetry-collector/tree/974da01f71487422c02fadadb8f66147162fcb14/internal/cmd/pdatagen)
  - Note that this is an internal tool to OpenTelemetry. It may not be easy to
    adapt to our needs.
  - pdatagen does not emit Protobuf code, rather it is used to create wrappers
    from Protobuf objects. This is useful for them, but exactly how useful it is
    in our project is unclear; evaluate it with the code-base as it exists at
    the time of reading.

Our primary requirements are:

- Similar or better performance characteristics than `gogo/protobuf` in
  benchmarks that simulate real-world deployment scenarios. Small regressions
  can be acceptable; open a proposal with numbers to decide whether a solution
  is sufficient.
- The new solution must be either backwards-compatible with the existing encoded
  data, OR have a clear and simple migration path. Ideally, the migration path
  does not entail any work for operators, although this can be considered given
  a thorough proposal.

[`gogo/protobuf`]: https://github.com/gogo/protobuf
