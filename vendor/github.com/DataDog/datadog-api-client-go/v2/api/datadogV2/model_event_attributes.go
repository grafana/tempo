// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// EventAttributes Object description of attributes from your event.
type EventAttributes struct {
	// Aggregation key of the event.
	AggregationKey *string `json:"aggregation_key,omitempty"`
	// POSIX timestamp of the event. Must be sent as an integer (no quotation marks).
	// Limited to events no older than 18 hours.
	DateHappened *int64 `json:"date_happened,omitempty"`
	// A device name.
	DeviceName *string `json:"device_name,omitempty"`
	// The duration between the triggering of the event and its recovery in nanoseconds.
	Duration *int64 `json:"duration,omitempty"`
	// The event title.
	EventObject *string `json:"event_object,omitempty"`
	// The metadata associated with a request.
	Evt *Event `json:"evt,omitempty"`
	// Host name to associate with the event.
	// Any tags associated with the host are also applied to this event.
	Hostname *string `json:"hostname,omitempty"`
	// Attributes from the monitor that triggered the event.
	Monitor NullableMonitorType `json:"monitor,omitempty"`
	// List of groups referred to in the event.
	MonitorGroups []string `json:"monitor_groups,omitempty"`
	// ID of the monitor that triggered the event. When an event isn't related to a monitor, this field is empty.
	MonitorId datadog.NullableInt64 `json:"monitor_id,omitempty"`
	// The priority of the event's monitor. For example, `normal` or `low`.
	Priority NullableEventPriority `json:"priority,omitempty"`
	// Related event ID.
	RelatedEventId *int64 `json:"related_event_id,omitempty"`
	// Service that triggered the event.
	Service *string `json:"service,omitempty"`
	// The type of event being posted.
	// For example, `nagios`, `hudson`, `jenkins`, `my_apps`, `chef`, `puppet`, `git` or `bitbucket`.
	// The list of standard source attribute values is [available here](https://docs.datadoghq.com/integrations/faq/list-of-api-source-attribute-value).
	SourceTypeName *string `json:"source_type_name,omitempty"`
	// Identifier for the source of the event, such as a monitor alert, an externally-submitted event, or an integration.
	Sourcecategory *string `json:"sourcecategory,omitempty"`
	// If an alert event is enabled, its status is one of the following:
	// `failure`, `error`, `warning`, `info`, `success`, `user_update`,
	// `recommendation`, or `snapshot`.
	Status *EventStatusType `json:"status,omitempty"`
	// A list of tags to apply to the event.
	Tags []string `json:"tags,omitempty"`
	// POSIX timestamp of your event in milliseconds.
	Timestamp *int64 `json:"timestamp,omitempty"`
	// The event title.
	Title *string `json:"title,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewEventAttributes instantiates a new EventAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewEventAttributes() *EventAttributes {
	this := EventAttributes{}
	return &this
}

// NewEventAttributesWithDefaults instantiates a new EventAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewEventAttributesWithDefaults() *EventAttributes {
	this := EventAttributes{}
	return &this
}

// GetAggregationKey returns the AggregationKey field value if set, zero value otherwise.
func (o *EventAttributes) GetAggregationKey() string {
	if o == nil || o.AggregationKey == nil {
		var ret string
		return ret
	}
	return *o.AggregationKey
}

// GetAggregationKeyOk returns a tuple with the AggregationKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetAggregationKeyOk() (*string, bool) {
	if o == nil || o.AggregationKey == nil {
		return nil, false
	}
	return o.AggregationKey, true
}

// HasAggregationKey returns a boolean if a field has been set.
func (o *EventAttributes) HasAggregationKey() bool {
	return o != nil && o.AggregationKey != nil
}

// SetAggregationKey gets a reference to the given string and assigns it to the AggregationKey field.
func (o *EventAttributes) SetAggregationKey(v string) {
	o.AggregationKey = &v
}

// GetDateHappened returns the DateHappened field value if set, zero value otherwise.
func (o *EventAttributes) GetDateHappened() int64 {
	if o == nil || o.DateHappened == nil {
		var ret int64
		return ret
	}
	return *o.DateHappened
}

// GetDateHappenedOk returns a tuple with the DateHappened field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetDateHappenedOk() (*int64, bool) {
	if o == nil || o.DateHappened == nil {
		return nil, false
	}
	return o.DateHappened, true
}

// HasDateHappened returns a boolean if a field has been set.
func (o *EventAttributes) HasDateHappened() bool {
	return o != nil && o.DateHappened != nil
}

// SetDateHappened gets a reference to the given int64 and assigns it to the DateHappened field.
func (o *EventAttributes) SetDateHappened(v int64) {
	o.DateHappened = &v
}

// GetDeviceName returns the DeviceName field value if set, zero value otherwise.
func (o *EventAttributes) GetDeviceName() string {
	if o == nil || o.DeviceName == nil {
		var ret string
		return ret
	}
	return *o.DeviceName
}

// GetDeviceNameOk returns a tuple with the DeviceName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetDeviceNameOk() (*string, bool) {
	if o == nil || o.DeviceName == nil {
		return nil, false
	}
	return o.DeviceName, true
}

// HasDeviceName returns a boolean if a field has been set.
func (o *EventAttributes) HasDeviceName() bool {
	return o != nil && o.DeviceName != nil
}

// SetDeviceName gets a reference to the given string and assigns it to the DeviceName field.
func (o *EventAttributes) SetDeviceName(v string) {
	o.DeviceName = &v
}

// GetDuration returns the Duration field value if set, zero value otherwise.
func (o *EventAttributes) GetDuration() int64 {
	if o == nil || o.Duration == nil {
		var ret int64
		return ret
	}
	return *o.Duration
}

// GetDurationOk returns a tuple with the Duration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetDurationOk() (*int64, bool) {
	if o == nil || o.Duration == nil {
		return nil, false
	}
	return o.Duration, true
}

// HasDuration returns a boolean if a field has been set.
func (o *EventAttributes) HasDuration() bool {
	return o != nil && o.Duration != nil
}

// SetDuration gets a reference to the given int64 and assigns it to the Duration field.
func (o *EventAttributes) SetDuration(v int64) {
	o.Duration = &v
}

// GetEventObject returns the EventObject field value if set, zero value otherwise.
func (o *EventAttributes) GetEventObject() string {
	if o == nil || o.EventObject == nil {
		var ret string
		return ret
	}
	return *o.EventObject
}

// GetEventObjectOk returns a tuple with the EventObject field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetEventObjectOk() (*string, bool) {
	if o == nil || o.EventObject == nil {
		return nil, false
	}
	return o.EventObject, true
}

// HasEventObject returns a boolean if a field has been set.
func (o *EventAttributes) HasEventObject() bool {
	return o != nil && o.EventObject != nil
}

// SetEventObject gets a reference to the given string and assigns it to the EventObject field.
func (o *EventAttributes) SetEventObject(v string) {
	o.EventObject = &v
}

// GetEvt returns the Evt field value if set, zero value otherwise.
func (o *EventAttributes) GetEvt() Event {
	if o == nil || o.Evt == nil {
		var ret Event
		return ret
	}
	return *o.Evt
}

// GetEvtOk returns a tuple with the Evt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetEvtOk() (*Event, bool) {
	if o == nil || o.Evt == nil {
		return nil, false
	}
	return o.Evt, true
}

// HasEvt returns a boolean if a field has been set.
func (o *EventAttributes) HasEvt() bool {
	return o != nil && o.Evt != nil
}

// SetEvt gets a reference to the given Event and assigns it to the Evt field.
func (o *EventAttributes) SetEvt(v Event) {
	o.Evt = &v
}

// GetHostname returns the Hostname field value if set, zero value otherwise.
func (o *EventAttributes) GetHostname() string {
	if o == nil || o.Hostname == nil {
		var ret string
		return ret
	}
	return *o.Hostname
}

// GetHostnameOk returns a tuple with the Hostname field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetHostnameOk() (*string, bool) {
	if o == nil || o.Hostname == nil {
		return nil, false
	}
	return o.Hostname, true
}

// HasHostname returns a boolean if a field has been set.
func (o *EventAttributes) HasHostname() bool {
	return o != nil && o.Hostname != nil
}

// SetHostname gets a reference to the given string and assigns it to the Hostname field.
func (o *EventAttributes) SetHostname(v string) {
	o.Hostname = &v
}

// GetMonitor returns the Monitor field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EventAttributes) GetMonitor() MonitorType {
	if o == nil || o.Monitor.Get() == nil {
		var ret MonitorType
		return ret
	}
	return *o.Monitor.Get()
}

// GetMonitorOk returns a tuple with the Monitor field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *EventAttributes) GetMonitorOk() (*MonitorType, bool) {
	if o == nil {
		return nil, false
	}
	return o.Monitor.Get(), o.Monitor.IsSet()
}

// HasMonitor returns a boolean if a field has been set.
func (o *EventAttributes) HasMonitor() bool {
	return o != nil && o.Monitor.IsSet()
}

// SetMonitor gets a reference to the given NullableMonitorType and assigns it to the Monitor field.
func (o *EventAttributes) SetMonitor(v MonitorType) {
	o.Monitor.Set(&v)
}

// SetMonitorNil sets the value for Monitor to be an explicit nil.
func (o *EventAttributes) SetMonitorNil() {
	o.Monitor.Set(nil)
}

// UnsetMonitor ensures that no value is present for Monitor, not even an explicit nil.
func (o *EventAttributes) UnsetMonitor() {
	o.Monitor.Unset()
}

// GetMonitorGroups returns the MonitorGroups field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EventAttributes) GetMonitorGroups() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.MonitorGroups
}

// GetMonitorGroupsOk returns a tuple with the MonitorGroups field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *EventAttributes) GetMonitorGroupsOk() (*[]string, bool) {
	if o == nil || o.MonitorGroups == nil {
		return nil, false
	}
	return &o.MonitorGroups, true
}

// HasMonitorGroups returns a boolean if a field has been set.
func (o *EventAttributes) HasMonitorGroups() bool {
	return o != nil && o.MonitorGroups != nil
}

// SetMonitorGroups gets a reference to the given []string and assigns it to the MonitorGroups field.
func (o *EventAttributes) SetMonitorGroups(v []string) {
	o.MonitorGroups = v
}

// GetMonitorId returns the MonitorId field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EventAttributes) GetMonitorId() int64 {
	if o == nil || o.MonitorId.Get() == nil {
		var ret int64
		return ret
	}
	return *o.MonitorId.Get()
}

// GetMonitorIdOk returns a tuple with the MonitorId field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *EventAttributes) GetMonitorIdOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.MonitorId.Get(), o.MonitorId.IsSet()
}

// HasMonitorId returns a boolean if a field has been set.
func (o *EventAttributes) HasMonitorId() bool {
	return o != nil && o.MonitorId.IsSet()
}

// SetMonitorId gets a reference to the given datadog.NullableInt64 and assigns it to the MonitorId field.
func (o *EventAttributes) SetMonitorId(v int64) {
	o.MonitorId.Set(&v)
}

// SetMonitorIdNil sets the value for MonitorId to be an explicit nil.
func (o *EventAttributes) SetMonitorIdNil() {
	o.MonitorId.Set(nil)
}

// UnsetMonitorId ensures that no value is present for MonitorId, not even an explicit nil.
func (o *EventAttributes) UnsetMonitorId() {
	o.MonitorId.Unset()
}

// GetPriority returns the Priority field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EventAttributes) GetPriority() EventPriority {
	if o == nil || o.Priority.Get() == nil {
		var ret EventPriority
		return ret
	}
	return *o.Priority.Get()
}

// GetPriorityOk returns a tuple with the Priority field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *EventAttributes) GetPriorityOk() (*EventPriority, bool) {
	if o == nil {
		return nil, false
	}
	return o.Priority.Get(), o.Priority.IsSet()
}

// HasPriority returns a boolean if a field has been set.
func (o *EventAttributes) HasPriority() bool {
	return o != nil && o.Priority.IsSet()
}

// SetPriority gets a reference to the given NullableEventPriority and assigns it to the Priority field.
func (o *EventAttributes) SetPriority(v EventPriority) {
	o.Priority.Set(&v)
}

// SetPriorityNil sets the value for Priority to be an explicit nil.
func (o *EventAttributes) SetPriorityNil() {
	o.Priority.Set(nil)
}

// UnsetPriority ensures that no value is present for Priority, not even an explicit nil.
func (o *EventAttributes) UnsetPriority() {
	o.Priority.Unset()
}

// GetRelatedEventId returns the RelatedEventId field value if set, zero value otherwise.
func (o *EventAttributes) GetRelatedEventId() int64 {
	if o == nil || o.RelatedEventId == nil {
		var ret int64
		return ret
	}
	return *o.RelatedEventId
}

// GetRelatedEventIdOk returns a tuple with the RelatedEventId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetRelatedEventIdOk() (*int64, bool) {
	if o == nil || o.RelatedEventId == nil {
		return nil, false
	}
	return o.RelatedEventId, true
}

// HasRelatedEventId returns a boolean if a field has been set.
func (o *EventAttributes) HasRelatedEventId() bool {
	return o != nil && o.RelatedEventId != nil
}

// SetRelatedEventId gets a reference to the given int64 and assigns it to the RelatedEventId field.
func (o *EventAttributes) SetRelatedEventId(v int64) {
	o.RelatedEventId = &v
}

// GetService returns the Service field value if set, zero value otherwise.
func (o *EventAttributes) GetService() string {
	if o == nil || o.Service == nil {
		var ret string
		return ret
	}
	return *o.Service
}

// GetServiceOk returns a tuple with the Service field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetServiceOk() (*string, bool) {
	if o == nil || o.Service == nil {
		return nil, false
	}
	return o.Service, true
}

// HasService returns a boolean if a field has been set.
func (o *EventAttributes) HasService() bool {
	return o != nil && o.Service != nil
}

// SetService gets a reference to the given string and assigns it to the Service field.
func (o *EventAttributes) SetService(v string) {
	o.Service = &v
}

// GetSourceTypeName returns the SourceTypeName field value if set, zero value otherwise.
func (o *EventAttributes) GetSourceTypeName() string {
	if o == nil || o.SourceTypeName == nil {
		var ret string
		return ret
	}
	return *o.SourceTypeName
}

// GetSourceTypeNameOk returns a tuple with the SourceTypeName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetSourceTypeNameOk() (*string, bool) {
	if o == nil || o.SourceTypeName == nil {
		return nil, false
	}
	return o.SourceTypeName, true
}

// HasSourceTypeName returns a boolean if a field has been set.
func (o *EventAttributes) HasSourceTypeName() bool {
	return o != nil && o.SourceTypeName != nil
}

// SetSourceTypeName gets a reference to the given string and assigns it to the SourceTypeName field.
func (o *EventAttributes) SetSourceTypeName(v string) {
	o.SourceTypeName = &v
}

// GetSourcecategory returns the Sourcecategory field value if set, zero value otherwise.
func (o *EventAttributes) GetSourcecategory() string {
	if o == nil || o.Sourcecategory == nil {
		var ret string
		return ret
	}
	return *o.Sourcecategory
}

// GetSourcecategoryOk returns a tuple with the Sourcecategory field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetSourcecategoryOk() (*string, bool) {
	if o == nil || o.Sourcecategory == nil {
		return nil, false
	}
	return o.Sourcecategory, true
}

// HasSourcecategory returns a boolean if a field has been set.
func (o *EventAttributes) HasSourcecategory() bool {
	return o != nil && o.Sourcecategory != nil
}

// SetSourcecategory gets a reference to the given string and assigns it to the Sourcecategory field.
func (o *EventAttributes) SetSourcecategory(v string) {
	o.Sourcecategory = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *EventAttributes) GetStatus() EventStatusType {
	if o == nil || o.Status == nil {
		var ret EventStatusType
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetStatusOk() (*EventStatusType, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *EventAttributes) HasStatus() bool {
	return o != nil && o.Status != nil
}

// SetStatus gets a reference to the given EventStatusType and assigns it to the Status field.
func (o *EventAttributes) SetStatus(v EventStatusType) {
	o.Status = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *EventAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *EventAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *EventAttributes) SetTags(v []string) {
	o.Tags = v
}

// GetTimestamp returns the Timestamp field value if set, zero value otherwise.
func (o *EventAttributes) GetTimestamp() int64 {
	if o == nil || o.Timestamp == nil {
		var ret int64
		return ret
	}
	return *o.Timestamp
}

// GetTimestampOk returns a tuple with the Timestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetTimestampOk() (*int64, bool) {
	if o == nil || o.Timestamp == nil {
		return nil, false
	}
	return o.Timestamp, true
}

// HasTimestamp returns a boolean if a field has been set.
func (o *EventAttributes) HasTimestamp() bool {
	return o != nil && o.Timestamp != nil
}

// SetTimestamp gets a reference to the given int64 and assigns it to the Timestamp field.
func (o *EventAttributes) SetTimestamp(v int64) {
	o.Timestamp = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *EventAttributes) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *EventAttributes) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *EventAttributes) HasTitle() bool {
	return o != nil && o.Title != nil
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *EventAttributes) SetTitle(v string) {
	o.Title = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o EventAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AggregationKey != nil {
		toSerialize["aggregation_key"] = o.AggregationKey
	}
	if o.DateHappened != nil {
		toSerialize["date_happened"] = o.DateHappened
	}
	if o.DeviceName != nil {
		toSerialize["device_name"] = o.DeviceName
	}
	if o.Duration != nil {
		toSerialize["duration"] = o.Duration
	}
	if o.EventObject != nil {
		toSerialize["event_object"] = o.EventObject
	}
	if o.Evt != nil {
		toSerialize["evt"] = o.Evt
	}
	if o.Hostname != nil {
		toSerialize["hostname"] = o.Hostname
	}
	if o.Monitor.IsSet() {
		toSerialize["monitor"] = o.Monitor.Get()
	}
	if o.MonitorGroups != nil {
		toSerialize["monitor_groups"] = o.MonitorGroups
	}
	if o.MonitorId.IsSet() {
		toSerialize["monitor_id"] = o.MonitorId.Get()
	}
	if o.Priority.IsSet() {
		toSerialize["priority"] = o.Priority.Get()
	}
	if o.RelatedEventId != nil {
		toSerialize["related_event_id"] = o.RelatedEventId
	}
	if o.Service != nil {
		toSerialize["service"] = o.Service
	}
	if o.SourceTypeName != nil {
		toSerialize["source_type_name"] = o.SourceTypeName
	}
	if o.Sourcecategory != nil {
		toSerialize["sourcecategory"] = o.Sourcecategory
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.Timestamp != nil {
		toSerialize["timestamp"] = o.Timestamp
	}
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *EventAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AggregationKey *string               `json:"aggregation_key,omitempty"`
		DateHappened   *int64                `json:"date_happened,omitempty"`
		DeviceName     *string               `json:"device_name,omitempty"`
		Duration       *int64                `json:"duration,omitempty"`
		EventObject    *string               `json:"event_object,omitempty"`
		Evt            *Event                `json:"evt,omitempty"`
		Hostname       *string               `json:"hostname,omitempty"`
		Monitor        NullableMonitorType   `json:"monitor,omitempty"`
		MonitorGroups  []string              `json:"monitor_groups,omitempty"`
		MonitorId      datadog.NullableInt64 `json:"monitor_id,omitempty"`
		Priority       NullableEventPriority `json:"priority,omitempty"`
		RelatedEventId *int64                `json:"related_event_id,omitempty"`
		Service        *string               `json:"service,omitempty"`
		SourceTypeName *string               `json:"source_type_name,omitempty"`
		Sourcecategory *string               `json:"sourcecategory,omitempty"`
		Status         *EventStatusType      `json:"status,omitempty"`
		Tags           []string              `json:"tags,omitempty"`
		Timestamp      *int64                `json:"timestamp,omitempty"`
		Title          *string               `json:"title,omitempty"`
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
	if v := all.Priority; v.Get() != nil && !v.Get().IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Status; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AggregationKey = all.AggregationKey
	o.DateHappened = all.DateHappened
	o.DeviceName = all.DeviceName
	o.Duration = all.Duration
	o.EventObject = all.EventObject
	if all.Evt != nil && all.Evt.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Evt = all.Evt
	o.Hostname = all.Hostname
	o.Monitor = all.Monitor
	o.MonitorGroups = all.MonitorGroups
	o.MonitorId = all.MonitorId
	o.Priority = all.Priority
	o.RelatedEventId = all.RelatedEventId
	o.Service = all.Service
	o.SourceTypeName = all.SourceTypeName
	o.Sourcecategory = all.Sourcecategory
	o.Status = all.Status
	o.Tags = all.Tags
	o.Timestamp = all.Timestamp
	o.Title = all.Title
	return nil
}
