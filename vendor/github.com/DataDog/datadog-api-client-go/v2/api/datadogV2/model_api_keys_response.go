// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// APIKeysResponse Response for a list of API keys.
type APIKeysResponse struct {
	// Array of API keys.
	Data []PartialAPIKey `json:"data,omitempty"`
	// Array of objects related to the API key.
	Included []APIKeyResponseIncludedItem `json:"included,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewAPIKeysResponse instantiates a new APIKeysResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewAPIKeysResponse() *APIKeysResponse {
	this := APIKeysResponse{}
	return &this
}

// NewAPIKeysResponseWithDefaults instantiates a new APIKeysResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewAPIKeysResponseWithDefaults() *APIKeysResponse {
	this := APIKeysResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *APIKeysResponse) GetData() []PartialAPIKey {
	if o == nil || o.Data == nil {
		var ret []PartialAPIKey
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *APIKeysResponse) GetDataOk() (*[]PartialAPIKey, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *APIKeysResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given []PartialAPIKey and assigns it to the Data field.
func (o *APIKeysResponse) SetData(v []PartialAPIKey) {
	o.Data = v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *APIKeysResponse) GetIncluded() []APIKeyResponseIncludedItem {
	if o == nil || o.Included == nil {
		var ret []APIKeyResponseIncludedItem
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *APIKeysResponse) GetIncludedOk() (*[]APIKeyResponseIncludedItem, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *APIKeysResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []APIKeyResponseIncludedItem and assigns it to the Included field.
func (o *APIKeysResponse) SetIncluded(v []APIKeyResponseIncludedItem) {
	o.Included = v
}

// MarshalJSON serializes the struct using spec logic.
func (o APIKeysResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}
	if o.Included != nil {
		toSerialize["included"] = o.Included
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *APIKeysResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data     []PartialAPIKey              `json:"data,omitempty"`
		Included []APIKeyResponseIncludedItem `json:"included,omitempty"`
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
	o.Data = all.Data
	o.Included = all.Included
	return nil
}
