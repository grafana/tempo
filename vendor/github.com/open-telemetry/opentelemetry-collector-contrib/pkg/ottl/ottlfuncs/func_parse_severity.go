// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

const (
	// http2xx is a special key that is represents a range from 200 to 299
	http2xx = "2xx"

	// http3xx is a special key that is represents a range from 300 to 399
	http3xx = "3xx"

	// http4xx is a special key that is represents a range from 400 to 499
	http4xx = "4xx"

	// http5xx is a special key that is represents a range from 500 to 599
	http5xx = "5xx"

	minKey = "min"
	maxKey = "max"

	rangeKey  = "range"
	equalsKey = "equals"
)

type ParseSeverityArguments[K any] struct {
	Target  ottl.Getter[K]
	Mapping ottl.PMapGetter[K]
}

func NewParseSeverityFactory[K any]() ottl.Factory[K] {
	return ottl.NewFactory("ParseSeverity", &ParseSeverityArguments[K]{}, createParseSeverityFunction[K])
}

func createParseSeverityFunction[K any](_ ottl.FunctionContext, oArgs ottl.Arguments) (ottl.ExprFunc[K], error) {
	args, ok := oArgs.(*ParseSeverityArguments[K])

	if !ok {
		return nil, errors.New("ParseSeverityFactory args must be of type *ParseSeverityArguments[K")
	}

	return parseSeverity[K](args.Target, args.Mapping), nil
}

func parseSeverity[K any](target ottl.Getter[K], mapping ottl.PMapGetter[K]) ottl.ExprFunc[K] {
	// retrieve the mapping as a literal PMap and use it for all evaluations
	mappingLiteral, ok := ottl.GetLiteralValue(mapping)
	if !ok {
		return func(_ context.Context, _ K) (any, error) {
			return nil, errors.New("severity mapping must be a literal PMap")
		}
	}

	severityMapping := map[string]criteriaSet{}

	// convert the mapping to criteria objects to validate its structure
	severityMap := mappingLiteral.AsRaw()
	for logLevel, criteriaListObj := range severityMap {
		severityMapping[logLevel] = []criteria{}
		criteriaList, ok := criteriaListObj.([]any)
		if !ok {
			return func(_ context.Context, _ K) (any, error) {
				return nil, errors.New("severity mapping criteria must be []any")
			}
		}
		for _, critObj := range criteriaList {
			critMap, ok := critObj.(map[string]any)
			if !ok {
				return func(_ context.Context, _ K) (any, error) {
					return nil, errors.New("severity mapping criteria items must be map[string]any")
				}
			}
			c, err := newCriteriaFromMap(critMap)
			if err != nil {
				return func(_ context.Context, _ K) (any, error) {
					return nil, fmt.Errorf("invalid severity mapping criteria: %w", err)
				}
			}

			severityMapping[logLevel] = append(severityMapping[logLevel], *c)
		}
	}

	return func(ctx context.Context, tCtx K) (any, error) {
		value, err := target.Get(ctx, tCtx)
		if err != nil {
			return nil, fmt.Errorf("could not get log level: %w", err)
		}

		logLevel, err := evaluateSeverity(value, severityMapping)
		if err != nil {
			return nil, fmt.Errorf("could not map log level: %w", err)
		}

		return logLevel, nil
	}
}

func evaluateSeverity(value any, severities map[string]criteriaSet) (string, error) {
	for level, cs := range severities {
		match, err := cs.evaluate(value)
		if err != nil {
			return "", fmt.Errorf("could not evaluate log level of value '%v': %w", value, err)
		}
		if match {
			return level, nil
		}
	}
	return "", fmt.Errorf("no matching log level found for value '%v'", value)
}

type criteriaSet []criteria

func (cs criteriaSet) evaluate(value any) (bool, error) {
	for _, crit := range cs {
		match, err := crit.evaluate(value)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

type criteria struct {
	Equals []string
	Range  *valueRange
}

type valueRange struct {
	Min int64
	Max int64
}

func (c *criteria) evaluate(value any) (bool, error) {
	switch v := value.(type) {
	case string:
		if slices.Contains(c.Equals, v) {
			return true, nil
		}
		return false, nil
	case int64:
		if c.Range != nil {
			if v >= c.Range.Min && v <= c.Range.Max {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported value type: %T", v)
	}
}

func newCriteriaFromMap(m map[string]any) (*criteria, error) {
	crit := &criteria{}

	if equalsObj, ok := m[equalsKey]; ok {
		equalsList, ok := equalsObj.([]any)
		if !ok {
			return nil, errors.New("equals must be a list")
		}
		for _, eq := range equalsList {
			eqStr, ok := eq.(string)
			if !ok {
				return nil, errors.New("equals values must be strings")
			}
			crit.Equals = append(crit.Equals, eqStr)
		}
	}
	if rangeObj, ok := m[rangeKey]; ok {
		switch v := rangeObj.(type) {
		case map[string]any:
			minObj, ok := v[minKey]
			if !ok {
				return nil, errors.New("range must have a min value")
			}
			maxObj, ok := v[maxKey]
			if !ok {
				return nil, errors.New("range must have a max value")
			}
			minInt, ok := minObj.(int64)
			if !ok {
				return nil, errors.New("min must be an int64")
			}
			maxInt, ok := maxObj.(int64)
			if !ok {
				return nil, errors.New("max must be an int64")
			}
			crit.Range = &valueRange{
				Min: minInt,
				Max: maxInt,
			}
		case string:
			switch v {
			case http2xx:
				crit.Range = &valueRange{Min: 200, Max: 299}
			case http3xx:
				crit.Range = &valueRange{Min: 300, Max: 399}
			case http4xx:
				crit.Range = &valueRange{Min: 400, Max: 499}
			case http5xx:
				crit.Range = &valueRange{Min: 500, Max: 599}
			default:
				return nil, fmt.Errorf("unknown range placeholder: %s", v)
			}
		default:
			return nil, errors.New("range must be a map or a known placeholder string")
		}
	}
	return crit, nil
}
