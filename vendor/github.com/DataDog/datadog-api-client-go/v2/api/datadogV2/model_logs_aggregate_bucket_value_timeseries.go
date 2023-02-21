// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// LogsAggregateBucketValueTimeseries A timeseries array
type LogsAggregateBucketValueTimeseries struct {
	Items []LogsAggregateBucketValueTimeseriesPoint

	// UnparsedObject contains the raw value of the array if there was an error when deserializing into the struct
	UnparsedObject []interface{} `json:"-"`
}

// NewLogsAggregateBucketValueTimeseries instantiates a new LogsAggregateBucketValueTimeseries object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewLogsAggregateBucketValueTimeseries() *LogsAggregateBucketValueTimeseries {
	this := LogsAggregateBucketValueTimeseries{}
	return &this
}

// NewLogsAggregateBucketValueTimeseriesWithDefaults instantiates a new LogsAggregateBucketValueTimeseries object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewLogsAggregateBucketValueTimeseriesWithDefaults() *LogsAggregateBucketValueTimeseries {
	this := LogsAggregateBucketValueTimeseries{}
	return &this
}

// MarshalJSON serializes the struct using spec logic.
func (o LogsAggregateBucketValueTimeseries) MarshalJSON() ([]byte, error) {
	toSerialize := make([]interface{}, len(o.Items))
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	for i, item := range o.Items {
		toSerialize[i] = item
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *LogsAggregateBucketValueTimeseries) UnmarshalJSON(bytes []byte) (err error) {
	return json.Unmarshal(bytes, &o.Items)
}
