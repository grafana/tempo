// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// SecurityMonitoringSignalListRequestFilter Search filters for listing security signals.
type SecurityMonitoringSignalListRequestFilter struct {
	// The minimum timestamp for requested security signals.
	From *time.Time `json:"from,omitempty"`
	// Search query for listing security signals.
	Query *string `json:"query,omitempty"`
	// The maximum timestamp for requested security signals.
	To *time.Time `json:"to,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSecurityMonitoringSignalListRequestFilter instantiates a new SecurityMonitoringSignalListRequestFilter object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSecurityMonitoringSignalListRequestFilter() *SecurityMonitoringSignalListRequestFilter {
	this := SecurityMonitoringSignalListRequestFilter{}
	return &this
}

// NewSecurityMonitoringSignalListRequestFilterWithDefaults instantiates a new SecurityMonitoringSignalListRequestFilter object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSecurityMonitoringSignalListRequestFilterWithDefaults() *SecurityMonitoringSignalListRequestFilter {
	this := SecurityMonitoringSignalListRequestFilter{}
	return &this
}

// GetFrom returns the From field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalListRequestFilter) GetFrom() time.Time {
	if o == nil || o.From == nil {
		var ret time.Time
		return ret
	}
	return *o.From
}

// GetFromOk returns a tuple with the From field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalListRequestFilter) GetFromOk() (*time.Time, bool) {
	if o == nil || o.From == nil {
		return nil, false
	}
	return o.From, true
}

// HasFrom returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalListRequestFilter) HasFrom() bool {
	return o != nil && o.From != nil
}

// SetFrom gets a reference to the given time.Time and assigns it to the From field.
func (o *SecurityMonitoringSignalListRequestFilter) SetFrom(v time.Time) {
	o.From = &v
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalListRequestFilter) GetQuery() string {
	if o == nil || o.Query == nil {
		var ret string
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalListRequestFilter) GetQueryOk() (*string, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalListRequestFilter) HasQuery() bool {
	return o != nil && o.Query != nil
}

// SetQuery gets a reference to the given string and assigns it to the Query field.
func (o *SecurityMonitoringSignalListRequestFilter) SetQuery(v string) {
	o.Query = &v
}

// GetTo returns the To field value if set, zero value otherwise.
func (o *SecurityMonitoringSignalListRequestFilter) GetTo() time.Time {
	if o == nil || o.To == nil {
		var ret time.Time
		return ret
	}
	return *o.To
}

// GetToOk returns a tuple with the To field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SecurityMonitoringSignalListRequestFilter) GetToOk() (*time.Time, bool) {
	if o == nil || o.To == nil {
		return nil, false
	}
	return o.To, true
}

// HasTo returns a boolean if a field has been set.
func (o *SecurityMonitoringSignalListRequestFilter) HasTo() bool {
	return o != nil && o.To != nil
}

// SetTo gets a reference to the given time.Time and assigns it to the To field.
func (o *SecurityMonitoringSignalListRequestFilter) SetTo(v time.Time) {
	o.To = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SecurityMonitoringSignalListRequestFilter) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.From != nil {
		if o.From.Nanosecond() == 0 {
			toSerialize["from"] = o.From.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["from"] = o.From.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.To != nil {
		if o.To.Nanosecond() == 0 {
			toSerialize["to"] = o.To.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["to"] = o.To.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SecurityMonitoringSignalListRequestFilter) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		From  *time.Time `json:"from,omitempty"`
		Query *string    `json:"query,omitempty"`
		To    *time.Time `json:"to,omitempty"`
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
	o.From = all.From
	o.Query = all.Query
	o.To = all.To
	return nil
}
