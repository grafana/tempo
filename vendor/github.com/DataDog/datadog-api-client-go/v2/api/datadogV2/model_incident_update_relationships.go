// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IncidentUpdateRelationships The incident's relationships for an update request.
type IncidentUpdateRelationships struct {
	// Relationship to user.
	CommanderUser *NullableRelationshipToUser `json:"commander_user,omitempty"`
	// A relationship reference for multiple integration metadata objects.
	Integrations *RelationshipToIncidentIntegrationMetadatas `json:"integrations,omitempty"`
	// A relationship reference for postmortems.
	Postmortem *RelationshipToIncidentPostmortem `json:"postmortem,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIncidentUpdateRelationships instantiates a new IncidentUpdateRelationships object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIncidentUpdateRelationships() *IncidentUpdateRelationships {
	this := IncidentUpdateRelationships{}
	return &this
}

// NewIncidentUpdateRelationshipsWithDefaults instantiates a new IncidentUpdateRelationships object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIncidentUpdateRelationshipsWithDefaults() *IncidentUpdateRelationships {
	this := IncidentUpdateRelationships{}
	return &this
}

// GetCommanderUser returns the CommanderUser field value if set, zero value otherwise.
func (o *IncidentUpdateRelationships) GetCommanderUser() NullableRelationshipToUser {
	if o == nil || o.CommanderUser == nil {
		var ret NullableRelationshipToUser
		return ret
	}
	return *o.CommanderUser
}

// GetCommanderUserOk returns a tuple with the CommanderUser field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentUpdateRelationships) GetCommanderUserOk() (*NullableRelationshipToUser, bool) {
	if o == nil || o.CommanderUser == nil {
		return nil, false
	}
	return o.CommanderUser, true
}

// HasCommanderUser returns a boolean if a field has been set.
func (o *IncidentUpdateRelationships) HasCommanderUser() bool {
	return o != nil && o.CommanderUser != nil
}

// SetCommanderUser gets a reference to the given NullableRelationshipToUser and assigns it to the CommanderUser field.
func (o *IncidentUpdateRelationships) SetCommanderUser(v NullableRelationshipToUser) {
	o.CommanderUser = &v
}

// GetIntegrations returns the Integrations field value if set, zero value otherwise.
func (o *IncidentUpdateRelationships) GetIntegrations() RelationshipToIncidentIntegrationMetadatas {
	if o == nil || o.Integrations == nil {
		var ret RelationshipToIncidentIntegrationMetadatas
		return ret
	}
	return *o.Integrations
}

// GetIntegrationsOk returns a tuple with the Integrations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentUpdateRelationships) GetIntegrationsOk() (*RelationshipToIncidentIntegrationMetadatas, bool) {
	if o == nil || o.Integrations == nil {
		return nil, false
	}
	return o.Integrations, true
}

// HasIntegrations returns a boolean if a field has been set.
func (o *IncidentUpdateRelationships) HasIntegrations() bool {
	return o != nil && o.Integrations != nil
}

// SetIntegrations gets a reference to the given RelationshipToIncidentIntegrationMetadatas and assigns it to the Integrations field.
func (o *IncidentUpdateRelationships) SetIntegrations(v RelationshipToIncidentIntegrationMetadatas) {
	o.Integrations = &v
}

// GetPostmortem returns the Postmortem field value if set, zero value otherwise.
func (o *IncidentUpdateRelationships) GetPostmortem() RelationshipToIncidentPostmortem {
	if o == nil || o.Postmortem == nil {
		var ret RelationshipToIncidentPostmortem
		return ret
	}
	return *o.Postmortem
}

// GetPostmortemOk returns a tuple with the Postmortem field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IncidentUpdateRelationships) GetPostmortemOk() (*RelationshipToIncidentPostmortem, bool) {
	if o == nil || o.Postmortem == nil {
		return nil, false
	}
	return o.Postmortem, true
}

// HasPostmortem returns a boolean if a field has been set.
func (o *IncidentUpdateRelationships) HasPostmortem() bool {
	return o != nil && o.Postmortem != nil
}

// SetPostmortem gets a reference to the given RelationshipToIncidentPostmortem and assigns it to the Postmortem field.
func (o *IncidentUpdateRelationships) SetPostmortem(v RelationshipToIncidentPostmortem) {
	o.Postmortem = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o IncidentUpdateRelationships) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CommanderUser != nil {
		toSerialize["commander_user"] = o.CommanderUser
	}
	if o.Integrations != nil {
		toSerialize["integrations"] = o.Integrations
	}
	if o.Postmortem != nil {
		toSerialize["postmortem"] = o.Postmortem
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IncidentUpdateRelationships) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CommanderUser *NullableRelationshipToUser                 `json:"commander_user,omitempty"`
		Integrations  *RelationshipToIncidentIntegrationMetadatas `json:"integrations,omitempty"`
		Postmortem    *RelationshipToIncidentPostmortem           `json:"postmortem,omitempty"`
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
	if all.CommanderUser != nil && all.CommanderUser.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.CommanderUser = all.CommanderUser
	if all.Integrations != nil && all.Integrations.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Integrations = all.Integrations
	if all.Postmortem != nil && all.Postmortem.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Postmortem = all.Postmortem
	return nil
}
