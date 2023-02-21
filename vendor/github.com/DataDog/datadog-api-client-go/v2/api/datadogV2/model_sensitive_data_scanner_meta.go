// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// SensitiveDataScannerMeta Meta response containing information about the API.
type SensitiveDataScannerMeta struct {
	// Maximum number of scanning rules allowed for the org.
	CountLimit *int64 `json:"count_limit,omitempty"`
	// Maximum number of scanning groups allowed for the org.
	GroupCountLimit *int64 `json:"group_count_limit,omitempty"`
	// Whether or not scanned events are highlighted in Logs or RUM for the org.
	HasHighlightEnabled *bool `json:"has_highlight_enabled,omitempty"`
	// Whether or not the org is compliant to the payment card industry standard.
	IsPciCompliant *bool `json:"is_pci_compliant,omitempty"`
	// Version of the API.
	Version *int64 `json:"version,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSensitiveDataScannerMeta instantiates a new SensitiveDataScannerMeta object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSensitiveDataScannerMeta() *SensitiveDataScannerMeta {
	this := SensitiveDataScannerMeta{}
	return &this
}

// NewSensitiveDataScannerMetaWithDefaults instantiates a new SensitiveDataScannerMeta object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSensitiveDataScannerMetaWithDefaults() *SensitiveDataScannerMeta {
	this := SensitiveDataScannerMeta{}
	return &this
}

// GetCountLimit returns the CountLimit field value if set, zero value otherwise.
func (o *SensitiveDataScannerMeta) GetCountLimit() int64 {
	if o == nil || o.CountLimit == nil {
		var ret int64
		return ret
	}
	return *o.CountLimit
}

// GetCountLimitOk returns a tuple with the CountLimit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerMeta) GetCountLimitOk() (*int64, bool) {
	if o == nil || o.CountLimit == nil {
		return nil, false
	}
	return o.CountLimit, true
}

// HasCountLimit returns a boolean if a field has been set.
func (o *SensitiveDataScannerMeta) HasCountLimit() bool {
	return o != nil && o.CountLimit != nil
}

// SetCountLimit gets a reference to the given int64 and assigns it to the CountLimit field.
func (o *SensitiveDataScannerMeta) SetCountLimit(v int64) {
	o.CountLimit = &v
}

// GetGroupCountLimit returns the GroupCountLimit field value if set, zero value otherwise.
func (o *SensitiveDataScannerMeta) GetGroupCountLimit() int64 {
	if o == nil || o.GroupCountLimit == nil {
		var ret int64
		return ret
	}
	return *o.GroupCountLimit
}

// GetGroupCountLimitOk returns a tuple with the GroupCountLimit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerMeta) GetGroupCountLimitOk() (*int64, bool) {
	if o == nil || o.GroupCountLimit == nil {
		return nil, false
	}
	return o.GroupCountLimit, true
}

// HasGroupCountLimit returns a boolean if a field has been set.
func (o *SensitiveDataScannerMeta) HasGroupCountLimit() bool {
	return o != nil && o.GroupCountLimit != nil
}

// SetGroupCountLimit gets a reference to the given int64 and assigns it to the GroupCountLimit field.
func (o *SensitiveDataScannerMeta) SetGroupCountLimit(v int64) {
	o.GroupCountLimit = &v
}

// GetHasHighlightEnabled returns the HasHighlightEnabled field value if set, zero value otherwise.
func (o *SensitiveDataScannerMeta) GetHasHighlightEnabled() bool {
	if o == nil || o.HasHighlightEnabled == nil {
		var ret bool
		return ret
	}
	return *o.HasHighlightEnabled
}

// GetHasHighlightEnabledOk returns a tuple with the HasHighlightEnabled field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerMeta) GetHasHighlightEnabledOk() (*bool, bool) {
	if o == nil || o.HasHighlightEnabled == nil {
		return nil, false
	}
	return o.HasHighlightEnabled, true
}

// HasHasHighlightEnabled returns a boolean if a field has been set.
func (o *SensitiveDataScannerMeta) HasHasHighlightEnabled() bool {
	return o != nil && o.HasHighlightEnabled != nil
}

// SetHasHighlightEnabled gets a reference to the given bool and assigns it to the HasHighlightEnabled field.
func (o *SensitiveDataScannerMeta) SetHasHighlightEnabled(v bool) {
	o.HasHighlightEnabled = &v
}

// GetIsPciCompliant returns the IsPciCompliant field value if set, zero value otherwise.
func (o *SensitiveDataScannerMeta) GetIsPciCompliant() bool {
	if o == nil || o.IsPciCompliant == nil {
		var ret bool
		return ret
	}
	return *o.IsPciCompliant
}

// GetIsPciCompliantOk returns a tuple with the IsPciCompliant field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerMeta) GetIsPciCompliantOk() (*bool, bool) {
	if o == nil || o.IsPciCompliant == nil {
		return nil, false
	}
	return o.IsPciCompliant, true
}

// HasIsPciCompliant returns a boolean if a field has been set.
func (o *SensitiveDataScannerMeta) HasIsPciCompliant() bool {
	return o != nil && o.IsPciCompliant != nil
}

// SetIsPciCompliant gets a reference to the given bool and assigns it to the IsPciCompliant field.
func (o *SensitiveDataScannerMeta) SetIsPciCompliant(v bool) {
	o.IsPciCompliant = &v
}

// GetVersion returns the Version field value if set, zero value otherwise.
func (o *SensitiveDataScannerMeta) GetVersion() int64 {
	if o == nil || o.Version == nil {
		var ret int64
		return ret
	}
	return *o.Version
}

// GetVersionOk returns a tuple with the Version field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SensitiveDataScannerMeta) GetVersionOk() (*int64, bool) {
	if o == nil || o.Version == nil {
		return nil, false
	}
	return o.Version, true
}

// HasVersion returns a boolean if a field has been set.
func (o *SensitiveDataScannerMeta) HasVersion() bool {
	return o != nil && o.Version != nil
}

// SetVersion gets a reference to the given int64 and assigns it to the Version field.
func (o *SensitiveDataScannerMeta) SetVersion(v int64) {
	o.Version = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SensitiveDataScannerMeta) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.CountLimit != nil {
		toSerialize["count_limit"] = o.CountLimit
	}
	if o.GroupCountLimit != nil {
		toSerialize["group_count_limit"] = o.GroupCountLimit
	}
	if o.HasHighlightEnabled != nil {
		toSerialize["has_highlight_enabled"] = o.HasHighlightEnabled
	}
	if o.IsPciCompliant != nil {
		toSerialize["is_pci_compliant"] = o.IsPciCompliant
	}
	if o.Version != nil {
		toSerialize["version"] = o.Version
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SensitiveDataScannerMeta) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		CountLimit          *int64 `json:"count_limit,omitempty"`
		GroupCountLimit     *int64 `json:"group_count_limit,omitempty"`
		HasHighlightEnabled *bool  `json:"has_highlight_enabled,omitempty"`
		IsPciCompliant      *bool  `json:"is_pci_compliant,omitempty"`
		Version             *int64 `json:"version,omitempty"`
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
	o.CountLimit = all.CountLimit
	o.GroupCountLimit = all.GroupCountLimit
	o.HasHighlightEnabled = all.HasHighlightEnabled
	o.IsPciCompliant = all.IsPciCompliant
	o.Version = all.Version
	return nil
}
