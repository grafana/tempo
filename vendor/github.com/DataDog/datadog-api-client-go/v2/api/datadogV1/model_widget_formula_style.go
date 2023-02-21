// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// WidgetFormulaStyle Styling options for widget formulas.
type WidgetFormulaStyle struct {
	// The color palette used to display the formula. A guide to the available color palettes can be found at https://docs.datadoghq.com/dashboards/guide/widget_colors
	Palette *string `json:"palette,omitempty"`
	// Index specifying which color to use within the palette.
	PaletteIndex *int64 `json:"palette_index,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewWidgetFormulaStyle instantiates a new WidgetFormulaStyle object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewWidgetFormulaStyle() *WidgetFormulaStyle {
	this := WidgetFormulaStyle{}
	return &this
}

// NewWidgetFormulaStyleWithDefaults instantiates a new WidgetFormulaStyle object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewWidgetFormulaStyleWithDefaults() *WidgetFormulaStyle {
	this := WidgetFormulaStyle{}
	return &this
}

// GetPalette returns the Palette field value if set, zero value otherwise.
func (o *WidgetFormulaStyle) GetPalette() string {
	if o == nil || o.Palette == nil {
		var ret string
		return ret
	}
	return *o.Palette
}

// GetPaletteOk returns a tuple with the Palette field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetFormulaStyle) GetPaletteOk() (*string, bool) {
	if o == nil || o.Palette == nil {
		return nil, false
	}
	return o.Palette, true
}

// HasPalette returns a boolean if a field has been set.
func (o *WidgetFormulaStyle) HasPalette() bool {
	return o != nil && o.Palette != nil
}

// SetPalette gets a reference to the given string and assigns it to the Palette field.
func (o *WidgetFormulaStyle) SetPalette(v string) {
	o.Palette = &v
}

// GetPaletteIndex returns the PaletteIndex field value if set, zero value otherwise.
func (o *WidgetFormulaStyle) GetPaletteIndex() int64 {
	if o == nil || o.PaletteIndex == nil {
		var ret int64
		return ret
	}
	return *o.PaletteIndex
}

// GetPaletteIndexOk returns a tuple with the PaletteIndex field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *WidgetFormulaStyle) GetPaletteIndexOk() (*int64, bool) {
	if o == nil || o.PaletteIndex == nil {
		return nil, false
	}
	return o.PaletteIndex, true
}

// HasPaletteIndex returns a boolean if a field has been set.
func (o *WidgetFormulaStyle) HasPaletteIndex() bool {
	return o != nil && o.PaletteIndex != nil
}

// SetPaletteIndex gets a reference to the given int64 and assigns it to the PaletteIndex field.
func (o *WidgetFormulaStyle) SetPaletteIndex(v int64) {
	o.PaletteIndex = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o WidgetFormulaStyle) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Palette != nil {
		toSerialize["palette"] = o.Palette
	}
	if o.PaletteIndex != nil {
		toSerialize["palette_index"] = o.PaletteIndex
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *WidgetFormulaStyle) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Palette      *string `json:"palette,omitempty"`
		PaletteIndex *int64  `json:"palette_index,omitempty"`
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
	o.Palette = all.Palette
	o.PaletteIndex = all.PaletteIndex
	return nil
}
