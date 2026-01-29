#!/usr/bin/env bash

set -e

run_tests() {
  golangci-lint run ./...
  SKIP_GO_TESTS=1 ./tests.sh
}

run_tests
