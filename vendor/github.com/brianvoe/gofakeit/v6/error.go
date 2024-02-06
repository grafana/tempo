package gofakeit

import (
	"errors"
	"math/rand"
)

// Error will return a random generic error
func Error() error {
	return err(globalFaker.Rand)
}

// Error will return a random generic error
func (f *Faker) Error() error {
	return err(f.Rand)
}

func err(r *rand.Rand) error {
	return errors.New(generate(r, getRandValue(r, []string{"error", "generic"})))
}

// ErrorObject will return a random error object word
func ErrorObject() error {
	return errorObject(globalFaker.Rand)
}

// ErrorObject will return a random error object word
func (f *Faker) ErrorObject() error {
	return errorObject(f.Rand)
}

func errorObject(r *rand.Rand) error {
	return errors.New(generate(r, getRandValue(r, []string{"error", "object"})))
}

// ErrorDatabase will return a random database error
func ErrorDatabase() error {
	return errorDatabase(globalFaker.Rand)
}

// ErrorDatabase will return a random database error
func (f *Faker) ErrorDatabase() error {
	return errorDatabase(f.Rand)
}

func errorDatabase(r *rand.Rand) error {
	return errors.New(generate(r, getRandValue(r, []string{"error", "database"})))
}

// ErrorGRPC will return a random gRPC error
func ErrorGRPC() error {
	return errorGRPC(globalFaker.Rand)
}

// ErrorGRPC will return a random gRPC error
func (f *Faker) ErrorGRPC() error {
	return errorGRPC(f.Rand)
}

func errorGRPC(r *rand.Rand) error {
	return errors.New(generate(r, getRandValue(r, []string{"error", "grpc"})))
}

// ErrorHTTP will return a random HTTP error
func ErrorHTTP() error {
	return errorHTTP(globalFaker.Rand)
}

// ErrorHTTP will return a random HTTP error
func (f *Faker) ErrorHTTP() error {
	return errorHTTP(f.Rand)
}

func errorHTTP(r *rand.Rand) error {
	return errors.New(generate(r, getRandValue(r, []string{"error", "http"})))
}

// ErrorHTTPClient will return a random HTTP client error response (400-418)
func ErrorHTTPClient() error {
	return errorHTTPClient(globalFaker.Rand)
}

// ErrorHTTPClient will return a random HTTP client error response (400-418)
func (f *Faker) ErrorHTTPClient() error {
	return errorHTTPClient(f.Rand)
}

func errorHTTPClient(r *rand.Rand) error {
	return errors.New(generate(r, getRandValue(r, []string{"error", "http_client"})))
}

// ErrorHTTPServer will return a random HTTP server error response (500-511)
func ErrorHTTPServer() error {
	return errorHTTPServer(globalFaker.Rand)
}

// ErrorHTTPServer will return a random HTTP server error response (500-511)
func (f *Faker) ErrorHTTPServer() error {
	return errorHTTPServer(f.Rand)
}

func errorHTTPServer(r *rand.Rand) error {
	return errors.New(generate(r, getRandValue(r, []string{"error", "http_server"})))
}

// ErrorRuntime will return a random runtime error
func ErrorRuntime() error {
	return errorRuntime(globalFaker.Rand)
}

// ErrorRuntime will return a random runtime error
func (f *Faker) ErrorRuntime() error {
	return errorRuntime(f.Rand)
}

func errorRuntime(r *rand.Rand) error {
	return errors.New(generate(r, getRandValue(r, []string{"error", "runtime"})))
}

// ErrorValidation will return a random validation error
func ErrorValidation() error {
	return errorValidation(globalFaker.Rand)
}

// ErrorValidation will return a random validation error
func (f *Faker) ErrorValidation() error {
	return errorValidation(f.Rand)
}

func errorValidation(r *rand.Rand) error {
	return errors.New(generate(r, getRandValue(r, []string{"error", "validation"})))
}

func addErrorLookup() {
	AddFuncLookup("error", Info{
		Display:     "Error",
		Category:    "error",
		Description: "Random error message",
		Example:     "syntax error",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return err(r), nil
		},
	})

	AddFuncLookup("errorobject", Info{
		Display:     "Error object word",
		Category:    "error",
		Description: "Random error object word",
		Example:     "protocol",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return errorObject(r), nil
		},
	})

	AddFuncLookup("errordatabase", Info{
		Display:     "Database error",
		Category:    "error",
		Description: "Random database error message",
		Example:     "sql error",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return errorDatabase(r), nil
		},
	})

	AddFuncLookup("errorgrpc", Info{
		Display:     "gRPC error",
		Category:    "error",
		Description: "Random gRPC error message",
		Example:     "client protocol error",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return errorGRPC(r), nil
		},
	})

	AddFuncLookup("errorhttp", Info{
		Display:     "HTTP error",
		Category:    "error",
		Description: "Random HTTP error message",
		Example:     "invalid method",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return errorHTTP(r), nil
		},
	})

	AddFuncLookup("errorhttpclient", Info{
		Display:     "HTTP client error",
		Category:    "error",
		Description: "Random HTTP client error message",
		Example:     "request timeout",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return errorHTTPClient(r), nil
		},
	})

	AddFuncLookup("errorhttpserver", Info{
		Display:     "HTTP server error",
		Category:    "error",
		Description: "Random HTTP server error message",
		Example:     "internal server error",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return errorHTTPServer(r), nil
		},
	})

	AddFuncLookup("errorruntime", Info{
		Display:     "Runtime error",
		Category:    "error",
		Description: "Random runtime error message",
		Example:     "address out of bounds",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return errorRuntime(r), nil
		},
	})

	AddFuncLookup("errorvalidation", Info{
		Display:     "Validation error",
		Category:    "error",
		Description: "Random validation error message",
		Example:     "missing required field",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return errorValidation(r), nil
		},
	})
}
