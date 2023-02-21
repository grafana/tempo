// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// OpsgenieServiceCreateAttributes The Opsgenie service attributes for a create request.
type OpsgenieServiceCreateAttributes struct {
	// The custom URL for a custom region.
	CustomUrl *string `json:"custom_url,omitempty"`
	// The name for the Opsgenie service.
	Name string `json:"name"`
	// The Opsgenie API key for your Opsgenie service.
	OpsgenieApiKey string `json:"opsgenie_api_key"`
	// The region for the Opsgenie service.
	Region OpsgenieServiceRegionType `json:"region"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewOpsgenieServiceCreateAttributes instantiates a new OpsgenieServiceCreateAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewOpsgenieServiceCreateAttributes(name string, opsgenieApiKey string, region OpsgenieServiceRegionType) *OpsgenieServiceCreateAttributes {
	this := OpsgenieServiceCreateAttributes{}
	this.Name = name
	this.OpsgenieApiKey = opsgenieApiKey
	this.Region = region
	return &this
}

// NewOpsgenieServiceCreateAttributesWithDefaults instantiates a new OpsgenieServiceCreateAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewOpsgenieServiceCreateAttributesWithDefaults() *OpsgenieServiceCreateAttributes {
	this := OpsgenieServiceCreateAttributes{}
	return &this
}

// GetCustomUrl returns the CustomUrl field value if set, zero value otherwise.
func (o *OpsgenieServiceCreateAttributes) GetCustomUrl() string {
	if o == nil || o.CustomUrl == nil {
		var ret string
		return ret
	}
	return *o.CustomUrl
}

// GetCustomUrlOk returns a tuple with the CustomUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceCreateAttributes) GetCustomUrlOk() (*string, bool) {
	if o == nil || o.CustomUrl == nil {
		return nil, false
	}
	return o.CustomUrl, true
}

// HasCustomUrl returns a boolean if a field has been set.
func (o *OpsgenieServiceCreateAttributes) HasCustomUrl() bool {
	return o != nil && o.CustomUrl != nil
}

// SetCustomUrl gets a reference to the given string and assigns it to the CustomUrl field.
func (o *OpsgenieServiceCreateAttributes) SetCustomUrl(v string) {
	o.CustomUrl = &v
}

// GetName returns the Name field value.
func (o *OpsgenieServiceCreateAttributes) GetName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceCreateAttributes) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value.
func (o *OpsgenieServiceCreateAttributes) SetName(v string) {
	o.Name = v
}

// GetOpsgenieApiKey returns the OpsgenieApiKey field value.
func (o *OpsgenieServiceCreateAttributes) GetOpsgenieApiKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.OpsgenieApiKey
}

// GetOpsgenieApiKeyOk returns a tuple with the OpsgenieApiKey field value
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceCreateAttributes) GetOpsgenieApiKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.OpsgenieApiKey, true
}

// SetOpsgenieApiKey sets field value.
func (o *OpsgenieServiceCreateAttributes) SetOpsgenieApiKey(v string) {
	o.OpsgenieApiKey = v
}

// GetRegion returns the Region field value.
func (o *OpsgenieServiceCreateAttributes) GetRegion() OpsgenieServiceRegionType {
	if o == nil {
		var ret OpsgenieServiceRegionType
		return ret
	}
	return o.Region
}

// GetRegionOk returns a tuple with the Region field value
// and a boolean to check if the value has been set.
func (o *OpsgenieServiceCreateAttributes) GetRegionOk() (*OpsgenieServiceRegionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Region, true
}

// SetRegion sets field value.
func (o *OpsgenieServiceCreateAttributes) SetRegion(v OpsgenieServiceRegionType) {
	o.Region = v
}

// MarshalJSON serializes the struct using spec logic.
func (o OpsgenieServiceCreateAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CustomUrl != nil {
		toSerialize["custom_url"] = o.CustomUrl
	}
	toSerialize["name"] = o.Name
	toSerialize["opsgenie_api_key"] = o.OpsgenieApiKey
	toSerialize["region"] = o.Region

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *OpsgenieServiceCreateAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Name           *string                    `json:"name"`
		OpsgenieApiKey *string                    `json:"opsgenie_api_key"`
		Region         *OpsgenieServiceRegionType `json:"region"`
	}{}
	all := struct {
		CustomUrl      *string                   `json:"custom_url,omitempty"`
		Name           string                    `json:"name"`
		OpsgenieApiKey string                    `json:"opsgenie_api_key"`
		Region         OpsgenieServiceRegionType `json:"region"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Name == nil {
		return fmt.Errorf("required field name missing")
	}
	if required.OpsgenieApiKey == nil {
		return fmt.Errorf("required field opsgenie_api_key missing")
	}
	if required.Region == nil {
		return fmt.Errorf("required field region missing")
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
	if v := all.Region; !v.IsValid() {
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
