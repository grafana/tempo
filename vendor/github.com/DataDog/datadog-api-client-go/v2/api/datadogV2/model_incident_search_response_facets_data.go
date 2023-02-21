// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentSearchResponseFacetsData Facet data for incidents returned by a search query.
type IncidentSearchResponseFacetsData struct {
	// Facet data for incident commander users.
	Commander []IncidentSearchResponseUserFacetData `json:"commander,omitempty"`
	// Facet data for incident creator users.
	CreatedBy []IncidentSearchResponseUserFacetData `json:"created_by,omitempty"`
	// Facet data for incident property fields.
	Fields []IncidentSearchResponsePropertyFieldFacetData `json:"fields,omitempty"`
	// Facet data for incident impact attributes.
	Impact []IncidentSearchResponseFieldFacetData `json:"impact,omitempty"`
	// Facet data for incident last modified by users.
	LastModifiedBy []IncidentSearchResponseUserFacetData `json:"last_modified_by,omitempty"`
	// Facet data for incident postmortem existence.
	Postmortem []IncidentSearchResponseFieldFacetData `json:"postmortem,omitempty"`
	// Facet data for incident responder users.
	Responder []IncidentSearchResponseUserFacetData `json:"responder,omitempty"`
	// Facet data for incident severity attributes.
	Severity []IncidentSearchResponseFieldFacetData `json:"severity,omitempty"`
	// Facet data for incident state attributes.
	State []IncidentSearchResponseFieldFacetData `json:"state,omitempty"`
	// Facet data for incident time to repair metrics.
	TimeToRepair []IncidentSearchResponseNumericFacetData `json:"time_to_repair,omitempty"`
	// Facet data for incident time to resolve metrics.
	TimeToResolve []IncidentSearchResponseNumericFacetData `json:"time_to_resolve,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentSearchResponseFacetsData instantiates a new IncidentSearchResponseFacetsData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentSearchResponseFacetsData() *IncidentSearchResponseFacetsData {
	this := IncidentSearchResponseFacetsData{}
	return &this
}

// NewIncidentSearchResponseFacetsDataWithDefaults instantiates a new IncidentSearchResponseFacetsData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentSearchResponseFacetsDataWithDefaults() *IncidentSearchResponseFacetsData {
	this := IncidentSearchResponseFacetsData{}
	return &this
}

// GetCommander returns the Commander field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetCommander() []IncidentSearchResponseUserFacetData {
	if o == nil || o.Commander == nil {
		var ret []IncidentSearchResponseUserFacetData
		return ret
	}
	return o.Commander
}

// GetCommanderOk returns a tuple with the Commander field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetCommanderOk() (*[]IncidentSearchResponseUserFacetData, bool) {
	if o == nil || o.Commander == nil {
		return nil, false
	}
	return &o.Commander, true
}

// HasCommander returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasCommander() bool {
	return o != nil && o.Commander != nil
}

// SetCommander gets a reference to the given []IncidentSearchResponseUserFacetData and assigns it to the Commander field.
func (o *IncidentSearchResponseFacetsData) SetCommander(v []IncidentSearchResponseUserFacetData) {
	o.Commander = v
}

// GetCreatedBy returns the CreatedBy field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetCreatedBy() []IncidentSearchResponseUserFacetData {
	if o == nil || o.CreatedBy == nil {
		var ret []IncidentSearchResponseUserFacetData
		return ret
	}
	return o.CreatedBy
}

// GetCreatedByOk returns a tuple with the CreatedBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetCreatedByOk() (*[]IncidentSearchResponseUserFacetData, bool) {
	if o == nil || o.CreatedBy == nil {
		return nil, false
	}
	return &o.CreatedBy, true
}

// HasCreatedBy returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasCreatedBy() bool {
	return o != nil && o.CreatedBy != nil
}

// SetCreatedBy gets a reference to the given []IncidentSearchResponseUserFacetData and assigns it to the CreatedBy field.
func (o *IncidentSearchResponseFacetsData) SetCreatedBy(v []IncidentSearchResponseUserFacetData) {
	o.CreatedBy = v
}

// GetFields returns the Fields field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetFields() []IncidentSearchResponsePropertyFieldFacetData {
	if o == nil || o.Fields == nil {
		var ret []IncidentSearchResponsePropertyFieldFacetData
		return ret
	}
	return o.Fields
}

// GetFieldsOk returns a tuple with the Fields field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetFieldsOk() (*[]IncidentSearchResponsePropertyFieldFacetData, bool) {
	if o == nil || o.Fields == nil {
		return nil, false
	}
	return &o.Fields, true
}

// HasFields returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasFields() bool {
	return o != nil && o.Fields != nil
}

// SetFields gets a reference to the given []IncidentSearchResponsePropertyFieldFacetData and assigns it to the Fields field.
func (o *IncidentSearchResponseFacetsData) SetFields(v []IncidentSearchResponsePropertyFieldFacetData) {
	o.Fields = v
}

// GetImpact returns the Impact field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetImpact() []IncidentSearchResponseFieldFacetData {
	if o == nil || o.Impact == nil {
		var ret []IncidentSearchResponseFieldFacetData
		return ret
	}
	return o.Impact
}

// GetImpactOk returns a tuple with the Impact field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetImpactOk() (*[]IncidentSearchResponseFieldFacetData, bool) {
	if o == nil || o.Impact == nil {
		return nil, false
	}
	return &o.Impact, true
}

// HasImpact returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasImpact() bool {
	return o != nil && o.Impact != nil
}

// SetImpact gets a reference to the given []IncidentSearchResponseFieldFacetData and assigns it to the Impact field.
func (o *IncidentSearchResponseFacetsData) SetImpact(v []IncidentSearchResponseFieldFacetData) {
	o.Impact = v
}

// GetLastModifiedBy returns the LastModifiedBy field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetLastModifiedBy() []IncidentSearchResponseUserFacetData {
	if o == nil || o.LastModifiedBy == nil {
		var ret []IncidentSearchResponseUserFacetData
		return ret
	}
	return o.LastModifiedBy
}

// GetLastModifiedByOk returns a tuple with the LastModifiedBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetLastModifiedByOk() (*[]IncidentSearchResponseUserFacetData, bool) {
	if o == nil || o.LastModifiedBy == nil {
		return nil, false
	}
	return &o.LastModifiedBy, true
}

// HasLastModifiedBy returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasLastModifiedBy() bool {
	return o != nil && o.LastModifiedBy != nil
}

// SetLastModifiedBy gets a reference to the given []IncidentSearchResponseUserFacetData and assigns it to the LastModifiedBy field.
func (o *IncidentSearchResponseFacetsData) SetLastModifiedBy(v []IncidentSearchResponseUserFacetData) {
	o.LastModifiedBy = v
}

// GetPostmortem returns the Postmortem field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetPostmortem() []IncidentSearchResponseFieldFacetData {
	if o == nil || o.Postmortem == nil {
		var ret []IncidentSearchResponseFieldFacetData
		return ret
	}
	return o.Postmortem
}

// GetPostmortemOk returns a tuple with the Postmortem field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetPostmortemOk() (*[]IncidentSearchResponseFieldFacetData, bool) {
	if o == nil || o.Postmortem == nil {
		return nil, false
	}
	return &o.Postmortem, true
}

// HasPostmortem returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasPostmortem() bool {
	return o != nil && o.Postmortem != nil
}

// SetPostmortem gets a reference to the given []IncidentSearchResponseFieldFacetData and assigns it to the Postmortem field.
func (o *IncidentSearchResponseFacetsData) SetPostmortem(v []IncidentSearchResponseFieldFacetData) {
	o.Postmortem = v
}

// GetResponder returns the Responder field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetResponder() []IncidentSearchResponseUserFacetData {
	if o == nil || o.Responder == nil {
		var ret []IncidentSearchResponseUserFacetData
		return ret
	}
	return o.Responder
}

// GetResponderOk returns a tuple with the Responder field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetResponderOk() (*[]IncidentSearchResponseUserFacetData, bool) {
	if o == nil || o.Responder == nil {
		return nil, false
	}
	return &o.Responder, true
}

// HasResponder returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasResponder() bool {
	return o != nil && o.Responder != nil
}

// SetResponder gets a reference to the given []IncidentSearchResponseUserFacetData and assigns it to the Responder field.
func (o *IncidentSearchResponseFacetsData) SetResponder(v []IncidentSearchResponseUserFacetData) {
	o.Responder = v
}

// GetSeverity returns the Severity field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetSeverity() []IncidentSearchResponseFieldFacetData {
	if o == nil || o.Severity == nil {
		var ret []IncidentSearchResponseFieldFacetData
		return ret
	}
	return o.Severity
}

// GetSeverityOk returns a tuple with the Severity field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetSeverityOk() (*[]IncidentSearchResponseFieldFacetData, bool) {
	if o == nil || o.Severity == nil {
		return nil, false
	}
	return &o.Severity, true
}

// HasSeverity returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasSeverity() bool {
	return o != nil && o.Severity != nil
}

// SetSeverity gets a reference to the given []IncidentSearchResponseFieldFacetData and assigns it to the Severity field.
func (o *IncidentSearchResponseFacetsData) SetSeverity(v []IncidentSearchResponseFieldFacetData) {
	o.Severity = v
}

// GetState returns the State field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetState() []IncidentSearchResponseFieldFacetData {
	if o == nil || o.State == nil {
		var ret []IncidentSearchResponseFieldFacetData
		return ret
	}
	return o.State
}

// GetStateOk returns a tuple with the State field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetStateOk() (*[]IncidentSearchResponseFieldFacetData, bool) {
	if o == nil || o.State == nil {
		return nil, false
	}
	return &o.State, true
}

// HasState returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasState() bool {
	return o != nil && o.State != nil
}

// SetState gets a reference to the given []IncidentSearchResponseFieldFacetData and assigns it to the State field.
func (o *IncidentSearchResponseFacetsData) SetState(v []IncidentSearchResponseFieldFacetData) {
	o.State = v
}

// GetTimeToRepair returns the TimeToRepair field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetTimeToRepair() []IncidentSearchResponseNumericFacetData {
	if o == nil || o.TimeToRepair == nil {
		var ret []IncidentSearchResponseNumericFacetData
		return ret
	}
	return o.TimeToRepair
}

// GetTimeToRepairOk returns a tuple with the TimeToRepair field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetTimeToRepairOk() (*[]IncidentSearchResponseNumericFacetData, bool) {
	if o == nil || o.TimeToRepair == nil {
		return nil, false
	}
	return &o.TimeToRepair, true
}

// HasTimeToRepair returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasTimeToRepair() bool {
	return o != nil && o.TimeToRepair != nil
}

// SetTimeToRepair gets a reference to the given []IncidentSearchResponseNumericFacetData and assigns it to the TimeToRepair field.
func (o *IncidentSearchResponseFacetsData) SetTimeToRepair(v []IncidentSearchResponseNumericFacetData) {
	o.TimeToRepair = v
}

// GetTimeToResolve returns the TimeToResolve field value if set, zero value otherwise.
func (o *IncidentSearchResponseFacetsData) GetTimeToResolve() []IncidentSearchResponseNumericFacetData {
	if o == nil || o.TimeToResolve == nil {
		var ret []IncidentSearchResponseNumericFacetData
		return ret
	}
	return o.TimeToResolve
}

// GetTimeToResolveOk returns a tuple with the TimeToResolve field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentSearchResponseFacetsData) GetTimeToResolveOk() (*[]IncidentSearchResponseNumericFacetData, bool) {
	if o == nil || o.TimeToResolve == nil {
		return nil, false
	}
	return &o.TimeToResolve, true
}

// HasTimeToResolve returns a boolean if a field has been set.
func (o *IncidentSearchResponseFacetsData) HasTimeToResolve() bool {
	return o != nil && o.TimeToResolve != nil
}

// SetTimeToResolve gets a reference to the given []IncidentSearchResponseNumericFacetData and assigns it to the TimeToResolve field.
func (o *IncidentSearchResponseFacetsData) SetTimeToResolve(v []IncidentSearchResponseNumericFacetData) {
	o.TimeToResolve = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentSearchResponseFacetsData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Commander != nil {
		toSerialize["commander"] = o.Commander
	}
	if o.CreatedBy != nil {
		toSerialize["created_by"] = o.CreatedBy
	}
	if o.Fields != nil {
		toSerialize["fields"] = o.Fields
	}
	if o.Impact != nil {
		toSerialize["impact"] = o.Impact
	}
	if o.LastModifiedBy != nil {
		toSerialize["last_modified_by"] = o.LastModifiedBy
	}
	if o.Postmortem != nil {
		toSerialize["postmortem"] = o.Postmortem
	}
	if o.Responder != nil {
		toSerialize["responder"] = o.Responder
	}
	if o.Severity != nil {
		toSerialize["severity"] = o.Severity
	}
	if o.State != nil {
		toSerialize["state"] = o.State
	}
	if o.TimeToRepair != nil {
		toSerialize["time_to_repair"] = o.TimeToRepair
	}
	if o.TimeToResolve != nil {
		toSerialize["time_to_resolve"] = o.TimeToResolve
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentSearchResponseFacetsData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Commander      []IncidentSearchResponseUserFacetData          `json:"commander,omitempty"`
		CreatedBy      []IncidentSearchResponseUserFacetData          `json:"created_by,omitempty"`
		Fields         []IncidentSearchResponsePropertyFieldFacetData `json:"fields,omitempty"`
		Impact         []IncidentSearchResponseFieldFacetData         `json:"impact,omitempty"`
		LastModifiedBy []IncidentSearchResponseUserFacetData          `json:"last_modified_by,omitempty"`
		Postmortem     []IncidentSearchResponseFieldFacetData         `json:"postmortem,omitempty"`
		Responder      []IncidentSearchResponseUserFacetData          `json:"responder,omitempty"`
		Severity       []IncidentSearchResponseFieldFacetData         `json:"severity,omitempty"`
		State          []IncidentSearchResponseFieldFacetData         `json:"state,omitempty"`
		TimeToRepair   []IncidentSearchResponseNumericFacetData       `json:"time_to_repair,omitempty"`
		TimeToResolve  []IncidentSearchResponseNumericFacetData       `json:"time_to_resolve,omitempty"`
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
	o.Commander = all.Commander
	o.CreatedBy = all.CreatedBy
	o.Fields = all.Fields
	o.Impact = all.Impact
	o.LastModifiedBy = all.LastModifiedBy
	o.Postmortem = all.Postmortem
	o.Responder = all.Responder
	o.Severity = all.Severity
	o.State = all.State
	o.TimeToRepair = all.TimeToRepair
	o.TimeToResolve = all.TimeToResolve
	return nil
}
