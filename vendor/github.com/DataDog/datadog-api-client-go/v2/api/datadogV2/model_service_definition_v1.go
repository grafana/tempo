// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// ServiceDefinitionV1 Deprecated - Service definition V1 for providing additional service metadata and integrations.
//
// Deprecated: This model is deprecated.
type ServiceDefinitionV1 struct {
	// Contact information about the service.
	Contact *ServiceDefinitionV1Contact `json:"contact,omitempty"`
	// Extensions to V1 schema.
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	// A list of external links related to the services.
	ExternalResources []ServiceDefinitionV1Resource `json:"external-resources,omitempty"`
	// Basic information about a service.
	Info ServiceDefinitionV1Info `json:"info"`
	// Third party integrations that Datadog supports.
	Integrations *ServiceDefinitionV1Integrations `json:"integrations,omitempty"`
	// Org related information about the service.
	Org *ServiceDefinitionV1Org `json:"org,omitempty"`
	// Schema version being used.
	SchemaVersion ServiceDefinitionV1Version `json:"schema-version"`
	// A set of custom tags.
	Tags []string `json:"tags,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewServiceDefinitionV1 instantiates a new ServiceDefinitionV1 object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewServiceDefinitionV1(info ServiceDefinitionV1Info, schemaVersion ServiceDefinitionV1Version) *ServiceDefinitionV1 {
	this := ServiceDefinitionV1{}
	this.Info = info
	this.SchemaVersion = schemaVersion
	return &this
}

// NewServiceDefinitionV1WithDefaults instantiates a new ServiceDefinitionV1 object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewServiceDefinitionV1WithDefaults() *ServiceDefinitionV1 {
	this := ServiceDefinitionV1{}
	var schemaVersion ServiceDefinitionV1Version = SERVICEDEFINITIONV1VERSION_V1
	this.SchemaVersion = schemaVersion
	return &this
}

// GetContact returns the Contact field value if set, zero value otherwise.
func (o *ServiceDefinitionV1) GetContact() ServiceDefinitionV1Contact {
	if o == nil || o.Contact == nil {
		var ret ServiceDefinitionV1Contact
		return ret
	}
	return *o.Contact
}

// GetContactOk returns a tuple with the Contact field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1) GetContactOk() (*ServiceDefinitionV1Contact, bool) {
	if o == nil || o.Contact == nil {
		return nil, false
	}
	return o.Contact, true
}

// HasContact returns a boolean if a field has been set.
func (o *ServiceDefinitionV1) HasContact() bool {
	return o != nil && o.Contact != nil
}

// SetContact gets a reference to the given ServiceDefinitionV1Contact and assigns it to the Contact field.
func (o *ServiceDefinitionV1) SetContact(v ServiceDefinitionV1Contact) {
	o.Contact = &v
}

// GetExtensions returns the Extensions field value if set, zero value otherwise.
func (o *ServiceDefinitionV1) GetExtensions() map[string]interface{} {
	if o == nil || o.Extensions == nil {
		var ret map[string]interface{}
		return ret
	}
	return o.Extensions
}

// GetExtensionsOk returns a tuple with the Extensions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1) GetExtensionsOk() (*map[string]interface{}, bool) {
	if o == nil || o.Extensions == nil {
		return nil, false
	}
	return &o.Extensions, true
}

// HasExtensions returns a boolean if a field has been set.
func (o *ServiceDefinitionV1) HasExtensions() bool {
	return o != nil && o.Extensions != nil
}

// SetExtensions gets a reference to the given map[string]interface{} and assigns it to the Extensions field.
func (o *ServiceDefinitionV1) SetExtensions(v map[string]interface{}) {
	o.Extensions = v
}

// GetExternalResources returns the ExternalResources field value if set, zero value otherwise.
func (o *ServiceDefinitionV1) GetExternalResources() []ServiceDefinitionV1Resource {
	if o == nil || o.ExternalResources == nil {
		var ret []ServiceDefinitionV1Resource
		return ret
	}
	return o.ExternalResources
}

// GetExternalResourcesOk returns a tuple with the ExternalResources field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1) GetExternalResourcesOk() (*[]ServiceDefinitionV1Resource, bool) {
	if o == nil || o.ExternalResources == nil {
		return nil, false
	}
	return &o.ExternalResources, true
}

// HasExternalResources returns a boolean if a field has been set.
func (o *ServiceDefinitionV1) HasExternalResources() bool {
	return o != nil && o.ExternalResources != nil
}

// SetExternalResources gets a reference to the given []ServiceDefinitionV1Resource and assigns it to the ExternalResources field.
func (o *ServiceDefinitionV1) SetExternalResources(v []ServiceDefinitionV1Resource) {
	o.ExternalResources = v
}

// GetInfo returns the Info field value.
func (o *ServiceDefinitionV1) GetInfo() ServiceDefinitionV1Info {
	if o == nil {
		var ret ServiceDefinitionV1Info
		return ret
	}
	return o.Info
}

// GetInfoOk returns a tuple with the Info field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1) GetInfoOk() (*ServiceDefinitionV1Info, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Info, true
}

// SetInfo sets field value.
func (o *ServiceDefinitionV1) SetInfo(v ServiceDefinitionV1Info) {
	o.Info = v
}

// GetIntegrations returns the Integrations field value if set, zero value otherwise.
func (o *ServiceDefinitionV1) GetIntegrations() ServiceDefinitionV1Integrations {
	if o == nil || o.Integrations == nil {
		var ret ServiceDefinitionV1Integrations
		return ret
	}
	return *o.Integrations
}

// GetIntegrationsOk returns a tuple with the Integrations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1) GetIntegrationsOk() (*ServiceDefinitionV1Integrations, bool) {
	if o == nil || o.Integrations == nil {
		return nil, false
	}
	return o.Integrations, true
}

// HasIntegrations returns a boolean if a field has been set.
func (o *ServiceDefinitionV1) HasIntegrations() bool {
	return o != nil && o.Integrations != nil
}

// SetIntegrations gets a reference to the given ServiceDefinitionV1Integrations and assigns it to the Integrations field.
func (o *ServiceDefinitionV1) SetIntegrations(v ServiceDefinitionV1Integrations) {
	o.Integrations = &v
}

// GetOrg returns the Org field value if set, zero value otherwise.
func (o *ServiceDefinitionV1) GetOrg() ServiceDefinitionV1Org {
	if o == nil || o.Org == nil {
		var ret ServiceDefinitionV1Org
		return ret
	}
	return *o.Org
}

// GetOrgOk returns a tuple with the Org field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1) GetOrgOk() (*ServiceDefinitionV1Org, bool) {
	if o == nil || o.Org == nil {
		return nil, false
	}
	return o.Org, true
}

// HasOrg returns a boolean if a field has been set.
func (o *ServiceDefinitionV1) HasOrg() bool {
	return o != nil && o.Org != nil
}

// SetOrg gets a reference to the given ServiceDefinitionV1Org and assigns it to the Org field.
func (o *ServiceDefinitionV1) SetOrg(v ServiceDefinitionV1Org) {
	o.Org = &v
}

// GetSchemaVersion returns the SchemaVersion field value.
func (o *ServiceDefinitionV1) GetSchemaVersion() ServiceDefinitionV1Version {
	if o == nil {
		var ret ServiceDefinitionV1Version
		return ret
	}
	return o.SchemaVersion
}

// GetSchemaVersionOk returns a tuple with the SchemaVersion field value
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1) GetSchemaVersionOk() (*ServiceDefinitionV1Version, bool) {
	if o == nil {
		return nil, false
	}
	return &o.SchemaVersion, true
}

// SetSchemaVersion sets field value.
func (o *ServiceDefinitionV1) SetSchemaVersion(v ServiceDefinitionV1Version) {
	o.SchemaVersion = v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *ServiceDefinitionV1) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ServiceDefinitionV1) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *ServiceDefinitionV1) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *ServiceDefinitionV1) SetTags(v []string) {
	o.Tags = v
}

// MarshalJSON serializes the struct using spec logic.
func (o ServiceDefinitionV1) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Contact != nil {
		toSerialize["contact"] = o.Contact
	}
	if o.Extensions != nil {
		toSerialize["extensions"] = o.Extensions
	}
	if o.ExternalResources != nil {
		toSerialize["external-resources"] = o.ExternalResources
	}
	toSerialize["info"] = o.Info
	if o.Integrations != nil {
		toSerialize["integrations"] = o.Integrations
	}
	if o.Org != nil {
		toSerialize["org"] = o.Org
	}
	toSerialize["schema-version"] = o.SchemaVersion
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ServiceDefinitionV1) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Info          *ServiceDefinitionV1Info    `json:"info"`
		SchemaVersion *ServiceDefinitionV1Version `json:"schema-version"`
	}{}
	all := struct {
		Contact           *ServiceDefinitionV1Contact      `json:"contact,omitempty"`
		Extensions        map[string]interface{}           `json:"extensions,omitempty"`
		ExternalResources []ServiceDefinitionV1Resource    `json:"external-resources,omitempty"`
		Info              ServiceDefinitionV1Info          `json:"info"`
		Integrations      *ServiceDefinitionV1Integrations `json:"integrations,omitempty"`
		Org               *ServiceDefinitionV1Org          `json:"org,omitempty"`
		SchemaVersion     ServiceDefinitionV1Version       `json:"schema-version"`
		Tags              []string                         `json:"tags,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Info == nil {
		return fmt.Errorf("required field info missing")
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
	if all.Contact != nil && all.Contact.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Contact = all.Contact
	o.Extensions = all.Extensions
	o.ExternalResources = all.ExternalResources
	if all.Info.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Info = all.Info
	if all.Integrations != nil && all.Integrations.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Integrations = all.Integrations
	if all.Org != nil && all.Org.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Org = all.Org
	o.SchemaVersion = all.SchemaVersion
	o.Tags = all.Tags
	return nil
}
