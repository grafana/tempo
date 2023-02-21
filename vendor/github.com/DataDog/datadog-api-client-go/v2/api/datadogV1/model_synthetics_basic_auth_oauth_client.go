// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SyntheticsBasicAuthOauthClient Object to handle `oauth client` authentication when performing the test.
type SyntheticsBasicAuthOauthClient struct {
	// Access token URL to use when performing the authentication.
	AccessTokenUrl string `json:"accessTokenUrl"`
	// Audience to use when performing the authentication.
	Audience *string `json:"audience,omitempty"`
	// Client ID to use when performing the authentication.
	ClientId string `json:"clientId"`
	// Client secret to use when performing the authentication.
	ClientSecret string `json:"clientSecret"`
	// Resource to use when performing the authentication.
	Resource *string `json:"resource,omitempty"`
	// Scope to use when performing the authentication.
	Scope *string `json:"scope,omitempty"`
	// Type of token to use when performing the authentication.
	TokenApiAuthentication SyntheticsBasicAuthOauthTokenApiAuthentication `json:"tokenApiAuthentication"`
	// The type of basic authentication to use when performing the test.
	Type *SyntheticsBasicAuthOauthClientType `json:"type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSyntheticsBasicAuthOauthClient instantiates a new SyntheticsBasicAuthOauthClient object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSyntheticsBasicAuthOauthClient(accessTokenUrl string, clientId string, clientSecret string, tokenApiAuthentication SyntheticsBasicAuthOauthTokenApiAuthentication) *SyntheticsBasicAuthOauthClient {
	this := SyntheticsBasicAuthOauthClient{}
	this.AccessTokenUrl = accessTokenUrl
	this.ClientId = clientId
	this.ClientSecret = clientSecret
	this.TokenApiAuthentication = tokenApiAuthentication
	var typeVar SyntheticsBasicAuthOauthClientType = SYNTHETICSBASICAUTHOAUTHCLIENTTYPE_OAUTH_CLIENT
	this.Type = &typeVar
	return &this
}

// NewSyntheticsBasicAuthOauthClientWithDefaults instantiates a new SyntheticsBasicAuthOauthClient object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSyntheticsBasicAuthOauthClientWithDefaults() *SyntheticsBasicAuthOauthClient {
	this := SyntheticsBasicAuthOauthClient{}
	var typeVar SyntheticsBasicAuthOauthClientType = SYNTHETICSBASICAUTHOAUTHCLIENTTYPE_OAUTH_CLIENT
	this.Type = &typeVar
	return &this
}

// GetAccessTokenUrl returns the AccessTokenUrl field value.
func (o *SyntheticsBasicAuthOauthClient) GetAccessTokenUrl() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.AccessTokenUrl
}

// GetAccessTokenUrlOk returns a tuple with the AccessTokenUrl field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthClient) GetAccessTokenUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AccessTokenUrl, true
}

// SetAccessTokenUrl sets field value.
func (o *SyntheticsBasicAuthOauthClient) SetAccessTokenUrl(v string) {
	o.AccessTokenUrl = v
}

// GetAudience returns the Audience field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthClient) GetAudience() string {
	if o == nil || o.Audience == nil {
		var ret string
		return ret
	}
	return *o.Audience
}

// GetAudienceOk returns a tuple with the Audience field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthClient) GetAudienceOk() (*string, bool) {
	if o == nil || o.Audience == nil {
		return nil, false
	}
	return o.Audience, true
}

// HasAudience returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthClient) HasAudience() bool {
	return o != nil && o.Audience != nil
}

// SetAudience gets a reference to the given string and assigns it to the Audience field.
func (o *SyntheticsBasicAuthOauthClient) SetAudience(v string) {
	o.Audience = &v
}

// GetClientId returns the ClientId field value.
func (o *SyntheticsBasicAuthOauthClient) GetClientId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ClientId
}

// GetClientIdOk returns a tuple with the ClientId field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthClient) GetClientIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ClientId, true
}

// SetClientId sets field value.
func (o *SyntheticsBasicAuthOauthClient) SetClientId(v string) {
	o.ClientId = v
}

// GetClientSecret returns the ClientSecret field value.
func (o *SyntheticsBasicAuthOauthClient) GetClientSecret() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ClientSecret
}

// GetClientSecretOk returns a tuple with the ClientSecret field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthClient) GetClientSecretOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ClientSecret, true
}

// SetClientSecret sets field value.
func (o *SyntheticsBasicAuthOauthClient) SetClientSecret(v string) {
	o.ClientSecret = v
}

// GetResource returns the Resource field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthClient) GetResource() string {
	if o == nil || o.Resource == nil {
		var ret string
		return ret
	}
	return *o.Resource
}

// GetResourceOk returns a tuple with the Resource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthClient) GetResourceOk() (*string, bool) {
	if o == nil || o.Resource == nil {
		return nil, false
	}
	return o.Resource, true
}

// HasResource returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthClient) HasResource() bool {
	return o != nil && o.Resource != nil
}

// SetResource gets a reference to the given string and assigns it to the Resource field.
func (o *SyntheticsBasicAuthOauthClient) SetResource(v string) {
	o.Resource = &v
}

// GetScope returns the Scope field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthClient) GetScope() string {
	if o == nil || o.Scope == nil {
		var ret string
		return ret
	}
	return *o.Scope
}

// GetScopeOk returns a tuple with the Scope field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthClient) GetScopeOk() (*string, bool) {
	if o == nil || o.Scope == nil {
		return nil, false
	}
	return o.Scope, true
}

// HasScope returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthClient) HasScope() bool {
	return o != nil && o.Scope != nil
}

// SetScope gets a reference to the given string and assigns it to the Scope field.
func (o *SyntheticsBasicAuthOauthClient) SetScope(v string) {
	o.Scope = &v
}

// GetTokenApiAuthentication returns the TokenApiAuthentication field value.
func (o *SyntheticsBasicAuthOauthClient) GetTokenApiAuthentication() SyntheticsBasicAuthOauthTokenApiAuthentication {
	if o == nil {
		var ret SyntheticsBasicAuthOauthTokenApiAuthentication
		return ret
	}
	return o.TokenApiAuthentication
}

// GetTokenApiAuthenticationOk returns a tuple with the TokenApiAuthentication field value
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthClient) GetTokenApiAuthenticationOk() (*SyntheticsBasicAuthOauthTokenApiAuthentication, bool) {
	if o == nil {
		return nil, false
	}
	return &o.TokenApiAuthentication, true
}

// SetTokenApiAuthentication sets field value.
func (o *SyntheticsBasicAuthOauthClient) SetTokenApiAuthentication(v SyntheticsBasicAuthOauthTokenApiAuthentication) {
	o.TokenApiAuthentication = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *SyntheticsBasicAuthOauthClient) GetType() SyntheticsBasicAuthOauthClientType {
	if o == nil || o.Type == nil {
		var ret SyntheticsBasicAuthOauthClientType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SyntheticsBasicAuthOauthClient) GetTypeOk() (*SyntheticsBasicAuthOauthClientType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *SyntheticsBasicAuthOauthClient) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given SyntheticsBasicAuthOauthClientType and assigns it to the Type field.
func (o *SyntheticsBasicAuthOauthClient) SetType(v SyntheticsBasicAuthOauthClientType) {
	o.Type = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SyntheticsBasicAuthOauthClient) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["accessTokenUrl"] = o.AccessTokenUrl
	if o.Audience != nil {
		toSerialize["audience"] = o.Audience
	}
	toSerialize["clientId"] = o.ClientId
	toSerialize["clientSecret"] = o.ClientSecret
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

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SyntheticsBasicAuthOauthClient) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		AccessTokenUrl         *string                                         `json:"accessTokenUrl"`
		ClientId               *string                                         `json:"clientId"`
		ClientSecret           *string                                         `json:"clientSecret"`
		TokenApiAuthentication *SyntheticsBasicAuthOauthTokenApiAuthentication `json:"tokenApiAuthentication"`
	}{}
	all := struct {
		AccessTokenUrl         string                                         `json:"accessTokenUrl"`
		Audience               *string                                        `json:"audience,omitempty"`
		ClientId               string                                         `json:"clientId"`
		ClientSecret           string                                         `json:"clientSecret"`
		Resource               *string                                        `json:"resource,omitempty"`
		Scope                  *string                                        `json:"scope,omitempty"`
		TokenApiAuthentication SyntheticsBasicAuthOauthTokenApiAuthentication `json:"tokenApiAuthentication"`
		Type                   *SyntheticsBasicAuthOauthClientType            `json:"type,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.AccessTokenUrl == nil {
		return fmt.Errorf("required field accessTokenUrl missing")
	}
	if required.ClientId == nil {
		return fmt.Errorf("required field clientId missing")
	}
	if required.ClientSecret == nil {
		return fmt.Errorf("required field clientSecret missing")
	}
	if required.TokenApiAuthentication == nil {
		return fmt.Errorf("required field tokenApiAuthentication missing")
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
	o.Resource = all.Resource
	o.Scope = all.Scope
	o.TokenApiAuthentication = all.TokenApiAuthentication
	o.Type = all.Type
	return nil
}
