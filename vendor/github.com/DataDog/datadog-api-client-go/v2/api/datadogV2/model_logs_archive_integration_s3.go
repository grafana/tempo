// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// LogsArchiveIntegrationS3 The S3 Archive's integration destination.
type LogsArchiveIntegrationS3 struct {
	// The account ID for the integration.
	AccountId string `json:"account_id"`
	// The path of the integration.
	RoleName string `json:"role_name"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewLogsArchiveIntegrationS3 instantiates a new LogsArchiveIntegrationS3 object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsArchiveIntegrationS3(accountId string, roleName string) *LogsArchiveIntegrationS3 {
	this := LogsArchiveIntegrationS3{}
	this.AccountId = accountId
	this.RoleName = roleName
	return &this
}

// NewLogsArchiveIntegrationS3WithDefaults instantiates a new LogsArchiveIntegrationS3 object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsArchiveIntegrationS3WithDefaults() *LogsArchiveIntegrationS3 {
	this := LogsArchiveIntegrationS3{}
	return &this
}

// GetAccountId returns the AccountId field value.
func (o *LogsArchiveIntegrationS3) GetAccountId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.AccountId
}

// GetAccountIdOk returns a tuple with the AccountId field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveIntegrationS3) GetAccountIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.AccountId, true
}

// SetAccountId sets field value.
func (o *LogsArchiveIntegrationS3) SetAccountId(v string) {
	o.AccountId = v
}

// GetRoleName returns the RoleName field value.
func (o *LogsArchiveIntegrationS3) GetRoleName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.RoleName
}

// GetRoleNameOk returns a tuple with the RoleName field value
// and a boolean to check if the value has been set.
func (o *LogsArchiveIntegrationS3) GetRoleNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.RoleName, true
}

// SetRoleName sets field value.
func (o *LogsArchiveIntegrationS3) SetRoleName(v string) {
	o.RoleName = v
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsArchiveIntegrationS3) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["account_id"] = o.AccountId
	toSerialize["role_name"] = o.RoleName

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsArchiveIntegrationS3) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		AccountId *string `json:"account_id"`
		RoleName  *string `json:"role_name"`
	}{}
	all := struct {
		AccountId string `json:"account_id"`
		RoleName  string `json:"role_name"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.AccountId == nil {
		return fmt.Errorf("required field account_id missing")
	}
	if required.RoleName == nil {
		return fmt.Errorf("required field role_name missing")
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
	o.AccountId = all.AccountId
	o.RoleName = all.RoleName
	return nil
}
