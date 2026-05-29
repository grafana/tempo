package v1

// internKey returns canonical strings for common OTLP attribute keys.
//
// KeyValue unmarshal is a hot path for metrics-generator decode workloads. The
// key space is usually bounded and dominated by semantic convention keys, while
// values can be high cardinality and must not be interned.
func internKey(b []byte) string {
	switch len(b) {
	case 6:
		if string(b) == "db.url" {
			return "db.url"
		}
	case 7:
		if string(b) == "os.type" {
			return "os.type"
		}
		if string(b) == "db.name" {
			return "db.name"
		}
	case 9:
		if string(b) == "db.system" {
			return "db.system"
		}
	case 10:
		if string(b) == "http.route" {
			return "http.route"
		}
	case 11:
		if string(b) == "process.pid" {
			return "process.pid"
		}
		if string(b) == "http.method" {
			return "http.method"
		}
	case 12:
		if string(b) == "service.name" {
			return "service.name"
		}
		if string(b) == "cloud.region" {
			return "cloud.region"
		}
		if string(b) == "k8s.pod.name" {
			return "k8s.pod.name"
		}
		if string(b) == "peer.service" {
			return "peer.service"
		}
		if string(b) == "db.namespace" {
			return "db.namespace"
		}
	case 13:
		if string(b) == "k8s.node.name" {
			return "k8s.node.name"
		}
		if string(b) == "net.peer.name" {
			return "net.peer.name"
		}
	case 14:
		if string(b) == "os.description" {
			return "os.description"
		}
		if string(b) == "server.address" {
			return "server.address"
		}
		if string(b) == "db.system.name" {
			return "db.system.name"
		}
	case 15:
		if string(b) == "service.version" {
			return "service.version"
		}
	case 16:
		if string(b) == "k8s.cluster.name" {
			return "k8s.cluster.name"
		}
		if string(b) == "http.status_code" {
			return "http.status_code"
		}
		if string(b) == "messaging.system" {
			return "messaging.system"
		}
	case 17:
		if string(b) == "service.namespace" {
			return "service.namespace"
		}
	case 18:
		if string(b) == "k8s.namespace.name" {
			return "k8s.namespace.name"
		}
		if string(b) == "k8s.pod.start_time" {
			return "k8s.pod.start_time"
		}
	case 19:
		if string(b) == "service.instance.id" {
			return "service.instance.id"
		}
	case 20:
		if string(b) == "process.command_args" {
			return "process.command_args"
		}
		if string(b) == "process.runtime.name" {
			return "process.runtime.name"
		}
	case 21:
		if string(b) == "telemetry.sdk.version" {
			return "telemetry.sdk.version"
		}
	case 22:
		if string(b) == "deployment.environment" {
			return "deployment.environment"
		}
		if string(b) == "telemetry.sdk.language" {
			return "telemetry.sdk.language"
		}
	case 23:
		if string(b) == "cloud.availability_zone" {
			return "cloud.availability_zone"
		}
		if string(b) == "process.executable.path" {
			return "process.executable.path"
		}
		if string(b) == "process.runtime.version" {
			return "process.runtime.version"
		}
	case 26:
		if string(b) == "resource.span.metrics.skip" {
			return "resource.span.metrics.skip"
		}
	case 27:
		if string(b) == "process.runtime.description" {
			return "process.runtime.description"
		}
	}
	return string(b)
}
