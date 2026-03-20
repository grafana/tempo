package overrides

import (
	"encoding/json"
	"flag"
	"fmt"
	"reflect"
	"sync"
)

// Extension describes a typed extension to the per-tenant overrides config.
// Implementations must use pointer receivers for all methods.
type Extension interface {
	// Key is the YAML/JSON property name used to store this extension's config.
	Key() string
	// RegisterFlagsAndApplyDefaults applies defaults for the extension config.
	RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet)
	// Validate validates the extension config after it has been decoded.
	Validate() error
	// LegacyKeys returns the flat-key names used in the legacy overrides format.
	// Return an empty slice if there are no legacy keys.
	LegacyKeys() []string
	// FromLegacy populates this extension from the flat legacy key map.
	// The full Extensions map is passed; implementations pick only their own keys.
	FromLegacy(map[string]any) error
	// ToLegacy serializes this extension to the flat legacy key map.
	ToLegacy() map[string]any
}

// registryEntry holds the reflect metadata needed to instantiate an extension.
type registryEntry struct {
	key        string
	legacyKeys []string
	elemType   reflect.Type // struct type (T without pointer indirection)
}

// newInstance creates a zeroed pointer instance of the extension type, cast to Extension.
func (e *registryEntry) newInstance() Extension {
	return reflect.New(e.elemType).Interface().(Extension)
}

var extensionRegistry = struct {
	sync.RWMutex
	entries map[string]*registryEntry
}{entries: make(map[string]*registryEntry)}

// RegisterExtension registers a per-tenant overrides extension.
// e must be a non-nil pointer to the extension struct (pointer receivers are required).
// Panics if an extension with the same Key() is already registered.
// Returns a typed getter that retrieves the extension value from an Overrides.
func RegisterExtension[T Extension](e T) func(*Overrides) T {
	key := e.Key()

	extensionRegistry.Lock()
	defer extensionRegistry.Unlock()

	if _, exists := extensionRegistry.entries[key]; exists {
		panic(fmt.Sprintf("overrides: extension %q already registered", key))
	}

	typ := reflect.TypeOf(e)
	if typ == nil || typ.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("overrides: extension %q must be registered as a pointer type", key))
	}

	extensionRegistry.entries[key] = &registryEntry{
		key:        key,
		legacyKeys: e.LegacyKeys(),
		elemType:   typ.Elem(),
	}

	return func(o *Overrides) T {
		if o == nil || o.Extensions == nil {
			var zero T
			return zero
		}
		v, _ := o.Extensions[key].(T)
		return v
	}
}

// processExtensions validates all entries in o.Extensions against the registry, converts raw
// decoded values (from YAML or JSON) to typed Extension instances, applies defaults, and
// calls Validate on each.
func processExtensions(extensions map[string]any) (map[string]Extension, error) {
	if len(extensions) == 0 {
		return nil, nil
	}

	extensionRegistry.RLock()
	defer extensionRegistry.RUnlock()

	processed := make(map[string]Extension, len(extensions))

	for key, raw := range extensions {
		entry, ok := extensionRegistry.entries[key]
		if !ok {
			return nil, fmt.Errorf("unknown extension key %q: must be registered via RegisterExtension before use", key)
		}

		// Create a new instance and apply defaults.
		instance := entry.newInstance()
		// Per-tenant extension configs have no CLI prefix.
		instance.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))

		// Decode via JSON round-trip, which also normalises map[any]any from YAML.
		b, err := json.Marshal(normalizeYAMLValue(raw))
		if err != nil {
			return nil, fmt.Errorf("extension %q: marshal: %w", key, err)
		}
		if err := json.Unmarshal(b, instance); err != nil {
			return nil, fmt.Errorf("extension %q: unmarshal: %w", key, err)
		}

		if err := instance.Validate(); err != nil {
			return nil, fmt.Errorf("extension %q: %w", key, err)
		}

		processed[key] = instance
	}

	return processed, nil
}

// processLegacyExtensions converts registered extension flat keys in raw to typed Extension
// instances, returning a map keyed by extension Key() (matching Overrides.Extensions semantics).
//
// For each registered extension whose LegacyKeys appear in raw, a typed instance is created,
// defaults applied, FromLegacy called, and the instance validated. All consumed flat keys are
// tracked; any remaining key in raw that was not consumed by a registered extension causes an error.
func processLegacyExtensions(raw map[string]any) (map[string]Extension, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	extensionRegistry.RLock()
	defer extensionRegistry.RUnlock()

	result := make(map[string]Extension)
	consumed := make(map[string]struct{})

	for _, entry := range extensionRegistry.entries {
		if len(entry.legacyKeys) == 0 {
			continue
		}
		hasFlatKey := false
		for _, fk := range entry.legacyKeys {
			if _, ok := raw[fk]; ok {
				hasFlatKey = true
				break
			}
		}
		if !hasFlatKey {
			continue
		}

		instance := entry.newInstance()
		// Per-tenant extension configs have no CLI prefix.
		instance.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))
		if err := instance.FromLegacy(raw); err != nil {
			return nil, fmt.Errorf("extension %q: from legacy: %w", entry.key, err)
		}
		if err := instance.Validate(); err != nil {
			return nil, fmt.Errorf("extension %q: %w", entry.key, err)
		}
		for _, fk := range entry.legacyKeys {
			consumed[fk] = struct{}{}
		}
		result[entry.key] = instance
	}

	for k := range raw {
		if _, ok := consumed[k]; !ok {
			return nil, fmt.Errorf("unknown extension flat key %q: must be registered via RegisterExtension before use", k)
		}
	}

	return result, nil
}

// flattenExtensionEntries returns a new map where typed Extension values are replaced by their
// flat legacy key-value pairs (via ToLegacy).
// Used when marshaling LegacyOverrides to produce the flat wire format.
func flattenExtensionEntries(m map[string]Extension) map[string]any {
	out := make(map[string]any, len(m))
	for _, ext := range m {
		for fk, fv := range ext.ToLegacy() {
			out[fk] = fv
		}
	}
	return out
}

// normalizeYAMLValue converts map[any]any produced by go-yaml to map[string]any recursively,
// making the value safe to pass to json.Marshal.
func normalizeYAMLValue(v any) any {
	switch val := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(val))
		for k, v2 := range val {
			switch key := k.(type) {
			case string:
				m[key] = normalizeYAMLValue(v2)
			case fmt.Stringer:
				m[key.String()] = normalizeYAMLValue(v2)
			default:
				m[fmt.Sprintf("%v", k)] = normalizeYAMLValue(v2)
			}
		}
		return m
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = normalizeYAMLValue(elem)
		}
		return out
	default:
		return v
	}
}

// ResetRegistryForTesting clears the extension registry and restores it after the test.
// This prevents extension registrations in one test from leaking into others.
func ResetRegistryForTesting(t interface{ Cleanup(func()) }) {
	extensionRegistry.Lock()
	saved := extensionRegistry.entries
	extensionRegistry.entries = make(map[string]*registryEntry)
	extensionRegistry.Unlock()

	t.Cleanup(func() {
		extensionRegistry.Lock()
		extensionRegistry.entries = saved
		extensionRegistry.Unlock()
	})
}
