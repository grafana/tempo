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
    wrappedSpansetPipeline Pipeline
    spansetPipeline Pipeline
    spansetFilter SpansetFilter
    scalarFilter ScalarFilter
    scalarFilterOperation int

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
%type <wrappedSpansetPipeline> wrappedSpansetPipeline
%type <spansetPipeline> spansetPipeline
%type <spansetFilter> spansetFilter
%type <scalarFilter> scalarFilter
%type <scalarFilterOperation> scalarFilterOperation

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
                        START END IDURATION CHILDCOUNT NAME STATUS
                        PARENT RESOURCE SPAN
                        COUNT AVG MAX MIN SUM
                        BY COALESCE

// Operators are listed with increasing precedence.
%left <binOp> PIPE
%left <binOp> EQ NEQ LT LTE GT GTE NRE RE DESC TILDE
%left <binOp> AND OR NOT
%left <binOp> ADD SUB
%left <binOp> MUL DIV MOD
%right <binOp> POW
%%

// **********************
// Pipeline
// **********************
root:
    spansetPipeline                             { yylex.(*lexer).expr = newRootExpr($1) }
  ;

groupOperation:
    BY OPEN_PARENS fieldExpression CLOSE_PARENS { $$ = newGroupOperation($3) }
  ;

coalesceOperation:
    COALESCE OPEN_PARENS CLOSE_PARENS           { $$ = newCoalesceOperation() }
  ;

// **********************
// Spanset Expressions
// **********************
spansetExpression:
    OPEN_PARENS spansetExpression CLOSE_PARENS   { $$ = $2 }
  | spansetExpression AND   spansetExpression    { $$ = newSpansetOperation(opSpansetAnd, $1, $3) }
  | spansetExpression GT    spansetExpression    { $$ = newSpansetOperation(opSpansetChild, $1, $3) }
  | spansetExpression DESC  spansetExpression    { $$ = newSpansetOperation(opSpansetDescendant, $1, $3) }
  | spansetExpression OR    spansetExpression    { $$ = newSpansetOperation(opSpansetUnion, $1, $3) }
  | spansetExpression TILDE spansetExpression    { $$ = newSpansetOperation(opSpansetSibling, $1, $3) }
  | wrappedSpansetPipeline                       { $$ = $1 }
  | spansetFilter                                { $$ = $1 } 
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
  | spansetPipeline PIPE coalesceOperation     { $$ = $1.addItem($3)  }  // can't start with coalesce
  ;

spansetFilter:
    OPEN_BRACE CLOSE_BRACE                      { $$ = newSpansetFilter(newStaticBool(true)) }
  | OPEN_BRACE fieldExpression CLOSE_BRACE      { $$ = newSpansetFilter($2) } // jpe - fieldExpression must resolve to a boolean
  ;

scalarFilter:
    scalarExpression scalarFilterOperation scalarExpression        { $$ = newScalarFilter($2, $1, $3) }
  | static           scalarFilterOperation scalarExpression        { $$ = newScalarFilter($2, $1, $3) }
  | scalarExpression scalarFilterOperation static                  { $$ = newScalarFilter($2, $1, $3) }
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
scalarExpression:
    OPEN_PARENS scalarExpression CLOSE_PARENS  { $$ = $2 }                                   
  | scalarExpression ADD scalarExpression      { $$ = newScalarOperation(opAdd, $1, $3) }
  | scalarExpression SUB scalarExpression      { $$ = newScalarOperation(opSub, $1, $3) }
  | scalarExpression MUL scalarExpression      { $$ = newScalarOperation(opMult, $1, $3) }
  | scalarExpression DIV scalarExpression      { $$ = newScalarOperation(opDiv, $1, $3) }
  | scalarExpression MOD scalarExpression      { $$ = newScalarOperation(opMod, $1, $3) }
  | scalarExpression POW scalarExpression      { $$ = newScalarOperation(opPower, $1, $3) }
  | wrappedScalarPipeline                      { $$ = $1 }
  | aggregate                                  { $$ = $1 }
  ;

wrappedScalarPipeline:
    OPEN_PARENS scalarPipeline CLOSE_PARENS    { $$ = $2 }
  ;

scalarPipeline:
    spansetPipeline PIPE scalarExpression      { $$ = $1.addItem($3)  }
  ;

aggregate:  // jpe isValid - fieldExpression must be numeric. all statics must be numeric
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
    START          { $$ = newIntrinsic(intrinsicStart)      }
  | END            { $$ = newIntrinsic(intrinsicEnd)        }
  | IDURATION      { $$ = newIntrinsic(intrinsicDuration)   }
  | CHILDCOUNT     { $$ = newIntrinsic(intrinsicChildCount) }
  | NAME           { $$ = newIntrinsic(intrinsicName)       }
  | STATUS         { $$ = newIntrinsic(intrinsicStatus)     }
  | PARENT         { $$ = newIntrinsic(intrinsicParent)     }
  ;

attributeField:
    DOT IDENTIFIER                 { $$ = newAttribute($2)               }
  | RESOURCE DOT IDENTIFIER        { $$ = newScopedAttribute(attributeScopeResource, $3) }
  | SPAN DOT IDENTIFIER            { $$ = newScopedAttribute(attributeScopeSpan, $3) }
  | PARENT DOT IDENTIFIER          { $$ = newScopedAttribute(attributeScopeParent, $3) }
  | attributeField DOT IDENTIFIER  { $$ = appendAttribute($1, $3) }
  ;