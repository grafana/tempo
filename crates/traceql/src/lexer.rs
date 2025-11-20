/// Lexer/Tokenizer for TraceQL
///
/// Converts TraceQL query strings into a stream of tokens for parsing.
use std::fmt;

/// Token types in TraceQL
#[derive(Debug, Clone, PartialEq)]
pub enum Token {
    // Literals
    String(String),
    Integer(i64),
    Float(f64),
    Identifier(String),

    // Keywords
    True,
    False,

    // Operators
    Eq,       // =
    NotEq,    // !=
    Gt,       // >
    Gte,      // >=
    Lt,       // <
    Lte,      // <=
    Regex,    // =~
    NotRegex, // !~
    And,      // &&
    Or,       // ||
    Not,      // !

    // Structural
    StructuralOp, // >>
    Pipe,         // |

    // Delimiters
    LBrace, // {
    RBrace, // }
    LParen, // (
    RParen, // )
    Dot,    // .
    Comma,  // ,

    // Special
    Eof,
}

impl fmt::Display for Token {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Token::String(s) => write!(f, "\"{}\"", s),
            Token::Integer(i) => write!(f, "{}", i),
            Token::Float(fl) => write!(f, "{}", fl),
            Token::Identifier(id) => write!(f, "{}", id),
            Token::True => write!(f, "true"),
            Token::False => write!(f, "false"),
            Token::Eq => write!(f, "="),
            Token::NotEq => write!(f, "!="),
            Token::Gt => write!(f, ">"),
            Token::Gte => write!(f, ">="),
            Token::Lt => write!(f, "<"),
            Token::Lte => write!(f, "<="),
            Token::Regex => write!(f, "=~"),
            Token::NotRegex => write!(f, "!~"),
            Token::And => write!(f, "&&"),
            Token::Or => write!(f, "||"),
            Token::Not => write!(f, "!"),
            Token::StructuralOp => write!(f, ">>"),
            Token::Pipe => write!(f, "|"),
            Token::LBrace => write!(f, "{{"),
            Token::RBrace => write!(f, "}}"),
            Token::LParen => write!(f, "("),
            Token::RParen => write!(f, ")"),
            Token::Dot => write!(f, "."),
            Token::Comma => write!(f, ","),
            Token::Eof => write!(f, "EOF"),
        }
    }
}

/// Lexer for TraceQL
pub struct Lexer {
    input: Vec<char>,
    position: usize,
}

impl Lexer {
    /// Create a new lexer from input string
    pub fn new(input: &str) -> Self {
        Self {
            input: input.chars().collect(),
            position: 0,
        }
    }

    /// Get the next token
    pub fn next_token(&mut self) -> Result<Token, LexerError> {
        self.skip_whitespace();

        if self.is_eof() {
            return Ok(Token::Eof);
        }

        let ch = self.current_char();

        match ch {
            '{' => {
                self.advance();
                Ok(Token::LBrace)
            }
            '}' => {
                self.advance();
                Ok(Token::RBrace)
            }
            '(' => {
                self.advance();
                Ok(Token::LParen)
            }
            ')' => {
                self.advance();
                Ok(Token::RParen)
            }
            '.' => {
                self.advance();
                Ok(Token::Dot)
            }
            ',' => {
                self.advance();
                Ok(Token::Comma)
            }
            '|' => {
                self.advance();
                if self.current_char() == '|' {
                    self.advance();
                    Ok(Token::Or)
                } else {
                    Ok(Token::Pipe)
                }
            }
            '=' => {
                self.advance();
                if self.current_char() == '~' {
                    self.advance();
                    Ok(Token::Regex)
                } else {
                    Ok(Token::Eq)
                }
            }
            '!' => {
                self.advance();
                if self.current_char() == '=' {
                    self.advance();
                    Ok(Token::NotEq)
                } else if self.current_char() == '~' {
                    self.advance();
                    Ok(Token::NotRegex)
                } else {
                    Ok(Token::Not)
                }
            }
            '>' => {
                self.advance();
                if self.current_char() == '=' {
                    self.advance();
                    Ok(Token::Gte)
                } else if self.current_char() == '>' {
                    self.advance();
                    Ok(Token::StructuralOp)
                } else {
                    Ok(Token::Gt)
                }
            }
            '<' => {
                self.advance();
                if self.current_char() == '=' {
                    self.advance();
                    Ok(Token::Lte)
                } else {
                    Ok(Token::Lt)
                }
            }
            '&' => {
                self.advance();
                if self.current_char() == '&' {
                    self.advance();
                    Ok(Token::And)
                } else {
                    Err(LexerError::UnexpectedCharacter(ch, self.position))
                }
            }
            '"' => self.read_string('"'),
            '`' => self.read_string('`'),
            _ if ch.is_ascii_digit()
                || (ch == '-' && self.peek().map(|c| c.is_ascii_digit()).unwrap_or(false)) =>
            {
                self.read_number()
            }
            _ if ch.is_alphabetic() || ch == '_' => self.read_identifier(),
            _ => Err(LexerError::UnexpectedCharacter(ch, self.position)),
        }
    }

    /// Tokenize the entire input
    pub fn tokenize(&mut self) -> Result<Vec<Token>, LexerError> {
        let mut tokens = Vec::new();

        loop {
            let token = self.next_token()?;
            if token == Token::Eof {
                tokens.push(token);
                break;
            }
            tokens.push(token);
        }

        Ok(tokens)
    }

    fn current_char(&self) -> char {
        if self.is_eof() {
            '\0'
        } else {
            self.input[self.position]
        }
    }

    fn peek(&self) -> Option<char> {
        if self.position + 1 < self.input.len() {
            Some(self.input[self.position + 1])
        } else {
            None
        }
    }

    fn advance(&mut self) {
        self.position += 1;
    }

    fn is_eof(&self) -> bool {
        self.position >= self.input.len()
    }

    fn skip_whitespace(&mut self) {
        while !self.is_eof() && self.current_char().is_whitespace() {
            self.advance();
        }
    }

    fn read_string(&mut self, quote_char: char) -> Result<Token, LexerError> {
        let start_pos = self.position;
        self.advance(); // skip opening quote

        let mut value = String::new();

        while !self.is_eof() && self.current_char() != quote_char {
            if self.current_char() == '\\' {
                self.advance();
                if self.is_eof() {
                    return Err(LexerError::UnterminatedString(start_pos));
                }
                // Handle escape sequences
                match self.current_char() {
                    'n' => value.push('\n'),
                    't' => value.push('\t'),
                    'r' => value.push('\r'),
                    '\\' => value.push('\\'),
                    '"' => value.push('"'),
                    '`' => value.push('`'),
                    _ => {
                        value.push('\\');
                        value.push(self.current_char());
                    }
                }
            } else {
                value.push(self.current_char());
            }
            self.advance();
        }

        if self.is_eof() {
            return Err(LexerError::UnterminatedString(start_pos));
        }

        self.advance(); // skip closing quote
        Ok(Token::String(value))
    }

    fn read_number(&mut self) -> Result<Token, LexerError> {
        let start_pos = self.position;
        let mut num_str = String::new();

        // Handle negative sign
        if self.current_char() == '-' {
            num_str.push('-');
            self.advance();
        }

        // Read digits
        while !self.is_eof() && self.current_char().is_ascii_digit() {
            num_str.push(self.current_char());
            self.advance();
        }

        // Check for decimal point
        if !self.is_eof() && self.current_char() == '.' {
            num_str.push('.');
            self.advance();

            while !self.is_eof() && self.current_char().is_ascii_digit() {
                num_str.push(self.current_char());
                self.advance();
            }

            // Check for duration unit (ms, s, m, h, us, ns)
            if !self.is_eof() && self.current_char().is_alphabetic() {
                let mut unit = String::new();
                while !self.is_eof() && self.current_char().is_alphabetic() {
                    unit.push(self.current_char());
                    self.advance();
                }
                // Return as identifier with number prefix for duration parsing
                return Ok(Token::Identifier(format!("{}{}", num_str, unit)));
            }

            num_str
                .parse::<f64>()
                .map(Token::Float)
                .map_err(|_| LexerError::InvalidNumber(num_str, start_pos))
        } else {
            // Check for duration unit on integer
            if !self.is_eof() && self.current_char().is_alphabetic() {
                let mut unit = String::new();
                while !self.is_eof() && self.current_char().is_alphabetic() {
                    unit.push(self.current_char());
                    self.advance();
                }
                // Return as identifier with number prefix for duration parsing
                return Ok(Token::Identifier(format!("{}{}", num_str, unit)));
            }

            num_str
                .parse::<i64>()
                .map(Token::Integer)
                .map_err(|_| LexerError::InvalidNumber(num_str, start_pos))
        }
    }

    fn read_identifier(&mut self) -> Result<Token, LexerError> {
        let mut ident = String::new();

        while !self.is_eof()
            && (self.current_char().is_alphanumeric() || self.current_char() == '_')
        {
            ident.push(self.current_char());
            self.advance();
        }

        // Check for keywords
        let token = match ident.as_str() {
            "true" => Token::True,
            "false" => Token::False,
            _ => Token::Identifier(ident),
        };

        Ok(token)
    }
}

/// Lexer errors
#[derive(Debug, Clone)]
pub enum LexerError {
    UnexpectedCharacter(char, usize),
    UnterminatedString(usize),
    InvalidNumber(String, usize),
}

impl fmt::Display for LexerError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            LexerError::UnexpectedCharacter(ch, pos) => {
                write!(f, "Unexpected character '{}' at position {}", ch, pos)
            }
            LexerError::UnterminatedString(pos) => {
                write!(f, "Unterminated string starting at position {}", pos)
            }
            LexerError::InvalidNumber(num, pos) => {
                write!(f, "Invalid number '{}' at position {}", num, pos)
            }
        }
    }
}

impl std::error::Error for LexerError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_simple_tokens() {
        let mut lexer = Lexer::new("{ }");
        let tokens = lexer.tokenize().unwrap();
        assert_eq!(tokens, vec![Token::LBrace, Token::RBrace, Token::Eof]);
    }

    #[test]
    fn test_operators() {
        let mut lexer = Lexer::new("= != > >= < <= =~ !~ && ||");
        let tokens = lexer.tokenize().unwrap();
        assert_eq!(
            tokens,
            vec![
                Token::Eq,
                Token::NotEq,
                Token::Gt,
                Token::Gte,
                Token::Lt,
                Token::Lte,
                Token::Regex,
                Token::NotRegex,
                Token::And,
                Token::Or,
                Token::Eof
            ]
        );
    }

    #[test]
    fn test_string() {
        let mut lexer = Lexer::new(r#""hello world""#);
        let tokens = lexer.tokenize().unwrap();
        assert_eq!(
            tokens,
            vec![Token::String("hello world".to_string()), Token::Eof]
        );
    }

    #[test]
    fn test_numbers() {
        let mut lexer = Lexer::new("42 3.14 100ms");
        let tokens = lexer.tokenize().unwrap();
        assert_eq!(
            tokens,
            vec![
                Token::Integer(42),
                Token::Float(3.14),
                Token::Identifier("100ms".to_string()),
                Token::Eof
            ]
        );
    }

    #[test]
    fn test_backtick_strings() {
        let mut lexer = Lexer::new("`hello world`");
        let tokens = lexer.tokenize().unwrap();
        assert_eq!(
            tokens,
            vec![Token::String("hello world".to_string()), Token::Eof]
        );
    }

    #[test]
    fn test_mixed_quote_styles() {
        let mut lexer = Lexer::new(r#"`backtick` "doublequote""#);
        let tokens = lexer.tokenize().unwrap();
        assert_eq!(
            tokens,
            vec![
                Token::String("backtick".to_string()),
                Token::String("doublequote".to_string()),
                Token::Eof
            ]
        );
    }
}
