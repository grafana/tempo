// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"time"
)

// CostByOrgAttributes Cost attributes data.
type CostByOrgAttributes struct {
	// List of charges data reported for the requested month.
	Charges []ChargebackBreakdown `json:"charges,omitempty"`
	// The month requested.
	Date *time.Time `json:"date,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// The region of the Datadog instance that the organization belongs to.
	Region *string `json:"region,omitempty"`
	// The total cost of products for the month.
	TotalCost *float64 `json:"total_cost,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCostByOrgAttributes instantiates a new CostByOrgAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCostByOrgAttributes() *CostByOrgAttributes {
	this := CostByOrgAttributes{}
	return &this
}

// NewCostByOrgAttributesWithDefaults instantiates a new CostByOrgAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCostByOrgAttributesWithDefaults() *CostByOrgAttributes {
	this := CostByOrgAttributes{}
	return &this
}

// GetCharges returns the Charges field value if set, zero value otherwise.
func (o *CostByOrgAttributes) GetCharges() []ChargebackBreakdown {
	if o == nil || o.Charges == nil {
		var ret []ChargebackBreakdown
		return ret
	}
	return o.Charges
}

// GetChargesOk returns a tuple with the Charges field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CostByOrgAttributes) GetChargesOk() (*[]ChargebackBreakdown, bool) {
	if o == nil || o.Charges == nil {
		return nil, false
	}
	return &o.Charges, true
}

// HasCharges returns a boolean if a field has been set.
func (o *CostByOrgAttributes) HasCharges() bool {
	return o != nil && o.Charges != nil
}

// SetCharges gets a reference to the given []ChargebackBreakdown and assigns it to the Charges field.
func (o *CostByOrgAttributes) SetCharges(v []ChargebackBreakdown) {
	o.Charges = v
}

// GetDate returns the Date field value if set, zero value otherwise.
func (o *CostByOrgAttributes) GetDate() time.Time {
	if o == nil || o.Date == nil {
		var ret time.Time
		return ret
	}
	return *o.Date
}

// GetDateOk returns a tuple with the Date field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CostByOrgAttributes) GetDateOk() (*time.Time, bool) {
	if o == nil || o.Date == nil {
		return nil, false
	}
	return o.Date, true
}

// HasDate returns a boolean if a field has been set.
func (o *CostByOrgAttributes) HasDate() bool {
	return o != nil && o.Date != nil
}

// SetDate gets a reference to the given time.Time and assigns it to the Date field.
func (o *CostByOrgAttributes) SetDate(v time.Time) {
	o.Date = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *CostByOrgAttributes) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CostByOrgAttributes) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *CostByOrgAttributes) HasOrgName() bool {
	return o != nil && o.OrgName != nil
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *CostByOrgAttributes) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *CostByOrgAttributes) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CostByOrgAttributes) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *CostByOrgAttributes) HasPublicId() bool {
	return o != nil && o.PublicId != nil
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *CostByOrgAttributes) SetPublicId(v string) {
	o.PublicId = &v
}

// GetRegion returns the Region field value if set, zero value otherwise.
func (o *CostByOrgAttributes) GetRegion() string {
	if o == nil || o.Region == nil {
		var ret string
		return ret
	}
	return *o.Region
}

// GetRegionOk returns a tuple with the Region field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CostByOrgAttributes) GetRegionOk() (*string, bool) {
	if o == nil || o.Region == nil {
		return nil, false
	}
	return o.Region, true
}

// HasRegion returns a boolean if a field has been set.
func (o *CostByOrgAttributes) HasRegion() bool {
	return o != nil && o.Region != nil
}

// SetRegion gets a reference to the given string and assigns it to the Region field.
func (o *CostByOrgAttributes) SetRegion(v string) {
	o.Region = &v
}

// GetTotalCost returns the TotalCost field value if set, zero value otherwise.
func (o *CostByOrgAttributes) GetTotalCost() float64 {
	if o == nil || o.TotalCost == nil {
		var ret float64
		return ret
	}
	return *o.TotalCost
}

// GetTotalCostOk returns a tuple with the TotalCost field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CostByOrgAttributes) GetTotalCostOk() (*float64, bool) {
	if o == nil || o.TotalCost == nil {
		return nil, false
	}
	return o.TotalCost, true
}

// HasTotalCost returns a boolean if a field has been set.
func (o *CostByOrgAttributes) HasTotalCost() bool {
	return o != nil && o.TotalCost != nil
}

// SetTotalCost gets a reference to the given float64 and assigns it to the TotalCost field.
func (o *CostByOrgAttributes) SetTotalCost(v float64) {
	o.TotalCost = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o CostByOrgAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Charges != nil {
		toSerialize["charges"] = o.Charges
	}
	if o.Date != nil {
		if o.Date.Nanosecond() == 0 {
			toSerialize["date"] = o.Date.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["date"] = o.Date.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.Region != nil {
		toSerialize["region"] = o.Region
	}
	if o.TotalCost != nil {
		toSerialize["total_cost"] = o.TotalCost
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CostByOrgAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Charges   []ChargebackBreakdown `json:"charges,omitempty"`
		Date      *time.Time            `json:"date,omitempty"`
		OrgName   *string               `json:"org_name,omitempty"`
		PublicId  *string               `json:"public_id,omitempty"`
		Region    *string               `json:"region,omitempty"`
		TotalCost *float64              `json:"total_cost,omitempty"`
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
	o.Charges = all.Charges
	o.Date = all.Date
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	o.Region = all.Region
	o.TotalCost = all.TotalCost
	return nil
}
