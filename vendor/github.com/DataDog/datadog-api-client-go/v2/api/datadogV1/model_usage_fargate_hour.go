// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"time"
)

// UsageFargateHour Number of Fargate tasks run and hourly usage.
type UsageFargateHour struct {
	// The high-water mark of APM ECS Fargate tasks during the given hour.
	ApmFargateCount *int64 `json:"apm_fargate_count,omitempty"`
	// The Application Security Monitoring ECS Fargate tasks during the given hour.
	AppsecFargateCount *int64 `json:"appsec_fargate_count,omitempty"`
	// The average profiled task count for Fargate Profiling.
	AvgProfiledFargateTasks *int64 `json:"avg_profiled_fargate_tasks,omitempty"`
	// The hour for the usage.
	Hour *time.Time `json:"hour,omitempty"`
	// The organization name.
	OrgName *string `json:"org_name,omitempty"`
	// The organization public ID.
	PublicId *string `json:"public_id,omitempty"`
	// The number of Fargate tasks run.
	TasksCount *int64 `json:"tasks_count,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUsageFargateHour instantiates a new UsageFargateHour object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageFargateHour() *UsageFargateHour {
	this := UsageFargateHour{}
	return &this
}

// NewUsageFargateHourWithDefaults instantiates a new UsageFargateHour object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageFargateHourWithDefaults() *UsageFargateHour {
	this := UsageFargateHour{}
	return &this
}

// GetApmFargateCount returns the ApmFargateCount field value if set, zero value otherwise.
func (o *UsageFargateHour) GetApmFargateCount() int64 {
	if o == nil || o.ApmFargateCount == nil {
		var ret int64
		return ret
	}
	return *o.ApmFargateCount
}

// GetApmFargateCountOk returns a tuple with the ApmFargateCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageFargateHour) GetApmFargateCountOk() (*int64, bool) {
	if o == nil || o.ApmFargateCount == nil {
		return nil, false
	}
	return o.ApmFargateCount, true
}

// HasApmFargateCount returns a boolean if a field has been set.
func (o *UsageFargateHour) HasApmFargateCount() bool {
	return o != nil && o.ApmFargateCount != nil
}

// SetApmFargateCount gets a reference to the given int64 and assigns it to the ApmFargateCount field.
func (o *UsageFargateHour) SetApmFargateCount(v int64) {
	o.ApmFargateCount = &v
}

// GetAppsecFargateCount returns the AppsecFargateCount field value if set, zero value otherwise.
func (o *UsageFargateHour) GetAppsecFargateCount() int64 {
	if o == nil || o.AppsecFargateCount == nil {
		var ret int64
		return ret
	}
	return *o.AppsecFargateCount
}

// GetAppsecFargateCountOk returns a tuple with the AppsecFargateCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageFargateHour) GetAppsecFargateCountOk() (*int64, bool) {
	if o == nil || o.AppsecFargateCount == nil {
		return nil, false
	}
	return o.AppsecFargateCount, true
}

// HasAppsecFargateCount returns a boolean if a field has been set.
func (o *UsageFargateHour) HasAppsecFargateCount() bool {
	return o != nil && o.AppsecFargateCount != nil
}

// SetAppsecFargateCount gets a reference to the given int64 and assigns it to the AppsecFargateCount field.
func (o *UsageFargateHour) SetAppsecFargateCount(v int64) {
	o.AppsecFargateCount = &v
}

// GetAvgProfiledFargateTasks returns the AvgProfiledFargateTasks field value if set, zero value otherwise.
func (o *UsageFargateHour) GetAvgProfiledFargateTasks() int64 {
	if o == nil || o.AvgProfiledFargateTasks == nil {
		var ret int64
		return ret
	}
	return *o.AvgProfiledFargateTasks
}

// GetAvgProfiledFargateTasksOk returns a tuple with the AvgProfiledFargateTasks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageFargateHour) GetAvgProfiledFargateTasksOk() (*int64, bool) {
	if o == nil || o.AvgProfiledFargateTasks == nil {
		return nil, false
	}
	return o.AvgProfiledFargateTasks, true
}

// HasAvgProfiledFargateTasks returns a boolean if a field has been set.
func (o *UsageFargateHour) HasAvgProfiledFargateTasks() bool {
	return o != nil && o.AvgProfiledFargateTasks != nil
}

// SetAvgProfiledFargateTasks gets a reference to the given int64 and assigns it to the AvgProfiledFargateTasks field.
func (o *UsageFargateHour) SetAvgProfiledFargateTasks(v int64) {
	o.AvgProfiledFargateTasks = &v
}

// GetHour returns the Hour field value if set, zero value otherwise.
func (o *UsageFargateHour) GetHour() time.Time {
	if o == nil || o.Hour == nil {
		var ret time.Time
		return ret
	}
	return *o.Hour
}

// GetHourOk returns a tuple with the Hour field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageFargateHour) GetHourOk() (*time.Time, bool) {
	if o == nil || o.Hour == nil {
		return nil, false
	}
	return o.Hour, true
}

// HasHour returns a boolean if a field has been set.
func (o *UsageFargateHour) HasHour() bool {
	return o != nil && o.Hour != nil
}

// SetHour gets a reference to the given time.Time and assigns it to the Hour field.
func (o *UsageFargateHour) SetHour(v time.Time) {
	o.Hour = &v
}

// GetOrgName returns the OrgName field value if set, zero value otherwise.
func (o *UsageFargateHour) GetOrgName() string {
	if o == nil || o.OrgName == nil {
		var ret string
		return ret
	}
	return *o.OrgName
}

// GetOrgNameOk returns a tuple with the OrgName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageFargateHour) GetOrgNameOk() (*string, bool) {
	if o == nil || o.OrgName == nil {
		return nil, false
	}
	return o.OrgName, true
}

// HasOrgName returns a boolean if a field has been set.
func (o *UsageFargateHour) HasOrgName() bool {
	return o != nil && o.OrgName != nil
}

// SetOrgName gets a reference to the given string and assigns it to the OrgName field.
func (o *UsageFargateHour) SetOrgName(v string) {
	o.OrgName = &v
}

// GetPublicId returns the PublicId field value if set, zero value otherwise.
func (o *UsageFargateHour) GetPublicId() string {
	if o == nil || o.PublicId == nil {
		var ret string
		return ret
	}
	return *o.PublicId
}

// GetPublicIdOk returns a tuple with the PublicId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageFargateHour) GetPublicIdOk() (*string, bool) {
	if o == nil || o.PublicId == nil {
		return nil, false
	}
	return o.PublicId, true
}

// HasPublicId returns a boolean if a field has been set.
func (o *UsageFargateHour) HasPublicId() bool {
	return o != nil && o.PublicId != nil
}

// SetPublicId gets a reference to the given string and assigns it to the PublicId field.
func (o *UsageFargateHour) SetPublicId(v string) {
	o.PublicId = &v
}

// GetTasksCount returns the TasksCount field value if set, zero value otherwise.
func (o *UsageFargateHour) GetTasksCount() int64 {
	if o == nil || o.TasksCount == nil {
		var ret int64
		return ret
	}
	return *o.TasksCount
}

// GetTasksCountOk returns a tuple with the TasksCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageFargateHour) GetTasksCountOk() (*int64, bool) {
	if o == nil || o.TasksCount == nil {
		return nil, false
	}
	return o.TasksCount, true
}

// HasTasksCount returns a boolean if a field has been set.
func (o *UsageFargateHour) HasTasksCount() bool {
	return o != nil && o.TasksCount != nil
}

// SetTasksCount gets a reference to the given int64 and assigns it to the TasksCount field.
func (o *UsageFargateHour) SetTasksCount(v int64) {
	o.TasksCount = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageFargateHour) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.ApmFargateCount != nil {
		toSerialize["apm_fargate_count"] = o.ApmFargateCount
	}
	if o.AppsecFargateCount != nil {
		toSerialize["appsec_fargate_count"] = o.AppsecFargateCount
	}
	if o.AvgProfiledFargateTasks != nil {
		toSerialize["avg_profiled_fargate_tasks"] = o.AvgProfiledFargateTasks
	}
	if o.Hour != nil {
		if o.Hour.Nanosecond() == 0 {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["hour"] = o.Hour.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.OrgName != nil {
		toSerialize["org_name"] = o.OrgName
	}
	if o.PublicId != nil {
		toSerialize["public_id"] = o.PublicId
	}
	if o.TasksCount != nil {
		toSerialize["tasks_count"] = o.TasksCount
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageFargateHour) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		ApmFargateCount         *int64     `json:"apm_fargate_count,omitempty"`
		AppsecFargateCount      *int64     `json:"appsec_fargate_count,omitempty"`
		AvgProfiledFargateTasks *int64     `json:"avg_profiled_fargate_tasks,omitempty"`
		Hour                    *time.Time `json:"hour,omitempty"`
		OrgName                 *string    `json:"org_name,omitempty"`
		PublicId                *string    `json:"public_id,omitempty"`
		TasksCount              *int64     `json:"tasks_count,omitempty"`
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
	o.ApmFargateCount = all.ApmFargateCount
	o.AppsecFargateCount = all.AppsecFargateCount
	o.AvgProfiledFargateTasks = all.AvgProfiledFargateTasks
	o.Hour = all.Hour
	o.OrgName = all.OrgName
	o.PublicId = all.PublicId
	o.TasksCount = all.TasksCount
	return nil
}
