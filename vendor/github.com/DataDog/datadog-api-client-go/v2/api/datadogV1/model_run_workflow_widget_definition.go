// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"fmt"
)

// RunWorkflowWidgetDefinition Run workflow is widget that allows you to run a workflow from a dashboard.
type RunWorkflowWidgetDefinition struct {
	// List of custom links.
	CustomLinks []WidgetCustomLink `json:"custom_links,omitempty"`
	// Array of workflow inputs to map to dashboard template variables.
	Inputs []RunWorkflowWidgetInput `json:"inputs,omitempty"`
	// Time setting for the widget.
	Time *WidgetTime `json:"time,omitempty"`
	// Title of your widget.
	Title *string `json:"title,omitempty"`
	// How to align the text on the widget.
	TitleAlign *WidgetTextAlign `json:"title_align,omitempty"`
	// Size of the title.
	TitleSize *string `json:"title_size,omitempty"`
	// Type of the run workflow widget.
	Type RunWorkflowWidgetDefinitionType `json:"type"`
	// Workflow id.
	WorkflowId string `json:"workflow_id"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewRunWorkflowWidgetDefinition instantiates a new RunWorkflowWidgetDefinition object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewRunWorkflowWidgetDefinition(typeVar RunWorkflowWidgetDefinitionType, workflowId string) *RunWorkflowWidgetDefinition {
	this := RunWorkflowWidgetDefinition{}
	this.Type = typeVar
	this.WorkflowId = workflowId
	return &this
}

// NewRunWorkflowWidgetDefinitionWithDefaults instantiates a new RunWorkflowWidgetDefinition object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewRunWorkflowWidgetDefinitionWithDefaults() *RunWorkflowWidgetDefinition {
	this := RunWorkflowWidgetDefinition{}
	var typeVar RunWorkflowWidgetDefinitionType = RUNWORKFLOWWIDGETDEFINITIONTYPE_RUN_WORKFLOW
	this.Type = typeVar
	return &this
}

// GetCustomLinks returns the CustomLinks field value if set, zero value otherwise.
func (o *RunWorkflowWidgetDefinition) GetCustomLinks() []WidgetCustomLink {
	if o == nil || o.CustomLinks == nil {
		var ret []WidgetCustomLink
		return ret
	}
	return o.CustomLinks
}

// GetCustomLinksOk returns a tuple with the CustomLinks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetDefinition) GetCustomLinksOk() (*[]WidgetCustomLink, bool) {
	if o == nil || o.CustomLinks == nil {
		return nil, false
	}
	return &o.CustomLinks, true
}

// HasCustomLinks returns a boolean if a field has been set.
func (o *RunWorkflowWidgetDefinition) HasCustomLinks() bool {
	return o != nil && o.CustomLinks != nil
}

// SetCustomLinks gets a reference to the given []WidgetCustomLink and assigns it to the CustomLinks field.
func (o *RunWorkflowWidgetDefinition) SetCustomLinks(v []WidgetCustomLink) {
	o.CustomLinks = v
}

// GetInputs returns the Inputs field value if set, zero value otherwise.
func (o *RunWorkflowWidgetDefinition) GetInputs() []RunWorkflowWidgetInput {
	if o == nil || o.Inputs == nil {
		var ret []RunWorkflowWidgetInput
		return ret
	}
	return o.Inputs
}

// GetInputsOk returns a tuple with the Inputs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetDefinition) GetInputsOk() (*[]RunWorkflowWidgetInput, bool) {
	if o == nil || o.Inputs == nil {
		return nil, false
	}
	return &o.Inputs, true
}

// HasInputs returns a boolean if a field has been set.
func (o *RunWorkflowWidgetDefinition) HasInputs() bool {
	return o != nil && o.Inputs != nil
}

// SetInputs gets a reference to the given []RunWorkflowWidgetInput and assigns it to the Inputs field.
func (o *RunWorkflowWidgetDefinition) SetInputs(v []RunWorkflowWidgetInput) {
	o.Inputs = v
}

// GetTime returns the Time field value if set, zero value otherwise.
func (o *RunWorkflowWidgetDefinition) GetTime() WidgetTime {
	if o == nil || o.Time == nil {
		var ret WidgetTime
		return ret
	}
	return *o.Time
}

// GetTimeOk returns a tuple with the Time field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetDefinition) GetTimeOk() (*WidgetTime, bool) {
	if o == nil || o.Time == nil {
		return nil, false
	}
	return o.Time, true
}

// HasTime returns a boolean if a field has been set.
func (o *RunWorkflowWidgetDefinition) HasTime() bool {
	return o != nil && o.Time != nil
}

// SetTime gets a reference to the given WidgetTime and assigns it to the Time field.
func (o *RunWorkflowWidgetDefinition) SetTime(v WidgetTime) {
	o.Time = &v
}

// GetTitle returns the Title field value if set, zero value otherwise.
func (o *RunWorkflowWidgetDefinition) GetTitle() string {
	if o == nil || o.Title == nil {
		var ret string
		return ret
	}
	return *o.Title
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetDefinition) GetTitleOk() (*string, bool) {
	if o == nil || o.Title == nil {
		return nil, false
	}
	return o.Title, true
}

// HasTitle returns a boolean if a field has been set.
func (o *RunWorkflowWidgetDefinition) HasTitle() bool {
	return o != nil && o.Title != nil
}

// SetTitle gets a reference to the given string and assigns it to the Title field.
func (o *RunWorkflowWidgetDefinition) SetTitle(v string) {
	o.Title = &v
}

// GetTitleAlign returns the TitleAlign field value if set, zero value otherwise.
func (o *RunWorkflowWidgetDefinition) GetTitleAlign() WidgetTextAlign {
	if o == nil || o.TitleAlign == nil {
		var ret WidgetTextAlign
		return ret
	}
	return *o.TitleAlign
}

// GetTitleAlignOk returns a tuple with the TitleAlign field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetDefinition) GetTitleAlignOk() (*WidgetTextAlign, bool) {
	if o == nil || o.TitleAlign == nil {
		return nil, false
	}
	return o.TitleAlign, true
}

// HasTitleAlign returns a boolean if a field has been set.
func (o *RunWorkflowWidgetDefinition) HasTitleAlign() bool {
	return o != nil && o.TitleAlign != nil
}

// SetTitleAlign gets a reference to the given WidgetTextAlign and assigns it to the TitleAlign field.
func (o *RunWorkflowWidgetDefinition) SetTitleAlign(v WidgetTextAlign) {
	o.TitleAlign = &v
}

// GetTitleSize returns the TitleSize field value if set, zero value otherwise.
func (o *RunWorkflowWidgetDefinition) GetTitleSize() string {
	if o == nil || o.TitleSize == nil {
		var ret string
		return ret
	}
	return *o.TitleSize
}

// GetTitleSizeOk returns a tuple with the TitleSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetDefinition) GetTitleSizeOk() (*string, bool) {
	if o == nil || o.TitleSize == nil {
		return nil, false
	}
	return o.TitleSize, true
}

// HasTitleSize returns a boolean if a field has been set.
func (o *RunWorkflowWidgetDefinition) HasTitleSize() bool {
	return o != nil && o.TitleSize != nil
}

// SetTitleSize gets a reference to the given string and assigns it to the TitleSize field.
func (o *RunWorkflowWidgetDefinition) SetTitleSize(v string) {
	o.TitleSize = &v
}

// GetType returns the Type field value.
func (o *RunWorkflowWidgetDefinition) GetType() RunWorkflowWidgetDefinitionType {
	if o == nil {
		var ret RunWorkflowWidgetDefinitionType
		return ret
	}
	return o.Type
}

// GetTypeOk returns a tuple with the Type field value
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetDefinition) GetTypeOk() (*RunWorkflowWidgetDefinitionType, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Type, true
}

// SetType sets field value.
func (o *RunWorkflowWidgetDefinition) SetType(v RunWorkflowWidgetDefinitionType) {
	o.Type = v
}

// GetWorkflowId returns the WorkflowId field value.
func (o *RunWorkflowWidgetDefinition) GetWorkflowId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.WorkflowId
}

// GetWorkflowIdOk returns a tuple with the WorkflowId field value
// and a boolean to check if the value has been set.
func (o *RunWorkflowWidgetDefinition) GetWorkflowIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.WorkflowId, true
}

// SetWorkflowId sets field value.
func (o *RunWorkflowWidgetDefinition) SetWorkflowId(v string) {
	o.WorkflowId = v
}

// MarshalJSON serializes the struct using spec logic.
func (o RunWorkflowWidgetDefinition) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CustomLinks != nil {
		toSerialize["custom_links"] = o.CustomLinks
	}
	if o.Inputs != nil {
		toSerialize["inputs"] = o.Inputs
	}
	if o.Time != nil {
		toSerialize["time"] = o.Time
	}
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
	toSerialize["workflow_id"] = o.WorkflowId

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *RunWorkflowWidgetDefinition) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Type       *RunWorkflowWidgetDefinitionType `json:"type"`
		WorkflowId *string                          `json:"workflow_id"`
	}{}
	all := struct {
		CustomLinks []WidgetCustomLink              `json:"custom_links,omitempty"`
		Inputs      []RunWorkflowWidgetInput        `json:"inputs,omitempty"`
		Time        *WidgetTime                     `json:"time,omitempty"`
		Title       *string                         `json:"title,omitempty"`
		TitleAlign  *WidgetTextAlign                `json:"title_align,omitempty"`
		TitleSize   *string                         `json:"title_size,omitempty"`
		Type        RunWorkflowWidgetDefinitionType `json:"type"`
		WorkflowId  string                          `json:"workflow_id"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Type == nil {
		return fmt.Errorf("required field type missing")
	}
	if required.WorkflowId == nil {
		return fmt.Errorf("required field workflow_id missing")
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
	o.CustomLinks = all.CustomLinks
	o.Inputs = all.Inputs
	if all.Time != nil && all.Time.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Time = all.Time
	o.Title = all.Title
	o.TitleAlign = all.TitleAlign
	o.TitleSize = all.TitleSize
	o.Type = all.Type
	o.WorkflowId = all.WorkflowId
	return nil
}
