package util

import (
	"context"
	"errors"

	"github.com/grafana/dskit/tenant"
	"github.com/grafana/dskit/user"
)

// ExtractValidOrgID extracts and validates tenant string from context.
func ExtractValidOrgID(ctx context.Context) (string, error) {
	id, err := tenant.TenantID(ctx)
	if err == nil {
		return id, nil
	}

	if !errors.Is(err, user.ErrTooManyOrgIDs) {
		return "", err
	}

	// If it's a multi-tenant ID, we validate individual IDs and return whole string
	if _, idsErr := tenant.TenantIDs(ctx); idsErr != nil {
		return "", idsErr
	}

	return user.ExtractOrgID(ctx)
}
