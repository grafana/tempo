// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthDigest Object to handle digest authentication when performing the test.
type SyntheticsBasicAuthDigest struct {
	// Password to use for the digest authentication.
	Password string `json:"password"`
	// The type of basic authentication to use when performing the test.
	Type *SyntheticsBasicAuthDigestType `json:"type,omitempty"`
	// Username to use for the digest authentication.
	Username string `json:"username"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBasicAuthDigest instantiates a new SyntheticsBasicAuthDigest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBasicAuthDigest(password string, username string) *SyntheticsBasicAuthDigest {
	this := SyntheticsBasicAuthDigest{}
	this.Password = password
	var typeVar SyntheticsBasicAuthDigestType = SYNTHETICSBASICAUTHDIGESTTYPE_DIGEST
	this.Type = &typeVar
	this.Username = username
	return &this
}

// NewSyntheticsBasicAuthDigestWithDefaults instantiates a new SyntheticsBasicAuthDigest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBasicAuthDigestWithDefaults() *SyntheticsBasicAuthDigest {
	this := SyntheticsBasicAuthDigest{}
	var typeVar SyntheticsBasicAuthDigestType = SYNTHETICSBASICAUTHDIGESTTYPE_DIGEST
	this.Type = &typeVar
	return &this
}

// GetPassword returns the Password field value.
func (o *SyntheticsBasicAuthDigest) GetPassword() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Password
}

// GetPasswordOk returns a tuple with the Password field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthDigest) GetPasswordOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Password, true
}

// SetPassword sets field value.
func (o *SyntheticsBasicAuthDigest) SetPassword(v string) {
	o.Password = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthDigest) GetType() SyntheticsBasicAuthDigestType {
	if o == nil || o.Type == nil {
		var ret SyntheticsBasicAuthDigestType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthDigest) GetTypeOk() (*SyntheticsBasicAuthDigestType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthDigest) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SyntheticsBasicAuthDigestType and assigns it to the Type field.
func (o *SyntheticsBasicAuthDigest) SetType(v SyntheticsBasicAuthDigestType) {
	o.Type = &v
}

// GetUsername returns the Username field value.
func (o *SyntheticsBasicAuthDigest) GetUsername() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Username
}

// GetUsernameOk returns a tuple with the Username field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthDigest) GetUsernameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Username, true
}

// SetUsername sets field value.
func (o *SyntheticsBasicAuthDigest) SetUsername(v string) {
	o.Username = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBasicAuthDigest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["password"] = o.Password
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}
	toSerialize["username"] = o.Username

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBasicAuthDigest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Password *string `json:"password"`
		Username *string `json:"username"`
	}{}
	all := struct {
		Password string                         `json:"password"`
		Type     *SyntheticsBasicAuthDigestType `json:"type,omitempty"`
		Username string                         `json:"username"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Password == nil {
		return fmt.Errorf("required field password missing")
	}
	if required.Username == nil {
		return fmt.Errorf("required field username missing")
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Password = all.Password
	o.Type = all.Type
	o.Username = all.Username
	return nil
}
