// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// SLOListWidgetDefinition Use the SLO List widget to track your SLOs (Service Level Objectives) on dashboards.
type SLOListWidgetDefinition struct {
	// Array of one request object to display in the widget.
	Requests []SLOListWidgetRequest `json:"requests"`
	// Title of the widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the SLO List widget.
	Type SLOListWidgetDefinitionType `json:"type"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSLOListWidgetDefinition instantiates a new SLOListWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSLOListWidgetDefinition(requests []SLOListWidgetRequest, typeVar SLOListWidgetDefinitionType) *SLOListWidgetDefinition {
	this := SLOListWidgetDefinition{}
	this.Requests = requests
	this.Type = typeVar
	return &this
}

// NewSLOListWidgetDefinitionWithDefaults instantiates a new SLOListWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSLOListWidgetDefinitionWithDefaults() *SLOListWidgetDefinition {
	this := SLOListWidgetDefinition{}
	var typeVar SLOListWidgetDefinitionType = SLOLISTWIDGETDEFINITIONTYPE_SLO_LIST
	this.Type = typeVar
	return &this
}

// GetRequests returns the Requests field value.
func (o *SLOListWidgetDefinition) GetRequests() []SLOListWidgetRequest {
	if o == nil {
		var ret []SLOListWidgetRequest
		return ret
	}
	return o.Requests
}

// GetRequestsOk returns a tuple with the Requests field value
// and a boolean to check if the value has been set.
func (o *SLOListWidgetDefinition) GetRequestsOk() (*[]SLOListWidgetRequest, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Requests, true
}

// SetRequests sets field value.
func (o *SLOListWidgetDefinition) SetRequests(v []SLOListWidgetRequest) {
	o.Requests = v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *SLOListWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOListWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *SLOListWidgetDefinition) HasTitle() bool {
	return o != nil && o.Title != nil
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *SLOListWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *SLOListWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOListWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *SLOListWidgetDefinition) HasTitleAlign() bool {
	return o != nil && o.TitleAlign != nil
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *SLOListWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *SLOListWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SLOListWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *SLOListWidgetDefinition) HasTitleSize() bool {
	return o != nil && o.TitleSize != nil
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *SLOListWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *SLOListWidgetDefinition) GetType() SLOListWidgetDefinitionType {
	if o == nil {
		var ret SLOListWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *SLOListWidgetDefinition) GetTypeOk() (*SLOListWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *SLOListWidgetDefinition) SetType(v SLOListWidgetDefinitionType) {
	o.Type = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SLOListWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["requests"] = o.Requests
	if o.Title != nil {
		toSerialize["title"] = o.Title
	}
	if o.TitleAlign != nil {
		toSerialize["title_align"] = o.TitleAlign
	}
	if o.TitleSize != nil {
		toSerialize["title_size"] = o.TitleSize
	}
	toSerialize["type"] = o.Type

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SLOListWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Requests *[]SLOListWidgetRequest      `json:"requests"`
		Type     *SLOListWidgetDefinitionType `json:"type"`
	}{}
	all := struct {
		Requests   []SLOListWidgetRequest      `json:"requests"`
		Title      *string                     `json:"title,omitempty"`
		TitleAlign *WidgetTextAlign            `json:"title_align,omitempty"`
		TitleSize  *string                     `json:"title_size,omitempty"`
		Type       SLOListWidgetDefinitionType `json:"type"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Requests == nil {
		return fmt.Errorf("required field requests missing")
	}
	if required.Type == nil {
		return fmt.Errorf("required field type missing")
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
	if v := all.TitleAlign; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.Type; !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Requests = all.Requests
	o.Title = all.Title
	o.TitleAlign = all.TitleAlign
	o.TitleSize = all.TitleSize
	o.Type = all.Type
	return nil
}
