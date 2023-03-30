// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// SlackIntegrationMetadataChannelItem Item in the Slack integration metadata channel array.
type SlackIntegrationMetadataChannelItem struct {
	// Slack channel ID.
	ChannelId string `json:"channel_id"`
	// Name of the Slack channel.
	ChannelName string `json:"channel_name"`
	// URL redirecting to the Slack channel.
	RedirectUrl string `json:"redirect_url"`
	// Slack team ID.
	TeamId *string `json:"team_id,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSlackIntegrationMetadataChannelItem instantiates a new SlackIntegrationMetadataChannelItem object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSlackIntegrationMetadataChannelItem(channelId string, channelName string, redirectUrl string) *SlackIntegrationMetadataChannelItem {
	this := SlackIntegrationMetadataChannelItem{}
	this.ChannelId = channelId
	this.ChannelName = channelName
	this.RedirectUrl = redirectUrl
	return &this
}

// NewSlackIntegrationMetadataChannelItemWithDefaults instantiates a new SlackIntegrationMetadataChannelItem object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSlackIntegrationMetadataChannelItemWithDefaults() *SlackIntegrationMetadataChannelItem {
	this := SlackIntegrationMetadataChannelItem{}
	return &this
}

// GetChannelId returns the ChannelId field value.
func (o *SlackIntegrationMetadataChannelItem) GetChannelId() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ChannelId
}

// GetChannelIdOk returns a tuple with the ChannelId field value
// and a boolean to check if the value has been set.
func (o *SlackIntegrationMetadataChannelItem) GetChannelIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ChannelId, true
}

// SetChannelId sets field value.
func (o *SlackIntegrationMetadataChannelItem) SetChannelId(v string) {
	o.ChannelId = v
}

// GetChannelName returns the ChannelName field value.
func (o *SlackIntegrationMetadataChannelItem) GetChannelName() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ChannelName
}

// GetChannelNameOk returns a tuple with the ChannelName field value
// and a boolean to check if the value has been set.
func (o *SlackIntegrationMetadataChannelItem) GetChannelNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ChannelName, true
}

// SetChannelName sets field value.
func (o *SlackIntegrationMetadataChannelItem) SetChannelName(v string) {
	o.ChannelName = v
}

// GetRedirectUrl returns the RedirectUrl field value.
func (o *SlackIntegrationMetadataChannelItem) GetRedirectUrl() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.RedirectUrl
}

// GetRedirectUrlOk returns a tuple with the RedirectUrl field value
// and a boolean to check if the value has been set.
func (o *SlackIntegrationMetadataChannelItem) GetRedirectUrlOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.RedirectUrl, true
}

// SetRedirectUrl sets field value.
func (o *SlackIntegrationMetadataChannelItem) SetRedirectUrl(v string) {
	o.RedirectUrl = v
}

// GetTeamId returns the TeamId field value if set, zero value otherwise.
func (o *SlackIntegrationMetadataChannelItem) GetTeamId() string {
	if o == nil || o.TeamId == nil {
		var ret string
		return ret
	}
	return *o.TeamId
}

// GetTeamIdOk returns a tuple with the TeamId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SlackIntegrationMetadataChannelItem) GetTeamIdOk() (*string, bool) {
	if o == nil || o.TeamId == nil {
		return nil, false
	}
	return o.TeamId, true
}

// HasTeamId returns a boolean if a field has been set.
func (o *SlackIntegrationMetadataChannelItem) HasTeamId() bool {
	return o != nil && o.TeamId != nil
}

// SetTeamId gets a reference to the given string and assigns it to the TeamId field.
func (o *SlackIntegrationMetadataChannelItem) SetTeamId(v string) {
	o.TeamId = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o SlackIntegrationMetadataChannelItem) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["channel_id"] = o.ChannelId
	toSerialize["channel_name"] = o.ChannelName
	toSerialize["redirect_url"] = o.RedirectUrl
	if o.TeamId != nil {
		toSerialize["team_id"] = o.TeamId
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SlackIntegrationMetadataChannelItem) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		ChannelId   *string `json:"channel_id"`
		ChannelName *string `json:"channel_name"`
		RedirectUrl *string `json:"redirect_url"`
	}{}
	all := struct {
		ChannelId   string  `json:"channel_id"`
		ChannelName string  `json:"channel_name"`
		RedirectUrl string  `json:"redirect_url"`
		TeamId      *string `json:"team_id,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.ChannelId == nil {
		return fmt.Errorf("required field channel_id missing")
	}
	if required.ChannelName == nil {
		return fmt.Errorf("required field channel_name missing")
	}
	if required.RedirectUrl == nil {
		return fmt.Errorf("required field redirect_url missing")
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
	o.ChannelId = all.ChannelId
	o.ChannelName = all.ChannelName
	o.RedirectUrl = all.RedirectUrl
	o.TeamId = all.TeamId
	return nil
}
