package v1

import "testing"

func TestInternKey(t *testing.T) {
	for _, key := range []string{
		"service.name",
		"k8s.cluster.name",
		"http.method",
		"db.system.name",
		"resource.span.metrics.skip",
	} {
		t.Run(key, func(t *testing.T) {
			if got := internKey([]byte(key)); got != key {
				t.Fatalf("internKey() = %q, want %q", got, key)
			}
		})
	}

	const customKey = "custom.tenant.attribute"
	if got := internKey([]byte(customKey)); got != customKey {
		t.Fatalf("internKey() = %q, want %q", got, customKey)
	}
}
