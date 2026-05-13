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

func TestUnmarshalKey(t *testing.T) {
	previous := "custom.tenant.attribute"
	if got := unmarshalKey(previous, []byte(previous)); got != previous {
		t.Fatalf("unmarshalKey() = %q, want %q", got, previous)
	}

	if got := unmarshalKey(previous, []byte("custom.tenant.other")); got != "custom.tenant.other" {
		t.Fatalf("unmarshalKey() = %q, want %q", got, "custom.tenant.other")
	}

	if got := unmarshalKey(previous, []byte("service.name")); got != "service.name" {
		t.Fatalf("unmarshalKey() = %q, want %q", got, "service.name")
	}
}

func TestInternStringValue(t *testing.T) {
	for _, value := range []string{
		"GET",
		"200",
		"prod",
		"go",
		"linux",
	} {
		t.Run(value, func(t *testing.T) {
			if got := internStringValue([]byte(value)); got != value {
				t.Fatalf("internStringValue() = %q, want %q", got, value)
			}
		})
	}

	const customValue = "tenant-specific-value"
	if got := internStringValue([]byte(customValue)); got != customValue {
		t.Fatalf("internStringValue() = %q, want %q", got, customValue)
	}
}
