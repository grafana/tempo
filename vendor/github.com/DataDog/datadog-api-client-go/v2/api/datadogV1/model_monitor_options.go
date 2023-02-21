// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
)

// MonitorOptions List of options associated with your monitor.
type MonitorOptions struct {
	// Type of aggregation performed in the monitor query.
	Aggregation *MonitorOptionsAggregation `json:"aggregation,omitempty"`
	// IDs of the device the Synthetics monitor is running on.
	// Deprecated
	DeviceIds []MonitorDeviceID `json:"device_ids,omitempty"`
	// Whether or not to send a log sample when the log monitor triggers.
	EnableLogsSample *bool `json:"enable_logs_sample,omitempty"`
	// Whether or not to send a list of samples when the monitor triggers. This is only used by CI Test and Pipeline monitors.
	EnableSamples *bool `json:"enable_samples,omitempty"`
	// We recommend using the [is_renotify](https://docs.datadoghq.com/monitors/notify/?tab=is_alert#renotify),
	// block in the original message instead.
	// A message to include with a re-notification. Supports the `@username` notification we allow elsewhere.
	// Not applicable if `renotify_interval` is `None`.
	EscalationMessage *string `json:"escalation_message,omitempty"`
	// Time (in seconds) to delay evaluation, as a non-negative integer. For example, if the value is set to `300` (5min),
	// the timeframe is set to `last_5m` and the time is 7:00, the monitor evaluates data from 6:50 to 6:55.
	// This is useful for AWS CloudWatch and other backfilled metrics to ensure the monitor always has data during evaluation.
	EvaluationDelay datadog.NullableInt64 `json:"evaluation_delay,omitempty"`
	// The time span after which groups with missing data are dropped from the monitor state.
	// The minimum value is one hour, and the maximum value is 72 hours.
	// Example values are: "60m", "1h", and "2d".
	// This option is only available for APM Trace Analytics, Audit Trail, CI, Error Tracking, Event, Logs, and RUM monitors.
	GroupRetentionDuration *string `json:"group_retention_duration,omitempty"`
	// Whether the log alert monitor triggers a single alert or multiple alerts when any group breaches a threshold.
	GroupbySimpleMonitor *bool `json:"groupby_simple_monitor,omitempty"`
	// A Boolean indicating whether notifications from this monitor automatically inserts its triggering tags into the title.
	//
	// **Examples**
	// - If `True`, `[Triggered on {host:h1}] Monitor Title`
	// - If `False`, `[Triggered] Monitor Title`
	IncludeTags *bool `json:"include_tags,omitempty"`
	// Whether or not the monitor is locked (only editable by creator and admins). Use `restricted_roles` instead.
	// Deprecated
	Locked *bool `json:"locked,omitempty"`
	// How long the test should be in failure before alerting (integer, number of seconds, max 7200).
	MinFailureDuration datadog.NullableInt64 `json:"min_failure_duration,omitempty"`
	// The minimum number of locations in failure at the same time during
	// at least one moment in the `min_failure_duration` period (`min_location_failed` and `min_failure_duration`
	// are part of the advanced alerting rules - integer, >= 1).
	MinLocationFailed datadog.NullableInt64 `json:"min_location_failed,omitempty"`
	// Time (in seconds) to skip evaluations for new groups.
	//
	// For example, this option can be used to skip evaluations for new hosts while they initialize.
	//
	// Must be a non negative integer.
	NewGroupDelay datadog.NullableInt64 `json:"new_group_delay,omitempty"`
	// Time (in seconds) to allow a host to boot and applications
	// to fully start before starting the evaluation of monitor results.
	// Should be a non negative integer.
	//
	// Use new_group_delay instead.
	// Deprecated
	NewHostDelay datadog.NullableInt64 `json:"new_host_delay,omitempty"`
	// The number of minutes before a monitor notifies after data stops reporting.
	// Datadog recommends at least 2x the monitor timeframe for query alerts or 2 minutes for service checks.
	// If omitted, 2x the evaluation timeframe is used for query alerts, and 24 hours is used for service checks.
	NoDataTimeframe datadog.NullableInt64 `json:"no_data_timeframe,omitempty"`
	// Toggles the display of additional content sent in the monitor notification.
	NotificationPresetName *MonitorOptionsNotificationPresets `json:"notification_preset_name,omitempty"`
	// A Boolean indicating whether tagged users is notified on changes to this monitor.
	NotifyAudit *bool `json:"notify_audit,omitempty"`
	// Controls what granularity a monitor alerts on. Only available for monitors with groupings.
	// For instance, a monitor grouped by `cluster`, `namespace`, and `pod` can be configured to only notify on each
	// new `cluster` violating the alert conditions by setting `notify_by` to `["cluster"]`. Tags mentioned
	// in `notify_by` must be a subset of the grouping tags in the query.
	// For example, a query grouped by `cluster` and `namespace` cannot notify on `region`.
	// Setting `notify_by` to `[*]` configures the monitor to notify as a simple-alert.
	NotifyBy []string `json:"notify_by,omitempty"`
	// A Boolean indicating whether this monitor notifies when data stops reporting.
	NotifyNoData *bool `json:"notify_no_data,omitempty"`
	// Controls how groups or monitors are treated if an evaluation does not return any data points.
	// The default option results in different behavior depending on the monitor query type.
	// For monitors using Count queries, an empty monitor evaluation is treated as 0 and is compared to the threshold conditions.
	// For monitors using any query type other than Count, for example Gauge, Measure, or Rate, the monitor shows the last known status.
	// This option is only available for APM Trace Analytics, Audit Trail, CI, Error Tracking, Event, Logs, and RUM monitors.
	OnMissingData *OnMissingDataOption `json:"on_missing_data,omitempty"`
	// The number of minutes after the last notification before a monitor re-notifies on the current status.
	// It only re-notifies if it’s not resolved.
	RenotifyInterval datadog.NullableInt64 `json:"renotify_interval,omitempty"`
	// The number of times re-notification messages should be sent on the current status at the provided re-notification interval.
	RenotifyOccurrences datadog.NullableInt64 `json:"renotify_occurrences,omitempty"`
	// The types of monitor statuses for which re-notification messages are sent.
	RenotifyStatuses []MonitorRenotifyStatusType `json:"renotify_statuses,omitempty"`
	// A Boolean indicating whether this monitor needs a full window of data before it’s evaluated.
	// We highly recommend you set this to `false` for sparse metrics,
	// otherwise some evaluations are skipped. Default is false.
	RequireFullWindow *bool `json:"require_full_window,omitempty"`
	// Configuration options for scheduling.
	SchedulingOptions *MonitorOptionsSchedulingOptions `json:"scheduling_options,omitempty"`
	// Information about the downtime applied to the monitor.
	// Deprecated
	Silenced map[string]int64 `json:"silenced,omitempty"`
	// ID of the corresponding Synthetic check.
	// Deprecated
	SyntheticsCheckId datadog.NullableString `json:"synthetics_check_id,omitempty"`
	// Alerting time window options.
	ThresholdWindows *MonitorThresholdWindowOptions `json:"threshold_windows,omitempty"`
	// List of the different monitor threshold available.
	Thresholds *MonitorThresholds `json:"thresholds,omitempty"`
	// The number of hours of the monitor not reporting data before it automatically resolves from a triggered state. The minimum allowed value is 0 hours. The maximum allowed value is 24 hours.
	TimeoutH datadog.NullableInt64 `json:"timeout_h,omitempty"`
	// List of requests that can be used in the monitor query. **This feature is currently in beta.**
	Variables []MonitorFormulaAndFunctionQueryDefinition `json:"variables,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewMonitorOptions instantiates a new MonitorOptions object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewMonitorOptions() *MonitorOptions {
	this := MonitorOptions{}
	var escalationMessage string = "none"
	this.EscalationMessage = &escalationMessage
	var includeTags bool = true
	this.IncludeTags = &includeTags
	var minFailureDuration int64 = 0
	this.MinFailureDuration = *datadog.NewNullableInt64(&minFailureDuration)
	var minLocationFailed int64 = 1
	this.MinLocationFailed = *datadog.NewNullableInt64(&minLocationFailed)
	var newHostDelay int64 = 300
	this.NewHostDelay = *datadog.NewNullableInt64(&newHostDelay)
	var notificationPresetName MonitorOptionsNotificationPresets = MONITOROPTIONSNOTIFICATIONPRESETS_SHOW_ALL
	this.NotificationPresetName = &notificationPresetName
	var notifyAudit bool = false
	this.NotifyAudit = &notifyAudit
	var notifyNoData bool = false
	this.NotifyNoData = &notifyNoData
	this.RenotifyInterval = *datadog.NewNullableInt64(nil)
	this.TimeoutH = *datadog.NewNullableInt64(nil)
	return &this
}

// NewMonitorOptionsWithDefaults instantiates a new MonitorOptions object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewMonitorOptionsWithDefaults() *MonitorOptions {
	this := MonitorOptions{}
	var escalationMessage string = "none"
	this.EscalationMessage = &escalationMessage
	var includeTags bool = true
	this.IncludeTags = &includeTags
	var minFailureDuration int64 = 0
	this.MinFailureDuration = *datadog.NewNullableInt64(&minFailureDuration)
	var minLocationFailed int64 = 1
	this.MinLocationFailed = *datadog.NewNullableInt64(&minLocationFailed)
	var newHostDelay int64 = 300
	this.NewHostDelay = *datadog.NewNullableInt64(&newHostDelay)
	var notificationPresetName MonitorOptionsNotificationPresets = MONITOROPTIONSNOTIFICATIONPRESETS_SHOW_ALL
	this.NotificationPresetName = &notificationPresetName
	var notifyAudit bool = false
	this.NotifyAudit = &notifyAudit
	var notifyNoData bool = false
	this.NotifyNoData = &notifyNoData
	this.RenotifyInterval = *datadog.NewNullableInt64(nil)
	this.TimeoutH = *datadog.NewNullableInt64(nil)
	return &this
}

// GetAggregation returns the Aggregation field value if set, zero value otherwise.
func (o *MonitorOptions) GetAggregation() MonitorOptionsAggregation {
	if o == nil || o.Aggregation == nil {
		var ret MonitorOptionsAggregation
		return ret
	}
	return *o.Aggregation
}

// GetAggregationOk returns a tuple with the Aggregation field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetAggregationOk() (*MonitorOptionsAggregation, bool) {
	if o == nil || o.Aggregation == nil {
		return nil, false
	}
	return o.Aggregation, true
}

// HasAggregation returns a boolean if a field has been set.
func (o *MonitorOptions) HasAggregation() bool {
	return o != nil && o.Aggregation != nil
}

// SetAggregation gets a reference to the given MonitorOptionsAggregation and assigns it to the Aggregation field.
func (o *MonitorOptions) SetAggregation(v MonitorOptionsAggregation) {
	o.Aggregation = &v
}

// GetDeviceIds returns the DeviceIds field value if set, zero value otherwise.
// Deprecated
func (o *MonitorOptions) GetDeviceIds() []MonitorDeviceID {
	if o == nil || o.DeviceIds == nil {
		var ret []MonitorDeviceID
		return ret
	}
	return o.DeviceIds
}

// GetDeviceIdsOk returns a tuple with the DeviceIds field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *MonitorOptions) GetDeviceIdsOk() (*[]MonitorDeviceID, bool) {
	if o == nil || o.DeviceIds == nil {
		return nil, false
	}
	return &o.DeviceIds, true
}

// HasDeviceIds returns a boolean if a field has been set.
func (o *MonitorOptions) HasDeviceIds() bool {
	return o != nil && o.DeviceIds != nil
}

// SetDeviceIds gets a reference to the given []MonitorDeviceID and assigns it to the DeviceIds field.
// Deprecated
func (o *MonitorOptions) SetDeviceIds(v []MonitorDeviceID) {
	o.DeviceIds = v
}

// GetEnableLogsSample returns the EnableLogsSample field value if set, zero value otherwise.
func (o *MonitorOptions) GetEnableLogsSample() bool {
	if o == nil || o.EnableLogsSample == nil {
		var ret bool
		return ret
	}
	return *o.EnableLogsSample
}

// GetEnableLogsSampleOk returns a tuple with the EnableLogsSample field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetEnableLogsSampleOk() (*bool, bool) {
	if o == nil || o.EnableLogsSample == nil {
		return nil, false
	}
	return o.EnableLogsSample, true
}

// HasEnableLogsSample returns a boolean if a field has been set.
func (o *MonitorOptions) HasEnableLogsSample() bool {
	return o != nil && o.EnableLogsSample != nil
}

// SetEnableLogsSample gets a reference to the given bool and assigns it to the EnableLogsSample field.
func (o *MonitorOptions) SetEnableLogsSample(v bool) {
	o.EnableLogsSample = &v
}

// GetEnableSamples returns the EnableSamples field value if set, zero value otherwise.
func (o *MonitorOptions) GetEnableSamples() bool {
	if o == nil || o.EnableSamples == nil {
		var ret bool
		return ret
	}
	return *o.EnableSamples
}

// GetEnableSamplesOk returns a tuple with the EnableSamples field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetEnableSamplesOk() (*bool, bool) {
	if o == nil || o.EnableSamples == nil {
		return nil, false
	}
	return o.EnableSamples, true
}

// HasEnableSamples returns a boolean if a field has been set.
func (o *MonitorOptions) HasEnableSamples() bool {
	return o != nil && o.EnableSamples != nil
}

// SetEnableSamples gets a reference to the given bool and assigns it to the EnableSamples field.
func (o *MonitorOptions) SetEnableSamples(v bool) {
	o.EnableSamples = &v
}

// GetEscalationMessage returns the EscalationMessage field value if set, zero value otherwise.
func (o *MonitorOptions) GetEscalationMessage() string {
	if o == nil || o.EscalationMessage == nil {
		var ret string
		return ret
	}
	return *o.EscalationMessage
}

// GetEscalationMessageOk returns a tuple with the EscalationMessage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetEscalationMessageOk() (*string, bool) {
	if o == nil || o.EscalationMessage == nil {
		return nil, false
	}
	return o.EscalationMessage, true
}

// HasEscalationMessage returns a boolean if a field has been set.
func (o *MonitorOptions) HasEscalationMessage() bool {
	return o != nil && o.EscalationMessage != nil
}

// SetEscalationMessage gets a reference to the given string and assigns it to the EscalationMessage field.
func (o *MonitorOptions) SetEscalationMessage(v string) {
	o.EscalationMessage = &v
}

// GetEvaluationDelay returns the EvaluationDelay field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorOptions) GetEvaluationDelay() int64 {
	if o == nil || o.EvaluationDelay.Get() == nil {
		var ret int64
		return ret
	}
	return *o.EvaluationDelay.Get()
}

// GetEvaluationDelayOk returns a tuple with the EvaluationDelay field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorOptions) GetEvaluationDelayOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.EvaluationDelay.Get(), o.EvaluationDelay.IsSet()
}

// HasEvaluationDelay returns a boolean if a field has been set.
func (o *MonitorOptions) HasEvaluationDelay() bool {
	return o != nil && o.EvaluationDelay.IsSet()
}

// SetEvaluationDelay gets a reference to the given datadog.NullableInt64 and assigns it to the EvaluationDelay field.
func (o *MonitorOptions) SetEvaluationDelay(v int64) {
	o.EvaluationDelay.Set(&v)
}

// SetEvaluationDelayNil sets the value for EvaluationDelay to be an explicit nil.
func (o *MonitorOptions) SetEvaluationDelayNil() {
	o.EvaluationDelay.Set(nil)
}

// UnsetEvaluationDelay ensures that no value is present for EvaluationDelay, not even an explicit nil.
func (o *MonitorOptions) UnsetEvaluationDelay() {
	o.EvaluationDelay.Unset()
}

// GetGroupRetentionDuration returns the GroupRetentionDuration field value if set, zero value otherwise.
func (o *MonitorOptions) GetGroupRetentionDuration() string {
	if o == nil || o.GroupRetentionDuration == nil {
		var ret string
		return ret
	}
	return *o.GroupRetentionDuration
}

// GetGroupRetentionDurationOk returns a tuple with the GroupRetentionDuration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetGroupRetentionDurationOk() (*string, bool) {
	if o == nil || o.GroupRetentionDuration == nil {
		return nil, false
	}
	return o.GroupRetentionDuration, true
}

// HasGroupRetentionDuration returns a boolean if a field has been set.
func (o *MonitorOptions) HasGroupRetentionDuration() bool {
	return o != nil && o.GroupRetentionDuration != nil
}

// SetGroupRetentionDuration gets a reference to the given string and assigns it to the GroupRetentionDuration field.
func (o *MonitorOptions) SetGroupRetentionDuration(v string) {
	o.GroupRetentionDuration = &v
}

// GetGroupbySimpleMonitor returns the GroupbySimpleMonitor field value if set, zero value otherwise.
func (o *MonitorOptions) GetGroupbySimpleMonitor() bool {
	if o == nil || o.GroupbySimpleMonitor == nil {
		var ret bool
		return ret
	}
	return *o.GroupbySimpleMonitor
}

// GetGroupbySimpleMonitorOk returns a tuple with the GroupbySimpleMonitor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetGroupbySimpleMonitorOk() (*bool, bool) {
	if o == nil || o.GroupbySimpleMonitor == nil {
		return nil, false
	}
	return o.GroupbySimpleMonitor, true
}

// HasGroupbySimpleMonitor returns a boolean if a field has been set.
func (o *MonitorOptions) HasGroupbySimpleMonitor() bool {
	return o != nil && o.GroupbySimpleMonitor != nil
}

// SetGroupbySimpleMonitor gets a reference to the given bool and assigns it to the GroupbySimpleMonitor field.
func (o *MonitorOptions) SetGroupbySimpleMonitor(v bool) {
	o.GroupbySimpleMonitor = &v
}

// GetIncludeTags returns the IncludeTags field value if set, zero value otherwise.
func (o *MonitorOptions) GetIncludeTags() bool {
	if o == nil || o.IncludeTags == nil {
		var ret bool
		return ret
	}
	return *o.IncludeTags
}

// GetIncludeTagsOk returns a tuple with the IncludeTags field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetIncludeTagsOk() (*bool, bool) {
	if o == nil || o.IncludeTags == nil {
		return nil, false
	}
	return o.IncludeTags, true
}

// HasIncludeTags returns a boolean if a field has been set.
func (o *MonitorOptions) HasIncludeTags() bool {
	return o != nil && o.IncludeTags != nil
}

// SetIncludeTags gets a reference to the given bool and assigns it to the IncludeTags field.
func (o *MonitorOptions) SetIncludeTags(v bool) {
	o.IncludeTags = &v
}

// GetLocked returns the Locked field value if set, zero value otherwise.
// Deprecated
func (o *MonitorOptions) GetLocked() bool {
	if o == nil || o.Locked == nil {
		var ret bool
		return ret
	}
	return *o.Locked
}

// GetLockedOk returns a tuple with the Locked field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *MonitorOptions) GetLockedOk() (*bool, bool) {
	if o == nil || o.Locked == nil {
		return nil, false
	}
	return o.Locked, true
}

// HasLocked returns a boolean if a field has been set.
func (o *MonitorOptions) HasLocked() bool {
	return o != nil && o.Locked != nil
}

// SetLocked gets a reference to the given bool and assigns it to the Locked field.
// Deprecated
func (o *MonitorOptions) SetLocked(v bool) {
	o.Locked = &v
}

// GetMinFailureDuration returns the MinFailureDuration field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorOptions) GetMinFailureDuration() int64 {
	if o == nil || o.MinFailureDuration.Get() == nil {
		var ret int64
		return ret
	}
	return *o.MinFailureDuration.Get()
}

// GetMinFailureDurationOk returns a tuple with the MinFailureDuration field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorOptions) GetMinFailureDurationOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.MinFailureDuration.Get(), o.MinFailureDuration.IsSet()
}

// HasMinFailureDuration returns a boolean if a field has been set.
func (o *MonitorOptions) HasMinFailureDuration() bool {
	return o != nil && o.MinFailureDuration.IsSet()
}

// SetMinFailureDuration gets a reference to the given datadog.NullableInt64 and assigns it to the MinFailureDuration field.
func (o *MonitorOptions) SetMinFailureDuration(v int64) {
	o.MinFailureDuration.Set(&v)
}

// SetMinFailureDurationNil sets the value for MinFailureDuration to be an explicit nil.
func (o *MonitorOptions) SetMinFailureDurationNil() {
	o.MinFailureDuration.Set(nil)
}

// UnsetMinFailureDuration ensures that no value is present for MinFailureDuration, not even an explicit nil.
func (o *MonitorOptions) UnsetMinFailureDuration() {
	o.MinFailureDuration.Unset()
}

// GetMinLocationFailed returns the MinLocationFailed field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorOptions) GetMinLocationFailed() int64 {
	if o == nil || o.MinLocationFailed.Get() == nil {
		var ret int64
		return ret
	}
	return *o.MinLocationFailed.Get()
}

// GetMinLocationFailedOk returns a tuple with the MinLocationFailed field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorOptions) GetMinLocationFailedOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.MinLocationFailed.Get(), o.MinLocationFailed.IsSet()
}

// HasMinLocationFailed returns a boolean if a field has been set.
func (o *MonitorOptions) HasMinLocationFailed() bool {
	return o != nil && o.MinLocationFailed.IsSet()
}

// SetMinLocationFailed gets a reference to the given datadog.NullableInt64 and assigns it to the MinLocationFailed field.
func (o *MonitorOptions) SetMinLocationFailed(v int64) {
	o.MinLocationFailed.Set(&v)
}

// SetMinLocationFailedNil sets the value for MinLocationFailed to be an explicit nil.
func (o *MonitorOptions) SetMinLocationFailedNil() {
	o.MinLocationFailed.Set(nil)
}

// UnsetMinLocationFailed ensures that no value is present for MinLocationFailed, not even an explicit nil.
func (o *MonitorOptions) UnsetMinLocationFailed() {
	o.MinLocationFailed.Unset()
}

// GetNewGroupDelay returns the NewGroupDelay field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorOptions) GetNewGroupDelay() int64 {
	if o == nil || o.NewGroupDelay.Get() == nil {
		var ret int64
		return ret
	}
	return *o.NewGroupDelay.Get()
}

// GetNewGroupDelayOk returns a tuple with the NewGroupDelay field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorOptions) GetNewGroupDelayOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.NewGroupDelay.Get(), o.NewGroupDelay.IsSet()
}

// HasNewGroupDelay returns a boolean if a field has been set.
func (o *MonitorOptions) HasNewGroupDelay() bool {
	return o != nil && o.NewGroupDelay.IsSet()
}

// SetNewGroupDelay gets a reference to the given datadog.NullableInt64 and assigns it to the NewGroupDelay field.
func (o *MonitorOptions) SetNewGroupDelay(v int64) {
	o.NewGroupDelay.Set(&v)
}

// SetNewGroupDelayNil sets the value for NewGroupDelay to be an explicit nil.
func (o *MonitorOptions) SetNewGroupDelayNil() {
	o.NewGroupDelay.Set(nil)
}

// UnsetNewGroupDelay ensures that no value is present for NewGroupDelay, not even an explicit nil.
func (o *MonitorOptions) UnsetNewGroupDelay() {
	o.NewGroupDelay.Unset()
}

// GetNewHostDelay returns the NewHostDelay field value if set, zero value otherwise (both if not set or set to explicit null).
// Deprecated
func (o *MonitorOptions) GetNewHostDelay() int64 {
	if o == nil || o.NewHostDelay.Get() == nil {
		var ret int64
		return ret
	}
	return *o.NewHostDelay.Get()
}

// GetNewHostDelayOk returns a tuple with the NewHostDelay field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
// Deprecated
func (o *MonitorOptions) GetNewHostDelayOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.NewHostDelay.Get(), o.NewHostDelay.IsSet()
}

// HasNewHostDelay returns a boolean if a field has been set.
func (o *MonitorOptions) HasNewHostDelay() bool {
	return o != nil && o.NewHostDelay.IsSet()
}

// SetNewHostDelay gets a reference to the given datadog.NullableInt64 and assigns it to the NewHostDelay field.
// Deprecated
func (o *MonitorOptions) SetNewHostDelay(v int64) {
	o.NewHostDelay.Set(&v)
}

// SetNewHostDelayNil sets the value for NewHostDelay to be an explicit nil.
func (o *MonitorOptions) SetNewHostDelayNil() {
	o.NewHostDelay.Set(nil)
}

// UnsetNewHostDelay ensures that no value is present for NewHostDelay, not even an explicit nil.
func (o *MonitorOptions) UnsetNewHostDelay() {
	o.NewHostDelay.Unset()
}

// GetNoDataTimeframe returns the NoDataTimeframe field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorOptions) GetNoDataTimeframe() int64 {
	if o == nil || o.NoDataTimeframe.Get() == nil {
		var ret int64
		return ret
	}
	return *o.NoDataTimeframe.Get()
}

// GetNoDataTimeframeOk returns a tuple with the NoDataTimeframe field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorOptions) GetNoDataTimeframeOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.NoDataTimeframe.Get(), o.NoDataTimeframe.IsSet()
}

// HasNoDataTimeframe returns a boolean if a field has been set.
func (o *MonitorOptions) HasNoDataTimeframe() bool {
	return o != nil && o.NoDataTimeframe.IsSet()
}

// SetNoDataTimeframe gets a reference to the given datadog.NullableInt64 and assigns it to the NoDataTimeframe field.
func (o *MonitorOptions) SetNoDataTimeframe(v int64) {
	o.NoDataTimeframe.Set(&v)
}

// SetNoDataTimeframeNil sets the value for NoDataTimeframe to be an explicit nil.
func (o *MonitorOptions) SetNoDataTimeframeNil() {
	o.NoDataTimeframe.Set(nil)
}

// UnsetNoDataTimeframe ensures that no value is present for NoDataTimeframe, not even an explicit nil.
func (o *MonitorOptions) UnsetNoDataTimeframe() {
	o.NoDataTimeframe.Unset()
}

// GetNotificationPresetName returns the NotificationPresetName field value if set, zero value otherwise.
func (o *MonitorOptions) GetNotificationPresetName() MonitorOptionsNotificationPresets {
	if o == nil || o.NotificationPresetName == nil {
		var ret MonitorOptionsNotificationPresets
		return ret
	}
	return *o.NotificationPresetName
}

// GetNotificationPresetNameOk returns a tuple with the NotificationPresetName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetNotificationPresetNameOk() (*MonitorOptionsNotificationPresets, bool) {
	if o == nil || o.NotificationPresetName == nil {
		return nil, false
	}
	return o.NotificationPresetName, true
}

// HasNotificationPresetName returns a boolean if a field has been set.
func (o *MonitorOptions) HasNotificationPresetName() bool {
	return o != nil && o.NotificationPresetName != nil
}

// SetNotificationPresetName gets a reference to the given MonitorOptionsNotificationPresets and assigns it to the NotificationPresetName field.
func (o *MonitorOptions) SetNotificationPresetName(v MonitorOptionsNotificationPresets) {
	o.NotificationPresetName = &v
}

// GetNotifyAudit returns the NotifyAudit field value if set, zero value otherwise.
func (o *MonitorOptions) GetNotifyAudit() bool {
	if o == nil || o.NotifyAudit == nil {
		var ret bool
		return ret
	}
	return *o.NotifyAudit
}

// GetNotifyAuditOk returns a tuple with the NotifyAudit field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetNotifyAuditOk() (*bool, bool) {
	if o == nil || o.NotifyAudit == nil {
		return nil, false
	}
	return o.NotifyAudit, true
}

// HasNotifyAudit returns a boolean if a field has been set.
func (o *MonitorOptions) HasNotifyAudit() bool {
	return o != nil && o.NotifyAudit != nil
}

// SetNotifyAudit gets a reference to the given bool and assigns it to the NotifyAudit field.
func (o *MonitorOptions) SetNotifyAudit(v bool) {
	o.NotifyAudit = &v
}

// GetNotifyBy returns the NotifyBy field value if set, zero value otherwise.
func (o *MonitorOptions) GetNotifyBy() []string {
	if o == nil || o.NotifyBy == nil {
		var ret []string
		return ret
	}
	return o.NotifyBy
}

// GetNotifyByOk returns a tuple with the NotifyBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetNotifyByOk() (*[]string, bool) {
	if o == nil || o.NotifyBy == nil {
		return nil, false
	}
	return &o.NotifyBy, true
}

// HasNotifyBy returns a boolean if a field has been set.
func (o *MonitorOptions) HasNotifyBy() bool {
	return o != nil && o.NotifyBy != nil
}

// SetNotifyBy gets a reference to the given []string and assigns it to the NotifyBy field.
func (o *MonitorOptions) SetNotifyBy(v []string) {
	o.NotifyBy = v
}

// GetNotifyNoData returns the NotifyNoData field value if set, zero value otherwise.
func (o *MonitorOptions) GetNotifyNoData() bool {
	if o == nil || o.NotifyNoData == nil {
		var ret bool
		return ret
	}
	return *o.NotifyNoData
}

// GetNotifyNoDataOk returns a tuple with the NotifyNoData field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetNotifyNoDataOk() (*bool, bool) {
	if o == nil || o.NotifyNoData == nil {
		return nil, false
	}
	return o.NotifyNoData, true
}

// HasNotifyNoData returns a boolean if a field has been set.
func (o *MonitorOptions) HasNotifyNoData() bool {
	return o != nil && o.NotifyNoData != nil
}

// SetNotifyNoData gets a reference to the given bool and assigns it to the NotifyNoData field.
func (o *MonitorOptions) SetNotifyNoData(v bool) {
	o.NotifyNoData = &v
}

// GetOnMissingData returns the OnMissingData field value if set, zero value otherwise.
func (o *MonitorOptions) GetOnMissingData() OnMissingDataOption {
	if o == nil || o.OnMissingData == nil {
		var ret OnMissingDataOption
		return ret
	}
	return *o.OnMissingData
}

// GetOnMissingDataOk returns a tuple with the OnMissingData field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetOnMissingDataOk() (*OnMissingDataOption, bool) {
	if o == nil || o.OnMissingData == nil {
		return nil, false
	}
	return o.OnMissingData, true
}

// HasOnMissingData returns a boolean if a field has been set.
func (o *MonitorOptions) HasOnMissingData() bool {
	return o != nil && o.OnMissingData != nil
}

// SetOnMissingData gets a reference to the given OnMissingDataOption and assigns it to the OnMissingData field.
func (o *MonitorOptions) SetOnMissingData(v OnMissingDataOption) {
	o.OnMissingData = &v
}

// GetRenotifyInterval returns the RenotifyInterval field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorOptions) GetRenotifyInterval() int64 {
	if o == nil || o.RenotifyInterval.Get() == nil {
		var ret int64
		return ret
	}
	return *o.RenotifyInterval.Get()
}

// GetRenotifyIntervalOk returns a tuple with the RenotifyInterval field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorOptions) GetRenotifyIntervalOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.RenotifyInterval.Get(), o.RenotifyInterval.IsSet()
}

// HasRenotifyInterval returns a boolean if a field has been set.
func (o *MonitorOptions) HasRenotifyInterval() bool {
	return o != nil && o.RenotifyInterval.IsSet()
}

// SetRenotifyInterval gets a reference to the given datadog.NullableInt64 and assigns it to the RenotifyInterval field.
func (o *MonitorOptions) SetRenotifyInterval(v int64) {
	o.RenotifyInterval.Set(&v)
}

// SetRenotifyIntervalNil sets the value for RenotifyInterval to be an explicit nil.
func (o *MonitorOptions) SetRenotifyIntervalNil() {
	o.RenotifyInterval.Set(nil)
}

// UnsetRenotifyInterval ensures that no value is present for RenotifyInterval, not even an explicit nil.
func (o *MonitorOptions) UnsetRenotifyInterval() {
	o.RenotifyInterval.Unset()
}

// GetRenotifyOccurrences returns the RenotifyOccurrences field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorOptions) GetRenotifyOccurrences() int64 {
	if o == nil || o.RenotifyOccurrences.Get() == nil {
		var ret int64
		return ret
	}
	return *o.RenotifyOccurrences.Get()
}

// GetRenotifyOccurrencesOk returns a tuple with the RenotifyOccurrences field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorOptions) GetRenotifyOccurrencesOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.RenotifyOccurrences.Get(), o.RenotifyOccurrences.IsSet()
}

// HasRenotifyOccurrences returns a boolean if a field has been set.
func (o *MonitorOptions) HasRenotifyOccurrences() bool {
	return o != nil && o.RenotifyOccurrences.IsSet()
}

// SetRenotifyOccurrences gets a reference to the given datadog.NullableInt64 and assigns it to the RenotifyOccurrences field.
func (o *MonitorOptions) SetRenotifyOccurrences(v int64) {
	o.RenotifyOccurrences.Set(&v)
}

// SetRenotifyOccurrencesNil sets the value for RenotifyOccurrences to be an explicit nil.
func (o *MonitorOptions) SetRenotifyOccurrencesNil() {
	o.RenotifyOccurrences.Set(nil)
}

// UnsetRenotifyOccurrences ensures that no value is present for RenotifyOccurrences, not even an explicit nil.
func (o *MonitorOptions) UnsetRenotifyOccurrences() {
	o.RenotifyOccurrences.Unset()
}

// GetRenotifyStatuses returns the RenotifyStatuses field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorOptions) GetRenotifyStatuses() []MonitorRenotifyStatusType {
	if o == nil {
		var ret []MonitorRenotifyStatusType
		return ret
	}
	return o.RenotifyStatuses
}

// GetRenotifyStatusesOk returns a tuple with the RenotifyStatuses field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorOptions) GetRenotifyStatusesOk() (*[]MonitorRenotifyStatusType, bool) {
	if o == nil || o.RenotifyStatuses == nil {
		return nil, false
	}
	return &o.RenotifyStatuses, true
}

// HasRenotifyStatuses returns a boolean if a field has been set.
func (o *MonitorOptions) HasRenotifyStatuses() bool {
	return o != nil && o.RenotifyStatuses != nil
}

// SetRenotifyStatuses gets a reference to the given []MonitorRenotifyStatusType and assigns it to the RenotifyStatuses field.
func (o *MonitorOptions) SetRenotifyStatuses(v []MonitorRenotifyStatusType) {
	o.RenotifyStatuses = v
}

// GetRequireFullWindow returns the RequireFullWindow field value if set, zero value otherwise.
func (o *MonitorOptions) GetRequireFullWindow() bool {
	if o == nil || o.RequireFullWindow == nil {
		var ret bool
		return ret
	}
	return *o.RequireFullWindow
}

// GetRequireFullWindowOk returns a tuple with the RequireFullWindow field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetRequireFullWindowOk() (*bool, bool) {
	if o == nil || o.RequireFullWindow == nil {
		return nil, false
	}
	return o.RequireFullWindow, true
}

// HasRequireFullWindow returns a boolean if a field has been set.
func (o *MonitorOptions) HasRequireFullWindow() bool {
	return o != nil && o.RequireFullWindow != nil
}

// SetRequireFullWindow gets a reference to the given bool and assigns it to the RequireFullWindow field.
func (o *MonitorOptions) SetRequireFullWindow(v bool) {
	o.RequireFullWindow = &v
}

// GetSchedulingOptions returns the SchedulingOptions field value if set, zero value otherwise.
func (o *MonitorOptions) GetSchedulingOptions() MonitorOptionsSchedulingOptions {
	if o == nil || o.SchedulingOptions == nil {
		var ret MonitorOptionsSchedulingOptions
		return ret
	}
	return *o.SchedulingOptions
}

// GetSchedulingOptionsOk returns a tuple with the SchedulingOptions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetSchedulingOptionsOk() (*MonitorOptionsSchedulingOptions, bool) {
	if o == nil || o.SchedulingOptions == nil {
		return nil, false
	}
	return o.SchedulingOptions, true
}

// HasSchedulingOptions returns a boolean if a field has been set.
func (o *MonitorOptions) HasSchedulingOptions() bool {
	return o != nil && o.SchedulingOptions != nil
}

// SetSchedulingOptions gets a reference to the given MonitorOptionsSchedulingOptions and assigns it to the SchedulingOptions field.
func (o *MonitorOptions) SetSchedulingOptions(v MonitorOptionsSchedulingOptions) {
	o.SchedulingOptions = &v
}

// GetSilenced returns the Silenced field value if set, zero value otherwise.
// Deprecated
func (o *MonitorOptions) GetSilenced() map[string]int64 {
	if o == nil || o.Silenced == nil {
		var ret map[string]int64
		return ret
	}
	return o.Silenced
}

// GetSilencedOk returns a tuple with the Silenced field value if set, nil otherwise
// and a boolean to check if the value has been set.
// Deprecated
func (o *MonitorOptions) GetSilencedOk() (*map[string]int64, bool) {
	if o == nil || o.Silenced == nil {
		return nil, false
	}
	return &o.Silenced, true
}

// HasSilenced returns a boolean if a field has been set.
func (o *MonitorOptions) HasSilenced() bool {
	return o != nil && o.Silenced != nil
}

// SetSilenced gets a reference to the given map[string]int64 and assigns it to the Silenced field.
// Deprecated
func (o *MonitorOptions) SetSilenced(v map[string]int64) {
	o.Silenced = v
}

// GetSyntheticsCheckId returns the SyntheticsCheckId field value if set, zero value otherwise (both if not set or set to explicit null).
// Deprecated
func (o *MonitorOptions) GetSyntheticsCheckId() string {
	if o == nil || o.SyntheticsCheckId.Get() == nil {
		var ret string
		return ret
	}
	return *o.SyntheticsCheckId.Get()
}

// GetSyntheticsCheckIdOk returns a tuple with the SyntheticsCheckId field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
// Deprecated
func (o *MonitorOptions) GetSyntheticsCheckIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.SyntheticsCheckId.Get(), o.SyntheticsCheckId.IsSet()
}

// HasSyntheticsCheckId returns a boolean if a field has been set.
func (o *MonitorOptions) HasSyntheticsCheckId() bool {
	return o != nil && o.SyntheticsCheckId.IsSet()
}

// SetSyntheticsCheckId gets a reference to the given datadog.NullableString and assigns it to the SyntheticsCheckId field.
// Deprecated
func (o *MonitorOptions) SetSyntheticsCheckId(v string) {
	o.SyntheticsCheckId.Set(&v)
}

// SetSyntheticsCheckIdNil sets the value for SyntheticsCheckId to be an explicit nil.
func (o *MonitorOptions) SetSyntheticsCheckIdNil() {
	o.SyntheticsCheckId.Set(nil)
}

// UnsetSyntheticsCheckId ensures that no value is present for SyntheticsCheckId, not even an explicit nil.
func (o *MonitorOptions) UnsetSyntheticsCheckId() {
	o.SyntheticsCheckId.Unset()
}

// GetThresholdWindows returns the ThresholdWindows field value if set, zero value otherwise.
func (o *MonitorOptions) GetThresholdWindows() MonitorThresholdWindowOptions {
	if o == nil || o.ThresholdWindows == nil {
		var ret MonitorThresholdWindowOptions
		return ret
	}
	return *o.ThresholdWindows
}

// GetThresholdWindowsOk returns a tuple with the ThresholdWindows field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetThresholdWindowsOk() (*MonitorThresholdWindowOptions, bool) {
	if o == nil || o.ThresholdWindows == nil {
		return nil, false
	}
	return o.ThresholdWindows, true
}

// HasThresholdWindows returns a boolean if a field has been set.
func (o *MonitorOptions) HasThresholdWindows() bool {
	return o != nil && o.ThresholdWindows != nil
}

// SetThresholdWindows gets a reference to the given MonitorThresholdWindowOptions and assigns it to the ThresholdWindows field.
func (o *MonitorOptions) SetThresholdWindows(v MonitorThresholdWindowOptions) {
	o.ThresholdWindows = &v
}

// GetThresholds returns the Thresholds field value if set, zero value otherwise.
func (o *MonitorOptions) GetThresholds() MonitorThresholds {
	if o == nil || o.Thresholds == nil {
		var ret MonitorThresholds
		return ret
	}
	return *o.Thresholds
}

// GetThresholdsOk returns a tuple with the Thresholds field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetThresholdsOk() (*MonitorThresholds, bool) {
	if o == nil || o.Thresholds == nil {
		return nil, false
	}
	return o.Thresholds, true
}

// HasThresholds returns a boolean if a field has been set.
func (o *MonitorOptions) HasThresholds() bool {
	return o != nil && o.Thresholds != nil
}

// SetThresholds gets a reference to the given MonitorThresholds and assigns it to the Thresholds field.
func (o *MonitorOptions) SetThresholds(v MonitorThresholds) {
	o.Thresholds = &v
}

// GetTimeoutH returns the TimeoutH field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *MonitorOptions) GetTimeoutH() int64 {
	if o == nil || o.TimeoutH.Get() == nil {
		var ret int64
		return ret
	}
	return *o.TimeoutH.Get()
}

// GetTimeoutHOk returns a tuple with the TimeoutH field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned.
func (o *MonitorOptions) GetTimeoutHOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return o.TimeoutH.Get(), o.TimeoutH.IsSet()
}

// HasTimeoutH returns a boolean if a field has been set.
func (o *MonitorOptions) HasTimeoutH() bool {
	return o != nil && o.TimeoutH.IsSet()
}

// SetTimeoutH gets a reference to the given datadog.NullableInt64 and assigns it to the TimeoutH field.
func (o *MonitorOptions) SetTimeoutH(v int64) {
	o.TimeoutH.Set(&v)
}

// SetTimeoutHNil sets the value for TimeoutH to be an explicit nil.
func (o *MonitorOptions) SetTimeoutHNil() {
	o.TimeoutH.Set(nil)
}

// UnsetTimeoutH ensures that no value is present for TimeoutH, not even an explicit nil.
func (o *MonitorOptions) UnsetTimeoutH() {
	o.TimeoutH.Unset()
}

// GetVariables returns the Variables field value if set, zero value otherwise.
func (o *MonitorOptions) GetVariables() []MonitorFormulaAndFunctionQueryDefinition {
	if o == nil || o.Variables == nil {
		var ret []MonitorFormulaAndFunctionQueryDefinition
		return ret
	}
	return o.Variables
}

// GetVariablesOk returns a tuple with the Variables field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *MonitorOptions) GetVariablesOk() (*[]MonitorFormulaAndFunctionQueryDefinition, bool) {
	if o == nil || o.Variables == nil {
		return nil, false
	}
	return &o.Variables, true
}

// HasVariables returns a boolean if a field has been set.
func (o *MonitorOptions) HasVariables() bool {
	return o != nil && o.Variables != nil
}

// SetVariables gets a reference to the given []MonitorFormulaAndFunctionQueryDefinition and assigns it to the Variables field.
func (o *MonitorOptions) SetVariables(v []MonitorFormulaAndFunctionQueryDefinition) {
	o.Variables = v
}

// MarshalJSON serializes the struct using spec logic.
func (o MonitorOptions) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.Aggregation != nil {
		toSerialize["aggregation"] = o.Aggregation
	}
	if o.DeviceIds != nil {
		toSerialize["device_ids"] = o.DeviceIds
	}
	if o.EnableLogsSample != nil {
		toSerialize["enable_logs_sample"] = o.EnableLogsSample
	}
	if o.EnableSamples != nil {
		toSerialize["enable_samples"] = o.EnableSamples
	}
	if o.EscalationMessage != nil {
		toSerialize["escalation_message"] = o.EscalationMessage
	}
	if o.EvaluationDelay.IsSet() {
		toSerialize["evaluation_delay"] = o.EvaluationDelay.Get()
	}
	if o.GroupRetentionDuration != nil {
		toSerialize["group_retention_duration"] = o.GroupRetentionDuration
	}
	if o.GroupbySimpleMonitor != nil {
		toSerialize["groupby_simple_monitor"] = o.GroupbySimpleMonitor
	}
	if o.IncludeTags != nil {
		toSerialize["include_tags"] = o.IncludeTags
	}
	if o.Locked != nil {
		toSerialize["locked"] = o.Locked
	}
	if o.MinFailureDuration.IsSet() {
		toSerialize["min_failure_duration"] = o.MinFailureDuration.Get()
	}
	if o.MinLocationFailed.IsSet() {
		toSerialize["min_location_failed"] = o.MinLocationFailed.Get()
	}
	if o.NewGroupDelay.IsSet() {
		toSerialize["new_group_delay"] = o.NewGroupDelay.Get()
	}
	if o.NewHostDelay.IsSet() {
		toSerialize["new_host_delay"] = o.NewHostDelay.Get()
	}
	if o.NoDataTimeframe.IsSet() {
		toSerialize["no_data_timeframe"] = o.NoDataTimeframe.Get()
	}
	if o.NotificationPresetName != nil {
		toSerialize["notification_preset_name"] = o.NotificationPresetName
	}
	if o.NotifyAudit != nil {
		toSerialize["notify_audit"] = o.NotifyAudit
	}
	if o.NotifyBy != nil {
		toSerialize["notify_by"] = o.NotifyBy
	}
	if o.NotifyNoData != nil {
		toSerialize["notify_no_data"] = o.NotifyNoData
	}
	if o.OnMissingData != nil {
		toSerialize["on_missing_data"] = o.OnMissingData
	}
	if o.RenotifyInterval.IsSet() {
		toSerialize["renotify_interval"] = o.RenotifyInterval.Get()
	}
	if o.RenotifyOccurrences.IsSet() {
		toSerialize["renotify_occurrences"] = o.RenotifyOccurrences.Get()
	}
	if o.RenotifyStatuses != nil {
		toSerialize["renotify_statuses"] = o.RenotifyStatuses
	}
	if o.RequireFullWindow != nil {
		toSerialize["require_full_window"] = o.RequireFullWindow
	}
	if o.SchedulingOptions != nil {
		toSerialize["scheduling_options"] = o.SchedulingOptions
	}
	if o.Silenced != nil {
		toSerialize["silenced"] = o.Silenced
	}
	if o.SyntheticsCheckId.IsSet() {
		toSerialize["synthetics_check_id"] = o.SyntheticsCheckId.Get()
	}
	if o.ThresholdWindows != nil {
		toSerialize["threshold_windows"] = o.ThresholdWindows
	}
	if o.Thresholds != nil {
		toSerialize["thresholds"] = o.Thresholds
	}
	if o.TimeoutH.IsSet() {
		toSerialize["timeout_h"] = o.TimeoutH.Get()
	}
	if o.Variables != nil {
		toSerialize["variables"] = o.Variables
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *MonitorOptions) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		Aggregation            *MonitorOptionsAggregation                 `json:"aggregation,omitempty"`
		DeviceIds              []MonitorDeviceID                          `json:"device_ids,omitempty"`
		EnableLogsSample       *bool                                      `json:"enable_logs_sample,omitempty"`
		EnableSamples          *bool                                      `json:"enable_samples,omitempty"`
		EscalationMessage      *string                                    `json:"escalation_message,omitempty"`
		EvaluationDelay        datadog.NullableInt64                      `json:"evaluation_delay,omitempty"`
		GroupRetentionDuration *string                                    `json:"group_retention_duration,omitempty"`
		GroupbySimpleMonitor   *bool                                      `json:"groupby_simple_monitor,omitempty"`
		IncludeTags            *bool                                      `json:"include_tags,omitempty"`
		Locked                 *bool                                      `json:"locked,omitempty"`
		MinFailureDuration     datadog.NullableInt64                      `json:"min_failure_duration,omitempty"`
		MinLocationFailed      datadog.NullableInt64                      `json:"min_location_failed,omitempty"`
		NewGroupDelay          datadog.NullableInt64                      `json:"new_group_delay,omitempty"`
		NewHostDelay           datadog.NullableInt64                      `json:"new_host_delay,omitempty"`
		NoDataTimeframe        datadog.NullableInt64                      `json:"no_data_timeframe,omitempty"`
		NotificationPresetName *MonitorOptionsNotificationPresets         `json:"notification_preset_name,omitempty"`
		NotifyAudit            *bool                                      `json:"notify_audit,omitempty"`
		NotifyBy               []string                                   `json:"notify_by,omitempty"`
		NotifyNoData           *bool                                      `json:"notify_no_data,omitempty"`
		OnMissingData          *OnMissingDataOption                       `json:"on_missing_data,omitempty"`
		RenotifyInterval       datadog.NullableInt64                      `json:"renotify_interval,omitempty"`
		RenotifyOccurrences    datadog.NullableInt64                      `json:"renotify_occurrences,omitempty"`
		RenotifyStatuses       []MonitorRenotifyStatusType                `json:"renotify_statuses,omitempty"`
		RequireFullWindow      *bool                                      `json:"require_full_window,omitempty"`
		SchedulingOptions      *MonitorOptionsSchedulingOptions           `json:"scheduling_options,omitempty"`
		Silenced               map[string]int64                           `json:"silenced,omitempty"`
		SyntheticsCheckId      datadog.NullableString                     `json:"synthetics_check_id,omitempty"`
		ThresholdWindows       *MonitorThresholdWindowOptions             `json:"threshold_windows,omitempty"`
		Thresholds             *MonitorThresholds                         `json:"thresholds,omitempty"`
		TimeoutH               datadog.NullableInt64                      `json:"timeout_h,omitempty"`
		Variables              []MonitorFormulaAndFunctionQueryDefinition `json:"variables,omitempty"`
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
	if v := all.NotificationPresetName; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if v := all.OnMissingData; v != nil && !v.IsValid() {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
		return nil
	}
	if all.Aggregation != nil && all.Aggregation.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Aggregation = all.Aggregation
	o.DeviceIds = all.DeviceIds
	o.EnableLogsSample = all.EnableLogsSample
	o.EnableSamples = all.EnableSamples
	o.EscalationMessage = all.EscalationMessage
	o.EvaluationDelay = all.EvaluationDelay
	o.GroupRetentionDuration = all.GroupRetentionDuration
	o.GroupbySimpleMonitor = all.GroupbySimpleMonitor
	o.IncludeTags = all.IncludeTags
	o.Locked = all.Locked
	o.MinFailureDuration = all.MinFailureDuration
	o.MinLocationFailed = all.MinLocationFailed
	o.NewGroupDelay = all.NewGroupDelay
	o.NewHostDelay = all.NewHostDelay
	o.NoDataTimeframe = all.NoDataTimeframe
	o.NotificationPresetName = all.NotificationPresetName
	o.NotifyAudit = all.NotifyAudit
	o.NotifyBy = all.NotifyBy
	o.NotifyNoData = all.NotifyNoData
	o.OnMissingData = all.OnMissingData
	o.RenotifyInterval = all.RenotifyInterval
	o.RenotifyOccurrences = all.RenotifyOccurrences
	o.RenotifyStatuses = all.RenotifyStatuses
	o.RequireFullWindow = all.RequireFullWindow
	if all.SchedulingOptions != nil && all.SchedulingOptions.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.SchedulingOptions = all.SchedulingOptions
	o.Silenced = all.Silenced
	o.SyntheticsCheckId = all.SyntheticsCheckId
	if all.ThresholdWindows != nil && all.ThresholdWindows.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.ThresholdWindows = all.ThresholdWindows
	if all.Thresholds != nil && all.Thresholds.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.Thresholds = all.Thresholds
	o.TimeoutH = all.TimeoutH
	o.Variables = all.Variables
	return nil
}
