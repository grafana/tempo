# DRAIN Test Data

This directory contains test fixtures for the DRAIN log parsing algorithm implementation.

## Overview

DRAIN is an online log parsing algorithm that clusters log messages and extracts
log patterns by replacing variable parts with wildcards (`<_>`). These test
fixtures validate that the algorithm correctly identifies patterns across
various span names.

## Directory Structure

### Input Files (`*.json`)

These files contain arrays of strings, each representing a span name.

### Output Files (`*.drain`)

These files contain the results of running DRAIN on the input files. The output
is a map of the original span name to the pattern that was extracted.

### Tools

#### `generate-random-span-names`

This tool generates random span names based on a set of features. The files in
this directory were generated from scratch by looking at real data and
extracting patterns manually. The tool can then be used to generate more data
which is free from real details which may be sensitive or proprietary.

dev01.json:

```sh
go run ./pkg/drain/testdata/generate-random-span-names \
  -features "<word>.<word>" \
  -features "*<word>.<word>.<word>" \
  -features "/<word-list>" \
  -features "<method> /<word-list>/<alphanumeric>" \
  -features "<method> /<word-list>/<base64>" \
  -features "<method> /<word-list>?h=<md5>" \
  -features "<sql>" \
  -features "/<word>.<word>/<word>" \
  -features "<word> <word> <word> <word>" \
  -features "batch job <number>: done"  | tee ./pkg/drain/testdata/dev01.json
```

ops.json:

```sh
go run ./pkg/drain/testdata/generate-random-span-names -seed 1 \
  -features "/<word-list>" \
  -features "GET /*/*/*/*/<word>.js" \
  -features "<method> /<word-list>/<alphanumeric>" \
  -features "<method> /api/v1/accesspolicies/grafana.com/<word-list>"  | tee ./pkg/drain/testdata/ops.json
```

prod1.json:

```
go run ./pkg/drain/testdata/generate-random-span-names -seed 1 \
  -features "/<word-list>" \
  -features "GET /*/*/*/*/<word>.js" \
  -features "<method> /<word-list>/<alphanumeric>" \
  -features "<sqlkeyword> <word>" \
  -features "queue://<alphanumeric>/ABC.REQUEST?name=<number> send" \
  -features "queue://<alphanumeric>/ABC.REPLY?name=<number> send" \
  -features "<sql>" \
  -features "<method> /api/v1/accesspolicies/grafana.com/<word-list>"  | tee ./pkg/drain/testdata/prod1.json
```

prod2.json:

```
go run ./pkg/drain/testdata/generate-random-span-names -seed 2 \
  -features "/<word-list>" \
  -features "<method> /<word-list>/{user}/<word>" \
  -features "<method> /<word-list>/Foo_<number>.<number>.<number>/<word>" \
  -features "<sqlkeyword> <word>.<word>" \
  -features "<sqlkeyword> <word>.<word>,cn=org,dc=place,dc=city" \
  -features "TRAMPOLINE COMMAND GO: commence jumping: <uuid>"  | tee ./pkg/drain/testdata/prod2.json
```

prod3.json:

```
go run ./pkg/drain/testdata/generate-random-span-names -seed 3 \
  -features "/<word-list>" \
  -features "<method> /<word-list>/{user}/<word>" \
  -features "Async - <method> /<word-list>/{user}/<word>" \
  -features "/bicycle/tires/bolts[<number>]/installed" \
  -features "[<word>-<word>]: ProductListing_<word><word>" \
  -features "<word><number>-<word>-events-processor send" \
  -features "CALL <word>_main.data.customer.com.<word>" \
  -features "<sqlkeyword> <word>.<word>" \
  -features "fetch GET http://api.namespace.svc.cluster.local/api/config?input=abc%3D<word>%26def%3D<word>%26height%3D<number>" \
  -features "fetch GET http://api.namespace.svc.cluster.local/api/apps/shared/inband/service.method?input=abc%3D<word>%26def%3D<word>%26height%3D<number>" \
  -features "fetch GET https://customer.com/users/v2/<uuid>" \
  -features "fetch PUT https://customer.com/sessions/v1/<base64>" | tee ./pkg/drain/testdata/prod3.json
```

### `update-fixtures`

This tool updates the output files based on the input files. It should only be
run manually when you are sure the drain algorithm is working correctly via
other tests. The test fixtures make it easier to see the impact of changes to
the drain algorithm.

```sh
go run ./pkg/drain/testdata/update-fixtures
```