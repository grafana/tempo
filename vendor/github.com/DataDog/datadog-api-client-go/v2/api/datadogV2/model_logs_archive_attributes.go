// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// LogsArchiveAttributes The attributes associated with the archive.
type LogsArchiveAttributes struct {
	// An archive's destination.
	Destination NullableLogsArchiveDestination `json:"destination"`
	// To store the tags in the archive, set the value "true".
	// If it is set to "false", the tags will be deleted when the logs are sent to the archive.
	IncludeTags *bool `json:"include_tags,omitempty"`
	// The archive name.
	Name string `json:"name"`
	// The archive query/filter. Logs matching this query are included in the archive.
	Query string `json:"query"`
	// Maximum scan size for rehydration from this archive.
	RehydrationMaxScanSizeInGb datadog.NullableInt64 `json:"rehydration_max_scan_size_in_gb,omitempty"`
	// An array of tags to add to rehydrated logs from an archive.
	RehydrationTags []string `json:"rehydration_tags,omitempty"`
	// The state of the archive.
	State *LogsArchiveState `json:"state,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsArchiveAttributes instantiates a new LogsArchiveAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsArchiveAttributes(destination NullableLogsArchiveDestination, name string, query string) *LogsArchiveAttributes {
	this := LogsArchiveAttributes{}
	this.Destination = destination
	var includeTags bool = false
	this.IncludeTags = &includeTags
	this.Name = name
	this.Query = query
	return &this
}

// NewLogsArchiveAttributesWithDefaults instantiates a new LogsArchiveAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsArchiveAttributesWithDefaults() *LogsArchiveAttributes {
	this := LogsArchiveAttributes{}
	var includeTags bool = false
	this.IncludeTags = &includeTags
	return &this
}

// GetDestination returns the Destination field value.
// If the value is explicit nil, the zero value for LogsArchiveDestination will be returned.
func (o *LogsArchiveAttributes) GetDestination() LogsArchiveDestination {
	if o == nil || o.Destination.Get() == nil {
		var ret LogsArchiveDestination
		return ret
	}
	return *o.Destination.Get()
}

// GetDestinationOk returns a tuple with the Destination field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *LogsArchiveAttributes) GetDestinationOk() (*LogsArchiveDestination, bool) {
	if o == nil {
		return nil, false
	}
	return o.Destination.Get(), o.Destination.IsSet()
}

// SetDestination sets field value.
func (o *LogsArchiveAttributes) SetDestination(v LogsArchiveDestination) {
	o.Destination.Set(&v)
}

// GetIncludeTags returns the IncludeTags field value if set, zero value otherwise.
func (o *LogsArchiveAttributes) GetIncludeTags() bool {
	if o == nil || o.IncludeTags == nil {
		var ret bool
		return ret
	}
	return *o.IncludeTags
}

// GetIncludeTagsOk returns a tuple with the IncludeTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsArchiveAttributes) GetIncludeTagsOk() (*bool, bool) {
	if o == nil || o.IncludeTags == nil {
		return nil, false
	}
	return o.IncludeTags, true
}

// HasIncludeTags returns a boolean if a field has been set.
func (o *LogsArchiveAttributes) HasIncludeTags() bool {
	return o != nil && o.IncludeTags != nil
}

// SetIncludeTags gets a reference to the given bool and assigns it to the IncludeTags field.
func (o *LogsArchiveAttributes) SetIncludeTags(v bool) {
	o.IncludeTags = &v
}

// GetName returns the Name field value.
func (o *LogsArchiveAttributes) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveAttributes) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *LogsArchiveAttributes) SetName(v string) {
	o.Name = v
}

// GetQuery returns the Query field value.
func (o *LogsArchiveAttributes) GetQuery() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Query
}

// GetQueryOk returns a tuple with the Query field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveAttributes) GetQueryOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Query, true
}

// SetQuery sets field value.
func (o *LogsArchiveAttributes) SetQuery(v string) {
	o.Query = v
}

// GetRehydrationMaxScanSizeInGb returns the RehydrationMaxScanSizeInGb field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *LogsArchiveAttributes) GetRehydrationMaxScanSizeInGb() int64 {
	if o == nil || o.RehydrationMaxScanSizeInGb.Get() == nil {
		var ret int64
		return ret
	}
	return *o.RehydrationMaxScanSizeInGb.Get()
}

// GetRehydrationMaxScanSizeInGbOk returns a tuple with the RehydrationMaxScanSizeInGb field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *LogsArchiveAttributes) GetRehydrationMaxScanSizeInGbOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.RehydrationMaxScanSizeInGb.Get(), o.RehydrationMaxScanSizeInGb.IsSet()
}

// HasRehydrationMaxScanSizeInGb returns a boolean if a field has been set.
func (o *LogsArchiveAttributes) HasRehydrationMaxScanSizeInGb() bool {
	return o != nil && o.RehydrationMaxScanSizeInGb.IsSet()
}

// SetRehydrationMaxScanSizeInGb gets a reference to the given datadog.NullableInt64 and assigns it to the RehydrationMaxScanSizeInGb field.
func (o *LogsArchiveAttributes) SetRehydrationMaxScanSizeInGb(v int64) {
	o.RehydrationMaxScanSizeInGb.Set(&v)
}

// SetRehydrationMaxScanSizeInGbNil sets the value for RehydrationMaxScanSizeInGb to be an explicit nil.
func (o *LogsArchiveAttributes) SetRehydrationMaxScanSizeInGbNil() {
	o.RehydrationMaxScanSizeInGb.Set(nil)
}

// UnsetRehydrationMaxScanSizeInGb ensures that no value is present for RehydrationMaxScanSizeInGb, not even an explicit nil.
func (o *LogsArchiveAttributes) UnsetRehydrationMaxScanSizeInGb() {
	o.RehydrationMaxScanSizeInGb.Unset()
}

// GetRehydrationTags returns the RehydrationTags field value if set, zero value otherwise.
func (o *LogsArchiveAttributes) GetRehydrationTags() []string {
	if o == nil || o.RehydrationTags == nil {
		var ret []string
		return ret
	}
	return o.RehydrationTags
}

// GetRehydrationTagsOk returns a tuple with the RehydrationTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsArchiveAttributes) GetRehydrationTagsOk() (*[]string, bool) {
	if o == nil || o.RehydrationTags == nil {
		return nil, false
	}
	return &o.RehydrationTags, true
}

// HasRehydrationTags returns a boolean if a field has been set.
func (o *LogsArchiveAttributes) HasRehydrationTags() bool {
	return o != nil && o.RehydrationTags != nil
}

// SetRehydrationTags gets a reference to the given []string and assigns it to the RehydrationTags field.
func (o *LogsArchiveAttributes) SetRehydrationTags(v []string) {
	o.RehydrationTags = v
}

// GetState returns the State field value if set, zero value otherwise.
func (o *LogsArchiveAttributes) GetState() LogsArchiveState {
	if o == nil || o.State == nil {
		var ret LogsArchiveState
		return ret
	}
	return *o.State
}

// GetStateOk returns a tuple with the State field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsArchiveAttributes) GetStateOk() (*LogsArchiveState, bool) {
	if o == nil || o.State == nil {
		return nil, false
	}
	return o.State, true
}

// HasState returns a boolean if a field has been set.
func (o *LogsArchiveAttributes) HasState() bool {
	return o != nil && o.State != nil
}

// SetState gets a reference to the given LogsArchiveState and assigns it to the State field.
func (o *LogsArchiveAttributes) SetState(v LogsArchiveState) {
	o.State = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsArchiveAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["destination"] = o.Destination.Get()
	if o.IncludeTags != nil {
		toSerialize["include_tags"] = o.IncludeTags
	}
	toSerialize["name"] = o.Name
	toSerialize["query"] = o.Query
	if o.RehydrationMaxScanSizeInGb.IsSet() {
		toSerialize["rehydration_max_scan_size_in_gb"] = o.RehydrationMaxScanSizeInGb.Get()
	}
	if o.RehydrationTags != nil {
		toSerialize["rehydration_tags"] = o.RehydrationTags
	}
	if o.State != nil {
		toSerialize["state"] = o.State
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsArchiveAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Destination NullableLogsArchiveDestination `json:"destination"`
		Name        *string                        `json:"name"`
		Query       *string                        `json:"query"`
	}{}
	all := struct {
		Destination                NullableLogsArchiveDestination `json:"destination"`
		IncludeTags                *bool                          `json:"include_tags,omitempty"`
		Name                       string                         `json:"name"`
		Query                      string                         `json:"query"`
		RehydrationMaxScanSizeInGb datadog.NullableInt64          `json:"rehydration_max_scan_size_in_gb,omitempty"`
		RehydrationTags            []string                       `json:"rehydration_tags,omitempty"`
		State                      *LogsArchiveState              `json:"state,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if !required.Destination.IsSet() {
		return fmt.Errorf("required field destination missing")
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
	}
	if required.Query == nil {
		return fmt.Errorf("required field query missing")
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
	if v := all.State; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Destination = all.Destination
	o.IncludeTags = all.IncludeTags
	o.Name = all.Name
	o.Query = all.Query
	o.RehydrationMaxScanSizeInGb = all.RehydrationMaxScanSizeInGb
	o.RehydrationTags = all.RehydrationTags
	o.State = all.State
	return nil
}
