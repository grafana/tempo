package overrides

import (
	"flag"
	"fmt"
	"testing"
)

func TestOverridesExtension_MarshalJSON(t *testing.T) {
	// TODO register the extension and test the marshaling
	// implement a sub-test for legacy and non legacy overrides
}

func TestOverridesExtension_UnmarshalJSON(t *testing.T) {
	// TODO register the extension and test the marshaling
	// implement a sub-test for legacy and non legacy overrides
	// also test a JSON payload with an unregistered extension and check that it fails
}

func TestOverridesExtension_MarshalYAML(t *testing.T) {
	// TODO register the extension and test the marshaling
	// implement a sub-test for legacy and non legacy overrides
}

func TestOverridesExtension_UnmarshalYAML(t *testing.T) {
	// TODO register the extension and test the marshaling
	// implement a sub-test for legacy and non legacy overrides
	// also test a JSON payload with an unregistered extension and check that it fails
}

var _ Extension = (*testExtension)(nil)

type testExtension struct {
	FieldA string `yaml:"field_a" json:"field_a"`
	FieldB *int   `yaml:"field_b" json:"field_b"`
}

func (t *testExtension) Key() string {
	return "test_extension"
}

func (t *testExtension) RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet) {
	t.FieldA = "field_a_default"
	t.FieldB = new(0)
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
	return []string{
		"test_extension_field_a",
		"test_extension_field_b",
	}
}

func (t *testExtension) FromLegacy(m map[string]any) {
	if t == nil {
		*t = testExtension{}
	}
	if v, ok := m["test_extension_field_a"].(string); ok {
		t.FieldA = v
	}
	if v, ok := m["test_extension_field_b"].(int); ok {
		t.FieldB = &v
	}

}

func (t *testExtension) ToLegacy() map[string]any {
	if t == nil {
		return nil
	}
	return map[string]any{
		"test_extension_field_a": t.FieldA,
		"test_extension_field_b": *t.FieldB,
	}
}
