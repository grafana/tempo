package app

import (
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/grafana/tempo/modules/overrides/userconfigurableapi"
)

func overridesValidator(cfg Config) func(limits *userconfigurableapi.UserConfigurableLimits) error {
	var validForwarders []string
	for _, f := range cfg.Distributor.Forwarders {
		validForwarders = append(validForwarders, f.Name)
	}

	return func(limits *userconfigurableapi.UserConfigurableLimits) error {
		if limits.Forwarders != nil {
			for _, f := range *limits.Forwarders {
				if !slices.Contains(validForwarders, f) {
					return fmt.Errorf("forwarder \"%s\" is not a known forwarder, contact your system administrator", f)
				}
			}
		}

		return nil
	}
}
