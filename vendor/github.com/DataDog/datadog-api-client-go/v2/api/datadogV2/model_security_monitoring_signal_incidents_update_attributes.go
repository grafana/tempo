// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringSignalIncidentsUpdateAttributes Attributes describing the new list of related signals for a security signal.
type SecurityMonitoringSignalIncidentsUpdateAttributes struct {
	// Array of incidents that are associated with this signal.
	IncidentIds []int64 `json:"incident_ids"`
	// Version of the updated signal. If server side version is higher, update will be rejected.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalIncidentsUpdateAttributes instantiates a new SecurityMonitoringSignalIncidentsUpdateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalIncidentsUpdateAttributes(incidentIds []int64) *SecurityMonitoringSignalIncidentsUpdateAttributes {
	this := SecurityMonitoringSignalIncidentsUpdateAttributes{}
	this.IncidentIds = incidentIds
	return &this
}

// NewSecurityMonitoringSignalIncidentsUpdateAttributesWithDefaults instantiates a new SecurityMonitoringSignalIncidentsUpdateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalIncidentsUpdateAttributesWithDefaults() *SecurityMonitoringSignalIncidentsUpdateAttributes {
	this := SecurityMonitoringSignalIncidentsUpdateAttributes{}
	return &this
}

// GetIncidentIds returns the IncidentIds field value.
func (o *SecurityMonitoringSignalIncidentsUpdateAttributes) GetIncidentIds() []int64 {
	if o == nil {
		var ret []int64
		return ret
	}
	return o.IncidentIds
}

// GetIncidentIdsOk returns a tuple with the IncidentIds field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalIncidentsUpdateAttributes) GetIncidentIdsOk() (*[]int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IncidentIds, true
}

// SetIncidentIds sets field value.
func (o *SecurityMonitoringSignalIncidentsUpdateAttributes) SetIncidentIds(v []int64) {
	o.IncidentIds = v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalIncidentsUpdateAttributes) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalIncidentsUpdateAttributes) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalIncidentsUpdateAttributes) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *SecurityMonitoringSignalIncidentsUpdateAttributes) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalIncidentsUpdateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["incident_ids"] = o.IncidentIds
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringSignalIncidentsUpdateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		IncidentIds *[]int64 `json:"incident_ids"`
	}{}
	all := struct {
		IncidentIds []int64 `json:"incident_ids"`
		Version     *int64  `json:"version,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.IncidentIds == nil {
		return fmt.Errorf("required field incident_ids missing")
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
	o.IncidentIds = all.IncidentIds
	o.Version = all.Version
	return nil
}
