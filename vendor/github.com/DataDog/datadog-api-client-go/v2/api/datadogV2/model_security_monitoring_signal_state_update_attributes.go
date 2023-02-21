// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringSignalStateUpdateAttributes Attributes describing the change of state of a security signal.
type SecurityMonitoringSignalStateUpdateAttributes struct {
	// Optional comment to display on archived signals.
	ArchiveComment *string `json:"archive_comment,omitempty"`
	// Reason a signal is archived.
	ArchiveReason *SecurityMonitoringSignalArchiveReason `json:"archive_reason,omitempty"`
	// The new triage state of the signal.
	State SecurityMonitoringSignalState `json:"state"`
	// Version of the updated signal. If server side version is higher, update will be rejected.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalStateUpdateAttributes instantiates a new SecurityMonitoringSignalStateUpdateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalStateUpdateAttributes(state SecurityMonitoringSignalState) *SecurityMonitoringSignalStateUpdateAttributes {
	this := SecurityMonitoringSignalStateUpdateAttributes{}
	this.State = state
	return &this
}

// NewSecurityMonitoringSignalStateUpdateAttributesWithDefaults instantiates a new SecurityMonitoringSignalStateUpdateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalStateUpdateAttributesWithDefaults() *SecurityMonitoringSignalStateUpdateAttributes {
	this := SecurityMonitoringSignalStateUpdateAttributes{}
	return &this
}

// GetArchiveComment returns the ArchiveComment field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalStateUpdateAttributes) GetArchiveComment() string {
	if o == nil || o.ArchiveComment == nil {
		var ret string
		return ret
	}
	return *o.ArchiveComment
}

// GetArchiveCommentOk returns a tuple with the ArchiveComment field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalStateUpdateAttributes) GetArchiveCommentOk() (*string, bool) {
	if o == nil || o.ArchiveComment == nil {
		return nil, false
	}
	return o.ArchiveComment, true
}

// HasArchiveComment returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalStateUpdateAttributes) HasArchiveComment() bool {
	return o != nil && o.ArchiveComment != nil
}

// SetArchiveComment gets a reference to the given string and assigns it to the ArchiveComment field.
func (o *SecurityMonitoringSignalStateUpdateAttributes) SetArchiveComment(v string) {
	o.ArchiveComment = &v
}

// GetArchiveReason returns the ArchiveReason field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalStateUpdateAttributes) GetArchiveReason() SecurityMonitoringSignalArchiveReason {
	if o == nil || o.ArchiveReason == nil {
		var ret SecurityMonitoringSignalArchiveReason
		return ret
	}
	return *o.ArchiveReason
}

// GetArchiveReasonOk returns a tuple with the ArchiveReason field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalStateUpdateAttributes) GetArchiveReasonOk() (*SecurityMonitoringSignalArchiveReason, bool) {
	if o == nil || o.ArchiveReason == nil {
		return nil, false
	}
	return o.ArchiveReason, true
}

// HasArchiveReason returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalStateUpdateAttributes) HasArchiveReason() bool {
	return o != nil && o.ArchiveReason != nil
}

// SetArchiveReason gets a reference to the given SecurityMonitoringSignalArchiveReason and assigns it to the ArchiveReason field.
func (o *SecurityMonitoringSignalStateUpdateAttributes) SetArchiveReason(v SecurityMonitoringSignalArchiveReason) {
	o.ArchiveReason = &v
}

// GetState returns the State field value.
func (o *SecurityMonitoringSignalStateUpdateAttributes) GetState() SecurityMonitoringSignalState {
	if o == nil {
		var ret SecurityMonitoringSignalState
		return ret
	}
	return o.State
}

// GetStateOk returns a tuple with the State field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalStateUpdateAttributes) GetStateOk() (*SecurityMonitoringSignalState, bool) {
	if o == nil {
		return nil, false
	}
	return &o.State, true
}

// SetState sets field value.
func (o *SecurityMonitoringSignalStateUpdateAttributes) SetState(v SecurityMonitoringSignalState) {
	o.State = v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalStateUpdateAttributes) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalStateUpdateAttributes) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalStateUpdateAttributes) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *SecurityMonitoringSignalStateUpdateAttributes) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalStateUpdateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ArchiveComment != nil {
		toSerialize["archive_comment"] = o.ArchiveComment
	}
	if o.ArchiveReason != nil {
		toSerialize["archive_reason"] = o.ArchiveReason
	}
	toSerialize["state"] = o.State
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringSignalStateUpdateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		State *SecurityMonitoringSignalState `json:"state"`
	}{}
	all := struct {
		ArchiveComment *string                                `json:"archive_comment,omitempty"`
		ArchiveReason  *SecurityMonitoringSignalArchiveReason `json:"archive_reason,omitempty"`
		State          SecurityMonitoringSignalState          `json:"state"`
		Version        *int64                                 `json:"version,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.State == nil {
		return fmt.Errorf("required field state missing")
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
	if v := all.ArchiveReason; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.State; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.ArchiveComment = all.ArchiveComment
	o.ArchiveReason = all.ArchiveReason
	o.State = all.State
	o.Version = all.Version
	return nil
}
