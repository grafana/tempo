package work

import "errors"

var (
	ErrJobNotFound      = errors.New("job not found")
	ErrJobAlreadyExists = errors.New("job already exists")
	ErrJobMissingTenant = errors.New("job tenant not specified")
	// ErrTenantNotFound is returned when a tenant is not found
	ErrTenantNotFound = errors.New("tenant not found")
	// ErrTenantMissing is returned when a tenant is not specified
	ErrTenantMissing = errors.New("tenant missing")
)
