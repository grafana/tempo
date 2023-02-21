// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
)

// CIAppPipelinesAggregationBucketsResponse The query results.
type CIAppPipelinesAggregationBucketsResponse struct {
	// The list of matching buckets, one item per bucket.
	Buckets []CIAppPipelinesBucketResponse `json:"buckets,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewCIAppPipelinesAggregationBucketsResponse instantiates a new CIAppPipelinesAggregationBucketsResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewCIAppPipelinesAggregationBucketsResponse() *CIAppPipelinesAggregationBucketsResponse {
	this := CIAppPipelinesAggregationBucketsResponse{}
	return &this
}

// NewCIAppPipelinesAggregationBucketsResponseWithDefaults instantiates a new CIAppPipelinesAggregationBucketsResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewCIAppPipelinesAggregationBucketsResponseWithDefaults() *CIAppPipelinesAggregationBucketsResponse {
	this := CIAppPipelinesAggregationBucketsResponse{}
	return &this
}

// GetBuckets returns the Buckets field value if set, zero value otherwise.
func (o *CIAppPipelinesAggregationBucketsResponse) GetBuckets() []CIAppPipelinesBucketResponse {
	if o == nil || o.Buckets == nil {
		var ret []CIAppPipelinesBucketResponse
		return ret
	}
	return o.Buckets
}

// GetBucketsOk returns a tuple with the Buckets field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *CIAppPipelinesAggregationBucketsResponse) GetBucketsOk() (*[]CIAppPipelinesBucketResponse, bool) {
	if o == nil || o.Buckets == nil {
		return nil, false
	}
	return &o.Buckets, true
}

// HasBuckets returns a boolean if a field has been set.
func (o *CIAppPipelinesAggregationBucketsResponse) HasBuckets() bool {
	return o != nil && o.Buckets != nil
}

// SetBuckets gets a reference to the given []CIAppPipelinesBucketResponse and assigns it to the Buckets field.
func (o *CIAppPipelinesAggregationBucketsResponse) SetBuckets(v []CIAppPipelinesBucketResponse) {
	o.Buckets = v
}

// MarshalJSON serializes the struct using spec logic.
func (o CIAppPipelinesAggregationBucketsResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Buckets != nil {
		toSerialize["buckets"] = o.Buckets
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *CIAppPipelinesAggregationBucketsResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Buckets []CIAppPipelinesBucketResponse `json:"buckets,omitempty"`
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
	o.Buckets = all.Buckets
	return nil
}
