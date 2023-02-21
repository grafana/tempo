// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ProcessSummaryAttributes Attributes for a process summary.
type ProcessSummaryAttributes struct {
	// Process command line.
	Cmdline *string `json:"cmdline,omitempty"`
	// Host running the process.
	Host *string `json:"host,omitempty"`
	// Process ID.
	Pid *int64 `json:"pid,omitempty"`
	// Parent process ID.
	Ppid *int64 `json:"ppid,omitempty"`
	// Time the process was started.
	Start *string `json:"start,omitempty"`
	// List of tags associated with the process.
	Tags []string `json:"tags,omitempty"`
	// Time the process was seen.
	Timestamp *string `json:"timestamp,omitempty"`
	// Process owner.
	User *string `json:"user,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewProcessSummaryAttributes instantiates a new ProcessSummaryAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewProcessSummaryAttributes() *ProcessSummaryAttributes {
	this := ProcessSummaryAttributes{}
	return &this
}

// NewProcessSummaryAttributesWithDefaults instantiates a new ProcessSummaryAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewProcessSummaryAttributesWithDefaults() *ProcessSummaryAttributes {
	this := ProcessSummaryAttributes{}
	return &this
}

// GetCmdline returns the Cmdline field value if set, zero value otherwise.
func (o *ProcessSummaryAttributes) GetCmdline() string {
	if o == nil || o.Cmdline == nil {
		var ret string
		return ret
	}
	return *o.Cmdline
}

// GetCmdlineOk returns a tuple with the Cmdline field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummaryAttributes) GetCmdlineOk() (*string, bool) {
	if o == nil || o.Cmdline == nil {
		return nil, false
	}
	return o.Cmdline, true
}

// HasCmdline returns a boolean if a field has been set.
func (o *ProcessSummaryAttributes) HasCmdline() bool {
	return o != nil && o.Cmdline != nil
}

// SetCmdline gets a reference to the given string and assigns it to the Cmdline field.
func (o *ProcessSummaryAttributes) SetCmdline(v string) {
	o.Cmdline = &v
}

// GetHost returns the Host field value if set, zero value otherwise.
func (o *ProcessSummaryAttributes) GetHost() string {
	if o == nil || o.Host == nil {
		var ret string
		return ret
	}
	return *o.Host
}

// GetHostOk returns a tuple with the Host field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummaryAttributes) GetHostOk() (*string, bool) {
	if o == nil || o.Host == nil {
		return nil, false
	}
	return o.Host, true
}

// HasHost returns a boolean if a field has been set.
func (o *ProcessSummaryAttributes) HasHost() bool {
	return o != nil && o.Host != nil
}

// SetHost gets a reference to the given string and assigns it to the Host field.
func (o *ProcessSummaryAttributes) SetHost(v string) {
	o.Host = &v
}

// GetPid returns the Pid field value if set, zero value otherwise.
func (o *ProcessSummaryAttributes) GetPid() int64 {
	if o == nil || o.Pid == nil {
		var ret int64
		return ret
	}
	return *o.Pid
}

// GetPidOk returns a tuple with the Pid field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummaryAttributes) GetPidOk() (*int64, bool) {
	if o == nil || o.Pid == nil {
		return nil, false
	}
	return o.Pid, true
}

// HasPid returns a boolean if a field has been set.
func (o *ProcessSummaryAttributes) HasPid() bool {
	return o != nil && o.Pid != nil
}

// SetPid gets a reference to the given int64 and assigns it to the Pid field.
func (o *ProcessSummaryAttributes) SetPid(v int64) {
	o.Pid = &v
}

// GetPpid returns the Ppid field value if set, zero value otherwise.
func (o *ProcessSummaryAttributes) GetPpid() int64 {
	if o == nil || o.Ppid == nil {
		var ret int64
		return ret
	}
	return *o.Ppid
}

// GetPpidOk returns a tuple with the Ppid field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummaryAttributes) GetPpidOk() (*int64, bool) {
	if o == nil || o.Ppid == nil {
		return nil, false
	}
	return o.Ppid, true
}

// HasPpid returns a boolean if a field has been set.
func (o *ProcessSummaryAttributes) HasPpid() bool {
	return o != nil && o.Ppid != nil
}

// SetPpid gets a reference to the given int64 and assigns it to the Ppid field.
func (o *ProcessSummaryAttributes) SetPpid(v int64) {
	o.Ppid = &v
}

// GetStart returns the Start field value if set, zero value otherwise.
func (o *ProcessSummaryAttributes) GetStart() string {
	if o == nil || o.Start == nil {
		var ret string
		return ret
	}
	return *o.Start
}

// GetStartOk returns a tuple with the Start field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummaryAttributes) GetStartOk() (*string, bool) {
	if o == nil || o.Start == nil {
		return nil, false
	}
	return o.Start, true
}

// HasStart returns a boolean if a field has been set.
func (o *ProcessSummaryAttributes) HasStart() bool {
	return o != nil && o.Start != nil
}

// SetStart gets a reference to the given string and assigns it to the Start field.
func (o *ProcessSummaryAttributes) SetStart(v string) {
	o.Start = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *ProcessSummaryAttributes) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummaryAttributes) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *ProcessSummaryAttributes) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *ProcessSummaryAttributes) SetTags(v []string) {
	o.Tags = v
}

// GetTimestamp returns the Timestamp field value if set, zero value otherwise.
func (o *ProcessSummaryAttributes) GetTimestamp() string {
	if o == nil || o.Timestamp == nil {
		var ret string
		return ret
	}
	return *o.Timestamp
}

// GetTimestampOk returns a tuple with the Timestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummaryAttributes) GetTimestampOk() (*string, bool) {
	if o == nil || o.Timestamp == nil {
		return nil, false
	}
	return o.Timestamp, true
}

// HasTimestamp returns a boolean if a field has been set.
func (o *ProcessSummaryAttributes) HasTimestamp() bool {
	return o != nil && o.Timestamp != nil
}

// SetTimestamp gets a reference to the given string and assigns it to the Timestamp field.
func (o *ProcessSummaryAttributes) SetTimestamp(v string) {
	o.Timestamp = &v
}

// GetUser returns the User field value if set, zero value otherwise.
func (o *ProcessSummaryAttributes) GetUser() string {
	if o == nil || o.User == nil {
		var ret string
		return ret
	}
	return *o.User
}

// GetUserOk returns a tuple with the User field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummaryAttributes) GetUserOk() (*string, bool) {
	if o == nil || o.User == nil {
		return nil, false
	}
	return o.User, true
}

// HasUser returns a boolean if a field has been set.
func (o *ProcessSummaryAttributes) HasUser() bool {
	return o != nil && o.User != nil
}

// SetUser gets a reference to the given string and assigns it to the User field.
func (o *ProcessSummaryAttributes) SetUser(v string) {
	o.User = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ProcessSummaryAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Cmdline != nil {
		toSerialize["cmdline"] = o.Cmdline
	}
	if o.Host != nil {
		toSerialize["host"] = o.Host
	}
	if o.Pid != nil {
		toSerialize["pid"] = o.Pid
	}
	if o.Ppid != nil {
		toSerialize["ppid"] = o.Ppid
	}
	if o.Start != nil {
		toSerialize["start"] = o.Start
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.Timestamp != nil {
		toSerialize["timestamp"] = o.Timestamp
	}
	if o.User != nil {
		toSerialize["user"] = o.User
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ProcessSummaryAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Cmdline   *string  `json:"cmdline,omitempty"`
		Host      *string  `json:"host,omitempty"`
		Pid       *int64   `json:"pid,omitempty"`
		Ppid      *int64   `json:"ppid,omitempty"`
		Start     *string  `json:"start,omitempty"`
		Tags      []string `json:"tags,omitempty"`
		Timestamp *string  `json:"timestamp,omitempty"`
		User      *string  `json:"user,omitempty"`
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
	o.Cmdline = all.Cmdline
	o.Host = all.Host
	o.Pid = all.Pid
	o.Ppid = all.Ppid
	o.Start = all.Start
	o.Tags = all.Tags
	o.Timestamp = all.Timestamp
	o.User = all.User
	return nil
}
