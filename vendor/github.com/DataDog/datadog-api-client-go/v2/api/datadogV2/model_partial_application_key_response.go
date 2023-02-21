// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// PartialApplicationKeyResponse Response for retrieving a partial application key.
type PartialApplicationKeyResponse struct {
	// Partial Datadog application key.
	Data *PartialApplicationKey `json:"data,omitempty"`
	// Array of objects related to the application key.
	Included []ApplicationKeyResponseIncludedItem `json:"included,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewPartialApplicationKeyResponse instantiates a new PartialApplicationKeyResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewPartialApplicationKeyResponse() *PartialApplicationKeyResponse {
	this := PartialApplicationKeyResponse{}
	return &this
}

// NewPartialApplicationKeyResponseWithDefaults instantiates a new PartialApplicationKeyResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewPartialApplicationKeyResponseWithDefaults() *PartialApplicationKeyResponse {
	this := PartialApplicationKeyResponse{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise.
func (o *PartialApplicationKeyResponse) GetData() PartialApplicationKey {
	if o == nil || o.Data == nil {
		var ret PartialApplicationKey
		return ret
	}
	return *o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialApplicationKeyResponse) GetDataOk() (*PartialApplicationKey, bool) {
	if o == nil || o.Data == nil {
		return nil, false
	}
	return o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *PartialApplicationKeyResponse) HasData() bool {
	return o != nil && o.Data != nil
}

// SetData gets a reference to the given PartialApplicationKey and assigns it to the Data field.
func (o *PartialApplicationKeyResponse) SetData(v PartialApplicationKey) {
	o.Data = &v
}

// GetIncluded returns the Included field value if set, zero value otherwise.
func (o *PartialApplicationKeyResponse) GetIncluded() []ApplicationKeyResponseIncludedItem {
	if o == nil || o.Included == nil {
		var ret []ApplicationKeyResponseIncludedItem
		return ret
	}
	return o.Included
}

// GetIncludedOk returns a tuple with the Included field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PartialApplicationKeyResponse) GetIncludedOk() (*[]ApplicationKeyResponseIncludedItem, bool) {
	if o == nil || o.Included == nil {
		return nil, false
	}
	return &o.Included, true
}

// HasIncluded returns a boolean if a field has been set.
func (o *PartialApplicationKeyResponse) HasIncluded() bool {
	return o != nil && o.Included != nil
}

// SetIncluded gets a reference to the given []ApplicationKeyResponseIncludedItem and assigns it to the Included field.
func (o *PartialApplicationKeyResponse) SetIncluded(v []ApplicationKeyResponseIncludedItem) {
	o.Included = v
}

// MarshalJSON serializes the struct using spec logic.
func (o PartialApplicationKeyResponse) MarshalJSON() ([]byte, error) {
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
func (o *PartialApplicationKeyResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Data     *PartialApplicationKey               `json:"data,omitempty"`
		Included []ApplicationKeyResponseIncludedItem `json:"included,omitempty"`
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
	if all.Data != nil && all.Data.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Data = all.Data
	o.Included = all.Included
	return nil
}
