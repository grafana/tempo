// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filterottl // import "github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterottl"

import (
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/expr"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter/filterset"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottllog"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlmetric"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlresource"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
)

const (
	serviceNameStaticStatement        = `resource.attributes["service.name"] == "%v"`
	nameStaticStatement               = `name == "%v"`
	spanKindStaticStatement           = `kind.deprecated_string == "%v"`
	scopeNameStaticStatement          = `instrumentation_scope.name == "%v"`
	scopeVersionStaticStatement       = `instrumentation_scope.version == "%v"`
	attributesStaticStatement         = `attributes["%v"] == %v`
	resourceAttributesStaticStatement = `resource.attributes["%v"] == %v`
	bodyStaticStatement               = `body.string == "%v"`
	severityTextStaticStatement       = `severity_text == "%v"`

	serviceNameRegexStatement        = `IsMatch(resource.attributes["service.name"], "%v")`
	nameRegexStatement               = `IsMatch(name, "%v")`
	spanKindRegexStatement           = `IsMatch(kind.deprecated_string, "%v")`
	scopeNameRegexStatement          = `IsMatch(instrumentation_scope.name, "%v")`
	scopeVersionRegexStatement       = `IsMatch(instrumentation_scope.version, "%v")`
	attributesRegexStatement         = `IsMatch(attributes["%v"], "%v")`
	resourceAttributesRegexStatement = `IsMatch(resource.attributes["%v"], "%v")`
	bodyRegexStatement               = `IsMatch(body.string, "%v")`
	severityTextRegexStatement       = `IsMatch(severity_text, "%v")`

	// Boolean expression for existing severity number matching
	// a -> lr.SeverityNumber() == plog.SeverityNumberUnspecified
	// b -> snm.matchUndefined
	// c -> lr.SeverityNumber() >= snm.minSeverityNumber
	// (a AND b) OR ( NOT a AND c)
	//  a  b  c  X
	//  0  0  0  0
	//  0  0  1  1
	//  0  1  0  0
	//  0  1  1  1
	//  1  0  0  0
	//  1  0  1  0
	//  1  1  0  1
	//  1  1  1  1
	severityNumberStatement = `((severity_number == SEVERITY_NUMBER_UNSPECIFIED and %v) or (severity_number != SEVERITY_NUMBER_UNSPECIFIED and severity_number >= %d))`
)

func NewLogSkipExprBridge(mc *filterconfig.MatchConfig) (expr.BoolExpr[ottllog.TransformContext], error) {
	statements := make([]string, 0, 2)
	if mc.Include != nil {
		if err := mc.Include.ValidateForLogs(); err != nil {
			return nil, err
		}
		statement, err := createStatement(*mc.Include)
		if err != nil {
			return nil, err
		}
		statements = append(statements, fmt.Sprintf("not (%v)", statement))
	}

	if mc.Exclude != nil {
		if err := mc.Exclude.ValidateForLogs(); err != nil {
			return nil, err
		}
		statement, err := createStatement(*mc.Exclude)
		if err != nil {
			return nil, err
		}
		statements = append(statements, fmt.Sprintf("%v", statement))
	}

	return NewBoolExprForLog(statements, StandardLogFuncs(), ottl.PropagateError, component.TelemetrySettings{Logger: zap.NewNop()})
}

func NewResourceSkipExprBridge(mc *filterconfig.MatchConfig) (expr.BoolExpr[ottlresource.TransformContext], error) {
	statements := make([]string, 0, 2)
	if mc.Include != nil {
		// OTTL treats resource attributes as attributes for the resource context.
		mc.Include.Attributes = mc.Include.Resources
		mc.Include.Resources = nil

		statement, err := createStatement(*mc.Include)
		if err != nil {
			return nil, err
		}
		statements = append(statements, fmt.Sprintf("not (%v)", statement))
	}

	if mc.Exclude != nil {
		// OTTL treats resource attributes as attributes for the resource context.
		mc.Exclude.Attributes = mc.Exclude.Resources
		mc.Exclude.Resources = nil

		statement, err := createStatement(*mc.Exclude)
		if err != nil {
			return nil, err
		}
		statements = append(statements, fmt.Sprintf("%v", statement))
	}

	return NewBoolExprForResource(statements, StandardResourceFuncs(), ottl.PropagateError, component.TelemetrySettings{Logger: zap.NewNop()})
}

func NewSpanSkipExprBridge(mc *filterconfig.MatchConfig) (expr.BoolExpr[ottlspan.TransformContext], error) {
	statements := make([]string, 0, 2)
	if mc.Include != nil {
		statement, err := createStatement(*mc.Include)
		if err != nil {
			return nil, err
		}
		statements = append(statements, fmt.Sprintf("not (%v)", statement))
	}

	if mc.Exclude != nil {
		statement, err := createStatement(*mc.Exclude)
		if err != nil {
			return nil, err
		}
		statements = append(statements, fmt.Sprintf("%v", statement))
	}

	return NewBoolExprForSpan(statements, StandardSpanFuncs(), ottl.PropagateError, component.TelemetrySettings{Logger: zap.NewNop()})
}

func createStatement(mp filterconfig.MatchProperties) (string, error) {
	c, err := createConditions(mp)
	if err != nil {
		return "", err
	}
	var conditions []string
	var format string
	if c.serviceNameConditions != nil {
		if len(c.serviceNameConditions) > 1 {
			format = "(%v)"
		} else {
			format = "%v"
		}
		conditions = append(conditions, fmt.Sprintf(format, strings.Join(c.serviceNameConditions, " or ")))
	}
	if c.spanNameConditions != nil {
		if len(c.spanNameConditions) > 1 {
			format = "(%v)"
		} else {
			format = "%v"
		}
		conditions = append(conditions, fmt.Sprintf(format, strings.Join(c.spanNameConditions, " or ")))
	}
	if c.spanKindConditions != nil {
		if len(c.spanKindConditions) > 1 {
			format = "(%v)"
		} else {
			format = "%v"
		}
		conditions = append(conditions, fmt.Sprintf(format, strings.Join(c.spanKindConditions, " or ")))
	}
	if c.scopeNameConditions != nil {
		if len(c.scopeNameConditions) > 1 {
			format = "(%v)"
		} else {
			format = "%v"
		}
		conditions = append(conditions, fmt.Sprintf(format, strings.Join(c.scopeNameConditions, " or ")))
	}
	if c.scopeVersionConditions != nil {
		if len(c.scopeVersionConditions) > 1 {
			format = "(%v)"
		} else {
			format = "%v"
		}
		conditions = append(conditions, fmt.Sprintf(format, strings.Join(c.scopeVersionConditions, " or ")))
	}
	if c.attributeConditions != nil {
		conditions = append(conditions, fmt.Sprintf("%v", strings.Join(c.attributeConditions, " and ")))
	}
	if c.resourceAttributeConditions != nil {
		conditions = append(conditions, fmt.Sprintf("%v", strings.Join(c.resourceAttributeConditions, " and ")))
	}
	if c.bodyConditions != nil {
		if len(c.bodyConditions) > 1 {
			format = "(%v)"
		} else {
			format = "%v"
		}
		conditions = append(conditions, fmt.Sprintf(format, strings.Join(c.bodyConditions, " or ")))
	}
	if c.severityTextConditions != nil {
		if len(c.severityTextConditions) > 1 {
			format = "(%v)"
		} else {
			format = "%v"
		}
		conditions = append(conditions, fmt.Sprintf(format, strings.Join(c.severityTextConditions, " or ")))
	}
	if c.severityNumberCondition != nil {
		conditions = append(conditions, *c.severityNumberCondition)
	}
	return strings.Join(conditions, " and "), nil
}

type conditionStatements struct {
	serviceNameConditions       []string
	spanNameConditions          []string
	spanKindConditions          []string
	scopeNameConditions         []string
	scopeVersionConditions      []string
	attributeConditions         []string
	resourceAttributeConditions []string
	bodyConditions              []string
	severityTextConditions      []string
	severityNumberCondition     *string
}

func createConditions(mp filterconfig.MatchProperties) (conditionStatements, error) {
	templates, err := createStatementTemplates(mp.MatchType)
	if err != nil {
		return conditionStatements{}, err
	}

	serviceNameConditions := createBasicConditions(templates.serviceNameStatement, mp.Services)
	spanNameConditions := createBasicConditions(templates.spanNameStatement, mp.SpanNames)
	spanKindConditions := createBasicConditions(templates.spanKindStatement, mp.SpanKinds)
	scopeNameConditions, scopeVersionConditions := createLibraryConditions(templates.scopeNameStatement, templates.scopeVersionStatement, mp.Libraries)
	attributeConditions := createAttributeConditions(templates.attrStatement, mp.Attributes, mp.MatchType)
	resourceAttributeConditions := createAttributeConditions(templates.resourceAttrStatement, mp.Resources, mp.MatchType)
	bodyConditions := createBasicConditions(templates.bodyStatement, mp.LogBodies)
	severityTextConditions := createBasicConditions(templates.severityTextStatement, mp.LogSeverityTexts)
	severityNumberCondition := createSeverityNumberConditions(mp.LogSeverityNumber)

	return conditionStatements{
		serviceNameConditions:       serviceNameConditions,
		spanNameConditions:          spanNameConditions,
		spanKindConditions:          spanKindConditions,
		scopeNameConditions:         scopeNameConditions,
		scopeVersionConditions:      scopeVersionConditions,
		attributeConditions:         attributeConditions,
		resourceAttributeConditions: resourceAttributeConditions,
		bodyConditions:              bodyConditions,
		severityTextConditions:      severityTextConditions,
		severityNumberCondition:     severityNumberCondition,
	}, nil
}

type statementTemplates struct {
	serviceNameStatement  string
	spanNameStatement     string
	spanKindStatement     string
	scopeNameStatement    string
	scopeVersionStatement string
	attrStatement         string
	resourceAttrStatement string
	bodyStatement         string
	severityTextStatement string
}

func createStatementTemplates(matchType filterset.MatchType) (statementTemplates, error) {
	switch matchType {
	case filterset.Strict:
		return statementTemplates{
			serviceNameStatement:  serviceNameStaticStatement,
			spanNameStatement:     nameStaticStatement,
			spanKindStatement:     spanKindStaticStatement,
			scopeNameStatement:    scopeNameStaticStatement,
			scopeVersionStatement: scopeVersionStaticStatement,
			attrStatement:         attributesStaticStatement,
			resourceAttrStatement: resourceAttributesStaticStatement,
			bodyStatement:         bodyStaticStatement,
			severityTextStatement: severityTextStaticStatement,
		}, nil
	case filterset.Regexp:
		return statementTemplates{
			serviceNameStatement:  serviceNameRegexStatement,
			spanNameStatement:     nameRegexStatement,
			spanKindStatement:     spanKindRegexStatement,
			scopeNameStatement:    scopeNameRegexStatement,
			scopeVersionStatement: scopeVersionRegexStatement,
			attrStatement:         attributesRegexStatement,
			resourceAttrStatement: resourceAttributesRegexStatement,
			bodyStatement:         bodyRegexStatement,
			severityTextStatement: severityTextRegexStatement,
		}, nil
	default:
		return statementTemplates{}, filterset.NewUnrecognizedMatchTypeError(matchType)
	}
}

func createBasicConditions(template string, input []string) []string {
	var conditions []string
	for _, i := range input {
		conditions = append(conditions, fmt.Sprintf(template, i))
	}
	return conditions
}

func createLibraryConditions(nameTemplate string, versionTemplate string, libraries []filterconfig.InstrumentationLibrary) ([]string, []string) {
	var scopeNameConditions []string
	var scopeVersionConditions []string
	for _, scope := range libraries {
		scopeNameConditions = append(scopeNameConditions, fmt.Sprintf(nameTemplate, scope.Name))
		if scope.Version != nil {
			scopeVersionConditions = append(scopeVersionConditions, fmt.Sprintf(versionTemplate, *scope.Version))
		}
	}
	return scopeNameConditions, scopeVersionConditions
}

func createAttributeConditions(template string, input []filterconfig.Attribute, matchType filterset.MatchType) []string {
	var attributeConditions []string
	for _, attribute := range input {
		var value any
		if matchType == filterset.Strict {
			value = convertAttribute(attribute.Value)
		} else {
			value = attribute.Value
		}
		attributeConditions = append(attributeConditions, fmt.Sprintf(template, attribute.Key, value))
	}
	return attributeConditions
}

func convertAttribute(value any) string {
	switch val := value.(type) {
	case string:
		return fmt.Sprintf(`"%v"`, val)
	default:
		return fmt.Sprintf(`%v`, val)
	}
}

func createSeverityNumberConditions(severityNumberProperties *filterconfig.LogSeverityNumberMatchProperties) *string {
	if severityNumberProperties == nil {
		return nil
	}
	severityNumberCondition := fmt.Sprintf(severityNumberStatement, severityNumberProperties.MatchUndefined, severityNumberProperties.Min)
	return &severityNumberCondition
}

func NewMetricSkipExprBridge(include *filterconfig.MetricMatchProperties, exclude *filterconfig.MetricMatchProperties) (expr.BoolExpr[ottlmetric.TransformContext], error) {
	statements := make([]string, 0, 2)
	if include != nil {
		statement, err := createMetricStatement(*include)
		if err != nil {
			return nil, err
		}
		if statement != nil {
			statements = append(statements, fmt.Sprintf("not (%v)", *statement))
		}
	}

	if exclude != nil {
		statement, err := createMetricStatement(*exclude)
		if err != nil {
			return nil, err
		}
		if statement != nil {
			statements = append(statements, fmt.Sprintf("%v", *statement))
		}
	}

	if len(statements) == 0 {
		return nil, nil
	}

	return NewBoolExprForMetric(statements, StandardMetricFuncs(), ottl.PropagateError, component.TelemetrySettings{Logger: zap.NewNop()})
}

func createMetricStatement(mp filterconfig.MetricMatchProperties) (*string, error) {
	if mp.MatchType == filterconfig.MetricExpr {
		if len(mp.Expressions) == 0 {
			return nil, nil
		}
		return nil, errors.New("expressions configuration cannot be converted to OTTL - see https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/filterprocessor#configuration for OTTL configuration")
	}

	if len(mp.MetricNames) == 0 {
		return nil, nil
	}

	metricNameStatement := nameStaticStatement
	if mp.MatchType == filterconfig.MetricRegexp {
		metricNameStatement = nameRegexStatement
	}
	metricNameConditions := createBasicConditions(metricNameStatement, mp.MetricNames)
	var format string
	if len(metricNameConditions) > 1 {
		format = "(%v)"
	} else {
		format = "%v"
	}
	statement := fmt.Sprintf(format, strings.Join(metricNameConditions, " or "))
	return &statement, nil
}
