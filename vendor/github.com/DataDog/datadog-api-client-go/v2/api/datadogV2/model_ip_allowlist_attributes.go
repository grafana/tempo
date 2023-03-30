// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// IPAllowlistAttributes Attributes of the IP allowlist.
type IPAllowlistAttributes struct {
	// Whether the IP allowlist logic is enabled or not.
	Enabled *bool `json:"enabled,omitempty"`
	// Array of entries in the IP allowlist.
	Entries []IPAllowlistEntry `json:"entries,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewIPAllowlistAttributes instantiates a new IPAllowlistAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewIPAllowlistAttributes() *IPAllowlistAttributes {
	this := IPAllowlistAttributes{}
	return &this
}

// NewIPAllowlistAttributesWithDefaults instantiates a new IPAllowlistAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewIPAllowlistAttributesWithDefaults() *IPAllowlistAttributes {
	this := IPAllowlistAttributes{}
	return &this
}

// GetEnabled returns the Enabled field value if set, zero value otherwise.
func (o *IPAllowlistAttributes) GetEnabled() bool {
	if o == nil || o.Enabled == nil {
		var ret bool
		return ret
	}
	return *o.Enabled
}

// GetEnabledOk returns a tuple with the Enabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistAttributes) GetEnabledOk() (*bool, bool) {
	if o == nil || o.Enabled == nil {
		return nil, false
	}
	return o.Enabled, true
}

// HasEnabled returns a boolean if a field has been set.
func (o *IPAllowlistAttributes) HasEnabled() bool {
	return o != nil && o.Enabled != nil
}

// SetEnabled gets a reference to the given bool and assigns it to the Enabled field.
func (o *IPAllowlistAttributes) SetEnabled(v bool) {
	o.Enabled = &v
}

// GetEntries returns the Entries field value if set, zero value otherwise.
func (o *IPAllowlistAttributes) GetEntries() []IPAllowlistEntry {
	if o == nil || o.Entries == nil {
		var ret []IPAllowlistEntry
		return ret
	}
	return o.Entries
}

// GetEntriesOk returns a tuple with the Entries field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IPAllowlistAttributes) GetEntriesOk() (*[]IPAllowlistEntry, bool) {
	if o == nil || o.Entries == nil {
		return nil, false
	}
	return &o.Entries, true
}

// HasEntries returns a boolean if a field has been set.
func (o *IPAllowlistAttributes) HasEntries() bool {
	return o != nil && o.Entries != nil
}

// SetEntries gets a reference to the given []IPAllowlistEntry and assigns it to the Entries field.
func (o *IPAllowlistAttributes) SetEntries(v []IPAllowlistEntry) {
	o.Entries = v
}

// MarshalJSON serializes the struct using spec logic.
func (o IPAllowlistAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Enabled != nil {
		toSerialize["enabled"] = o.Enabled
	}
	if o.Entries != nil {
		toSerialize["entries"] = o.Entries
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *IPAllowlistAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Enabled *bool              `json:"enabled,omitempty"`
		Entries []IPAllowlistEntry `json:"entries,omitempty"`
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
	o.Enabled = all.Enabled
	o.Entries = all.Entries
	return nil
}
