package app

import (
	"fmt"

	"github.com/grafana/tempo/modules/overrides/userconfigurableapi"
)

type overridesValidator struct {
	cfg *Config

	validForwarders map[string]struct{}
}

func NewOverridesValidator(cfg *Config) userconfigurableapi.Validator {
	validForwarders := map[string]struct{}{}
	for _, f := range cfg.Distributor.Forwarders {
		validForwarders[f.Name] = struct{}{}
	}

	return &overridesValidator{
		cfg: cfg,

		validForwarders: validForwarders,
	}
}

func (v *overridesValidator) Validate(limits *userconfigurableapi.UserConfigurableLimits) error {
	if limits.Forwarders != nil {
		for _, f := range *limits.Forwarders {
			if _, ok := v.validForwarders[f]; !ok {
				return fmt.Errorf("forwarder \"%s\" is not a known forwarder, contact your system administrator", f)
			}
		}
	}

	return nil
}
