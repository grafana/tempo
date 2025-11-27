---
Authors: Mariell Hoversholm (@Proximyst)
Created: 2025 November
Last updated: 2025-11-25
---

# Usage of `gogo/protobuf`

## Summary

This is a retroactive document, intended to document a decision that has already
been made.

We use a fork of Protobuf, [`gogo/protobuf`], rather than upstream Protobuf
libraries. This is fuelled by one major reason: upstream doesn't behave well
enough in terms of performance. This is a decision that we inherited from
OpenTelemetry, see also [this issue](https://github.com/open-telemetry/opentelemetry-collector/issues/7095).

The primary problem with upstream Protobuf, as maintained by Google, is that it
generates _a lot_ of memory allocations from its lack of supporting embedding
structs as values rather than pointers. I.e. we often want `[]Value` rather than
`[]*Value` because we already know all the values are non-null inside.

## Alternatives

As of writing, there are many alternatives that may be considered in an eventual
migration in the future, including but not limited to:

* [vtprotobuf](https://github.com/planetscale/vtprotobuf)
* [csproto](https://github.com/CrowdStrike/csproto)
* [hyperpb](https://github.com/bufbuild/hyperpb-go)
* [pdatagen](https://github.com/open-telemetry/opentelemetry-collector/tree/974da01f71487422c02fadadb8f66147162fcb14/internal/cmd/pdatagen)
  * Note that this is an internal tool to OpenTelemetry. It may not be easy to
    adapt to our needs.

We notably use the following features, which are required in a potential
replacement:

* `gogoproto.customtype`: this lets us represent a field with a different type
  than what is strictly prescribed by the Protobuf schema. For example, this
  lets us represent a `bytes` type as `[16]byte` for a `uuid.UUID`; this saves
  on memory pressure.
* `gogoproto.nullable`: this lets us embed values as value structs rather than
  pointers. E.g. `time.Time` rather than `*time.Time`. We know these values will
  never be `nil` OR that the zero-type is sufficient to indicate its `nil`-ness,
  and so we can save ourselves some stack/heap pressure.
* `gogoproto.stdtime`: use `time.Time` rather than `timestamppb.Timestamp`.
  This makes for simpler interop with Go.
* `gogoproto.jsontag`: upstream Protobuf will output JSON tags with snake case.
  We use camel-case, so e.g. `block_id` => `blockID` (upstream would use
  `block_id`).
  * Upstream Protobuf can replicate this with a field-level `[json_name]` tag.

[`gogo/protobuf`]: https://github.com/gogo/protobuf
