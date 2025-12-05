//! Column path constants for vParquet4 schema
//!
//! These constants provide type-safe access to vParquet4 columns and nested fields.
//! They match the schema defined in tempodb/encoding/vparquet4/schema.go

/// Trace-level column paths
pub mod trace {
    /// Binary trace ID (16 bytes)
    pub const TRACE_ID: &str = "TraceID";

    /// Hex string trace ID (for human readability)
    pub const TRACE_ID_TEXT: &str = "TraceIDText";

    /// Start time in nanoseconds since Unix epoch
    pub const START_TIME_UNIX_NANO: &str = "StartTimeUnixNano";

    /// End time in nanoseconds since Unix epoch
    pub const END_TIME_UNIX_NANO: &str = "EndTimeUnixNano";

    /// Duration in nanoseconds
    pub const DURATION_NANO: &str = "DurationNano";

    /// Root service name (first span's service)
    pub const ROOT_SERVICE_NAME: &str = "RootServiceName";

    /// Root span name (first span's name)
    pub const ROOT_SPAN_NAME: &str = "RootSpanName";

    /// Service statistics (list of ServiceStats)
    pub const SERVICE_STATS: &str = "ServiceStats";

    /// Resource spans (list of ResourceSpans) - abbreviated as "rs" in actual files
    pub const RESOURCE_SPANS: &str = "rs";
}

/// ServiceStats column paths
pub mod service_stats {
    /// Service name key
    pub const KEY: &str = "Key";

    /// Number of spans for this service
    pub const SPAN_COUNT: &str = "SpanCount";

    /// Number of error spans for this service
    pub const ERROR_COUNT: &str = "ErrorCount";
}

/// Resource column paths
pub mod resource {
    /// Service name (service.name attribute)
    pub const SERVICE_NAME: &str = "ServiceName";

    /// Cluster name
    pub const CLUSTER: &str = "Cluster";

    /// Namespace
    pub const NAMESPACE: &str = "Namespace";

    /// Pod name
    pub const POD: &str = "Pod";

    /// Container name
    pub const CONTAINER: &str = "Container";

    /// Kubernetes cluster name (k8s.cluster.name)
    pub const K8S_CLUSTER_NAME: &str = "K8sClusterName";

    /// Kubernetes namespace name (k8s.namespace.name)
    pub const K8S_NAMESPACE_NAME: &str = "K8sNamespaceName";

    /// Kubernetes pod name (k8s.pod.name)
    pub const K8S_POD_NAME: &str = "K8sPodName";

    /// Kubernetes container name (k8s.container.name)
    pub const K8S_CONTAINER_NAME: &str = "K8sContainerName";

    /// Generic attributes list
    pub const ATTRS: &str = "Attrs";

    /// Number of dropped attributes
    pub const DROPPED_ATTRIBUTES_COUNT: &str = "DroppedAttributesCount";

    /// Dedicated attributes storage
    pub const DEDICATED_ATTRIBUTES: &str = "DedicatedAttributes";
}

/// Span column paths
pub mod span {
    /// Binary span ID (8 bytes)
    pub const SPAN_ID: &str = "SpanID";

    /// Binary parent span ID (8 bytes)
    pub const PARENT_SPAN_ID: &str = "ParentSpanID";

    /// Parent ID (for tree structure)
    pub const PARENT_ID: &str = "ParentID";

    /// Nested set left (for tree structure)
    pub const NESTED_SET_LEFT: &str = "NestedSetLeft";

    /// Nested set right (for tree structure)
    pub const NESTED_SET_RIGHT: &str = "NestedSetRight";

    /// Span name
    pub const NAME: &str = "Name";

    /// Span kind (INTERNAL=1, SERVER=2, CLIENT=3, PRODUCER=4, CONSUMER=5)
    pub const KIND: &str = "Kind";

    /// Trace state (W3C trace context)
    pub const TRACE_STATE: &str = "TraceState";

    /// Start time in nanoseconds since Unix epoch
    pub const START_TIME_UNIX_NANO: &str = "StartTimeUnixNano";

    /// Duration in nanoseconds
    pub const DURATION_NANO: &str = "DurationNano";

    /// Status code (UNSET=0, OK=1, ERROR=2)
    pub const STATUS_CODE: &str = "StatusCode";

    /// Status message
    pub const STATUS_MESSAGE: &str = "StatusMessage";

    /// HTTP method (dedicated column)
    pub const HTTP_METHOD: &str = "HttpMethod";

    /// HTTP URL (dedicated column)
    pub const HTTP_URL: &str = "HttpUrl";

    /// HTTP status code (dedicated column)
    pub const HTTP_STATUS_CODE: &str = "HttpStatusCode";

    /// Generic attributes list
    pub const ATTRS: &str = "Attrs";

    /// Events list
    pub const EVENTS: &str = "Events";

    /// Links list
    pub const LINKS: &str = "Links";

    /// Number of dropped attributes
    pub const DROPPED_ATTRIBUTES_COUNT: &str = "DroppedAttributesCount";

    /// Number of dropped events
    pub const DROPPED_EVENTS_COUNT: &str = "DroppedEventsCount";

    /// Number of dropped links
    pub const DROPPED_LINKS_COUNT: &str = "DroppedLinksCount";

    /// Dedicated attributes storage
    pub const DEDICATED_ATTRIBUTES: &str = "DedicatedAttributes";
}

/// Attribute column paths
pub mod attr {
    /// Attribute key
    pub const KEY: &str = "Key";

    /// Whether this attribute is an array
    pub const IS_ARRAY: &str = "IsArray";

    /// String value(s)
    pub const VALUE: &str = "Value";

    /// Integer value(s)
    pub const VALUE_INT: &str = "ValueInt";

    /// Double value(s)
    pub const VALUE_DOUBLE: &str = "ValueDouble";

    /// Boolean value(s)
    pub const VALUE_BOOL: &str = "ValueBool";

    /// Unsupported value types (JSON serialized)
    pub const VALUE_UNSUPPORTED: &str = "ValueUnsupported";
}

/// Event column paths
pub mod event {
    /// Event name
    pub const NAME: &str = "Name";

    /// Time since span start in nanoseconds
    pub const TIME_SINCE_START_NANO: &str = "TimeSinceStartNano";

    /// Event attributes
    pub const ATTRS: &str = "Attrs";

    /// Number of dropped attributes
    pub const DROPPED_ATTRIBUTES_COUNT: &str = "DroppedAttributesCount";
}

/// Link column paths
pub mod link {
    /// Linked trace ID (binary)
    pub const TRACE_ID: &str = "TraceID";

    /// Linked span ID (binary)
    pub const SPAN_ID: &str = "SpanID";

    /// Trace state
    pub const TRACE_STATE: &str = "TraceState";

    /// Link attributes
    pub const ATTRS: &str = "Attrs";

    /// Number of dropped attributes
    pub const DROPPED_ATTRIBUTES_COUNT: &str = "DroppedAttributesCount";
}

/// Dedicated attributes column paths
pub mod dedicated_attrs {
    /// Dedicated string attribute slots (String01-String10)
    pub const STRING_01: &str = "String01";
    pub const STRING_02: &str = "String02";
    pub const STRING_03: &str = "String03";
    pub const STRING_04: &str = "String04";
    pub const STRING_05: &str = "String05";
    pub const STRING_06: &str = "String06";
    pub const STRING_07: &str = "String07";
    pub const STRING_08: &str = "String08";
    pub const STRING_09: &str = "String09";
    pub const STRING_10: &str = "String10";

    /// All dedicated string columns as an array
    pub const ALL: &[&str] = &[
        STRING_01, STRING_02, STRING_03, STRING_04, STRING_05,
        STRING_06, STRING_07, STRING_08, STRING_09, STRING_10,
    ];
}

/// InstrumentationScope column paths
pub mod scope {
    /// Scope name
    pub const NAME: &str = "Name";

    /// Scope version
    pub const VERSION: &str = "Version";

    /// Scope attributes
    pub const ATTRS: &str = "Attrs";

    /// Number of dropped attributes
    pub const DROPPED_ATTRIBUTES_COUNT: &str = "DroppedAttributesCount";
}

/// Span kind enumeration values
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(i32)]
pub enum SpanKind {
    Unspecified = 0,
    Internal = 1,
    Server = 2,
    Client = 3,
    Producer = 4,
    Consumer = 5,
}

impl From<i32> for SpanKind {
    fn from(value: i32) -> Self {
        match value {
            1 => SpanKind::Internal,
            2 => SpanKind::Server,
            3 => SpanKind::Client,
            4 => SpanKind::Producer,
            5 => SpanKind::Consumer,
            _ => SpanKind::Unspecified,
        }
    }
}

/// Status code enumeration values
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(i32)]
pub enum StatusCode {
    Unset = 0,
    Ok = 1,
    Error = 2,
}

impl From<i32> for StatusCode {
    fn from(value: i32) -> Self {
        match value {
            1 => StatusCode::Ok,
            2 => StatusCode::Error,
            _ => StatusCode::Unset,
        }
    }
}
