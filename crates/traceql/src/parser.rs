/// Parser for TraceQL
///
/// Converts a stream of tokens into an Abstract Syntax Tree (AST).
use super::ast::*;
use super::lexer::{Lexer, LexerError, Token};
use std::fmt;

/// Parser for TraceQL
pub struct Parser {
    tokens: Vec<Token>,
    position: usize,
}

impl Parser {
    /// Create a new parser from a TraceQL query string
    pub fn new(input: &str) -> Result<Self, ParserError> {
        let mut lexer = Lexer::new(input);
        let tokens = lexer.tokenize().map_err(ParserError::LexerError)?;
        Ok(Self {
            tokens,
            position: 0,
        })
    }

    /// Parse the TraceQL query
    pub fn parse(&mut self) -> Result<TraceQLQuery, ParserError> {
        // Parse the main query (span filter or structural query)
        let query = self.parse_query_expr()?;

        // Parse optional pipeline operations
        let mut pipeline = Vec::new();
        while self.current_token() == &Token::Pipe {
            self.advance(); // consume pipe
            pipeline.push(self.parse_pipeline_op()?);
        }

        // Parse optional having condition (comparison after pipeline operations)
        let having = self.parse_having_condition()?;

        // Ensure we're at EOF
        if self.current_token() != &Token::Eof {
            return Err(ParserError::UnexpectedToken(
                self.current_token().clone(),
                "expected EOF or pipeline operator".to_string(),
            ));
        }

        Ok(TraceQLQuery {
            query,
            pipeline,
            having,
        })
    }

    /// Parse query expression (simple filter or structural query)
    fn parse_query_expr(&mut self) -> Result<QueryExpr, ParserError> {
        // Parse first span filter
        let first_filter = self.parse_span_filter()?;

        // Check for structural operator >>
        if self.current_token() == &Token::StructuralOp {
            self.advance(); // consume >>
            let second_filter = self.parse_span_filter()?;
            Ok(QueryExpr::Structural {
                parent: first_filter,
                child: second_filter,
            })
        }
        // Check for union operator ||
        else if self.current_token() == &Token::Or {
            let mut filters = vec![first_filter];
            while self.current_token() == &Token::Or {
                self.advance(); // consume ||
                filters.push(self.parse_span_filter()?);
            }
            Ok(QueryExpr::Union(filters))
        } else {
            Ok(QueryExpr::SpanFilter(first_filter))
        }
    }

    /// Parse a span filter: { expr }
    fn parse_span_filter(&mut self) -> Result<SpanFilter, ParserError> {
        self.expect_token(Token::LBrace)?;

        // Check for empty filter { }
        if self.current_token() == &Token::RBrace {
            self.advance();
            return Ok(SpanFilter { expr: None });
        }

        // Parse the expression
        let expr = self.parse_expr()?;

        self.expect_token(Token::RBrace)?;

        Ok(SpanFilter { expr: Some(expr) })
    }

    /// Parse an expression (handles binary operations with precedence)
    fn parse_expr(&mut self) -> Result<Expr, ParserError> {
        self.parse_or_expr()
    }

    /// Parse OR expression (lowest precedence)
    fn parse_or_expr(&mut self) -> Result<Expr, ParserError> {
        let mut left = self.parse_and_expr()?;

        while self.current_token() == &Token::Or {
            self.advance();
            let right = self.parse_and_expr()?;
            left = Expr::BinaryOp {
                left: Box::new(left),
                op: BinaryOperator::Or,
                right: Box::new(right),
            };
        }

        Ok(left)
    }

    /// Parse AND expression (higher precedence than OR)
    fn parse_and_expr(&mut self) -> Result<Expr, ParserError> {
        let mut left = self.parse_unary_expr()?;

        while self.current_token() == &Token::And {
            self.advance();
            let right = self.parse_unary_expr()?;
            left = Expr::BinaryOp {
                left: Box::new(left),
                op: BinaryOperator::And,
                right: Box::new(right),
            };
        }

        Ok(left)
    }

    /// Parse unary expression (NOT)
    fn parse_unary_expr(&mut self) -> Result<Expr, ParserError> {
        if self.current_token() == &Token::Not {
            self.advance();
            let expr = self.parse_unary_expr()?;
            Ok(Expr::UnaryOp {
                op: UnaryOperator::Not,
                expr: Box::new(expr),
            })
        } else {
            self.parse_comparison_expr()
        }
    }

    /// Parse comparison expression
    fn parse_comparison_expr(&mut self) -> Result<Expr, ParserError> {
        // Check for parentheses
        if self.current_token() == &Token::LParen {
            self.advance();
            let expr = self.parse_expr()?;
            self.expect_token(Token::RParen)?;
            return Ok(expr);
        }

        // Parse field reference
        let field = self.parse_field_ref()?;

        // Parse comparison operator
        let op = self.parse_comparison_op()?;

        // Parse value
        let value = self.parse_value()?;

        Ok(Expr::Comparison { field, op, value })
    }

    /// Parse field reference (e.g., span.http.method, duration, .service.name)
    fn parse_field_ref(&mut self) -> Result<FieldRef, ParserError> {
        // Check for leading dot (unscoped field)
        if self.current_token() == &Token::Dot {
            self.advance();
            let name = self.expect_identifier()?;
            return Ok(FieldRef {
                scope: FieldScope::Unscoped,
                name,
            });
        }

        // Parse identifier
        let first = self.expect_identifier()?;

        // Check if it's an intrinsic field (no dot after)
        if self.current_token() != &Token::Dot {
            if FieldScope::is_intrinsic(&first) {
                return Ok(FieldRef {
                    scope: FieldScope::Intrinsic,
                    name: first,
                });
            } else {
                // Treat as intrinsic anyway (for forward compatibility)
                return Ok(FieldRef {
                    scope: FieldScope::Intrinsic,
                    name: first,
                });
            }
        }

        // Consume dot
        self.advance();

        // Parse the rest of the field path
        let mut parts = vec![self.expect_identifier()?];
        while self.current_token() == &Token::Dot {
            self.advance();
            parts.push(self.expect_identifier()?);
        }

        // Determine scope based on first identifier
        let (scope, name) = match first.as_str() {
            "span" => (FieldScope::Span, parts.join(".")),
            "resource" => (FieldScope::Resource, parts.join(".")),
            _ => {
                // If not span or resource, treat as intrinsic with full path
                let full_name = format!("{}.{}", first, parts.join("."));
                (FieldScope::Intrinsic, full_name)
            }
        };

        Ok(FieldRef { scope, name })
    }

    /// Parse comparison operator
    fn parse_comparison_op(&mut self) -> Result<ComparisonOperator, ParserError> {
        let op = match self.current_token() {
            Token::Eq => ComparisonOperator::Eq,
            Token::NotEq => ComparisonOperator::NotEq,
            Token::Gt => ComparisonOperator::Gt,
            Token::Gte => ComparisonOperator::Gte,
            Token::Lt => ComparisonOperator::Lt,
            Token::Lte => ComparisonOperator::Lte,
            Token::Regex => ComparisonOperator::Regex,
            Token::NotRegex => ComparisonOperator::NotRegex,
            _ => {
                return Err(ParserError::UnexpectedToken(
                    self.current_token().clone(),
                    "expected comparison operator".to_string(),
                ));
            }
        };
        self.advance();
        Ok(op)
    }

    /// Parse value (string, number, boolean, duration, status, kind)
    fn parse_value(&mut self) -> Result<Value, ParserError> {
        match self.current_token() {
            Token::String(s) => {
                let value = Value::String(s.clone());
                self.advance();
                Ok(value)
            }
            Token::Integer(i) => {
                let value = Value::Integer(*i);
                self.advance();
                Ok(value)
            }
            Token::Float(f) => {
                let value = Value::Float(*f);
                self.advance();
                Ok(value)
            }
            Token::True => {
                self.advance();
                Ok(Value::Bool(true))
            }
            Token::False => {
                self.advance();
                Ok(Value::Bool(false))
            }
            Token::Identifier(id) => {
                let value = self.parse_identifier_value(id)?;
                self.advance();
                Ok(value)
            }
            _ => Err(ParserError::UnexpectedToken(
                self.current_token().clone(),
                "expected value".to_string(),
            )),
        }
    }

    /// Parse identifier as a value (duration, status, or kind)
    fn parse_identifier_value(&self, id: &str) -> Result<Value, ParserError> {
        // Try parsing as duration (e.g., "100ms", "1s")
        if let Some(duration) = self.parse_duration(id) {
            return Ok(Value::Duration(duration));
        }

        // Try parsing as status
        match id {
            "unset" => return Ok(Value::Status(Status::Unset)),
            "ok" => return Ok(Value::Status(Status::Ok)),
            "error" => return Ok(Value::Status(Status::Error)),
            _ => {}
        }

        // Try parsing as span kind
        match id {
            "unspecified" => return Ok(Value::SpanKind(SpanKind::Unspecified)),
            "internal" => return Ok(Value::SpanKind(SpanKind::Internal)),
            "server" => return Ok(Value::SpanKind(SpanKind::Server)),
            "client" => return Ok(Value::SpanKind(SpanKind::Client)),
            "producer" => return Ok(Value::SpanKind(SpanKind::Producer)),
            "consumer" => return Ok(Value::SpanKind(SpanKind::Consumer)),
            _ => {}
        }

        Err(ParserError::InvalidValue(id.to_string()))
    }

    /// Parse duration string (e.g., "100ms", "1s")
    fn parse_duration(&self, s: &str) -> Option<Duration> {
        // Find where the unit starts
        let num_end = s
            .chars()
            .take_while(|c| c.is_numeric() || *c == '.')
            .count();
        if num_end == 0 {
            return None;
        }

        let num_str = &s[..num_end];
        let unit_str = &s[num_end..];

        let value: f64 = num_str.parse().ok()?;

        let unit = match unit_str {
            "ns" => DurationUnit::Nanoseconds,
            "us" => DurationUnit::Microseconds,
            "ms" => DurationUnit::Milliseconds,
            "s" => DurationUnit::Seconds,
            "m" => DurationUnit::Minutes,
            "h" => DurationUnit::Hours,
            _ => return None,
        };

        Some(Duration { value, unit })
    }

    /// Parse pipeline operation (e.g., rate())
    fn parse_pipeline_op(&mut self) -> Result<PipelineOp, ParserError> {
        let op_name = self.expect_identifier()?;

        match op_name.as_str() {
            "rate" => {
                // rate() - may or may not have parentheses
                if self.current_token() == &Token::LParen {
                    self.advance();
                    self.expect_token(Token::RParen)?;
                }

                // Check for "by" clause
                let group_by = self.parse_group_by()?;

                Ok(PipelineOp::Rate { group_by })
            }
            "count" => {
                if self.current_token() == &Token::LParen {
                    self.advance();
                    self.expect_token(Token::RParen)?;
                }

                // Check for "by" clause
                let group_by = self.parse_group_by()?;

                Ok(PipelineOp::Count { group_by })
            }
            "select" => {
                // select(field1, field2, ...)
                self.expect_token(Token::LParen)?;

                let mut fields = Vec::new();
                loop {
                    fields.push(self.parse_field_ref()?);

                    match self.current_token() {
                        Token::Comma => {
                            self.advance();
                        }
                        Token::RParen => {
                            self.advance();
                            break;
                        }
                        _ => {
                            return Err(ParserError::UnexpectedToken(
                                self.current_token().clone(),
                                "expected ',' or ')' in select field list".to_string(),
                            ))
                        }
                    }
                }

                Ok(PipelineOp::Select { fields })
            }
            _ => Err(ParserError::UnknownPipelineOp(op_name)),
        }
    }

    /// Parse optional "by (field1, field2, ...)" clause
    fn parse_group_by(&mut self) -> Result<Vec<String>, ParserError> {
        // Check if next token is "by"
        if let Token::Identifier(id) = self.current_token() {
            if id == "by" {
                self.advance();

                // Expect opening parenthesis
                self.expect_token(Token::LParen)?;

                // Parse field list
                let mut fields = Vec::new();
                loop {
                    let field = self.expect_identifier()?;
                    fields.push(field);

                    // Check for comma or closing parenthesis
                    match self.current_token() {
                        Token::Comma => {
                            self.advance();
                            continue;
                        }
                        Token::RParen => {
                            self.advance();
                            break;
                        }
                        _ => {
                            return Err(ParserError::UnexpectedToken(
                                self.current_token().clone(),
                                "expected ',' or ')' in group by clause".to_string(),
                            ));
                        }
                    }
                }

                return Ok(fields);
            }
        }

        // No "by" clause found
        Ok(Vec::new())
    }

    /// Parse optional having condition (e.g., > 1, >= 10)
    fn parse_having_condition(&mut self) -> Result<Option<HavingCondition>, ParserError> {
        // Check if current token is a comparison operator
        let op = match self.current_token() {
            Token::Eq => ComparisonOperator::Eq,
            Token::NotEq => ComparisonOperator::NotEq,
            Token::Gt => ComparisonOperator::Gt,
            Token::Gte => ComparisonOperator::Gte,
            Token::Lt => ComparisonOperator::Lt,
            Token::Lte => ComparisonOperator::Lte,
            Token::Regex => ComparisonOperator::Regex,
            Token::NotRegex => ComparisonOperator::NotRegex,
            _ => return Ok(None), // No having condition
        };

        self.advance(); // consume comparison operator

        // Parse value
        let value = self.parse_value()?;

        Ok(Some(HavingCondition { op, value }))
    }

    // Helper methods

    fn current_token(&self) -> &Token {
        self.tokens.get(self.position).unwrap_or(&Token::Eof)
    }

    fn advance(&mut self) {
        if self.position < self.tokens.len() {
            self.position += 1;
        }
    }

    fn expect_token(&mut self, expected: Token) -> Result<(), ParserError> {
        if self.current_token() == &expected {
            self.advance();
            Ok(())
        } else {
            Err(ParserError::UnexpectedToken(
                self.current_token().clone(),
                format!("expected {:?}", expected),
            ))
        }
    }

    fn expect_identifier(&mut self) -> Result<String, ParserError> {
        match self.current_token() {
            Token::Identifier(id) => {
                let result = id.clone();
                self.advance();
                Ok(result)
            }
            _ => Err(ParserError::UnexpectedToken(
                self.current_token().clone(),
                "expected identifier".to_string(),
            )),
        }
    }
}

/// Parser errors
#[derive(Debug)]
pub enum ParserError {
    LexerError(LexerError),
    UnexpectedToken(Token, String),
    InvalidValue(String),
    UnknownPipelineOp(String),
}

impl fmt::Display for ParserError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ParserError::LexerError(e) => write!(f, "Lexer error: {}", e),
            ParserError::UnexpectedToken(token, expected) => {
                write!(f, "Unexpected token {:?}, {}", token, expected)
            }
            ParserError::InvalidValue(v) => write!(f, "Invalid value: {}", v),
            ParserError::UnknownPipelineOp(op) => write!(f, "Unknown pipeline operation: {}", op),
        }
    }
}

impl std::error::Error for ParserError {}

/// Parse a TraceQL query string into an AST
pub fn parse(input: &str) -> Result<TraceQLQuery, ParserError> {
    let mut parser = Parser::new(input)?;
    parser.parse()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_empty_filter() {
        let query = parse("{ }").unwrap();
        assert_eq!(
            query.query,
            QueryExpr::SpanFilter(SpanFilter { expr: None })
        );
    }

    #[test]
    fn test_simple_comparison() {
        let query = parse(r#"{ span.http.method = "GET" }"#).unwrap();
        if let QueryExpr::SpanFilter(filter) = query.query {
            assert!(filter.expr.is_some());
        } else {
            panic!("Expected SpanFilter");
        }
    }

    #[test]
    fn test_and_operation() {
        let query =
            parse(r#"{ span.http.method = "POST" && span.http.status_code = 500 }"#).unwrap();
        if let QueryExpr::SpanFilter(filter) = query.query {
            assert!(filter.expr.is_some());
        } else {
            panic!("Expected SpanFilter");
        }
    }

    #[test]
    fn test_duration() {
        let query = parse("{ duration > 100ms }").unwrap();
        if let QueryExpr::SpanFilter(filter) = query.query {
            assert!(filter.expr.is_some());
        } else {
            panic!("Expected SpanFilter");
        }
    }

    #[test]
    fn test_pipeline() {
        let query = parse("{ } | rate()").unwrap();
        assert_eq!(query.pipeline.len(), 1);
    }

    #[test]
    fn test_structural() {
        let query = parse("{ } >> { }").unwrap();
        assert!(matches!(query.query, QueryExpr::Structural { .. }));
    }
}
