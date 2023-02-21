// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ServiceDefinitionV1Org Org related information about the service.
type ServiceDefinitionV1Org struct {
	// App feature this service supports.
	Application *string `json:"application,omitempty"`
	// Team that owns the service.
	Team *string `json:"team,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV1Org instantiates a new ServiceDefinitionV1Org object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV1Org() *ServiceDefinitionV1Org {
	this := ServiceDefinitionV1Org{}
	return &this
}

// NewServiceDefinitionV1OrgWithDefaults instantiates a new ServiceDefinitionV1Org object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV1OrgWithDefaults() *ServiceDefinitionV1Org {
	this := ServiceDefinitionV1Org{}
	return &this
}

// GetApplication returns the Application field value if set, zero value otherwise.
func (o *ServiceDefinitionV1Org) GetApplication() string {
	if o == nil || o.Application == nil {
		var ret string
		return ret
	}
	return *o.Application
}

// GetApplicationOk returns a tuple with the Application field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Org) GetApplicationOk() (*string, bool) {
	if o == nil || o.Application == nil {
		return nil, false
	}
	return o.Application, true
}

// HasApplication returns a boolean if a field has been set.
func (o *ServiceDefinitionV1Org) HasApplication() bool {
	return o != nil && o.Application != nil
}

// SetApplication gets a reference to the given string and assigns it to the Application field.
func (o *ServiceDefinitionV1Org) SetApplication(v string) {
	o.Application = &v
}

// GetTeam returns the Team field value if set, zero value otherwise.
func (o *ServiceDefinitionV1Org) GetTeam() string {
	if o == nil || o.Team == nil {
		var ret string
		return ret
	}
	return *o.Team
}

// GetTeamOk returns a tuple with the Team field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1Org) GetTeamOk() (*string, bool) {
	if o == nil || o.Team == nil {
		return nil, false
	}
	return o.Team, true
}

// HasTeam returns a boolean if a field has been set.
func (o *ServiceDefinitionV1Org) HasTeam() bool {
	return o != nil && o.Team != nil
}

// SetTeam gets a reference to the given string and assigns it to the Team field.
func (o *ServiceDefinitionV1Org) SetTeam(v string) {
	o.Team = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV1Org) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Application != nil {
		toSerialize["application"] = o.Application
	}
	if o.Team != nil {
		toSerialize["team"] = o.Team
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionV1Org) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Application *string `json:"application,omitempty"`
		Team        *string `json:"team,omitempty"`
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
	o.Application = all.Application
	o.Team = all.Team
	return nil
}
