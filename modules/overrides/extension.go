package overrides

import (
	"bytes"
	"encoding/json"
	"errors"
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
	// Validate validates the extension config after it has been decoded. Validate must be idempotent.
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
	allLegacyKeys map[string]struct{} // union of LegacyKeys() across all registered extensions
	sync.RWMutex
	entries map[string]*registryEntry
}{entries: make(map[string]*registryEntry), allLegacyKeys: make(map[string]struct{})}

// RegisterExtension registers a per-tenant overrides extension. The extension e must be a non-nil
// pointer to the extension struct (pointer receivers are required).
// Panics if:
//   - e is nil or not a pointer
//   - Key() is empty or already registered
//   - Key() conflicts with any built-in Overrides or LegacyOverrides field name
//   - Key() conflicts with a legacy flat key of an already-registered extension
//   - any LegacyKeys() entry is empty, duplicate, or conflicts with a built-in or already-registered name
//
// Returns a typed getter that retrieves the extension value from an Overrides.
func RegisterExtension[T Extension](e T) func(*Overrides) T {
	// Check for nil / typed-nil before any method call so callers get a clear
	// panic message instead of a nil-dereference inside Key() or LegacyKeys().
	typ := reflect.TypeOf(e)
	if typ == nil || typ.Kind() != reflect.Ptr {
		panic("overrides: RegisterExtension requires a non-nil pointer to the extension struct")
	}
	if reflect.ValueOf(e).IsNil() {
		panic("overrides: RegisterExtension requires a non-nil pointer to the extension struct")
	}

	key := e.Key()
	if key == "" {
		panic("overrides: extension Key() cannot be empty")
	}
	legacyKeys := e.LegacyKeys()

	extensionRegistry.Lock()
	defer extensionRegistry.Unlock()

	validateExtensionKey(key)
	validateExtensionLegacyKeys(key, legacyKeys)

	extensionRegistry.entries[key] = &registryEntry{
		key:        key,
		legacyKeys: legacyKeys,
		elemType:   typ.Elem(),
	}
	for _, lk := range legacyKeys {
		extensionRegistry.allLegacyKeys[lk] = struct{}{}
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

// validateExtensionKey checks that key does not conflict with any reserved name.
// Must be called with extensionRegistry.Lock held.
func validateExtensionKey(key string) {
	if _, exists := extensionRegistry.entries[key]; exists {
		panic(fmt.Sprintf("overrides: extension %q already registered", key))
	}
	if _, conflict := knownOverridesJSONFields()[key]; conflict {
		panic(fmt.Sprintf("overrides: extension key %q conflicts with a built-in Overrides field; choose a different key", key))
	}
	if isKnownLegacyOverridesField(key) {
		panic(fmt.Sprintf("overrides: extension key %q conflicts with a built-in LegacyOverrides field; choose a different key", key))
	}
	if _, conflict := extensionRegistry.allLegacyKeys[key]; conflict {
		panic(fmt.Sprintf("overrides: extension key %q conflicts with a legacy key of an already-registered extension; choose a different key", key))
	}
}

// validateExtensionLegacyKeys checks that legacyKeys declared by extension key are valid.
// Must be called with extensionRegistry.Lock held.
func validateExtensionLegacyKeys(key string, legacyKeys []string) {
	seen := make(map[string]struct{}, len(legacyKeys))
	for _, lk := range legacyKeys {
		if lk == "" {
			panic(fmt.Sprintf("overrides: extension %q has an empty legacy key", key))
		}
		if _, dup := seen[lk]; dup {
			panic(fmt.Sprintf("overrides: extension %q has duplicate legacy key %q", key, lk))
		}
		seen[lk] = struct{}{}
		if isKnownLegacyOverridesField(lk) {
			panic(fmt.Sprintf("overrides: extension %q legacy key %q conflicts with a built-in LegacyOverrides field; choose a different legacy key", key, lk))
		}
		// Guard against collisions with already-registered extensions' nested keys and legacy keys.
		if _, conflict := extensionRegistry.entries[lk]; conflict {
			panic(fmt.Sprintf("overrides: extension %q legacy key %q conflicts with the key of extension %q", key, lk, lk))
		}
		if _, conflict := extensionRegistry.allLegacyKeys[lk]; conflict {
			panic(fmt.Sprintf("overrides: extension %q legacy key %q conflicts with a legacy key of an already-registered extension", key, lk))
		}
	}
}

// processExtensions validates all entries in o.Extensions against the registry, converts raw
// decoded values (from YAML or JSON) to typed Extension instances, applies defaults, and
// calls Validate on each. It is idempotent: already-typed entries are only re-validated.
func processExtensions(o *Overrides) error {
	if len(o.Extensions) == 0 {
		return nil
	}

	extensionRegistry.RLock()
	defer extensionRegistry.RUnlock()

	for key, raw := range o.Extensions {
		entry, ok := extensionRegistry.entries[key]
		if !ok {
			isLegacy := isKnownLegacyOverridesField(key) || isRegisteredExtensionLegacyKey(key)
			return &unknownExtensionKeyError{key: key, isLegacy: isLegacy}
		}

		// Already a typed Extension (set programmatically or after legacy conversion)
		if ext, alreadyTyped := raw.(Extension); alreadyTyped {
			if err := ext.Validate(); err != nil {
				return &extensionError{fmt.Errorf("extension %q: %w", key, err)}
			}
			continue
		}

		// Create a new instance and apply defaults.
		instance := entry.newInstance()
		// Per-tenant extension configs have no CLI prefix.
		instance.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError))

		// Decode via JSON round-trip, which also normalizes map[any]any from YAML.
		b, err := json.Marshal(normalizeYAMLValue(raw))
		if err != nil {
			return &extensionError{fmt.Errorf("extension %q: marshal: %w", key, err)}
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()

		err = dec.Decode(instance)
		if err != nil {
			return &extensionError{fmt.Errorf("extension %q: unmarshal: %w", key, err)}
		}
		if err := instance.Validate(); err != nil {
			return &extensionError{fmt.Errorf("extension %q: %w", key, err)}
		}

		o.Extensions[key] = instance
	}
	return nil
}

// processLegacyExtensions converts registered extension flat keys in l.Extensions to typed
// Extension instances, giving LegacyOverrides.Extensions the same semantics as
// Overrides.Extensions after processExtensions: typed instances keyed by their nested Key().
//
// For each registered extension whose LegacyKeys appear in l.Extensions, a typed instance is
// created, defaults applied, FromLegacy called, and the instance validated. The flat keys are
// then removed and the typed instance stored under the extension's Key().
//
// Any unregistered flat keys are rejected.
func processLegacyExtensions(l *LegacyOverrides) error {
	if len(l.Extensions) == 0 {
		return nil
	}

	extensionRegistry.RLock()
	defer extensionRegistry.RUnlock()

	for _, entry := range extensionRegistry.entries {
		hasFlatKey := false
		for _, fk := range entry.legacyKeys {
			if _, ok := l.Extensions[fk]; ok {
				hasFlatKey = true
				break
			}
		}
		if !hasFlatKey {
			continue
		}

		instance := entry.newInstance()
		instance.RegisterFlagsAndApplyDefaults("", flag.NewFlagSet("", flag.ContinueOnError)) // extension configs have no CLI prefix

		err := instance.FromLegacy(l.Extensions)
		if err != nil {
			return fmt.Errorf("extension %q: from legacy: %w", entry.key, err)
		}
		if err := instance.Validate(); err != nil {
			return fmt.Errorf("extension %q: %w", entry.key, err)
		}

		for _, fk := range entry.legacyKeys {
			delete(l.Extensions, fk)
		}
		l.Extensions[entry.key] = instance
	}

	// Reject any remaining / unregistered flat keys
	for k, v := range l.Extensions {
		if _, typed := v.(Extension); !typed {
			return fmt.Errorf("unknown legacy extension key %q: must be registered via RegisterExtension and declared in LegacyKeys() before use", k)
		}
	}
	return nil
}

// flattenExtensionEntries returns a new map where typed Extension values are replaced by their
// flat legacy key-value pairs (via ToLegacy).
// Used when marshaling LegacyOverrides to produce the flat wire format.
func flattenExtensionEntries(m map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(m))
	for k, v := range m {
		ext, ok := v.(Extension)
		if !ok {
			return nil, fmt.Errorf("overrides extensions: entry %q is not an Extension", k)
		}

		for fk, fv := range ext.ToLegacy() {
			out[fk] = fv
		}
	}
	return out, nil
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

// extensionError wraps errors that originate from extension unmarshal or validation.
type extensionError struct{ err error }

func (e *extensionError) Error() string { return e.err.Error() }
func (e *extensionError) Unwrap() error { return e.err }

// isExtensionError reports whether err (or any error in its chain) is an extensionError.
func isExtensionError(err error) bool {
	var e *extensionError
	return errors.As(err, &e)
}

// isRegisteredExtensionLegacyKey reports whether key matches a flat legacy key declared by any
// registered extension. Must be called with extensionRegistry.RLock held.
func isRegisteredExtensionLegacyKey(key string) bool {
	_, ok := extensionRegistry.allLegacyKeys[key]
	return ok
}

// unknownExtensionKeyError is returned by processExtensions when a key in
// o.Extensions is not found in the registry
type unknownExtensionKeyError struct {
	key      string // the unknown extension key
	isLegacy bool   // true if key matches a known LegacyOverrides field name
}

func (e *unknownExtensionKeyError) Error() string {
	return fmt.Sprintf("unknown extension key %q: must be registered via RegisterExtension before use", e.key)
}

// isLegacyExtensionKeyError reports whether err is an unknownExtensionKeyError that matches a known
// legacy field name. If true, this signals that it's safe to fallback to legacy format.
func isLegacyExtensionKeyError(err error) bool {
	var e *unknownExtensionKeyError
	return errors.As(err, &e) && e.isLegacy
}

// isExtensionKeyError reports whether err is an unknownExtensionKeyError that does not match a known
// legacy field name.
func isExtensionKeyError(err error) bool {
	var e *unknownExtensionKeyError
	return errors.As(err, &e) && !e.isLegacy
}
