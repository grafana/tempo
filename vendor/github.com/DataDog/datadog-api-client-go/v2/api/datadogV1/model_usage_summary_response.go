// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadogV1

import (
	"encoding/json"
	"time"
)

// UsageSummaryResponse Response summarizing all usage aggregated across the months in the request for all organizations, and broken down by month and by organization.
type UsageSummaryResponse struct {
	// Shows the 99th percentile of all agent hosts over all hours in the current months for all organizations.
	AgentHostTop99pSum *int64 `json:"agent_host_top99p_sum,omitempty"`
	// Shows the 99th percentile of all Azure app services using APM over all hours in the current months all organizations.
	ApmAzureAppServiceHostTop99pSum *int64 `json:"apm_azure_app_service_host_top99p_sum,omitempty"`
	// Shows the average of all APM ECS Fargate tasks over all hours in the current months for all organizations.
	ApmFargateCountAvgSum *int64 `json:"apm_fargate_count_avg_sum,omitempty"`
	// Shows the 99th percentile of all distinct APM hosts over all hours in the current months for all organizations.
	ApmHostTop99pSum *int64 `json:"apm_host_top99p_sum,omitempty"`
	// Shows the average of all Application Security Monitoring ECS Fargate tasks over all hours in the current months for all organizations.
	AppsecFargateCountAvgSum *int64 `json:"appsec_fargate_count_avg_sum,omitempty"`
	// Shows the sum of all audit logs lines indexed over all hours in the current months for all organizations.
	AuditLogsLinesIndexedAggSum *int64 `json:"audit_logs_lines_indexed_agg_sum,omitempty"`
	// Shows the average of all profiled Fargate tasks over all hours in the current months for all organizations.
	AvgProfiledFargateTasksSum *int64 `json:"avg_profiled_fargate_tasks_sum,omitempty"`
	// Shows the 99th percentile of all AWS hosts over all hours in the current months for all organizations.
	AwsHostTop99pSum *int64 `json:"aws_host_top99p_sum,omitempty"`
	// Shows the average of the number of functions that executed 1 or more times each hour in the current months for all organizations.
	AwsLambdaFuncCount *int64 `json:"aws_lambda_func_count,omitempty"`
	// Shows the sum of all AWS Lambda invocations over all hours in the current months for all organizations.
	AwsLambdaInvocationsSum *int64 `json:"aws_lambda_invocations_sum,omitempty"`
	// Shows the 99th percentile of all Azure app services over all hours in the current months for all organizations.
	AzureAppServiceTop99pSum *int64 `json:"azure_app_service_top99p_sum,omitempty"`
	// Shows the 99th percentile of all Azure hosts over all hours in the current months for all organizations.
	AzureHostTop99pSum *int64 `json:"azure_host_top99p_sum,omitempty"`
	// Shows the sum of all log bytes ingested over all hours in the current months for all organizations.
	BillableIngestedBytesAggSum *int64 `json:"billable_ingested_bytes_agg_sum,omitempty"`
	// Shows the sum of all browser lite sessions over all hours in the current months for all organizations.
	BrowserRumLiteSessionCountAggSum *int64 `json:"browser_rum_lite_session_count_agg_sum,omitempty"`
	// Shows the sum of all browser replay sessions over all hours in the current months for all organizations.
	BrowserRumReplaySessionCountAggSum *int64 `json:"browser_rum_replay_session_count_agg_sum,omitempty"`
	// Shows the sum of all browser RUM units over all hours in the current months for all organizations.
	BrowserRumUnitsAggSum *int64 `json:"browser_rum_units_agg_sum,omitempty"`
	// Shows the sum of all CI pipeline indexed spans over all hours in the current months for all organizations.
	CiPipelineIndexedSpansAggSum *int64 `json:"ci_pipeline_indexed_spans_agg_sum,omitempty"`
	// Shows the sum of all CI test indexed spans over all hours in the current months for all organizations.
	CiTestIndexedSpansAggSum *int64 `json:"ci_test_indexed_spans_agg_sum,omitempty"`
	// Shows the high-water mark of all CI visibility pipeline committers over all hours in the current months for all organizations.
	CiVisibilityPipelineCommittersHwmSum *int64 `json:"ci_visibility_pipeline_committers_hwm_sum,omitempty"`
	// Shows the high-water mark of all CI visibility test committers over all hours in the current months for all organizations.
	CiVisibilityTestCommittersHwmSum *int64 `json:"ci_visibility_test_committers_hwm_sum,omitempty"`
	// Sum of the host count average for Cloud Cost Management.
	CloudCostManagementHostCountAvgSum *int64 `json:"cloud_cost_management_host_count_avg_sum,omitempty"`
	// Shows the average of all distinct containers over all hours in the current months for all organizations.
	ContainerAvgSum *int64 `json:"container_avg_sum,omitempty"`
	// Shows the average of the containers without the Datadog Agent over all hours in the current month for all organizations.
	ContainerExclAgentAvgSum *int64 `json:"container_excl_agent_avg_sum,omitempty"`
	// Shows the sum of the high-water marks of all distinct containers over all hours in the current months for all organizations.
	ContainerHwmSum *int64 `json:"container_hwm_sum,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management Azure app services hosts over all hours in the current months for all organizations.
	CspmAasHostTop99pSum *int64 `json:"cspm_aas_host_top99p_sum,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management AWS hosts over all hours in the current months for all organizations.
	CspmAwsHostTop99pSum *int64 `json:"cspm_aws_host_top99p_sum,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management Azure hosts over all hours in the current months for all organizations.
	CspmAzureHostTop99pSum *int64 `json:"cspm_azure_host_top99p_sum,omitempty"`
	// Shows the average number of Cloud Security Posture Management containers over all hours in the current months for all organizations.
	CspmContainerAvgSum *int64 `json:"cspm_container_avg_sum,omitempty"`
	// Shows the sum of the the high-water marks of Cloud Security Posture Management containers over all hours in the current months for all organizations.
	CspmContainerHwmSum *int64 `json:"cspm_container_hwm_sum,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management GCP hosts over all hours in the current months for all organizations.
	CspmGcpHostTop99pSum *int64 `json:"cspm_gcp_host_top99p_sum,omitempty"`
	// Shows the 99th percentile of all Cloud Security Posture Management hosts over all hours in the current months for all organizations.
	CspmHostTop99pSum *int64 `json:"cspm_host_top99p_sum,omitempty"`
	// Shows the average number of distinct custom metrics over all hours in the current months for all organizations.
	CustomTsSum *int64 `json:"custom_ts_sum,omitempty"`
	// Shows the average of all distinct Cloud Workload Security containers over all hours in the current months for all organizations.
	CwsContainersAvgSum *int64 `json:"cws_containers_avg_sum,omitempty"`
	// Shows the 99th percentile of all Cloud Workload Security hosts over all hours in the current months for all organizations.
	CwsHostTop99pSum *int64 `json:"cws_host_top99p_sum,omitempty"`
	// Shows the 99th percentile of all Database Monitoring hosts over all hours in the current month for all organizations.
	DbmHostTop99pSum *int64 `json:"dbm_host_top99p_sum,omitempty"`
	// Shows the average of all distinct Database Monitoring Normalized Queries over all hours in the current month for all organizations.
	DbmQueriesAvgSum *int64 `json:"dbm_queries_avg_sum,omitempty"`
	// Shows the last date of usage in the current months for all organizations.
	EndDate *time.Time `json:"end_date,omitempty"`
	// Shows the average of all Fargate tasks over all hours in the current months for all organizations.
	FargateTasksCountAvgSum *int64 `json:"fargate_tasks_count_avg_sum,omitempty"`
	// Shows the sum of the high-water marks of all Fargate tasks over all hours in the current months for all organizations.
	FargateTasksCountHwmSum *int64 `json:"fargate_tasks_count_hwm_sum,omitempty"`
	// Shows the 99th percentile of all GCP hosts over all hours in the current months for all organizations.
	GcpHostTop99pSum *int64 `json:"gcp_host_top99p_sum,omitempty"`
	// Shows the 99th percentile of all Heroku dynos over all hours in the current months for all organizations.
	HerokuHostTop99pSum *int64 `json:"heroku_host_top99p_sum,omitempty"`
	// Shows sum of the the high-water marks of incident management monthly active users in the current months for all organizations.
	IncidentManagementMonthlyActiveUsersHwmSum *int64 `json:"incident_management_monthly_active_users_hwm_sum,omitempty"`
	// Shows the sum of all log events indexed over all hours in the current months for all organizations.
	IndexedEventsCountAggSum *int64 `json:"indexed_events_count_agg_sum,omitempty"`
	// Shows the 99th percentile of all distinct infrastructure hosts over all hours in the current months for all organizations.
	InfraHostTop99pSum *int64 `json:"infra_host_top99p_sum,omitempty"`
	// Shows the sum of all log bytes ingested over all hours in the current months for all organizations.
	IngestedEventsBytesAggSum *int64 `json:"ingested_events_bytes_agg_sum,omitempty"`
	// Shows the sum of all IoT devices over all hours in the current months for all organizations.
	IotDeviceAggSum *int64 `json:"iot_device_agg_sum,omitempty"`
	// Shows the 99th percentile of all IoT devices over all hours in the current months of all organizations.
	IotDeviceTop99pSum *int64 `json:"iot_device_top99p_sum,omitempty"`
	// Shows the the most recent hour in the current months for all organizations for which all usages were calculated.
	LastUpdated *time.Time `json:"last_updated,omitempty"`
	// Shows the sum of all live logs indexed over all hours in the current months for all organizations (data available as of December 1, 2020).
	LiveIndexedEventsAggSum *int64 `json:"live_indexed_events_agg_sum,omitempty"`
	// Shows the sum of all live logs bytes ingested over all hours in the current months for all organizations (data available as of December 1, 2020).
	LiveIngestedBytesAggSum *int64 `json:"live_ingested_bytes_agg_sum,omitempty"`
	// Object containing logs usage data broken down by retention period.
	LogsByRetention *LogsByRetention `json:"logs_by_retention,omitempty"`
	// Shows the sum of all mobile lite sessions over all hours in the current months for all organizations.
	MobileRumLiteSessionCountAggSum *int64 `json:"mobile_rum_lite_session_count_agg_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions over all hours in the current months for all organizations.
	MobileRumSessionCountAggSum *int64 `json:"mobile_rum_session_count_agg_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions on Android over all hours in the current months for all organizations.
	MobileRumSessionCountAndroidAggSum *int64 `json:"mobile_rum_session_count_android_agg_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions on iOS over all hours in the current months for all organizations.
	MobileRumSessionCountIosAggSum *int64 `json:"mobile_rum_session_count_ios_agg_sum,omitempty"`
	// Shows the sum of all mobile RUM Sessions on React Native over all hours in the current months for all organizations.
	MobileRumSessionCountReactnativeAggSum *int64 `json:"mobile_rum_session_count_reactnative_agg_sum,omitempty"`
	// Shows the sum of all mobile RUM units over all hours in the current months for all organizations.
	MobileRumUnitsAggSum *int64 `json:"mobile_rum_units_agg_sum,omitempty"`
	// Shows the sum of all Network flows indexed over all hours in the current months for all organizations.
	NetflowIndexedEventsCountAggSum *int64 `json:"netflow_indexed_events_count_agg_sum,omitempty"`
	// Shows the 99th percentile of all distinct Networks hosts over all hours in the current months for all organizations.
	NpmHostTop99pSum *int64 `json:"npm_host_top99p_sum,omitempty"`
	// Sum of all observability pipelines bytes processed over all hours in the current months for all organizations.
	ObservabilityPipelinesBytesProcessedAggSum *int64 `json:"observability_pipelines_bytes_processed_agg_sum,omitempty"`
	// Sum of all online archived events over all hours in the current months for all organizations.
	OnlineArchiveEventsCountAggSum *int64 `json:"online_archive_events_count_agg_sum,omitempty"`
	// Shows the 99th percentile of APM hosts reported by the Datadog exporter for the OpenTelemetry Collector over all hours in the current months for all organizations.
	OpentelemetryApmHostTop99pSum *int64 `json:"opentelemetry_apm_host_top99p_sum,omitempty"`
	// Shows the 99th percentile of all hosts reported by the Datadog exporter for the OpenTelemetry Collector over all hours in the current months for all organizations.
	OpentelemetryHostTop99pSum *int64 `json:"opentelemetry_host_top99p_sum,omitempty"`
	// Shows the average number of profiled containers over all hours in the current months for all organizations.
	ProfilingContainerAgentCountAvg *int64 `json:"profiling_container_agent_count_avg,omitempty"`
	// Shows the 99th percentile of all profiled hosts over all hours in the current months for all organizations.
	ProfilingHostCountTop99pSum *int64 `json:"profiling_host_count_top99p_sum,omitempty"`
	// Shows the sum of all rehydrated logs indexed over all hours in the current months for all organizations (data available as of December 1, 2020).
	RehydratedIndexedEventsAggSum *int64 `json:"rehydrated_indexed_events_agg_sum,omitempty"`
	// Shows the sum of all rehydrated logs bytes ingested over all hours in the current months for all organizations (data available as of December 1, 2020).
	RehydratedIngestedBytesAggSum *int64 `json:"rehydrated_ingested_bytes_agg_sum,omitempty"`
	// Shows the sum of all mobile sessions and all browser lite and legacy sessions over all hours in the current month for all organizations.
	RumBrowserAndMobileSessionCount *int64 `json:"rum_browser_and_mobile_session_count,omitempty"`
	// Shows the sum of all browser RUM Lite Sessions over all hours in the current months for all organizations.
	RumSessionCountAggSum *int64 `json:"rum_session_count_agg_sum,omitempty"`
	// Shows the sum of RUM Sessions (browser and mobile) over all hours in the current months for all organizations.
	RumTotalSessionCountAggSum *int64 `json:"rum_total_session_count_agg_sum,omitempty"`
	// Shows the sum of all browser and mobile RUM units over all hours in the current months for all organizations.
	RumUnitsAggSum *int64 `json:"rum_units_agg_sum,omitempty"`
	// Sum of all APM bytes scanned with sensitive data scanner in the current months for all organizations.
	SdsApmScannedBytesSum *int64 `json:"sds_apm_scanned_bytes_sum,omitempty"`
	// Sum of all event stream events bytes scanned with sensitive data scanner in the current months for all organizations.
	SdsEventsScannedBytesSum *int64 `json:"sds_events_scanned_bytes_sum,omitempty"`
	// Shows the sum of all bytes scanned of logs usage by the Sensitive Data Scanner over all hours in the current month for all organizations.
	SdsLogsScannedBytesSum *int64 `json:"sds_logs_scanned_bytes_sum,omitempty"`
	// Sum of all RUM bytes scanned with sensitive data scanner in the current months for all organizations.
	SdsRumScannedBytesSum *int64 `json:"sds_rum_scanned_bytes_sum,omitempty"`
	// Shows the sum of all bytes scanned across all usage types by the Sensitive Data Scanner over all hours in the current month for all organizations.
	SdsTotalScannedBytesSum *int64 `json:"sds_total_scanned_bytes_sum,omitempty"`
	// Shows the first date of usage in the current months for all organizations.
	StartDate *time.Time `json:"start_date,omitempty"`
	// Shows the sum of all Synthetic browser tests over all hours in the current months for all organizations.
	SyntheticsBrowserCheckCallsCountAggSum *int64 `json:"synthetics_browser_check_calls_count_agg_sum,omitempty"`
	// Shows the sum of all Synthetic API tests over all hours in the current months for all organizations.
	SyntheticsCheckCallsCountAggSum *int64 `json:"synthetics_check_calls_count_agg_sum,omitempty"`
	// Shows the sum of the high-water marks of used synthetics parallel testing slots over all hours in the current month for all organizations.
	SyntheticsParallelTestingMaxSlotsHwmSum *int64 `json:"synthetics_parallel_testing_max_slots_hwm_sum,omitempty"`
	// Shows the sum of all Indexed Spans indexed over all hours in the current months for all organizations.
	TraceSearchIndexedEventsCountAggSum *int64 `json:"trace_search_indexed_events_count_agg_sum,omitempty"`
	// Shows the sum of all ingested APM span bytes over all hours in the current months for all organizations.
	TwolIngestedEventsBytesAggSum *int64 `json:"twol_ingested_events_bytes_agg_sum,omitempty"`
	// An array of objects regarding hourly usage.
	Usage []UsageSummaryDate `json:"usage,omitempty"`
	// Shows the 99th percentile of all vSphere hosts over all hours in the current months for all organizations.
	VsphereHostTop99pSum *int64 `json:"vsphere_host_top99p_sum,omitempty"`
	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject       map[string]interface{} `json:"-"`
	AdditionalProperties map[string]interface{}
}

// NewUsageSummaryResponse instantiates a new UsageSummaryResponse object.
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed.
func NewUsageSummaryResponse() *UsageSummaryResponse {
	this := UsageSummaryResponse{}
	return &this
}

// NewUsageSummaryResponseWithDefaults instantiates a new UsageSummaryResponse object.
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set.
func NewUsageSummaryResponseWithDefaults() *UsageSummaryResponse {
	this := UsageSummaryResponse{}
	return &this
}

// GetAgentHostTop99pSum returns the AgentHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetAgentHostTop99pSum() int64 {
	if o == nil || o.AgentHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.AgentHostTop99pSum
}

// GetAgentHostTop99pSumOk returns a tuple with the AgentHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetAgentHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.AgentHostTop99pSum == nil {
		return nil, false
	}
	return o.AgentHostTop99pSum, true
}

// HasAgentHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasAgentHostTop99pSum() bool {
	return o != nil && o.AgentHostTop99pSum != nil
}

// SetAgentHostTop99pSum gets a reference to the given int64 and assigns it to the AgentHostTop99pSum field.
func (o *UsageSummaryResponse) SetAgentHostTop99pSum(v int64) {
	o.AgentHostTop99pSum = &v
}

// GetApmAzureAppServiceHostTop99pSum returns the ApmAzureAppServiceHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetApmAzureAppServiceHostTop99pSum() int64 {
	if o == nil || o.ApmAzureAppServiceHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.ApmAzureAppServiceHostTop99pSum
}

// GetApmAzureAppServiceHostTop99pSumOk returns a tuple with the ApmAzureAppServiceHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetApmAzureAppServiceHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.ApmAzureAppServiceHostTop99pSum == nil {
		return nil, false
	}
	return o.ApmAzureAppServiceHostTop99pSum, true
}

// HasApmAzureAppServiceHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasApmAzureAppServiceHostTop99pSum() bool {
	return o != nil && o.ApmAzureAppServiceHostTop99pSum != nil
}

// SetApmAzureAppServiceHostTop99pSum gets a reference to the given int64 and assigns it to the ApmAzureAppServiceHostTop99pSum field.
func (o *UsageSummaryResponse) SetApmAzureAppServiceHostTop99pSum(v int64) {
	o.ApmAzureAppServiceHostTop99pSum = &v
}

// GetApmFargateCountAvgSum returns the ApmFargateCountAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetApmFargateCountAvgSum() int64 {
	if o == nil || o.ApmFargateCountAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.ApmFargateCountAvgSum
}

// GetApmFargateCountAvgSumOk returns a tuple with the ApmFargateCountAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetApmFargateCountAvgSumOk() (*int64, bool) {
	if o == nil || o.ApmFargateCountAvgSum == nil {
		return nil, false
	}
	return o.ApmFargateCountAvgSum, true
}

// HasApmFargateCountAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasApmFargateCountAvgSum() bool {
	return o != nil && o.ApmFargateCountAvgSum != nil
}

// SetApmFargateCountAvgSum gets a reference to the given int64 and assigns it to the ApmFargateCountAvgSum field.
func (o *UsageSummaryResponse) SetApmFargateCountAvgSum(v int64) {
	o.ApmFargateCountAvgSum = &v
}

// GetApmHostTop99pSum returns the ApmHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetApmHostTop99pSum() int64 {
	if o == nil || o.ApmHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.ApmHostTop99pSum
}

// GetApmHostTop99pSumOk returns a tuple with the ApmHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetApmHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.ApmHostTop99pSum == nil {
		return nil, false
	}
	return o.ApmHostTop99pSum, true
}

// HasApmHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasApmHostTop99pSum() bool {
	return o != nil && o.ApmHostTop99pSum != nil
}

// SetApmHostTop99pSum gets a reference to the given int64 and assigns it to the ApmHostTop99pSum field.
func (o *UsageSummaryResponse) SetApmHostTop99pSum(v int64) {
	o.ApmHostTop99pSum = &v
}

// GetAppsecFargateCountAvgSum returns the AppsecFargateCountAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetAppsecFargateCountAvgSum() int64 {
	if o == nil || o.AppsecFargateCountAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.AppsecFargateCountAvgSum
}

// GetAppsecFargateCountAvgSumOk returns a tuple with the AppsecFargateCountAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetAppsecFargateCountAvgSumOk() (*int64, bool) {
	if o == nil || o.AppsecFargateCountAvgSum == nil {
		return nil, false
	}
	return o.AppsecFargateCountAvgSum, true
}

// HasAppsecFargateCountAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasAppsecFargateCountAvgSum() bool {
	return o != nil && o.AppsecFargateCountAvgSum != nil
}

// SetAppsecFargateCountAvgSum gets a reference to the given int64 and assigns it to the AppsecFargateCountAvgSum field.
func (o *UsageSummaryResponse) SetAppsecFargateCountAvgSum(v int64) {
	o.AppsecFargateCountAvgSum = &v
}

// GetAuditLogsLinesIndexedAggSum returns the AuditLogsLinesIndexedAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetAuditLogsLinesIndexedAggSum() int64 {
	if o == nil || o.AuditLogsLinesIndexedAggSum == nil {
		var ret int64
		return ret
	}
	return *o.AuditLogsLinesIndexedAggSum
}

// GetAuditLogsLinesIndexedAggSumOk returns a tuple with the AuditLogsLinesIndexedAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetAuditLogsLinesIndexedAggSumOk() (*int64, bool) {
	if o == nil || o.AuditLogsLinesIndexedAggSum == nil {
		return nil, false
	}
	return o.AuditLogsLinesIndexedAggSum, true
}

// HasAuditLogsLinesIndexedAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasAuditLogsLinesIndexedAggSum() bool {
	return o != nil && o.AuditLogsLinesIndexedAggSum != nil
}

// SetAuditLogsLinesIndexedAggSum gets a reference to the given int64 and assigns it to the AuditLogsLinesIndexedAggSum field.
func (o *UsageSummaryResponse) SetAuditLogsLinesIndexedAggSum(v int64) {
	o.AuditLogsLinesIndexedAggSum = &v
}

// GetAvgProfiledFargateTasksSum returns the AvgProfiledFargateTasksSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetAvgProfiledFargateTasksSum() int64 {
	if o == nil || o.AvgProfiledFargateTasksSum == nil {
		var ret int64
		return ret
	}
	return *o.AvgProfiledFargateTasksSum
}

// GetAvgProfiledFargateTasksSumOk returns a tuple with the AvgProfiledFargateTasksSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetAvgProfiledFargateTasksSumOk() (*int64, bool) {
	if o == nil || o.AvgProfiledFargateTasksSum == nil {
		return nil, false
	}
	return o.AvgProfiledFargateTasksSum, true
}

// HasAvgProfiledFargateTasksSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasAvgProfiledFargateTasksSum() bool {
	return o != nil && o.AvgProfiledFargateTasksSum != nil
}

// SetAvgProfiledFargateTasksSum gets a reference to the given int64 and assigns it to the AvgProfiledFargateTasksSum field.
func (o *UsageSummaryResponse) SetAvgProfiledFargateTasksSum(v int64) {
	o.AvgProfiledFargateTasksSum = &v
}

// GetAwsHostTop99pSum returns the AwsHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetAwsHostTop99pSum() int64 {
	if o == nil || o.AwsHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.AwsHostTop99pSum
}

// GetAwsHostTop99pSumOk returns a tuple with the AwsHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetAwsHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.AwsHostTop99pSum == nil {
		return nil, false
	}
	return o.AwsHostTop99pSum, true
}

// HasAwsHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasAwsHostTop99pSum() bool {
	return o != nil && o.AwsHostTop99pSum != nil
}

// SetAwsHostTop99pSum gets a reference to the given int64 and assigns it to the AwsHostTop99pSum field.
func (o *UsageSummaryResponse) SetAwsHostTop99pSum(v int64) {
	o.AwsHostTop99pSum = &v
}

// GetAwsLambdaFuncCount returns the AwsLambdaFuncCount field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetAwsLambdaFuncCount() int64 {
	if o == nil || o.AwsLambdaFuncCount == nil {
		var ret int64
		return ret
	}
	return *o.AwsLambdaFuncCount
}

// GetAwsLambdaFuncCountOk returns a tuple with the AwsLambdaFuncCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetAwsLambdaFuncCountOk() (*int64, bool) {
	if o == nil || o.AwsLambdaFuncCount == nil {
		return nil, false
	}
	return o.AwsLambdaFuncCount, true
}

// HasAwsLambdaFuncCount returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasAwsLambdaFuncCount() bool {
	return o != nil && o.AwsLambdaFuncCount != nil
}

// SetAwsLambdaFuncCount gets a reference to the given int64 and assigns it to the AwsLambdaFuncCount field.
func (o *UsageSummaryResponse) SetAwsLambdaFuncCount(v int64) {
	o.AwsLambdaFuncCount = &v
}

// GetAwsLambdaInvocationsSum returns the AwsLambdaInvocationsSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetAwsLambdaInvocationsSum() int64 {
	if o == nil || o.AwsLambdaInvocationsSum == nil {
		var ret int64
		return ret
	}
	return *o.AwsLambdaInvocationsSum
}

// GetAwsLambdaInvocationsSumOk returns a tuple with the AwsLambdaInvocationsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetAwsLambdaInvocationsSumOk() (*int64, bool) {
	if o == nil || o.AwsLambdaInvocationsSum == nil {
		return nil, false
	}
	return o.AwsLambdaInvocationsSum, true
}

// HasAwsLambdaInvocationsSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasAwsLambdaInvocationsSum() bool {
	return o != nil && o.AwsLambdaInvocationsSum != nil
}

// SetAwsLambdaInvocationsSum gets a reference to the given int64 and assigns it to the AwsLambdaInvocationsSum field.
func (o *UsageSummaryResponse) SetAwsLambdaInvocationsSum(v int64) {
	o.AwsLambdaInvocationsSum = &v
}

// GetAzureAppServiceTop99pSum returns the AzureAppServiceTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetAzureAppServiceTop99pSum() int64 {
	if o == nil || o.AzureAppServiceTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.AzureAppServiceTop99pSum
}

// GetAzureAppServiceTop99pSumOk returns a tuple with the AzureAppServiceTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetAzureAppServiceTop99pSumOk() (*int64, bool) {
	if o == nil || o.AzureAppServiceTop99pSum == nil {
		return nil, false
	}
	return o.AzureAppServiceTop99pSum, true
}

// HasAzureAppServiceTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasAzureAppServiceTop99pSum() bool {
	return o != nil && o.AzureAppServiceTop99pSum != nil
}

// SetAzureAppServiceTop99pSum gets a reference to the given int64 and assigns it to the AzureAppServiceTop99pSum field.
func (o *UsageSummaryResponse) SetAzureAppServiceTop99pSum(v int64) {
	o.AzureAppServiceTop99pSum = &v
}

// GetAzureHostTop99pSum returns the AzureHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetAzureHostTop99pSum() int64 {
	if o == nil || o.AzureHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.AzureHostTop99pSum
}

// GetAzureHostTop99pSumOk returns a tuple with the AzureHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetAzureHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.AzureHostTop99pSum == nil {
		return nil, false
	}
	return o.AzureHostTop99pSum, true
}

// HasAzureHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasAzureHostTop99pSum() bool {
	return o != nil && o.AzureHostTop99pSum != nil
}

// SetAzureHostTop99pSum gets a reference to the given int64 and assigns it to the AzureHostTop99pSum field.
func (o *UsageSummaryResponse) SetAzureHostTop99pSum(v int64) {
	o.AzureHostTop99pSum = &v
}

// GetBillableIngestedBytesAggSum returns the BillableIngestedBytesAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetBillableIngestedBytesAggSum() int64 {
	if o == nil || o.BillableIngestedBytesAggSum == nil {
		var ret int64
		return ret
	}
	return *o.BillableIngestedBytesAggSum
}

// GetBillableIngestedBytesAggSumOk returns a tuple with the BillableIngestedBytesAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetBillableIngestedBytesAggSumOk() (*int64, bool) {
	if o == nil || o.BillableIngestedBytesAggSum == nil {
		return nil, false
	}
	return o.BillableIngestedBytesAggSum, true
}

// HasBillableIngestedBytesAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasBillableIngestedBytesAggSum() bool {
	return o != nil && o.BillableIngestedBytesAggSum != nil
}

// SetBillableIngestedBytesAggSum gets a reference to the given int64 and assigns it to the BillableIngestedBytesAggSum field.
func (o *UsageSummaryResponse) SetBillableIngestedBytesAggSum(v int64) {
	o.BillableIngestedBytesAggSum = &v
}

// GetBrowserRumLiteSessionCountAggSum returns the BrowserRumLiteSessionCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetBrowserRumLiteSessionCountAggSum() int64 {
	if o == nil || o.BrowserRumLiteSessionCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.BrowserRumLiteSessionCountAggSum
}

// GetBrowserRumLiteSessionCountAggSumOk returns a tuple with the BrowserRumLiteSessionCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetBrowserRumLiteSessionCountAggSumOk() (*int64, bool) {
	if o == nil || o.BrowserRumLiteSessionCountAggSum == nil {
		return nil, false
	}
	return o.BrowserRumLiteSessionCountAggSum, true
}

// HasBrowserRumLiteSessionCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasBrowserRumLiteSessionCountAggSum() bool {
	return o != nil && o.BrowserRumLiteSessionCountAggSum != nil
}

// SetBrowserRumLiteSessionCountAggSum gets a reference to the given int64 and assigns it to the BrowserRumLiteSessionCountAggSum field.
func (o *UsageSummaryResponse) SetBrowserRumLiteSessionCountAggSum(v int64) {
	o.BrowserRumLiteSessionCountAggSum = &v
}

// GetBrowserRumReplaySessionCountAggSum returns the BrowserRumReplaySessionCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetBrowserRumReplaySessionCountAggSum() int64 {
	if o == nil || o.BrowserRumReplaySessionCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.BrowserRumReplaySessionCountAggSum
}

// GetBrowserRumReplaySessionCountAggSumOk returns a tuple with the BrowserRumReplaySessionCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetBrowserRumReplaySessionCountAggSumOk() (*int64, bool) {
	if o == nil || o.BrowserRumReplaySessionCountAggSum == nil {
		return nil, false
	}
	return o.BrowserRumReplaySessionCountAggSum, true
}

// HasBrowserRumReplaySessionCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasBrowserRumReplaySessionCountAggSum() bool {
	return o != nil && o.BrowserRumReplaySessionCountAggSum != nil
}

// SetBrowserRumReplaySessionCountAggSum gets a reference to the given int64 and assigns it to the BrowserRumReplaySessionCountAggSum field.
func (o *UsageSummaryResponse) SetBrowserRumReplaySessionCountAggSum(v int64) {
	o.BrowserRumReplaySessionCountAggSum = &v
}

// GetBrowserRumUnitsAggSum returns the BrowserRumUnitsAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetBrowserRumUnitsAggSum() int64 {
	if o == nil || o.BrowserRumUnitsAggSum == nil {
		var ret int64
		return ret
	}
	return *o.BrowserRumUnitsAggSum
}

// GetBrowserRumUnitsAggSumOk returns a tuple with the BrowserRumUnitsAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetBrowserRumUnitsAggSumOk() (*int64, bool) {
	if o == nil || o.BrowserRumUnitsAggSum == nil {
		return nil, false
	}
	return o.BrowserRumUnitsAggSum, true
}

// HasBrowserRumUnitsAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasBrowserRumUnitsAggSum() bool {
	return o != nil && o.BrowserRumUnitsAggSum != nil
}

// SetBrowserRumUnitsAggSum gets a reference to the given int64 and assigns it to the BrowserRumUnitsAggSum field.
func (o *UsageSummaryResponse) SetBrowserRumUnitsAggSum(v int64) {
	o.BrowserRumUnitsAggSum = &v
}

// GetCiPipelineIndexedSpansAggSum returns the CiPipelineIndexedSpansAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCiPipelineIndexedSpansAggSum() int64 {
	if o == nil || o.CiPipelineIndexedSpansAggSum == nil {
		var ret int64
		return ret
	}
	return *o.CiPipelineIndexedSpansAggSum
}

// GetCiPipelineIndexedSpansAggSumOk returns a tuple with the CiPipelineIndexedSpansAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCiPipelineIndexedSpansAggSumOk() (*int64, bool) {
	if o == nil || o.CiPipelineIndexedSpansAggSum == nil {
		return nil, false
	}
	return o.CiPipelineIndexedSpansAggSum, true
}

// HasCiPipelineIndexedSpansAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCiPipelineIndexedSpansAggSum() bool {
	return o != nil && o.CiPipelineIndexedSpansAggSum != nil
}

// SetCiPipelineIndexedSpansAggSum gets a reference to the given int64 and assigns it to the CiPipelineIndexedSpansAggSum field.
func (o *UsageSummaryResponse) SetCiPipelineIndexedSpansAggSum(v int64) {
	o.CiPipelineIndexedSpansAggSum = &v
}

// GetCiTestIndexedSpansAggSum returns the CiTestIndexedSpansAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCiTestIndexedSpansAggSum() int64 {
	if o == nil || o.CiTestIndexedSpansAggSum == nil {
		var ret int64
		return ret
	}
	return *o.CiTestIndexedSpansAggSum
}

// GetCiTestIndexedSpansAggSumOk returns a tuple with the CiTestIndexedSpansAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCiTestIndexedSpansAggSumOk() (*int64, bool) {
	if o == nil || o.CiTestIndexedSpansAggSum == nil {
		return nil, false
	}
	return o.CiTestIndexedSpansAggSum, true
}

// HasCiTestIndexedSpansAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCiTestIndexedSpansAggSum() bool {
	return o != nil && o.CiTestIndexedSpansAggSum != nil
}

// SetCiTestIndexedSpansAggSum gets a reference to the given int64 and assigns it to the CiTestIndexedSpansAggSum field.
func (o *UsageSummaryResponse) SetCiTestIndexedSpansAggSum(v int64) {
	o.CiTestIndexedSpansAggSum = &v
}

// GetCiVisibilityPipelineCommittersHwmSum returns the CiVisibilityPipelineCommittersHwmSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCiVisibilityPipelineCommittersHwmSum() int64 {
	if o == nil || o.CiVisibilityPipelineCommittersHwmSum == nil {
		var ret int64
		return ret
	}
	return *o.CiVisibilityPipelineCommittersHwmSum
}

// GetCiVisibilityPipelineCommittersHwmSumOk returns a tuple with the CiVisibilityPipelineCommittersHwmSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCiVisibilityPipelineCommittersHwmSumOk() (*int64, bool) {
	if o == nil || o.CiVisibilityPipelineCommittersHwmSum == nil {
		return nil, false
	}
	return o.CiVisibilityPipelineCommittersHwmSum, true
}

// HasCiVisibilityPipelineCommittersHwmSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCiVisibilityPipelineCommittersHwmSum() bool {
	return o != nil && o.CiVisibilityPipelineCommittersHwmSum != nil
}

// SetCiVisibilityPipelineCommittersHwmSum gets a reference to the given int64 and assigns it to the CiVisibilityPipelineCommittersHwmSum field.
func (o *UsageSummaryResponse) SetCiVisibilityPipelineCommittersHwmSum(v int64) {
	o.CiVisibilityPipelineCommittersHwmSum = &v
}

// GetCiVisibilityTestCommittersHwmSum returns the CiVisibilityTestCommittersHwmSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCiVisibilityTestCommittersHwmSum() int64 {
	if o == nil || o.CiVisibilityTestCommittersHwmSum == nil {
		var ret int64
		return ret
	}
	return *o.CiVisibilityTestCommittersHwmSum
}

// GetCiVisibilityTestCommittersHwmSumOk returns a tuple with the CiVisibilityTestCommittersHwmSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCiVisibilityTestCommittersHwmSumOk() (*int64, bool) {
	if o == nil || o.CiVisibilityTestCommittersHwmSum == nil {
		return nil, false
	}
	return o.CiVisibilityTestCommittersHwmSum, true
}

// HasCiVisibilityTestCommittersHwmSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCiVisibilityTestCommittersHwmSum() bool {
	return o != nil && o.CiVisibilityTestCommittersHwmSum != nil
}

// SetCiVisibilityTestCommittersHwmSum gets a reference to the given int64 and assigns it to the CiVisibilityTestCommittersHwmSum field.
func (o *UsageSummaryResponse) SetCiVisibilityTestCommittersHwmSum(v int64) {
	o.CiVisibilityTestCommittersHwmSum = &v
}

// GetCloudCostManagementHostCountAvgSum returns the CloudCostManagementHostCountAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCloudCostManagementHostCountAvgSum() int64 {
	if o == nil || o.CloudCostManagementHostCountAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.CloudCostManagementHostCountAvgSum
}

// GetCloudCostManagementHostCountAvgSumOk returns a tuple with the CloudCostManagementHostCountAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCloudCostManagementHostCountAvgSumOk() (*int64, bool) {
	if o == nil || o.CloudCostManagementHostCountAvgSum == nil {
		return nil, false
	}
	return o.CloudCostManagementHostCountAvgSum, true
}

// HasCloudCostManagementHostCountAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCloudCostManagementHostCountAvgSum() bool {
	return o != nil && o.CloudCostManagementHostCountAvgSum != nil
}

// SetCloudCostManagementHostCountAvgSum gets a reference to the given int64 and assigns it to the CloudCostManagementHostCountAvgSum field.
func (o *UsageSummaryResponse) SetCloudCostManagementHostCountAvgSum(v int64) {
	o.CloudCostManagementHostCountAvgSum = &v
}

// GetContainerAvgSum returns the ContainerAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetContainerAvgSum() int64 {
	if o == nil || o.ContainerAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.ContainerAvgSum
}

// GetContainerAvgSumOk returns a tuple with the ContainerAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetContainerAvgSumOk() (*int64, bool) {
	if o == nil || o.ContainerAvgSum == nil {
		return nil, false
	}
	return o.ContainerAvgSum, true
}

// HasContainerAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasContainerAvgSum() bool {
	return o != nil && o.ContainerAvgSum != nil
}

// SetContainerAvgSum gets a reference to the given int64 and assigns it to the ContainerAvgSum field.
func (o *UsageSummaryResponse) SetContainerAvgSum(v int64) {
	o.ContainerAvgSum = &v
}

// GetContainerExclAgentAvgSum returns the ContainerExclAgentAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetContainerExclAgentAvgSum() int64 {
	if o == nil || o.ContainerExclAgentAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.ContainerExclAgentAvgSum
}

// GetContainerExclAgentAvgSumOk returns a tuple with the ContainerExclAgentAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetContainerExclAgentAvgSumOk() (*int64, bool) {
	if o == nil || o.ContainerExclAgentAvgSum == nil {
		return nil, false
	}
	return o.ContainerExclAgentAvgSum, true
}

// HasContainerExclAgentAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasContainerExclAgentAvgSum() bool {
	return o != nil && o.ContainerExclAgentAvgSum != nil
}

// SetContainerExclAgentAvgSum gets a reference to the given int64 and assigns it to the ContainerExclAgentAvgSum field.
func (o *UsageSummaryResponse) SetContainerExclAgentAvgSum(v int64) {
	o.ContainerExclAgentAvgSum = &v
}

// GetContainerHwmSum returns the ContainerHwmSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetContainerHwmSum() int64 {
	if o == nil || o.ContainerHwmSum == nil {
		var ret int64
		return ret
	}
	return *o.ContainerHwmSum
}

// GetContainerHwmSumOk returns a tuple with the ContainerHwmSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetContainerHwmSumOk() (*int64, bool) {
	if o == nil || o.ContainerHwmSum == nil {
		return nil, false
	}
	return o.ContainerHwmSum, true
}

// HasContainerHwmSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasContainerHwmSum() bool {
	return o != nil && o.ContainerHwmSum != nil
}

// SetContainerHwmSum gets a reference to the given int64 and assigns it to the ContainerHwmSum field.
func (o *UsageSummaryResponse) SetContainerHwmSum(v int64) {
	o.ContainerHwmSum = &v
}

// GetCspmAasHostTop99pSum returns the CspmAasHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCspmAasHostTop99pSum() int64 {
	if o == nil || o.CspmAasHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.CspmAasHostTop99pSum
}

// GetCspmAasHostTop99pSumOk returns a tuple with the CspmAasHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCspmAasHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.CspmAasHostTop99pSum == nil {
		return nil, false
	}
	return o.CspmAasHostTop99pSum, true
}

// HasCspmAasHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCspmAasHostTop99pSum() bool {
	return o != nil && o.CspmAasHostTop99pSum != nil
}

// SetCspmAasHostTop99pSum gets a reference to the given int64 and assigns it to the CspmAasHostTop99pSum field.
func (o *UsageSummaryResponse) SetCspmAasHostTop99pSum(v int64) {
	o.CspmAasHostTop99pSum = &v
}

// GetCspmAwsHostTop99pSum returns the CspmAwsHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCspmAwsHostTop99pSum() int64 {
	if o == nil || o.CspmAwsHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.CspmAwsHostTop99pSum
}

// GetCspmAwsHostTop99pSumOk returns a tuple with the CspmAwsHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCspmAwsHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.CspmAwsHostTop99pSum == nil {
		return nil, false
	}
	return o.CspmAwsHostTop99pSum, true
}

// HasCspmAwsHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCspmAwsHostTop99pSum() bool {
	return o != nil && o.CspmAwsHostTop99pSum != nil
}

// SetCspmAwsHostTop99pSum gets a reference to the given int64 and assigns it to the CspmAwsHostTop99pSum field.
func (o *UsageSummaryResponse) SetCspmAwsHostTop99pSum(v int64) {
	o.CspmAwsHostTop99pSum = &v
}

// GetCspmAzureHostTop99pSum returns the CspmAzureHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCspmAzureHostTop99pSum() int64 {
	if o == nil || o.CspmAzureHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.CspmAzureHostTop99pSum
}

// GetCspmAzureHostTop99pSumOk returns a tuple with the CspmAzureHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCspmAzureHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.CspmAzureHostTop99pSum == nil {
		return nil, false
	}
	return o.CspmAzureHostTop99pSum, true
}

// HasCspmAzureHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCspmAzureHostTop99pSum() bool {
	return o != nil && o.CspmAzureHostTop99pSum != nil
}

// SetCspmAzureHostTop99pSum gets a reference to the given int64 and assigns it to the CspmAzureHostTop99pSum field.
func (o *UsageSummaryResponse) SetCspmAzureHostTop99pSum(v int64) {
	o.CspmAzureHostTop99pSum = &v
}

// GetCspmContainerAvgSum returns the CspmContainerAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCspmContainerAvgSum() int64 {
	if o == nil || o.CspmContainerAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.CspmContainerAvgSum
}

// GetCspmContainerAvgSumOk returns a tuple with the CspmContainerAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCspmContainerAvgSumOk() (*int64, bool) {
	if o == nil || o.CspmContainerAvgSum == nil {
		return nil, false
	}
	return o.CspmContainerAvgSum, true
}

// HasCspmContainerAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCspmContainerAvgSum() bool {
	return o != nil && o.CspmContainerAvgSum != nil
}

// SetCspmContainerAvgSum gets a reference to the given int64 and assigns it to the CspmContainerAvgSum field.
func (o *UsageSummaryResponse) SetCspmContainerAvgSum(v int64) {
	o.CspmContainerAvgSum = &v
}

// GetCspmContainerHwmSum returns the CspmContainerHwmSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCspmContainerHwmSum() int64 {
	if o == nil || o.CspmContainerHwmSum == nil {
		var ret int64
		return ret
	}
	return *o.CspmContainerHwmSum
}

// GetCspmContainerHwmSumOk returns a tuple with the CspmContainerHwmSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCspmContainerHwmSumOk() (*int64, bool) {
	if o == nil || o.CspmContainerHwmSum == nil {
		return nil, false
	}
	return o.CspmContainerHwmSum, true
}

// HasCspmContainerHwmSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCspmContainerHwmSum() bool {
	return o != nil && o.CspmContainerHwmSum != nil
}

// SetCspmContainerHwmSum gets a reference to the given int64 and assigns it to the CspmContainerHwmSum field.
func (o *UsageSummaryResponse) SetCspmContainerHwmSum(v int64) {
	o.CspmContainerHwmSum = &v
}

// GetCspmGcpHostTop99pSum returns the CspmGcpHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCspmGcpHostTop99pSum() int64 {
	if o == nil || o.CspmGcpHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.CspmGcpHostTop99pSum
}

// GetCspmGcpHostTop99pSumOk returns a tuple with the CspmGcpHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCspmGcpHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.CspmGcpHostTop99pSum == nil {
		return nil, false
	}
	return o.CspmGcpHostTop99pSum, true
}

// HasCspmGcpHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCspmGcpHostTop99pSum() bool {
	return o != nil && o.CspmGcpHostTop99pSum != nil
}

// SetCspmGcpHostTop99pSum gets a reference to the given int64 and assigns it to the CspmGcpHostTop99pSum field.
func (o *UsageSummaryResponse) SetCspmGcpHostTop99pSum(v int64) {
	o.CspmGcpHostTop99pSum = &v
}

// GetCspmHostTop99pSum returns the CspmHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCspmHostTop99pSum() int64 {
	if o == nil || o.CspmHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.CspmHostTop99pSum
}

// GetCspmHostTop99pSumOk returns a tuple with the CspmHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCspmHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.CspmHostTop99pSum == nil {
		return nil, false
	}
	return o.CspmHostTop99pSum, true
}

// HasCspmHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCspmHostTop99pSum() bool {
	return o != nil && o.CspmHostTop99pSum != nil
}

// SetCspmHostTop99pSum gets a reference to the given int64 and assigns it to the CspmHostTop99pSum field.
func (o *UsageSummaryResponse) SetCspmHostTop99pSum(v int64) {
	o.CspmHostTop99pSum = &v
}

// GetCustomTsSum returns the CustomTsSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCustomTsSum() int64 {
	if o == nil || o.CustomTsSum == nil {
		var ret int64
		return ret
	}
	return *o.CustomTsSum
}

// GetCustomTsSumOk returns a tuple with the CustomTsSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCustomTsSumOk() (*int64, bool) {
	if o == nil || o.CustomTsSum == nil {
		return nil, false
	}
	return o.CustomTsSum, true
}

// HasCustomTsSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCustomTsSum() bool {
	return o != nil && o.CustomTsSum != nil
}

// SetCustomTsSum gets a reference to the given int64 and assigns it to the CustomTsSum field.
func (o *UsageSummaryResponse) SetCustomTsSum(v int64) {
	o.CustomTsSum = &v
}

// GetCwsContainersAvgSum returns the CwsContainersAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCwsContainersAvgSum() int64 {
	if o == nil || o.CwsContainersAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.CwsContainersAvgSum
}

// GetCwsContainersAvgSumOk returns a tuple with the CwsContainersAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCwsContainersAvgSumOk() (*int64, bool) {
	if o == nil || o.CwsContainersAvgSum == nil {
		return nil, false
	}
	return o.CwsContainersAvgSum, true
}

// HasCwsContainersAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCwsContainersAvgSum() bool {
	return o != nil && o.CwsContainersAvgSum != nil
}

// SetCwsContainersAvgSum gets a reference to the given int64 and assigns it to the CwsContainersAvgSum field.
func (o *UsageSummaryResponse) SetCwsContainersAvgSum(v int64) {
	o.CwsContainersAvgSum = &v
}

// GetCwsHostTop99pSum returns the CwsHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetCwsHostTop99pSum() int64 {
	if o == nil || o.CwsHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.CwsHostTop99pSum
}

// GetCwsHostTop99pSumOk returns a tuple with the CwsHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetCwsHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.CwsHostTop99pSum == nil {
		return nil, false
	}
	return o.CwsHostTop99pSum, true
}

// HasCwsHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasCwsHostTop99pSum() bool {
	return o != nil && o.CwsHostTop99pSum != nil
}

// SetCwsHostTop99pSum gets a reference to the given int64 and assigns it to the CwsHostTop99pSum field.
func (o *UsageSummaryResponse) SetCwsHostTop99pSum(v int64) {
	o.CwsHostTop99pSum = &v
}

// GetDbmHostTop99pSum returns the DbmHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetDbmHostTop99pSum() int64 {
	if o == nil || o.DbmHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.DbmHostTop99pSum
}

// GetDbmHostTop99pSumOk returns a tuple with the DbmHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetDbmHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.DbmHostTop99pSum == nil {
		return nil, false
	}
	return o.DbmHostTop99pSum, true
}

// HasDbmHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasDbmHostTop99pSum() bool {
	return o != nil && o.DbmHostTop99pSum != nil
}

// SetDbmHostTop99pSum gets a reference to the given int64 and assigns it to the DbmHostTop99pSum field.
func (o *UsageSummaryResponse) SetDbmHostTop99pSum(v int64) {
	o.DbmHostTop99pSum = &v
}

// GetDbmQueriesAvgSum returns the DbmQueriesAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetDbmQueriesAvgSum() int64 {
	if o == nil || o.DbmQueriesAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.DbmQueriesAvgSum
}

// GetDbmQueriesAvgSumOk returns a tuple with the DbmQueriesAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetDbmQueriesAvgSumOk() (*int64, bool) {
	if o == nil || o.DbmQueriesAvgSum == nil {
		return nil, false
	}
	return o.DbmQueriesAvgSum, true
}

// HasDbmQueriesAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasDbmQueriesAvgSum() bool {
	return o != nil && o.DbmQueriesAvgSum != nil
}

// SetDbmQueriesAvgSum gets a reference to the given int64 and assigns it to the DbmQueriesAvgSum field.
func (o *UsageSummaryResponse) SetDbmQueriesAvgSum(v int64) {
	o.DbmQueriesAvgSum = &v
}

// GetEndDate returns the EndDate field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetEndDate() time.Time {
	if o == nil || o.EndDate == nil {
		var ret time.Time
		return ret
	}
	return *o.EndDate
}

// GetEndDateOk returns a tuple with the EndDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetEndDateOk() (*time.Time, bool) {
	if o == nil || o.EndDate == nil {
		return nil, false
	}
	return o.EndDate, true
}

// HasEndDate returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasEndDate() bool {
	return o != nil && o.EndDate != nil
}

// SetEndDate gets a reference to the given time.Time and assigns it to the EndDate field.
func (o *UsageSummaryResponse) SetEndDate(v time.Time) {
	o.EndDate = &v
}

// GetFargateTasksCountAvgSum returns the FargateTasksCountAvgSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetFargateTasksCountAvgSum() int64 {
	if o == nil || o.FargateTasksCountAvgSum == nil {
		var ret int64
		return ret
	}
	return *o.FargateTasksCountAvgSum
}

// GetFargateTasksCountAvgSumOk returns a tuple with the FargateTasksCountAvgSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetFargateTasksCountAvgSumOk() (*int64, bool) {
	if o == nil || o.FargateTasksCountAvgSum == nil {
		return nil, false
	}
	return o.FargateTasksCountAvgSum, true
}

// HasFargateTasksCountAvgSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasFargateTasksCountAvgSum() bool {
	return o != nil && o.FargateTasksCountAvgSum != nil
}

// SetFargateTasksCountAvgSum gets a reference to the given int64 and assigns it to the FargateTasksCountAvgSum field.
func (o *UsageSummaryResponse) SetFargateTasksCountAvgSum(v int64) {
	o.FargateTasksCountAvgSum = &v
}

// GetFargateTasksCountHwmSum returns the FargateTasksCountHwmSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetFargateTasksCountHwmSum() int64 {
	if o == nil || o.FargateTasksCountHwmSum == nil {
		var ret int64
		return ret
	}
	return *o.FargateTasksCountHwmSum
}

// GetFargateTasksCountHwmSumOk returns a tuple with the FargateTasksCountHwmSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetFargateTasksCountHwmSumOk() (*int64, bool) {
	if o == nil || o.FargateTasksCountHwmSum == nil {
		return nil, false
	}
	return o.FargateTasksCountHwmSum, true
}

// HasFargateTasksCountHwmSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasFargateTasksCountHwmSum() bool {
	return o != nil && o.FargateTasksCountHwmSum != nil
}

// SetFargateTasksCountHwmSum gets a reference to the given int64 and assigns it to the FargateTasksCountHwmSum field.
func (o *UsageSummaryResponse) SetFargateTasksCountHwmSum(v int64) {
	o.FargateTasksCountHwmSum = &v
}

// GetGcpHostTop99pSum returns the GcpHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetGcpHostTop99pSum() int64 {
	if o == nil || o.GcpHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.GcpHostTop99pSum
}

// GetGcpHostTop99pSumOk returns a tuple with the GcpHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetGcpHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.GcpHostTop99pSum == nil {
		return nil, false
	}
	return o.GcpHostTop99pSum, true
}

// HasGcpHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasGcpHostTop99pSum() bool {
	return o != nil && o.GcpHostTop99pSum != nil
}

// SetGcpHostTop99pSum gets a reference to the given int64 and assigns it to the GcpHostTop99pSum field.
func (o *UsageSummaryResponse) SetGcpHostTop99pSum(v int64) {
	o.GcpHostTop99pSum = &v
}

// GetHerokuHostTop99pSum returns the HerokuHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetHerokuHostTop99pSum() int64 {
	if o == nil || o.HerokuHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.HerokuHostTop99pSum
}

// GetHerokuHostTop99pSumOk returns a tuple with the HerokuHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetHerokuHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.HerokuHostTop99pSum == nil {
		return nil, false
	}
	return o.HerokuHostTop99pSum, true
}

// HasHerokuHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasHerokuHostTop99pSum() bool {
	return o != nil && o.HerokuHostTop99pSum != nil
}

// SetHerokuHostTop99pSum gets a reference to the given int64 and assigns it to the HerokuHostTop99pSum field.
func (o *UsageSummaryResponse) SetHerokuHostTop99pSum(v int64) {
	o.HerokuHostTop99pSum = &v
}

// GetIncidentManagementMonthlyActiveUsersHwmSum returns the IncidentManagementMonthlyActiveUsersHwmSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetIncidentManagementMonthlyActiveUsersHwmSum() int64 {
	if o == nil || o.IncidentManagementMonthlyActiveUsersHwmSum == nil {
		var ret int64
		return ret
	}
	return *o.IncidentManagementMonthlyActiveUsersHwmSum
}

// GetIncidentManagementMonthlyActiveUsersHwmSumOk returns a tuple with the IncidentManagementMonthlyActiveUsersHwmSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetIncidentManagementMonthlyActiveUsersHwmSumOk() (*int64, bool) {
	if o == nil || o.IncidentManagementMonthlyActiveUsersHwmSum == nil {
		return nil, false
	}
	return o.IncidentManagementMonthlyActiveUsersHwmSum, true
}

// HasIncidentManagementMonthlyActiveUsersHwmSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasIncidentManagementMonthlyActiveUsersHwmSum() bool {
	return o != nil && o.IncidentManagementMonthlyActiveUsersHwmSum != nil
}

// SetIncidentManagementMonthlyActiveUsersHwmSum gets a reference to the given int64 and assigns it to the IncidentManagementMonthlyActiveUsersHwmSum field.
func (o *UsageSummaryResponse) SetIncidentManagementMonthlyActiveUsersHwmSum(v int64) {
	o.IncidentManagementMonthlyActiveUsersHwmSum = &v
}

// GetIndexedEventsCountAggSum returns the IndexedEventsCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetIndexedEventsCountAggSum() int64 {
	if o == nil || o.IndexedEventsCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.IndexedEventsCountAggSum
}

// GetIndexedEventsCountAggSumOk returns a tuple with the IndexedEventsCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetIndexedEventsCountAggSumOk() (*int64, bool) {
	if o == nil || o.IndexedEventsCountAggSum == nil {
		return nil, false
	}
	return o.IndexedEventsCountAggSum, true
}

// HasIndexedEventsCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasIndexedEventsCountAggSum() bool {
	return o != nil && o.IndexedEventsCountAggSum != nil
}

// SetIndexedEventsCountAggSum gets a reference to the given int64 and assigns it to the IndexedEventsCountAggSum field.
func (o *UsageSummaryResponse) SetIndexedEventsCountAggSum(v int64) {
	o.IndexedEventsCountAggSum = &v
}

// GetInfraHostTop99pSum returns the InfraHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetInfraHostTop99pSum() int64 {
	if o == nil || o.InfraHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.InfraHostTop99pSum
}

// GetInfraHostTop99pSumOk returns a tuple with the InfraHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetInfraHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.InfraHostTop99pSum == nil {
		return nil, false
	}
	return o.InfraHostTop99pSum, true
}

// HasInfraHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasInfraHostTop99pSum() bool {
	return o != nil && o.InfraHostTop99pSum != nil
}

// SetInfraHostTop99pSum gets a reference to the given int64 and assigns it to the InfraHostTop99pSum field.
func (o *UsageSummaryResponse) SetInfraHostTop99pSum(v int64) {
	o.InfraHostTop99pSum = &v
}

// GetIngestedEventsBytesAggSum returns the IngestedEventsBytesAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetIngestedEventsBytesAggSum() int64 {
	if o == nil || o.IngestedEventsBytesAggSum == nil {
		var ret int64
		return ret
	}
	return *o.IngestedEventsBytesAggSum
}

// GetIngestedEventsBytesAggSumOk returns a tuple with the IngestedEventsBytesAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetIngestedEventsBytesAggSumOk() (*int64, bool) {
	if o == nil || o.IngestedEventsBytesAggSum == nil {
		return nil, false
	}
	return o.IngestedEventsBytesAggSum, true
}

// HasIngestedEventsBytesAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasIngestedEventsBytesAggSum() bool {
	return o != nil && o.IngestedEventsBytesAggSum != nil
}

// SetIngestedEventsBytesAggSum gets a reference to the given int64 and assigns it to the IngestedEventsBytesAggSum field.
func (o *UsageSummaryResponse) SetIngestedEventsBytesAggSum(v int64) {
	o.IngestedEventsBytesAggSum = &v
}

// GetIotDeviceAggSum returns the IotDeviceAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetIotDeviceAggSum() int64 {
	if o == nil || o.IotDeviceAggSum == nil {
		var ret int64
		return ret
	}
	return *o.IotDeviceAggSum
}

// GetIotDeviceAggSumOk returns a tuple with the IotDeviceAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetIotDeviceAggSumOk() (*int64, bool) {
	if o == nil || o.IotDeviceAggSum == nil {
		return nil, false
	}
	return o.IotDeviceAggSum, true
}

// HasIotDeviceAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasIotDeviceAggSum() bool {
	return o != nil && o.IotDeviceAggSum != nil
}

// SetIotDeviceAggSum gets a reference to the given int64 and assigns it to the IotDeviceAggSum field.
func (o *UsageSummaryResponse) SetIotDeviceAggSum(v int64) {
	o.IotDeviceAggSum = &v
}

// GetIotDeviceTop99pSum returns the IotDeviceTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetIotDeviceTop99pSum() int64 {
	if o == nil || o.IotDeviceTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.IotDeviceTop99pSum
}

// GetIotDeviceTop99pSumOk returns a tuple with the IotDeviceTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetIotDeviceTop99pSumOk() (*int64, bool) {
	if o == nil || o.IotDeviceTop99pSum == nil {
		return nil, false
	}
	return o.IotDeviceTop99pSum, true
}

// HasIotDeviceTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasIotDeviceTop99pSum() bool {
	return o != nil && o.IotDeviceTop99pSum != nil
}

// SetIotDeviceTop99pSum gets a reference to the given int64 and assigns it to the IotDeviceTop99pSum field.
func (o *UsageSummaryResponse) SetIotDeviceTop99pSum(v int64) {
	o.IotDeviceTop99pSum = &v
}

// GetLastUpdated returns the LastUpdated field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetLastUpdated() time.Time {
	if o == nil || o.LastUpdated == nil {
		var ret time.Time
		return ret
	}
	return *o.LastUpdated
}

// GetLastUpdatedOk returns a tuple with the LastUpdated field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetLastUpdatedOk() (*time.Time, bool) {
	if o == nil || o.LastUpdated == nil {
		return nil, false
	}
	return o.LastUpdated, true
}

// HasLastUpdated returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasLastUpdated() bool {
	return o != nil && o.LastUpdated != nil
}

// SetLastUpdated gets a reference to the given time.Time and assigns it to the LastUpdated field.
func (o *UsageSummaryResponse) SetLastUpdated(v time.Time) {
	o.LastUpdated = &v
}

// GetLiveIndexedEventsAggSum returns the LiveIndexedEventsAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetLiveIndexedEventsAggSum() int64 {
	if o == nil || o.LiveIndexedEventsAggSum == nil {
		var ret int64
		return ret
	}
	return *o.LiveIndexedEventsAggSum
}

// GetLiveIndexedEventsAggSumOk returns a tuple with the LiveIndexedEventsAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetLiveIndexedEventsAggSumOk() (*int64, bool) {
	if o == nil || o.LiveIndexedEventsAggSum == nil {
		return nil, false
	}
	return o.LiveIndexedEventsAggSum, true
}

// HasLiveIndexedEventsAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasLiveIndexedEventsAggSum() bool {
	return o != nil && o.LiveIndexedEventsAggSum != nil
}

// SetLiveIndexedEventsAggSum gets a reference to the given int64 and assigns it to the LiveIndexedEventsAggSum field.
func (o *UsageSummaryResponse) SetLiveIndexedEventsAggSum(v int64) {
	o.LiveIndexedEventsAggSum = &v
}

// GetLiveIngestedBytesAggSum returns the LiveIngestedBytesAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetLiveIngestedBytesAggSum() int64 {
	if o == nil || o.LiveIngestedBytesAggSum == nil {
		var ret int64
		return ret
	}
	return *o.LiveIngestedBytesAggSum
}

// GetLiveIngestedBytesAggSumOk returns a tuple with the LiveIngestedBytesAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetLiveIngestedBytesAggSumOk() (*int64, bool) {
	if o == nil || o.LiveIngestedBytesAggSum == nil {
		return nil, false
	}
	return o.LiveIngestedBytesAggSum, true
}

// HasLiveIngestedBytesAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasLiveIngestedBytesAggSum() bool {
	return o != nil && o.LiveIngestedBytesAggSum != nil
}

// SetLiveIngestedBytesAggSum gets a reference to the given int64 and assigns it to the LiveIngestedBytesAggSum field.
func (o *UsageSummaryResponse) SetLiveIngestedBytesAggSum(v int64) {
	o.LiveIngestedBytesAggSum = &v
}

// GetLogsByRetention returns the LogsByRetention field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetLogsByRetention() LogsByRetention {
	if o == nil || o.LogsByRetention == nil {
		var ret LogsByRetention
		return ret
	}
	return *o.LogsByRetention
}

// GetLogsByRetentionOk returns a tuple with the LogsByRetention field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetLogsByRetentionOk() (*LogsByRetention, bool) {
	if o == nil || o.LogsByRetention == nil {
		return nil, false
	}
	return o.LogsByRetention, true
}

// HasLogsByRetention returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasLogsByRetention() bool {
	return o != nil && o.LogsByRetention != nil
}

// SetLogsByRetention gets a reference to the given LogsByRetention and assigns it to the LogsByRetention field.
func (o *UsageSummaryResponse) SetLogsByRetention(v LogsByRetention) {
	o.LogsByRetention = &v
}

// GetMobileRumLiteSessionCountAggSum returns the MobileRumLiteSessionCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetMobileRumLiteSessionCountAggSum() int64 {
	if o == nil || o.MobileRumLiteSessionCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumLiteSessionCountAggSum
}

// GetMobileRumLiteSessionCountAggSumOk returns a tuple with the MobileRumLiteSessionCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetMobileRumLiteSessionCountAggSumOk() (*int64, bool) {
	if o == nil || o.MobileRumLiteSessionCountAggSum == nil {
		return nil, false
	}
	return o.MobileRumLiteSessionCountAggSum, true
}

// HasMobileRumLiteSessionCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasMobileRumLiteSessionCountAggSum() bool {
	return o != nil && o.MobileRumLiteSessionCountAggSum != nil
}

// SetMobileRumLiteSessionCountAggSum gets a reference to the given int64 and assigns it to the MobileRumLiteSessionCountAggSum field.
func (o *UsageSummaryResponse) SetMobileRumLiteSessionCountAggSum(v int64) {
	o.MobileRumLiteSessionCountAggSum = &v
}

// GetMobileRumSessionCountAggSum returns the MobileRumSessionCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetMobileRumSessionCountAggSum() int64 {
	if o == nil || o.MobileRumSessionCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountAggSum
}

// GetMobileRumSessionCountAggSumOk returns a tuple with the MobileRumSessionCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetMobileRumSessionCountAggSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountAggSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountAggSum, true
}

// HasMobileRumSessionCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasMobileRumSessionCountAggSum() bool {
	return o != nil && o.MobileRumSessionCountAggSum != nil
}

// SetMobileRumSessionCountAggSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountAggSum field.
func (o *UsageSummaryResponse) SetMobileRumSessionCountAggSum(v int64) {
	o.MobileRumSessionCountAggSum = &v
}

// GetMobileRumSessionCountAndroidAggSum returns the MobileRumSessionCountAndroidAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetMobileRumSessionCountAndroidAggSum() int64 {
	if o == nil || o.MobileRumSessionCountAndroidAggSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountAndroidAggSum
}

// GetMobileRumSessionCountAndroidAggSumOk returns a tuple with the MobileRumSessionCountAndroidAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetMobileRumSessionCountAndroidAggSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountAndroidAggSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountAndroidAggSum, true
}

// HasMobileRumSessionCountAndroidAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasMobileRumSessionCountAndroidAggSum() bool {
	return o != nil && o.MobileRumSessionCountAndroidAggSum != nil
}

// SetMobileRumSessionCountAndroidAggSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountAndroidAggSum field.
func (o *UsageSummaryResponse) SetMobileRumSessionCountAndroidAggSum(v int64) {
	o.MobileRumSessionCountAndroidAggSum = &v
}

// GetMobileRumSessionCountIosAggSum returns the MobileRumSessionCountIosAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetMobileRumSessionCountIosAggSum() int64 {
	if o == nil || o.MobileRumSessionCountIosAggSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountIosAggSum
}

// GetMobileRumSessionCountIosAggSumOk returns a tuple with the MobileRumSessionCountIosAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetMobileRumSessionCountIosAggSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountIosAggSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountIosAggSum, true
}

// HasMobileRumSessionCountIosAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasMobileRumSessionCountIosAggSum() bool {
	return o != nil && o.MobileRumSessionCountIosAggSum != nil
}

// SetMobileRumSessionCountIosAggSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountIosAggSum field.
func (o *UsageSummaryResponse) SetMobileRumSessionCountIosAggSum(v int64) {
	o.MobileRumSessionCountIosAggSum = &v
}

// GetMobileRumSessionCountReactnativeAggSum returns the MobileRumSessionCountReactnativeAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetMobileRumSessionCountReactnativeAggSum() int64 {
	if o == nil || o.MobileRumSessionCountReactnativeAggSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumSessionCountReactnativeAggSum
}

// GetMobileRumSessionCountReactnativeAggSumOk returns a tuple with the MobileRumSessionCountReactnativeAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetMobileRumSessionCountReactnativeAggSumOk() (*int64, bool) {
	if o == nil || o.MobileRumSessionCountReactnativeAggSum == nil {
		return nil, false
	}
	return o.MobileRumSessionCountReactnativeAggSum, true
}

// HasMobileRumSessionCountReactnativeAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasMobileRumSessionCountReactnativeAggSum() bool {
	return o != nil && o.MobileRumSessionCountReactnativeAggSum != nil
}

// SetMobileRumSessionCountReactnativeAggSum gets a reference to the given int64 and assigns it to the MobileRumSessionCountReactnativeAggSum field.
func (o *UsageSummaryResponse) SetMobileRumSessionCountReactnativeAggSum(v int64) {
	o.MobileRumSessionCountReactnativeAggSum = &v
}

// GetMobileRumUnitsAggSum returns the MobileRumUnitsAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetMobileRumUnitsAggSum() int64 {
	if o == nil || o.MobileRumUnitsAggSum == nil {
		var ret int64
		return ret
	}
	return *o.MobileRumUnitsAggSum
}

// GetMobileRumUnitsAggSumOk returns a tuple with the MobileRumUnitsAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetMobileRumUnitsAggSumOk() (*int64, bool) {
	if o == nil || o.MobileRumUnitsAggSum == nil {
		return nil, false
	}
	return o.MobileRumUnitsAggSum, true
}

// HasMobileRumUnitsAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasMobileRumUnitsAggSum() bool {
	return o != nil && o.MobileRumUnitsAggSum != nil
}

// SetMobileRumUnitsAggSum gets a reference to the given int64 and assigns it to the MobileRumUnitsAggSum field.
func (o *UsageSummaryResponse) SetMobileRumUnitsAggSum(v int64) {
	o.MobileRumUnitsAggSum = &v
}

// GetNetflowIndexedEventsCountAggSum returns the NetflowIndexedEventsCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetNetflowIndexedEventsCountAggSum() int64 {
	if o == nil || o.NetflowIndexedEventsCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.NetflowIndexedEventsCountAggSum
}

// GetNetflowIndexedEventsCountAggSumOk returns a tuple with the NetflowIndexedEventsCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetNetflowIndexedEventsCountAggSumOk() (*int64, bool) {
	if o == nil || o.NetflowIndexedEventsCountAggSum == nil {
		return nil, false
	}
	return o.NetflowIndexedEventsCountAggSum, true
}

// HasNetflowIndexedEventsCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasNetflowIndexedEventsCountAggSum() bool {
	return o != nil && o.NetflowIndexedEventsCountAggSum != nil
}

// SetNetflowIndexedEventsCountAggSum gets a reference to the given int64 and assigns it to the NetflowIndexedEventsCountAggSum field.
func (o *UsageSummaryResponse) SetNetflowIndexedEventsCountAggSum(v int64) {
	o.NetflowIndexedEventsCountAggSum = &v
}

// GetNpmHostTop99pSum returns the NpmHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetNpmHostTop99pSum() int64 {
	if o == nil || o.NpmHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.NpmHostTop99pSum
}

// GetNpmHostTop99pSumOk returns a tuple with the NpmHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetNpmHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.NpmHostTop99pSum == nil {
		return nil, false
	}
	return o.NpmHostTop99pSum, true
}

// HasNpmHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasNpmHostTop99pSum() bool {
	return o != nil && o.NpmHostTop99pSum != nil
}

// SetNpmHostTop99pSum gets a reference to the given int64 and assigns it to the NpmHostTop99pSum field.
func (o *UsageSummaryResponse) SetNpmHostTop99pSum(v int64) {
	o.NpmHostTop99pSum = &v
}

// GetObservabilityPipelinesBytesProcessedAggSum returns the ObservabilityPipelinesBytesProcessedAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetObservabilityPipelinesBytesProcessedAggSum() int64 {
	if o == nil || o.ObservabilityPipelinesBytesProcessedAggSum == nil {
		var ret int64
		return ret
	}
	return *o.ObservabilityPipelinesBytesProcessedAggSum
}

// GetObservabilityPipelinesBytesProcessedAggSumOk returns a tuple with the ObservabilityPipelinesBytesProcessedAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetObservabilityPipelinesBytesProcessedAggSumOk() (*int64, bool) {
	if o == nil || o.ObservabilityPipelinesBytesProcessedAggSum == nil {
		return nil, false
	}
	return o.ObservabilityPipelinesBytesProcessedAggSum, true
}

// HasObservabilityPipelinesBytesProcessedAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasObservabilityPipelinesBytesProcessedAggSum() bool {
	return o != nil && o.ObservabilityPipelinesBytesProcessedAggSum != nil
}

// SetObservabilityPipelinesBytesProcessedAggSum gets a reference to the given int64 and assigns it to the ObservabilityPipelinesBytesProcessedAggSum field.
func (o *UsageSummaryResponse) SetObservabilityPipelinesBytesProcessedAggSum(v int64) {
	o.ObservabilityPipelinesBytesProcessedAggSum = &v
}

// GetOnlineArchiveEventsCountAggSum returns the OnlineArchiveEventsCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetOnlineArchiveEventsCountAggSum() int64 {
	if o == nil || o.OnlineArchiveEventsCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.OnlineArchiveEventsCountAggSum
}

// GetOnlineArchiveEventsCountAggSumOk returns a tuple with the OnlineArchiveEventsCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetOnlineArchiveEventsCountAggSumOk() (*int64, bool) {
	if o == nil || o.OnlineArchiveEventsCountAggSum == nil {
		return nil, false
	}
	return o.OnlineArchiveEventsCountAggSum, true
}

// HasOnlineArchiveEventsCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasOnlineArchiveEventsCountAggSum() bool {
	return o != nil && o.OnlineArchiveEventsCountAggSum != nil
}

// SetOnlineArchiveEventsCountAggSum gets a reference to the given int64 and assigns it to the OnlineArchiveEventsCountAggSum field.
func (o *UsageSummaryResponse) SetOnlineArchiveEventsCountAggSum(v int64) {
	o.OnlineArchiveEventsCountAggSum = &v
}

// GetOpentelemetryApmHostTop99pSum returns the OpentelemetryApmHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetOpentelemetryApmHostTop99pSum() int64 {
	if o == nil || o.OpentelemetryApmHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.OpentelemetryApmHostTop99pSum
}

// GetOpentelemetryApmHostTop99pSumOk returns a tuple with the OpentelemetryApmHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetOpentelemetryApmHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.OpentelemetryApmHostTop99pSum == nil {
		return nil, false
	}
	return o.OpentelemetryApmHostTop99pSum, true
}

// HasOpentelemetryApmHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasOpentelemetryApmHostTop99pSum() bool {
	return o != nil && o.OpentelemetryApmHostTop99pSum != nil
}

// SetOpentelemetryApmHostTop99pSum gets a reference to the given int64 and assigns it to the OpentelemetryApmHostTop99pSum field.
func (o *UsageSummaryResponse) SetOpentelemetryApmHostTop99pSum(v int64) {
	o.OpentelemetryApmHostTop99pSum = &v
}

// GetOpentelemetryHostTop99pSum returns the OpentelemetryHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetOpentelemetryHostTop99pSum() int64 {
	if o == nil || o.OpentelemetryHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.OpentelemetryHostTop99pSum
}

// GetOpentelemetryHostTop99pSumOk returns a tuple with the OpentelemetryHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetOpentelemetryHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.OpentelemetryHostTop99pSum == nil {
		return nil, false
	}
	return o.OpentelemetryHostTop99pSum, true
}

// HasOpentelemetryHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasOpentelemetryHostTop99pSum() bool {
	return o != nil && o.OpentelemetryHostTop99pSum != nil
}

// SetOpentelemetryHostTop99pSum gets a reference to the given int64 and assigns it to the OpentelemetryHostTop99pSum field.
func (o *UsageSummaryResponse) SetOpentelemetryHostTop99pSum(v int64) {
	o.OpentelemetryHostTop99pSum = &v
}

// GetProfilingContainerAgentCountAvg returns the ProfilingContainerAgentCountAvg field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetProfilingContainerAgentCountAvg() int64 {
	if o == nil || o.ProfilingContainerAgentCountAvg == nil {
		var ret int64
		return ret
	}
	return *o.ProfilingContainerAgentCountAvg
}

// GetProfilingContainerAgentCountAvgOk returns a tuple with the ProfilingContainerAgentCountAvg field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetProfilingContainerAgentCountAvgOk() (*int64, bool) {
	if o == nil || o.ProfilingContainerAgentCountAvg == nil {
		return nil, false
	}
	return o.ProfilingContainerAgentCountAvg, true
}

// HasProfilingContainerAgentCountAvg returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasProfilingContainerAgentCountAvg() bool {
	return o != nil && o.ProfilingContainerAgentCountAvg != nil
}

// SetProfilingContainerAgentCountAvg gets a reference to the given int64 and assigns it to the ProfilingContainerAgentCountAvg field.
func (o *UsageSummaryResponse) SetProfilingContainerAgentCountAvg(v int64) {
	o.ProfilingContainerAgentCountAvg = &v
}

// GetProfilingHostCountTop99pSum returns the ProfilingHostCountTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetProfilingHostCountTop99pSum() int64 {
	if o == nil || o.ProfilingHostCountTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.ProfilingHostCountTop99pSum
}

// GetProfilingHostCountTop99pSumOk returns a tuple with the ProfilingHostCountTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetProfilingHostCountTop99pSumOk() (*int64, bool) {
	if o == nil || o.ProfilingHostCountTop99pSum == nil {
		return nil, false
	}
	return o.ProfilingHostCountTop99pSum, true
}

// HasProfilingHostCountTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasProfilingHostCountTop99pSum() bool {
	return o != nil && o.ProfilingHostCountTop99pSum != nil
}

// SetProfilingHostCountTop99pSum gets a reference to the given int64 and assigns it to the ProfilingHostCountTop99pSum field.
func (o *UsageSummaryResponse) SetProfilingHostCountTop99pSum(v int64) {
	o.ProfilingHostCountTop99pSum = &v
}

// GetRehydratedIndexedEventsAggSum returns the RehydratedIndexedEventsAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetRehydratedIndexedEventsAggSum() int64 {
	if o == nil || o.RehydratedIndexedEventsAggSum == nil {
		var ret int64
		return ret
	}
	return *o.RehydratedIndexedEventsAggSum
}

// GetRehydratedIndexedEventsAggSumOk returns a tuple with the RehydratedIndexedEventsAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetRehydratedIndexedEventsAggSumOk() (*int64, bool) {
	if o == nil || o.RehydratedIndexedEventsAggSum == nil {
		return nil, false
	}
	return o.RehydratedIndexedEventsAggSum, true
}

// HasRehydratedIndexedEventsAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasRehydratedIndexedEventsAggSum() bool {
	return o != nil && o.RehydratedIndexedEventsAggSum != nil
}

// SetRehydratedIndexedEventsAggSum gets a reference to the given int64 and assigns it to the RehydratedIndexedEventsAggSum field.
func (o *UsageSummaryResponse) SetRehydratedIndexedEventsAggSum(v int64) {
	o.RehydratedIndexedEventsAggSum = &v
}

// GetRehydratedIngestedBytesAggSum returns the RehydratedIngestedBytesAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetRehydratedIngestedBytesAggSum() int64 {
	if o == nil || o.RehydratedIngestedBytesAggSum == nil {
		var ret int64
		return ret
	}
	return *o.RehydratedIngestedBytesAggSum
}

// GetRehydratedIngestedBytesAggSumOk returns a tuple with the RehydratedIngestedBytesAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetRehydratedIngestedBytesAggSumOk() (*int64, bool) {
	if o == nil || o.RehydratedIngestedBytesAggSum == nil {
		return nil, false
	}
	return o.RehydratedIngestedBytesAggSum, true
}

// HasRehydratedIngestedBytesAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasRehydratedIngestedBytesAggSum() bool {
	return o != nil && o.RehydratedIngestedBytesAggSum != nil
}

// SetRehydratedIngestedBytesAggSum gets a reference to the given int64 and assigns it to the RehydratedIngestedBytesAggSum field.
func (o *UsageSummaryResponse) SetRehydratedIngestedBytesAggSum(v int64) {
	o.RehydratedIngestedBytesAggSum = &v
}

// GetRumBrowserAndMobileSessionCount returns the RumBrowserAndMobileSessionCount field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetRumBrowserAndMobileSessionCount() int64 {
	if o == nil || o.RumBrowserAndMobileSessionCount == nil {
		var ret int64
		return ret
	}
	return *o.RumBrowserAndMobileSessionCount
}

// GetRumBrowserAndMobileSessionCountOk returns a tuple with the RumBrowserAndMobileSessionCount field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetRumBrowserAndMobileSessionCountOk() (*int64, bool) {
	if o == nil || o.RumBrowserAndMobileSessionCount == nil {
		return nil, false
	}
	return o.RumBrowserAndMobileSessionCount, true
}

// HasRumBrowserAndMobileSessionCount returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasRumBrowserAndMobileSessionCount() bool {
	return o != nil && o.RumBrowserAndMobileSessionCount != nil
}

// SetRumBrowserAndMobileSessionCount gets a reference to the given int64 and assigns it to the RumBrowserAndMobileSessionCount field.
func (o *UsageSummaryResponse) SetRumBrowserAndMobileSessionCount(v int64) {
	o.RumBrowserAndMobileSessionCount = &v
}

// GetRumSessionCountAggSum returns the RumSessionCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetRumSessionCountAggSum() int64 {
	if o == nil || o.RumSessionCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.RumSessionCountAggSum
}

// GetRumSessionCountAggSumOk returns a tuple with the RumSessionCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetRumSessionCountAggSumOk() (*int64, bool) {
	if o == nil || o.RumSessionCountAggSum == nil {
		return nil, false
	}
	return o.RumSessionCountAggSum, true
}

// HasRumSessionCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasRumSessionCountAggSum() bool {
	return o != nil && o.RumSessionCountAggSum != nil
}

// SetRumSessionCountAggSum gets a reference to the given int64 and assigns it to the RumSessionCountAggSum field.
func (o *UsageSummaryResponse) SetRumSessionCountAggSum(v int64) {
	o.RumSessionCountAggSum = &v
}

// GetRumTotalSessionCountAggSum returns the RumTotalSessionCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetRumTotalSessionCountAggSum() int64 {
	if o == nil || o.RumTotalSessionCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.RumTotalSessionCountAggSum
}

// GetRumTotalSessionCountAggSumOk returns a tuple with the RumTotalSessionCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetRumTotalSessionCountAggSumOk() (*int64, bool) {
	if o == nil || o.RumTotalSessionCountAggSum == nil {
		return nil, false
	}
	return o.RumTotalSessionCountAggSum, true
}

// HasRumTotalSessionCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasRumTotalSessionCountAggSum() bool {
	return o != nil && o.RumTotalSessionCountAggSum != nil
}

// SetRumTotalSessionCountAggSum gets a reference to the given int64 and assigns it to the RumTotalSessionCountAggSum field.
func (o *UsageSummaryResponse) SetRumTotalSessionCountAggSum(v int64) {
	o.RumTotalSessionCountAggSum = &v
}

// GetRumUnitsAggSum returns the RumUnitsAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetRumUnitsAggSum() int64 {
	if o == nil || o.RumUnitsAggSum == nil {
		var ret int64
		return ret
	}
	return *o.RumUnitsAggSum
}

// GetRumUnitsAggSumOk returns a tuple with the RumUnitsAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetRumUnitsAggSumOk() (*int64, bool) {
	if o == nil || o.RumUnitsAggSum == nil {
		return nil, false
	}
	return o.RumUnitsAggSum, true
}

// HasRumUnitsAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasRumUnitsAggSum() bool {
	return o != nil && o.RumUnitsAggSum != nil
}

// SetRumUnitsAggSum gets a reference to the given int64 and assigns it to the RumUnitsAggSum field.
func (o *UsageSummaryResponse) SetRumUnitsAggSum(v int64) {
	o.RumUnitsAggSum = &v
}

// GetSdsApmScannedBytesSum returns the SdsApmScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetSdsApmScannedBytesSum() int64 {
	if o == nil || o.SdsApmScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsApmScannedBytesSum
}

// GetSdsApmScannedBytesSumOk returns a tuple with the SdsApmScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetSdsApmScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsApmScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsApmScannedBytesSum, true
}

// HasSdsApmScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasSdsApmScannedBytesSum() bool {
	return o != nil && o.SdsApmScannedBytesSum != nil
}

// SetSdsApmScannedBytesSum gets a reference to the given int64 and assigns it to the SdsApmScannedBytesSum field.
func (o *UsageSummaryResponse) SetSdsApmScannedBytesSum(v int64) {
	o.SdsApmScannedBytesSum = &v
}

// GetSdsEventsScannedBytesSum returns the SdsEventsScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetSdsEventsScannedBytesSum() int64 {
	if o == nil || o.SdsEventsScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsEventsScannedBytesSum
}

// GetSdsEventsScannedBytesSumOk returns a tuple with the SdsEventsScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetSdsEventsScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsEventsScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsEventsScannedBytesSum, true
}

// HasSdsEventsScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasSdsEventsScannedBytesSum() bool {
	return o != nil && o.SdsEventsScannedBytesSum != nil
}

// SetSdsEventsScannedBytesSum gets a reference to the given int64 and assigns it to the SdsEventsScannedBytesSum field.
func (o *UsageSummaryResponse) SetSdsEventsScannedBytesSum(v int64) {
	o.SdsEventsScannedBytesSum = &v
}

// GetSdsLogsScannedBytesSum returns the SdsLogsScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetSdsLogsScannedBytesSum() int64 {
	if o == nil || o.SdsLogsScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsLogsScannedBytesSum
}

// GetSdsLogsScannedBytesSumOk returns a tuple with the SdsLogsScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetSdsLogsScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsLogsScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsLogsScannedBytesSum, true
}

// HasSdsLogsScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasSdsLogsScannedBytesSum() bool {
	return o != nil && o.SdsLogsScannedBytesSum != nil
}

// SetSdsLogsScannedBytesSum gets a reference to the given int64 and assigns it to the SdsLogsScannedBytesSum field.
func (o *UsageSummaryResponse) SetSdsLogsScannedBytesSum(v int64) {
	o.SdsLogsScannedBytesSum = &v
}

// GetSdsRumScannedBytesSum returns the SdsRumScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetSdsRumScannedBytesSum() int64 {
	if o == nil || o.SdsRumScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsRumScannedBytesSum
}

// GetSdsRumScannedBytesSumOk returns a tuple with the SdsRumScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetSdsRumScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsRumScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsRumScannedBytesSum, true
}

// HasSdsRumScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasSdsRumScannedBytesSum() bool {
	return o != nil && o.SdsRumScannedBytesSum != nil
}

// SetSdsRumScannedBytesSum gets a reference to the given int64 and assigns it to the SdsRumScannedBytesSum field.
func (o *UsageSummaryResponse) SetSdsRumScannedBytesSum(v int64) {
	o.SdsRumScannedBytesSum = &v
}

// GetSdsTotalScannedBytesSum returns the SdsTotalScannedBytesSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetSdsTotalScannedBytesSum() int64 {
	if o == nil || o.SdsTotalScannedBytesSum == nil {
		var ret int64
		return ret
	}
	return *o.SdsTotalScannedBytesSum
}

// GetSdsTotalScannedBytesSumOk returns a tuple with the SdsTotalScannedBytesSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetSdsTotalScannedBytesSumOk() (*int64, bool) {
	if o == nil || o.SdsTotalScannedBytesSum == nil {
		return nil, false
	}
	return o.SdsTotalScannedBytesSum, true
}

// HasSdsTotalScannedBytesSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasSdsTotalScannedBytesSum() bool {
	return o != nil && o.SdsTotalScannedBytesSum != nil
}

// SetSdsTotalScannedBytesSum gets a reference to the given int64 and assigns it to the SdsTotalScannedBytesSum field.
func (o *UsageSummaryResponse) SetSdsTotalScannedBytesSum(v int64) {
	o.SdsTotalScannedBytesSum = &v
}

// GetStartDate returns the StartDate field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetStartDate() time.Time {
	if o == nil || o.StartDate == nil {
		var ret time.Time
		return ret
	}
	return *o.StartDate
}

// GetStartDateOk returns a tuple with the StartDate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetStartDateOk() (*time.Time, bool) {
	if o == nil || o.StartDate == nil {
		return nil, false
	}
	return o.StartDate, true
}

// HasStartDate returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasStartDate() bool {
	return o != nil && o.StartDate != nil
}

// SetStartDate gets a reference to the given time.Time and assigns it to the StartDate field.
func (o *UsageSummaryResponse) SetStartDate(v time.Time) {
	o.StartDate = &v
}

// GetSyntheticsBrowserCheckCallsCountAggSum returns the SyntheticsBrowserCheckCallsCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetSyntheticsBrowserCheckCallsCountAggSum() int64 {
	if o == nil || o.SyntheticsBrowserCheckCallsCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.SyntheticsBrowserCheckCallsCountAggSum
}

// GetSyntheticsBrowserCheckCallsCountAggSumOk returns a tuple with the SyntheticsBrowserCheckCallsCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetSyntheticsBrowserCheckCallsCountAggSumOk() (*int64, bool) {
	if o == nil || o.SyntheticsBrowserCheckCallsCountAggSum == nil {
		return nil, false
	}
	return o.SyntheticsBrowserCheckCallsCountAggSum, true
}

// HasSyntheticsBrowserCheckCallsCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasSyntheticsBrowserCheckCallsCountAggSum() bool {
	return o != nil && o.SyntheticsBrowserCheckCallsCountAggSum != nil
}

// SetSyntheticsBrowserCheckCallsCountAggSum gets a reference to the given int64 and assigns it to the SyntheticsBrowserCheckCallsCountAggSum field.
func (o *UsageSummaryResponse) SetSyntheticsBrowserCheckCallsCountAggSum(v int64) {
	o.SyntheticsBrowserCheckCallsCountAggSum = &v
}

// GetSyntheticsCheckCallsCountAggSum returns the SyntheticsCheckCallsCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetSyntheticsCheckCallsCountAggSum() int64 {
	if o == nil || o.SyntheticsCheckCallsCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.SyntheticsCheckCallsCountAggSum
}

// GetSyntheticsCheckCallsCountAggSumOk returns a tuple with the SyntheticsCheckCallsCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetSyntheticsCheckCallsCountAggSumOk() (*int64, bool) {
	if o == nil || o.SyntheticsCheckCallsCountAggSum == nil {
		return nil, false
	}
	return o.SyntheticsCheckCallsCountAggSum, true
}

// HasSyntheticsCheckCallsCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasSyntheticsCheckCallsCountAggSum() bool {
	return o != nil && o.SyntheticsCheckCallsCountAggSum != nil
}

// SetSyntheticsCheckCallsCountAggSum gets a reference to the given int64 and assigns it to the SyntheticsCheckCallsCountAggSum field.
func (o *UsageSummaryResponse) SetSyntheticsCheckCallsCountAggSum(v int64) {
	o.SyntheticsCheckCallsCountAggSum = &v
}

// GetSyntheticsParallelTestingMaxSlotsHwmSum returns the SyntheticsParallelTestingMaxSlotsHwmSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetSyntheticsParallelTestingMaxSlotsHwmSum() int64 {
	if o == nil || o.SyntheticsParallelTestingMaxSlotsHwmSum == nil {
		var ret int64
		return ret
	}
	return *o.SyntheticsParallelTestingMaxSlotsHwmSum
}

// GetSyntheticsParallelTestingMaxSlotsHwmSumOk returns a tuple with the SyntheticsParallelTestingMaxSlotsHwmSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetSyntheticsParallelTestingMaxSlotsHwmSumOk() (*int64, bool) {
	if o == nil || o.SyntheticsParallelTestingMaxSlotsHwmSum == nil {
		return nil, false
	}
	return o.SyntheticsParallelTestingMaxSlotsHwmSum, true
}

// HasSyntheticsParallelTestingMaxSlotsHwmSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasSyntheticsParallelTestingMaxSlotsHwmSum() bool {
	return o != nil && o.SyntheticsParallelTestingMaxSlotsHwmSum != nil
}

// SetSyntheticsParallelTestingMaxSlotsHwmSum gets a reference to the given int64 and assigns it to the SyntheticsParallelTestingMaxSlotsHwmSum field.
func (o *UsageSummaryResponse) SetSyntheticsParallelTestingMaxSlotsHwmSum(v int64) {
	o.SyntheticsParallelTestingMaxSlotsHwmSum = &v
}

// GetTraceSearchIndexedEventsCountAggSum returns the TraceSearchIndexedEventsCountAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetTraceSearchIndexedEventsCountAggSum() int64 {
	if o == nil || o.TraceSearchIndexedEventsCountAggSum == nil {
		var ret int64
		return ret
	}
	return *o.TraceSearchIndexedEventsCountAggSum
}

// GetTraceSearchIndexedEventsCountAggSumOk returns a tuple with the TraceSearchIndexedEventsCountAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetTraceSearchIndexedEventsCountAggSumOk() (*int64, bool) {
	if o == nil || o.TraceSearchIndexedEventsCountAggSum == nil {
		return nil, false
	}
	return o.TraceSearchIndexedEventsCountAggSum, true
}

// HasTraceSearchIndexedEventsCountAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasTraceSearchIndexedEventsCountAggSum() bool {
	return o != nil && o.TraceSearchIndexedEventsCountAggSum != nil
}

// SetTraceSearchIndexedEventsCountAggSum gets a reference to the given int64 and assigns it to the TraceSearchIndexedEventsCountAggSum field.
func (o *UsageSummaryResponse) SetTraceSearchIndexedEventsCountAggSum(v int64) {
	o.TraceSearchIndexedEventsCountAggSum = &v
}

// GetTwolIngestedEventsBytesAggSum returns the TwolIngestedEventsBytesAggSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetTwolIngestedEventsBytesAggSum() int64 {
	if o == nil || o.TwolIngestedEventsBytesAggSum == nil {
		var ret int64
		return ret
	}
	return *o.TwolIngestedEventsBytesAggSum
}

// GetTwolIngestedEventsBytesAggSumOk returns a tuple with the TwolIngestedEventsBytesAggSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetTwolIngestedEventsBytesAggSumOk() (*int64, bool) {
	if o == nil || o.TwolIngestedEventsBytesAggSum == nil {
		return nil, false
	}
	return o.TwolIngestedEventsBytesAggSum, true
}

// HasTwolIngestedEventsBytesAggSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasTwolIngestedEventsBytesAggSum() bool {
	return o != nil && o.TwolIngestedEventsBytesAggSum != nil
}

// SetTwolIngestedEventsBytesAggSum gets a reference to the given int64 and assigns it to the TwolIngestedEventsBytesAggSum field.
func (o *UsageSummaryResponse) SetTwolIngestedEventsBytesAggSum(v int64) {
	o.TwolIngestedEventsBytesAggSum = &v
}

// GetUsage returns the Usage field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetUsage() []UsageSummaryDate {
	if o == nil || o.Usage == nil {
		var ret []UsageSummaryDate
		return ret
	}
	return o.Usage
}

// GetUsageOk returns a tuple with the Usage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetUsageOk() (*[]UsageSummaryDate, bool) {
	if o == nil || o.Usage == nil {
		return nil, false
	}
	return &o.Usage, true
}

// HasUsage returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasUsage() bool {
	return o != nil && o.Usage != nil
}

// SetUsage gets a reference to the given []UsageSummaryDate and assigns it to the Usage field.
func (o *UsageSummaryResponse) SetUsage(v []UsageSummaryDate) {
	o.Usage = v
}

// GetVsphereHostTop99pSum returns the VsphereHostTop99pSum field value if set, zero value otherwise.
func (o *UsageSummaryResponse) GetVsphereHostTop99pSum() int64 {
	if o == nil || o.VsphereHostTop99pSum == nil {
		var ret int64
		return ret
	}
	return *o.VsphereHostTop99pSum
}

// GetVsphereHostTop99pSumOk returns a tuple with the VsphereHostTop99pSum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsageSummaryResponse) GetVsphereHostTop99pSumOk() (*int64, bool) {
	if o == nil || o.VsphereHostTop99pSum == nil {
		return nil, false
	}
	return o.VsphereHostTop99pSum, true
}

// HasVsphereHostTop99pSum returns a boolean if a field has been set.
func (o *UsageSummaryResponse) HasVsphereHostTop99pSum() bool {
	return o != nil && o.VsphereHostTop99pSum != nil
}

// SetVsphereHostTop99pSum gets a reference to the given int64 and assigns it to the VsphereHostTop99pSum field.
func (o *UsageSummaryResponse) SetVsphereHostTop99pSum(v int64) {
	o.VsphereHostTop99pSum = &v
}

// MarshalJSON serializes the struct using spec logic.
func (o UsageSummaryResponse) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.UnparsedObject != nil {
		return json.Marshal(o.UnparsedObject)
	}
	if o.AgentHostTop99pSum != nil {
		toSerialize["agent_host_top99p_sum"] = o.AgentHostTop99pSum
	}
	if o.ApmAzureAppServiceHostTop99pSum != nil {
		toSerialize["apm_azure_app_service_host_top99p_sum"] = o.ApmAzureAppServiceHostTop99pSum
	}
	if o.ApmFargateCountAvgSum != nil {
		toSerialize["apm_fargate_count_avg_sum"] = o.ApmFargateCountAvgSum
	}
	if o.ApmHostTop99pSum != nil {
		toSerialize["apm_host_top99p_sum"] = o.ApmHostTop99pSum
	}
	if o.AppsecFargateCountAvgSum != nil {
		toSerialize["appsec_fargate_count_avg_sum"] = o.AppsecFargateCountAvgSum
	}
	if o.AuditLogsLinesIndexedAggSum != nil {
		toSerialize["audit_logs_lines_indexed_agg_sum"] = o.AuditLogsLinesIndexedAggSum
	}
	if o.AvgProfiledFargateTasksSum != nil {
		toSerialize["avg_profiled_fargate_tasks_sum"] = o.AvgProfiledFargateTasksSum
	}
	if o.AwsHostTop99pSum != nil {
		toSerialize["aws_host_top99p_sum"] = o.AwsHostTop99pSum
	}
	if o.AwsLambdaFuncCount != nil {
		toSerialize["aws_lambda_func_count"] = o.AwsLambdaFuncCount
	}
	if o.AwsLambdaInvocationsSum != nil {
		toSerialize["aws_lambda_invocations_sum"] = o.AwsLambdaInvocationsSum
	}
	if o.AzureAppServiceTop99pSum != nil {
		toSerialize["azure_app_service_top99p_sum"] = o.AzureAppServiceTop99pSum
	}
	if o.AzureHostTop99pSum != nil {
		toSerialize["azure_host_top99p_sum"] = o.AzureHostTop99pSum
	}
	if o.BillableIngestedBytesAggSum != nil {
		toSerialize["billable_ingested_bytes_agg_sum"] = o.BillableIngestedBytesAggSum
	}
	if o.BrowserRumLiteSessionCountAggSum != nil {
		toSerialize["browser_rum_lite_session_count_agg_sum"] = o.BrowserRumLiteSessionCountAggSum
	}
	if o.BrowserRumReplaySessionCountAggSum != nil {
		toSerialize["browser_rum_replay_session_count_agg_sum"] = o.BrowserRumReplaySessionCountAggSum
	}
	if o.BrowserRumUnitsAggSum != nil {
		toSerialize["browser_rum_units_agg_sum"] = o.BrowserRumUnitsAggSum
	}
	if o.CiPipelineIndexedSpansAggSum != nil {
		toSerialize["ci_pipeline_indexed_spans_agg_sum"] = o.CiPipelineIndexedSpansAggSum
	}
	if o.CiTestIndexedSpansAggSum != nil {
		toSerialize["ci_test_indexed_spans_agg_sum"] = o.CiTestIndexedSpansAggSum
	}
	if o.CiVisibilityPipelineCommittersHwmSum != nil {
		toSerialize["ci_visibility_pipeline_committers_hwm_sum"] = o.CiVisibilityPipelineCommittersHwmSum
	}
	if o.CiVisibilityTestCommittersHwmSum != nil {
		toSerialize["ci_visibility_test_committers_hwm_sum"] = o.CiVisibilityTestCommittersHwmSum
	}
	if o.CloudCostManagementHostCountAvgSum != nil {
		toSerialize["cloud_cost_management_host_count_avg_sum"] = o.CloudCostManagementHostCountAvgSum
	}
	if o.ContainerAvgSum != nil {
		toSerialize["container_avg_sum"] = o.ContainerAvgSum
	}
	if o.ContainerExclAgentAvgSum != nil {
		toSerialize["container_excl_agent_avg_sum"] = o.ContainerExclAgentAvgSum
	}
	if o.ContainerHwmSum != nil {
		toSerialize["container_hwm_sum"] = o.ContainerHwmSum
	}
	if o.CspmAasHostTop99pSum != nil {
		toSerialize["cspm_aas_host_top99p_sum"] = o.CspmAasHostTop99pSum
	}
	if o.CspmAwsHostTop99pSum != nil {
		toSerialize["cspm_aws_host_top99p_sum"] = o.CspmAwsHostTop99pSum
	}
	if o.CspmAzureHostTop99pSum != nil {
		toSerialize["cspm_azure_host_top99p_sum"] = o.CspmAzureHostTop99pSum
	}
	if o.CspmContainerAvgSum != nil {
		toSerialize["cspm_container_avg_sum"] = o.CspmContainerAvgSum
	}
	if o.CspmContainerHwmSum != nil {
		toSerialize["cspm_container_hwm_sum"] = o.CspmContainerHwmSum
	}
	if o.CspmGcpHostTop99pSum != nil {
		toSerialize["cspm_gcp_host_top99p_sum"] = o.CspmGcpHostTop99pSum
	}
	if o.CspmHostTop99pSum != nil {
		toSerialize["cspm_host_top99p_sum"] = o.CspmHostTop99pSum
	}
	if o.CustomTsSum != nil {
		toSerialize["custom_ts_sum"] = o.CustomTsSum
	}
	if o.CwsContainersAvgSum != nil {
		toSerialize["cws_containers_avg_sum"] = o.CwsContainersAvgSum
	}
	if o.CwsHostTop99pSum != nil {
		toSerialize["cws_host_top99p_sum"] = o.CwsHostTop99pSum
	}
	if o.DbmHostTop99pSum != nil {
		toSerialize["dbm_host_top99p_sum"] = o.DbmHostTop99pSum
	}
	if o.DbmQueriesAvgSum != nil {
		toSerialize["dbm_queries_avg_sum"] = o.DbmQueriesAvgSum
	}
	if o.EndDate != nil {
		if o.EndDate.Nanosecond() == 0 {
			toSerialize["end_date"] = o.EndDate.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["end_date"] = o.EndDate.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.FargateTasksCountAvgSum != nil {
		toSerialize["fargate_tasks_count_avg_sum"] = o.FargateTasksCountAvgSum
	}
	if o.FargateTasksCountHwmSum != nil {
		toSerialize["fargate_tasks_count_hwm_sum"] = o.FargateTasksCountHwmSum
	}
	if o.GcpHostTop99pSum != nil {
		toSerialize["gcp_host_top99p_sum"] = o.GcpHostTop99pSum
	}
	if o.HerokuHostTop99pSum != nil {
		toSerialize["heroku_host_top99p_sum"] = o.HerokuHostTop99pSum
	}
	if o.IncidentManagementMonthlyActiveUsersHwmSum != nil {
		toSerialize["incident_management_monthly_active_users_hwm_sum"] = o.IncidentManagementMonthlyActiveUsersHwmSum
	}
	if o.IndexedEventsCountAggSum != nil {
		toSerialize["indexed_events_count_agg_sum"] = o.IndexedEventsCountAggSum
	}
	if o.InfraHostTop99pSum != nil {
		toSerialize["infra_host_top99p_sum"] = o.InfraHostTop99pSum
	}
	if o.IngestedEventsBytesAggSum != nil {
		toSerialize["ingested_events_bytes_agg_sum"] = o.IngestedEventsBytesAggSum
	}
	if o.IotDeviceAggSum != nil {
		toSerialize["iot_device_agg_sum"] = o.IotDeviceAggSum
	}
	if o.IotDeviceTop99pSum != nil {
		toSerialize["iot_device_top99p_sum"] = o.IotDeviceTop99pSum
	}
	if o.LastUpdated != nil {
		if o.LastUpdated.Nanosecond() == 0 {
			toSerialize["last_updated"] = o.LastUpdated.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["last_updated"] = o.LastUpdated.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.LiveIndexedEventsAggSum != nil {
		toSerialize["live_indexed_events_agg_sum"] = o.LiveIndexedEventsAggSum
	}
	if o.LiveIngestedBytesAggSum != nil {
		toSerialize["live_ingested_bytes_agg_sum"] = o.LiveIngestedBytesAggSum
	}
	if o.LogsByRetention != nil {
		toSerialize["logs_by_retention"] = o.LogsByRetention
	}
	if o.MobileRumLiteSessionCountAggSum != nil {
		toSerialize["mobile_rum_lite_session_count_agg_sum"] = o.MobileRumLiteSessionCountAggSum
	}
	if o.MobileRumSessionCountAggSum != nil {
		toSerialize["mobile_rum_session_count_agg_sum"] = o.MobileRumSessionCountAggSum
	}
	if o.MobileRumSessionCountAndroidAggSum != nil {
		toSerialize["mobile_rum_session_count_android_agg_sum"] = o.MobileRumSessionCountAndroidAggSum
	}
	if o.MobileRumSessionCountIosAggSum != nil {
		toSerialize["mobile_rum_session_count_ios_agg_sum"] = o.MobileRumSessionCountIosAggSum
	}
	if o.MobileRumSessionCountReactnativeAggSum != nil {
		toSerialize["mobile_rum_session_count_reactnative_agg_sum"] = o.MobileRumSessionCountReactnativeAggSum
	}
	if o.MobileRumUnitsAggSum != nil {
		toSerialize["mobile_rum_units_agg_sum"] = o.MobileRumUnitsAggSum
	}
	if o.NetflowIndexedEventsCountAggSum != nil {
		toSerialize["netflow_indexed_events_count_agg_sum"] = o.NetflowIndexedEventsCountAggSum
	}
	if o.NpmHostTop99pSum != nil {
		toSerialize["npm_host_top99p_sum"] = o.NpmHostTop99pSum
	}
	if o.ObservabilityPipelinesBytesProcessedAggSum != nil {
		toSerialize["observability_pipelines_bytes_processed_agg_sum"] = o.ObservabilityPipelinesBytesProcessedAggSum
	}
	if o.OnlineArchiveEventsCountAggSum != nil {
		toSerialize["online_archive_events_count_agg_sum"] = o.OnlineArchiveEventsCountAggSum
	}
	if o.OpentelemetryApmHostTop99pSum != nil {
		toSerialize["opentelemetry_apm_host_top99p_sum"] = o.OpentelemetryApmHostTop99pSum
	}
	if o.OpentelemetryHostTop99pSum != nil {
		toSerialize["opentelemetry_host_top99p_sum"] = o.OpentelemetryHostTop99pSum
	}
	if o.ProfilingContainerAgentCountAvg != nil {
		toSerialize["profiling_container_agent_count_avg"] = o.ProfilingContainerAgentCountAvg
	}
	if o.ProfilingHostCountTop99pSum != nil {
		toSerialize["profiling_host_count_top99p_sum"] = o.ProfilingHostCountTop99pSum
	}
	if o.RehydratedIndexedEventsAggSum != nil {
		toSerialize["rehydrated_indexed_events_agg_sum"] = o.RehydratedIndexedEventsAggSum
	}
	if o.RehydratedIngestedBytesAggSum != nil {
		toSerialize["rehydrated_ingested_bytes_agg_sum"] = o.RehydratedIngestedBytesAggSum
	}
	if o.RumBrowserAndMobileSessionCount != nil {
		toSerialize["rum_browser_and_mobile_session_count"] = o.RumBrowserAndMobileSessionCount
	}
	if o.RumSessionCountAggSum != nil {
		toSerialize["rum_session_count_agg_sum"] = o.RumSessionCountAggSum
	}
	if o.RumTotalSessionCountAggSum != nil {
		toSerialize["rum_total_session_count_agg_sum"] = o.RumTotalSessionCountAggSum
	}
	if o.RumUnitsAggSum != nil {
		toSerialize["rum_units_agg_sum"] = o.RumUnitsAggSum
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
	if o.StartDate != nil {
		if o.StartDate.Nanosecond() == 0 {
			toSerialize["start_date"] = o.StartDate.Format("2006-01-02T15:04:05Z07:00")
		} else {
			toSerialize["start_date"] = o.StartDate.Format("2006-01-02T15:04:05.000Z07:00")
		}
	}
	if o.SyntheticsBrowserCheckCallsCountAggSum != nil {
		toSerialize["synthetics_browser_check_calls_count_agg_sum"] = o.SyntheticsBrowserCheckCallsCountAggSum
	}
	if o.SyntheticsCheckCallsCountAggSum != nil {
		toSerialize["synthetics_check_calls_count_agg_sum"] = o.SyntheticsCheckCallsCountAggSum
	}
	if o.SyntheticsParallelTestingMaxSlotsHwmSum != nil {
		toSerialize["synthetics_parallel_testing_max_slots_hwm_sum"] = o.SyntheticsParallelTestingMaxSlotsHwmSum
	}
	if o.TraceSearchIndexedEventsCountAggSum != nil {
		toSerialize["trace_search_indexed_events_count_agg_sum"] = o.TraceSearchIndexedEventsCountAggSum
	}
	if o.TwolIngestedEventsBytesAggSum != nil {
		toSerialize["twol_ingested_events_bytes_agg_sum"] = o.TwolIngestedEventsBytesAggSum
	}
	if o.Usage != nil {
		toSerialize["usage"] = o.Usage
	}
	if o.VsphereHostTop99pSum != nil {
		toSerialize["vsphere_host_top99p_sum"] = o.VsphereHostTop99pSum
	}

	for key, value := range o.AdditionalProperties {
		toSerialize[key] = value
	}
	return json.Marshal(toSerialize)
}

// UnmarshalJSON deserializes the given payload.
func (o *UsageSummaryResponse) UnmarshalJSON(bytes []byte) (err error) {
	raw := map[string]interface{}{}
	all := struct {
		AgentHostTop99pSum                         *int64             `json:"agent_host_top99p_sum,omitempty"`
		ApmAzureAppServiceHostTop99pSum            *int64             `json:"apm_azure_app_service_host_top99p_sum,omitempty"`
		ApmFargateCountAvgSum                      *int64             `json:"apm_fargate_count_avg_sum,omitempty"`
		ApmHostTop99pSum                           *int64             `json:"apm_host_top99p_sum,omitempty"`
		AppsecFargateCountAvgSum                   *int64             `json:"appsec_fargate_count_avg_sum,omitempty"`
		AuditLogsLinesIndexedAggSum                *int64             `json:"audit_logs_lines_indexed_agg_sum,omitempty"`
		AvgProfiledFargateTasksSum                 *int64             `json:"avg_profiled_fargate_tasks_sum,omitempty"`
		AwsHostTop99pSum                           *int64             `json:"aws_host_top99p_sum,omitempty"`
		AwsLambdaFuncCount                         *int64             `json:"aws_lambda_func_count,omitempty"`
		AwsLambdaInvocationsSum                    *int64             `json:"aws_lambda_invocations_sum,omitempty"`
		AzureAppServiceTop99pSum                   *int64             `json:"azure_app_service_top99p_sum,omitempty"`
		AzureHostTop99pSum                         *int64             `json:"azure_host_top99p_sum,omitempty"`
		BillableIngestedBytesAggSum                *int64             `json:"billable_ingested_bytes_agg_sum,omitempty"`
		BrowserRumLiteSessionCountAggSum           *int64             `json:"browser_rum_lite_session_count_agg_sum,omitempty"`
		BrowserRumReplaySessionCountAggSum         *int64             `json:"browser_rum_replay_session_count_agg_sum,omitempty"`
		BrowserRumUnitsAggSum                      *int64             `json:"browser_rum_units_agg_sum,omitempty"`
		CiPipelineIndexedSpansAggSum               *int64             `json:"ci_pipeline_indexed_spans_agg_sum,omitempty"`
		CiTestIndexedSpansAggSum                   *int64             `json:"ci_test_indexed_spans_agg_sum,omitempty"`
		CiVisibilityPipelineCommittersHwmSum       *int64             `json:"ci_visibility_pipeline_committers_hwm_sum,omitempty"`
		CiVisibilityTestCommittersHwmSum           *int64             `json:"ci_visibility_test_committers_hwm_sum,omitempty"`
		CloudCostManagementHostCountAvgSum         *int64             `json:"cloud_cost_management_host_count_avg_sum,omitempty"`
		ContainerAvgSum                            *int64             `json:"container_avg_sum,omitempty"`
		ContainerExclAgentAvgSum                   *int64             `json:"container_excl_agent_avg_sum,omitempty"`
		ContainerHwmSum                            *int64             `json:"container_hwm_sum,omitempty"`
		CspmAasHostTop99pSum                       *int64             `json:"cspm_aas_host_top99p_sum,omitempty"`
		CspmAwsHostTop99pSum                       *int64             `json:"cspm_aws_host_top99p_sum,omitempty"`
		CspmAzureHostTop99pSum                     *int64             `json:"cspm_azure_host_top99p_sum,omitempty"`
		CspmContainerAvgSum                        *int64             `json:"cspm_container_avg_sum,omitempty"`
		CspmContainerHwmSum                        *int64             `json:"cspm_container_hwm_sum,omitempty"`
		CspmGcpHostTop99pSum                       *int64             `json:"cspm_gcp_host_top99p_sum,omitempty"`
		CspmHostTop99pSum                          *int64             `json:"cspm_host_top99p_sum,omitempty"`
		CustomTsSum                                *int64             `json:"custom_ts_sum,omitempty"`
		CwsContainersAvgSum                        *int64             `json:"cws_containers_avg_sum,omitempty"`
		CwsHostTop99pSum                           *int64             `json:"cws_host_top99p_sum,omitempty"`
		DbmHostTop99pSum                           *int64             `json:"dbm_host_top99p_sum,omitempty"`
		DbmQueriesAvgSum                           *int64             `json:"dbm_queries_avg_sum,omitempty"`
		EndDate                                    *time.Time         `json:"end_date,omitempty"`
		FargateTasksCountAvgSum                    *int64             `json:"fargate_tasks_count_avg_sum,omitempty"`
		FargateTasksCountHwmSum                    *int64             `json:"fargate_tasks_count_hwm_sum,omitempty"`
		GcpHostTop99pSum                           *int64             `json:"gcp_host_top99p_sum,omitempty"`
		HerokuHostTop99pSum                        *int64             `json:"heroku_host_top99p_sum,omitempty"`
		IncidentManagementMonthlyActiveUsersHwmSum *int64             `json:"incident_management_monthly_active_users_hwm_sum,omitempty"`
		IndexedEventsCountAggSum                   *int64             `json:"indexed_events_count_agg_sum,omitempty"`
		InfraHostTop99pSum                         *int64             `json:"infra_host_top99p_sum,omitempty"`
		IngestedEventsBytesAggSum                  *int64             `json:"ingested_events_bytes_agg_sum,omitempty"`
		IotDeviceAggSum                            *int64             `json:"iot_device_agg_sum,omitempty"`
		IotDeviceTop99pSum                         *int64             `json:"iot_device_top99p_sum,omitempty"`
		LastUpdated                                *time.Time         `json:"last_updated,omitempty"`
		LiveIndexedEventsAggSum                    *int64             `json:"live_indexed_events_agg_sum,omitempty"`
		LiveIngestedBytesAggSum                    *int64             `json:"live_ingested_bytes_agg_sum,omitempty"`
		LogsByRetention                            *LogsByRetention   `json:"logs_by_retention,omitempty"`
		MobileRumLiteSessionCountAggSum            *int64             `json:"mobile_rum_lite_session_count_agg_sum,omitempty"`
		MobileRumSessionCountAggSum                *int64             `json:"mobile_rum_session_count_agg_sum,omitempty"`
		MobileRumSessionCountAndroidAggSum         *int64             `json:"mobile_rum_session_count_android_agg_sum,omitempty"`
		MobileRumSessionCountIosAggSum             *int64             `json:"mobile_rum_session_count_ios_agg_sum,omitempty"`
		MobileRumSessionCountReactnativeAggSum     *int64             `json:"mobile_rum_session_count_reactnative_agg_sum,omitempty"`
		MobileRumUnitsAggSum                       *int64             `json:"mobile_rum_units_agg_sum,omitempty"`
		NetflowIndexedEventsCountAggSum            *int64             `json:"netflow_indexed_events_count_agg_sum,omitempty"`
		NpmHostTop99pSum                           *int64             `json:"npm_host_top99p_sum,omitempty"`
		ObservabilityPipelinesBytesProcessedAggSum *int64             `json:"observability_pipelines_bytes_processed_agg_sum,omitempty"`
		OnlineArchiveEventsCountAggSum             *int64             `json:"online_archive_events_count_agg_sum,omitempty"`
		OpentelemetryApmHostTop99pSum              *int64             `json:"opentelemetry_apm_host_top99p_sum,omitempty"`
		OpentelemetryHostTop99pSum                 *int64             `json:"opentelemetry_host_top99p_sum,omitempty"`
		ProfilingContainerAgentCountAvg            *int64             `json:"profiling_container_agent_count_avg,omitempty"`
		ProfilingHostCountTop99pSum                *int64             `json:"profiling_host_count_top99p_sum,omitempty"`
		RehydratedIndexedEventsAggSum              *int64             `json:"rehydrated_indexed_events_agg_sum,omitempty"`
		RehydratedIngestedBytesAggSum              *int64             `json:"rehydrated_ingested_bytes_agg_sum,omitempty"`
		RumBrowserAndMobileSessionCount            *int64             `json:"rum_browser_and_mobile_session_count,omitempty"`
		RumSessionCountAggSum                      *int64             `json:"rum_session_count_agg_sum,omitempty"`
		RumTotalSessionCountAggSum                 *int64             `json:"rum_total_session_count_agg_sum,omitempty"`
		RumUnitsAggSum                             *int64             `json:"rum_units_agg_sum,omitempty"`
		SdsApmScannedBytesSum                      *int64             `json:"sds_apm_scanned_bytes_sum,omitempty"`
		SdsEventsScannedBytesSum                   *int64             `json:"sds_events_scanned_bytes_sum,omitempty"`
		SdsLogsScannedBytesSum                     *int64             `json:"sds_logs_scanned_bytes_sum,omitempty"`
		SdsRumScannedBytesSum                      *int64             `json:"sds_rum_scanned_bytes_sum,omitempty"`
		SdsTotalScannedBytesSum                    *int64             `json:"sds_total_scanned_bytes_sum,omitempty"`
		StartDate                                  *time.Time         `json:"start_date,omitempty"`
		SyntheticsBrowserCheckCallsCountAggSum     *int64             `json:"synthetics_browser_check_calls_count_agg_sum,omitempty"`
		SyntheticsCheckCallsCountAggSum            *int64             `json:"synthetics_check_calls_count_agg_sum,omitempty"`
		SyntheticsParallelTestingMaxSlotsHwmSum    *int64             `json:"synthetics_parallel_testing_max_slots_hwm_sum,omitempty"`
		TraceSearchIndexedEventsCountAggSum        *int64             `json:"trace_search_indexed_events_count_agg_sum,omitempty"`
		TwolIngestedEventsBytesAggSum              *int64             `json:"twol_ingested_events_bytes_agg_sum,omitempty"`
		Usage                                      []UsageSummaryDate `json:"usage,omitempty"`
		VsphereHostTop99pSum                       *int64             `json:"vsphere_host_top99p_sum,omitempty"`
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
	o.AgentHostTop99pSum = all.AgentHostTop99pSum
	o.ApmAzureAppServiceHostTop99pSum = all.ApmAzureAppServiceHostTop99pSum
	o.ApmFargateCountAvgSum = all.ApmFargateCountAvgSum
	o.ApmHostTop99pSum = all.ApmHostTop99pSum
	o.AppsecFargateCountAvgSum = all.AppsecFargateCountAvgSum
	o.AuditLogsLinesIndexedAggSum = all.AuditLogsLinesIndexedAggSum
	o.AvgProfiledFargateTasksSum = all.AvgProfiledFargateTasksSum
	o.AwsHostTop99pSum = all.AwsHostTop99pSum
	o.AwsLambdaFuncCount = all.AwsLambdaFuncCount
	o.AwsLambdaInvocationsSum = all.AwsLambdaInvocationsSum
	o.AzureAppServiceTop99pSum = all.AzureAppServiceTop99pSum
	o.AzureHostTop99pSum = all.AzureHostTop99pSum
	o.BillableIngestedBytesAggSum = all.BillableIngestedBytesAggSum
	o.BrowserRumLiteSessionCountAggSum = all.BrowserRumLiteSessionCountAggSum
	o.BrowserRumReplaySessionCountAggSum = all.BrowserRumReplaySessionCountAggSum
	o.BrowserRumUnitsAggSum = all.BrowserRumUnitsAggSum
	o.CiPipelineIndexedSpansAggSum = all.CiPipelineIndexedSpansAggSum
	o.CiTestIndexedSpansAggSum = all.CiTestIndexedSpansAggSum
	o.CiVisibilityPipelineCommittersHwmSum = all.CiVisibilityPipelineCommittersHwmSum
	o.CiVisibilityTestCommittersHwmSum = all.CiVisibilityTestCommittersHwmSum
	o.CloudCostManagementHostCountAvgSum = all.CloudCostManagementHostCountAvgSum
	o.ContainerAvgSum = all.ContainerAvgSum
	o.ContainerExclAgentAvgSum = all.ContainerExclAgentAvgSum
	o.ContainerHwmSum = all.ContainerHwmSum
	o.CspmAasHostTop99pSum = all.CspmAasHostTop99pSum
	o.CspmAwsHostTop99pSum = all.CspmAwsHostTop99pSum
	o.CspmAzureHostTop99pSum = all.CspmAzureHostTop99pSum
	o.CspmContainerAvgSum = all.CspmContainerAvgSum
	o.CspmContainerHwmSum = all.CspmContainerHwmSum
	o.CspmGcpHostTop99pSum = all.CspmGcpHostTop99pSum
	o.CspmHostTop99pSum = all.CspmHostTop99pSum
	o.CustomTsSum = all.CustomTsSum
	o.CwsContainersAvgSum = all.CwsContainersAvgSum
	o.CwsHostTop99pSum = all.CwsHostTop99pSum
	o.DbmHostTop99pSum = all.DbmHostTop99pSum
	o.DbmQueriesAvgSum = all.DbmQueriesAvgSum
	o.EndDate = all.EndDate
	o.FargateTasksCountAvgSum = all.FargateTasksCountAvgSum
	o.FargateTasksCountHwmSum = all.FargateTasksCountHwmSum
	o.GcpHostTop99pSum = all.GcpHostTop99pSum
	o.HerokuHostTop99pSum = all.HerokuHostTop99pSum
	o.IncidentManagementMonthlyActiveUsersHwmSum = all.IncidentManagementMonthlyActiveUsersHwmSum
	o.IndexedEventsCountAggSum = all.IndexedEventsCountAggSum
	o.InfraHostTop99pSum = all.InfraHostTop99pSum
	o.IngestedEventsBytesAggSum = all.IngestedEventsBytesAggSum
	o.IotDeviceAggSum = all.IotDeviceAggSum
	o.IotDeviceTop99pSum = all.IotDeviceTop99pSum
	o.LastUpdated = all.LastUpdated
	o.LiveIndexedEventsAggSum = all.LiveIndexedEventsAggSum
	o.LiveIngestedBytesAggSum = all.LiveIngestedBytesAggSum
	if all.LogsByRetention != nil && all.LogsByRetention.UnparsedObject != nil && o.UnparsedObject == nil {
		err = json.Unmarshal(bytes, &raw)
		if err != nil {
			return err
		}
		o.UnparsedObject = raw
	}
	o.LogsByRetention = all.LogsByRetention
	o.MobileRumLiteSessionCountAggSum = all.MobileRumLiteSessionCountAggSum
	o.MobileRumSessionCountAggSum = all.MobileRumSessionCountAggSum
	o.MobileRumSessionCountAndroidAggSum = all.MobileRumSessionCountAndroidAggSum
	o.MobileRumSessionCountIosAggSum = all.MobileRumSessionCountIosAggSum
	o.MobileRumSessionCountReactnativeAggSum = all.MobileRumSessionCountReactnativeAggSum
	o.MobileRumUnitsAggSum = all.MobileRumUnitsAggSum
	o.NetflowIndexedEventsCountAggSum = all.NetflowIndexedEventsCountAggSum
	o.NpmHostTop99pSum = all.NpmHostTop99pSum
	o.ObservabilityPipelinesBytesProcessedAggSum = all.ObservabilityPipelinesBytesProcessedAggSum
	o.OnlineArchiveEventsCountAggSum = all.OnlineArchiveEventsCountAggSum
	o.OpentelemetryApmHostTop99pSum = all.OpentelemetryApmHostTop99pSum
	o.OpentelemetryHostTop99pSum = all.OpentelemetryHostTop99pSum
	o.ProfilingContainerAgentCountAvg = all.ProfilingContainerAgentCountAvg
	o.ProfilingHostCountTop99pSum = all.ProfilingHostCountTop99pSum
	o.RehydratedIndexedEventsAggSum = all.RehydratedIndexedEventsAggSum
	o.RehydratedIngestedBytesAggSum = all.RehydratedIngestedBytesAggSum
	o.RumBrowserAndMobileSessionCount = all.RumBrowserAndMobileSessionCount
	o.RumSessionCountAggSum = all.RumSessionCountAggSum
	o.RumTotalSessionCountAggSum = all.RumTotalSessionCountAggSum
	o.RumUnitsAggSum = all.RumUnitsAggSum
	o.SdsApmScannedBytesSum = all.SdsApmScannedBytesSum
	o.SdsEventsScannedBytesSum = all.SdsEventsScannedBytesSum
	o.SdsLogsScannedBytesSum = all.SdsLogsScannedBytesSum
	o.SdsRumScannedBytesSum = all.SdsRumScannedBytesSum
	o.SdsTotalScannedBytesSum = all.SdsTotalScannedBytesSum
	o.StartDate = all.StartDate
	o.SyntheticsBrowserCheckCallsCountAggSum = all.SyntheticsBrowserCheckCallsCountAggSum
	o.SyntheticsCheckCallsCountAggSum = all.SyntheticsCheckCallsCountAggSum
	o.SyntheticsParallelTestingMaxSlotsHwmSum = all.SyntheticsParallelTestingMaxSlotsHwmSum
	o.TraceSearchIndexedEventsCountAggSum = all.TraceSearchIndexedEventsCountAggSum
	o.TwolIngestedEventsBytesAggSum = all.TwolIngestedEventsBytesAggSum
	o.Usage = all.Usage
	o.VsphereHostTop99pSum = all.VsphereHostTop99pSum
	return nil
}
