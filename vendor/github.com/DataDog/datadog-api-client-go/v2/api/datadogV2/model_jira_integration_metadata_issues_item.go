// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV2

import (
	"encoding/json"
	"fmt"
)

// JiraIntegrationMetadataIssuesItem Item in the Jira integration metadata issue array.
type JiraIntegrationMetadataIssuesItem struct {
	// URL of issue's Jira account.
	Account string `json:"account"`
	// Jira issue's issue key.
	IssueKey *string `json:"issue_key,omitempty"`
	// Jira issue's issue type.
	IssuetypeId *string `json:"issuetype_id,omitempty"`
	// Jira issue's project keys.
	ProjectKey string `json:"project_key"`
	// URL redirecting to the Jira issue.
	RedirectUrl *string `json:"redirect_url,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewJiraIntegrationMetadataIssuesItem instantiates a new JiraIntegrationMetadataIssuesItem object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewJiraIntegrationMetadataIssuesItem(account string, projectKey string) *JiraIntegrationMetadataIssuesItem {
	this := JiraIntegrationMetadataIssuesItem{}
	this.Account = account
	this.ProjectKey = projectKey
	return &this
}

// NewJiraIntegrationMetadataIssuesItemWithDefaults instantiates a new JiraIntegrationMetadataIssuesItem object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewJiraIntegrationMetadataIssuesItemWithDefaults() *JiraIntegrationMetadataIssuesItem {
	this := JiraIntegrationMetadataIssuesItem{}
	return &this
}

// GetAccount returns the Account field value.
func (o *JiraIntegrationMetadataIssuesItem) GetAccount() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.Account
}

// GetAccountOk returns a tuple with the Account field value
// and a boolean to check if the value has been set.
func (o *JiraIntegrationMetadataIssuesItem) GetAccountOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Account, true
}

// SetAccount sets field value.
func (o *JiraIntegrationMetadataIssuesItem) SetAccount(v string) {
	o.Account = v
}

// GetIssueKey returns the IssueKey field value if set, zero value otherwise.
func (o *JiraIntegrationMetadataIssuesItem) GetIssueKey() string {
	if o == nil || o.IssueKey == nil {
		var ret string
		return ret
	}
	return *o.IssueKey
}

// GetIssueKeyOk returns a tuple with the IssueKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *JiraIntegrationMetadataIssuesItem) GetIssueKeyOk() (*string, bool) {
	if o == nil || o.IssueKey == nil {
		return nil, false
	}
	return o.IssueKey, true
}

// HasIssueKey returns a boolean if a field has been set.
func (o *JiraIntegrationMetadataIssuesItem) HasIssueKey() bool {
	return o != nil && o.IssueKey != nil
}

// SetIssueKey gets a reference to the given string and assigns it to the IssueKey field.
func (o *JiraIntegrationMetadataIssuesItem) SetIssueKey(v string) {
	o.IssueKey = &v
}

// GetIssuetypeId returns the IssuetypeId field value if set, zero value otherwise.
func (o *JiraIntegrationMetadataIssuesItem) GetIssuetypeId() string {
	if o == nil || o.IssuetypeId == nil {
		var ret string
		return ret
	}
	return *o.IssuetypeId
}

// GetIssuetypeIdOk returns a tuple with the IssuetypeId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *JiraIntegrationMetadataIssuesItem) GetIssuetypeIdOk() (*string, bool) {
	if o == nil || o.IssuetypeId == nil {
		return nil, false
	}
	return o.IssuetypeId, true
}

// HasIssuetypeId returns a boolean if a field has been set.
func (o *JiraIntegrationMetadataIssuesItem) HasIssuetypeId() bool {
	return o != nil && o.IssuetypeId != nil
}

// SetIssuetypeId gets a reference to the given string and assigns it to the IssuetypeId field.
func (o *JiraIntegrationMetadataIssuesItem) SetIssuetypeId(v string) {
	o.IssuetypeId = &v
}

// GetProjectKey returns the ProjectKey field value.
func (o *JiraIntegrationMetadataIssuesItem) GetProjectKey() string {
	if o == nil {
		var ret string
		return ret
	}
	return o.ProjectKey
}

// GetProjectKeyOk returns a tuple with the ProjectKey field value
// and a boolean to check if the value has been set.
func (o *JiraIntegrationMetadataIssuesItem) GetProjectKeyOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ProjectKey, true
}

// SetProjectKey sets field value.
func (o *JiraIntegrationMetadataIssuesItem) SetProjectKey(v string) {
	o.ProjectKey = v
}

// GetRedirectUrl returns the RedirectUrl field value if set, zero value otherwise.
func (o *JiraIntegrationMetadataIssuesItem) GetRedirectUrl() string {
	if o == nil || o.RedirectUrl == nil {
		var ret string
		return ret
	}
	return *o.RedirectUrl
}

// GetRedirectUrlOk returns a tuple with the RedirectUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *JiraIntegrationMetadataIssuesItem) GetRedirectUrlOk() (*string, bool) {
	if o == nil || o.RedirectUrl == nil {
		return nil, false
	}
	return o.RedirectUrl, true
}

// HasRedirectUrl returns a boolean if a field has been set.
func (o *JiraIntegrationMetadataIssuesItem) HasRedirectUrl() bool {
	return o != nil && o.RedirectUrl != nil
}

// SetRedirectUrl gets a reference to the given string and assigns it to the RedirectUrl field.
func (o *JiraIntegrationMetadataIssuesItem) SetRedirectUrl(v string) {
	o.RedirectUrl = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o JiraIntegrationMetadataIssuesItem) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	toSerialize["account"] = o.Account
	if o.IssueKey != nil {
		toSerialize["issue_key"] = o.IssueKey
	}
	if o.IssuetypeId != nil {
		toSerialize["issuetype_id"] = o.IssuetypeId
	}
	toSerialize["project_key"] = o.ProjectKey
	if o.RedirectUrl != nil {
		toSerialize["redirect_url"] = o.RedirectUrl
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *JiraIntegrationMetadataIssuesItem) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	required := struct {
		Account    *string `json:"account"`
		ProjectKey *string `json:"project_key"`
	}{}
	all := struct {
		Account     string  `json:"account"`
		IssueKey    *string `json:"issue_key,omitempty"`
		IssuetypeId *string `json:"issuetype_id,omitempty"`
		ProjectKey  string  `json:"project_key"`
		RedirectUrl *string `json:"redirect_url,omitempty"`
	}{}
	err = json.Unmarshal(bytes, &required)
	if err != nil {
		return err
	}
	if required.Account == nil {
		return fmt.Errorf("required field account missing")
	}
	if required.ProjectKey == nil {
		return fmt.Errorf("required field project_key missing")
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
	o.Account = all.Account
	o.IssueKey = all.IssueKey
	o.IssuetypeId = all.IssuetypeId
	o.ProjectKey = all.ProjectKey
	o.RedirectUrl = all.RedirectUrl
	return nil
}
