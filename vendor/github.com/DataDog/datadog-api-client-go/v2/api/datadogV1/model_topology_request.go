// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
)

// TopologyRequest Request that will return nodes and edges to be used by topology map.
type TopologyRequest struct {
	// Query to service-based topology data sources like the service map or data streams.
	Query *TopologyQuery `json:"query,omitempty"`
	// Widget request type.
	RequestType *TopologyRequestType `json:"request_type,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewTopologyRequest instantiates a new TopologyRequest object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewTopologyRequest() *TopologyRequest {
	this := TopologyRequest{}
	return &this
}

// NewTopologyRequestWithDefaults instantiates a new TopologyRequest object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewTopologyRequestWithDefaults() *TopologyRequest {
	this := TopologyRequest{}
	return &this
}

// GetQuery returns the Query field value if set, zero value otherwise.
func (o *TopologyRequest) GetQuery() TopologyQuery {
	if o == nil || o.Query == nil {
		var ret TopologyQuery
		return ret
	}
	return *o.Query
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TopologyRequest) GetQueryOk() (*TopologyQuery, bool) {
	if o == nil || o.Query == nil {
		return nil, false
	}
	return o.Query, true
}

// HasQuery returns a boolean if a field has been set.
func (o *TopologyRequest) HasQuery() bool {
	return o != nil && o.Query != nil
}

// SetQuery gets a reference to the given TopologyQuery and assigns it to the Query field.
func (o *TopologyRequest) SetQuery(v TopologyQuery) {
	o.Query = &v
}

// GetRequestType returns the RequestType field value if set, zero value otherwise.
func (o *TopologyRequest) GetRequestType() TopologyRequestType {
	if o == nil || o.RequestType == nil {
		var ret TopologyRequestType
		return ret
	}
	return *o.RequestType
}

// GetRequestTypeOk returns a tuple with the RequestType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *TopologyRequest) GetRequestTypeOk() (*TopologyRequestType, bool) {
	if o == nil || o.RequestType == nil {
		return nil, false
	}
	return o.RequestType, true
}

// HasRequestType returns a boolean if a field has been set.
func (o *TopologyRequest) HasRequestType() bool {
	return o != nil && o.RequestType != nil
}

// SetRequestType gets a reference to the given TopologyRequestType and assigns it to the RequestType field.
func (o *TopologyRequest) SetRequestType(v TopologyRequestType) {
	o.RequestType = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o TopologyRequest) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Query != nil {
		toSerialize["query"] = o.Query
	}
	if o.RequestType != nil {
		toSerialize["request_type"] = o.RequestType
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *TopologyRequest) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Query       *TopologyQuery       `json:"query,omitempty"`
		RequestType *TopologyRequestType `json:"request_type,omitempty"`
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
	if v := all.RequestType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Query != nil && all.Query.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Query = all.Query
	o.RequestType = all.RequestType
	return nil
}
