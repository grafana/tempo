// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// OpsgenieServiceUpdateAttributes The Opsgenie service attributes for an update request.
type OpsgenieServiceUpdateAttributes struct {
	// The custom URL for a custom region.
	CustomUrl datadog.NullableString `json:"custom_url,omitempty"`
	// The name for the Opsgenie service.
	Name *string `json:"name,omitempty"`
	// The Opsgenie API key for your Opsgenie service.
	OpsgenieApiKey *string `json:"opsgenie_api_key,omitempty"`
	// The region for the Opsgenie service.
	Region *OpsgenieServiceRegionType `json:"region,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewOpsgenieServiceUpdateAttributes instantiates a new OpsgenieServiceUpdateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOpsgenieServiceUpdateAttributes() *OpsgenieServiceUpdateAttributes {
	this := OpsgenieServiceUpdateAttributes{}
	return &this
}

// NewOpsgenieServiceUpdateAttributesWithDefaults instantiates a new OpsgenieServiceUpdateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOpsgenieServiceUpdateAttributesWithDefaults() *OpsgenieServiceUpdateAttributes {
	this := OpsgenieServiceUpdateAttributes{}
	return &this
}

// GetCustomUrl returns the CustomUrl field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *OpsgenieServiceUpdateAttributes) GetCustomUrl() string {
	if o == nil || o.CustomUrl.Get() == nil {
		var ret string
		return ret
	}
	return *o.CustomUrl.Get()
}

// GetCustomUrlOk returns a tuple with the CustomUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *OpsgenieServiceUpdateAttributes) GetCustomUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.CustomUrl.Get(), o.CustomUrl.IsSet()
}

// HasCustomUrl returns a boolean if a field has been set.
func (o *OpsgenieServiceUpdateAttributes) HasCustomUrl() bool {
	return o != nil && o.CustomUrl.IsSet()
}

// SetCustomUrl gets a reference to the given datadog.NullableString and assigns it to the CustomUrl field.
func (o *OpsgenieServiceUpdateAttributes) SetCustomUrl(v string) {
	o.CustomUrl.Set(&v)
}

// SetCustomUrlNil sets the value for CustomUrl to be an explicit nil.
func (o *OpsgenieServiceUpdateAttributes) SetCustomUrlNil() {
	o.CustomUrl.Set(nil)
}

// UnsetCustomUrl ensures that no value is present for CustomUrl, not even an explicit nil.
func (o *OpsgenieServiceUpdateAttributes) UnsetCustomUrl() {
	o.CustomUrl.Unset()
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *OpsgenieServiceUpdateAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceUpdateAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *OpsgenieServiceUpdateAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *OpsgenieServiceUpdateAttributes) SetName(v string) {
	o.Name = &v
}

// GetOpsgenieApiKey returns the OpsgenieApiKey field value if set, zero value otherwise.
func (o *OpsgenieServiceUpdateAttributes) GetOpsgenieApiKey() string {
	if o == nil || o.OpsgenieApiKey == nil {
		var ret string
		return ret
	}
	return *o.OpsgenieApiKey
}

// GetOpsgenieApiKeyOk returns a tuple with the OpsgenieApiKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceUpdateAttributes) GetOpsgenieApiKeyOk() (*string, bool) {
	if o == nil || o.OpsgenieApiKey == nil {
		return nil, false
	}
	return o.OpsgenieApiKey, true
}

// HasOpsgenieApiKey returns a boolean if a field has been set.
func (o *OpsgenieServiceUpdateAttributes) HasOpsgenieApiKey() bool {
	return o != nil && o.OpsgenieApiKey != nil
}

// SetOpsgenieApiKey gets a reference to the given string and assigns it to the OpsgenieApiKey field.
func (o *OpsgenieServiceUpdateAttributes) SetOpsgenieApiKey(v string) {
	o.OpsgenieApiKey = &v
}

// GetRegion returns the Region field value if set, zero value otherwise.
func (o *OpsgenieServiceUpdateAttributes) GetRegion() OpsgenieServiceRegionType {
	if o == nil || o.Region == nil {
		var ret OpsgenieServiceRegionType
		return ret
	}
	return *o.Region
}

// GetRegionOk returns a tuple with the Region field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceUpdateAttributes) GetRegionOk() (*OpsgenieServiceRegionType, bool) {
	if o == nil || o.Region == nil {
		return nil, false
	}
	return o.Region, true
}

// HasRegion returns a boolean if a field has been set.
func (o *OpsgenieServiceUpdateAttributes) HasRegion() bool {
	return o != nil && o.Region != nil
}

// SetRegion gets a reference to the given OpsgenieServiceRegionType and assigns it to the Region field.
func (o *OpsgenieServiceUpdateAttributes) SetRegion(v OpsgenieServiceRegionType) {
	o.Region = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o OpsgenieServiceUpdateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CustomUrl.IsSet() {
		toSerialize["custom_url"] = o.CustomUrl.Get()
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.OpsgenieApiKey != nil {
		toSerialize["opsgenie_api_key"] = o.OpsgenieApiKey
	}
	if o.Region != nil {
		toSerialize["region"] = o.Region
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *OpsgenieServiceUpdateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CustomUrl      datadog.NullableString     `json:"custom_url,omitempty"`
		Name           *string                    `json:"name,omitempty"`
		OpsgenieApiKey *string                    `json:"opsgenie_api_key,omitempty"`
		Region         *OpsgenieServiceRegionType `json:"region,omitempty"`
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
	if v := all.Region; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.CustomUrl = all.CustomUrl
	o.Name = all.Name
	o.OpsgenieApiKey = all.OpsgenieApiKey
	o.Region = all.Region
	return nil
}
