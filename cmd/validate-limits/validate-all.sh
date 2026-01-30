#!/bin/bash

# Validate all overrides.json files in config_buckets directory
# Usage: ./validate-all.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_BUCKETS_DIR="./config_buckets"
OUTPUT_FILE="./validate-limits-results.txt"

echo "Validating all overrides.json files in $CONFIG_BUCKETS_DIR"
echo "Output will be written to: $OUTPUT_FILE"
echo ""

# Clear output file
> "$OUTPUT_FILE"

find "$CONFIG_BUCKETS_DIR" -name "overrides.json" | while read -r f; do
    output=$("$SCRIPT_DIR/validate-limits" "$f" 2>&1)
    exit_code=$?

    if [ $exit_code -ne 0 ]; then
        echo "FAILED: $f" | tee -a "$OUTPUT_FILE"
        echo "  $output" | tee -a "$OUTPUT_FILE"
    else
        echo "OK: $f" | tee -a "$OUTPUT_FILE"
    fi
done

echo ""
echo "Results written to: $OUTPUT_FILE"
