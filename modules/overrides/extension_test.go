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
	ResetRegistryForTesting(t)
	RegisterExtension(&testExtension{})
	assert.Panics(t, func() { RegisterExtension(&testExtension{}) })
}

func TestRegisterExtension_TypedGetter(t *testing.T) {
	ResetRegistryForTesting(t)
	get := RegisterExtension(&testExtension{})

	fieldB := 99
	o := Overrides{
		Extensions: map[string]Extension{"test_extension": &testExtension{FieldA: "hello", FieldB: &fieldB}},
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
		ResetRegistryForTesting(t)
		get := RegisterExtension(&testExtension{})

		fieldB := 42
		o := Overrides{}
		o.Ingestion.MaxLocalTracesPerUser = 1000
		o.Extensions = map[string]Extension{
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
		ResetRegistryForTesting(t)
		RegisterExtension(&testExtension{})

		fieldB := 7
		o := Overrides{}
		o.Ingestion.MaxLocalTracesPerUser = 500
		o.Extensions = map[string]Extension{
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
		ResetRegistryForTesting(t)
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
		ResetRegistryForTesting(t)
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
		ResetRegistryForTesting(t)

		input := `{"ingestion": {"max_traces_per_user": 1}, "unknown_ext": {"x": 1}}`
		var o Overrides
		err := json.Unmarshal([]byte(input), &o)
		require.ErrorContains(t, err, "unknown extension key")
	})

	t.Run("overrides validate error", func(t *testing.T) {
		ResetRegistryForTesting(t)
		RegisterExtension(&testExtension{})

		// field_a is empty — Validate must reject it.
		input := `{"test_extension": {"field_a": ""}}`
		var o Overrides
		err := json.Unmarshal([]byte(input), &o)
		require.ErrorContains(t, err, "field_a cannot be empty")
	})

	t.Run("legacy overrides", func(t *testing.T) {
		ResetRegistryForTesting(t)
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
		ResetRegistryForTesting(t)
		get := RegisterExtension(&testExtension{})

		fieldB := 3
		o := Overrides{}
		o.Ingestion.MaxLocalTracesPerUser = 1000
		o.Extensions = map[string]Extension{
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
		ResetRegistryForTesting(t)
		RegisterExtension(&testExtension{})

		fieldB := 8
		o := Overrides{}
		o.Ingestion.MaxLocalTracesPerUser = 500
		o.Extensions = map[string]Extension{
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
		ResetRegistryForTesting(t)
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
		ResetRegistryForTesting(t)
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
		ResetRegistryForTesting(t)

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
		ResetRegistryForTesting(t)
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
		o, err := l.toNewLimits()
		require.NoError(t, err)
		for _, ext := range o.Extensions {
			err = ext.Validate()
			require.NoError(t, err)
		}

		ext := get(&o)
		require.NotNil(t, ext)
		assert.Equal(t, "from_legacy_yaml", ext.FieldA)
		require.NotNil(t, ext.FieldB)
		assert.Equal(t, 0, *ext.FieldB)
	})
}

func TestExtension_FullLegacyRoundTrip_perTenantOverrides(t *testing.T) {
	ResetRegistryForTesting(t)
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
	ResetRegistryForTesting(t)
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
	ResetRegistryForTesting(t)
	get := RegisterExtension(&testExtension{})

	fieldB := 13
	o := Overrides{}
	o.Ingestion.MaxLocalTracesPerUser = 777
	o.Extensions = map[string]Extension{
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
	ResetRegistryForTesting(t)
	get := RegisterExtension(&testExtension{})

	fieldB := 21
	// Build an Overrides with a typed extension, convert to legacy and back.
	o := Overrides{}
	o.Extensions = map[string]Extension{
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
	o2, err := l.toNewLimits()
	require.NoError(t, err)
	for _, ext := range o2.Extensions {
		err = ext.Validate()
		require.NoError(t, err)
	}

	ext := get(&o2)
	require.NotNil(t, ext)
	assert.Equal(t, "converted", ext.FieldA)
	assert.Equal(t, 21, *ext.FieldB)
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
	if v, ok := m["test_extension_field_b"].(int); ok {
		t.FieldB = &v
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
