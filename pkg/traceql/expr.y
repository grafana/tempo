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
    selectOperation SelectOperation
    selectArgs []FieldExpression

    spansetExpression SpansetExpression
    spansetPipelineExpression SpansetExpression
    wrappedSpansetPipeline Pipeline
    spansetPipeline Pipeline
    spansetFilter *SpansetFilter
    scalarFilter ScalarFilter
    scalarFilterOperation Operator

    scalarPipelineExpressionFilter ScalarFilter
    scalarPipelineExpression ScalarExpression
    scalarExpression ScalarExpression
    wrappedScalarPipeline Pipeline
    scalarPipeline Pipeline
    aggregate Aggregate

    fieldExpression FieldExpression
    static Static
    intrinsicField Attribute
    attributeField Attribute

    binOp       Operator
    staticInt   int
    staticStr   string
    staticFloat float64
    staticDuration time.Duration
}

%type <RootExpr> root
%type <groupOperation> groupOperation
%type <coalesceOperation> coalesceOperation
%type <selectOperation> selectOperation
%type <selectArgs> selectArgs

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
%token <val>            DOT OPEN_BRACE CLOSE_BRACE OPEN_PARENS CLOSE_PARENS COMMA
                        NIL TRUE FALSE STATUS_ERROR STATUS_OK STATUS_UNSET
                        KIND_UNSPECIFIED KIND_INTERNAL KIND_SERVER KIND_CLIENT KIND_PRODUCER KIND_CONSUMER
                        IDURATION CHILDCOUNT NAME STATUS STATUS_MESSAGE PARENT KIND ROOTNAME ROOTSERVICENAME TRACEDURATION
                        PARENT_DOT RESOURCE_DOT SPAN_DOT
                        COUNT AVG MAX MIN SUM
                        BY COALESCE SELECT
                        END_ATTRIBUTE

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
  | spansetPipelineExpression AND   spansetPipelineExpression    { $$ = newSpansetOperation(OpSpansetAnd, $1, $3) }
  | spansetPipelineExpression GT    spansetPipelineExpression    { $$ = newSpansetOperation(OpSpansetChild, $1, $3) }
  | spansetPipelineExpression DESC  spansetPipelineExpression    { $$ = newSpansetOperation(OpSpansetDescendant, $1, $3) }
  | spansetPipelineExpression OR    spansetPipelineExpression    { $$ = newSpansetOperation(OpSpansetUnion, $1, $3) }
  | spansetPipelineExpression TILDE spansetPipelineExpression    { $$ = newSpansetOperation(OpSpansetSibling, $1, $3) }
  | wrappedSpansetPipeline                                       { $$ = $1 }
  ;

wrappedSpansetPipeline:
    OPEN_PARENS spansetPipeline CLOSE_PARENS   { $$ = $2 }

spansetPipeline:
    spansetExpression                          { $$ = newPipeline($1) }
  | scalarFilter                               { $$ = newPipeline($1) }
  | groupOperation                             { $$ = newPipeline($1) }
  | selectOperation                            { $$ = newPipeline($1) }
  | spansetPipeline PIPE spansetExpression     { $$ = $1.addItem($3)  }
  | spansetPipeline PIPE scalarFilter          { $$ = $1.addItem($3)  }
  | spansetPipeline PIPE groupOperation        { $$ = $1.addItem($3)  }
  | spansetPipeline PIPE coalesceOperation     { $$ = $1.addItem($3)  }
  | spansetPipeline PIPE selectOperation       { $$ = $1.addItem($3)  }
  ;

groupOperation:
    BY OPEN_PARENS fieldExpression CLOSE_PARENS { $$ = newGroupOperation($3) }
  ;

coalesceOperation:
    COALESCE OPEN_PARENS CLOSE_PARENS           { $$ = newCoalesceOperation() }
  ;

selectOperation:
    SELECT OPEN_PARENS selectArgs CLOSE_PARENS { $$ = newSelectOperation($3) }
  ;

selectArgs:
    fieldExpression                  { $$ = []FieldExpression{$1} }
  | selectArgs COMMA fieldExpression { $$ = append($1, $3) }
  ;

spansetExpression: // shares the same operators as scalarPipelineExpression. split out for readability
    OPEN_PARENS spansetExpression CLOSE_PARENS   { $$ = $2 }
  | spansetExpression AND   spansetExpression    { $$ = newSpansetOperation(OpSpansetAnd, $1, $3) }
  | spansetExpression GT    spansetExpression    { $$ = newSpansetOperation(OpSpansetChild, $1, $3) }
  | spansetExpression DESC  spansetExpression    { $$ = newSpansetOperation(OpSpansetDescendant, $1, $3) }
  | spansetExpression OR    spansetExpression    { $$ = newSpansetOperation(OpSpansetUnion, $1, $3) }
  | spansetExpression TILDE spansetExpression    { $$ = newSpansetOperation(OpSpansetSibling, $1, $3) }
  | spansetFilter                                { $$ = $1 } 
  ;

spansetFilter:
    OPEN_BRACE CLOSE_BRACE                      { $$ = newSpansetFilter(NewStaticBool(true)) }
  | OPEN_BRACE fieldExpression CLOSE_BRACE      { $$ = newSpansetFilter($2) }
  ;

scalarFilter:
    scalarExpression          scalarFilterOperation scalarExpression          { $$ = newScalarFilter($2, $1, $3) }
  ;

scalarFilterOperation:
    EQ     { $$ = OpEqual        }
  | NEQ    { $$ = OpNotEqual     }
  | LT     { $$ = OpLess         }
  | LTE    { $$ = OpLessEqual    }
  | GT     { $$ = OpGreater      }
  | GTE    { $$ = OpGreaterEqual }
  ;

// **********************
// Scalar Expressions
// **********************
scalarPipelineExpressionFilter:
    scalarPipelineExpression scalarFilterOperation scalarPipelineExpression { $$ = newScalarFilter($2, $1, $3) }
  | scalarPipelineExpression scalarFilterOperation static                   { $$ = newScalarFilter($2, $1, $3) }
  ;

scalarPipelineExpression: // shares the same operators as scalarExpression. split out for readability
    OPEN_PARENS scalarPipelineExpression CLOSE_PARENS        { $$ = $2 }                                   
  | scalarPipelineExpression ADD scalarPipelineExpression    { $$ = newScalarOperation(OpAdd, $1, $3) }
  | scalarPipelineExpression SUB scalarPipelineExpression    { $$ = newScalarOperation(OpSub, $1, $3) }
  | scalarPipelineExpression MUL scalarPipelineExpression    { $$ = newScalarOperation(OpMult, $1, $3) }
  | scalarPipelineExpression DIV scalarPipelineExpression    { $$ = newScalarOperation(OpDiv, $1, $3) }
  | scalarPipelineExpression MOD scalarPipelineExpression    { $$ = newScalarOperation(OpMod, $1, $3) }
  | scalarPipelineExpression POW scalarPipelineExpression    { $$ = newScalarOperation(OpPower, $1, $3) }
  | wrappedScalarPipeline                                    { $$ = $1 }
  ;

wrappedScalarPipeline:
    OPEN_PARENS scalarPipeline CLOSE_PARENS    { $$ = $2 }
  ;

scalarPipeline:
    spansetPipeline PIPE aggregate      { $$ = $1.addItem($3)  }
  ;

scalarExpression: // shares the same operators as scalarPipelineExpression. split out for readability
    OPEN_PARENS scalarExpression CLOSE_PARENS  { $$ = $2 }                                   
  | scalarExpression ADD scalarExpression      { $$ = newScalarOperation(OpAdd, $1, $3) }
  | scalarExpression SUB scalarExpression      { $$ = newScalarOperation(OpSub, $1, $3) }
  | scalarExpression MUL scalarExpression      { $$ = newScalarOperation(OpMult, $1, $3) }
  | scalarExpression DIV scalarExpression      { $$ = newScalarOperation(OpDiv, $1, $3) }
  | scalarExpression MOD scalarExpression      { $$ = newScalarOperation(OpMod, $1, $3) }
  | scalarExpression POW scalarExpression      { $$ = newScalarOperation(OpPower, $1, $3) }
  | aggregate                                  { $$ = $1 }
  | INTEGER                                    { $$ = NewStaticInt($1)              }
  | FLOAT                                      { $$ = NewStaticFloat($1)            }
  | DURATION                                   { $$ = NewStaticDuration($1)         }
  | SUB INTEGER                                { $$ = NewStaticInt(-$2)             }
  | SUB FLOAT                                  { $$ = NewStaticFloat(-$2)           }
  | SUB DURATION                               { $$ = NewStaticDuration(-$2)        }
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
  | fieldExpression ADD fieldExpression      { $$ = newBinaryOperation(OpAdd, $1, $3) }
  | fieldExpression SUB fieldExpression      { $$ = newBinaryOperation(OpSub, $1, $3) }
  | fieldExpression MUL fieldExpression      { $$ = newBinaryOperation(OpMult, $1, $3) }
  | fieldExpression DIV fieldExpression      { $$ = newBinaryOperation(OpDiv, $1, $3) }
  | fieldExpression MOD fieldExpression      { $$ = newBinaryOperation(OpMod, $1, $3) }
  | fieldExpression EQ fieldExpression       { $$ = newBinaryOperation(OpEqual, $1, $3) }
  | fieldExpression NEQ fieldExpression      { $$ = newBinaryOperation(OpNotEqual, $1, $3) }
  | fieldExpression LT fieldExpression       { $$ = newBinaryOperation(OpLess, $1, $3) }
  | fieldExpression LTE fieldExpression      { $$ = newBinaryOperation(OpLessEqual, $1, $3) }
  | fieldExpression GT fieldExpression       { $$ = newBinaryOperation(OpGreater, $1, $3) }
  | fieldExpression GTE fieldExpression      { $$ = newBinaryOperation(OpGreaterEqual, $1, $3) }
  | fieldExpression RE fieldExpression       { $$ = newBinaryOperation(OpRegex, $1, $3) }
  | fieldExpression NRE fieldExpression      { $$ = newBinaryOperation(OpNotRegex, $1, $3) }
  | fieldExpression POW fieldExpression      { $$ = newBinaryOperation(OpPower, $1, $3) }
  | fieldExpression AND fieldExpression      { $$ = newBinaryOperation(OpAnd, $1, $3) }
  | fieldExpression OR fieldExpression       { $$ = newBinaryOperation(OpOr, $1, $3) }
  | SUB fieldExpression                      { $$ = newUnaryOperation(OpSub, $2) }
  | NOT fieldExpression                      { $$ = newUnaryOperation(OpNot, $2) }
  | static                                   { $$ = $1 }
  | intrinsicField                           { $$ = $1 }
  | attributeField                           { $$ = $1 }
  ;

// **********************
// Statics
// **********************
static:
    STRING           { $$ = NewStaticString($1)           }
  | INTEGER          { $$ = NewStaticInt($1)              }
  | FLOAT            { $$ = NewStaticFloat($1)            }
  | TRUE             { $$ = NewStaticBool(true)           }
  | FALSE            { $$ = NewStaticBool(false)          }
  | NIL              { $$ = NewStaticNil()                }
  | DURATION         { $$ = NewStaticDuration($1)         }
  | STATUS_OK        { $$ = NewStaticStatus(StatusOk)     }
  | STATUS_ERROR     { $$ = NewStaticStatus(StatusError)  }
  | STATUS_UNSET     { $$ = NewStaticStatus(StatusUnset)  } 
  | KIND_UNSPECIFIED { $$ = NewStaticKind(KindUnspecified)}
  | KIND_INTERNAL    { $$ = NewStaticKind(KindInternal)   }
  | KIND_SERVER      { $$ = NewStaticKind(KindServer)     }
  | KIND_CLIENT      { $$ = NewStaticKind(KindClient)     }
  | KIND_PRODUCER    { $$ = NewStaticKind(KindProducer)   }
  | KIND_CONSUMER    { $$ = NewStaticKind(KindConsumer)   }
  ;

intrinsicField:
    IDURATION       { $$ = NewIntrinsic(IntrinsicDuration)         }
  | CHILDCOUNT      { $$ = NewIntrinsic(IntrinsicChildCount)       }
  | NAME            { $$ = NewIntrinsic(IntrinsicName)             }
  | STATUS          { $$ = NewIntrinsic(IntrinsicStatus)           }
  | STATUS_MESSAGE  { $$ = NewIntrinsic(IntrinsicStatusMessage)    }
  | KIND            { $$ = NewIntrinsic(IntrinsicKind)             }
  | PARENT          { $$ = NewIntrinsic(IntrinsicParent)           }
  | ROOTNAME        { $$ = NewIntrinsic(IntrinsicTraceRootSpan)    }
  | ROOTSERVICENAME { $$ = NewIntrinsic(IntrinsicTraceRootService) }
  | TRACEDURATION   { $$ = NewIntrinsic(IntrinsicTraceDuration)    }
  ;

attributeField:
    DOT IDENTIFIER END_ATTRIBUTE                      { $$ = NewAttribute($2)                                      }
  | RESOURCE_DOT IDENTIFIER END_ATTRIBUTE             { $$ = NewScopedAttribute(AttributeScopeResource, false, $2) }
  | SPAN_DOT IDENTIFIER END_ATTRIBUTE                 { $$ = NewScopedAttribute(AttributeScopeSpan, false, $2)     }
  | PARENT_DOT IDENTIFIER END_ATTRIBUTE               { $$ = NewScopedAttribute(AttributeScopeNone, true, $2)      }
  | PARENT_DOT RESOURCE_DOT IDENTIFIER END_ATTRIBUTE  { $$ = NewScopedAttribute(AttributeScopeResource, true, $3)  }
  | PARENT_DOT SPAN_DOT IDENTIFIER END_ATTRIBUTE      { $$ = NewScopedAttribute(AttributeScopeSpan, true, $3)      }
  ;
