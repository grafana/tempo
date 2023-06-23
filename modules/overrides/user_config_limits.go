package overrides

type UserConfigurableLimits struct {
	// TODO how do we ensure Version is always filled in?
	Version string `json:"version" yaml:"version"`

	// Forwarders
	Forwarders *[]string `json:"forwarders" yaml:"forwarders"`
}

func newUserConfigurableLimits() *UserConfigurableLimits {
	return &UserConfigurableLimits{
		Version: "v1",
	}
}
