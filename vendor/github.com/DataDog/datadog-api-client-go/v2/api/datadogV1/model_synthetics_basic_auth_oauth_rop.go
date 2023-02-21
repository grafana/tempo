// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthOauthROP Object to handle `oauth rop` authentication when performing the test.
type SyntheticsBasicAuthOauthROP struct {
	// Access token URL to use when performing the authentication.
	AccessTokenUrl string `json:"accessTokenUrl"`
	// Audience to use when performing the authentication.
	Audience *string `json:"audience,omitempty"`
	// Client ID to use when performing the authentication.
	ClientId *string `json:"clientId,omitempty"`
	// Client secret to use when performing the authentication.
	ClientSecret *string `json:"clientSecret,omitempty"`
	// Password to use when performing the authentication.
	Password string `json:"password"`
	// Resource to use when performing the authentication.
	Resource *string `json:"resource,omitempty"`
	// Scope to use when performing the authentication.
	Scope *string `json:"scope,omitempty"`
	// Type of token to use when performing the authentication.
	TokenApiAuthentication SyntheticsBasicAuthOauthTokenApiAuthentication `json:"tokenApiAuthentication"`
	// The type of basic authentication to use when performing the test.
	Type *SyntheticsBasicAuthOauthROPType `json:"type,omitempty"`
	// Username to use when performing the authentication.
	Username string `json:"username"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBasicAuthOauthROP instantiates a new SyntheticsBasicAuthOauthROP object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBasicAuthOauthROP(accessTokenUrl string, password string, tokenApiAuthentication SyntheticsBasicAuthOauthTokenApiAuthentication, username string) *SyntheticsBasicAuthOauthROP {
	this := SyntheticsBasicAuthOauthROP{}
	this.AccessTokenUrl = accessTokenUrl
	this.Password = password
	this.TokenApiAuthentication = tokenApiAuthentication
	var typeVar SyntheticsBasicAuthOauthROPType = SYNTHETICSBASICAUTHOAUTHROPTYPE_OAUTH_ROP
	this.Type = &typeVar
	this.Username = username
	return &this
}

// NewSyntheticsBasicAuthOauthROPWithDefaults instantiates a new SyntheticsBasicAuthOauthROP object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBasicAuthOauthROPWithDefaults() *SyntheticsBasicAuthOauthROP {
	this := SyntheticsBasicAuthOauthROP{}
	var typeVar SyntheticsBasicAuthOauthROPType = SYNTHETICSBASICAUTHOAUTHROPTYPE_OAUTH_ROP
	this.Type = &typeVar
	return &this
}

// GetAccessTokenUrl returns the AccessTokenUrl field value.
func (o *SyntheticsBasicAuthOauthROP) GetAccessTokenUrl() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.AccessTokenUrl
}

// GetAccessTokenUrlOk returns a tuple with the AccessTokenUrl field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetAccessTokenUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AccessTokenUrl, true
}

// SetAccessTokenUrl sets field value.
func (o *SyntheticsBasicAuthOauthROP) SetAccessTokenUrl(v string) {
	o.AccessTokenUrl = v
}

// GetAudience returns the Audience field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthROP) GetAudience() string {
	if o == nil || o.Audience == nil {
		var ret string
		return ret
	}
	return *o.Audience
}

// GetAudienceOk returns a tuple with the Audience field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetAudienceOk() (*string, bool) {
	if o == nil || o.Audience == nil {
		return nil, false
	}
	return o.Audience, true
}

// HasAudience returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthROP) HasAudience() bool {
	return o != nil && o.Audience != nil
}

// SetAudience gets a reference to the given string and assigns it to the Audience field.
func (o *SyntheticsBasicAuthOauthROP) SetAudience(v string) {
	o.Audience = &v
}

// GetClientId returns the ClientId field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthROP) GetClientId() string {
	if o == nil || o.ClientId == nil {
		var ret string
		return ret
	}
	return *o.ClientId
}

// GetClientIdOk returns a tuple with the ClientId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetClientIdOk() (*string, bool) {
	if o == nil || o.ClientId == nil {
		return nil, false
	}
	return o.ClientId, true
}

// HasClientId returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthROP) HasClientId() bool {
	return o != nil && o.ClientId != nil
}

// SetClientId gets a reference to the given string and assigns it to the ClientId field.
func (o *SyntheticsBasicAuthOauthROP) SetClientId(v string) {
	o.ClientId = &v
}

// GetClientSecret returns the ClientSecret field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthROP) GetClientSecret() string {
	if o == nil || o.ClientSecret == nil {
		var ret string
		return ret
	}
	return *o.ClientSecret
}

// GetClientSecretOk returns a tuple with the ClientSecret field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetClientSecretOk() (*string, bool) {
	if o == nil || o.ClientSecret == nil {
		return nil, false
	}
	return o.ClientSecret, true
}

// HasClientSecret returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthROP) HasClientSecret() bool {
	return o != nil && o.ClientSecret != nil
}

// SetClientSecret gets a reference to the given string and assigns it to the ClientSecret field.
func (o *SyntheticsBasicAuthOauthROP) SetClientSecret(v string) {
	o.ClientSecret = &v
}

// GetPassword returns the Password field value.
func (o *SyntheticsBasicAuthOauthROP) GetPassword() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Password
}

// GetPasswordOk returns a tuple with the Password field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetPasswordOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Password, true
}

// SetPassword sets field value.
func (o *SyntheticsBasicAuthOauthROP) SetPassword(v string) {
	o.Password = v
}

// GetResource returns the Resource field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthROP) GetResource() string {
	if o == nil || o.Resource == nil {
		var ret string
		return ret
	}
	return *o.Resource
}

// GetResourceOk returns a tuple with the Resource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetResourceOk() (*string, bool) {
	if o == nil || o.Resource == nil {
		return nil, false
	}
	return o.Resource, true
}

// HasResource returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthROP) HasResource() bool {
	return o != nil && o.Resource != nil
}

// SetResource gets a reference to the given string and assigns it to the Resource field.
func (o *SyntheticsBasicAuthOauthROP) SetResource(v string) {
	o.Resource = &v
}

// GetScope returns the Scope field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthROP) GetScope() string {
	if o == nil || o.Scope == nil {
		var ret string
		return ret
	}
	return *o.Scope
}

// GetScopeOk returns a tuple with the Scope field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetScopeOk() (*string, bool) {
	if o == nil || o.Scope == nil {
		return nil, false
	}
	return o.Scope, true
}

// HasScope returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthROP) HasScope() bool {
	return o != nil && o.Scope != nil
}

// SetScope gets a reference to the given string and assigns it to the Scope field.
func (o *SyntheticsBasicAuthOauthROP) SetScope(v string) {
	o.Scope = &v
}

// GetTokenApiAuthentication returns the TokenApiAuthentication field value.
func (o *SyntheticsBasicAuthOauthROP) GetTokenApiAuthentication() SyntheticsBasicAuthOauthTokenApiAuthentication {
	if o == nil {
		var ret SyntheticsBasicAuthOauthTokenApiAuthentication
		return ret
	}
	return o.TokenApiAuthentication
}

// GetTokenApiAuthenticationOk returns a tuple with the TokenApiAuthentication field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetTokenApiAuthenticationOk() (*SyntheticsBasicAuthOauthTokenApiAuthentication, bool) {
	if o == nil {
		return nil, false
	}
	return &o.TokenApiAuthentication, true
}

// SetTokenApiAuthentication sets field value.
func (o *SyntheticsBasicAuthOauthROP) SetTokenApiAuthentication(v SyntheticsBasicAuthOauthTokenApiAuthentication) {
	o.TokenApiAuthentication = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthROP) GetType() SyntheticsBasicAuthOauthROPType {
	if o == nil || o.Type == nil {
		var ret SyntheticsBasicAuthOauthROPType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetTypeOk() (*SyntheticsBasicAuthOauthROPType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthROP) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SyntheticsBasicAuthOauthROPType and assigns it to the Type field.
func (o *SyntheticsBasicAuthOauthROP) SetType(v SyntheticsBasicAuthOauthROPType) {
	o.Type = &v
}

// GetUsername returns the Username field value.
func (o *SyntheticsBasicAuthOauthROP) GetUsername() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Username
}

// GetUsernameOk returns a tuple with the Username field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthROP) GetUsernameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Username, true
}

// SetUsername sets field value.
func (o *SyntheticsBasicAuthOauthROP) SetUsername(v string) {
	o.Username = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBasicAuthOauthROP) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["accessTokenUrl"] = o.AccessTokenUrl
	if o.Audience != nil {
		toSerialize["audience"] = o.Audience
	}
	if o.ClientId != nil {
		toSerialize["clientId"] = o.ClientId
	}
	if o.ClientSecret != nil {
		toSerialize["clientSecret"] = o.ClientSecret
	}
	toSerialize["password"] = o.Password
	if o.Resource != nil {
		toSerialize["resource"] = o.Resource
	}
	if o.Scope != nil {
		toSerialize["scope"] = o.Scope
	}
	toSerialize["tokenApiAuthentication"] = o.TokenApiAuthentication
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
func (o *SyntheticsBasicAuthOauthROP) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		AccessTokenUrl         *string                                         `json:"accessTokenUrl"`
		Password               *string                                         `json:"password"`
		TokenApiAuthentication *SyntheticsBasicAuthOauthTokenApiAuthentication `json:"tokenApiAuthentication"`
		Username               *string                                         `json:"username"`
	}{}
	all := struct {
		AccessTokenUrl         string                                         `json:"accessTokenUrl"`
		Audience               *string                                        `json:"audience,omitempty"`
		ClientId               *string                                        `json:"clientId,omitempty"`
		ClientSecret           *string                                        `json:"clientSecret,omitempty"`
		Password               string                                         `json:"password"`
		Resource               *string                                        `json:"resource,omitempty"`
		Scope                  *string                                        `json:"scope,omitempty"`
		TokenApiAuthentication SyntheticsBasicAuthOauthTokenApiAuthentication `json:"tokenApiAuthentication"`
		Type                   *SyntheticsBasicAuthOauthROPType               `json:"type,omitempty"`
		Username               string                                         `json:"username"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.AccessTokenUrl == nil {
		return fmt.Errorf("required field accessTokenUrl missing")
	}
	if required.Password == nil {
		return fmt.Errorf("required field password missing")
	}
	if required.TokenApiAuthentication == nil {
		return fmt.Errorf("required field tokenApiAuthentication missing")
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
	if v := all.TokenApiAuthentication; !v.IsValid() {
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
	o.AccessTokenUrl = all.AccessTokenUrl
	o.Audience = all.Audience
	o.ClientId = all.ClientId
	o.ClientSecret = all.ClientSecret
	o.Password = all.Password
	o.Resource = all.Resource
	o.Scope = all.Scope
	o.TokenApiAuthentication = all.TokenApiAuthentication
	o.Type = all.Type
	o.Username = all.Username
	return nil
}
