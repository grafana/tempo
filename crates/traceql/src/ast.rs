/// Abstract Syntax Tree (AST) definitions for TraceQL
///
/// TraceQL is a query language for distributed traces with syntax like:
/// { span.http.method = "GET" && duration > 100ms }
use std::fmt;

/// Root query structure
#[derive(Debug, Clone, PartialEq)]
pub struct TraceQLQuery {
    /// The span filter expression or structural query
    pub query: QueryExpr,
    /// Optional pipeline operations
    pub pipeline: Vec<PipelineOp>,
    /// Optional having condition applied to aggregated results
    pub having: Option<HavingCondition>,
}

/// Query expression - can be a simple filter or structural query
#[derive(Debug, Clone, PartialEq)]
pub enum QueryExpr {
    /// Simple span filter: { expr }
    SpanFilter(SpanFilter),
    /// Structural query: { expr } >> { expr }
    Structural {
        parent: SpanFilter,
        child: SpanFilter,
    },
    /// Union query: { expr } || { expr } || ...
    Union(Vec<SpanFilter>),
}

/// Span filter - the expression inside { }
#[derive(Debug, Clone, PartialEq)]
pub struct SpanFilter {
    /// The filter expression, or None for empty filter { }
    pub expr: Option<Expr>,
}

/// Having condition applied to aggregated results (e.g., count() > 1)
#[derive(Debug, Clone, PartialEq)]
pub struct HavingCondition {
    /// Comparison operator
    pub op: ComparisonOperator,
    /// Value to compare against
    pub value: Value,
}

/// Expression types
#[derive(Debug, Clone, PartialEq)]
pub enum Expr {
    /// Binary operation (e.g., a && b, x > 5)
    BinaryOp {
        left: Box<Expr>,
        op: BinaryOperator,
        right: Box<Expr>,
    },
    /// Unary operation (e.g., !expr)
    UnaryOp { op: UnaryOperator, expr: Box<Expr> },
    /// Comparison (e.g., span.http.method = "GET")
    Comparison {
        field: FieldRef,
        op: ComparisonOperator,
        value: Value,
    },
}

/// Binary operators
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum BinaryOperator {
    And, // &&
    Or,  // ||
}

impl fmt::Display for BinaryOperator {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            BinaryOperator::And => write!(f, "&&"),
            BinaryOperator::Or => write!(f, "||"),
        }
    }
}

/// Unary operators
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum UnaryOperator {
    Not, // !
}

impl fmt::Display for UnaryOperator {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            UnaryOperator::Not => write!(f, "!"),
        }
    }
}

/// Comparison operators
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum ComparisonOperator {
    Eq,       // =
    NotEq,    // !=
    Gt,       // >
    Gte,      // >=
    Lt,       // <
    Lte,      // <=
    Regex,    // =~
    NotRegex, // !~
}

impl fmt::Display for ComparisonOperator {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ComparisonOperator::Eq => write!(f, "="),
            ComparisonOperator::NotEq => write!(f, "!="),
            ComparisonOperator::Gt => write!(f, ">"),
            ComparisonOperator::Gte => write!(f, ">="),
            ComparisonOperator::Lt => write!(f, "<"),
            ComparisonOperator::Lte => write!(f, "<="),
            ComparisonOperator::Regex => write!(f, "=~"),
            ComparisonOperator::NotRegex => write!(f, "!~"),
        }
    }
}

/// Field reference in TraceQL
#[derive(Debug, Clone, PartialEq)]
pub struct FieldRef {
    /// The scope (span, resource, or intrinsic)
    pub scope: FieldScope,
    /// The field name (e.g., "http.method", "duration", "nestedSetParent")
    pub name: String,
}

impl fmt::Display for FieldRef {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match &self.scope {
            FieldScope::Span => write!(f, "span.{}", self.name),
            FieldScope::Resource => write!(f, "resource.{}", self.name),
            FieldScope::Intrinsic => write!(f, "{}", self.name),
            FieldScope::Unscoped => write!(f, ".{}", self.name),
        }
    }
}

/// Field scope
#[derive(Debug, Clone, PartialEq)]
pub enum FieldScope {
    /// span.* attributes
    Span,
    /// resource.* attributes
    Resource,
    /// Intrinsic fields (name, duration, status, kind, nestedSetParent, etc.)
    Intrinsic,
    /// Unscoped field starting with . (searches both span and resource)
    Unscoped,
}

impl FieldScope {
    /// Check if a field name is an intrinsic field
    pub fn is_intrinsic(name: &str) -> bool {
        matches!(
            name,
            "name"
                | "duration"
                | "status"
                | "kind"
                | "nestedSetParent"
                | "nestedSetLeft"
                | "nestedSetRight"
                | "rootServiceName"
                | "rootName"
        )
    }
}

/// Value types in TraceQL
#[derive(Debug, Clone, PartialEq)]
pub enum Value {
    String(String),
    Integer(i64),
    Float(f64),
    Bool(bool),
    Duration(Duration),
    Status(Status),
    SpanKind(SpanKind),
}

impl fmt::Display for Value {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Value::String(s) => write!(f, "\"{}\"", s),
            Value::Integer(i) => write!(f, "{}", i),
            Value::Float(fl) => write!(f, "{}", fl),
            Value::Bool(b) => write!(f, "{}", b),
            Value::Duration(d) => write!(f, "{}", d),
            Value::Status(s) => write!(f, "{}", s),
            Value::SpanKind(k) => write!(f, "{}", k),
        }
    }
}

/// Duration value with unit
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct Duration {
    pub value: f64,
    pub unit: DurationUnit,
}

impl Duration {
    /// Convert duration to nanoseconds
    pub fn to_nanos(&self) -> i64 {
        let nanos = match self.unit {
            DurationUnit::Nanoseconds => self.value,
            DurationUnit::Microseconds => self.value * 1_000.0,
            DurationUnit::Milliseconds => self.value * 1_000_000.0,
            DurationUnit::Seconds => self.value * 1_000_000_000.0,
            DurationUnit::Minutes => self.value * 60_000_000_000.0,
            DurationUnit::Hours => self.value * 3_600_000_000_000.0,
        };
        nanos as i64
    }
}

impl fmt::Display for Duration {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}{}", self.value, self.unit)
    }
}

/// Duration units
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum DurationUnit {
    Nanoseconds,  // ns
    Microseconds, // us
    Milliseconds, // ms
    Seconds,      // s
    Minutes,      // m
    Hours,        // h
}

impl fmt::Display for DurationUnit {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            DurationUnit::Nanoseconds => write!(f, "ns"),
            DurationUnit::Microseconds => write!(f, "us"),
            DurationUnit::Milliseconds => write!(f, "ms"),
            DurationUnit::Seconds => write!(f, "s"),
            DurationUnit::Minutes => write!(f, "m"),
            DurationUnit::Hours => write!(f, "h"),
        }
    }
}

/// Span status values
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum Status {
    Unset,
    Ok,
    Error,
}

impl fmt::Display for Status {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Status::Unset => write!(f, "unset"),
            Status::Ok => write!(f, "ok"),
            Status::Error => write!(f, "error"),
        }
    }
}

/// Span kind values
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum SpanKind {
    Unspecified,
    Internal,
    Server,
    Client,
    Producer,
    Consumer,
}

impl fmt::Display for SpanKind {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            SpanKind::Unspecified => write!(f, "unspecified"),
            SpanKind::Internal => write!(f, "internal"),
            SpanKind::Server => write!(f, "server"),
            SpanKind::Client => write!(f, "client"),
            SpanKind::Producer => write!(f, "producer"),
            SpanKind::Consumer => write!(f, "consumer"),
        }
    }
}

/// Pipeline operations (not yet implemented, placeholder for future)
#[derive(Debug, Clone, PartialEq)]
pub enum PipelineOp {
    Rate {
        group_by: Vec<String>,
    },
    Count {
        group_by: Vec<String>,
    },
    Avg {
        field: String,
        group_by: Vec<String>,
    },
    Sum {
        field: String,
        group_by: Vec<String>,
    },
    Min {
        field: String,
        group_by: Vec<String>,
    },
    Max {
        field: String,
        group_by: Vec<String>,
    },
    Select {
        fields: Vec<FieldRef>,
    },
}
