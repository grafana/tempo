<!-- TOC depthFrom:2 insertAnchor:true updateOnSave:true -->

- [v2](#v2)

<!-- /TOC -->

<a id="markdown-v2" name="v2"></a>
## v2

v2 was released in November 2020. It contains the following changes, some of
which are backwards-incompatible:

- Added optional `LexString()` and `LexBytes()` methods that lexer
  definitions can implement to fast-path lexing of bytes and strings.
- A new stateful lexer has been added.
- A `filename` must now be passed to all `Parse*()` and `Lex*()` methods.
- The `text/scanner` lexer no longer automatically unquotes strings or
  supports arbitary length single quoted strings. The tokens it produces are
  identical to that of the `text/scanner` package. Use `Unquote()` to remove
  quotes.
- `Tok` and `EndTok` will no longer be populated.
- If a field named `Token []lexer.Token` exists it will be populated with the
  raw tokens that the node parsed from the lexer.
- Support capturing directly into lexer.Token fields. eg.

      type ast struct {
          Head lexer.Token   `@Ident`
          Tail []lexer.Token `@(Ident*)`
      }
- Add an `experimental/codegen` for stateful lexers. This provides ~10x
  performance improvement with zero garbage when lexing strings.
- The `regex` lexer has been removed.
- The `ebnf` lexer has been removed.
- All future work on lexing will be put into the stateful lexer.
- The need for `DropToken` has been removed.

