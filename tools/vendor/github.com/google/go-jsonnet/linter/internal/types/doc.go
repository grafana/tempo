// Package types provides type inference functionality. 
//
// Even though Jsonnet doesn't have a concept of static types
// we can infer for each expression what values it can take.
// Of course we cannot do this accurately at all times, but even
// coarse grained information about "types" can help with some bugs.
// We are mostly interested in simple issues - like using a nonexistent
// field of an object or treating an array like a function.
//
// Main assumptions:
// * It has to work well with existing programs.
// * It needs to be conservative - strong preference for false negatives over false positives.
// * It must be practical to use with existing Jsonnet code.
// * It should "preserve abstractions". Calling a function with some specific arguments should not cause errors in the definition.
//   In general, reasoning about the definition from usage is not allowed.
//
// First of all type processing split into two very distinct phases:
// 1) Finding a type - an upper bound for the set of possible values for each expression.
// 2) Checking all expressions in the AST using this information.
package types

