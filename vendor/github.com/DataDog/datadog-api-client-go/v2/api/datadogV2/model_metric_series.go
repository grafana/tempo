// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// MetricSeries A metric to submit to Datadog.
// See [Datadog metrics](https://docs.datadoghq.com/developers/metrics/#custom-metrics-properties).
type MetricSeries struct {
	// If the type of the metric is rate or count, define the corresponding interval.
	Interval *int64 `json:"interval,omitempty"`
	// Metadata for the metric.
	Metadata *MetricMetadata `json:"metadata,omitempty"`
	// The name of the timeseries.
	Metric string `json:"metric"`
	// Points relating to a metric. All points must be objects with timestamp and a scalar value (cannot be a string). Timestamps should be in POSIX time in seconds, and cannot be more than ten minutes in the future or more than one hour in the past.
	Points []MetricPoint `json:"points"`
	// A list of resources to associate with this metric.
	Resources []MetricResource `json:"resources,omitempty"`
	// The source type name.
	SourceTypeName *string `json:"source_type_name,omitempty"`
	// A list of tags associated with the metric.
	Tags []string `json:"tags,omitempty"`
	// The type of metric. The available types are `0` (unspecified), `1` (count), `2` (rate), and `3` (gauge).
	Type *MetricIntakeType `json:"type,omitempty"`
	// The unit of point value.
	Unit *string `json:"unit,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMetricSeries instantiates a new MetricSeries object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMetricSeries(metric string, points []MetricPoint) *MetricSeries {
	this := MetricSeries{}
	this.Metric = metric
	this.Points = points
	return &this
}

// NewMetricSeriesWithDefaults instantiates a new MetricSeries object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMetricSeriesWithDefaults() *MetricSeries {
	this := MetricSeries{}
	return &this
}

// GetInterval returns the Interval field value if set, zero value otherwise.
func (o *MetricSeries) GetInterval() int64 {
	if o == nil || o.Interval == nil {
		var ret int64
		return ret
	}
	return *o.Interval
}

// GetIntervalOk returns a tuple with the Interval field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSeries) GetIntervalOk() (*int64, bool) {
	if o == nil || o.Interval == nil {
		return nil, false
	}
	return o.Interval, true
}

// HasInterval returns a boolean if a field has been set.
func (o *MetricSeries) HasInterval() bool {
	return o != nil && o.Interval != nil
}

// SetInterval gets a reference to the given int64 and assigns it to the Interval field.
func (o *MetricSeries) SetInterval(v int64) {
	o.Interval = &v
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *MetricSeries) GetMetadata() MetricMetadata {
	if o == nil || o.Metadata == nil {
		var ret MetricMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSeries) GetMetadataOk() (*MetricMetadata, bool) {
	if o == nil || o.Metadata == nil {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *MetricSeries) HasMetadata() bool {
	return o != nil && o.Metadata != nil
}

// SetMetadata gets a reference to the given MetricMetadata and assigns it to the Metadata field.
func (o *MetricSeries) SetMetadata(v MetricMetadata) {
	o.Metadata = &v
}

// GetMetric returns the Metric field value.
func (o *MetricSeries) GetMetric() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Metric
}

// GetMetricOk returns a tuple with the Metric field value
// and a boolean to check if the value has been set.
func (o *MetricSeries) GetMetricOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Metric, true
}

// SetMetric sets field value.
func (o *MetricSeries) SetMetric(v string) {
	o.Metric = v
}

// GetPoints returns the Points field value.
func (o *MetricSeries) GetPoints() []MetricPoint {
	if o == nil {
		var ret []MetricPoint
		return ret
	}
	return o.Points
}

// GetPointsOk returns a tuple with the Points field value
// and a boolean to check if the value has been set.
func (o *MetricSeries) GetPointsOk() (*[]MetricPoint, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Points, true
}

// SetPoints sets field value.
func (o *MetricSeries) SetPoints(v []MetricPoint) {
	o.Points = v
}

// GetResources returns the Resources field value if set, zero value otherwise.
func (o *MetricSeries) GetResources() []MetricResource {
	if o == nil || o.Resources == nil {
		var ret []MetricResource
		return ret
	}
	return o.Resources
}

// GetResourcesOk returns a tuple with the Resources field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSeries) GetResourcesOk() (*[]MetricResource, bool) {
	if o == nil || o.Resources == nil {
		return nil, false
	}
	return &o.Resources, true
}

// HasResources returns a boolean if a field has been set.
func (o *MetricSeries) HasResources() bool {
	return o != nil && o.Resources != nil
}

// SetResources gets a reference to the given []MetricResource and assigns it to the Resources field.
func (o *MetricSeries) SetResources(v []MetricResource) {
	o.Resources = v
}

// GetSourceTypeName returns the SourceTypeName field value if set, zero value otherwise.
func (o *MetricSeries) GetSourceTypeName() string {
	if o == nil || o.SourceTypeName == nil {
		var ret string
		return ret
	}
	return *o.SourceTypeName
}

// GetSourceTypeNameOk returns a tuple with the SourceTypeName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSeries) GetSourceTypeNameOk() (*string, bool) {
	if o == nil || o.SourceTypeName == nil {
		return nil, false
	}
	return o.SourceTypeName, true
}

// HasSourceTypeName returns a boolean if a field has been set.
func (o *MetricSeries) HasSourceTypeName() bool {
	return o != nil && o.SourceTypeName != nil
}

// SetSourceTypeName gets a reference to the given string and assigns it to the SourceTypeName field.
func (o *MetricSeries) SetSourceTypeName(v string) {
	o.SourceTypeName = &v
}

// GetTags returns the Tags field value if set, zero value otherwise.
func (o *MetricSeries) GetTags() []string {
	if o == nil || o.Tags == nil {
		var ret []string
		return ret
	}
	return o.Tags
}

// GetTagsOk returns a tuple with the Tags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSeries) GetTagsOk() (*[]string, bool) {
	if o == nil || o.Tags == nil {
		return nil, false
	}
	return &o.Tags, true
}

// HasTags returns a boolean if a field has been set.
func (o *MetricSeries) HasTags() bool {
	return o != nil && o.Tags != nil
}

// SetTags gets a reference to the given []string and assigns it to the Tags field.
func (o *MetricSeries) SetTags(v []string) {
	o.Tags = v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *MetricSeries) GetType() MetricIntakeType {
	if o == nil || o.Type == nil {
		var ret MetricIntakeType
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSeries) GetTypeOk() (*MetricIntakeType, bool) {
	if o == nil || o.Type == nil {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *MetricSeries) HasType() bool {
	return o != nil && o.Type != nil
}

// SetType gets a reference to the given MetricIntakeType and assigns it to the Type field.
func (o *MetricSeries) SetType(v MetricIntakeType) {
	o.Type = &v
}

// GetUnit returns the Unit field value if set, zero value otherwise.
func (o *MetricSeries) GetUnit() string {
	if o == nil || o.Unit == nil {
		var ret string
		return ret
	}
	return *o.Unit
}

// GetUnitOk returns a tuple with the Unit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MetricSeries) GetUnitOk() (*string, bool) {
	if o == nil || o.Unit == nil {
		return nil, false
	}
	return o.Unit, true
}

// HasUnit returns a boolean if a field has been set.
func (o *MetricSeries) HasUnit() bool {
	return o != nil && o.Unit != nil
}

// SetUnit gets a reference to the given string and assigns it to the Unit field.
func (o *MetricSeries) SetUnit(v string) {
	o.Unit = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o MetricSeries) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Interval != nil {
		toSerialize["interval"] = o.Interval
	}
	if o.Metadata != nil {
		toSerialize["metadata"] = o.Metadata
	}
	toSerialize["metric"] = o.Metric
	toSerialize["points"] = o.Points
	if o.Resources != nil {
		toSerialize["resources"] = o.Resources
	}
	if o.SourceTypeName != nil {
		toSerialize["source_type_name"] = o.SourceTypeName
	}
	if o.Tags != nil {
		toSerialize["tags"] = o.Tags
	}
	if o.Type != nil {
		toSerialize["type"] = o.Type
	}
	if o.Unit != nil {
		toSerialize["unit"] = o.Unit
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MetricSeries) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Metric *string        `json:"metric"`
		Points *[]MetricPoint `json:"points"`
	}{}
	all := struct {
		Interval       *int64            `json:"interval,omitempty"`
		Metadata       *MetricMetadata   `json:"metadata,omitempty"`
		Metric         string            `json:"metric"`
		Points         []MetricPoint     `json:"points"`
		Resources      []MetricResource  `json:"resources,omitempty"`
		SourceTypeName *string           `json:"source_type_name,omitempty"`
		Tags           []string          `json:"tags,omitempty"`
		Type           *MetricIntakeType `json:"type,omitempty"`
		Unit           *string           `json:"unit,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Metric == nil {
		return fmt.Errorf("required field metric missing")
	}
	if required.Points == nil {
		return fmt.Errorf("required field points missing")
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
	if v := all.Type; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.Interval = all.Interval
	if all.Metadata != nil && all.Metadata.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Metadata = all.Metadata
	o.Metric = all.Metric
	o.Points = all.Points
	o.Resources = all.Resources
	o.SourceTypeName = all.SourceTypeName
	o.Tags = all.Tags
	o.Type = all.Type
	o.Unit = all.Unit
	return nil
}
