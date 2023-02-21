// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// IncidentResponseAttributes The incident's attributes from a response.
type IncidentResponseAttributes struct {
	// Timestamp when the incident was created.
	Created *time.Time `json:"created,omitempty"`
	// Length of the incident's customer impact in seconds.
	// Equals the difference between `customer_impact_start` and `customer_impact_end`.
	CustomerImpactDuration *int64 `json:"customer_impact_duration,omitempty"`
	// Timestamp when customers were no longer impacted by the incident.
	CustomerImpactEnd datadog.NullableTime `json:"customer_impact_end,omitempty"`
	// A summary of the impact customers experienced during the incident.
	CustomerImpactScope datadog.NullableString `json:"customer_impact_scope,omitempty"`
	// Timestamp when customers began being impacted by the incident.
	CustomerImpactStart datadog.NullableTime `json:"customer_impact_start,omitempty"`
	// A flag indicating whether the incident caused customer impact.
	CustomerImpacted *bool `json:"customer_impacted,omitempty"`
	// Timestamp when the incident was detected.
	Detected datadog.NullableTime `json:"detected,omitempty"`
	// A condensed view of the user-defined fields attached to incidents.
	Fields map[string]IncidentFieldAttributes `json:"fields,omitempty"`
	// Timestamp when the incident was last modified.
	Modified *time.Time `json:"modified,omitempty"`
	// Notification handles that will be notified of the incident during update.
	NotificationHandles []IncidentNotificationHandle `json:"notification_handles,omitempty"`
	// The monotonically increasing integer ID for the incident.
	PublicId *int64 `json:"public_id,omitempty"`
	// Timestamp when the incident's state was last changed from active or stable to resolved or completed.
	Resolved datadog.NullableTime `json:"resolved,omitempty"`
	// The amount of time in seconds to detect the incident.
	// Equals the difference between `customer_impact_start` and `detected`.
	TimeToDetect *int64 `json:"time_to_detect,omitempty"`
	// The amount of time in seconds to call incident after detection. Equals the difference of `detected` and `created`.
	TimeToInternalResponse *int64 `json:"time_to_internal_response,omitempty"`
	// The amount of time in seconds to resolve customer impact after detecting the issue. Equals the difference between `customer_impact_end` and `detected`.
	TimeToRepair *int64 `json:"time_to_repair,omitempty"`
	// The amount of time in seconds to resolve the incident after it was created. Equals the difference between `created` and `resolved`.
	TimeToResolve *int64 `json:"time_to_resolve,omitempty"`
	// The title of the incident, which summarizes what happened.
	Title string `json:"title"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentResponseAttributes instantiates a new IncidentResponseAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentResponseAttributes(title string) *IncidentResponseAttributes {
	this := IncidentResponseAttributes{}
	this.Title = title
	return &this
}

// NewIncidentResponseAttributesWithDefaults instantiates a new IncidentResponseAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentResponseAttributesWithDefaults() *IncidentResponseAttributes {
	this := IncidentResponseAttributes{}
	return &this
}

// GetCreated returns the Created field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetCreated() time.Time {
	if o == nil || o.Created == nil {
		var ret time.Time
		return ret
	}
	return *o.Created
}

// GetCreatedOk returns a tuple with the Created field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetCreatedOk() (*time.Time, bool) {
	if o == nil || o.Created == nil {
		return nil, false
	}
	return o.Created, true
}

// HasCreated returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasCreated() bool {
	return o != nil && o.Created != nil
}

// SetCreated gets a reference to the given time.Time and assigns it to the Created field.
func (o *IncidentResponseAttributes) SetCreated(v time.Time) {
	o.Created = &v
}

// GetCustomerImpactDuration returns the CustomerImpactDuration field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetCustomerImpactDuration() int64 {
	if o == nil || o.CustomerImpactDuration == nil {
		var ret int64
		return ret
	}
	return *o.CustomerImpactDuration
}

// GetCustomerImpactDurationOk returns a tuple with the CustomerImpactDuration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetCustomerImpactDurationOk() (*int64, bool) {
	if o == nil || o.CustomerImpactDuration == nil {
		return nil, false
	}
	return o.CustomerImpactDuration, true
}

// HasCustomerImpactDuration returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasCustomerImpactDuration() bool {
	return o != nil && o.CustomerImpactDuration != nil
}

// SetCustomerImpactDuration gets a reference to the given int64 and assigns it to the CustomerImpactDuration field.
func (o *IncidentResponseAttributes) SetCustomerImpactDuration(v int64) {
	o.CustomerImpactDuration = &v
}

// GetCustomerImpactEnd returns the CustomerImpactEnd field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *IncidentResponseAttributes) GetCustomerImpactEnd() time.Time {
	if o == nil || o.CustomerImpactEnd.Get() == nil {
		var ret time.Time
		return ret
	}
	return *o.CustomerImpactEnd.Get()
}

// GetCustomerImpactEndOk returns a tuple with the CustomerImpactEnd field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *IncidentResponseAttributes) GetCustomerImpactEndOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return o.CustomerImpactEnd.Get(), o.CustomerImpactEnd.IsSet()
}

// HasCustomerImpactEnd returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasCustomerImpactEnd() bool {
	return o != nil && o.CustomerImpactEnd.IsSet()
}

// SetCustomerImpactEnd gets a reference to the given datadog.NullableTime and assigns it to the CustomerImpactEnd field.
func (o *IncidentResponseAttributes) SetCustomerImpactEnd(v time.Time) {
	o.CustomerImpactEnd.Set(&v)
}

// SetCustomerImpactEndNil sets the value for CustomerImpactEnd to be an explicit nil.
func (o *IncidentResponseAttributes) SetCustomerImpactEndNil() {
	o.CustomerImpactEnd.Set(nil)
}

// UnsetCustomerImpactEnd ensures that no value is present for CustomerImpactEnd, not even an explicit nil.
func (o *IncidentResponseAttributes) UnsetCustomerImpactEnd() {
	o.CustomerImpactEnd.Unset()
}

// GetCustomerImpactScope returns the CustomerImpactScope field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *IncidentResponseAttributes) GetCustomerImpactScope() string {
	if o == nil || o.CustomerImpactScope.Get() == nil {
		var ret string
		return ret
	}
	return *o.CustomerImpactScope.Get()
}

// GetCustomerImpactScopeOk returns a tuple with the CustomerImpactScope field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *IncidentResponseAttributes) GetCustomerImpactScopeOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.CustomerImpactScope.Get(), o.CustomerImpactScope.IsSet()
}

// HasCustomerImpactScope returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasCustomerImpactScope() bool {
	return o != nil && o.CustomerImpactScope.IsSet()
}

// SetCustomerImpactScope gets a reference to the given datadog.NullableString and assigns it to the CustomerImpactScope field.
func (o *IncidentResponseAttributes) SetCustomerImpactScope(v string) {
	o.CustomerImpactScope.Set(&v)
}

// SetCustomerImpactScopeNil sets the value for CustomerImpactScope to be an explicit nil.
func (o *IncidentResponseAttributes) SetCustomerImpactScopeNil() {
	o.CustomerImpactScope.Set(nil)
}

// UnsetCustomerImpactScope ensures that no value is present for CustomerImpactScope, not even an explicit nil.
func (o *IncidentResponseAttributes) UnsetCustomerImpactScope() {
	o.CustomerImpactScope.Unset()
}

// GetCustomerImpactStart returns the CustomerImpactStart field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *IncidentResponseAttributes) GetCustomerImpactStart() time.Time {
	if o == nil || o.CustomerImpactStart.Get() == nil {
		var ret time.Time
		return ret
	}
	return *o.CustomerImpactStart.Get()
}

// GetCustomerImpactStartOk returns a tuple with the CustomerImpactStart field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *IncidentResponseAttributes) GetCustomerImpactStartOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return o.CustomerImpactStart.Get(), o.CustomerImpactStart.IsSet()
}

// HasCustomerImpactStart returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasCustomerImpactStart() bool {
	return o != nil && o.CustomerImpactStart.IsSet()
}

// SetCustomerImpactStart gets a reference to the given datadog.NullableTime and assigns it to the CustomerImpactStart field.
func (o *IncidentResponseAttributes) SetCustomerImpactStart(v time.Time) {
	o.CustomerImpactStart.Set(&v)
}

// SetCustomerImpactStartNil sets the value for CustomerImpactStart to be an explicit nil.
func (o *IncidentResponseAttributes) SetCustomerImpactStartNil() {
	o.CustomerImpactStart.Set(nil)
}

// UnsetCustomerImpactStart ensures that no value is present for CustomerImpactStart, not even an explicit nil.
func (o *IncidentResponseAttributes) UnsetCustomerImpactStart() {
	o.CustomerImpactStart.Unset()
}

// GetCustomerImpacted returns the CustomerImpacted field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetCustomerImpacted() bool {
	if o == nil || o.CustomerImpacted == nil {
		var ret bool
		return ret
	}
	return *o.CustomerImpacted
}

// GetCustomerImpactedOk returns a tuple with the CustomerImpacted field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetCustomerImpactedOk() (*bool, bool) {
	if o == nil || o.CustomerImpacted == nil {
		return nil, false
	}
	return o.CustomerImpacted, true
}

// HasCustomerImpacted returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasCustomerImpacted() bool {
	return o != nil && o.CustomerImpacted != nil
}

// SetCustomerImpacted gets a reference to the given bool and assigns it to the CustomerImpacted field.
func (o *IncidentResponseAttributes) SetCustomerImpacted(v bool) {
	o.CustomerImpacted = &v
}

// GetDetected returns the Detected field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *IncidentResponseAttributes) GetDetected() time.Time {
	if o == nil || o.Detected.Get() == nil {
		var ret time.Time
		return ret
	}
	return *o.Detected.Get()
}

// GetDetectedOk returns a tuple with the Detected field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *IncidentResponseAttributes) GetDetectedOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return o.Detected.Get(), o.Detected.IsSet()
}

// HasDetected returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasDetected() bool {
	return o != nil && o.Detected.IsSet()
}

// SetDetected gets a reference to the given datadog.NullableTime and assigns it to the Detected field.
func (o *IncidentResponseAttributes) SetDetected(v time.Time) {
	o.Detected.Set(&v)
}

// SetDetectedNil sets the value for Detected to be an explicit nil.
func (o *IncidentResponseAttributes) SetDetectedNil() {
	o.Detected.Set(nil)
}

// UnsetDetected ensures that no value is present for Detected, not even an explicit nil.
func (o *IncidentResponseAttributes) UnsetDetected() {
	o.Detected.Unset()
}

// GetFields returns the Fields field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetFields() map[string]IncidentFieldAttributes {
	if o == nil || o.Fields == nil {
		var ret map[string]IncidentFieldAttributes
		return ret
	}
	return o.Fields
}

// GetFieldsOk returns a tuple with the Fields field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetFieldsOk() (*map[string]IncidentFieldAttributes, bool) {
	if o == nil || o.Fields == nil {
		return nil, false
	}
	return &o.Fields, true
}

// HasFields returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasFields() bool {
	return o != nil && o.Fields != nil
}

// SetFields gets a reference to the given map[string]IncidentFieldAttributes and assigns it to the Fields field.
func (o *IncidentResponseAttributes) SetFields(v map[string]IncidentFieldAttributes) {
	o.Fields = v
}

// GetModified returns the Modified field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetModified() time.Time {
	if o == nil || o.Modified == nil {
		var ret time.Time
		return ret
	}
	return *o.Modified
}

// GetModifiedOk returns a tuple with the Modified field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetModifiedOk() (*time.Time, bool) {
	if o == nil || o.Modified == nil {
		return nil, false
	}
	return o.Modified, true
}

// HasModified returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasModified() bool {
	return o != nil && o.Modified != nil
}

// SetModified gets a reference to the given time.Time and assigns it to the Modified field.
func (o *IncidentResponseAttributes) SetModified(v time.Time) {
	o.Modified = &v
}

// GetNotificationHandles returns the NotificationHandles field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *IncidentResponseAttributes) GetNotificationHandles() []IncidentNotificationHandle {
	if o == nil {
		var ret []IncidentNotificationHandle
		return ret
	}
	return o.NotificationHandles
}

// GetNotificationHandlesOk returns a tuple with the NotificationHandles field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *IncidentResponseAttributes) GetNotificationHandlesOk() (*[]IncidentNotificationHandle, bool) {
	if o == nil || o.NotificationHandles == nil {
		return nil, false
	}
	return &o.NotificationHandles, true
}

// HasNotificationHandles returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasNotificationHandles() bool {
	return o != nil && o.NotificationHandles != nil
}

// SetNotificationHandles gets a reference to the given []IncidentNotificationHandle and assigns it to the NotificationHandles field.
func (o *IncidentResponseAttributes) SetNotificationHandles(v []IncidentNotificationHandle) {
	o.NotificationHandles = v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetPublicId() int64 {
	if o == nil || o.PublicId == nil {
		var ret int64
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetPublicIdOk() (*int64, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasPublicId() bool {
	return o != nil && o.PublicId != nil
}

// SetPublicId gets a reference to the given int64 and assigns it to the PublicId field.
func (o *IncidentResponseAttributes) SetPublicId(v int64) {
	o.PublicId = &v
}

// GetResolved returns the Resolved field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *IncidentResponseAttributes) GetResolved() time.Time {
	if o == nil || o.Resolved.Get() == nil {
		var ret time.Time
		return ret
	}
	return *o.Resolved.Get()
}

// GetResolvedOk returns a tuple with the Resolved field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *IncidentResponseAttributes) GetResolvedOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return o.Resolved.Get(), o.Resolved.IsSet()
}

// HasResolved returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasResolved() bool {
	return o != nil && o.Resolved.IsSet()
}

// SetResolved gets a reference to the given datadog.NullableTime and assigns it to the Resolved field.
func (o *IncidentResponseAttributes) SetResolved(v time.Time) {
	o.Resolved.Set(&v)
}

// SetResolvedNil sets the value for Resolved to be an explicit nil.
func (o *IncidentResponseAttributes) SetResolvedNil() {
	o.Resolved.Set(nil)
}

// UnsetResolved ensures that no value is present for Resolved, not even an explicit nil.
func (o *IncidentResponseAttributes) UnsetResolved() {
	o.Resolved.Unset()
}

// GetTimeToDetect returns the TimeToDetect field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetTimeToDetect() int64 {
	if o == nil || o.TimeToDetect == nil {
		var ret int64
		return ret
	}
	return *o.TimeToDetect
}

// GetTimeToDetectOk returns a tuple with the TimeToDetect field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetTimeToDetectOk() (*int64, bool) {
	if o == nil || o.TimeToDetect == nil {
		return nil, false
	}
	return o.TimeToDetect, true
}

// HasTimeToDetect returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasTimeToDetect() bool {
	return o != nil && o.TimeToDetect != nil
}

// SetTimeToDetect gets a reference to the given int64 and assigns it to the TimeToDetect field.
func (o *IncidentResponseAttributes) SetTimeToDetect(v int64) {
	o.TimeToDetect = &v
}

// GetTimeToInternalResponse returns the TimeToInternalResponse field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetTimeToInternalResponse() int64 {
	if o == nil || o.TimeToInternalResponse == nil {
		var ret int64
		return ret
	}
	return *o.TimeToInternalResponse
}

// GetTimeToInternalResponseOk returns a tuple with the TimeToInternalResponse field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetTimeToInternalResponseOk() (*int64, bool) {
	if o == nil || o.TimeToInternalResponse == nil {
		return nil, false
	}
	return o.TimeToInternalResponse, true
}

// HasTimeToInternalResponse returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasTimeToInternalResponse() bool {
	return o != nil && o.TimeToInternalResponse != nil
}

// SetTimeToInternalResponse gets a reference to the given int64 and assigns it to the TimeToInternalResponse field.
func (o *IncidentResponseAttributes) SetTimeToInternalResponse(v int64) {
	o.TimeToInternalResponse = &v
}

// GetTimeToRepair returns the TimeToRepair field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetTimeToRepair() int64 {
	if o == nil || o.TimeToRepair == nil {
		var ret int64
		return ret
	}
	return *o.TimeToRepair
}

// GetTimeToRepairOk returns a tuple with the TimeToRepair field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetTimeToRepairOk() (*int64, bool) {
	if o == nil || o.TimeToRepair == nil {
		return nil, false
	}
	return o.TimeToRepair, true
}

// HasTimeToRepair returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasTimeToRepair() bool {
	return o != nil && o.TimeToRepair != nil
}

// SetTimeToRepair gets a reference to the given int64 and assigns it to the TimeToRepair field.
func (o *IncidentResponseAttributes) SetTimeToRepair(v int64) {
	o.TimeToRepair = &v
}

// GetTimeToResolve returns the TimeToResolve field value if set, zero value otherwise.
func (o *IncidentResponseAttributes) GetTimeToResolve() int64 {
	if o == nil || o.TimeToResolve == nil {
		var ret int64
		return ret
	}
	return *o.TimeToResolve
}

// GetTimeToResolveOk returns a tuple with the TimeToResolve field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetTimeToResolveOk() (*int64, bool) {
	if o == nil || o.TimeToResolve == nil {
		return nil, false
	}
	return o.TimeToResolve, true
}

// HasTimeToResolve returns a boolean if a field has been set.
func (o *IncidentResponseAttributes) HasTimeToResolve() bool {
	return o != nil && o.TimeToResolve != nil
}

// SetTimeToResolve gets a reference to the given int64 and assigns it to the TimeToResolve field.
func (o *IncidentResponseAttributes) SetTimeToResolve(v int64) {
	o.TimeToResolve = &v
}

// GetTitle returns the Title field value.
func (o *IncidentResponseAttributes) GetTitle() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Title
}

// GetTitleOk returns a tuple with the Title field value
// and a boolean to check if the value has been set.
func (o *IncidentResponseAttributes) GetTitleOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Title, true
}

// SetTitle sets field value.
func (o *IncidentResponseAttributes) SetTitle(v string) {
	o.Title = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentResponseAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Created != nil {
		if o.Created.Nanosecond() == 0 {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["created"] = o.Created.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.CustomerImpactDuration != nil {
		toSerialize["customer_impact_duration"] = o.CustomerImpactDuration
	}
	if o.CustomerImpactEnd.IsSet() {
		toSerialize["customer_impact_end"] = o.CustomerImpactEnd.Get()
	}
	if o.CustomerImpactScope.IsSet() {
		toSerialize["customer_impact_scope"] = o.CustomerImpactScope.Get()
	}
	if o.CustomerImpactStart.IsSet() {
		toSerialize["customer_impact_start"] = o.CustomerImpactStart.Get()
	}
	if o.CustomerImpacted != nil {
		toSerialize["customer_impacted"] = o.CustomerImpacted
	}
	if o.Detected.IsSet() {
		toSerialize["detected"] = o.Detected.Get()
	}
	if o.Fields != nil {
		toSerialize["fields"] = o.Fields
	}
	if o.Modified != nil {
		if o.Modified.Nanosecond() == 0 {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["modified"] = o.Modified.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.NotificationHandles != nil {
		toSerialize["notification_handles"] = o.NotificationHandles
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.Resolved.IsSet() {
		toSerialize["resolved"] = o.Resolved.Get()
	}
	if o.TimeToDetect != nil {
		toSerialize["time_to_detect"] = o.TimeToDetect
	}
	if o.TimeToInternalResponse != nil {
		toSerialize["time_to_internal_response"] = o.TimeToInternalResponse
	}
	if o.TimeToRepair != nil {
		toSerialize["time_to_repair"] = o.TimeToRepair
	}
	if o.TimeToResolve != nil {
		toSerialize["time_to_resolve"] = o.TimeToResolve
	}
	toSerialize["title"] = o.Title

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentResponseAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Title *string `json:"title"`
	}{}
	all := struct {
		Created                *time.Time                         `json:"created,omitempty"`
		CustomerImpactDuration *int64                             `json:"customer_impact_duration,omitempty"`
		CustomerImpactEnd      datadog.NullableTime               `json:"customer_impact_end,omitempty"`
		CustomerImpactScope    datadog.NullableString             `json:"customer_impact_scope,omitempty"`
		CustomerImpactStart    datadog.NullableTime               `json:"customer_impact_start,omitempty"`
		CustomerImpacted       *bool                              `json:"customer_impacted,omitempty"`
		Detected               datadog.NullableTime               `json:"detected,omitempty"`
		Fields                 map[string]IncidentFieldAttributes `json:"fields,omitempty"`
		Modified               *time.Time                         `json:"modified,omitempty"`
		NotificationHandles    []IncidentNotificationHandle       `json:"notification_handles,omitempty"`
		PublicId               *int64                             `json:"public_id,omitempty"`
		Resolved               datadog.NullableTime               `json:"resolved,omitempty"`
		TimeToDetect           *int64                             `json:"time_to_detect,omitempty"`
		TimeToInternalResponse *int64                             `json:"time_to_internal_response,omitempty"`
		TimeToRepair           *int64                             `json:"time_to_repair,omitempty"`
		TimeToResolve          *int64                             `json:"time_to_resolve,omitempty"`
		Title                  string                             `json:"title"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Title == nil {
		return fmt.Errorf("required field title missing")
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
	o.Created = all.Created
	o.CustomerImpactDuration = all.CustomerImpactDuration
	o.CustomerImpactEnd = all.CustomerImpactEnd
	o.CustomerImpactScope = all.CustomerImpactScope
	o.CustomerImpactStart = all.CustomerImpactStart
	o.CustomerImpacted = all.CustomerImpacted
	o.Detected = all.Detected
	o.Fields = all.Fields
	o.Modified = all.Modified
	o.NotificationHandles = all.NotificationHandles
	o.PublicId = all.PublicId
	o.Resolved = all.Resolved
	o.TimeToDetect = all.TimeToDetect
	o.TimeToInternalResponse = all.TimeToInternalResponse
	o.TimeToRepair = all.TimeToRepair
	o.TimeToResolve = all.TimeToResolve
	o.Title = all.Title
	return nil
}
