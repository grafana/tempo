package app

import (
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/grafana/tempo/modules/overrides/userconfigurableapi"
)

type overridesValidator struct {
	cfg Config
}

func (v *overridesValidator) Validate(limits *userconfigurableapi.UserConfigurableLimits) error {
	var validForwarders []string
	for _, f := range v.cfg.Distributor.Forwarders {
		validForwarders = append(validForwarders, f.Name)
	}

	if limits.Forwarders != nil {
		for _, f := range *limits.Forwarders {
			if !slices.Contains(validForwarders, f) {
				return fmt.Errorf("forwarder \"%s\" is not a known forwarder, contact your system administrator", f)
			}
		}
	}

	return nil
}
