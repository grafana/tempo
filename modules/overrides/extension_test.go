package overrides

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"
)

func TestRegisterExtension_PanicsOnDuplicate(t *testing.T) {
	resetRegistryForTesting(t)
	RegisterExtension(&testExtension{})
	assert.Panics(t, func() { RegisterExtension(&testExtension{}) })
}

func TestRegisterExtension_TypedGetter(t *testing.T) {
	resetRegistryForTesting(t)
	get := RegisterExtension(&testExtension{})

	fieldB := 99
	o := Overrides{
		Extensions: map[string]any{"test_extension": &testExtension{FieldA: "hello", FieldB: &fieldB}},
	}
	ext := get(&o)
	require.NotNil(t, ext)
	assert.Equal(t, "hello", ext.FieldA)
	assert.Equal(t, 99, *ext.FieldB)

	// Nil Overrides and empty Extensions both return zero value.
	assert.Nil(t, get(nil))
	assert.Nil(t, get(&Overrides{}))
}

func TestOverridesExtension_MarshalJSON(t *testing.T) {
	t.Run("overrides", func(t *testing.T) {
		resetRegistryForTesting(t)
		get := RegisterExtension(&testExtension{})

		fieldB := 42
		o := Overrides{}
		o.Ingestion.MaxLocalTracesPerUser = 1000
		o.Extensions = map[string]any{
			"test_extension": &testExtension{FieldA: "custom", FieldB: &fieldB},
		}

		b, err := json.Marshal(&o)
		require.NoError(t, err)

		var o2 Overrides
		require.NoError(t, json.Unmarshal(b, &o2))

		assert.Equal(t, 1000, o2.Ingestion.MaxLocalTracesPerUser)
		ext := get(&o2)
		require.NotNil(t, ext)
		assert.Equal(t, "custom", ext.FieldA)
		assert.Equal(t, 42, *ext.FieldB)
	})

	t.Run("legacy overrides", func(t *testing.T) {
		resetRegistryForTesting(t)
		RegisterExtension(&testExtension{})

		fieldB := 7
		o := Overrides{}
		o.Ingestion.MaxLocalTracesPerUser = 500
		o.Extensions = map[string]any{
			"test_extension": &testExtension{FieldA: "flat_val", FieldB: &fieldB},
		}
		l := o.toLegacy()

		// After toLegacy, Extensions holds the typed instance; flat keys appear only when marshaled.
		extInLegacy, ok := l.Extensions["test_extension"].(*testExtension)
		require.True(t, ok, "expected typed test_extension in LegacyOverrides.Extensions")
		assert.Equal(t, "flat_val", extInLegacy.FieldA)

		b, err := json.Marshal(&l)
		require.NoError(t, err)

		// The nested extension key must be replaced by its flat keys in JSON.
		var m map[string]any
		require.NoError(t, json.Unmarshal(b, &m))
		assert.Equal(t, "flat_val", m["test_extension_field_a"], "flat key must appear in JSON")
		assert.Nil(t, m["test_extension"], "nested key must not appear in JSON")

		// Unmarshal the JSON back to LegacyOverrides; processLegacyExtensions converts flat keys
		// to a typed instance under the nested key.
		var l2 LegacyOverrides
		require.NoError(t, json.Unmarshal(b, &l2))
		ext2, ok2 := l2.Extensions["test_extension"].(*testExtension)
		require.True(t, ok2, "expected typed test_extension in l2.Extensions after unmarshal")
		assert.Equal(t, "flat_val", ext2.FieldA)
	})
}

func TestOverridesExtension_UnmarshalJSON(t *testing.T) {
	t.Run("overrides", func(t *testing.T) {
		resetRegistryForTesting(t)
		get := RegisterExtension(&testExtension{})

		input := `{
			"ingestion": {"max_traces_per_user": 1000},
			"test_extension": {"field_a": "from_json", "field_b": 55}
		}`
		var o Overrides
		require.NoError(t, json.Unmarshal([]byte(input), &o))

		assert.Equal(t, 1000, o.Ingestion.MaxLocalTracesPerUser)
		ext := get(&o)
		require.NotNil(t, ext)
		assert.Equal(t, "from_json", ext.FieldA)
		assert.Equal(t, 55, *ext.FieldB)
	})

	t.Run("overrides defaults applied", func(t *testing.T) {
		resetRegistryForTesting(t)
		get := RegisterExtension(&testExtension{})

		// field_b is omitted — the default (pointer to 0) must be used.
		input := `{"test_extension": {"field_a": "only_a"}}`
		var o Overrides
		require.NoError(t, json.Unmarshal([]byte(input), &o))

		ext := get(&o)
		require.NotNil(t, ext)
		assert.Equal(t, "only_a", ext.FieldA)
		require.NotNil(t, ext.FieldB, "default FieldB must be applied")
		assert.Equal(t, 0, *ext.FieldB)
	})

	t.Run("overrides unregistered key", func(t *testing.T) {
		resetRegistryForTesting(t)

		input := `{"ingestion": {"max_traces_per_user": 1}, "unknown_ext": {"x": 1}}`
		var o Overrides
		err := json.Unmarshal([]byte(input), &o)
		require.ErrorContains(t, err, "unknown extension key")
	})

	t.Run("overrides validate error", func(t *testing.T) {
		resetRegistryForTesting(t)
		RegisterExtension(&testExtension{})

		// field_a is empty — Validate must reject it.
		input := `{"test_extension": {"field_a": ""}}`
		var o Overrides
		err := json.Unmarshal([]byte(input), &o)
		require.ErrorContains(t, err, "field_a cannot be empty")
	})

	t.Run("unknown field rejected", func(t *testing.T) {
		resetRegistryForTesting(t)
		RegisterExtension(&testExtension{})

		// A typo in an extension field name must be rejected, not silently dropped.
		input := `{"test_extension": {"field_a": "ok", "field_b": 1, "typo_field": "bad"}}`
		var o Overrides
		err := json.Unmarshal([]byte(input), &o)
		require.ErrorContains(t, err, "typo_field")
	})

	t.Run("legacy overrides", func(t *testing.T) {
		resetRegistryForTesting(t)
		RegisterExtension(&testExtension{})

		// LegacyOverrides JSON may contain flat extension keys; processLegacyExtensions converts
		// them to typed instances under the nested key immediately at unmarshal time.
		input := `{"max_traces_per_user": 1000, "test_extension_field_a": "from_legacy_json"}`
		var l LegacyOverrides
		require.NoError(t, json.Unmarshal([]byte(input), &l))
		assert.Equal(t, 1000, l.MaxLocalTracesPerUser)
		ext, ok := l.Extensions["test_extension"].(*testExtension)
		require.True(t, ok, "expected typed test_extension in l.Extensions after unmarshal")
		assert.Equal(t, "from_legacy_json", ext.FieldA)
		require.NotNil(t, ext.FieldB, "default for FieldB must be applied")
		assert.Equal(t, 0, *ext.FieldB)
	})
}

func TestOverridesExtension_MarshalYAML(t *testing.T) {
	t.Run("overrides", func(t *testing.T) {
		resetRegistryForTesting(t)
		get := RegisterExtension(&testExtension{})

		fieldB := 3
		o := Overrides{}
		o.Ingestion.MaxLocalTracesPerUser = 1000
		o.Extensions = map[string]any{
			"test_extension": &testExtension{FieldA: "yaml_val", FieldB: &fieldB},
		}

		b, err := yaml.Marshal(&o)
		require.NoError(t, err)

		// The marshaled YAML must contain the nested extension key.
		assert.Contains(t, string(b), "test_extension:")

		// Round-trip: unmarshal recovers the extension (Overrides.UnmarshalYAML calls processExtensions).
		var o2 Overrides
		require.NoError(t, yaml.Unmarshal(b, &o2))

		assert.Equal(t, 1000, o2.Ingestion.MaxLocalTracesPerUser)
		ext := get(&o2)
		require.NotNil(t, ext)
		assert.Equal(t, "yaml_val", ext.FieldA)
		assert.Equal(t, 3, *ext.FieldB)
	})

	t.Run("legacy overrides", func(t *testing.T) {
		resetRegistryForTesting(t)
		RegisterExtension(&testExtension{})

		fieldB := 8
		o := Overrides{}
		o.Ingestion.MaxLocalTracesPerUser = 500
		o.Extensions = map[string]any{
			"test_extension": &testExtension{FieldA: "legacy_yaml", FieldB: &fieldB},
		}
		l := o.toLegacy()

		b, err := yaml.Marshal(&l)
		require.NoError(t, err)

		// Flat keys must appear at the top level; nested key must not.
		yamlStr := string(b)
		assert.Contains(t, yamlStr, "test_extension_field_a:")
		assert.NotContains(t, yamlStr, "test_extension:")
	})
}

func TestOverridesExtension_UnmarshalYAML(t *testing.T) {
	t.Run("overrides", func(t *testing.T) {
		resetRegistryForTesting(t)
		get := RegisterExtension(&testExtension{})

		input := `
ingestion:
  max_traces_per_user: 1000
test_extension:
  field_a: from_yaml
  field_b: 11
`
		var o Overrides
		require.NoError(t, yaml.Unmarshal([]byte(input), &o))

		assert.Equal(t, 1000, o.Ingestion.MaxLocalTracesPerUser)
		ext := get(&o)
		require.NotNil(t, ext)
		assert.Equal(t, "from_yaml", ext.FieldA)
		assert.Equal(t, 11, *ext.FieldB)
	})

	t.Run("Overrides_strict_decoder_absorbs_extension_key", func(t *testing.T) {
		resetRegistryForTesting(t)
		RegisterExtension(&testExtension{})

		input := `
ingestion:
  max_traces_per_user: 500
test_extension:
  field_a: strict_ok
`
		var o Overrides
		decoder := yaml.NewDecoder(strings.NewReader(input))
		decoder.SetStrict(true)
		require.NoError(t, decoder.Decode(&o), "strict YAML decoder must not error on registered extension keys")
	})

	t.Run("overrides unregistered key", func(t *testing.T) {
		resetRegistryForTesting(t)

		input := `
ingestion:
  max_traces_per_user: 1
unknown_ext:
  x: 1
`
		var o Overrides
		err := yaml.Unmarshal([]byte(input), &o)
		require.ErrorContains(t, err, "unknown extension key")
	})

	t.Run("legacy overrides", func(t *testing.T) {
		resetRegistryForTesting(t)
		get := RegisterExtension(&testExtension{})

		input := `
max_traces_per_user: 1000
test_extension_field_a: from_legacy_yaml
`
		var l LegacyOverrides
		require.NoError(t, yaml.Unmarshal([]byte(input), &l))

		assert.Equal(t, 1000, l.MaxLocalTracesPerUser)
		// processLegacyExtensions converts the flat key to a typed instance at unmarshal time.
		lExt, ok := l.Extensions["test_extension"].(*testExtension)
		require.True(t, ok, "expected typed test_extension in l.Extensions after unmarshal")
		assert.Equal(t, "from_legacy_yaml", lExt.FieldA)
		// field_b was not in the YAML; the default (pointer to 0) must be used.
		require.NotNil(t, lExt.FieldB)
		assert.Equal(t, 0, *lExt.FieldB)

		// toNewLimits copies typed extensions; processExtensions validates them.
		o := l.toNewLimits()
		require.NoError(t, processExtensions(o))

		ext := get(o)
		require.NotNil(t, ext)
		assert.Equal(t, "from_legacy_yaml", ext.FieldA)
		require.NotNil(t, ext.FieldB)
		assert.Equal(t, 0, *ext.FieldB)
	})
}

func TestExtension_FullLegacyRoundTrip_perTenantOverrides(t *testing.T) {
	resetRegistryForTesting(t)
	get := RegisterExtension(&testExtension{})

	input := `
overrides:
  tenant-1:
    max_traces_per_user: 1000
    test_extension_field_a: roundtrip_val
`
	var pto perTenantOverrides
	decoder := yaml.NewDecoder(strings.NewReader(input))
	decoder.SetStrict(true)
	require.NoError(t, decoder.Decode(&pto))
	require.Equal(t, ConfigTypeLegacy, pto.ConfigType)

	limits := pto.TenantLimits["tenant-1"]
	require.NotNil(t, limits)
	assert.Equal(t, 1000, limits.Ingestion.MaxLocalTracesPerUser)
	ext := get(limits)
	require.NotNil(t, ext)
	assert.Equal(t, "roundtrip_val", ext.FieldA)
}

func TestExtension_FullNewFormatRoundTrip_perTenantOverrides(t *testing.T) {
	resetRegistryForTesting(t)
	get := RegisterExtension(&testExtension{})

	input := `
overrides:
  tenant-1:
    ingestion:
      max_traces_per_user: 2000
    test_extension:
      field_a: new_format_val
      field_b: 5
`
	var pto perTenantOverrides
	decoder := yaml.NewDecoder(strings.NewReader(input))
	decoder.SetStrict(true)
	require.NoError(t, decoder.Decode(&pto))
	require.Equal(t, ConfigTypeNew, pto.ConfigType)

	limits := pto.TenantLimits["tenant-1"]
	require.NotNil(t, limits)
	assert.Equal(t, 2000, limits.Ingestion.MaxLocalTracesPerUser)
	ext := get(limits)
	require.NotNil(t, ext)
	assert.Equal(t, "new_format_val", ext.FieldA)
	assert.Equal(t, 5, *ext.FieldB)
}

func TestExtension_JSONRoundTrip_Overrides(t *testing.T) {
	resetRegistryForTesting(t)
	get := RegisterExtension(&testExtension{})

	fieldB := 13
	o := Overrides{}
	o.Ingestion.MaxLocalTracesPerUser = 777
	o.Extensions = map[string]any{
		"test_extension": &testExtension{FieldA: "json_rt", FieldB: &fieldB},
	}

	b, err := json.Marshal(&o)
	require.NoError(t, err)

	var o2 Overrides
	require.NoError(t, json.Unmarshal(b, &o2))

	assert.Equal(t, 777, o2.Ingestion.MaxLocalTracesPerUser)
	ext := get(&o2)
	require.NotNil(t, ext)
	assert.Equal(t, "json_rt", ext.FieldA)
	assert.Equal(t, 13, *ext.FieldB)
}

func TestExtension_LegacyConversionRoundTrip(t *testing.T) {
	resetRegistryForTesting(t)
	get := RegisterExtension(&testExtension{})

	fieldB := 21
	// Build an Overrides with a typed extension, convert to legacy and back.
	o := Overrides{}
	o.Extensions = map[string]any{
		"test_extension": &testExtension{FieldA: "converted", FieldB: &fieldB},
	}
	l := o.toLegacy()

	// After toLegacy(), Extensions holds the typed instance keyed by nested key.
	// Flat keys only appear in the marshaled wire format.
	extInLegacy, ok := l.Extensions["test_extension"].(*testExtension)
	require.True(t, ok, "expected typed test_extension in LegacyOverrides.Extensions")
	assert.Equal(t, "converted", extInLegacy.FieldA)

	// When marshaled to JSON/YAML, the typed instance must produce flat legacy keys.
	b, err := json.Marshal(&l)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, "converted", m["test_extension_field_a"], "flat key must appear in JSON")
	assert.Nil(t, m["test_extension"], "nested key must not appear in JSON")

	// Round-trip through toNewLimits recovers the typed extension.
	o2 := l.toNewLimits()
	require.NoError(t, processExtensions(o2))

	ext := get(o2)
	require.NotNil(t, ext)
	assert.Equal(t, "converted", ext.FieldA)
	assert.Equal(t, 21, *ext.FieldB)
}

func TestRegisterExtension_PanicsOnKnownFieldConflict(t *testing.T) {
	resetRegistryForTesting(t)
	// "ingestion" is a known Overrides JSON field; registering an extension with
	// that key must panic rather than silently cause data loss during marshal.
	assert.Panics(t, func() {
		RegisterExtension(&conflictingExtension{})
	})
}

func TestExtensionValidationError_NotSwallowedByLegacyFallback(t *testing.T) {
	resetRegistryForTesting(t)
	RegisterExtension(&testExtension{})

	// New-format config with a registered extension that fails validation (field_a is empty).
	// The error must propagate to the caller rather than being silently swallowed by the
	// legacy-format fallback path.
	input := `
overrides:
  tenant-1:
    ingestion:
      max_traces_per_user: 1000
    test_extension:
      field_a: ""
`
	var pto perTenantOverrides
	decoder := yaml.NewDecoder(strings.NewReader(input))
	decoder.SetStrict(true)
	err := decoder.Decode(&pto)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field_a cannot be empty", "validation error must not be swallowed by legacy fallback")
}

func TestUnknownExtensionKey_NewFormat_BlocksLegacyFallback(t *testing.T) {
	resetRegistryForTesting(t)

	input := `
overrides:
  tenant-1:
    ingestion:
      max_traces_per_user: 1000
    my_unregistered_ext:
      field: value
`
	var pto perTenantOverrides
	decoder := yaml.NewDecoder(strings.NewReader(input))
	decoder.SetStrict(true)
	err := decoder.Decode(&pto)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown extension key", "must propagate the unknown-key error, not a legacy-fallback error")
	assert.Contains(t, err.Error(), "my_unregistered_ext")
	assert.NotEqual(t, ConfigTypeLegacy, pto.ConfigType, "must not fall back to legacy format")
}

func TestUnknownExtensionKey_LegacyFormat_AllowsFallback(t *testing.T) {
	resetRegistryForTesting(t)

	input := `
overrides:
  tenant-1:
    max_traces_per_user: 1000
`
	var pto perTenantOverrides
	decoder := yaml.NewDecoder(strings.NewReader(input))
	decoder.SetStrict(true)
	require.NoError(t, decoder.Decode(&pto))
	assert.Equal(t, ConfigTypeLegacy, pto.ConfigType)
	require.NotNil(t, pto.TenantLimits["tenant-1"])
	assert.Equal(t, 1000, pto.TenantLimits["tenant-1"].Ingestion.MaxLocalTracesPerUser)
}

func TestUnknownExtensionKey_Config_BlocksLegacyFallback(t *testing.T) {
	resetRegistryForTesting(t)

	input := `
defaults:
  ingestion:
    max_traces_per_user: 500
  my_unregistered_ext:
    field: value
`
	var cfg Config
	decoder := yaml.NewDecoder(strings.NewReader(input))
	decoder.SetStrict(true)
	err := decoder.Decode(&cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown extension key")
	assert.Contains(t, err.Error(), "my_unregistered_ext")
}

// resetRegistryForTesting clears the extension registry for the duration of the test,
// restoring the original state via t.Cleanup. This prevents extension registrations
// in one test from leaking into others.
func resetRegistryForTesting(t *testing.T) {
	t.Helper()
	extensionRegistry.Lock()
	savedEntries := extensionRegistry.entries
	savedLegacyKeys := extensionRegistry.allLegacyKeys
	extensionRegistry.entries = make(map[string]*registryEntry)
	extensionRegistry.allLegacyKeys = make(map[string]struct{})
	extensionRegistry.Unlock()

	t.Cleanup(func() {
		extensionRegistry.Lock()
		extensionRegistry.entries = savedEntries
		extensionRegistry.allLegacyKeys = savedLegacyKeys
		extensionRegistry.Unlock()
	})
}

var _ Extension = (*testExtension)(nil)

type testExtension struct {
	FieldA string `yaml:"field_a" json:"field_a"`
	FieldB *int   `yaml:"field_b" json:"field_b"`
}

func (t *testExtension) Key() string { return "test_extension" }

func (t *testExtension) RegisterFlagsAndApplyDefaults(_ string, _ *flag.FlagSet) {
	t.FieldA = "field_a_default"
	t.FieldB = new(int) // default: pointer to 0
}

func (t *testExtension) Validate() error {
	if t.FieldA == "" {
		return fmt.Errorf("field_a cannot be empty")
	}
	if t.FieldB == nil {
		return fmt.Errorf("field_b cannot be nil")
	}
	return nil
}

func (t *testExtension) LegacyKeys() []string {
	return []string{"test_extension_field_a", "test_extension_field_b"}
}

func (t *testExtension) FromLegacy(m map[string]any) error {
	if v, ok := m["test_extension_field_a"].(string); ok {
		t.FieldA = v
	}
	// JSON-decoded map[string]any represents numbers as float64; handle both int (YAML) and float64 (JSON).
	switch v := m["test_extension_field_b"].(type) {
	case int:
		t.FieldB = &v
	case float64:
		vv := int(v)
		t.FieldB = &vv
	}
	return nil
}

func (t *testExtension) ToLegacy() map[string]any {
	if t == nil {
		return nil
	}
	fieldB := 0
	if t.FieldB != nil {
		fieldB = *t.FieldB
	}
	return map[string]any{
		"test_extension_field_a": t.FieldA,
		"test_extension_field_b": fieldB,
	}
}

var _ Extension = (*conflictingExtension)(nil)

type conflictingExtension struct{}

func (c *conflictingExtension) Key() string                                             { return "ingestion" }
func (c *conflictingExtension) RegisterFlagsAndApplyDefaults(_ string, _ *flag.FlagSet) {}
func (c *conflictingExtension) Validate() error                                         { return nil }
func (c *conflictingExtension) LegacyKeys() []string                                    { return nil }
func (c *conflictingExtension) FromLegacy(_ map[string]any) error                       { return nil }
func (c *conflictingExtension) ToLegacy() map[string]any                                { return nil }
