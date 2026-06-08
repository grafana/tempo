package process

import (
	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

const (
	// AnnotationNamespaced can be set on any resource to override the decision
	// whether 'metadata.namespace' is set by Tanka
	AnnotationNamespaced = MetadataPrefix + "/namespaced"
)

// This is a list of "cluster-wide" resources harvested from `kubectl api-resources --namespaced=false`
// This helps us to know which objects we should NOT apply namespaces to automatically.
// We can add to this list periodically if new types are added.
// This only applies to built-in kubernetes types. CRDs will need to be handled with annotations.
var clusterWideKinds = map[string]bool{
	"APIService":                     true,
	"CertificateSigningRequest":      true,
	"ClusterRole":                    true,
	"ClusterRoleBinding":             true,
	"ComponentStatus":                true,
	"CSIDriver":                      true,
	"CSINode":                        true,
	"CustomResourceDefinition":       true,
	"MutatingWebhookConfiguration":   true,
	"Namespace":                      true,
	"Node":                           true,
	"NodeMetrics":                    true,
	"PersistentVolume":               true,
	"PodSecurityPolicy":              true,
	"PriorityClass":                  true,
	"RuntimeClass":                   true,
	"SelfSubjectAccessReview":        true,
	"SelfSubjectRulesReview":         true,
	"StorageClass":                   true,
	"SubjectAccessReview":            true,
	"TokenReview":                    true,
	"ValidatingWebhookConfiguration": true,
	"VolumeAttachment":               true,
}

// Namespace injects the default namespace of the environment into each
// resources, that does not already define one. AnnotationNamespaced can be used
// to disable this per resource
func Namespace(list manifest.List, def string) manifest.List {
	if def == "" {
		return list
	}

	for i, m := range list {
		namespaced := true
		if clusterWideKinds[m.Kind()] {
			namespaced = false
		}
		// check for annotation override
		if s, ok := m.Metadata().Annotations()[AnnotationNamespaced]; ok {
			namespaced = s == "true"
		}

		if namespaced && !m.Metadata().HasNamespace() {
			m.Metadata()["namespace"] = def
		}

		// remove annotations if empty (we always create those by accessing them)
		if len(m.Metadata().Annotations()) == 0 {
			delete(m.Metadata(), "annotations")
		}

		list[i] = m
	}

	return list
}
