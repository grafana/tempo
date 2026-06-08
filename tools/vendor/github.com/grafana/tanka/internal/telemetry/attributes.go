package telemetry

import (
	"github.com/grafana/tanka/pkg/spec/v1alpha1"
	"go.opentelemetry.io/otel/attribute"
)

func AttrPath(v string) attribute.KeyValue {
	return attribute.String("tanka.path", v)
}

func AttrLoader(v string) attribute.KeyValue {
	return attribute.String("tanka.loader", v)
}

func AttrNumEnvs(v int) attribute.KeyValue {
	return attribute.Int("tanka.envs.num", v)
}

func AttrEnv(v *v1alpha1.Environment) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("tanka.env.name", v.Metadata.Name),
		attribute.String("tanka.env.namespace", v.Spec.Namespace),
	}
}
