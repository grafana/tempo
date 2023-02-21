// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// SearchServiceLevelObjectiveAttributes A service level objective object includes a service level indicator, thresholds
// for one or more timeframes, and metadata (`name`, `description`, and `tags`).
type SearchServiceLevelObjectiveAttributes struct {
	// A list of tags associated with this service level objective.
	// Always included in service level objective responses (but may be empty).
	AllTags []string `json:"all_tags,omitempty"`
	// Creation timestamp (UNIX time in seconds)
	//
	// Always included in service level objective responses.
	CreatedAt *int64 `json:"created_at,omitempty"`
	// The creator of the SLO
	Creator NullableSLOCreator `json:"creator,omitempty"`
	// A user-defined description of the service level objective.
	//
	// Always included in service level objective responses (but may be `null`).
	// Optional in create/update requests.
	Description datadog.NullableString `json:"description,omitempty"`
	// Tags with the `env` tag key.
	EnvTags []string `json:"env_tags,omitempty"`
	// A list of (up to 100) monitor groups that narrow the scope of a monitor service level objective.
	// Included in service level objective responses if it is not empty.
	Groups []string `json:"groups,omitempty"`
	// Modification timestamp (UNIX time in seconds)
	//
	// Always included in service level objective responses.
	ModifiedAt *int64 `json:"modified_at,omitempty"`
	// A list of monitor ids that defines the scope of a monitor service level
	// objective.
	MonitorIds []int64 `json:"monitor_ids,omitempty"`
	// The name of the service level objective object.
	Name *string `json:"name,omitempty"`
	// calculated status and error budget remaining.
	OverallStatus []SLOOverallStatuses `json:"overall_status,omitempty"`
	// A metric-based SLO. **Required if type is `metric`**. Note that Datadog only allows the sum by aggregator
	// to be used because this will sum up all request counts instead of averaging them, or taking the max or
	// min of all of those requests.
	Query NullableSearchSLOQuery `json:"query,omitempty"`
	// Tags with the `service` tag key.
	ServiceTags []string `json:"service_tags,omitempty"`
	// The type of the service level objective.
	SloType *SLOType `json:"slo_type,omitempty"`
	// Status of the SLO's primary timeframe.
	Status *SLOStatus `json:"status,omitempty"`
	// Tags with the `team` tag key.
	TeamTags []string `json:"team_tags,omitempty"`
	// The thresholds (timeframes and associated targets) for this service level
	// objective object.
	Thresholds []SearchSLOThreshold `json:"thresholds,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewSearchServiceLevelObjectiveAttributes instantiates a new SearchServiceLevelObjectiveAttributes object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewSearchServiceLevelObjectiveAttributes() *SearchServiceLevelObjectiveAttributes {
	this := SearchServiceLevelObjectiveAttributes{}
	return &this
}

// NewSearchServiceLevelObjectiveAttributesWithDefaults instantiates a new SearchServiceLevelObjectiveAttributes object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewSearchServiceLevelObjectiveAttributesWithDefaults() *SearchServiceLevelObjectiveAttributes {
	this := SearchServiceLevelObjectiveAttributes{}
	return &this
}

// GetAllTags returns the AllTags field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetAllTags() []string {
	if o == nil || o.AllTags == nil {
		var ret []string
		return ret
	}
	return o.AllTags
}

// GetAllTagsOk returns a tuple with the AllTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetAllTagsOk() (*[]string, bool) {
	if o == nil || o.AllTags == nil {
		return nil, false
	}
	return &o.AllTags, true
}

// HasAllTags returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasAllTags() bool {
	return o != nil && o.AllTags != nil
}

// SetAllTags gets a reference to the given []string and assigns it to the AllTags field.
func (o *SearchServiceLevelObjectiveAttributes) SetAllTags(v []string) {
	o.AllTags = v
}

// GetCreatedAt returns the CreatedAt field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetCreatedAt() int64 {
	if o == nil || o.CreatedAt == nil {
		var ret int64
		return ret
	}
	return *o.CreatedAt
}

// GetCreatedAtOk returns a tuple with the CreatedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetCreatedAtOk() (*int64, bool) {
	if o == nil || o.CreatedAt == nil {
		return nil, false
	}
	return o.CreatedAt, true
}

// HasCreatedAt returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasCreatedAt() bool {
	return o != nil && o.CreatedAt != nil
}

// SetCreatedAt gets a reference to the given int64 and assigns it to the CreatedAt field.
func (o *SearchServiceLevelObjectiveAttributes) SetCreatedAt(v int64) {
	o.CreatedAt = &v
}

// GetCreator returns the Creator field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SearchServiceLevelObjectiveAttributes) GetCreator() SLOCreator {
	if o == nil || o.Creator.Get() == nil {
		var ret SLOCreator
		return ret
	}
	return *o.Creator.Get()
}

// GetCreatorOk returns a tuple with the Creator field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SearchServiceLevelObjectiveAttributes) GetCreatorOk() (*SLOCreator, bool) {
	if o == nil {
		return nil, false
	}
	return o.Creator.Get(), o.Creator.IsSet()
}

// HasCreator returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasCreator() bool {
	return o != nil && o.Creator.IsSet()
}

// SetCreator gets a reference to the given NullableSLOCreator and assigns it to the Creator field.
func (o *SearchServiceLevelObjectiveAttributes) SetCreator(v SLOCreator) {
	o.Creator.Set(&v)
}

// SetCreatorNil sets the value for Creator to be an explicit nil.
func (o *SearchServiceLevelObjectiveAttributes) SetCreatorNil() {
	o.Creator.Set(nil)
}

// UnsetCreator ensures that no value is present for Creator, not even an explicit nil.
func (o *SearchServiceLevelObjectiveAttributes) UnsetCreator() {
	o.Creator.Unset()
}

// GetDescription returns the Description field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SearchServiceLevelObjectiveAttributes) GetDescription() string {
	if o == nil || o.Description.Get() == nil {
		var ret string
		return ret
	}
	return *o.Description.Get()
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SearchServiceLevelObjectiveAttributes) GetDescriptionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Description.Get(), o.Description.IsSet()
}

// HasDescription returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasDescription() bool {
	return o != nil && o.Description.IsSet()
}

// SetDescription gets a reference to the given datadog.NullableString and assigns it to the Description field.
func (o *SearchServiceLevelObjectiveAttributes) SetDescription(v string) {
	o.Description.Set(&v)
}

// SetDescriptionNil sets the value for Description to be an explicit nil.
func (o *SearchServiceLevelObjectiveAttributes) SetDescriptionNil() {
	o.Description.Set(nil)
}

// UnsetDescription ensures that no value is present for Description, not even an explicit nil.
func (o *SearchServiceLevelObjectiveAttributes) UnsetDescription() {
	o.Description.Unset()
}

// GetEnvTags returns the EnvTags field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetEnvTags() []string {
	if o == nil || o.EnvTags == nil {
		var ret []string
		return ret
	}
	return o.EnvTags
}

// GetEnvTagsOk returns a tuple with the EnvTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetEnvTagsOk() (*[]string, bool) {
	if o == nil || o.EnvTags == nil {
		return nil, false
	}
	return &o.EnvTags, true
}

// HasEnvTags returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasEnvTags() bool {
	return o != nil && o.EnvTags != nil
}

// SetEnvTags gets a reference to the given []string and assigns it to the EnvTags field.
func (o *SearchServiceLevelObjectiveAttributes) SetEnvTags(v []string) {
	o.EnvTags = v
}

// GetGroups returns the Groups field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SearchServiceLevelObjectiveAttributes) GetGroups() []string {
	if o == nil {
		var ret []string
		return ret
	}
	return o.Groups
}

// GetGroupsOk returns a tuple with the Groups field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SearchServiceLevelObjectiveAttributes) GetGroupsOk() (*[]string, bool) {
	if o == nil || o.Groups == nil {
		return nil, false
	}
	return &o.Groups, true
}

// HasGroups returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasGroups() bool {
	return o != nil && o.Groups != nil
}

// SetGroups gets a reference to the given []string and assigns it to the Groups field.
func (o *SearchServiceLevelObjectiveAttributes) SetGroups(v []string) {
	o.Groups = v
}

// GetModifiedAt returns the ModifiedAt field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetModifiedAt() int64 {
	if o == nil || o.ModifiedAt == nil {
		var ret int64
		return ret
	}
	return *o.ModifiedAt
}

// GetModifiedAtOk returns a tuple with the ModifiedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetModifiedAtOk() (*int64, bool) {
	if o == nil || o.ModifiedAt == nil {
		return nil, false
	}
	return o.ModifiedAt, true
}

// HasModifiedAt returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasModifiedAt() bool {
	return o != nil && o.ModifiedAt != nil
}

// SetModifiedAt gets a reference to the given int64 and assigns it to the ModifiedAt field.
func (o *SearchServiceLevelObjectiveAttributes) SetModifiedAt(v int64) {
	o.ModifiedAt = &v
}

// GetMonitorIds returns the MonitorIds field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SearchServiceLevelObjectiveAttributes) GetMonitorIds() []int64 {
	if o == nil {
		var ret []int64
		return ret
	}
	return o.MonitorIds
}

// GetMonitorIdsOk returns a tuple with the MonitorIds field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SearchServiceLevelObjectiveAttributes) GetMonitorIdsOk() (*[]int64, bool) {
	if o == nil || o.MonitorIds == nil {
		return nil, false
	}
	return &o.MonitorIds, true
}

// HasMonitorIds returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasMonitorIds() bool {
	return o != nil && o.MonitorIds != nil
}

// SetMonitorIds gets a reference to the given []int64 and assigns it to the MonitorIds field.
func (o *SearchServiceLevelObjectiveAttributes) SetMonitorIds(v []int64) {
	o.MonitorIds = v
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetName() string {
	if o == nil || o.Name == nil {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetNameOk() (*string, bool) {
	if o == nil || o.Name == nil {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasName() bool {
	return o != nil && o.Name != nil
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SearchServiceLevelObjectiveAttributes) SetName(v string) {
	o.Name = &v
}

// GetOverallStatus returns the OverallStatus field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetOverallStatus() []SLOOverallStatuses {
	if o == nil || o.OverallStatus == nil {
		var ret []SLOOverallStatuses
		return ret
	}
	return o.OverallStatus
}

// GetOverallStatusOk returns a tuple with the OverallStatus field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetOverallStatusOk() (*[]SLOOverallStatuses, bool) {
	if o == nil || o.OverallStatus == nil {
		return nil, false
	}
	return &o.OverallStatus, true
}

// HasOverallStatus returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasOverallStatus() bool {
	return o != nil && o.OverallStatus != nil
}

// SetOverallStatus gets a reference to the given []SLOOverallStatuses and assigns it to the OverallStatus field.
func (o *SearchServiceLevelObjectiveAttributes) SetOverallStatus(v []SLOOverallStatuses) {
	o.OverallStatus = v
}

// GetQuery returns the Query field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SearchServiceLevelObjectiveAttributes) GetQuery() SearchSLOQuery {
	if o == nil || o.Query.Get() == nil {
		var ret SearchSLOQuery
		return ret
	}
	return *o.Query.Get()
}

// GetQueryOk returns a tuple with the Query field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *SearchServiceLevelObjectiveAttributes) GetQueryOk() (*SearchSLOQuery, bool) {
	if o == nil {
		return nil, false
	}
	return o.Query.Get(), o.Query.IsSet()
}

// HasQuery returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasQuery() bool {
	return o != nil && o.Query.IsSet()
}

// SetQuery gets a reference to the given NullableSearchSLOQuery and assigns it to the Query field.
func (o *SearchServiceLevelObjectiveAttributes) SetQuery(v SearchSLOQuery) {
	o.Query.Set(&v)
}

// SetQueryNil sets the value for Query to be an explicit nil.
func (o *SearchServiceLevelObjectiveAttributes) SetQueryNil() {
	o.Query.Set(nil)
}

// UnsetQuery ensures that no value is present for Query, not even an explicit nil.
func (o *SearchServiceLevelObjectiveAttributes) UnsetQuery() {
	o.Query.Unset()
}

// GetServiceTags returns the ServiceTags field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetServiceTags() []string {
	if o == nil || o.ServiceTags == nil {
		var ret []string
		return ret
	}
	return o.ServiceTags
}

// GetServiceTagsOk returns a tuple with the ServiceTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetServiceTagsOk() (*[]string, bool) {
	if o == nil || o.ServiceTags == nil {
		return nil, false
	}
	return &o.ServiceTags, true
}

// HasServiceTags returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasServiceTags() bool {
	return o != nil && o.ServiceTags != nil
}

// SetServiceTags gets a reference to the given []string and assigns it to the ServiceTags field.
func (o *SearchServiceLevelObjectiveAttributes) SetServiceTags(v []string) {
	o.ServiceTags = v
}

// GetSloType returns the SloType field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetSloType() SLOType {
	if o == nil || o.SloType == nil {
		var ret SLOType
		return ret
	}
	return *o.SloType
}

// GetSloTypeOk returns a tuple with the SloType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetSloTypeOk() (*SLOType, bool) {
	if o == nil || o.SloType == nil {
		return nil, false
	}
	return o.SloType, true
}

// HasSloType returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasSloType() bool {
	return o != nil && o.SloType != nil
}

// SetSloType gets a reference to the given SLOType and assigns it to the SloType field.
func (o *SearchServiceLevelObjectiveAttributes) SetSloType(v SLOType) {
	o.SloType = &v
}

// GetStatus returns the Status field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetStatus() SLOStatus {
	if o == nil || o.Status == nil {
		var ret SLOStatus
		return ret
	}
	return *o.Status
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetStatusOk() (*SLOStatus, bool) {
	if o == nil || o.Status == nil {
		return nil, false
	}
	return o.Status, true
}

// HasStatus returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasStatus() bool {
	return o != nil && o.Status != nil
}

// SetStatus gets a reference to the given SLOStatus and assigns it to the Status field.
func (o *SearchServiceLevelObjectiveAttributes) SetStatus(v SLOStatus) {
	o.Status = &v
}

// GetTeamTags returns the TeamTags field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetTeamTags() []string {
	if o == nil || o.TeamTags == nil {
		var ret []string
		return ret
	}
	return o.TeamTags
}

// GetTeamTagsOk returns a tuple with the TeamTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetTeamTagsOk() (*[]string, bool) {
	if o == nil || o.TeamTags == nil {
		return nil, false
	}
	return &o.TeamTags, true
}

// HasTeamTags returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasTeamTags() bool {
	return o != nil && o.TeamTags != nil
}

// SetTeamTags gets a reference to the given []string and assigns it to the TeamTags field.
func (o *SearchServiceLevelObjectiveAttributes) SetTeamTags(v []string) {
	o.TeamTags = v
}

// GetThresholds returns the Thresholds field value if set, zero value otherwise.
func (o *SearchServiceLevelObjectiveAttributes) GetThresholds() []SearchSLOThreshold {
	if o == nil || o.Thresholds == nil {
		var ret []SearchSLOThreshold
		return ret
	}
	return o.Thresholds
}

// GetThresholdsOk returns a tuple with the Thresholds field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SearchServiceLevelObjectiveAttributes) GetThresholdsOk() (*[]SearchSLOThreshold, bool) {
	if o == nil || o.Thresholds == nil {
		return nil, false
	}
	return &o.Thresholds, true
}

// HasThresholds returns a boolean if a field has been set.
func (o *SearchServiceLevelObjectiveAttributes) HasThresholds() bool {
	return o != nil && o.Thresholds != nil
}

// SetThresholds gets a reference to the given []SearchSLOThreshold and assigns it to the Thresholds field.
func (o *SearchServiceLevelObjectiveAttributes) SetThresholds(v []SearchSLOThreshold) {
	o.Thresholds = v
}

// MarshalJSON serializes the struct using spec logic.
func (o SearchServiceLevelObjectiveAttributes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AllTags != nil {
		toSerialize["all_tags"] = o.AllTags
	}
	if o.CreatedAt != nil {
		toSerialize["created_at"] = o.CreatedAt
	}
	if o.Creator.IsSet() {
		toSerialize["creator"] = o.Creator.Get()
	}
	if o.Description.IsSet() {
		toSerialize["description"] = o.Description.Get()
	}
	if o.EnvTags != nil {
		toSerialize["env_tags"] = o.EnvTags
	}
	if o.Groups != nil {
		toSerialize["groups"] = o.Groups
	}
	if o.ModifiedAt != nil {
		toSerialize["modified_at"] = o.ModifiedAt
	}
	if o.MonitorIds != nil {
		toSerialize["monitor_ids"] = o.MonitorIds
	}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.OverallStatus != nil {
		toSerialize["overall_status"] = o.OverallStatus
	}
	if o.Query.IsSet() {
		toSerialize["query"] = o.Query.Get()
	}
	if o.ServiceTags != nil {
		toSerialize["service_tags"] = o.ServiceTags
	}
	if o.SloType != nil {
		toSerialize["slo_type"] = o.SloType
	}
	if o.Status != nil {
		toSerialize["status"] = o.Status
	}
	if o.TeamTags != nil {
		toSerialize["team_tags"] = o.TeamTags
	}
	if o.Thresholds != nil {
		toSerialize["thresholds"] = o.Thresholds
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *SearchServiceLevelObjectiveAttributes) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AllTags       []string               `json:"all_tags,omitempty"`
		CreatedAt     *int64                 `json:"created_at,omitempty"`
		Creator       NullableSLOCreator     `json:"creator,omitempty"`
		Description   datadog.NullableString `json:"description,omitempty"`
		EnvTags       []string               `json:"env_tags,omitempty"`
		Groups        []string               `json:"groups,omitempty"`
		ModifiedAt    *int64                 `json:"modified_at,omitempty"`
		MonitorIds    []int64                `json:"monitor_ids,omitempty"`
		Name          *string                `json:"name,omitempty"`
		OverallStatus []SLOOverallStatuses   `json:"overall_status,omitempty"`
		Query         NullableSearchSLOQuery `json:"query,omitempty"`
		ServiceTags   []string               `json:"service_tags,omitempty"`
		SloType       *SLOType               `json:"slo_type,omitempty"`
		Status        *SLOStatus             `json:"status,omitempty"`
		TeamTags      []string               `json:"team_tags,omitempty"`
		Thresholds    []SearchSLOThreshold   `json:"thresholds,omitempty"`
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
	if v := all.SloType; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	o.AllTags = all.AllTags
	o.CreatedAt = all.CreatedAt
	o.Creator = all.Creator
	o.Description = all.Description
	o.EnvTags = all.EnvTags
	o.Groups = all.Groups
	o.ModifiedAt = all.ModifiedAt
	o.MonitorIds = all.MonitorIds
	o.Name = all.Name
	o.OverallStatus = all.OverallStatus
	o.Query = all.Query
	o.ServiceTags = all.ServiceTags
	o.SloType = all.SloType
	if all.Status != nil && all.Status.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Status = all.Status
	o.TeamTags = all.TeamTags
	o.Thresholds = all.Thresholds
	return nil
}
