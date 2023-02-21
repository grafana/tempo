// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"time"
)

// UsageSummaryDate Response with hourly report of all data billed by Datadog all organizations.
type UsageSummaryDate struct {
	// Shows the 99th percentile of all agent hosts over all hours in the current date for all organizations.
	AgentHostTop99p *int64 `json:"agent_host_top99p,omitempty"`
	// Shows the 99th percentile of all Azure app services using APM over all hours in the current date all organizations.
	ApmAzureAppServiceHostTop99p *int64 `json:"apm_azure_app_service_host_top99p,omitempty"`
	// Shows the average of all APM ECS Fargate tasks over all hours in the current date for all organizations.
	ApmFargateCountAvg *int64 `json:"apm_fargate_count_avg,omitempty"`
	// Shows the 99th percentile of all distinct APM hosts over all hours in the current date for all organizations.
	ApmHostTop99p *int64 `json:"apm_host_top99p,omitempty"`
	// Shows the average of all Application Security Monitoring ECS Fargate tasks over all hours in the current date for all organizations.
	AppsecFargateCountAvg *int64 `json:"appsec_fargate_count_avg,omitempty"`
	// Shows the sum of audit logs lines indexed over all hours in the current date for all organizations.
	AuditLogsLinesIndexedSum *int64 `json:"audit_logs_lines_indexed_sum,omitempty"`
	// The average profiled task count for Fargate Profiling.
	AvgProfiledFargateTasks *int64 `json:"avg_profiled_fargate_tasks,omitempty"`
	// Shows the 99th percentile of all AWS hosts over all hours in the current date for all organizations.
	AwsHostTop99p *int64 `json:"aws_host_top99p,omitempty"`
	// Shows the average of the number of functions that executed 1 or more times each hour in the current date for all organizations.
	AwsLambdaFuncCount *int64 `json:"aws_lambda_func_count,omitempty"`
	// Shows the sum of all AWS Lambda invocations over all hours in the current date for all organizations.
	AwsLambdaInvocationsSum *int64 `json:"aws_lambda_invocations_sum,omitempty"`
	// Shows the 99th percentile of all Azure app services over all hours in the current date for all organizations.
	AzureAppServiceTop99p *int64 `json:"azure_app_service_top99p,omitempty"`
	// Shows the sum of all log bytes ingested over all hours in the current date for all organizations.
	BillableIngestedBytesSum *int64 `json:"billable_ingested_bytes_sum,omitempty"`
	// Shows the sum of all browser lite sessions over all hours in the current date for all organizations.
	BrowserRumLiteSessionCountSum *int64 `json:"browser_rum_lite_session_count_sum,omitempty"`
	// Shows the sum of all browser replay sessions over all hours in the current date for all organizations.
	BrowserRumReplaySessionCountSum *int64 `json:"browser_rum_replay_session_count_sum,omitempty"`
	// Shows the sum of all browser RUM units over all hours in the current date for all organizations.
	BrowserRumUnitsSum *int64 `json:"browser_rum_units_sum,omitempty"`
	// Shows the sum of all CI pipeline indexed spans over all hours in the current month for all organizations.
	CiPipelineIndexedSpansSum *int64 `json:"ci_pipeline_indexed_spans_sum,omitempty"`
	// Shows the sum of all CI test indexed spans over all hours in the current month for all organizations.
	CiTestIndexedSpansSum *int64 `json:"ci_test_indexed_spans_sum,omitempty"`
	// Shows the high-water mark of all CI visibility pipeline committers over all hours in the current month for all organizations.
	CiVisibilityPipelineCommittersHwm *int64 `json:"ci_visibility_pipeline_committers_hwm,omitempty"`
	// Shows the high-water mark of all CI visibility test committers over all hours in the current month for all organizations.
	CiVisibilityTestCommittersHwm *int64 `json:"ci_visibility_test_committers_hwm,omitempty"`
	// Host count average of Cloud Cost Management for the given date and given organization.
	CloudCostManagementHostCountAvg *int64 `json:"cloud_cost_management_host_count_avg,omitempty"`
	// Shows the average of all distinct containers over all hours in the current date for all organizations.
	ContainerAvg *int64 `json:"container_avg,omitempty"`
	// Shows the average of containers without the Datadog Agent over all hours in the current date for all organizations.
	ContainerExclAgentAvg *int64 `json:"container_excl_agent_avg,omitempty"`
	// Shows the high-water mark of all distinct containers over all hours in the current date for all organizations.
	ContainerHwm *int64 `json:"container_hwm,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management Azure app services hosts over all hours in the current date for all organizations.
	CspmAasHostTop99p *int64 `json:"cspm_aas_host_top99p,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management AWS hosts over all hours in the current date for all organizations.
	CspmAwsHostTop99p *int64 `json:"cspm_aws_host_top99p,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management Azure hosts over all hours in the current date for all organizations.
	CspmAzureHostTop99p *int64 `json:"cspm_azure_host_top99p,omitempty"`
	// Shows the average number of Cloud Security Posture Management containers over all hours in the current date for all organizations.
	CspmContainerAvg *int64 `json:"cspm_container_avg,omitempty"`
	// Shows the high-water mark of Cloud Security Posture Management containers over all hours in the current date for all organizations.
	CspmContainerHwm *int64 `json:"cspm_container_hwm,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management GCP hosts over all hours in the current date for all organizations.
	CspmGcpHostTop99p *int64 `json:"cspm_gcp_host_top99p,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management hosts over all hours in the current date for all organizations.
	CspmHostTop99p *int64 `json:"cspm_host_top99p,omitempty"`
	// Shows the average number of distinct custom metrics over all hours in the current date for all organizations.
	CustomTsAvg *int64 `json:"custom_ts_avg,omitempty"`
	// Shows the average of all distinct Cloud Workload Security containers over all hours in the current date for all organizations.
	CwsContainerCountAvg *int64 `json:"cws_container_count_avg,omitempty"`
	// Shows the 99th percentile of all Cloud Workload Security hosts over all hours in the current date for all organizations.
	CwsHostTop99p *int64 `json:"cws_host_top99p,omitempty"`
	// The date for the usage.
	Date *time.Time `json:"date,omitempty"`
	// Shows the 99th percentile of all Database Monitoring hosts over all hours in the current date for all organizations.
	DbmHostTop99p *int64 `json:"dbm_host_top99p,omitempty"`
	// Shows the average of all normalized Database Monitoring queries over all hours in the current date for all organizations.
	DbmQueriesCountAvg *int64 `json:"dbm_queries_count_avg,omitempty"`
	// Shows the high-watermark of all Fargate tasks over all hours in the current date for all organizations.
	FargateTasksCountAvg *int64 `json:"fargate_tasks_count_avg,omitempty"`
	// Shows the average of all Fargate tasks over all hours in the current date for all organizations.
	FargateTasksCountHwm *int64 `json:"fargate_tasks_count_hwm,omitempty"`
	// Shows the 99th percentile of all GCP hosts over all hours in the current date for all organizations.
	GcpHostTop99p *int64 `json:"gcp_host_top99p,omitempty"`
	// Shows the 99th percentile of all Heroku dynos over all hours in the current date for all organizations.
	HerokuHostTop99p *int64 `json:"heroku_host_top99p,omitempty"`
	// Shows the high-water mark of incident management monthly active users over all hours in the current date for all organizations.
	IncidentManagementMonthlyActiveUsersHwm *int64 `json:"incident_management_monthly_active_users_hwm,omitempty"`
	// Shows the sum of all log events indexed over all hours in the current date for all organizations.
	IndexedEventsCountSum *int64 `json:"indexed_events_count_sum,omitempty"`
	// Shows the 99th percentile of all distinct infrastructure hosts over all hours in the current date for all organizations.
	InfraHostTop99p *int64 `json:"infra_host_top99p,omitempty"`
	// Shows the sum of all log bytes ingested over all hours in the current date for all organizations.
	IngestedEventsBytesSum *int64 `json:"ingested_events_bytes_sum,omitempty"`
	// Shows the sum of all IoT devices over all hours in the current date for all organizations.
	IotDeviceSum *int64 `json:"iot_device_sum,omitempty"`
	// Shows the 99th percentile of all IoT devices over all hours in the current date all organizations.
	IotDeviceTop99p *int64 `json:"iot_device_top99p,omitempty"`
	// Shows the sum of all mobile lite sessions over all hours in the current date for all organizations.
	MobileRumLiteSessionCountSum *int64 `json:"mobile_rum_lite_session_count_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions on Android over all hours in the current date for all organizations.
	MobileRumSessionCountAndroidSum *int64 `json:"mobile_rum_session_count_android_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions on iOS over all hours in the current date for all organizations.
	MobileRumSessionCountIosSum *int64 `json:"mobile_rum_session_count_ios_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions on React Native over all hours in the current date for all organizations.
	MobileRumSessionCountReactnativeSum *int64 `json:"mobile_rum_session_count_reactnative_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions over all hours in the current date for all organizations
	MobileRumSessionCountSum *int64 `json:"mobile_rum_session_count_sum,omitempty"`
	// Shows the sum of all mobile RUM units over all hours in the current date for all organizations.
	MobileRumUnitsSum *int64 `json:"mobile_rum_units_sum,omitempty"`
	// Shows the sum of all Network flows indexed over all hours in the current date for all organizations.
	NetflowIndexedEventsCountSum *int64 `json:"netflow_indexed_events_count_sum,omitempty"`
	// Shows the 99th percentile of all distinct Networks hosts over all hours in the current date for all organizations.
	NpmHostTop99p *int64 `json:"npm_host_top99p,omitempty"`
	// Sum of all observability pipelines bytes processed over all hours in the current date for the given org.
	ObservabilityPipelinesBytesProcessedSum *int64 `json:"observability_pipelines_bytes_processed_sum,omitempty"`
	// Sum of all online archived events over all hours in the current date for all organizations.
	OnlineArchiveEventsCountSum *int64 `json:"online_archive_events_count_sum,omitempty"`
	// Shows the 99th percentile of APM hosts reported by the Datadog exporter for the OpenTelemetry Collector over all hours in the current date for all organizations.
	OpentelemetryApmHostTop99p *int64 `json:"opentelemetry_apm_host_top99p,omitempty"`
	// Shows the 99th percentile of all hosts reported by the Datadog exporter for the OpenTelemetry Collector over all hours in the current date for all organizations.
	OpentelemetryHostTop99p *int64 `json:"opentelemetry_host_top99p,omitempty"`
	// Organizations associated with a user.
	Orgs []UsageSummaryDateOrg `json:"orgs,omitempty"`
	// Shows the 99th percentile of all profiled hosts over all hours in the current date for all organizations.
	ProfilingHostTop99p *int64 `json:"profiling_host_top99p,omitempty"`
	// Shows the sum of all mobile sessions and all browser lite and legacy sessions over all hours in the current month for all organizations.
	RumBrowserAndMobileSessionCount *int64 `json:"rum_browser_and_mobile_session_count,omitempty"`
	// Shows the sum of all browser RUM Lite Sessions over all hours in the current date for all organizations
	RumSessionCountSum *int64 `json:"rum_session_count_sum,omitempty"`
	// Shows the sum of RUM Sessions (browser and mobile) over all hours in the current date for all organizations.
	RumTotalSessionCountSum *int64 `json:"rum_total_session_count_sum,omitempty"`
	// Shows the sum of all browser and mobile RUM units over all hours in the current date for all organizations.
	RumUnitsSum *int64 `json:"rum_units_sum,omitempty"`
	// Sum of all APM bytes scanned with sensitive data scanner over all hours in the current date for all organizations.
	SdsApmScannedBytesSum *int64 `json:"sds_apm_scanned_bytes_sum,omitempty"`
	// Sum of all event stream events bytes scanned with sensitive data scanner over all hours in the current date for all organizations.
	SdsEventsScannedBytesSum *int64 `json:"sds_events_scanned_bytes_sum,omitempty"`
	// Shows the sum of all bytes scanned of logs usage by the Sensitive Data Scanner over all hours in the current month for all organizations.
	SdsLogsScannedBytesSum *int64 `json:"sds_logs_scanned_bytes_sum,omitempty"`
	// Sum of all RUM bytes scanned with sensitive data scanner over all hours in the current date for all organizations.
	SdsRumScannedBytesSum *int64 `json:"sds_rum_scanned_bytes_sum,omitempty"`
	// Shows the sum of all bytes scanned across all usage types by the Sensitive Data Scanner over all hours in the current month for all organizations.
	SdsTotalScannedBytesSum *int64 `json:"sds_total_scanned_bytes_sum,omitempty"`
	// Shows the sum of all Synthetic browser tests over all hours in the current date for all organizations.
	SyntheticsBrowserCheckCallsCountSum *int64 `json:"synthetics_browser_check_calls_count_sum,omitempty"`
	// Shows the sum of all Synthetic API tests over all hours in the current date for all organizations.
	SyntheticsCheckCallsCountSum *int64 `json:"synthetics_check_calls_count_sum,omitempty"`
	// Shows the high-water mark of used synthetics parallel testing slots over all hours in the current date for all organizations.
	SyntheticsParallelTestingMaxSlotsHwm *int64 `json:"synthetics_parallel_testing_max_slots_hwm,omitempty"`
	// Shows the sum of all Indexed Spans indexed over all hours in the current date for all organizations.
	TraceSearchIndexedEventsCountSum *int64 `json:"trace_search_indexed_events_count_sum,omitempty"`
	// Shows the sum of all ingested APM span bytes over all hours in the current date for all organizations.
	TwolIngestedEventsBytesSum *int64 `json:"twol_ingested_events_bytes_sum,omitempty"`
	// Shows the 99th percentile of all vSphere hosts over all hours in the current date for all organizations.
	VsphereHostTop99p *int64 `json:"vsphere_host_top99p,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUsageSummaryDate instantiates a new UsageSummaryDate object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageSummaryDate() *UsageSummaryDate {
	this := UsageSummaryDate{}
	return &this
}

// NewUsageSummaryDateWithDefaults instantiates a new UsageSummaryDate object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageSummaryDateWithDefaults() *UsageSummaryDate {
	this := UsageSummaryDate{}
	return &this
}

// GetAgentHostTop99p returns the AgentHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetAgentHostTop99p() int64 {
	if o == nil || o.AgentHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.AgentHostTop99p
}

// GetAgentHostTop99pOk returns a tuple with the AgentHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetAgentHostTop99pOk() (*int64, bool) {
	if o == nil || o.AgentHostTop99p == nil {
		return nil, false
	}
	return o.AgentHostTop99p, true
}

// HasAgentHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasAgentHostTop99p() bool {
	return o != nil && o.AgentHostTop99p != nil
}

// SetAgentHostTop99p gets a reference to the given int64 and assigns it to the AgentHostTop99p field.
func (o *UsageSummaryDate) SetAgentHostTop99p(v int64) {
	o.AgentHostTop99p = &v
}

// GetApmAzureAppServiceHostTop99p returns the ApmAzureAppServiceHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetApmAzureAppServiceHostTop99p() int64 {
	if o == nil || o.ApmAzureAppServiceHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.ApmAzureAppServiceHostTop99p
}

// GetApmAzureAppServiceHostTop99pOk returns a tuple with the ApmAzureAppServiceHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetApmAzureAppServiceHostTop99pOk() (*int64, bool) {
	if o == nil || o.ApmAzureAppServiceHostTop99p == nil {
		return nil, false
	}
	return o.ApmAzureAppServiceHostTop99p, true
}

// HasApmAzureAppServiceHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasApmAzureAppServiceHostTop99p() bool {
	return o != nil && o.ApmAzureAppServiceHostTop99p != nil
}

// SetApmAzureAppServiceHostTop99p gets a reference to the given int64 and assigns it to the ApmAzureAppServiceHostTop99p field.
func (o *UsageSummaryDate) SetApmAzureAppServiceHostTop99p(v int64) {
	o.ApmAzureAppServiceHostTop99p = &v
}

// GetApmFargateCountAvg returns the ApmFargateCountAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetApmFargateCountAvg() int64 {
	if o == nil || o.ApmFargateCountAvg == nil {
		var ret int64
		return ret
	}
	return *o.ApmFargateCountAvg
}

// GetApmFargateCountAvgOk returns a tuple with the ApmFargateCountAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetApmFargateCountAvgOk() (*int64, bool) {
	if o == nil || o.ApmFargateCountAvg == nil {
		return nil, false
	}
	return o.ApmFargateCountAvg, true
}

// HasApmFargateCountAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasApmFargateCountAvg() bool {
	return o != nil && o.ApmFargateCountAvg != nil
}

// SetApmFargateCountAvg gets a reference to the given int64 and assigns it to the ApmFargateCountAvg field.
func (o *UsageSummaryDate) SetApmFargateCountAvg(v int64) {
	o.ApmFargateCountAvg = &v
}

// GetApmHostTop99p returns the ApmHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetApmHostTop99p() int64 {
	if o == nil || o.ApmHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.ApmHostTop99p
}

// GetApmHostTop99pOk returns a tuple with the ApmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetApmHostTop99pOk() (*int64, bool) {
	if o == nil || o.ApmHostTop99p == nil {
		return nil, false
	}
	return o.ApmHostTop99p, true
}

// HasApmHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasApmHostTop99p() bool {
	return o != nil && o.ApmHostTop99p != nil
}

// SetApmHostTop99p gets a reference to the given int64 and assigns it to the ApmHostTop99p field.
func (o *UsageSummaryDate) SetApmHostTop99p(v int64) {
	o.ApmHostTop99p = &v
}

// GetAppsecFargateCountAvg returns the AppsecFargateCountAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetAppsecFargateCountAvg() int64 {
	if o == nil || o.AppsecFargateCountAvg == nil {
		var ret int64
		return ret
	}
	return *o.AppsecFargateCountAvg
}

// GetAppsecFargateCountAvgOk returns a tuple with the AppsecFargateCountAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetAppsecFargateCountAvgOk() (*int64, bool) {
	if o == nil || o.AppsecFargateCountAvg == nil {
		return nil, false
	}
	return o.AppsecFargateCountAvg, true
}

// HasAppsecFargateCountAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasAppsecFargateCountAvg() bool {
	return o != nil && o.AppsecFargateCountAvg != nil
}

// SetAppsecFargateCountAvg gets a reference to the given int64 and assigns it to the AppsecFargateCountAvg field.
func (o *UsageSummaryDate) SetAppsecFargateCountAvg(v int64) {
	o.AppsecFargateCountAvg = &v
}

// GetAuditLogsLinesIndexedSum returns the AuditLogsLinesIndexedSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetAuditLogsLinesIndexedSum() int64 {
	if o == nil || o.AuditLogsLinesIndexedSum == nil {
		var ret int64
		return ret
	}
	return *o.AuditLogsLinesIndexedSum
}

// GetAuditLogsLinesIndexedSumOk returns a tuple with the AuditLogsLinesIndexedSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetAuditLogsLinesIndexedSumOk() (*int64, bool) {
	if o == nil || o.AuditLogsLinesIndexedSum == nil {
		return nil, false
	}
	return o.AuditLogsLinesIndexedSum, true
}

// HasAuditLogsLinesIndexedSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasAuditLogsLinesIndexedSum() bool {
	return o != nil && o.AuditLogsLinesIndexedSum != nil
}

// SetAuditLogsLinesIndexedSum gets a reference to the given int64 and assigns it to the AuditLogsLinesIndexedSum field.
func (o *UsageSummaryDate) SetAuditLogsLinesIndexedSum(v int64) {
	o.AuditLogsLinesIndexedSum = &v
}

// GetAvgProfiledFargateTasks returns the AvgProfiledFargateTasks field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetAvgProfiledFargateTasks() int64 {
	if o == nil || o.AvgProfiledFargateTasks == nil {
		var ret int64
		return ret
	}
	return *o.AvgProfiledFargateTasks
}

// GetAvgProfiledFargateTasksOk returns a tuple with the AvgProfiledFargateTasks field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetAvgProfiledFargateTasksOk() (*int64, bool) {
	if o == nil || o.AvgProfiledFargateTasks == nil {
		return nil, false
	}
	return o.AvgProfiledFargateTasks, true
}

// HasAvgProfiledFargateTasks returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasAvgProfiledFargateTasks() bool {
	return o != nil && o.AvgProfiledFargateTasks != nil
}

// SetAvgProfiledFargateTasks gets a reference to the given int64 and assigns it to the AvgProfiledFargateTasks field.
func (o *UsageSummaryDate) SetAvgProfiledFargateTasks(v int64) {
	o.AvgProfiledFargateTasks = &v
}

// GetAwsHostTop99p returns the AwsHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetAwsHostTop99p() int64 {
	if o == nil || o.AwsHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.AwsHostTop99p
}

// GetAwsHostTop99pOk returns a tuple with the AwsHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetAwsHostTop99pOk() (*int64, bool) {
	if o == nil || o.AwsHostTop99p == nil {
		return nil, false
	}
	return o.AwsHostTop99p, true
}

// HasAwsHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasAwsHostTop99p() bool {
	return o != nil && o.AwsHostTop99p != nil
}

// SetAwsHostTop99p gets a reference to the given int64 and assigns it to the AwsHostTop99p field.
func (o *UsageSummaryDate) SetAwsHostTop99p(v int64) {
	o.AwsHostTop99p = &v
}

// GetAwsLambdaFuncCount returns the AwsLambdaFuncCount field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetAwsLambdaFuncCount() int64 {
	if o == nil || o.AwsLambdaFuncCount == nil {
		var ret int64
		return ret
	}
	return *o.AwsLambdaFuncCount
}

// GetAwsLambdaFuncCountOk returns a tuple with the AwsLambdaFuncCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetAwsLambdaFuncCountOk() (*int64, bool) {
	if o == nil || o.AwsLambdaFuncCount == nil {
		return nil, false
	}
	return o.AwsLambdaFuncCount, true
}

// HasAwsLambdaFuncCount returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasAwsLambdaFuncCount() bool {
	return o != nil && o.AwsLambdaFuncCount != nil
}

// SetAwsLambdaFuncCount gets a reference to the given int64 and assigns it to the AwsLambdaFuncCount field.
func (o *UsageSummaryDate) SetAwsLambdaFuncCount(v int64) {
	o.AwsLambdaFuncCount = &v
}

// GetAwsLambdaInvocationsSum returns the AwsLambdaInvocationsSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetAwsLambdaInvocationsSum() int64 {
	if o == nil || o.AwsLambdaInvocationsSum == nil {
		var ret int64
		return ret
	}
	return *o.AwsLambdaInvocationsSum
}

// GetAwsLambdaInvocationsSumOk returns a tuple with the AwsLambdaInvocationsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetAwsLambdaInvocationsSumOk() (*int64, bool) {
	if o == nil || o.AwsLambdaInvocationsSum == nil {
		return nil, false
	}
	return o.AwsLambdaInvocationsSum, true
}

// HasAwsLambdaInvocationsSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasAwsLambdaInvocationsSum() bool {
	return o != nil && o.AwsLambdaInvocationsSum != nil
}

// SetAwsLambdaInvocationsSum gets a reference to the given int64 and assigns it to the AwsLambdaInvocationsSum field.
func (o *UsageSummaryDate) SetAwsLambdaInvocationsSum(v int64) {
	o.AwsLambdaInvocationsSum = &v
}

// GetAzureAppServiceTop99p returns the AzureAppServiceTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetAzureAppServiceTop99p() int64 {
	if o == nil || o.AzureAppServiceTop99p == nil {
		var ret int64
		return ret
	}
	return *o.AzureAppServiceTop99p
}

// GetAzureAppServiceTop99pOk returns a tuple with the AzureAppServiceTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetAzureAppServiceTop99pOk() (*int64, bool) {
	if o == nil || o.AzureAppServiceTop99p == nil {
		return nil, false
	}
	return o.AzureAppServiceTop99p, true
}

// HasAzureAppServiceTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasAzureAppServiceTop99p() bool {
	return o != nil && o.AzureAppServiceTop99p != nil
}

// SetAzureAppServiceTop99p gets a reference to the given int64 and assigns it to the AzureAppServiceTop99p field.
func (o *UsageSummaryDate) SetAzureAppServiceTop99p(v int64) {
	o.AzureAppServiceTop99p = &v
}

// GetBillableIngestedBytesSum returns the BillableIngestedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetBillableIngestedBytesSum() int64 {
	if o == nil || o.BillableIngestedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.BillableIngestedBytesSum
}

// GetBillableIngestedBytesSumOk returns a tuple with the BillableIngestedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetBillableIngestedBytesSumOk() (*int64, bool) {
	if o == nil || o.BillableIngestedBytesSum == nil {
		return nil, false
	}
	return o.BillableIngestedBytesSum, true
}

// HasBillableIngestedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasBillableIngestedBytesSum() bool {
	return o != nil && o.BillableIngestedBytesSum != nil
}

// SetBillableIngestedBytesSum gets a reference to the given int64 and assigns it to the BillableIngestedBytesSum field.
func (o *UsageSummaryDate) SetBillableIngestedBytesSum(v int64) {
	o.BillableIngestedBytesSum = &v
}

// GetBrowserRumLiteSessionCountSum returns the BrowserRumLiteSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetBrowserRumLiteSessionCountSum() int64 {
	if o == nil || o.BrowserRumLiteSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.BrowserRumLiteSessionCountSum
}

// GetBrowserRumLiteSessionCountSumOk returns a tuple with the BrowserRumLiteSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetBrowserRumLiteSessionCountSumOk() (*int64, bool) {
	if o == nil || o.BrowserRumLiteSessionCountSum == nil {
		return nil, false
	}
	return o.BrowserRumLiteSessionCountSum, true
}

// HasBrowserRumLiteSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasBrowserRumLiteSessionCountSum() bool {
	return o != nil && o.BrowserRumLiteSessionCountSum != nil
}

// SetBrowserRumLiteSessionCountSum gets a reference to the given int64 and assigns it to the BrowserRumLiteSessionCountSum field.
func (o *UsageSummaryDate) SetBrowserRumLiteSessionCountSum(v int64) {
	o.BrowserRumLiteSessionCountSum = &v
}

// GetBrowserRumReplaySessionCountSum returns the BrowserRumReplaySessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetBrowserRumReplaySessionCountSum() int64 {
	if o == nil || o.BrowserRumReplaySessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.BrowserRumReplaySessionCountSum
}

// GetBrowserRumReplaySessionCountSumOk returns a tuple with the BrowserRumReplaySessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetBrowserRumReplaySessionCountSumOk() (*int64, bool) {
	if o == nil || o.BrowserRumReplaySessionCountSum == nil {
		return nil, false
	}
	return o.BrowserRumReplaySessionCountSum, true
}

// HasBrowserRumReplaySessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasBrowserRumReplaySessionCountSum() bool {
	return o != nil && o.BrowserRumReplaySessionCountSum != nil
}

// SetBrowserRumReplaySessionCountSum gets a reference to the given int64 and assigns it to the BrowserRumReplaySessionCountSum field.
func (o *UsageSummaryDate) SetBrowserRumReplaySessionCountSum(v int64) {
	o.BrowserRumReplaySessionCountSum = &v
}

// GetBrowserRumUnitsSum returns the BrowserRumUnitsSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetBrowserRumUnitsSum() int64 {
	if o == nil || o.BrowserRumUnitsSum == nil {
		var ret int64
		return ret
	}
	return *o.BrowserRumUnitsSum
}

// GetBrowserRumUnitsSumOk returns a tuple with the BrowserRumUnitsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetBrowserRumUnitsSumOk() (*int64, bool) {
	if o == nil || o.BrowserRumUnitsSum == nil {
		return nil, false
	}
	return o.BrowserRumUnitsSum, true
}

// HasBrowserRumUnitsSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasBrowserRumUnitsSum() bool {
	return o != nil && o.BrowserRumUnitsSum != nil
}

// SetBrowserRumUnitsSum gets a reference to the given int64 and assigns it to the BrowserRumUnitsSum field.
func (o *UsageSummaryDate) SetBrowserRumUnitsSum(v int64) {
	o.BrowserRumUnitsSum = &v
}

// GetCiPipelineIndexedSpansSum returns the CiPipelineIndexedSpansSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCiPipelineIndexedSpansSum() int64 {
	if o == nil || o.CiPipelineIndexedSpansSum == nil {
		var ret int64
		return ret
	}
	return *o.CiPipelineIndexedSpansSum
}

// GetCiPipelineIndexedSpansSumOk returns a tuple with the CiPipelineIndexedSpansSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCiPipelineIndexedSpansSumOk() (*int64, bool) {
	if o == nil || o.CiPipelineIndexedSpansSum == nil {
		return nil, false
	}
	return o.CiPipelineIndexedSpansSum, true
}

// HasCiPipelineIndexedSpansSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCiPipelineIndexedSpansSum() bool {
	return o != nil && o.CiPipelineIndexedSpansSum != nil
}

// SetCiPipelineIndexedSpansSum gets a reference to the given int64 and assigns it to the CiPipelineIndexedSpansSum field.
func (o *UsageSummaryDate) SetCiPipelineIndexedSpansSum(v int64) {
	o.CiPipelineIndexedSpansSum = &v
}

// GetCiTestIndexedSpansSum returns the CiTestIndexedSpansSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCiTestIndexedSpansSum() int64 {
	if o == nil || o.CiTestIndexedSpansSum == nil {
		var ret int64
		return ret
	}
	return *o.CiTestIndexedSpansSum
}

// GetCiTestIndexedSpansSumOk returns a tuple with the CiTestIndexedSpansSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCiTestIndexedSpansSumOk() (*int64, bool) {
	if o == nil || o.CiTestIndexedSpansSum == nil {
		return nil, false
	}
	return o.CiTestIndexedSpansSum, true
}

// HasCiTestIndexedSpansSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCiTestIndexedSpansSum() bool {
	return o != nil && o.CiTestIndexedSpansSum != nil
}

// SetCiTestIndexedSpansSum gets a reference to the given int64 and assigns it to the CiTestIndexedSpansSum field.
func (o *UsageSummaryDate) SetCiTestIndexedSpansSum(v int64) {
	o.CiTestIndexedSpansSum = &v
}

// GetCiVisibilityPipelineCommittersHwm returns the CiVisibilityPipelineCommittersHwm field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCiVisibilityPipelineCommittersHwm() int64 {
	if o == nil || o.CiVisibilityPipelineCommittersHwm == nil {
		var ret int64
		return ret
	}
	return *o.CiVisibilityPipelineCommittersHwm
}

// GetCiVisibilityPipelineCommittersHwmOk returns a tuple with the CiVisibilityPipelineCommittersHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCiVisibilityPipelineCommittersHwmOk() (*int64, bool) {
	if o == nil || o.CiVisibilityPipelineCommittersHwm == nil {
		return nil, false
	}
	return o.CiVisibilityPipelineCommittersHwm, true
}

// HasCiVisibilityPipelineCommittersHwm returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCiVisibilityPipelineCommittersHwm() bool {
	return o != nil && o.CiVisibilityPipelineCommittersHwm != nil
}

// SetCiVisibilityPipelineCommittersHwm gets a reference to the given int64 and assigns it to the CiVisibilityPipelineCommittersHwm field.
func (o *UsageSummaryDate) SetCiVisibilityPipelineCommittersHwm(v int64) {
	o.CiVisibilityPipelineCommittersHwm = &v
}

// GetCiVisibilityTestCommittersHwm returns the CiVisibilityTestCommittersHwm field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCiVisibilityTestCommittersHwm() int64 {
	if o == nil || o.CiVisibilityTestCommittersHwm == nil {
		var ret int64
		return ret
	}
	return *o.CiVisibilityTestCommittersHwm
}

// GetCiVisibilityTestCommittersHwmOk returns a tuple with the CiVisibilityTestCommittersHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCiVisibilityTestCommittersHwmOk() (*int64, bool) {
	if o == nil || o.CiVisibilityTestCommittersHwm == nil {
		return nil, false
	}
	return o.CiVisibilityTestCommittersHwm, true
}

// HasCiVisibilityTestCommittersHwm returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCiVisibilityTestCommittersHwm() bool {
	return o != nil && o.CiVisibilityTestCommittersHwm != nil
}

// SetCiVisibilityTestCommittersHwm gets a reference to the given int64 and assigns it to the CiVisibilityTestCommittersHwm field.
func (o *UsageSummaryDate) SetCiVisibilityTestCommittersHwm(v int64) {
	o.CiVisibilityTestCommittersHwm = &v
}

// GetCloudCostManagementHostCountAvg returns the CloudCostManagementHostCountAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCloudCostManagementHostCountAvg() int64 {
	if o == nil || o.CloudCostManagementHostCountAvg == nil {
		var ret int64
		return ret
	}
	return *o.CloudCostManagementHostCountAvg
}

// GetCloudCostManagementHostCountAvgOk returns a tuple with the CloudCostManagementHostCountAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCloudCostManagementHostCountAvgOk() (*int64, bool) {
	if o == nil || o.CloudCostManagementHostCountAvg == nil {
		return nil, false
	}
	return o.CloudCostManagementHostCountAvg, true
}

// HasCloudCostManagementHostCountAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCloudCostManagementHostCountAvg() bool {
	return o != nil && o.CloudCostManagementHostCountAvg != nil
}

// SetCloudCostManagementHostCountAvg gets a reference to the given int64 and assigns it to the CloudCostManagementHostCountAvg field.
func (o *UsageSummaryDate) SetCloudCostManagementHostCountAvg(v int64) {
	o.CloudCostManagementHostCountAvg = &v
}

// GetContainerAvg returns the ContainerAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetContainerAvg() int64 {
	if o == nil || o.ContainerAvg == nil {
		var ret int64
		return ret
	}
	return *o.ContainerAvg
}

// GetContainerAvgOk returns a tuple with the ContainerAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetContainerAvgOk() (*int64, bool) {
	if o == nil || o.ContainerAvg == nil {
		return nil, false
	}
	return o.ContainerAvg, true
}

// HasContainerAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasContainerAvg() bool {
	return o != nil && o.ContainerAvg != nil
}

// SetContainerAvg gets a reference to the given int64 and assigns it to the ContainerAvg field.
func (o *UsageSummaryDate) SetContainerAvg(v int64) {
	o.ContainerAvg = &v
}

// GetContainerExclAgentAvg returns the ContainerExclAgentAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetContainerExclAgentAvg() int64 {
	if o == nil || o.ContainerExclAgentAvg == nil {
		var ret int64
		return ret
	}
	return *o.ContainerExclAgentAvg
}

// GetContainerExclAgentAvgOk returns a tuple with the ContainerExclAgentAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetContainerExclAgentAvgOk() (*int64, bool) {
	if o == nil || o.ContainerExclAgentAvg == nil {
		return nil, false
	}
	return o.ContainerExclAgentAvg, true
}

// HasContainerExclAgentAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasContainerExclAgentAvg() bool {
	return o != nil && o.ContainerExclAgentAvg != nil
}

// SetContainerExclAgentAvg gets a reference to the given int64 and assigns it to the ContainerExclAgentAvg field.
func (o *UsageSummaryDate) SetContainerExclAgentAvg(v int64) {
	o.ContainerExclAgentAvg = &v
}

// GetContainerHwm returns the ContainerHwm field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetContainerHwm() int64 {
	if o == nil || o.ContainerHwm == nil {
		var ret int64
		return ret
	}
	return *o.ContainerHwm
}

// GetContainerHwmOk returns a tuple with the ContainerHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetContainerHwmOk() (*int64, bool) {
	if o == nil || o.ContainerHwm == nil {
		return nil, false
	}
	return o.ContainerHwm, true
}

// HasContainerHwm returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasContainerHwm() bool {
	return o != nil && o.ContainerHwm != nil
}

// SetContainerHwm gets a reference to the given int64 and assigns it to the ContainerHwm field.
func (o *UsageSummaryDate) SetContainerHwm(v int64) {
	o.ContainerHwm = &v
}

// GetCspmAasHostTop99p returns the CspmAasHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCspmAasHostTop99p() int64 {
	if o == nil || o.CspmAasHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CspmAasHostTop99p
}

// GetCspmAasHostTop99pOk returns a tuple with the CspmAasHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCspmAasHostTop99pOk() (*int64, bool) {
	if o == nil || o.CspmAasHostTop99p == nil {
		return nil, false
	}
	return o.CspmAasHostTop99p, true
}

// HasCspmAasHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCspmAasHostTop99p() bool {
	return o != nil && o.CspmAasHostTop99p != nil
}

// SetCspmAasHostTop99p gets a reference to the given int64 and assigns it to the CspmAasHostTop99p field.
func (o *UsageSummaryDate) SetCspmAasHostTop99p(v int64) {
	o.CspmAasHostTop99p = &v
}

// GetCspmAwsHostTop99p returns the CspmAwsHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCspmAwsHostTop99p() int64 {
	if o == nil || o.CspmAwsHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CspmAwsHostTop99p
}

// GetCspmAwsHostTop99pOk returns a tuple with the CspmAwsHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCspmAwsHostTop99pOk() (*int64, bool) {
	if o == nil || o.CspmAwsHostTop99p == nil {
		return nil, false
	}
	return o.CspmAwsHostTop99p, true
}

// HasCspmAwsHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCspmAwsHostTop99p() bool {
	return o != nil && o.CspmAwsHostTop99p != nil
}

// SetCspmAwsHostTop99p gets a reference to the given int64 and assigns it to the CspmAwsHostTop99p field.
func (o *UsageSummaryDate) SetCspmAwsHostTop99p(v int64) {
	o.CspmAwsHostTop99p = &v
}

// GetCspmAzureHostTop99p returns the CspmAzureHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCspmAzureHostTop99p() int64 {
	if o == nil || o.CspmAzureHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CspmAzureHostTop99p
}

// GetCspmAzureHostTop99pOk returns a tuple with the CspmAzureHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCspmAzureHostTop99pOk() (*int64, bool) {
	if o == nil || o.CspmAzureHostTop99p == nil {
		return nil, false
	}
	return o.CspmAzureHostTop99p, true
}

// HasCspmAzureHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCspmAzureHostTop99p() bool {
	return o != nil && o.CspmAzureHostTop99p != nil
}

// SetCspmAzureHostTop99p gets a reference to the given int64 and assigns it to the CspmAzureHostTop99p field.
func (o *UsageSummaryDate) SetCspmAzureHostTop99p(v int64) {
	o.CspmAzureHostTop99p = &v
}

// GetCspmContainerAvg returns the CspmContainerAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCspmContainerAvg() int64 {
	if o == nil || o.CspmContainerAvg == nil {
		var ret int64
		return ret
	}
	return *o.CspmContainerAvg
}

// GetCspmContainerAvgOk returns a tuple with the CspmContainerAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCspmContainerAvgOk() (*int64, bool) {
	if o == nil || o.CspmContainerAvg == nil {
		return nil, false
	}
	return o.CspmContainerAvg, true
}

// HasCspmContainerAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCspmContainerAvg() bool {
	return o != nil && o.CspmContainerAvg != nil
}

// SetCspmContainerAvg gets a reference to the given int64 and assigns it to the CspmContainerAvg field.
func (o *UsageSummaryDate) SetCspmContainerAvg(v int64) {
	o.CspmContainerAvg = &v
}

// GetCspmContainerHwm returns the CspmContainerHwm field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCspmContainerHwm() int64 {
	if o == nil || o.CspmContainerHwm == nil {
		var ret int64
		return ret
	}
	return *o.CspmContainerHwm
}

// GetCspmContainerHwmOk returns a tuple with the CspmContainerHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCspmContainerHwmOk() (*int64, bool) {
	if o == nil || o.CspmContainerHwm == nil {
		return nil, false
	}
	return o.CspmContainerHwm, true
}

// HasCspmContainerHwm returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCspmContainerHwm() bool {
	return o != nil && o.CspmContainerHwm != nil
}

// SetCspmContainerHwm gets a reference to the given int64 and assigns it to the CspmContainerHwm field.
func (o *UsageSummaryDate) SetCspmContainerHwm(v int64) {
	o.CspmContainerHwm = &v
}

// GetCspmGcpHostTop99p returns the CspmGcpHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCspmGcpHostTop99p() int64 {
	if o == nil || o.CspmGcpHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CspmGcpHostTop99p
}

// GetCspmGcpHostTop99pOk returns a tuple with the CspmGcpHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCspmGcpHostTop99pOk() (*int64, bool) {
	if o == nil || o.CspmGcpHostTop99p == nil {
		return nil, false
	}
	return o.CspmGcpHostTop99p, true
}

// HasCspmGcpHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCspmGcpHostTop99p() bool {
	return o != nil && o.CspmGcpHostTop99p != nil
}

// SetCspmGcpHostTop99p gets a reference to the given int64 and assigns it to the CspmGcpHostTop99p field.
func (o *UsageSummaryDate) SetCspmGcpHostTop99p(v int64) {
	o.CspmGcpHostTop99p = &v
}

// GetCspmHostTop99p returns the CspmHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCspmHostTop99p() int64 {
	if o == nil || o.CspmHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CspmHostTop99p
}

// GetCspmHostTop99pOk returns a tuple with the CspmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCspmHostTop99pOk() (*int64, bool) {
	if o == nil || o.CspmHostTop99p == nil {
		return nil, false
	}
	return o.CspmHostTop99p, true
}

// HasCspmHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCspmHostTop99p() bool {
	return o != nil && o.CspmHostTop99p != nil
}

// SetCspmHostTop99p gets a reference to the given int64 and assigns it to the CspmHostTop99p field.
func (o *UsageSummaryDate) SetCspmHostTop99p(v int64) {
	o.CspmHostTop99p = &v
}

// GetCustomTsAvg returns the CustomTsAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCustomTsAvg() int64 {
	if o == nil || o.CustomTsAvg == nil {
		var ret int64
		return ret
	}
	return *o.CustomTsAvg
}

// GetCustomTsAvgOk returns a tuple with the CustomTsAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCustomTsAvgOk() (*int64, bool) {
	if o == nil || o.CustomTsAvg == nil {
		return nil, false
	}
	return o.CustomTsAvg, true
}

// HasCustomTsAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCustomTsAvg() bool {
	return o != nil && o.CustomTsAvg != nil
}

// SetCustomTsAvg gets a reference to the given int64 and assigns it to the CustomTsAvg field.
func (o *UsageSummaryDate) SetCustomTsAvg(v int64) {
	o.CustomTsAvg = &v
}

// GetCwsContainerCountAvg returns the CwsContainerCountAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCwsContainerCountAvg() int64 {
	if o == nil || o.CwsContainerCountAvg == nil {
		var ret int64
		return ret
	}
	return *o.CwsContainerCountAvg
}

// GetCwsContainerCountAvgOk returns a tuple with the CwsContainerCountAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCwsContainerCountAvgOk() (*int64, bool) {
	if o == nil || o.CwsContainerCountAvg == nil {
		return nil, false
	}
	return o.CwsContainerCountAvg, true
}

// HasCwsContainerCountAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCwsContainerCountAvg() bool {
	return o != nil && o.CwsContainerCountAvg != nil
}

// SetCwsContainerCountAvg gets a reference to the given int64 and assigns it to the CwsContainerCountAvg field.
func (o *UsageSummaryDate) SetCwsContainerCountAvg(v int64) {
	o.CwsContainerCountAvg = &v
}

// GetCwsHostTop99p returns the CwsHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetCwsHostTop99p() int64 {
	if o == nil || o.CwsHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.CwsHostTop99p
}

// GetCwsHostTop99pOk returns a tuple with the CwsHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetCwsHostTop99pOk() (*int64, bool) {
	if o == nil || o.CwsHostTop99p == nil {
		return nil, false
	}
	return o.CwsHostTop99p, true
}

// HasCwsHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasCwsHostTop99p() bool {
	return o != nil && o.CwsHostTop99p != nil
}

// SetCwsHostTop99p gets a reference to the given int64 and assigns it to the CwsHostTop99p field.
func (o *UsageSummaryDate) SetCwsHostTop99p(v int64) {
	o.CwsHostTop99p = &v
}

// GetDate returns the Date field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetDate() time.Time {
	if o == nil || o.Date == nil {
		var ret time.Time
		return ret
	}
	return *o.Date
}

// GetDateOk returns a tuple with the Date field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetDateOk() (*time.Time, bool) {
	if o == nil || o.Date == nil {
		return nil, false
	}
	return o.Date, true
}

// HasDate returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasDate() bool {
	return o != nil && o.Date != nil
}

// SetDate gets a reference to the given time.Time and assigns it to the Date field.
func (o *UsageSummaryDate) SetDate(v time.Time) {
	o.Date = &v
}

// GetDbmHostTop99p returns the DbmHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetDbmHostTop99p() int64 {
	if o == nil || o.DbmHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.DbmHostTop99p
}

// GetDbmHostTop99pOk returns a tuple with the DbmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetDbmHostTop99pOk() (*int64, bool) {
	if o == nil || o.DbmHostTop99p == nil {
		return nil, false
	}
	return o.DbmHostTop99p, true
}

// HasDbmHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasDbmHostTop99p() bool {
	return o != nil && o.DbmHostTop99p != nil
}

// SetDbmHostTop99p gets a reference to the given int64 and assigns it to the DbmHostTop99p field.
func (o *UsageSummaryDate) SetDbmHostTop99p(v int64) {
	o.DbmHostTop99p = &v
}

// GetDbmQueriesCountAvg returns the DbmQueriesCountAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetDbmQueriesCountAvg() int64 {
	if o == nil || o.DbmQueriesCountAvg == nil {
		var ret int64
		return ret
	}
	return *o.DbmQueriesCountAvg
}

// GetDbmQueriesCountAvgOk returns a tuple with the DbmQueriesCountAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetDbmQueriesCountAvgOk() (*int64, bool) {
	if o == nil || o.DbmQueriesCountAvg == nil {
		return nil, false
	}
	return o.DbmQueriesCountAvg, true
}

// HasDbmQueriesCountAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasDbmQueriesCountAvg() bool {
	return o != nil && o.DbmQueriesCountAvg != nil
}

// SetDbmQueriesCountAvg gets a reference to the given int64 and assigns it to the DbmQueriesCountAvg field.
func (o *UsageSummaryDate) SetDbmQueriesCountAvg(v int64) {
	o.DbmQueriesCountAvg = &v
}

// GetFargateTasksCountAvg returns the FargateTasksCountAvg field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetFargateTasksCountAvg() int64 {
	if o == nil || o.FargateTasksCountAvg == nil {
		var ret int64
		return ret
	}
	return *o.FargateTasksCountAvg
}

// GetFargateTasksCountAvgOk returns a tuple with the FargateTasksCountAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetFargateTasksCountAvgOk() (*int64, bool) {
	if o == nil || o.FargateTasksCountAvg == nil {
		return nil, false
	}
	return o.FargateTasksCountAvg, true
}

// HasFargateTasksCountAvg returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasFargateTasksCountAvg() bool {
	return o != nil && o.FargateTasksCountAvg != nil
}

// SetFargateTasksCountAvg gets a reference to the given int64 and assigns it to the FargateTasksCountAvg field.
func (o *UsageSummaryDate) SetFargateTasksCountAvg(v int64) {
	o.FargateTasksCountAvg = &v
}

// GetFargateTasksCountHwm returns the FargateTasksCountHwm field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetFargateTasksCountHwm() int64 {
	if o == nil || o.FargateTasksCountHwm == nil {
		var ret int64
		return ret
	}
	return *o.FargateTasksCountHwm
}

// GetFargateTasksCountHwmOk returns a tuple with the FargateTasksCountHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetFargateTasksCountHwmOk() (*int64, bool) {
	if o == nil || o.FargateTasksCountHwm == nil {
		return nil, false
	}
	return o.FargateTasksCountHwm, true
}

// HasFargateTasksCountHwm returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasFargateTasksCountHwm() bool {
	return o != nil && o.FargateTasksCountHwm != nil
}

// SetFargateTasksCountHwm gets a reference to the given int64 and assigns it to the FargateTasksCountHwm field.
func (o *UsageSummaryDate) SetFargateTasksCountHwm(v int64) {
	o.FargateTasksCountHwm = &v
}

// GetGcpHostTop99p returns the GcpHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetGcpHostTop99p() int64 {
	if o == nil || o.GcpHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.GcpHostTop99p
}

// GetGcpHostTop99pOk returns a tuple with the GcpHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetGcpHostTop99pOk() (*int64, bool) {
	if o == nil || o.GcpHostTop99p == nil {
		return nil, false
	}
	return o.GcpHostTop99p, true
}

// HasGcpHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasGcpHostTop99p() bool {
	return o != nil && o.GcpHostTop99p != nil
}

// SetGcpHostTop99p gets a reference to the given int64 and assigns it to the GcpHostTop99p field.
func (o *UsageSummaryDate) SetGcpHostTop99p(v int64) {
	o.GcpHostTop99p = &v
}

// GetHerokuHostTop99p returns the HerokuHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetHerokuHostTop99p() int64 {
	if o == nil || o.HerokuHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.HerokuHostTop99p
}

// GetHerokuHostTop99pOk returns a tuple with the HerokuHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetHerokuHostTop99pOk() (*int64, bool) {
	if o == nil || o.HerokuHostTop99p == nil {
		return nil, false
	}
	return o.HerokuHostTop99p, true
}

// HasHerokuHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasHerokuHostTop99p() bool {
	return o != nil && o.HerokuHostTop99p != nil
}

// SetHerokuHostTop99p gets a reference to the given int64 and assigns it to the HerokuHostTop99p field.
func (o *UsageSummaryDate) SetHerokuHostTop99p(v int64) {
	o.HerokuHostTop99p = &v
}

// GetIncidentManagementMonthlyActiveUsersHwm returns the IncidentManagementMonthlyActiveUsersHwm field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetIncidentManagementMonthlyActiveUsersHwm() int64 {
	if o == nil || o.IncidentManagementMonthlyActiveUsersHwm == nil {
		var ret int64
		return ret
	}
	return *o.IncidentManagementMonthlyActiveUsersHwm
}

// GetIncidentManagementMonthlyActiveUsersHwmOk returns a tuple with the IncidentManagementMonthlyActiveUsersHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetIncidentManagementMonthlyActiveUsersHwmOk() (*int64, bool) {
	if o == nil || o.IncidentManagementMonthlyActiveUsersHwm == nil {
		return nil, false
	}
	return o.IncidentManagementMonthlyActiveUsersHwm, true
}

// HasIncidentManagementMonthlyActiveUsersHwm returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasIncidentManagementMonthlyActiveUsersHwm() bool {
	return o != nil && o.IncidentManagementMonthlyActiveUsersHwm != nil
}

// SetIncidentManagementMonthlyActiveUsersHwm gets a reference to the given int64 and assigns it to the IncidentManagementMonthlyActiveUsersHwm field.
func (o *UsageSummaryDate) SetIncidentManagementMonthlyActiveUsersHwm(v int64) {
	o.IncidentManagementMonthlyActiveUsersHwm = &v
}

// GetIndexedEventsCountSum returns the IndexedEventsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetIndexedEventsCountSum() int64 {
	if o == nil || o.IndexedEventsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.IndexedEventsCountSum
}

// GetIndexedEventsCountSumOk returns a tuple with the IndexedEventsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetIndexedEventsCountSumOk() (*int64, bool) {
	if o == nil || o.IndexedEventsCountSum == nil {
		return nil, false
	}
	return o.IndexedEventsCountSum, true
}

// HasIndexedEventsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasIndexedEventsCountSum() bool {
	return o != nil && o.IndexedEventsCountSum != nil
}

// SetIndexedEventsCountSum gets a reference to the given int64 and assigns it to the IndexedEventsCountSum field.
func (o *UsageSummaryDate) SetIndexedEventsCountSum(v int64) {
	o.IndexedEventsCountSum = &v
}

// GetInfraHostTop99p returns the InfraHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetInfraHostTop99p() int64 {
	if o == nil || o.InfraHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.InfraHostTop99p
}

// GetInfraHostTop99pOk returns a tuple with the InfraHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetInfraHostTop99pOk() (*int64, bool) {
	if o == nil || o.InfraHostTop99p == nil {
		return nil, false
	}
	return o.InfraHostTop99p, true
}

// HasInfraHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasInfraHostTop99p() bool {
	return o != nil && o.InfraHostTop99p != nil
}

// SetInfraHostTop99p gets a reference to the given int64 and assigns it to the InfraHostTop99p field.
func (o *UsageSummaryDate) SetInfraHostTop99p(v int64) {
	o.InfraHostTop99p = &v
}

// GetIngestedEventsBytesSum returns the IngestedEventsBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetIngestedEventsBytesSum() int64 {
	if o == nil || o.IngestedEventsBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.IngestedEventsBytesSum
}

// GetIngestedEventsBytesSumOk returns a tuple with the IngestedEventsBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetIngestedEventsBytesSumOk() (*int64, bool) {
	if o == nil || o.IngestedEventsBytesSum == nil {
		return nil, false
	}
	return o.IngestedEventsBytesSum, true
}

// HasIngestedEventsBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasIngestedEventsBytesSum() bool {
	return o != nil && o.IngestedEventsBytesSum != nil
}

// SetIngestedEventsBytesSum gets a reference to the given int64 and assigns it to the IngestedEventsBytesSum field.
func (o *UsageSummaryDate) SetIngestedEventsBytesSum(v int64) {
	o.IngestedEventsBytesSum = &v
}

// GetIotDeviceSum returns the IotDeviceSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetIotDeviceSum() int64 {
	if o == nil || o.IotDeviceSum == nil {
		var ret int64
		return ret
	}
	return *o.IotDeviceSum
}

// GetIotDeviceSumOk returns a tuple with the IotDeviceSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetIotDeviceSumOk() (*int64, bool) {
	if o == nil || o.IotDeviceSum == nil {
		return nil, false
	}
	return o.IotDeviceSum, true
}

// HasIotDeviceSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasIotDeviceSum() bool {
	return o != nil && o.IotDeviceSum != nil
}

// SetIotDeviceSum gets a reference to the given int64 and assigns it to the IotDeviceSum field.
func (o *UsageSummaryDate) SetIotDeviceSum(v int64) {
	o.IotDeviceSum = &v
}

// GetIotDeviceTop99p returns the IotDeviceTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetIotDeviceTop99p() int64 {
	if o == nil || o.IotDeviceTop99p == nil {
		var ret int64
		return ret
	}
	return *o.IotDeviceTop99p
}

// GetIotDeviceTop99pOk returns a tuple with the IotDeviceTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetIotDeviceTop99pOk() (*int64, bool) {
	if o == nil || o.IotDeviceTop99p == nil {
		return nil, false
	}
	return o.IotDeviceTop99p, true
}

// HasIotDeviceTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasIotDeviceTop99p() bool {
	return o != nil && o.IotDeviceTop99p != nil
}

// SetIotDeviceTop99p gets a reference to the given int64 and assigns it to the IotDeviceTop99p field.
func (o *UsageSummaryDate) SetIotDeviceTop99p(v int64) {
	o.IotDeviceTop99p = &v
}

// GetMobileRumLiteSessionCountSum returns the MobileRumLiteSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetMobileRumLiteSessionCountSum() int64 {
	if o == nil || o.MobileRumLiteSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumLiteSessionCountSum
}

// GetMobileRumLiteSessionCountSumOk returns a tuple with the MobileRumLiteSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetMobileRumLiteSessionCountSumOk() (*int64, bool) {
	if o == nil || o.MobileRumLiteSessionCountSum == nil {
		return nil, false
	}
	return o.MobileRumLiteSessionCountSum, true
}

// HasMobileRumLiteSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasMobileRumLiteSessionCountSum() bool {
	return o != nil && o.MobileRumLiteSessionCountSum != nil
}

// SetMobileRumLiteSessionCountSum gets a reference to the given int64 and assigns it to the MobileRumLiteSessionCountSum field.
func (o *UsageSummaryDate) SetMobileRumLiteSessionCountSum(v int64) {
	o.MobileRumLiteSessionCountSum = &v
}

// GetMobileRumSessionCountAndroidSum returns the MobileRumSessionCountAndroidSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetMobileRumSessionCountAndroidSum() int64 {
	if o == nil || o.MobileRumSessionCountAndroidSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountAndroidSum
}

// GetMobileRumSessionCountAndroidSumOk returns a tuple with the MobileRumSessionCountAndroidSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetMobileRumSessionCountAndroidSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountAndroidSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountAndroidSum, true
}

// HasMobileRumSessionCountAndroidSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasMobileRumSessionCountAndroidSum() bool {
	return o != nil && o.MobileRumSessionCountAndroidSum != nil
}

// SetMobileRumSessionCountAndroidSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountAndroidSum field.
func (o *UsageSummaryDate) SetMobileRumSessionCountAndroidSum(v int64) {
	o.MobileRumSessionCountAndroidSum = &v
}

// GetMobileRumSessionCountIosSum returns the MobileRumSessionCountIosSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetMobileRumSessionCountIosSum() int64 {
	if o == nil || o.MobileRumSessionCountIosSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountIosSum
}

// GetMobileRumSessionCountIosSumOk returns a tuple with the MobileRumSessionCountIosSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetMobileRumSessionCountIosSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountIosSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountIosSum, true
}

// HasMobileRumSessionCountIosSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasMobileRumSessionCountIosSum() bool {
	return o != nil && o.MobileRumSessionCountIosSum != nil
}

// SetMobileRumSessionCountIosSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountIosSum field.
func (o *UsageSummaryDate) SetMobileRumSessionCountIosSum(v int64) {
	o.MobileRumSessionCountIosSum = &v
}

// GetMobileRumSessionCountReactnativeSum returns the MobileRumSessionCountReactnativeSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetMobileRumSessionCountReactnativeSum() int64 {
	if o == nil || o.MobileRumSessionCountReactnativeSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountReactnativeSum
}

// GetMobileRumSessionCountReactnativeSumOk returns a tuple with the MobileRumSessionCountReactnativeSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetMobileRumSessionCountReactnativeSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountReactnativeSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountReactnativeSum, true
}

// HasMobileRumSessionCountReactnativeSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasMobileRumSessionCountReactnativeSum() bool {
	return o != nil && o.MobileRumSessionCountReactnativeSum != nil
}

// SetMobileRumSessionCountReactnativeSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountReactnativeSum field.
func (o *UsageSummaryDate) SetMobileRumSessionCountReactnativeSum(v int64) {
	o.MobileRumSessionCountReactnativeSum = &v
}

// GetMobileRumSessionCountSum returns the MobileRumSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetMobileRumSessionCountSum() int64 {
	if o == nil || o.MobileRumSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountSum
}

// GetMobileRumSessionCountSumOk returns a tuple with the MobileRumSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetMobileRumSessionCountSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountSum, true
}

// HasMobileRumSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasMobileRumSessionCountSum() bool {
	return o != nil && o.MobileRumSessionCountSum != nil
}

// SetMobileRumSessionCountSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountSum field.
func (o *UsageSummaryDate) SetMobileRumSessionCountSum(v int64) {
	o.MobileRumSessionCountSum = &v
}

// GetMobileRumUnitsSum returns the MobileRumUnitsSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetMobileRumUnitsSum() int64 {
	if o == nil || o.MobileRumUnitsSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumUnitsSum
}

// GetMobileRumUnitsSumOk returns a tuple with the MobileRumUnitsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetMobileRumUnitsSumOk() (*int64, bool) {
	if o == nil || o.MobileRumUnitsSum == nil {
		return nil, false
	}
	return o.MobileRumUnitsSum, true
}

// HasMobileRumUnitsSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasMobileRumUnitsSum() bool {
	return o != nil && o.MobileRumUnitsSum != nil
}

// SetMobileRumUnitsSum gets a reference to the given int64 and assigns it to the MobileRumUnitsSum field.
func (o *UsageSummaryDate) SetMobileRumUnitsSum(v int64) {
	o.MobileRumUnitsSum = &v
}

// GetNetflowIndexedEventsCountSum returns the NetflowIndexedEventsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetNetflowIndexedEventsCountSum() int64 {
	if o == nil || o.NetflowIndexedEventsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.NetflowIndexedEventsCountSum
}

// GetNetflowIndexedEventsCountSumOk returns a tuple with the NetflowIndexedEventsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetNetflowIndexedEventsCountSumOk() (*int64, bool) {
	if o == nil || o.NetflowIndexedEventsCountSum == nil {
		return nil, false
	}
	return o.NetflowIndexedEventsCountSum, true
}

// HasNetflowIndexedEventsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasNetflowIndexedEventsCountSum() bool {
	return o != nil && o.NetflowIndexedEventsCountSum != nil
}

// SetNetflowIndexedEventsCountSum gets a reference to the given int64 and assigns it to the NetflowIndexedEventsCountSum field.
func (o *UsageSummaryDate) SetNetflowIndexedEventsCountSum(v int64) {
	o.NetflowIndexedEventsCountSum = &v
}

// GetNpmHostTop99p returns the NpmHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetNpmHostTop99p() int64 {
	if o == nil || o.NpmHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.NpmHostTop99p
}

// GetNpmHostTop99pOk returns a tuple with the NpmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetNpmHostTop99pOk() (*int64, bool) {
	if o == nil || o.NpmHostTop99p == nil {
		return nil, false
	}
	return o.NpmHostTop99p, true
}

// HasNpmHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasNpmHostTop99p() bool {
	return o != nil && o.NpmHostTop99p != nil
}

// SetNpmHostTop99p gets a reference to the given int64 and assigns it to the NpmHostTop99p field.
func (o *UsageSummaryDate) SetNpmHostTop99p(v int64) {
	o.NpmHostTop99p = &v
}

// GetObservabilityPipelinesBytesProcessedSum returns the ObservabilityPipelinesBytesProcessedSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetObservabilityPipelinesBytesProcessedSum() int64 {
	if o == nil || o.ObservabilityPipelinesBytesProcessedSum == nil {
		var ret int64
		return ret
	}
	return *o.ObservabilityPipelinesBytesProcessedSum
}

// GetObservabilityPipelinesBytesProcessedSumOk returns a tuple with the ObservabilityPipelinesBytesProcessedSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetObservabilityPipelinesBytesProcessedSumOk() (*int64, bool) {
	if o == nil || o.ObservabilityPipelinesBytesProcessedSum == nil {
		return nil, false
	}
	return o.ObservabilityPipelinesBytesProcessedSum, true
}

// HasObservabilityPipelinesBytesProcessedSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasObservabilityPipelinesBytesProcessedSum() bool {
	return o != nil && o.ObservabilityPipelinesBytesProcessedSum != nil
}

// SetObservabilityPipelinesBytesProcessedSum gets a reference to the given int64 and assigns it to the ObservabilityPipelinesBytesProcessedSum field.
func (o *UsageSummaryDate) SetObservabilityPipelinesBytesProcessedSum(v int64) {
	o.ObservabilityPipelinesBytesProcessedSum = &v
}

// GetOnlineArchiveEventsCountSum returns the OnlineArchiveEventsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetOnlineArchiveEventsCountSum() int64 {
	if o == nil || o.OnlineArchiveEventsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.OnlineArchiveEventsCountSum
}

// GetOnlineArchiveEventsCountSumOk returns a tuple with the OnlineArchiveEventsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetOnlineArchiveEventsCountSumOk() (*int64, bool) {
	if o == nil || o.OnlineArchiveEventsCountSum == nil {
		return nil, false
	}
	return o.OnlineArchiveEventsCountSum, true
}

// HasOnlineArchiveEventsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasOnlineArchiveEventsCountSum() bool {
	return o != nil && o.OnlineArchiveEventsCountSum != nil
}

// SetOnlineArchiveEventsCountSum gets a reference to the given int64 and assigns it to the OnlineArchiveEventsCountSum field.
func (o *UsageSummaryDate) SetOnlineArchiveEventsCountSum(v int64) {
	o.OnlineArchiveEventsCountSum = &v
}

// GetOpentelemetryApmHostTop99p returns the OpentelemetryApmHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetOpentelemetryApmHostTop99p() int64 {
	if o == nil || o.OpentelemetryApmHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.OpentelemetryApmHostTop99p
}

// GetOpentelemetryApmHostTop99pOk returns a tuple with the OpentelemetryApmHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetOpentelemetryApmHostTop99pOk() (*int64, bool) {
	if o == nil || o.OpentelemetryApmHostTop99p == nil {
		return nil, false
	}
	return o.OpentelemetryApmHostTop99p, true
}

// HasOpentelemetryApmHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasOpentelemetryApmHostTop99p() bool {
	return o != nil && o.OpentelemetryApmHostTop99p != nil
}

// SetOpentelemetryApmHostTop99p gets a reference to the given int64 and assigns it to the OpentelemetryApmHostTop99p field.
func (o *UsageSummaryDate) SetOpentelemetryApmHostTop99p(v int64) {
	o.OpentelemetryApmHostTop99p = &v
}

// GetOpentelemetryHostTop99p returns the OpentelemetryHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetOpentelemetryHostTop99p() int64 {
	if o == nil || o.OpentelemetryHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.OpentelemetryHostTop99p
}

// GetOpentelemetryHostTop99pOk returns a tuple with the OpentelemetryHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetOpentelemetryHostTop99pOk() (*int64, bool) {
	if o == nil || o.OpentelemetryHostTop99p == nil {
		return nil, false
	}
	return o.OpentelemetryHostTop99p, true
}

// HasOpentelemetryHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasOpentelemetryHostTop99p() bool {
	return o != nil && o.OpentelemetryHostTop99p != nil
}

// SetOpentelemetryHostTop99p gets a reference to the given int64 and assigns it to the OpentelemetryHostTop99p field.
func (o *UsageSummaryDate) SetOpentelemetryHostTop99p(v int64) {
	o.OpentelemetryHostTop99p = &v
}

// GetOrgs returns the Orgs field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetOrgs() []UsageSummaryDateOrg {
	if o == nil || o.Orgs == nil {
		var ret []UsageSummaryDateOrg
		return ret
	}
	return o.Orgs
}

// GetOrgsOk returns a tuple with the Orgs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetOrgsOk() (*[]UsageSummaryDateOrg, bool) {
	if o == nil || o.Orgs == nil {
		return nil, false
	}
	return &o.Orgs, true
}

// HasOrgs returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasOrgs() bool {
	return o != nil && o.Orgs != nil
}

// SetOrgs gets a reference to the given []UsageSummaryDateOrg and assigns it to the Orgs field.
func (o *UsageSummaryDate) SetOrgs(v []UsageSummaryDateOrg) {
	o.Orgs = v
}

// GetProfilingHostTop99p returns the ProfilingHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetProfilingHostTop99p() int64 {
	if o == nil || o.ProfilingHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.ProfilingHostTop99p
}

// GetProfilingHostTop99pOk returns a tuple with the ProfilingHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetProfilingHostTop99pOk() (*int64, bool) {
	if o == nil || o.ProfilingHostTop99p == nil {
		return nil, false
	}
	return o.ProfilingHostTop99p, true
}

// HasProfilingHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasProfilingHostTop99p() bool {
	return o != nil && o.ProfilingHostTop99p != nil
}

// SetProfilingHostTop99p gets a reference to the given int64 and assigns it to the ProfilingHostTop99p field.
func (o *UsageSummaryDate) SetProfilingHostTop99p(v int64) {
	o.ProfilingHostTop99p = &v
}

// GetRumBrowserAndMobileSessionCount returns the RumBrowserAndMobileSessionCount field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetRumBrowserAndMobileSessionCount() int64 {
	if o == nil || o.RumBrowserAndMobileSessionCount == nil {
		var ret int64
		return ret
	}
	return *o.RumBrowserAndMobileSessionCount
}

// GetRumBrowserAndMobileSessionCountOk returns a tuple with the RumBrowserAndMobileSessionCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetRumBrowserAndMobileSessionCountOk() (*int64, bool) {
	if o == nil || o.RumBrowserAndMobileSessionCount == nil {
		return nil, false
	}
	return o.RumBrowserAndMobileSessionCount, true
}

// HasRumBrowserAndMobileSessionCount returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasRumBrowserAndMobileSessionCount() bool {
	return o != nil && o.RumBrowserAndMobileSessionCount != nil
}

// SetRumBrowserAndMobileSessionCount gets a reference to the given int64 and assigns it to the RumBrowserAndMobileSessionCount field.
func (o *UsageSummaryDate) SetRumBrowserAndMobileSessionCount(v int64) {
	o.RumBrowserAndMobileSessionCount = &v
}

// GetRumSessionCountSum returns the RumSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetRumSessionCountSum() int64 {
	if o == nil || o.RumSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.RumSessionCountSum
}

// GetRumSessionCountSumOk returns a tuple with the RumSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetRumSessionCountSumOk() (*int64, bool) {
	if o == nil || o.RumSessionCountSum == nil {
		return nil, false
	}
	return o.RumSessionCountSum, true
}

// HasRumSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasRumSessionCountSum() bool {
	return o != nil && o.RumSessionCountSum != nil
}

// SetRumSessionCountSum gets a reference to the given int64 and assigns it to the RumSessionCountSum field.
func (o *UsageSummaryDate) SetRumSessionCountSum(v int64) {
	o.RumSessionCountSum = &v
}

// GetRumTotalSessionCountSum returns the RumTotalSessionCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetRumTotalSessionCountSum() int64 {
	if o == nil || o.RumTotalSessionCountSum == nil {
		var ret int64
		return ret
	}
	return *o.RumTotalSessionCountSum
}

// GetRumTotalSessionCountSumOk returns a tuple with the RumTotalSessionCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetRumTotalSessionCountSumOk() (*int64, bool) {
	if o == nil || o.RumTotalSessionCountSum == nil {
		return nil, false
	}
	return o.RumTotalSessionCountSum, true
}

// HasRumTotalSessionCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasRumTotalSessionCountSum() bool {
	return o != nil && o.RumTotalSessionCountSum != nil
}

// SetRumTotalSessionCountSum gets a reference to the given int64 and assigns it to the RumTotalSessionCountSum field.
func (o *UsageSummaryDate) SetRumTotalSessionCountSum(v int64) {
	o.RumTotalSessionCountSum = &v
}

// GetRumUnitsSum returns the RumUnitsSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetRumUnitsSum() int64 {
	if o == nil || o.RumUnitsSum == nil {
		var ret int64
		return ret
	}
	return *o.RumUnitsSum
}

// GetRumUnitsSumOk returns a tuple with the RumUnitsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetRumUnitsSumOk() (*int64, bool) {
	if o == nil || o.RumUnitsSum == nil {
		return nil, false
	}
	return o.RumUnitsSum, true
}

// HasRumUnitsSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasRumUnitsSum() bool {
	return o != nil && o.RumUnitsSum != nil
}

// SetRumUnitsSum gets a reference to the given int64 and assigns it to the RumUnitsSum field.
func (o *UsageSummaryDate) SetRumUnitsSum(v int64) {
	o.RumUnitsSum = &v
}

// GetSdsApmScannedBytesSum returns the SdsApmScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetSdsApmScannedBytesSum() int64 {
	if o == nil || o.SdsApmScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsApmScannedBytesSum
}

// GetSdsApmScannedBytesSumOk returns a tuple with the SdsApmScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetSdsApmScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsApmScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsApmScannedBytesSum, true
}

// HasSdsApmScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasSdsApmScannedBytesSum() bool {
	return o != nil && o.SdsApmScannedBytesSum != nil
}

// SetSdsApmScannedBytesSum gets a reference to the given int64 and assigns it to the SdsApmScannedBytesSum field.
func (o *UsageSummaryDate) SetSdsApmScannedBytesSum(v int64) {
	o.SdsApmScannedBytesSum = &v
}

// GetSdsEventsScannedBytesSum returns the SdsEventsScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetSdsEventsScannedBytesSum() int64 {
	if o == nil || o.SdsEventsScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsEventsScannedBytesSum
}

// GetSdsEventsScannedBytesSumOk returns a tuple with the SdsEventsScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetSdsEventsScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsEventsScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsEventsScannedBytesSum, true
}

// HasSdsEventsScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasSdsEventsScannedBytesSum() bool {
	return o != nil && o.SdsEventsScannedBytesSum != nil
}

// SetSdsEventsScannedBytesSum gets a reference to the given int64 and assigns it to the SdsEventsScannedBytesSum field.
func (o *UsageSummaryDate) SetSdsEventsScannedBytesSum(v int64) {
	o.SdsEventsScannedBytesSum = &v
}

// GetSdsLogsScannedBytesSum returns the SdsLogsScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetSdsLogsScannedBytesSum() int64 {
	if o == nil || o.SdsLogsScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsLogsScannedBytesSum
}

// GetSdsLogsScannedBytesSumOk returns a tuple with the SdsLogsScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetSdsLogsScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsLogsScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsLogsScannedBytesSum, true
}

// HasSdsLogsScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasSdsLogsScannedBytesSum() bool {
	return o != nil && o.SdsLogsScannedBytesSum != nil
}

// SetSdsLogsScannedBytesSum gets a reference to the given int64 and assigns it to the SdsLogsScannedBytesSum field.
func (o *UsageSummaryDate) SetSdsLogsScannedBytesSum(v int64) {
	o.SdsLogsScannedBytesSum = &v
}

// GetSdsRumScannedBytesSum returns the SdsRumScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetSdsRumScannedBytesSum() int64 {
	if o == nil || o.SdsRumScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsRumScannedBytesSum
}

// GetSdsRumScannedBytesSumOk returns a tuple with the SdsRumScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetSdsRumScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsRumScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsRumScannedBytesSum, true
}

// HasSdsRumScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasSdsRumScannedBytesSum() bool {
	return o != nil && o.SdsRumScannedBytesSum != nil
}

// SetSdsRumScannedBytesSum gets a reference to the given int64 and assigns it to the SdsRumScannedBytesSum field.
func (o *UsageSummaryDate) SetSdsRumScannedBytesSum(v int64) {
	o.SdsRumScannedBytesSum = &v
}

// GetSdsTotalScannedBytesSum returns the SdsTotalScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetSdsTotalScannedBytesSum() int64 {
	if o == nil || o.SdsTotalScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsTotalScannedBytesSum
}

// GetSdsTotalScannedBytesSumOk returns a tuple with the SdsTotalScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetSdsTotalScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsTotalScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsTotalScannedBytesSum, true
}

// HasSdsTotalScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasSdsTotalScannedBytesSum() bool {
	return o != nil && o.SdsTotalScannedBytesSum != nil
}

// SetSdsTotalScannedBytesSum gets a reference to the given int64 and assigns it to the SdsTotalScannedBytesSum field.
func (o *UsageSummaryDate) SetSdsTotalScannedBytesSum(v int64) {
	o.SdsTotalScannedBytesSum = &v
}

// GetSyntheticsBrowserCheckCallsCountSum returns the SyntheticsBrowserCheckCallsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetSyntheticsBrowserCheckCallsCountSum() int64 {
	if o == nil || o.SyntheticsBrowserCheckCallsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.SyntheticsBrowserCheckCallsCountSum
}

// GetSyntheticsBrowserCheckCallsCountSumOk returns a tuple with the SyntheticsBrowserCheckCallsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetSyntheticsBrowserCheckCallsCountSumOk() (*int64, bool) {
	if o == nil || o.SyntheticsBrowserCheckCallsCountSum == nil {
		return nil, false
	}
	return o.SyntheticsBrowserCheckCallsCountSum, true
}

// HasSyntheticsBrowserCheckCallsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasSyntheticsBrowserCheckCallsCountSum() bool {
	return o != nil && o.SyntheticsBrowserCheckCallsCountSum != nil
}

// SetSyntheticsBrowserCheckCallsCountSum gets a reference to the given int64 and assigns it to the SyntheticsBrowserCheckCallsCountSum field.
func (o *UsageSummaryDate) SetSyntheticsBrowserCheckCallsCountSum(v int64) {
	o.SyntheticsBrowserCheckCallsCountSum = &v
}

// GetSyntheticsCheckCallsCountSum returns the SyntheticsCheckCallsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetSyntheticsCheckCallsCountSum() int64 {
	if o == nil || o.SyntheticsCheckCallsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.SyntheticsCheckCallsCountSum
}

// GetSyntheticsCheckCallsCountSumOk returns a tuple with the SyntheticsCheckCallsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetSyntheticsCheckCallsCountSumOk() (*int64, bool) {
	if o == nil || o.SyntheticsCheckCallsCountSum == nil {
		return nil, false
	}
	return o.SyntheticsCheckCallsCountSum, true
}

// HasSyntheticsCheckCallsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasSyntheticsCheckCallsCountSum() bool {
	return o != nil && o.SyntheticsCheckCallsCountSum != nil
}

// SetSyntheticsCheckCallsCountSum gets a reference to the given int64 and assigns it to the SyntheticsCheckCallsCountSum field.
func (o *UsageSummaryDate) SetSyntheticsCheckCallsCountSum(v int64) {
	o.SyntheticsCheckCallsCountSum = &v
}

// GetSyntheticsParallelTestingMaxSlotsHwm returns the SyntheticsParallelTestingMaxSlotsHwm field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetSyntheticsParallelTestingMaxSlotsHwm() int64 {
	if o == nil || o.SyntheticsParallelTestingMaxSlotsHwm == nil {
		var ret int64
		return ret
	}
	return *o.SyntheticsParallelTestingMaxSlotsHwm
}

// GetSyntheticsParallelTestingMaxSlotsHwmOk returns a tuple with the SyntheticsParallelTestingMaxSlotsHwm field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetSyntheticsParallelTestingMaxSlotsHwmOk() (*int64, bool) {
	if o == nil || o.SyntheticsParallelTestingMaxSlotsHwm == nil {
		return nil, false
	}
	return o.SyntheticsParallelTestingMaxSlotsHwm, true
}

// HasSyntheticsParallelTestingMaxSlotsHwm returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasSyntheticsParallelTestingMaxSlotsHwm() bool {
	return o != nil && o.SyntheticsParallelTestingMaxSlotsHwm != nil
}

// SetSyntheticsParallelTestingMaxSlotsHwm gets a reference to the given int64 and assigns it to the SyntheticsParallelTestingMaxSlotsHwm field.
func (o *UsageSummaryDate) SetSyntheticsParallelTestingMaxSlotsHwm(v int64) {
	o.SyntheticsParallelTestingMaxSlotsHwm = &v
}

// GetTraceSearchIndexedEventsCountSum returns the TraceSearchIndexedEventsCountSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetTraceSearchIndexedEventsCountSum() int64 {
	if o == nil || o.TraceSearchIndexedEventsCountSum == nil {
		var ret int64
		return ret
	}
	return *o.TraceSearchIndexedEventsCountSum
}

// GetTraceSearchIndexedEventsCountSumOk returns a tuple with the TraceSearchIndexedEventsCountSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetTraceSearchIndexedEventsCountSumOk() (*int64, bool) {
	if o == nil || o.TraceSearchIndexedEventsCountSum == nil {
		return nil, false
	}
	return o.TraceSearchIndexedEventsCountSum, true
}

// HasTraceSearchIndexedEventsCountSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasTraceSearchIndexedEventsCountSum() bool {
	return o != nil && o.TraceSearchIndexedEventsCountSum != nil
}

// SetTraceSearchIndexedEventsCountSum gets a reference to the given int64 and assigns it to the TraceSearchIndexedEventsCountSum field.
func (o *UsageSummaryDate) SetTraceSearchIndexedEventsCountSum(v int64) {
	o.TraceSearchIndexedEventsCountSum = &v
}

// GetTwolIngestedEventsBytesSum returns the TwolIngestedEventsBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetTwolIngestedEventsBytesSum() int64 {
	if o == nil || o.TwolIngestedEventsBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.TwolIngestedEventsBytesSum
}

// GetTwolIngestedEventsBytesSumOk returns a tuple with the TwolIngestedEventsBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetTwolIngestedEventsBytesSumOk() (*int64, bool) {
	if o == nil || o.TwolIngestedEventsBytesSum == nil {
		return nil, false
	}
	return o.TwolIngestedEventsBytesSum, true
}

// HasTwolIngestedEventsBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasTwolIngestedEventsBytesSum() bool {
	return o != nil && o.TwolIngestedEventsBytesSum != nil
}

// SetTwolIngestedEventsBytesSum gets a reference to the given int64 and assigns it to the TwolIngestedEventsBytesSum field.
func (o *UsageSummaryDate) SetTwolIngestedEventsBytesSum(v int64) {
	o.TwolIngestedEventsBytesSum = &v
}

// GetVsphereHostTop99p returns the VsphereHostTop99p field value if set, zero value otherwise.
func (o *UsageSummaryDate) GetVsphereHostTop99p() int64 {
	if o == nil || o.VsphereHostTop99p == nil {
		var ret int64
		return ret
	}
	return *o.VsphereHostTop99p
}

// GetVsphereHostTop99pOk returns a tuple with the VsphereHostTop99p field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryDate) GetVsphereHostTop99pOk() (*int64, bool) {
	if o == nil || o.VsphereHostTop99p == nil {
		return nil, false
	}
	return o.VsphereHostTop99p, true
}

// HasVsphereHostTop99p returns a boolean if a field has been set.
func (o *UsageSummaryDate) HasVsphereHostTop99p() bool {
	return o != nil && o.VsphereHostTop99p != nil
}

// SetVsphereHostTop99p gets a reference to the given int64 and assigns it to the VsphereHostTop99p field.
func (o *UsageSummaryDate) SetVsphereHostTop99p(v int64) {
	o.VsphereHostTop99p = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageSummaryDate) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AgentHostTop99p != nil {
		toSerialize["agent_host_top99p"] = o.AgentHostTop99p
	}
	if o.ApmAzureAppServiceHostTop99p != nil {
		toSerialize["apm_azure_app_service_host_top99p"] = o.ApmAzureAppServiceHostTop99p
	}
	if o.ApmFargateCountAvg != nil {
		toSerialize["apm_fargate_count_avg"] = o.ApmFargateCountAvg
	}
	if o.ApmHostTop99p != nil {
		toSerialize["apm_host_top99p"] = o.ApmHostTop99p
	}
	if o.AppsecFargateCountAvg != nil {
		toSerialize["appsec_fargate_count_avg"] = o.AppsecFargateCountAvg
	}
	if o.AuditLogsLinesIndexedSum != nil {
		toSerialize["audit_logs_lines_indexed_sum"] = o.AuditLogsLinesIndexedSum
	}
	if o.AvgProfiledFargateTasks != nil {
		toSerialize["avg_profiled_fargate_tasks"] = o.AvgProfiledFargateTasks
	}
	if o.AwsHostTop99p != nil {
		toSerialize["aws_host_top99p"] = o.AwsHostTop99p
	}
	if o.AwsLambdaFuncCount != nil {
		toSerialize["aws_lambda_func_count"] = o.AwsLambdaFuncCount
	}
	if o.AwsLambdaInvocationsSum != nil {
		toSerialize["aws_lambda_invocations_sum"] = o.AwsLambdaInvocationsSum
	}
	if o.AzureAppServiceTop99p != nil {
		toSerialize["azure_app_service_top99p"] = o.AzureAppServiceTop99p
	}
	if o.BillableIngestedBytesSum != nil {
		toSerialize["billable_ingested_bytes_sum"] = o.BillableIngestedBytesSum
	}
	if o.BrowserRumLiteSessionCountSum != nil {
		toSerialize["browser_rum_lite_session_count_sum"] = o.BrowserRumLiteSessionCountSum
	}
	if o.BrowserRumReplaySessionCountSum != nil {
		toSerialize["browser_rum_replay_session_count_sum"] = o.BrowserRumReplaySessionCountSum
	}
	if o.BrowserRumUnitsSum != nil {
		toSerialize["browser_rum_units_sum"] = o.BrowserRumUnitsSum
	}
	if o.CiPipelineIndexedSpansSum != nil {
		toSerialize["ci_pipeline_indexed_spans_sum"] = o.CiPipelineIndexedSpansSum
	}
	if o.CiTestIndexedSpansSum != nil {
		toSerialize["ci_test_indexed_spans_sum"] = o.CiTestIndexedSpansSum
	}
	if o.CiVisibilityPipelineCommittersHwm != nil {
		toSerialize["ci_visibility_pipeline_committers_hwm"] = o.CiVisibilityPipelineCommittersHwm
	}
	if o.CiVisibilityTestCommittersHwm != nil {
		toSerialize["ci_visibility_test_committers_hwm"] = o.CiVisibilityTestCommittersHwm
	}
	if o.CloudCostManagementHostCountAvg != nil {
		toSerialize["cloud_cost_management_host_count_avg"] = o.CloudCostManagementHostCountAvg
	}
	if o.ContainerAvg != nil {
		toSerialize["container_avg"] = o.ContainerAvg
	}
	if o.ContainerExclAgentAvg != nil {
		toSerialize["container_excl_agent_avg"] = o.ContainerExclAgentAvg
	}
	if o.ContainerHwm != nil {
		toSerialize["container_hwm"] = o.ContainerHwm
	}
	if o.CspmAasHostTop99p != nil {
		toSerialize["cspm_aas_host_top99p"] = o.CspmAasHostTop99p
	}
	if o.CspmAwsHostTop99p != nil {
		toSerialize["cspm_aws_host_top99p"] = o.CspmAwsHostTop99p
	}
	if o.CspmAzureHostTop99p != nil {
		toSerialize["cspm_azure_host_top99p"] = o.CspmAzureHostTop99p
	}
	if o.CspmContainerAvg != nil {
		toSerialize["cspm_container_avg"] = o.CspmContainerAvg
	}
	if o.CspmContainerHwm != nil {
		toSerialize["cspm_container_hwm"] = o.CspmContainerHwm
	}
	if o.CspmGcpHostTop99p != nil {
		toSerialize["cspm_gcp_host_top99p"] = o.CspmGcpHostTop99p
	}
	if o.CspmHostTop99p != nil {
		toSerialize["cspm_host_top99p"] = o.CspmHostTop99p
	}
	if o.CustomTsAvg != nil {
		toSerialize["custom_ts_avg"] = o.CustomTsAvg
	}
	if o.CwsContainerCountAvg != nil {
		toSerialize["cws_container_count_avg"] = o.CwsContainerCountAvg
	}
	if o.CwsHostTop99p != nil {
		toSerialize["cws_host_top99p"] = o.CwsHostTop99p
	}
	if o.Date != nil {
		if o.Date.Nanosecond() == 0 {
			toSerialize["date"] = o.Date.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["date"] = o.Date.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.DbmHostTop99p != nil {
		toSerialize["dbm_host_top99p"] = o.DbmHostTop99p
	}
	if o.DbmQueriesCountAvg != nil {
		toSerialize["dbm_queries_count_avg"] = o.DbmQueriesCountAvg
	}
	if o.FargateTasksCountAvg != nil {
		toSerialize["fargate_tasks_count_avg"] = o.FargateTasksCountAvg
	}
	if o.FargateTasksCountHwm != nil {
		toSerialize["fargate_tasks_count_hwm"] = o.FargateTasksCountHwm
	}
	if o.GcpHostTop99p != nil {
		toSerialize["gcp_host_top99p"] = o.GcpHostTop99p
	}
	if o.HerokuHostTop99p != nil {
		toSerialize["heroku_host_top99p"] = o.HerokuHostTop99p
	}
	if o.IncidentManagementMonthlyActiveUsersHwm != nil {
		toSerialize["incident_management_monthly_active_users_hwm"] = o.IncidentManagementMonthlyActiveUsersHwm
	}
	if o.IndexedEventsCountSum != nil {
		toSerialize["indexed_events_count_sum"] = o.IndexedEventsCountSum
	}
	if o.InfraHostTop99p != nil {
		toSerialize["infra_host_top99p"] = o.InfraHostTop99p
	}
	if o.IngestedEventsBytesSum != nil {
		toSerialize["ingested_events_bytes_sum"] = o.IngestedEventsBytesSum
	}
	if o.IotDeviceSum != nil {
		toSerialize["iot_device_sum"] = o.IotDeviceSum
	}
	if o.IotDeviceTop99p != nil {
		toSerialize["iot_device_top99p"] = o.IotDeviceTop99p
	}
	if o.MobileRumLiteSessionCountSum != nil {
		toSerialize["mobile_rum_lite_session_count_sum"] = o.MobileRumLiteSessionCountSum
	}
	if o.MobileRumSessionCountAndroidSum != nil {
		toSerialize["mobile_rum_session_count_android_sum"] = o.MobileRumSessionCountAndroidSum
	}
	if o.MobileRumSessionCountIosSum != nil {
		toSerialize["mobile_rum_session_count_ios_sum"] = o.MobileRumSessionCountIosSum
	}
	if o.MobileRumSessionCountReactnativeSum != nil {
		toSerialize["mobile_rum_session_count_reactnative_sum"] = o.MobileRumSessionCountReactnativeSum
	}
	if o.MobileRumSessionCountSum != nil {
		toSerialize["mobile_rum_session_count_sum"] = o.MobileRumSessionCountSum
	}
	if o.MobileRumUnitsSum != nil {
		toSerialize["mobile_rum_units_sum"] = o.MobileRumUnitsSum
	}
	if o.NetflowIndexedEventsCountSum != nil {
		toSerialize["netflow_indexed_events_count_sum"] = o.NetflowIndexedEventsCountSum
	}
	if o.NpmHostTop99p != nil {
		toSerialize["npm_host_top99p"] = o.NpmHostTop99p
	}
	if o.ObservabilityPipelinesBytesProcessedSum != nil {
		toSerialize["observability_pipelines_bytes_processed_sum"] = o.ObservabilityPipelinesBytesProcessedSum
	}
	if o.OnlineArchiveEventsCountSum != nil {
		toSerialize["online_archive_events_count_sum"] = o.OnlineArchiveEventsCountSum
	}
	if o.OpentelemetryApmHostTop99p != nil {
		toSerialize["opentelemetry_apm_host_top99p"] = o.OpentelemetryApmHostTop99p
	}
	if o.OpentelemetryHostTop99p != nil {
		toSerialize["opentelemetry_host_top99p"] = o.OpentelemetryHostTop99p
	}
	if o.Orgs != nil {
		toSerialize["orgs"] = o.Orgs
	}
	if o.ProfilingHostTop99p != nil {
		toSerialize["profiling_host_top99p"] = o.ProfilingHostTop99p
	}
	if o.RumBrowserAndMobileSessionCount != nil {
		toSerialize["rum_browser_and_mobile_session_count"] = o.RumBrowserAndMobileSessionCount
	}
	if o.RumSessionCountSum != nil {
		toSerialize["rum_session_count_sum"] = o.RumSessionCountSum
	}
	if o.RumTotalSessionCountSum != nil {
		toSerialize["rum_total_session_count_sum"] = o.RumTotalSessionCountSum
	}
	if o.RumUnitsSum != nil {
		toSerialize["rum_units_sum"] = o.RumUnitsSum
	}
	if o.SdsApmScannedBytesSum != nil {
		toSerialize["sds_apm_scanned_bytes_sum"] = o.SdsApmScannedBytesSum
	}
	if o.SdsEventsScannedBytesSum != nil {
		toSerialize["sds_events_scanned_bytes_sum"] = o.SdsEventsScannedBytesSum
	}
	if o.SdsLogsScannedBytesSum != nil {
		toSerialize["sds_logs_scanned_bytes_sum"] = o.SdsLogsScannedBytesSum
	}
	if o.SdsRumScannedBytesSum != nil {
		toSerialize["sds_rum_scanned_bytes_sum"] = o.SdsRumScannedBytesSum
	}
	if o.SdsTotalScannedBytesSum != nil {
		toSerialize["sds_total_scanned_bytes_sum"] = o.SdsTotalScannedBytesSum
	}
	if o.SyntheticsBrowserCheckCallsCountSum != nil {
		toSerialize["synthetics_browser_check_calls_count_sum"] = o.SyntheticsBrowserCheckCallsCountSum
	}
	if o.SyntheticsCheckCallsCountSum != nil {
		toSerialize["synthetics_check_calls_count_sum"] = o.SyntheticsCheckCallsCountSum
	}
	if o.SyntheticsParallelTestingMaxSlotsHwm != nil {
		toSerialize["synthetics_parallel_testing_max_slots_hwm"] = o.SyntheticsParallelTestingMaxSlotsHwm
	}
	if o.TraceSearchIndexedEventsCountSum != nil {
		toSerialize["trace_search_indexed_events_count_sum"] = o.TraceSearchIndexedEventsCountSum
	}
	if o.TwolIngestedEventsBytesSum != nil {
		toSerialize["twol_ingested_events_bytes_sum"] = o.TwolIngestedEventsBytesSum
	}
	if o.VsphereHostTop99p != nil {
		toSerialize["vsphere_host_top99p"] = o.VsphereHostTop99p
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageSummaryDate) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AgentHostTop99p                         *int64                `json:"agent_host_top99p,omitempty"`
		ApmAzureAppServiceHostTop99p            *int64                `json:"apm_azure_app_service_host_top99p,omitempty"`
		ApmFargateCountAvg                      *int64                `json:"apm_fargate_count_avg,omitempty"`
		ApmHostTop99p                           *int64                `json:"apm_host_top99p,omitempty"`
		AppsecFargateCountAvg                   *int64                `json:"appsec_fargate_count_avg,omitempty"`
		AuditLogsLinesIndexedSum                *int64                `json:"audit_logs_lines_indexed_sum,omitempty"`
		AvgProfiledFargateTasks                 *int64                `json:"avg_profiled_fargate_tasks,omitempty"`
		AwsHostTop99p                           *int64                `json:"aws_host_top99p,omitempty"`
		AwsLambdaFuncCount                      *int64                `json:"aws_lambda_func_count,omitempty"`
		AwsLambdaInvocationsSum                 *int64                `json:"aws_lambda_invocations_sum,omitempty"`
		AzureAppServiceTop99p                   *int64                `json:"azure_app_service_top99p,omitempty"`
		BillableIngestedBytesSum                *int64                `json:"billable_ingested_bytes_sum,omitempty"`
		BrowserRumLiteSessionCountSum           *int64                `json:"browser_rum_lite_session_count_sum,omitempty"`
		BrowserRumReplaySessionCountSum         *int64                `json:"browser_rum_replay_session_count_sum,omitempty"`
		BrowserRumUnitsSum                      *int64                `json:"browser_rum_units_sum,omitempty"`
		CiPipelineIndexedSpansSum               *int64                `json:"ci_pipeline_indexed_spans_sum,omitempty"`
		CiTestIndexedSpansSum                   *int64                `json:"ci_test_indexed_spans_sum,omitempty"`
		CiVisibilityPipelineCommittersHwm       *int64                `json:"ci_visibility_pipeline_committers_hwm,omitempty"`
		CiVisibilityTestCommittersHwm           *int64                `json:"ci_visibility_test_committers_hwm,omitempty"`
		CloudCostManagementHostCountAvg         *int64                `json:"cloud_cost_management_host_count_avg,omitempty"`
		ContainerAvg                            *int64                `json:"container_avg,omitempty"`
		ContainerExclAgentAvg                   *int64                `json:"container_excl_agent_avg,omitempty"`
		ContainerHwm                            *int64                `json:"container_hwm,omitempty"`
		CspmAasHostTop99p                       *int64                `json:"cspm_aas_host_top99p,omitempty"`
		CspmAwsHostTop99p                       *int64                `json:"cspm_aws_host_top99p,omitempty"`
		CspmAzureHostTop99p                     *int64                `json:"cspm_azure_host_top99p,omitempty"`
		CspmContainerAvg                        *int64                `json:"cspm_container_avg,omitempty"`
		CspmContainerHwm                        *int64                `json:"cspm_container_hwm,omitempty"`
		CspmGcpHostTop99p                       *int64                `json:"cspm_gcp_host_top99p,omitempty"`
		CspmHostTop99p                          *int64                `json:"cspm_host_top99p,omitempty"`
		CustomTsAvg                             *int64                `json:"custom_ts_avg,omitempty"`
		CwsContainerCountAvg                    *int64                `json:"cws_container_count_avg,omitempty"`
		CwsHostTop99p                           *int64                `json:"cws_host_top99p,omitempty"`
		Date                                    *time.Time            `json:"date,omitempty"`
		DbmHostTop99p                           *int64                `json:"dbm_host_top99p,omitempty"`
		DbmQueriesCountAvg                      *int64                `json:"dbm_queries_count_avg,omitempty"`
		FargateTasksCountAvg                    *int64                `json:"fargate_tasks_count_avg,omitempty"`
		FargateTasksCountHwm                    *int64                `json:"fargate_tasks_count_hwm,omitempty"`
		GcpHostTop99p                           *int64                `json:"gcp_host_top99p,omitempty"`
		HerokuHostTop99p                        *int64                `json:"heroku_host_top99p,omitempty"`
		IncidentManagementMonthlyActiveUsersHwm *int64                `json:"incident_management_monthly_active_users_hwm,omitempty"`
		IndexedEventsCountSum                   *int64                `json:"indexed_events_count_sum,omitempty"`
		InfraHostTop99p                         *int64                `json:"infra_host_top99p,omitempty"`
		IngestedEventsBytesSum                  *int64                `json:"ingested_events_bytes_sum,omitempty"`
		IotDeviceSum                            *int64                `json:"iot_device_sum,omitempty"`
		IotDeviceTop99p                         *int64                `json:"iot_device_top99p,omitempty"`
		MobileRumLiteSessionCountSum            *int64                `json:"mobile_rum_lite_session_count_sum,omitempty"`
		MobileRumSessionCountAndroidSum         *int64                `json:"mobile_rum_session_count_android_sum,omitempty"`
		MobileRumSessionCountIosSum             *int64                `json:"mobile_rum_session_count_ios_sum,omitempty"`
		MobileRumSessionCountReactnativeSum     *int64                `json:"mobile_rum_session_count_reactnative_sum,omitempty"`
		MobileRumSessionCountSum                *int64                `json:"mobile_rum_session_count_sum,omitempty"`
		MobileRumUnitsSum                       *int64                `json:"mobile_rum_units_sum,omitempty"`
		NetflowIndexedEventsCountSum            *int64                `json:"netflow_indexed_events_count_sum,omitempty"`
		NpmHostTop99p                           *int64                `json:"npm_host_top99p,omitempty"`
		ObservabilityPipelinesBytesProcessedSum *int64                `json:"observability_pipelines_bytes_processed_sum,omitempty"`
		OnlineArchiveEventsCountSum             *int64                `json:"online_archive_events_count_sum,omitempty"`
		OpentelemetryApmHostTop99p              *int64                `json:"opentelemetry_apm_host_top99p,omitempty"`
		OpentelemetryHostTop99p                 *int64                `json:"opentelemetry_host_top99p,omitempty"`
		Orgs                                    []UsageSummaryDateOrg `json:"orgs,omitempty"`
		ProfilingHostTop99p                     *int64                `json:"profiling_host_top99p,omitempty"`
		RumBrowserAndMobileSessionCount         *int64                `json:"rum_browser_and_mobile_session_count,omitempty"`
		RumSessionCountSum                      *int64                `json:"rum_session_count_sum,omitempty"`
		RumTotalSessionCountSum                 *int64                `json:"rum_total_session_count_sum,omitempty"`
		RumUnitsSum                             *int64                `json:"rum_units_sum,omitempty"`
		SdsApmScannedBytesSum                   *int64                `json:"sds_apm_scanned_bytes_sum,omitempty"`
		SdsEventsScannedBytesSum                *int64                `json:"sds_events_scanned_bytes_sum,omitempty"`
		SdsLogsScannedBytesSum                  *int64                `json:"sds_logs_scanned_bytes_sum,omitempty"`
		SdsRumScannedBytesSum                   *int64                `json:"sds_rum_scanned_bytes_sum,omitempty"`
		SdsTotalScannedBytesSum                 *int64                `json:"sds_total_scanned_bytes_sum,omitempty"`
		SyntheticsBrowserCheckCallsCountSum     *int64                `json:"synthetics_browser_check_calls_count_sum,omitempty"`
		SyntheticsCheckCallsCountSum            *int64                `json:"synthetics_check_calls_count_sum,omitempty"`
		SyntheticsParallelTestingMaxSlotsHwm    *int64                `json:"synthetics_parallel_testing_max_slots_hwm,omitempty"`
		TraceSearchIndexedEventsCountSum        *int64                `json:"trace_search_indexed_events_count_sum,omitempty"`
		TwolIngestedEventsBytesSum              *int64                `json:"twol_ingested_events_bytes_sum,omitempty"`
		VsphereHostTop99p                       *int64                `json:"vsphere_host_top99p,omitempty"`
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
	o.AgentHostTop99p = all.AgentHostTop99p
	o.ApmAzureAppServiceHostTop99p = all.ApmAzureAppServiceHostTop99p
	o.ApmFargateCountAvg = all.ApmFargateCountAvg
	o.ApmHostTop99p = all.ApmHostTop99p
	o.AppsecFargateCountAvg = all.AppsecFargateCountAvg
	o.AuditLogsLinesIndexedSum = all.AuditLogsLinesIndexedSum
	o.AvgProfiledFargateTasks = all.AvgProfiledFargateTasks
	o.AwsHostTop99p = all.AwsHostTop99p
	o.AwsLambdaFuncCount = all.AwsLambdaFuncCount
	o.AwsLambdaInvocationsSum = all.AwsLambdaInvocationsSum
	o.AzureAppServiceTop99p = all.AzureAppServiceTop99p
	o.BillableIngestedBytesSum = all.BillableIngestedBytesSum
	o.BrowserRumLiteSessionCountSum = all.BrowserRumLiteSessionCountSum
	o.BrowserRumReplaySessionCountSum = all.BrowserRumReplaySessionCountSum
	o.BrowserRumUnitsSum = all.BrowserRumUnitsSum
	o.CiPipelineIndexedSpansSum = all.CiPipelineIndexedSpansSum
	o.CiTestIndexedSpansSum = all.CiTestIndexedSpansSum
	o.CiVisibilityPipelineCommittersHwm = all.CiVisibilityPipelineCommittersHwm
	o.CiVisibilityTestCommittersHwm = all.CiVisibilityTestCommittersHwm
	o.CloudCostManagementHostCountAvg = all.CloudCostManagementHostCountAvg
	o.ContainerAvg = all.ContainerAvg
	o.ContainerExclAgentAvg = all.ContainerExclAgentAvg
	o.ContainerHwm = all.ContainerHwm
	o.CspmAasHostTop99p = all.CspmAasHostTop99p
	o.CspmAwsHostTop99p = all.CspmAwsHostTop99p
	o.CspmAzureHostTop99p = all.CspmAzureHostTop99p
	o.CspmContainerAvg = all.CspmContainerAvg
	o.CspmContainerHwm = all.CspmContainerHwm
	o.CspmGcpHostTop99p = all.CspmGcpHostTop99p
	o.CspmHostTop99p = all.CspmHostTop99p
	o.CustomTsAvg = all.CustomTsAvg
	o.CwsContainerCountAvg = all.CwsContainerCountAvg
	o.CwsHostTop99p = all.CwsHostTop99p
	o.Date = all.Date
	o.DbmHostTop99p = all.DbmHostTop99p
	o.DbmQueriesCountAvg = all.DbmQueriesCountAvg
	o.FargateTasksCountAvg = all.FargateTasksCountAvg
	o.FargateTasksCountHwm = all.FargateTasksCountHwm
	o.GcpHostTop99p = all.GcpHostTop99p
	o.HerokuHostTop99p = all.HerokuHostTop99p
	o.IncidentManagementMonthlyActiveUsersHwm = all.IncidentManagementMonthlyActiveUsersHwm
	o.IndexedEventsCountSum = all.IndexedEventsCountSum
	o.InfraHostTop99p = all.InfraHostTop99p
	o.IngestedEventsBytesSum = all.IngestedEventsBytesSum
	o.IotDeviceSum = all.IotDeviceSum
	o.IotDeviceTop99p = all.IotDeviceTop99p
	o.MobileRumLiteSessionCountSum = all.MobileRumLiteSessionCountSum
	o.MobileRumSessionCountAndroidSum = all.MobileRumSessionCountAndroidSum
	o.MobileRumSessionCountIosSum = all.MobileRumSessionCountIosSum
	o.MobileRumSessionCountReactnativeSum = all.MobileRumSessionCountReactnativeSum
	o.MobileRumSessionCountSum = all.MobileRumSessionCountSum
	o.MobileRumUnitsSum = all.MobileRumUnitsSum
	o.NetflowIndexedEventsCountSum = all.NetflowIndexedEventsCountSum
	o.NpmHostTop99p = all.NpmHostTop99p
	o.ObservabilityPipelinesBytesProcessedSum = all.ObservabilityPipelinesBytesProcessedSum
	o.OnlineArchiveEventsCountSum = all.OnlineArchiveEventsCountSum
	o.OpentelemetryApmHostTop99p = all.OpentelemetryApmHostTop99p
	o.OpentelemetryHostTop99p = all.OpentelemetryHostTop99p
	o.Orgs = all.Orgs
	o.ProfilingHostTop99p = all.ProfilingHostTop99p
	o.RumBrowserAndMobileSessionCount = all.RumBrowserAndMobileSessionCount
	o.RumSessionCountSum = all.RumSessionCountSum
	o.RumTotalSessionCountSum = all.RumTotalSessionCountSum
	o.RumUnitsSum = all.RumUnitsSum
	o.SdsApmScannedBytesSum = all.SdsApmScannedBytesSum
	o.SdsEventsScannedBytesSum = all.SdsEventsScannedBytesSum
	o.SdsLogsScannedBytesSum = all.SdsLogsScannedBytesSum
	o.SdsRumScannedBytesSum = all.SdsRumScannedBytesSum
	o.SdsTotalScannedBytesSum = all.SdsTotalScannedBytesSum
	o.SyntheticsBrowserCheckCallsCountSum = all.SyntheticsBrowserCheckCallsCountSum
	o.SyntheticsCheckCallsCountSum = all.SyntheticsCheckCallsCountSum
	o.SyntheticsParallelTestingMaxSlotsHwm = all.SyntheticsParallelTestingMaxSlotsHwm
	o.TraceSearchIndexedEventsCountSum = all.TraceSearchIndexedEventsCountSum
	o.TwolIngestedEventsBytesSum = all.TwolIngestedEventsBytesSum
	o.VsphereHostTop99p = all.VsphereHostTop99p
	return nil
}
