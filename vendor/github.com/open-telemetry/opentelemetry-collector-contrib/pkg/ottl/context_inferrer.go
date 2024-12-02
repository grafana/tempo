// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"

import "math"

var (
	defaultContextInferPriority = []string{
		"log",
		"metric",
		"datapoint",
		"spanevent",
		"span",
		"resource",
		"scope",
		"instrumentation_scope",
	}
)

// contextInferrer is an interface used to infer the OTTL context from statements paths.
type contextInferrer interface {
	// infer returns the OTTL context inferred from the given statements paths.
	infer(statements []string) (string, error)
}

type priorityContextInferrer struct {
	contextPriority map[string]int
}

func (s *priorityContextInferrer) infer(statements []string) (string, error) {
	var inferredContext string
	var inferredContextPriority int

	for _, statement := range statements {
		parsed, err := parseStatement(statement)
		if err != nil {
			return inferredContext, err
		}

		for _, p := range getParsedStatementPaths(parsed) {
			pathContextPriority, ok := s.contextPriority[p.Context]
			if !ok {
				// Lowest priority
				pathContextPriority = math.MaxInt
			}

			if inferredContext == "" || pathContextPriority < inferredContextPriority {
				inferredContext = p.Context
				inferredContextPriority = pathContextPriority
			}
		}
	}

	return inferredContext, nil
}

// defaultPriorityContextInferrer is like newPriorityContextInferrer, but using the default
// context priorities and ignoring unknown/non-prioritized contexts.
func defaultPriorityContextInferrer() contextInferrer {
	return newPriorityContextInferrer(defaultContextInferPriority)
}

// newPriorityContextInferrer creates a new priority-based context inferrer.
// To infer the context, it compares all [ottl.Path.Context] values, prioritizing them based
// on the provide contextsPriority argument, the lower the context position is in the array,
// the more priority it will have over other items.
// If unknown/non-prioritized contexts are found on the statements, they can be either ignored
// or considered when no other prioritized context is found. To skip unknown contexts, the
// ignoreUnknownContext argument must be set to false.
func newPriorityContextInferrer(contextsPriority []string) contextInferrer {
	contextPriority := make(map[string]int, len(contextsPriority))
	for i, ctx := range contextsPriority {
		contextPriority[ctx] = i
	}
	return &priorityContextInferrer{
		contextPriority: contextPriority,
	}
}
