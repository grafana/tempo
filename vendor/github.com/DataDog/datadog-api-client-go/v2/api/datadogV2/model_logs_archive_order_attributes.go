// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsArchiveOrderAttributes The attributes associated with the archive order.
type LogsArchiveOrderAttributes struct {
	// An ordered array of `<ARCHIVE_ID>` strings, the order of archive IDs in the array
	// define the overall archives order for Datadog.
	ArchiveIds []string `json:"archive_ids"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsArchiveOrderAttributes instantiates a new LogsArchiveOrderAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsArchiveOrderAttributes(archiveIds []string) *LogsArchiveOrderAttributes {
	this := LogsArchiveOrderAttributes{}
	this.ArchiveIds = archiveIds
	return &this
}

// NewLogsArchiveOrderAttributesWithDefaults instantiates a new LogsArchiveOrderAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsArchiveOrderAttributesWithDefaults() *LogsArchiveOrderAttributes {
	this := LogsArchiveOrderAttributes{}
	return &this
}

// GetArchiveIds returns the ArchiveIds field value.
func (o *LogsArchiveOrderAttributes) GetArchiveIds() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.ArchiveIds
}

// GetArchiveIdsOk returns a tuple with the ArchiveIds field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveOrderAttributes) GetArchiveIdsOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ArchiveIds, true
}

// SetArchiveIds sets field value.
func (o *LogsArchiveOrderAttributes) SetArchiveIds(v []string) {
	o.ArchiveIds = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsArchiveOrderAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["archive_ids"] = o.ArchiveIds

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsArchiveOrderAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ArchiveIds *[]string `json:"archive_ids"`
	}{}
	all := struct {
		ArchiveIds []string `json:"archive_ids"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ArchiveIds == nil {
		return fmt.Errorf("required field archive_ids missing")
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
	o.ArchiveIds = all.ArchiveIds
	return nil
}
