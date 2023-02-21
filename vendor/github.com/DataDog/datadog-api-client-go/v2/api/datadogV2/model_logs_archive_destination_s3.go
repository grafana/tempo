// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsArchiveDestinationS3 The S3 archive destination.
type LogsArchiveDestinationS3 struct {
	// The bucket where the archive will be stored.
	Bucket string `json:"bucket"`
	// The S3 Archive's integration destination.
	Integration LogsArchiveIntegrationS3 `json:"integration"`
	// The archive path.
	Path *string `json:"path,omitempty"`
	// Type of the S3 archive destination.
	Type LogsArchiveDestinationS3Type `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsArchiveDestinationS3 instantiates a new LogsArchiveDestinationS3 object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsArchiveDestinationS3(bucket string, integration LogsArchiveIntegrationS3, typeVar LogsArchiveDestinationS3Type) *LogsArchiveDestinationS3 {
	this := LogsArchiveDestinationS3{}
	this.Bucket = bucket
	this.Integration = integration
	this.Type = typeVar
	return &this
}

// NewLogsArchiveDestinationS3WithDefaults instantiates a new LogsArchiveDestinationS3 object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsArchiveDestinationS3WithDefaults() *LogsArchiveDestinationS3 {
	this := LogsArchiveDestinationS3{}
	var typeVar LogsArchiveDestinationS3Type = LOGSARCHIVEDESTINATIONS3TYPE_S3
	this.Type = typeVar
	return &this
}

// GetBucket returns the Bucket field value.
func (o *LogsArchiveDestinationS3) GetBucket() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Bucket
}

// GetBucketOk returns a tuple with the Bucket field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationS3) GetBucketOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Bucket, true
}

// SetBucket sets field value.
func (o *LogsArchiveDestinationS3) SetBucket(v string) {
	o.Bucket = v
}

// GetIntegration returns the Integration field value.
func (o *LogsArchiveDestinationS3) GetIntegration() LogsArchiveIntegrationS3 {
	if o == nil {
		var ret LogsArchiveIntegrationS3
		return ret
	}
	return o.Integration
}

// GetIntegrationOk returns a tuple with the Integration field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationS3) GetIntegrationOk() (*LogsArchiveIntegrationS3, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Integration, true
}

// SetIntegration sets field value.
func (o *LogsArchiveDestinationS3) SetIntegration(v LogsArchiveIntegrationS3) {
	o.Integration = v
}

// GetPath returns the Path field value if set, zero value otherwise.
func (o *LogsArchiveDestinationS3) GetPath() string {
	if o == nil || o.Path == nil {
		var ret string
		return ret
	}
	return *o.Path
}

// GetPathOk returns a tuple with the Path field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationS3) GetPathOk() (*string, bool) {
	if o == nil || o.Path == nil {
		return nil, false
	}
	return o.Path, true
}

// HasPath returns a boolean if a field has been set.
func (o *LogsArchiveDestinationS3) HasPath() bool {
	return o != nil && o.Path != nil
}

// SetPath gets a reference to the given string and assigns it to the Path field.
func (o *LogsArchiveDestinationS3) SetPath(v string) {
	o.Path = &v
}

// GetType returns the Type field value.
func (o *LogsArchiveDestinationS3) GetType() LogsArchiveDestinationS3Type {
	if o == nil {
		var ret LogsArchiveDestinationS3Type
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationS3) GetTypeOk() (*LogsArchiveDestinationS3Type, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *LogsArchiveDestinationS3) SetType(v LogsArchiveDestinationS3Type) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsArchiveDestinationS3) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["bucket"] = o.Bucket
	toSerialize["integration"] = o.Integration
	if o.Path != nil {
		toSerialize["path"] = o.Path
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsArchiveDestinationS3) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Bucket      *string                       `json:"bucket"`
		Integration *LogsArchiveIntegrationS3     `json:"integration"`
		Type        *LogsArchiveDestinationS3Type `json:"type"`
	}{}
	all := struct {
		Bucket      string                       `json:"bucket"`
		Integration LogsArchiveIntegrationS3     `json:"integration"`
		Path        *string                      `json:"path,omitempty"`
		Type        LogsArchiveDestinationS3Type `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Bucket == nil {
		return fmt.Errorf("required field bucket missing")
	}
	if required.Integration == nil {
		return fmt.Errorf("required field integration missing")
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
	o.Bucket = all.Bucket
	if all.Integration.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Integration = all.Integration
	o.Path = all.Path
	o.Type = all.Type
	return nil
}
