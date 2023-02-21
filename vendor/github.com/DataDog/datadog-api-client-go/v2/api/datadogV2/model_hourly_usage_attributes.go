// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// HourlyUsageAttributes Attributes of hourly usage for a product family for an org for a time period.
type HourlyUsageAttributes struct {
	// List of the measured usage values for the product family for the org for the time period.
	Measurements []HourlyUsageMeasurement `json:"measurements,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The product for which usage is being reported.
	ProductFamily *string `json:"product_family,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// The region of the Datadog instance that the organization belongs to.
	Region *string `json:"region,omitempty"`
	// Datetime in ISO-8601 format, UTC. The hour for the usage.
	Timestamp *time.Time `json:"timestamp,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewHourlyUsageAttributes instantiates a new HourlyUsageAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewHourlyUsageAttributes() *HourlyUsageAttributes {
	this := HourlyUsageAttributes{}
	return &this
}

// NewHourlyUsageAttributesWithDefaults instantiates a new HourlyUsageAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewHourlyUsageAttributesWithDefaults() *HourlyUsageAttributes {
	this := HourlyUsageAttributes{}
	return &this
}

// GetMeasurements returns the Measurements field value if set, zero value otherwise.
func (o *HourlyUsageAttributes) GetMeasurements() []HourlyUsageMeasurement {
	if o == nil || o.Measurements == nil {
		var ret []HourlyUsageMeasurement
		return ret
	}
	return o.Measurements
}

// GetMeasurementsOk returns a tuple with the Measurements field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributes) GetMeasurementsOk() (*[]HourlyUsageMeasurement, bool) {
	if o == nil || o.Measurements == nil {
		return nil, false
	}
	return &o.Measurements, true
}

// HasMeasurements returns a boolean if a field has been set.
func (o *HourlyUsageAttributes) HasMeasurements() bool {
	return o != nil && o.Measurements != nil
}

// SetMeasurements gets a reference to the given []HourlyUsageMeasurement and assigns it to the Measurements field.
func (o *HourlyUsageAttributes) SetMeasurements(v []HourlyUsageMeasurement) {
	o.Measurements = v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *HourlyUsageAttributes) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributes) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *HourlyUsageAttributes) HasOrgName() bool {
	return o != nil && o.OrgName != nil
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *HourlyUsageAttributes) SetOrgName(v string) {
	o.OrgName = &v
}

// GetProductFamily returns the ProductFamily field value if set, zero value otherwise.
func (o *HourlyUsageAttributes) GetProductFamily() string {
	if o == nil || o.ProductFamily == nil {
		var ret string
		return ret
	}
	return *o.ProductFamily
}

// GetProductFamilyOk returns a tuple with the ProductFamily field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributes) GetProductFamilyOk() (*string, bool) {
	if o == nil || o.ProductFamily == nil {
		return nil, false
	}
	return o.ProductFamily, true
}

// HasProductFamily returns a boolean if a field has been set.
func (o *HourlyUsageAttributes) HasProductFamily() bool {
	return o != nil && o.ProductFamily != nil
}

// SetProductFamily gets a reference to the given string and assigns it to the ProductFamily field.
func (o *HourlyUsageAttributes) SetProductFamily(v string) {
	o.ProductFamily = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *HourlyUsageAttributes) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributes) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *HourlyUsageAttributes) HasPublicId() bool {
	return o != nil && o.PublicId != nil
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *HourlyUsageAttributes) SetPublicId(v string) {
	o.PublicId = &v
}

// GetRegion returns the Region field value if set, zero value otherwise.
func (o *HourlyUsageAttributes) GetRegion() string {
	if o == nil || o.Region == nil {
		var ret string
		return ret
	}
	return *o.Region
}

// GetRegionOk returns a tuple with the Region field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributes) GetRegionOk() (*string, bool) {
	if o == nil || o.Region == nil {
		return nil, false
	}
	return o.Region, true
}

// HasRegion returns a boolean if a field has been set.
func (o *HourlyUsageAttributes) HasRegion() bool {
	return o != nil && o.Region != nil
}

// SetRegion gets a reference to the given string and assigns it to the Region field.
func (o *HourlyUsageAttributes) SetRegion(v string) {
	o.Region = &v
}

// GetTimestamp returns the Timestamp field value if set, zero value otherwise.
func (o *HourlyUsageAttributes) GetTimestamp() time.Time {
	if o == nil || o.Timestamp == nil {
		var ret time.Time
		return ret
	}
	return *o.Timestamp
}

// GetTimestampOk returns a tuple with the Timestamp field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *HourlyUsageAttributes) GetTimestampOk() (*time.Time, bool) {
	if o == nil || o.Timestamp == nil {
		return nil, false
	}
	return o.Timestamp, true
}

// HasTimestamp returns a boolean if a field has been set.
func (o *HourlyUsageAttributes) HasTimestamp() bool {
	return o != nil && o.Timestamp != nil
}

// SetTimestamp gets a reference to the given time.Time and assigns it to the Timestamp field.
func (o *HourlyUsageAttributes) SetTimestamp(v time.Time) {
	o.Timestamp = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o HourlyUsageAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Measurements != nil {
		toSerialize["measurements"] = o.Measurements
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.ProductFamily != nil {
		toSerialize["product_family"] = o.ProductFamily
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.Region != nil {
		toSerialize["region"] = o.Region
	}
	if o.Timestamp != nil {
		if o.Timestamp.Nanosecond() == 0 {
			toSerialize["timestamp"] = o.Timestamp.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["timestamp"] = o.Timestamp.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *HourlyUsageAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Measurements  []HourlyUsageMeasurement `json:"measurements,omitempty"`
		OrgName       *string                  `json:"org_name,omitempty"`
		ProductFamily *string                  `json:"product_family,omitempty"`
		PublicId      *string                  `json:"public_id,omitempty"`
		Region        *string                  `json:"region,omitempty"`
		Timestamp     *time.Time               `json:"timestamp,omitempty"`
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
	o.Measurements = all.Measurements
	o.OrgName = all.OrgName
	o.ProductFamily = all.ProductFamily
	o.PublicId = all.PublicId
	o.Region = all.Region
	o.Timestamp = all.Timestamp
	return nil
}
