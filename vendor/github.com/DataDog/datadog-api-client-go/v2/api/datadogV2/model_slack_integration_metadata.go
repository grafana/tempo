// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SlackIntegrationMetadata Incident integration metadata for the Slack integration.
type SlackIntegrationMetadata struct {
	// Array of Slack channels in this integration metadata.
	Channels []SlackIntegrationMetadataChannelItem `json:"channels"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSlackIntegrationMetadata instantiates a new SlackIntegrationMetadata object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSlackIntegrationMetadata(channels []SlackIntegrationMetadataChannelItem) *SlackIntegrationMetadata {
	this := SlackIntegrationMetadata{}
	this.Channels = channels
	return &this
}

// NewSlackIntegrationMetadataWithDefaults instantiates a new SlackIntegrationMetadata object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSlackIntegrationMetadataWithDefaults() *SlackIntegrationMetadata {
	this := SlackIntegrationMetadata{}
	return &this
}

// GetChannels returns the Channels field value.
func (o *SlackIntegrationMetadata) GetChannels() []SlackIntegrationMetadataChannelItem {
	if o == nil {
		var ret []SlackIntegrationMetadataChannelItem
		return ret
	}
	return o.Channels
}

// GetChannelsOk returns a tuple with the Channels field value
// and a boolean to check if the value has been set.
func (o *SlackIntegrationMetadata) GetChannelsOk() (*[]SlackIntegrationMetadataChannelItem, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Channels, true
}

// SetChannels sets field value.
func (o *SlackIntegrationMetadata) SetChannels(v []SlackIntegrationMetadataChannelItem) {
	o.Channels = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SlackIntegrationMetadata) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["channels"] = o.Channels

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SlackIntegrationMetadata) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Channels *[]SlackIntegrationMetadataChannelItem `json:"channels"`
	}{}
	all := struct {
		Channels []SlackIntegrationMetadataChannelItem `json:"channels"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Channels == nil {
		return fmt.Errorf("required field channels missing")
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
	o.Channels = all.Channels
	return nil
}
