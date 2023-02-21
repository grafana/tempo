// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// ReferenceTableLogsLookupProcessor **Note**: Reference Tables are in public beta.
// Use the Lookup Processor to define a mapping between a log attribute
// and a human readable value saved in a Reference Table.
// For example, you can use the Lookup Processor to map an internal service ID
// into a human readable service name. Alternatively, you could also use it to check
// if the MAC address that just attempted to connect to the production
// environment belongs to your list of stolen machines.
type ReferenceTableLogsLookupProcessor struct {
	// Whether or not the processor is enabled.
	IsEnabled *bool `json:"is_enabled,omitempty"`
	// Name of the Reference Table for the source attribute and their associated target attribute values.
	LookupEnrichmentTable string `json:"lookup_enrichment_table"`
	// Name of the processor.
	Name *string `json:"name,omitempty"`
	// Source attribute used to perform the lookup.
	Source string `json:"source"`
	// Name of the attribute that contains the corresponding value in the mapping list.
	Target string `json:"target"`
	// Type of logs lookup processor.
	Type LogsLookupProcessorType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewReferenceTableLogsLookupProcessor instantiates a new ReferenceTableLogsLookupProcessor object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewReferenceTableLogsLookupProcessor(lookupEnrichmentTable string, source string, target string, typeVar LogsLookupProcessorType) *ReferenceTableLogsLookupProcessor {
	this := ReferenceTableLogsLookupProcessor{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	this.LookupEnrichmentTable = lookupEnrichmentTable
	this.Source = source
	this.Target = target
	this.Type = typeVar
	return &this
}

// NewReferenceTableLogsLookupProcessorWithDefaults instantiates a new ReferenceTableLogsLookupProcessor object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewReferenceTableLogsLookupProcessorWithDefaults() *ReferenceTableLogsLookupProcessor {
	this := ReferenceTableLogsLookupProcessor{}
	var isEnabled bool = false
	this.IsEnabled = &isEnabled
	var typeVar LogsLookupProcessorType = LOGSLOOKUPPROCESSORTYPE_LOOKUP_PROCESSOR
	this.Type = typeVar
	return &this
}

// GetIsEnabled returns the IsEnabled field value if set, zero value otherwise.
func (o *ReferenceTableLogsLookupProcessor) GetIsEnabled() bool {
	if o == nil || o.IsEnabled == nil {
		var ret bool
		return ret
	}
	return *o.IsEnabled
}

// GetIsEnabledOk returns a tuple with the IsEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ReferenceTableLogsLookupProcessor) GetIsEnabledOk() (*bool, bool) {
	if o == nil || o.IsEnabled == nil {
		return nil, false
	}
	return o.IsEnabled, true
}

// HasIsEnabled returns a boolean if a field has been set.
func (o *ReferenceTableLogsLookupProcessor) HasIsEnabled() bool {
	return o != nil && o.IsEnabled != nil
}

// SetIsEnabled gets a reference to the given bool and assigns it to the IsEnabled field.
func (o *ReferenceTableLogsLookupProcessor) SetIsEnabled(v bool) {
	o.IsEnabled = &v
}

// GetLookupEnrichmentTable returns the LookupEnrichmentTable field value.
func (o *ReferenceTableLogsLookupProcessor) GetLookupEnrichmentTable() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.LookupEnrichmentTable
}

// GetLookupEnrichmentTableOk returns a tuple with the LookupEnrichmentTable field value
// and a boolean to check if the value has been set.
func (o *ReferenceTableLogsLookupProcessor) GetLookupEnrichmentTableOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.LookupEnrichmentTable, true
}

// SetLookupEnrichmentTable sets field value.
func (o *ReferenceTableLogsLookupProcessor) SetLookupEnrichmentTable(v string) {
	o.LookupEnrichmentTable = v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *ReferenceTableLogsLookupProcessor) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ReferenceTableLogsLookupProcessor) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *ReferenceTableLogsLookupProcessor) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *ReferenceTableLogsLookupProcessor) SetName(v string) {
	o.Name = &v
}

// GetSource returns the Source field value.
func (o *ReferenceTableLogsLookupProcessor) GetSource() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Source
}

// GetSourceOk returns a tuple with the Source field value
// and a boolean to check if the value has been set.
func (o *ReferenceTableLogsLookupProcessor) GetSourceOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Source, true
}

// SetSource sets field value.
func (o *ReferenceTableLogsLookupProcessor) SetSource(v string) {
	o.Source = v
}

// GetTarget returns the Target field value.
func (o *ReferenceTableLogsLookupProcessor) GetTarget() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Target
}

// GetTargetOk returns a tuple with the Target field value
// and a boolean to check if the value has been set.
func (o *ReferenceTableLogsLookupProcessor) GetTargetOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Target, true
}

// SetTarget sets field value.
func (o *ReferenceTableLogsLookupProcessor) SetTarget(v string) {
	o.Target = v
}

// GetType returns the Type field value.
func (o *ReferenceTableLogsLookupProcessor) GetType() LogsLookupProcessorType {
	if o == nil {
		var ret LogsLookupProcessorType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *ReferenceTableLogsLookupProcessor) GetTypeOk() (*LogsLookupProcessorType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *ReferenceTableLogsLookupProcessor) SetType(v LogsLookupProcessorType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ReferenceTableLogsLookupProcessor) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.IsEnabled != nil {
		toSerialize["is_enabled"] = o.IsEnabled
	}
	toSerialize["lookup_enrichment_table"] = o.LookupEnrichmentTable
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	toSerialize["source"] = o.Source
	toSerialize["target"] = o.Target
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ReferenceTableLogsLookupProcessor) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		LookupEnrichmentTable *string                  `json:"lookup_enrichment_table"`
		Source                *string                  `json:"source"`
		Target                *string                  `json:"target"`
		Type                  *LogsLookupProcessorType `json:"type"`
	}{}
	all := struct {
		IsEnabled             *bool                   `json:"is_enabled,omitempty"`
		LookupEnrichmentTable string                  `json:"lookup_enrichment_table"`
		Name                  *string                 `json:"name,omitempty"`
		Source                string                  `json:"source"`
		Target                string                  `json:"target"`
		Type                  LogsLookupProcessorType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.LookupEnrichmentTable == nil {
		return fmt.Errorf("required field lookup_enrichment_table missing")
	}
	if required.Source == nil {
		return fmt.Errorf("required field source missing")
	}
	if required.Target == nil {
		return fmt.Errorf("required field target missing")
	}
	if required.Type == nil {
		return fmt.Errorf("required field type missing")
	}
	err = json.Unmarshal(bytes, &all)
	if err != nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.IsEnabled = all.IsEnabled
	o.LookupEnrichmentTable = all.LookupEnrichmentTable
	o.Name = all.Name
	o.Source = all.Source
	o.Target = all.Target
	o.Type = all.Type
	return nil
}
