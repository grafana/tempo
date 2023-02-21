// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsArchiveDestinationAzure The Azure archive destination.
type LogsArchiveDestinationAzure struct {
	// The container where the archive will be stored.
	Container string `json:"container"`
	// The Azure archive's integration destination.
	Integration LogsArchiveIntegrationAzure `json:"integration"`
	// The archive path.
	Path *string `json:"path,omitempty"`
	// The region where the archive will be stored.
	Region *string `json:"region,omitempty"`
	// The associated storage account.
	StorageAccount string `json:"storage_account"`
	// Type of the Azure archive destination.
	Type LogsArchiveDestinationAzureType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsArchiveDestinationAzure instantiates a new LogsArchiveDestinationAzure object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsArchiveDestinationAzure(container string, integration LogsArchiveIntegrationAzure, storageAccount string, typeVar LogsArchiveDestinationAzureType) *LogsArchiveDestinationAzure {
	this := LogsArchiveDestinationAzure{}
	this.Container = container
	this.Integration = integration
	this.StorageAccount = storageAccount
	this.Type = typeVar
	return &this
}

// NewLogsArchiveDestinationAzureWithDefaults instantiates a new LogsArchiveDestinationAzure object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsArchiveDestinationAzureWithDefaults() *LogsArchiveDestinationAzure {
	this := LogsArchiveDestinationAzure{}
	var typeVar LogsArchiveDestinationAzureType = LOGSARCHIVEDESTINATIONAZURETYPE_AZURE
	this.Type = typeVar
	return &this
}

// GetContainer returns the Container field value.
func (o *LogsArchiveDestinationAzure) GetContainer() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Container
}

// GetContainerOk returns a tuple with the Container field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationAzure) GetContainerOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Container, true
}

// SetContainer sets field value.
func (o *LogsArchiveDestinationAzure) SetContainer(v string) {
	o.Container = v
}

// GetIntegration returns the Integration field value.
func (o *LogsArchiveDestinationAzure) GetIntegration() LogsArchiveIntegrationAzure {
	if o == nil {
		var ret LogsArchiveIntegrationAzure
		return ret
	}
	return o.Integration
}

// GetIntegrationOk returns a tuple with the Integration field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationAzure) GetIntegrationOk() (*LogsArchiveIntegrationAzure, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Integration, true
}

// SetIntegration sets field value.
func (o *LogsArchiveDestinationAzure) SetIntegration(v LogsArchiveIntegrationAzure) {
	o.Integration = v
}

// GetPath returns the Path field value if set, zero value otherwise.
func (o *LogsArchiveDestinationAzure) GetPath() string {
	if o == nil || o.Path == nil {
		var ret string
		return ret
	}
	return *o.Path
}

// GetPathOk returns a tuple with the Path field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationAzure) GetPathOk() (*string, bool) {
	if o == nil || o.Path == nil {
		return nil, false
	}
	return o.Path, true
}

// HasPath returns a boolean if a field has been set.
func (o *LogsArchiveDestinationAzure) HasPath() bool {
	return o != nil && o.Path != nil
}

// SetPath gets a reference to the given string and assigns it to the Path field.
func (o *LogsArchiveDestinationAzure) SetPath(v string) {
	o.Path = &v
}

// GetRegion returns the Region field value if set, zero value otherwise.
func (o *LogsArchiveDestinationAzure) GetRegion() string {
	if o == nil || o.Region == nil {
		var ret string
		return ret
	}
	return *o.Region
}

// GetRegionOk returns a tuple with the Region field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationAzure) GetRegionOk() (*string, bool) {
	if o == nil || o.Region == nil {
		return nil, false
	}
	return o.Region, true
}

// HasRegion returns a boolean if a field has been set.
func (o *LogsArchiveDestinationAzure) HasRegion() bool {
	return o != nil && o.Region != nil
}

// SetRegion gets a reference to the given string and assigns it to the Region field.
func (o *LogsArchiveDestinationAzure) SetRegion(v string) {
	o.Region = &v
}

// GetStorageAccount returns the StorageAccount field value.
func (o *LogsArchiveDestinationAzure) GetStorageAccount() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.StorageAccount
}

// GetStorageAccountOk returns a tuple with the StorageAccount field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationAzure) GetStorageAccountOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.StorageAccount, true
}

// SetStorageAccount sets field value.
func (o *LogsArchiveDestinationAzure) SetStorageAccount(v string) {
	o.StorageAccount = v
}

// GetType returns the Type field value.
func (o *LogsArchiveDestinationAzure) GetType() LogsArchiveDestinationAzureType {
	if o == nil {
		var ret LogsArchiveDestinationAzureType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveDestinationAzure) GetTypeOk() (*LogsArchiveDestinationAzureType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *LogsArchiveDestinationAzure) SetType(v LogsArchiveDestinationAzureType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsArchiveDestinationAzure) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["container"] = o.Container
	toSerialize["integration"] = o.Integration
	if o.Path != nil {
		toSerialize["path"] = o.Path
	}
	if o.Region != nil {
		toSerialize["region"] = o.Region
	}
	toSerialize["storage_account"] = o.StorageAccount
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsArchiveDestinationAzure) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Container      *string                          `json:"container"`
		Integration    *LogsArchiveIntegrationAzure     `json:"integration"`
		StorageAccount *string                          `json:"storage_account"`
		Type           *LogsArchiveDestinationAzureType `json:"type"`
	}{}
	all := struct {
		Container      string                          `json:"container"`
		Integration    LogsArchiveIntegrationAzure     `json:"integration"`
		Path           *string                         `json:"path,omitempty"`
		Region         *string                         `json:"region,omitempty"`
		StorageAccount string                          `json:"storage_account"`
		Type           LogsArchiveDestinationAzureType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Container == nil {
		return fmt.Errorf("required field container missing")
	}
	if required.Integration == nil {
		return fmt.Errorf("required field integration missing")
	}
	if required.StorageAccount == nil {
		return fmt.Errorf("required field storage_account missing")
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
	o.Container = all.Container
	if all.Integration.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Integration = all.Integration
	o.Path = all.Path
	o.Region = all.Region
	o.StorageAccount = all.StorageAccount
	o.Type = all.Type
	return nil
}
