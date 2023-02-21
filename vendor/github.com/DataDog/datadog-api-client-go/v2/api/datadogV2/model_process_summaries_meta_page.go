// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// ProcessSummariesMetaPage Paging attributes.
type ProcessSummariesMetaPage struct {
	// The cursor used to get the next results, if any. To make the next request, use the same
	// parameters with the addition of the `page[cursor]`.
	After *string `json:"after,omitempty"`
	// Number of results returned.
	Size *int32 `json:"size,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewProcessSummariesMetaPage instantiates a new ProcessSummariesMetaPage object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewProcessSummariesMetaPage() *ProcessSummariesMetaPage {
	this := ProcessSummariesMetaPage{}
	return &this
}

// NewProcessSummariesMetaPageWithDefaults instantiates a new ProcessSummariesMetaPage object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewProcessSummariesMetaPageWithDefaults() *ProcessSummariesMetaPage {
	this := ProcessSummariesMetaPage{}
	return &this
}

// GetAfter returns the After field value if set, zero value otherwise.
func (o *ProcessSummariesMetaPage) GetAfter() string {
	if o == nil || o.After == nil {
		var ret string
		return ret
	}
	return *o.After
}

// GetAfterOk returns a tuple with the After field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummariesMetaPage) GetAfterOk() (*string, bool) {
	if o == nil || o.After == nil {
		return nil, false
	}
	return o.After, true
}

// HasAfter returns a boolean if a field has been set.
func (o *ProcessSummariesMetaPage) HasAfter() bool {
	return o != nil && o.After != nil
}

// SetAfter gets a reference to the given string and assigns it to the After field.
func (o *ProcessSummariesMetaPage) SetAfter(v string) {
	o.After = &v
}

// GetSize returns the Size field value if set, zero value otherwise.
func (o *ProcessSummariesMetaPage) GetSize() int32 {
	if o == nil || o.Size == nil {
		var ret int32
		return ret
	}
	return *o.Size
}

// GetSizeOk returns a tuple with the Size field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ProcessSummariesMetaPage) GetSizeOk() (*int32, bool) {
	if o == nil || o.Size == nil {
		return nil, false
	}
	return o.Size, true
}

// HasSize returns a boolean if a field has been set.
func (o *ProcessSummariesMetaPage) HasSize() bool {
	return o != nil && o.Size != nil
}

// SetSize gets a reference to the given int32 and assigns it to the Size field.
func (o *ProcessSummariesMetaPage) SetSize(v int32) {
	o.Size = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o ProcessSummariesMetaPage) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.After != nil {
		toSerialize["after"] = o.After
	}
	if o.Size != nil {
		toSerialize["size"] = o.Size
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *ProcessSummariesMetaPage) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		After *string `json:"after,omitempty"`
		Size  *int32  `json:"size,omitempty"`
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
	o.After = all.After
	o.Size = all.Size
	return nil
}
