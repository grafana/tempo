// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionMeta Metadata about a service definition.
type ServiceDefinitionMeta struct {
	// GitHub HTML URL.
	GithubHtmlUrl *string `json:"github-html-url,omitempty"`
	// Ingestion schema version.
	IngestedSchemaVersion *string `json:"ingested-schema-version,omitempty"`
	// Ingestion source of the service definition.
	IngestionSource *string `json:"ingestion-source,omitempty"`
	// Last modified time of the service definition.
	LastModifiedTime *string `json:"last-modified-time,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionMeta instantiates a new ServiceDefinitionMeta object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionMeta() *ServiceDefinitionMeta {
	this := ServiceDefinitionMeta{}
	return &this
}

// NewServiceDefinitionMetaWithDefaults instantiates a new ServiceDefinitionMeta object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionMetaWithDefaults() *ServiceDefinitionMeta {
	this := ServiceDefinitionMeta{}
	return &this
}

// GetGithubHtmlUrl returns the GithubHtmlUrl field value if set, zero value otherwise.
func (o *ServiceDefinitionMeta) GetGithubHtmlUrl() string {
	if o == nil || o.GithubHtmlUrl == nil {
		var ret string
		return ret
	}
	return *o.GithubHtmlUrl
}

// GetGithubHtmlUrlOk returns a tuple with the GithubHtmlUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionMeta) GetGithubHtmlUrlOk() (*string, bool) {
	if o == nil || o.GithubHtmlUrl == nil {
		return nil, false
	}
	return o.GithubHtmlUrl, true
}

// HasGithubHtmlUrl returns a boolean if a field has been set.
func (o *ServiceDefinitionMeta) HasGithubHtmlUrl() bool {
	return o != nil && o.GithubHtmlUrl != nil
}

// SetGithubHtmlUrl gets a reference to the given string and assigns it to the GithubHtmlUrl field.
func (o *ServiceDefinitionMeta) SetGithubHtmlUrl(v string) {
	o.GithubHtmlUrl = &v
}

// GetIngestedSchemaVersion returns the IngestedSchemaVersion field value if set, zero value otherwise.
func (o *ServiceDefinitionMeta) GetIngestedSchemaVersion() string {
	if o == nil || o.IngestedSchemaVersion == nil {
		var ret string
		return ret
	}
	return *o.IngestedSchemaVersion
}

// GetIngestedSchemaVersionOk returns a tuple with the IngestedSchemaVersion field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionMeta) GetIngestedSchemaVersionOk() (*string, bool) {
	if o == nil || o.IngestedSchemaVersion == nil {
		return nil, false
	}
	return o.IngestedSchemaVersion, true
}

// HasIngestedSchemaVersion returns a boolean if a field has been set.
func (o *ServiceDefinitionMeta) HasIngestedSchemaVersion() bool {
	return o != nil && o.IngestedSchemaVersion != nil
}

// SetIngestedSchemaVersion gets a reference to the given string and assigns it to the IngestedSchemaVersion field.
func (o *ServiceDefinitionMeta) SetIngestedSchemaVersion(v string) {
	o.IngestedSchemaVersion = &v
}

// GetIngestionSource returns the IngestionSource field value if set, zero value otherwise.
func (o *ServiceDefinitionMeta) GetIngestionSource() string {
	if o == nil || o.IngestionSource == nil {
		var ret string
		return ret
	}
	return *o.IngestionSource
}

// GetIngestionSourceOk returns a tuple with the IngestionSource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionMeta) GetIngestionSourceOk() (*string, bool) {
	if o == nil || o.IngestionSource == nil {
		return nil, false
	}
	return o.IngestionSource, true
}

// HasIngestionSource returns a boolean if a field has been set.
func (o *ServiceDefinitionMeta) HasIngestionSource() bool {
	return o != nil && o.IngestionSource != nil
}

// SetIngestionSource gets a reference to the given string and assigns it to the IngestionSource field.
func (o *ServiceDefinitionMeta) SetIngestionSource(v string) {
	o.IngestionSource = &v
}

// GetLastModifiedTime returns the LastModifiedTime field value if set, zero value otherwise.
func (o *ServiceDefinitionMeta) GetLastModifiedTime() string {
	if o == nil || o.LastModifiedTime == nil {
		var ret string
		return ret
	}
	return *o.LastModifiedTime
}

// GetLastModifiedTimeOk returns a tuple with the LastModifiedTime field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionMeta) GetLastModifiedTimeOk() (*string, bool) {
	if o == nil || o.LastModifiedTime == nil {
		return nil, false
	}
	return o.LastModifiedTime, true
}

// HasLastModifiedTime returns a boolean if a field has been set.
func (o *ServiceDefinitionMeta) HasLastModifiedTime() bool {
	return o != nil && o.LastModifiedTime != nil
}

// SetLastModifiedTime gets a reference to the given string and assigns it to the LastModifiedTime field.
func (o *ServiceDefinitionMeta) SetLastModifiedTime(v string) {
	o.LastModifiedTime = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionMeta) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.GithubHtmlUrl != nil {
		toSerialize["github-html-url"] = o.GithubHtmlUrl
	}
	if o.IngestedSchemaVersion != nil {
		toSerialize["ingested-schema-version"] = o.IngestedSchemaVersion
	}
	if o.IngestionSource != nil {
		toSerialize["ingestion-source"] = o.IngestionSource
	}
	if o.LastModifiedTime != nil {
		toSerialize["last-modified-time"] = o.LastModifiedTime
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionMeta) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		GithubHtmlUrl         *string `json:"github-html-url,omitempty"`
		IngestedSchemaVersion *string `json:"ingested-schema-version,omitempty"`
		IngestionSource       *string `json:"ingestion-source,omitempty"`
		LastModifiedTime      *string `json:"last-modified-time,omitempty"`
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
	o.GithubHtmlUrl = all.GithubHtmlUrl
	o.IngestedSchemaVersion = all.IngestedSchemaVersion
	o.IngestionSource = all.IngestionSource
	o.LastModifiedTime = all.LastModifiedTime
	return nil
}
