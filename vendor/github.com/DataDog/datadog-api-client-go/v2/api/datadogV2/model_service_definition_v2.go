// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV2 Service definition V2 for providing service metadata and integrations.
type ServiceDefinitionV2 struct {
	// A list of contacts related to the services.
	Contacts []ServiceDefinitionV2Contact `json:"contacts,omitempty"`
	// Unique identifier of the service. Must be unique across all services and is used to match with a service in Datadog.
	DdService string `json:"dd-service"`
	// Experimental feature. A Team handle that matches a Team in the Datadog Teams product.
	DdTeam *string `json:"dd-team,omitempty"`
	// A list of documentation related to the services.
	Docs []ServiceDefinitionV2Doc `json:"docs,omitempty"`
	// Extensions to V2 schema.
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	// Third party integrations that Datadog supports.
	Integrations *ServiceDefinitionV2Integrations `json:"integrations,omitempty"`
	// A list of links related to the services.
	Links []ServiceDefinitionV2Link `json:"links,omitempty"`
	// A list of code repositories related to the services.
	Repos []ServiceDefinitionV2Repo `json:"repos,omitempty"`
	// Schema version being used.
	SchemaVersion ServiceDefinitionV2Version `json:"schema-version"`
	// A set of custom tags.
	Tags []string `json:"tags,omitempty"`
	// Team that owns the service.
	Team *string `json:"team,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV2 instantiates a new ServiceDefinitionV2 object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV2(ddService string, schemaVersion ServiceDefinitionV2Version) *ServiceDefinitionV2 {
	this := ServiceDefinitionV2{}
	this.DdService = ddService
	this.SchemaVersion = schemaVersion
	return &this
}

// NewServiceDefinitionV2WithDefaults instantiates a new ServiceDefinitionV2 object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV2WithDefaults() *ServiceDefinitionV2 {
	this := ServiceDefinitionV2{}
	var schemaVersion ServiceDefinitionV2Version = SERVICEDEFINITIONV2VERSION_V2
	this.SchemaVersion = schemaVersion
	return &this
}

// GetContacts returns the Contacts field value if set, zero value otherwise.
func (o *ServiceDefinitionV2) GetContacts() []ServiceDefinitionV2Contact {
	if o == nil || o.Contacts == nil {
		var ret []ServiceDefinitionV2Contact
		return ret
	}
	return o.Contacts
}

// GetContactsOk returns a tuple with the Contacts field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetContactsOk() (*[]ServiceDefinitionV2Contact, bool) {
	if o == nil || o.Contacts == nil {
		return nil, false
	}
	return &o.Contacts, true
}

// HasContacts returns a boolean if a field has been set.
func (o *ServiceDefinitionV2) HasContacts() bool {
	return o != nil && o.Contacts != nil
}

// SetContacts gets a reference to the given []ServiceDefinitionV2Contact and assigns it to the Contacts field.
func (o *ServiceDefinitionV2) SetContacts(v []ServiceDefinitionV2Contact) {
	o.Contacts = v
}

// GetDdService returns the DdService field value.
func (o *ServiceDefinitionV2) GetDdService() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.DdService
}

// GetDdServiceOk returns a tuple with the DdService field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetDdServiceOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.DdService, true
}

// SetDdService sets field value.
func (o *ServiceDefinitionV2) SetDdService(v string) {
	o.DdService = v
}

// GetDdTeam returns the DdTeam field value if set, zero value otherwise.
func (o *ServiceDefinitionV2) GetDdTeam() string {
	if o == nil || o.DdTeam == nil {
		var ret string
		return ret
	}
	return *o.DdTeam
}

// GetDdTeamOk returns a tuple with the DdTeam field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetDdTeamOk() (*string, bool) {
	if o == nil || o.DdTeam == nil {
		return nil, false
	}
	return o.DdTeam, true
}

// HasDdTeam returns a boolean if a field has been set.
func (o *ServiceDefinitionV2) HasDdTeam() bool {
	return o != nil && o.DdTeam != nil
}

// SetDdTeam gets a reference to the given string and assigns it to the DdTeam field.
func (o *ServiceDefinitionV2) SetDdTeam(v string) {
	o.DdTeam = &v
}

// GetDocs returns the Docs field value if set, zero value otherwise.
func (o *ServiceDefinitionV2) GetDocs() []ServiceDefinitionV2Doc {
	if o == nil || o.Docs == nil {
		var ret []ServiceDefinitionV2Doc
		return ret
	}
	return o.Docs
}

// GetDocsOk returns a tuple with the Docs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetDocsOk() (*[]ServiceDefinitionV2Doc, bool) {
	if o == nil || o.Docs == nil {
		return nil, false
	}
	return &o.Docs, true
}

// HasDocs returns a boolean if a field has been set.
func (o *ServiceDefinitionV2) HasDocs() bool {
	return o != nil && o.Docs != nil
}

// SetDocs gets a reference to the given []ServiceDefinitionV2Doc and assigns it to the Docs field.
func (o *ServiceDefinitionV2) SetDocs(v []ServiceDefinitionV2Doc) {
	o.Docs = v
}

// GetExtensions returns the Extensions field value if set, zero value otherwise.
func (o *ServiceDefinitionV2) GetExtensions() map[string]interface{} {
	if o == nil || o.Extensions == nil {
		var ret map[string]interface{}
		return ret
	}
	return o.Extensions
}

// GetExtensionsOk returns a tuple with the Extensions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetExtensionsOk() (*map[string]interface{}, bool) {
	if o == nil || o.Extensions == nil {
		return nil, false
	}
	return &o.Extensions, true
}

// HasExtensions returns a boolean if a field has been set.
func (o *ServiceDefinitionV2) HasExtensions() bool {
	return o != nil && o.Extensions != nil
}

// SetExtensions gets a reference to the given map[string]interface{} and assigns it to the Extensions field.
func (o *ServiceDefinitionV2) SetExtensions(v map[string]interface{}) {
	o.Extensions = v
}

// GetIntegrations returns the Integrations field value if set, zero value otherwise.
func (o *ServiceDefinitionV2) GetIntegrations() ServiceDefinitionV2Integrations {
	if o == nil || o.Integrations == nil {
		var ret ServiceDefinitionV2Integrations
		return ret
	}
	return *o.Integrations
}

// GetIntegrationsOk returns a tuple with the Integrations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetIntegrationsOk() (*ServiceDefinitionV2Integrations, bool) {
	if o == nil || o.Integrations == nil {
		return nil, false
	}
	return o.Integrations, true
}

// HasIntegrations returns a boolean if a field has been set.
func (o *ServiceDefinitionV2) HasIntegrations() bool {
	return o != nil && o.Integrations != nil
}

// SetIntegrations gets a reference to the given ServiceDefinitionV2Integrations and assigns it to the Integrations field.
func (o *ServiceDefinitionV2) SetIntegrations(v ServiceDefinitionV2Integrations) {
	o.Integrations = &v
}

// GetLinks returns the Links field value if set, zero value otherwise.
func (o *ServiceDefinitionV2) GetLinks() []ServiceDefinitionV2Link {
	if o == nil || o.Links == nil {
		var ret []ServiceDefinitionV2Link
		return ret
	}
	return o.Links
}

// GetLinksOk returns a tuple with the Links field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetLinksOk() (*[]ServiceDefinitionV2Link, bool) {
	if o == nil || o.Links == nil {
		return nil, false
	}
	return &o.Links, true
}

// HasLinks returns a boolean if a field has been set.
func (o *ServiceDefinitionV2) HasLinks() bool {
	return o != nil && o.Links != nil
}

// SetLinks gets a reference to the given []ServiceDefinitionV2Link and assigns it to the Links field.
func (o *ServiceDefinitionV2) SetLinks(v []ServiceDefinitionV2Link) {
	o.Links = v
}

// GetRepos returns the Repos field value if set, zero value otherwise.
func (o *ServiceDefinitionV2) GetRepos() []ServiceDefinitionV2Repo {
	if o == nil || o.Repos == nil {
		var ret []ServiceDefinitionV2Repo
		return ret
	}
	return o.Repos
}

// GetReposOk returns a tuple with the Repos field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetReposOk() (*[]ServiceDefinitionV2Repo, bool) {
	if o == nil || o.Repos == nil {
		return nil, false
	}
	return &o.Repos, true
}

// HasRepos returns a boolean if a field has been set.
func (o *ServiceDefinitionV2) HasRepos() bool {
	return o != nil && o.Repos != nil
}

// SetRepos gets a reference to the given []ServiceDefinitionV2Repo and assigns it to the Repos field.
func (o *ServiceDefinitionV2) SetRepos(v []ServiceDefinitionV2Repo) {
	o.Repos = v
}

// GetSchemaVersion returns the SchemaVersion field value.
func (o *ServiceDefinitionV2) GetSchemaVersion() ServiceDefinitionV2Version {
	if o == nil {
		var ret ServiceDefinitionV2Version
		return ret
	}
	return o.SchemaVersion
}

// GetSchemaVersionOk returns a tuple with the SchemaVersion field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetSchemaVersionOk() (*ServiceDefinitionV2Version, bool) {
	if o == nil {
		return nil, false
	}
	return &o.SchemaVersion, true
}

// SetSchemaVersion sets field value.
func (o *ServiceDefinitionV2) SetSchemaVersion(v ServiceDefinitionV2Version) {
	o.SchemaVersion = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *ServiceDefinitionV2) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *ServiceDefinitionV2) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *ServiceDefinitionV2) SetTags(v []string) {
	o.Tags = v
}

// GetTeam returns the Team field value if set, zero value otherwise.
func (o *ServiceDefinitionV2) GetTeam() string {
	if o == nil || o.Team == nil {
		var ret string
		return ret
	}
	return *o.Team
}

// GetTeamOk returns a tuple with the Team field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV2) GetTeamOk() (*string, bool) {
	if o == nil || o.Team == nil {
		return nil, false
	}
	return o.Team, true
}

// HasTeam returns a boolean if a field has been set.
func (o *ServiceDefinitionV2) HasTeam() bool {
	return o != nil && o.Team != nil
}

// SetTeam gets a reference to the given string and assigns it to the Team field.
func (o *ServiceDefinitionV2) SetTeam(v string) {
	o.Team = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV2) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Contacts != nil {
		toSerialize["contacts"] = o.Contacts
	}
	toSerialize["dd-service"] = o.DdService
	if o.DdTeam != nil {
		toSerialize["dd-team"] = o.DdTeam
	}
	if o.Docs != nil {
		toSerialize["docs"] = o.Docs
	}
	if o.Extensions != nil {
		toSerialize["extensions"] = o.Extensions
	}
	if o.Integrations != nil {
		toSerialize["integrations"] = o.Integrations
	}
	if o.Links != nil {
		toSerialize["links"] = o.Links
	}
	if o.Repos != nil {
		toSerialize["repos"] = o.Repos
	}
	toSerialize["schema-version"] = o.SchemaVersion
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
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
func (o *ServiceDefinitionV2) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		DdService     *string                     `json:"dd-service"`
		SchemaVersion *ServiceDefinitionV2Version `json:"schema-version"`
	}{}
	all := struct {
		Contacts      []ServiceDefinitionV2Contact     `json:"contacts,omitempty"`
		DdService     string                           `json:"dd-service"`
		DdTeam        *string                          `json:"dd-team,omitempty"`
		Docs          []ServiceDefinitionV2Doc         `json:"docs,omitempty"`
		Extensions    map[string]interface{}           `json:"extensions,omitempty"`
		Integrations  *ServiceDefinitionV2Integrations `json:"integrations,omitempty"`
		Links         []ServiceDefinitionV2Link        `json:"links,omitempty"`
		Repos         []ServiceDefinitionV2Repo        `json:"repos,omitempty"`
		SchemaVersion ServiceDefinitionV2Version       `json:"schema-version"`
		Tags          []string                         `json:"tags,omitempty"`
		Team          *string                          `json:"team,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.DdService == nil {
		return fmt.Errorf("required field dd-service missing")
	}
	if required.SchemaVersion == nil {
		return fmt.Errorf("required field schema-version missing")
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
	if v := all.SchemaVersion; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Contacts = all.Contacts
	o.DdService = all.DdService
	o.DdTeam = all.DdTeam
	o.Docs = all.Docs
	o.Extensions = all.Extensions
	if all.Integrations != nil && all.Integrations.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Integrations = all.Integrations
	o.Links = all.Links
	o.Repos = all.Repos
	o.SchemaVersion = all.SchemaVersion
	o.Tags = all.Tags
	o.Team = all.Team
	return nil
}
