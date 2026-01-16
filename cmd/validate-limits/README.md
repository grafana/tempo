# validate-limits

## Usage

```bash
# Build
go build -o validate-limits ./cmd/validate-limits/

# Run
./validate-limits limits.json
```

The tool:

- Reads a JSON file containing a Limits struct
- Validates it using the existing `overridesValidator.Validate()` logic
- Only accepts `"k6-cloud-insights"` as a valid forwarder (hardcoded)
- Exits with code 0 on success, code 1 on validation failure
