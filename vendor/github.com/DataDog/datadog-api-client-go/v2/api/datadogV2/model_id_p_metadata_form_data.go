// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"os"
)

// IdPMetadataFormData The form data submitted to upload IdP metadata
type IdPMetadataFormData struct {
	// The IdP metadata XML file
	IdpFile **os.File `json:"idp_file,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIdPMetadataFormData instantiates a new IdPMetadataFormData object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIdPMetadataFormData() *IdPMetadataFormData {
	this := IdPMetadataFormData{}
	return &this
}

// NewIdPMetadataFormDataWithDefaults instantiates a new IdPMetadataFormData object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIdPMetadataFormDataWithDefaults() *IdPMetadataFormData {
	this := IdPMetadataFormData{}
	return &this
}

// GetIdpFile returns the IdpFile field value if set, zero value otherwise.
func (o *IdPMetadataFormData) GetIdpFile() *os.File {
	if o == nil || o.IdpFile == nil {
		var ret *os.File
		return ret
	}
	return *o.IdpFile
}

// GetIdpFileOk returns a tuple with the IdpFile field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IdPMetadataFormData) GetIdpFileOk() (**os.File, bool) {
	if o == nil || o.IdpFile == nil {
		return nil, false
	}
	return o.IdpFile, true
}

// HasIdpFile returns a boolean if a field has been set.
func (o *IdPMetadataFormData) HasIdpFile() bool {
	return o != nil && o.IdpFile != nil
}

// SetIdpFile gets a reference to the given *os.File and assigns it to the IdpFile field.
func (o *IdPMetadataFormData) SetIdpFile(v *os.File) {
	o.IdpFile = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IdPMetadataFormData) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.IdpFile != nil {
		toSerialize["idp_file"] = o.IdpFile
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IdPMetadataFormData) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		IdpFile **os.File `json:"idp_file,omitempty"`
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
	o.IdpFile = all.IdpFile
	return nil
}
