// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SecurityMonitoringSignalTriageAttributes Attributes describing a triage state update operation over a security signal.
type SecurityMonitoringSignalTriageAttributes struct {
	// Optional comment to display on archived signals.
	ArchiveComment *string `json:"archive_comment,omitempty"`
	// Timestamp of the last edit to the comment.
	ArchiveCommentTimestamp *int64 `json:"archive_comment_timestamp,omitempty"`
	// Object representing a given user entity.
	ArchiveCommentUser *SecurityMonitoringTriageUser `json:"archive_comment_user,omitempty"`
	// Reason a signal is archived.
	ArchiveReason *SecurityMonitoringSignalArchiveReason `json:"archive_reason,omitempty"`
	// Object representing a given user entity.
	Assignee SecurityMonitoringTriageUser `json:"assignee"`
	// Array of incidents that are associated with this signal.
	IncidentIds []int64 `json:"incident_ids"`
	// The new triage state of the signal.
	State SecurityMonitoringSignalState `json:"state"`
	// Timestamp of the last update to the signal state.
	StateUpdateTimestamp *int64 `json:"state_update_timestamp,omitempty"`
	// Object representing a given user entity.
	StateUpdateUser *SecurityMonitoringTriageUser `json:"state_update_user,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalTriageAttributes instantiates a new SecurityMonitoringSignalTriageAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalTriageAttributes(assignee SecurityMonitoringTriageUser, incidentIds []int64, state SecurityMonitoringSignalState) *SecurityMonitoringSignalTriageAttributes {
	this := SecurityMonitoringSignalTriageAttributes{}
	this.Assignee = assignee
	this.IncidentIds = incidentIds
	this.State = state
	return &this
}

// NewSecurityMonitoringSignalTriageAttributesWithDefaults instantiates a new SecurityMonitoringSignalTriageAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalTriageAttributesWithDefaults() *SecurityMonitoringSignalTriageAttributes {
	this := SecurityMonitoringSignalTriageAttributes{}
	return &this
}

// GetArchiveComment returns the ArchiveComment field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalTriageAttributes) GetArchiveComment() string {
	if o == nil || o.ArchiveComment == nil {
		var ret string
		return ret
	}
	return *o.ArchiveComment
}

// GetArchiveCommentOk returns a tuple with the ArchiveComment field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageAttributes) GetArchiveCommentOk() (*string, bool) {
	if o == nil || o.ArchiveComment == nil {
		return nil, false
	}
	return o.ArchiveComment, true
}

// HasArchiveComment returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalTriageAttributes) HasArchiveComment() bool {
	return o != nil && o.ArchiveComment != nil
}

// SetArchiveComment gets a reference to the given string and assigns it to the ArchiveComment field.
func (o *SecurityMonitoringSignalTriageAttributes) SetArchiveComment(v string) {
	o.ArchiveComment = &v
}

// GetArchiveCommentTimestamp returns the ArchiveCommentTimestamp field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalTriageAttributes) GetArchiveCommentTimestamp() int64 {
	if o == nil || o.ArchiveCommentTimestamp == nil {
		var ret int64
		return ret
	}
	return *o.ArchiveCommentTimestamp
}

// GetArchiveCommentTimestampOk returns a tuple with the ArchiveCommentTimestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageAttributes) GetArchiveCommentTimestampOk() (*int64, bool) {
	if o == nil || o.ArchiveCommentTimestamp == nil {
		return nil, false
	}
	return o.ArchiveCommentTimestamp, true
}

// HasArchiveCommentTimestamp returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalTriageAttributes) HasArchiveCommentTimestamp() bool {
	return o != nil && o.ArchiveCommentTimestamp != nil
}

// SetArchiveCommentTimestamp gets a reference to the given int64 and assigns it to the ArchiveCommentTimestamp field.
func (o *SecurityMonitoringSignalTriageAttributes) SetArchiveCommentTimestamp(v int64) {
	o.ArchiveCommentTimestamp = &v
}

// GetArchiveCommentUser returns the ArchiveCommentUser field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalTriageAttributes) GetArchiveCommentUser() SecurityMonitoringTriageUser {
	if o == nil || o.ArchiveCommentUser == nil {
		var ret SecurityMonitoringTriageUser
		return ret
	}
	return *o.ArchiveCommentUser
}

// GetArchiveCommentUserOk returns a tuple with the ArchiveCommentUser field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageAttributes) GetArchiveCommentUserOk() (*SecurityMonitoringTriageUser, bool) {
	if o == nil || o.ArchiveCommentUser == nil {
		return nil, false
	}
	return o.ArchiveCommentUser, true
}

// HasArchiveCommentUser returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalTriageAttributes) HasArchiveCommentUser() bool {
	return o != nil && o.ArchiveCommentUser != nil
}

// SetArchiveCommentUser gets a reference to the given SecurityMonitoringTriageUser and assigns it to the ArchiveCommentUser field.
func (o *SecurityMonitoringSignalTriageAttributes) SetArchiveCommentUser(v SecurityMonitoringTriageUser) {
	o.ArchiveCommentUser = &v
}

// GetArchiveReason returns the ArchiveReason field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalTriageAttributes) GetArchiveReason() SecurityMonitoringSignalArchiveReason {
	if o == nil || o.ArchiveReason == nil {
		var ret SecurityMonitoringSignalArchiveReason
		return ret
	}
	return *o.ArchiveReason
}

// GetArchiveReasonOk returns a tuple with the ArchiveReason field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageAttributes) GetArchiveReasonOk() (*SecurityMonitoringSignalArchiveReason, bool) {
	if o == nil || o.ArchiveReason == nil {
		return nil, false
	}
	return o.ArchiveReason, true
}

// HasArchiveReason returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalTriageAttributes) HasArchiveReason() bool {
	return o != nil && o.ArchiveReason != nil
}

// SetArchiveReason gets a reference to the given SecurityMonitoringSignalArchiveReason and assigns it to the ArchiveReason field.
func (o *SecurityMonitoringSignalTriageAttributes) SetArchiveReason(v SecurityMonitoringSignalArchiveReason) {
	o.ArchiveReason = &v
}

// GetAssignee returns the Assignee field value.
func (o *SecurityMonitoringSignalTriageAttributes) GetAssignee() SecurityMonitoringTriageUser {
	if o == nil {
		var ret SecurityMonitoringTriageUser
		return ret
	}
	return o.Assignee
}

// GetAssigneeOk returns a tuple with the Assignee field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageAttributes) GetAssigneeOk() (*SecurityMonitoringTriageUser, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Assignee, true
}

// SetAssignee sets field value.
func (o *SecurityMonitoringSignalTriageAttributes) SetAssignee(v SecurityMonitoringTriageUser) {
	o.Assignee = v
}

// GetIncidentIds returns the IncidentIds field value.
func (o *SecurityMonitoringSignalTriageAttributes) GetIncidentIds() []int64 {
	if o == nil {
		var ret []int64
		return ret
	}
	return o.IncidentIds
}

// GetIncidentIdsOk returns a tuple with the IncidentIds field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageAttributes) GetIncidentIdsOk() (*[]int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.IncidentIds, true
}

// SetIncidentIds sets field value.
func (o *SecurityMonitoringSignalTriageAttributes) SetIncidentIds(v []int64) {
	o.IncidentIds = v
}

// GetState returns the State field value.
func (o *SecurityMonitoringSignalTriageAttributes) GetState() SecurityMonitoringSignalState {
	if o == nil {
		var ret SecurityMonitoringSignalState
		return ret
	}
	return o.State
}

// GetStateOk returns a tuple with the State field value
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageAttributes) GetStateOk() (*SecurityMonitoringSignalState, bool) {
	if o == nil {
		return nil, false
	}
	return &o.State, true
}

// SetState sets field value.
func (o *SecurityMonitoringSignalTriageAttributes) SetState(v SecurityMonitoringSignalState) {
	o.State = v
}

// GetStateUpdateTimestamp returns the StateUpdateTimestamp field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalTriageAttributes) GetStateUpdateTimestamp() int64 {
	if o == nil || o.StateUpdateTimestamp == nil {
		var ret int64
		return ret
	}
	return *o.StateUpdateTimestamp
}

// GetStateUpdateTimestampOk returns a tuple with the StateUpdateTimestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageAttributes) GetStateUpdateTimestampOk() (*int64, bool) {
	if o == nil || o.StateUpdateTimestamp == nil {
		return nil, false
	}
	return o.StateUpdateTimestamp, true
}

// HasStateUpdateTimestamp returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalTriageAttributes) HasStateUpdateTimestamp() bool {
	return o != nil && o.StateUpdateTimestamp != nil
}

// SetStateUpdateTimestamp gets a reference to the given int64 and assigns it to the StateUpdateTimestamp field.
func (o *SecurityMonitoringSignalTriageAttributes) SetStateUpdateTimestamp(v int64) {
	o.StateUpdateTimestamp = &v
}

// GetStateUpdateUser returns the StateUpdateUser field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalTriageAttributes) GetStateUpdateUser() SecurityMonitoringTriageUser {
	if o == nil || o.StateUpdateUser == nil {
		var ret SecurityMonitoringTriageUser
		return ret
	}
	return *o.StateUpdateUser
}

// GetStateUpdateUserOk returns a tuple with the StateUpdateUser field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalTriageAttributes) GetStateUpdateUserOk() (*SecurityMonitoringTriageUser, bool) {
	if o == nil || o.StateUpdateUser == nil {
		return nil, false
	}
	return o.StateUpdateUser, true
}

// HasStateUpdateUser returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalTriageAttributes) HasStateUpdateUser() bool {
	return o != nil && o.StateUpdateUser != nil
}

// SetStateUpdateUser gets a reference to the given SecurityMonitoringTriageUser and assigns it to the StateUpdateUser field.
func (o *SecurityMonitoringSignalTriageAttributes) SetStateUpdateUser(v SecurityMonitoringTriageUser) {
	o.StateUpdateUser = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalTriageAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ArchiveComment != nil {
		toSerialize["archive_comment"] = o.ArchiveComment
	}
	if o.ArchiveCommentTimestamp != nil {
		toSerialize["archive_comment_timestamp"] = o.ArchiveCommentTimestamp
	}
	if o.ArchiveCommentUser != nil {
		toSerialize["archive_comment_user"] = o.ArchiveCommentUser
	}
	if o.ArchiveReason != nil {
		toSerialize["archive_reason"] = o.ArchiveReason
	}
	toSerialize["assignee"] = o.Assignee
	toSerialize["incident_ids"] = o.IncidentIds
	toSerialize["state"] = o.State
	if o.StateUpdateTimestamp != nil {
		toSerialize["state_update_timestamp"] = o.StateUpdateTimestamp
	}
	if o.StateUpdateUser != nil {
		toSerialize["state_update_user"] = o.StateUpdateUser
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringSignalTriageAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Assignee    *SecurityMonitoringTriageUser  `json:"assignee"`
		IncidentIds *[]int64                       `json:"incident_ids"`
		State       *SecurityMonitoringSignalState `json:"state"`
	}{}
	all := struct {
		ArchiveComment          *string                                `json:"archive_comment,omitempty"`
		ArchiveCommentTimestamp *int64                                 `json:"archive_comment_timestamp,omitempty"`
		ArchiveCommentUser      *SecurityMonitoringTriageUser          `json:"archive_comment_user,omitempty"`
		ArchiveReason           *SecurityMonitoringSignalArchiveReason `json:"archive_reason,omitempty"`
		Assignee                SecurityMonitoringTriageUser           `json:"assignee"`
		IncidentIds             []int64                                `json:"incident_ids"`
		State                   SecurityMonitoringSignalState          `json:"state"`
		StateUpdateTimestamp    *int64                                 `json:"state_update_timestamp,omitempty"`
		StateUpdateUser         *SecurityMonitoringTriageUser          `json:"state_update_user,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Assignee == nil {
		return fmt.Errorf("required field assignee missing")
	}
	if required.IncidentIds == nil {
		return fmt.Errorf("required field incident_ids missing")
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
	o.ArchiveCommentTimestamp = all.ArchiveCommentTimestamp
	if all.ArchiveCommentUser != nil && all.ArchiveCommentUser.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ArchiveCommentUser = all.ArchiveCommentUser
	o.ArchiveReason = all.ArchiveReason
	if all.Assignee.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Assignee = all.Assignee
	o.IncidentIds = all.IncidentIds
	o.State = all.State
	o.StateUpdateTimestamp = all.StateUpdateTimestamp
	if all.StateUpdateUser != nil && all.StateUpdateUser.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.StateUpdateUser = all.StateUpdateUser
	return nil
}
