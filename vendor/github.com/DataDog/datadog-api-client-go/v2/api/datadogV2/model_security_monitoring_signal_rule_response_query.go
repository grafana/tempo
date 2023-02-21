// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SecurityMonitoringSignalRuleResponseQuery Query for matching rule on signals.
type SecurityMonitoringSignalRuleResponseQuery struct {
	// The aggregation type.
	Aggregation *SecurityMonitoringRuleQueryAggregation `json:"aggregation,omitempty"`
	// Fields to group by.
	CorrelatedByFields []string `json:"correlatedByFields,omitempty"`
	// Index of the rule query used to retrieve the correlated field.
	CorrelatedQueryIndex *int32 `json:"correlatedQueryIndex,omitempty"`
	// Default Rule ID to match on signals.
	DefaultRuleId *string `json:"defaultRuleId,omitempty"`
	// Group of target fields to aggregate over.
	Metrics []string `json:"metrics,omitempty"`
	// Name of the query.
	Name *string `json:"name,omitempty"`
	// Rule ID to match on signals.
	RuleId *string `json:"ruleId,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalRuleResponseQuery instantiates a new SecurityMonitoringSignalRuleResponseQuery object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalRuleResponseQuery() *SecurityMonitoringSignalRuleResponseQuery {
	this := SecurityMonitoringSignalRuleResponseQuery{}
	return &this
}

// NewSecurityMonitoringSignalRuleResponseQueryWithDefaults instantiates a new SecurityMonitoringSignalRuleResponseQuery object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalRuleResponseQueryWithDefaults() *SecurityMonitoringSignalRuleResponseQuery {
	this := SecurityMonitoringSignalRuleResponseQuery{}
	return &this
}

// GetAggregation returns the Aggregation field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetAggregation() SecurityMonitoringRuleQueryAggregation {
	if o == nil || o.Aggregation == nil {
		var ret SecurityMonitoringRuleQueryAggregation
		return ret
	}
	return *o.Aggregation
}

// GetAggregationOk returns a tuple with the Aggregation field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetAggregationOk() (*SecurityMonitoringRuleQueryAggregation, bool) {
	if o == nil || o.Aggregation == nil {
		return nil, false
	}
	return o.Aggregation, true
}

// HasAggregation returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) HasAggregation() bool {
	return o != nil && o.Aggregation != nil
}

// SetAggregation gets a reference to the given SecurityMonitoringRuleQueryAggregation and assigns it to the Aggregation field.
func (o *SecurityMonitoringSignalRuleResponseQuery) SetAggregation(v SecurityMonitoringRuleQueryAggregation) {
	o.Aggregation = &v
}

// GetCorrelatedByFields returns the CorrelatedByFields field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetCorrelatedByFields() []string {
	if o == nil || o.CorrelatedByFields == nil {
		var ret []string
		return ret
	}
	return o.CorrelatedByFields
}

// GetCorrelatedByFieldsOk returns a tuple with the CorrelatedByFields field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetCorrelatedByFieldsOk() (*[]string, bool) {
	if o == nil || o.CorrelatedByFields == nil {
		return nil, false
	}
	return &o.CorrelatedByFields, true
}

// HasCorrelatedByFields returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) HasCorrelatedByFields() bool {
	return o != nil && o.CorrelatedByFields != nil
}

// SetCorrelatedByFields gets a reference to the given []string and assigns it to the CorrelatedByFields field.
func (o *SecurityMonitoringSignalRuleResponseQuery) SetCorrelatedByFields(v []string) {
	o.CorrelatedByFields = v
}

// GetCorrelatedQueryIndex returns the CorrelatedQueryIndex field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetCorrelatedQueryIndex() int32 {
	if o == nil || o.CorrelatedQueryIndex == nil {
		var ret int32
		return ret
	}
	return *o.CorrelatedQueryIndex
}

// GetCorrelatedQueryIndexOk returns a tuple with the CorrelatedQueryIndex field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetCorrelatedQueryIndexOk() (*int32, bool) {
	if o == nil || o.CorrelatedQueryIndex == nil {
		return nil, false
	}
	return o.CorrelatedQueryIndex, true
}

// HasCorrelatedQueryIndex returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) HasCorrelatedQueryIndex() bool {
	return o != nil && o.CorrelatedQueryIndex != nil
}

// SetCorrelatedQueryIndex gets a reference to the given int32 and assigns it to the CorrelatedQueryIndex field.
func (o *SecurityMonitoringSignalRuleResponseQuery) SetCorrelatedQueryIndex(v int32) {
	o.CorrelatedQueryIndex = &v
}

// GetDefaultRuleId returns the DefaultRuleId field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetDefaultRuleId() string {
	if o == nil || o.DefaultRuleId == nil {
		var ret string
		return ret
	}
	return *o.DefaultRuleId
}

// GetDefaultRuleIdOk returns a tuple with the DefaultRuleId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetDefaultRuleIdOk() (*string, bool) {
	if o == nil || o.DefaultRuleId == nil {
		return nil, false
	}
	return o.DefaultRuleId, true
}

// HasDefaultRuleId returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) HasDefaultRuleId() bool {
	return o != nil && o.DefaultRuleId != nil
}

// SetDefaultRuleId gets a reference to the given string and assigns it to the DefaultRuleId field.
func (o *SecurityMonitoringSignalRuleResponseQuery) SetDefaultRuleId(v string) {
	o.DefaultRuleId = &v
}

// GetMetrics returns the Metrics field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetMetrics() []string {
	if o == nil || o.Metrics == nil {
		var ret []string
		return ret
	}
	return o.Metrics
}

// GetMetricsOk returns a tuple with the Metrics field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetMetricsOk() (*[]string, bool) {
	if o == nil || o.Metrics == nil {
		return nil, false
	}
	return &o.Metrics, true
}

// HasMetrics returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) HasMetrics() bool {
	return o != nil && o.Metrics != nil
}

// SetMetrics gets a reference to the given []string and assigns it to the Metrics field.
func (o *SecurityMonitoringSignalRuleResponseQuery) SetMetrics(v []string) {
	o.Metrics = v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SecurityMonitoringSignalRuleResponseQuery) SetName(v string) {
	o.Name = &v
}

// GetRuleId returns the RuleId field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetRuleId() string {
	if o == nil || o.RuleId == nil {
		var ret string
		return ret
	}
	return *o.RuleId
}

// GetRuleIdOk returns a tuple with the RuleId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) GetRuleIdOk() (*string, bool) {
	if o == nil || o.RuleId == nil {
		return nil, false
	}
	return o.RuleId, true
}

// HasRuleId returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalRuleResponseQuery) HasRuleId() bool {
	return o != nil && o.RuleId != nil
}

// SetRuleId gets a reference to the given string and assigns it to the RuleId field.
func (o *SecurityMonitoringSignalRuleResponseQuery) SetRuleId(v string) {
	o.RuleId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalRuleResponseQuery) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Aggregation != nil {
		toSerialize["aggregation"] = o.Aggregation
	}
	if o.CorrelatedByFields != nil {
		toSerialize["correlatedByFields"] = o.CorrelatedByFields
	}
	if o.CorrelatedQueryIndex != nil {
		toSerialize["correlatedQueryIndex"] = o.CorrelatedQueryIndex
	}
	if o.DefaultRuleId != nil {
		toSerialize["defaultRuleId"] = o.DefaultRuleId
	}
	if o.Metrics != nil {
		toSerialize["metrics"] = o.Metrics
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.RuleId != nil {
		toSerialize["ruleId"] = o.RuleId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringSignalRuleResponseQuery) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Aggregation          *SecurityMonitoringRuleQueryAggregation `json:"aggregation,omitempty"`
		CorrelatedByFields   []string                                `json:"correlatedByFields,omitempty"`
		CorrelatedQueryIndex *int32                                  `json:"correlatedQueryIndex,omitempty"`
		DefaultRuleId        *string                                 `json:"defaultRuleId,omitempty"`
		Metrics              []string                                `json:"metrics,omitempty"`
		Name                 *string                                 `json:"name,omitempty"`
		RuleId               *string                                 `json:"ruleId,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Aggregation; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Aggregation = all.Aggregation
	o.CorrelatedByFields = all.CorrelatedByFields
	o.CorrelatedQueryIndex = all.CorrelatedQueryIndex
	o.DefaultRuleId = all.DefaultRuleId
	o.Metrics = all.Metrics
	o.Name = all.Name
	o.RuleId = all.RuleId
	return nil
}
