package v1

// internKey returns canonical strings for common OTLP attribute keys.
//
// KeyValue unmarshal is a hot path for metrics-generator decode workloads. The
// key space is usually bounded and dominated by semantic convention keys, while
// values can be high cardinality and must not be interned.
func internKey(b []byte) string {
	if key, ok := internStaticKey(b); ok {
		return key
	}
	return string(b)
}

func unmarshalKey(previous string, b []byte) string {
	if key, ok := internStaticKey(b); ok {
		return key
	}
	if stringEqualBytes(previous, b) {
		return previous
	}
	return string(b)
}

func internStaticKey(b []byte) (string, bool) {
	switch len(b) {
	case 6:
		if string(b) == "db.url" {
			return "db.url", true
		}
	case 7:
		if string(b) == "os.type" {
			return "os.type", true
		}
		if string(b) == "db.name" {
			return "db.name", true
		}
	case 9:
		if string(b) == "db.system" {
			return "db.system", true
		}
	case 10:
		if string(b) == "http.route" {
			return "http.route", true
		}
	case 11:
		if string(b) == "process.pid" {
			return "process.pid", true
		}
		if string(b) == "http.method" {
			return "http.method", true
		}
	case 12:
		if string(b) == "service.name" {
			return "service.name", true
		}
		if string(b) == "cloud.region" {
			return "cloud.region", true
		}
		if string(b) == "k8s.pod.name" {
			return "k8s.pod.name", true
		}
		if string(b) == "peer.service" {
			return "peer.service", true
		}
		if string(b) == "db.namespace" {
			return "db.namespace", true
		}
	case 13:
		if string(b) == "k8s.node.name" {
			return "k8s.node.name", true
		}
		if string(b) == "net.peer.name" {
			return "net.peer.name", true
		}
	case 14:
		if string(b) == "os.description" {
			return "os.description", true
		}
		if string(b) == "server.address" {
			return "server.address", true
		}
		if string(b) == "db.system.name" {
			return "db.system.name", true
		}
	case 15:
		if string(b) == "service.version" {
			return "service.version", true
		}
	case 16:
		if string(b) == "k8s.cluster.name" {
			return "k8s.cluster.name", true
		}
		if string(b) == "http.status_code" {
			return "http.status_code", true
		}
		if string(b) == "messaging.system" {
			return "messaging.system", true
		}
	case 17:
		if string(b) == "service.namespace" {
			return "service.namespace", true
		}
	case 18:
		if string(b) == "k8s.namespace.name" {
			return "k8s.namespace.name", true
		}
		if string(b) == "k8s.pod.start_time" {
			return "k8s.pod.start_time", true
		}
	case 19:
		if string(b) == "service.instance.id" {
			return "service.instance.id", true
		}
	case 20:
		if string(b) == "process.command_args" {
			return "process.command_args", true
		}
		if string(b) == "process.runtime.name" {
			return "process.runtime.name", true
		}
	case 21:
		if string(b) == "telemetry.sdk.version" {
			return "telemetry.sdk.version", true
		}
	case 22:
		if string(b) == "deployment.environment" {
			return "deployment.environment", true
		}
		if string(b) == "telemetry.sdk.language" {
			return "telemetry.sdk.language", true
		}
	case 23:
		if string(b) == "cloud.availability_zone" {
			return "cloud.availability_zone", true
		}
		if string(b) == "process.executable.path" {
			return "process.executable.path", true
		}
		if string(b) == "process.runtime.version" {
			return "process.runtime.version", true
		}
	case 26:
		if string(b) == "resource.span.metrics.skip" {
			return "resource.span.metrics.skip", true
		}
	case 27:
		if string(b) == "process.runtime.description" {
			return "process.runtime.description", true
		}
	}
	return "", false
}

// internStringValue returns canonical strings for common low-cardinality OTLP
// attribute values. Values can be high-cardinality tenant data, so only static
// well-known values are interned here.
func internStringValue(b []byte) string {
	if value, ok := internStaticStringValue(b); ok {
		return value
	}
	return string(b)
}

func unmarshalStringValue(previous string, b []byte) string {
	if value, ok := internStaticStringValue(b); ok {
		return value
	}
	if stringEqualBytes(previous, b) {
		return previous
	}
	return string(b)
}

func internStaticStringValue(b []byte) (string, bool) {
	switch len(b) {
	case 2:
		if b[0] == 'g' && b[1] == 'o' {
			return "go", true
		}
	case 3:
		switch b[0] {
		case 'G':
			if b[1] == 'E' && b[2] == 'T' {
				return "GET", true
			}
		case '2':
			if b[1] == '0' && b[2] == '0' {
				return "200", true
			}
		}
	case 4:
		if b[0] == 'p' && b[1] == 'r' && b[2] == 'o' && b[3] == 'd' {
			return "prod", true
		}
	case 5:
		if b[0] == 'l' && b[1] == 'i' && b[2] == 'n' && b[3] == 'u' && b[4] == 'x' {
			return "linux", true
		}
	}
	return "", false
}

func stringEqualBytes(s string, b []byte) bool {
	return s == string(b)
}
