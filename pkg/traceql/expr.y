%{
package traceql

import (
  "time"
)
%}

%start root

%union {
    root RootExpr
    groupOperation GroupOperation
    coalesceOperation CoalesceOperation

    spansetExpression SpansetExpression
    spansetPipelineExpression SpansetExpression
    wrappedSpansetPipeline Pipeline
    spansetPipeline Pipeline
    spansetFilter SpansetFilter
    scalarFilter ScalarFilter
    scalarFilterOperation int

    scalarPipelineExpressionFilter ScalarFilter
    scalarPipelineExpression ScalarExpression
    scalarExpression ScalarExpression
    wrappedScalarPipeline Pipeline
    scalarPipeline Pipeline
    aggregate Aggregate

    fieldExpression FieldExpression
    static Static
    intrinsicField Static
    attributeField Attribute

    binOp       int
    staticInt   int
    staticStr   string
    staticFloat float64
    staticDuration time.Duration
}

%type <RootExpr> root
%type <groupOperation> groupOperation
%type <coalesceOperation> coalesceOperation

%type <spansetExpression> spansetExpression
%type <spansetPipelineExpression> spansetPipelineExpression
%type <wrappedSpansetPipeline> wrappedSpansetPipeline
%type <spansetPipeline> spansetPipeline
%type <spansetFilter> spansetFilter
%type <scalarFilter> scalarFilter
%type <scalarFilterOperation> scalarFilterOperation

%type <scalarPipelineExpressionFilter> scalarPipelineExpressionFilter
%type <scalarPipelineExpression> scalarPipelineExpression
%type <scalarExpression> scalarExpression
%type <wrappedScalarPipeline> wrappedScalarPipeline
%type <scalarPipeline> scalarPipeline
%type <aggregate> aggregate 

%type <fieldExpression> fieldExpression
%type <static> static
%type <intrinsicField> intrinsicField
%type <attributeField> attributeField

%token <staticStr>      IDENTIFIER STRING
%token <staticInt>      INTEGER
%token <staticFloat>    FLOAT
%token <staticDuration> DURATION
%token <val>            DOT OPEN_BRACE CLOSE_BRACE OPEN_PARENS CLOSE_PARENS
                        NIL TRUE FALSE STATUS_ERROR STATUS_OK STATUS_UNSET
                        IDURATION CHILDCOUNT NAME STATUS
                        PARENT RESOURCE SPAN
                        COUNT AVG MAX MIN SUM
                        BY COALESCE

// Operators are listed with increasing precedence.
%left <binOp> PIPE
%left <binOp> AND OR
%left <binOp> EQ NEQ LT LTE GT GTE NRE RE DESC TILDE
%left <binOp> ADD SUB
%left <binOp> NOT
%left <binOp> MUL DIV MOD
%right <binOp> POW
%%

// **********************
// Pipeline
// **********************
root:
    spansetPipeline                             { yylex.(*lexer).expr = newRootExpr($1) }
  | spansetPipelineExpression                   { yylex.(*lexer).expr = newRootExpr($1) }
  | scalarPipelineExpressionFilter              { yylex.(*lexer).expr = newRootExpr($1) }
  ;

// **********************
// Spanset Expressions
// **********************
spansetPipelineExpression: // shares the same operators as spansetExpression. split out for readability
    OPEN_PARENS spansetPipelineExpression CLOSE_PARENS           { $$ = $2 }
  | spansetPipelineExpression AND   spansetPipelineExpression    { $$ = newSpansetOperation(opSpansetAnd, $1, $3) }
  | spansetPipelineExpression GT    spansetPipelineExpression    { $$ = newSpansetOperation(opSpansetChild, $1, $3) }
  | spansetPipelineExpression DESC  spansetPipelineExpression    { $$ = newSpansetOperation(opSpansetDescendant, $1, $3) }
  | spansetPipelineExpression OR    spansetPipelineExpression    { $$ = newSpansetOperation(opSpansetUnion, $1, $3) }
  | spansetPipelineExpression TILDE spansetPipelineExpression    { $$ = newSpansetOperation(opSpansetSibling, $1, $3) }
  | wrappedSpansetPipeline                                       { $$ = $1 }
  ;

wrappedSpansetPipeline:
    OPEN_PARENS spansetPipeline CLOSE_PARENS   { $$ = $2 }

spansetPipeline:
    spansetExpression                          { $$ = newPipeline($1) }
  | scalarFilter                               { $$ = newPipeline($1) }
  | groupOperation                             { $$ = newPipeline($1) }
  | spansetPipeline PIPE scalarFilter          { $$ = $1.addItem($3)  }
  | spansetPipeline PIPE spansetExpression     { $$ = $1.addItem($3)  }
  | spansetPipeline PIPE groupOperation        { $$ = $1.addItem($3)  }
  | spansetPipeline PIPE coalesceOperation     { $$ = $1.addItem($3)  }
  ;

groupOperation:
    BY OPEN_PARENS fieldExpression CLOSE_PARENS { $$ = newGroupOperation($3) }
  ;

coalesceOperation:
    COALESCE OPEN_PARENS CLOSE_PARENS           { $$ = newCoalesceOperation() }
  ;

spansetExpression: // shares the same operators as scalarPipelineExpression. split out for readability
    OPEN_PARENS spansetExpression CLOSE_PARENS   { $$ = $2 }
  | spansetExpression AND   spansetExpression    { $$ = newSpansetOperation(opSpansetAnd, $1, $3) }
  | spansetExpression GT    spansetExpression    { $$ = newSpansetOperation(opSpansetChild, $1, $3) }
  | spansetExpression DESC  spansetExpression    { $$ = newSpansetOperation(opSpansetDescendant, $1, $3) }
  | spansetExpression OR    spansetExpression    { $$ = newSpansetOperation(opSpansetUnion, $1, $3) }
  | spansetExpression TILDE spansetExpression    { $$ = newSpansetOperation(opSpansetSibling, $1, $3) }
  | spansetFilter                                { $$ = $1 } 
  ;

spansetFilter:
    OPEN_BRACE fieldExpression CLOSE_BRACE      { $$ = newSpansetFilter($2) }
  ;

scalarFilter:
    scalarExpression          scalarFilterOperation scalarExpression          { $$ = newScalarFilter($2, $1, $3) }
  | static                    scalarFilterOperation scalarExpression          { $$ = newScalarFilter($2, $1, $3) }
  | scalarExpression          scalarFilterOperation static                    { $$ = newScalarFilter($2, $1, $3) }
  ;

scalarFilterOperation:
    EQ     { $$ = opEqual        }
  | NEQ    { $$ = opNotEqual     }
  | LT     { $$ = opLess         }
  | LTE    { $$ = opLessEqual    }
  | GT     { $$ = opGreater      }
  | GTE    { $$ = opGreaterEqual }
  ;

// **********************
// Scalar Expressions
// **********************
scalarPipelineExpressionFilter:
    scalarPipelineExpression scalarFilterOperation scalarPipelineExpression { $$ = newScalarFilter($2, $1, $3) }
  | static                   scalarFilterOperation scalarPipelineExpression { $$ = newScalarFilter($2, $1, $3) }
  | scalarPipelineExpression scalarFilterOperation static                   { $$ = newScalarFilter($2, $1, $3) }
  ;

scalarPipelineExpression: // shares the same operators as scalarExpression. split out for readability
    OPEN_PARENS scalarPipelineExpression CLOSE_PARENS        { $$ = $2 }                                   
  | scalarPipelineExpression ADD scalarPipelineExpression    { $$ = newScalarOperation(opAdd, $1, $3) }
  | scalarPipelineExpression SUB scalarPipelineExpression    { $$ = newScalarOperation(opSub, $1, $3) }
  | scalarPipelineExpression MUL scalarPipelineExpression    { $$ = newScalarOperation(opMult, $1, $3) }
  | scalarPipelineExpression DIV scalarPipelineExpression    { $$ = newScalarOperation(opDiv, $1, $3) }
  | scalarPipelineExpression MOD scalarPipelineExpression    { $$ = newScalarOperation(opMod, $1, $3) }
  | scalarPipelineExpression POW scalarPipelineExpression    { $$ = newScalarOperation(opPower, $1, $3) }
  | wrappedScalarPipeline                                    { $$ = $1 }

wrappedScalarPipeline:
    OPEN_PARENS scalarPipeline CLOSE_PARENS    { $$ = $2 }
  ;

scalarPipeline:
    spansetPipeline PIPE scalarExpression      { $$ = $1.addItem($3)  }
  ;

scalarExpression: // shares the same operators as scalarPipelineExpression. split out for readability
    OPEN_PARENS scalarExpression CLOSE_PARENS  { $$ = $2 }                                   
  | scalarExpression ADD scalarExpression      { $$ = newScalarOperation(opAdd, $1, $3) }
  | scalarExpression SUB scalarExpression      { $$ = newScalarOperation(opSub, $1, $3) }
  | scalarExpression MUL scalarExpression      { $$ = newScalarOperation(opMult, $1, $3) }
  | scalarExpression DIV scalarExpression      { $$ = newScalarOperation(opDiv, $1, $3) }
  | scalarExpression MOD scalarExpression      { $$ = newScalarOperation(opMod, $1, $3) }
  | scalarExpression POW scalarExpression      { $$ = newScalarOperation(opPower, $1, $3) }
  | aggregate                                  { $$ = $1 }
  ;

aggregate:
    COUNT OPEN_PARENS CLOSE_PARENS                { $$ = newAggregate(aggregateCount, nil) }
  | MAX OPEN_PARENS fieldExpression CLOSE_PARENS  { $$ = newAggregate(aggregateMax, $3) }
  | MIN OPEN_PARENS fieldExpression CLOSE_PARENS  { $$ = newAggregate(aggregateMin, $3) }
  | AVG OPEN_PARENS fieldExpression CLOSE_PARENS  { $$ = newAggregate(aggregateAvg, $3) }
  | SUM OPEN_PARENS fieldExpression CLOSE_PARENS  { $$ = newAggregate(aggregateSum, $3) }
  ;

// **********************
// FieldExpressions
// **********************
fieldExpression:
    OPEN_PARENS fieldExpression CLOSE_PARENS { $$ = $2 }                                   
  | fieldExpression ADD fieldExpression      { $$ = newBinaryOperation(opAdd, $1, $3) }
  | fieldExpression SUB fieldExpression      { $$ = newBinaryOperation(opSub, $1, $3) }
  | fieldExpression MUL fieldExpression      { $$ = newBinaryOperation(opMult, $1, $3) }
  | fieldExpression DIV fieldExpression      { $$ = newBinaryOperation(opDiv, $1, $3) }
  | fieldExpression MOD fieldExpression      { $$ = newBinaryOperation(opMod, $1, $3) }
  | fieldExpression EQ fieldExpression       { $$ = newBinaryOperation(opEqual, $1, $3) }
  | fieldExpression NEQ fieldExpression      { $$ = newBinaryOperation(opNotEqual, $1, $3) }
  | fieldExpression LT fieldExpression       { $$ = newBinaryOperation(opLess, $1, $3) }
  | fieldExpression LTE fieldExpression      { $$ = newBinaryOperation(opLessEqual, $1, $3) }
  | fieldExpression GT fieldExpression       { $$ = newBinaryOperation(opGreater, $1, $3) }
  | fieldExpression GTE fieldExpression      { $$ = newBinaryOperation(opGreaterEqual, $1, $3) }
  | fieldExpression RE fieldExpression       { $$ = newBinaryOperation(opRegex, $1, $3) }
  | fieldExpression NRE fieldExpression      { $$ = newBinaryOperation(opNotRegex, $1, $3) }
  | fieldExpression POW fieldExpression      { $$ = newBinaryOperation(opPower, $1, $3) }
  | fieldExpression AND fieldExpression      { $$ = newBinaryOperation(opAnd, $1, $3) }
  | fieldExpression OR fieldExpression       { $$ = newBinaryOperation(opOr, $1, $3) }
  | SUB fieldExpression                      { $$ = newUnaryOperation(opSub, $2) }
  | NOT fieldExpression                      { $$ = newUnaryOperation(opNot, $2) }
  | static                                   { $$ = $1 }
  | intrinsicField                           { $$ = $1 }
  | attributeField                           { $$ = $1 }
  ;

// **********************
// Statics
// **********************
static:
    STRING        { $$ = newStaticString($1)          }
  | INTEGER       { $$ = newStaticInt($1)             }
  | FLOAT         { $$ = newStaticFloat($1)           }
  | TRUE          { $$ = newStaticBool(true)          }
  | FALSE         { $$ = newStaticBool(false)         }
  | NIL           { $$ = newStaticNil()               }
  | DURATION      { $$ = newStaticDuration($1)        }
  | STATUS_OK     { $$ = newStaticStatus(statusOk)    }
  | STATUS_ERROR  { $$ = newStaticStatus(statusError) }
  | STATUS_UNSET  { $$ = newStaticStatus(statusUnset) }
  ;

intrinsicField:
    IDURATION      { $$ = newIntrinsic(intrinsicDuration)   }
  | CHILDCOUNT     { $$ = newIntrinsic(intrinsicChildCount) }
  | NAME           { $$ = newIntrinsic(intrinsicName)       }
  | STATUS         { $$ = newIntrinsic(intrinsicStatus)     }
  | PARENT         { $$ = newIntrinsic(intrinsicParent)     }
  ;

// jpe - nested scopes? parent.resource, parent.parent?
// how to select an attribute on the parent called span?
attributeField:
    DOT IDENTIFIER                 { $$ = newAttribute($2)               }
  | RESOURCE DOT IDENTIFIER        { $$ = newScopedAttribute(attributeScopeResource, $3) }
  | SPAN DOT IDENTIFIER            { $$ = newScopedAttribute(attributeScopeSpan, $3) }
  | PARENT DOT IDENTIFIER          { $$ = newScopedAttribute(attributeScopeParent, $3) }
  | attributeField DOT IDENTIFIER  { $$ = appendAttribute($1, $3) }
  ;