package client

import "fmt"

// ErrorNotFound means that the requested object is not found on the server
type ErrorNotFound struct {
	errOut string
}

func (e ErrorNotFound) Error() string {
	return e.errOut
}

// ErrorUnknownResource means that the requested resource type is unknown to the
// server
type ErrorUnknownResource struct {
	errOut string
}

func (e ErrorUnknownResource) Error() string {
	return e.errOut
}

// ErrorNoContext means that the context that was searched for couldn't be found
type ErrorNoContext string

func (e ErrorNoContext) Error() string {
	return fmt.Sprintf("no context named `%s` was found. Please check your $KUBECONFIG", string(e))
}

// ErrorNoCluster means that the cluster that was searched for couldn't be found
type ErrorNoCluster string

func (e ErrorNoCluster) Error() string {
	return fmt.Sprintf("no cluster that matches the apiServer `%s` was found. Please check your $KUBECONFIG", string(e))
}

// ErrorNothingReturned means that there was no output returned
type ErrorNothingReturned struct{}

func (e ErrorNothingReturned) Error() string {
	return "kubectl returned no output"
}
