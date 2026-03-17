package overrides

import (
	"flag"
	"sync"
)

// Extension describes an extension to the overrides config
type Extension interface {
	// Key used as a property in YAML/JSON to store the extended config
	Key() string
	// RegisterFlagsAndApplyDefaults registers flags and applies defaults for this overrides extension
	RegisterFlagsAndApplyDefaults(prefix string, f *flag.FlagSet)
	// Validate validates the config for this overrides extension
	Validate() error
	// LegacyKeys from the flattened legacy config
	LegacyKeys() []string
	// FromLegacy converts this overrides extension from the legacy config to the new config
	FromLegacy(map[string]any)
	// ToLegacy converts this overrides extension from the new config to the legacy config
	ToLegacy() map[string]any
}

var registeredExtensions = struct {
	sync.RWMutex
	elements map[string]Extension
}{elements: make(map[string]Extension)}

func RegisterExtension(e Extension) {
	registeredExtensions.Lock()
	defer registeredExtensions.Unlock()

	registeredExtensions.elements[e.Key()] = e
}
