package v1alpha1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// New creates a new Environment object with internal values already set
func New() *Environment {
	c := Environment{}

	// constants
	c.APIVersion = "tanka.dev/v1alpha1"
	c.Kind = "Environment"

	// default namespace
	c.Spec.Namespace = "default"

	c.Metadata.Labels = make(map[string]string)

	return &c
}

// Environment represents a set of resources in relation to its Kubernetes cluster
type Environment struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Spec       Spec     `json:"spec"`
	Data       any      `json:"data,omitempty"`
}

func (e Environment) NameLabel() (string, error) {
	envLabelFields := e.Spec.TankaEnvLabelFromFields
	if len(envLabelFields) == 0 {
		envLabelFields = []string{
			".metadata.name",
			".metadata.namespace",
		}
	}

	envLabelFieldValues, err := e.getFieldValuesByLabel(envLabelFields)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve field values for label: %w", err)
	}

	labelParts := strings.Join(envLabelFieldValues, ":")
	partsHash := sha256.Sum256([]byte(labelParts))
	chars := []rune(hex.EncodeToString(partsHash[:]))
	return string(chars[:48]), nil
}

func (e Environment) getFieldValuesByLabel(labels []string) ([]string, error) {
	if len(labels) == 0 {
		return nil, errors.New("labels must be set")
	}

	fieldValues := make([]string, len(labels))
	for idx, label := range labels {
		keyPath := strings.Split(strings.TrimPrefix(label, "."), ".")

		labelValue, err := getDeepFieldAsString(e, keyPath)
		if err != nil {
			return nil, fmt.Errorf("could not get struct value at path: %w", err)
		}

		fieldValues[idx] = labelValue
	}

	return fieldValues, nil
}

// Metadata is meant for humans and not parsed
type Metadata struct {
	Name      string            `json:"name,omitempty"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// Has and Get make Metadata a simple wrapper for labels.Labels to use our map in their querier
func (m Metadata) Has(label string) (exists bool) {
	_, exists = m.Labels[label]
	return exists
}

// Get implements Get for labels.Labels interface
func (m Metadata) Get(label string) (value string) {
	return m.Labels[label]
}

func (m Metadata) Lookup(label string) (value string, exists bool) {
	if m.Has(label) {
		return m.Get(label), true
	}
	return "", false
}

// Spec defines Kubernetes properties
type Spec struct {
	APIServer                   string           `json:"apiServer,omitempty"`
	ContextNames                []string         `json:"contextNames,omitempty"`
	Namespace                   string           `json:"namespace"`
	DiffStrategy                string           `json:"diffStrategy,omitempty"`
	ApplyStrategy               string           `json:"applyStrategy,omitempty"`
	InjectLabels                bool             `json:"injectLabels,omitempty"`
	TankaEnvLabelFromFields     []string         `json:"tankaEnvLabelFromFields,omitempty"`
	ResourceDefaults            ResourceDefaults `json:"resourceDefaults"`
	ExpectVersions              ExpectVersions   `json:"expectVersions"`
	ExportJsonnetImplementation string           `json:"exportJsonnetImplementation,omitempty"`
}

// ExpectVersions holds semantic version constraints
// TODO: extend this to handle more than Tanka
type ExpectVersions struct {
	Tanka string `json:"tanka,omitempty"`
}

// ResourceDefaults will be inserted in any manifests that tanka processes.
type ResourceDefaults struct {
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}
