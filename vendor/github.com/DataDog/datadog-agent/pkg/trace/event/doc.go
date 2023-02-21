// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package event contains functionality related to APM event extraction from traces.
//
// APM Events constitute the core of Datadog's Trace Search functionality. These are, in a nutshell, individual spans
// containing important information (but not full trace tree) about an execution and which can therefore be sampled at a
// different rate (retaining greater cardinality than that of complete traces). Furthermore, all information in APM events
// can be indexed, allowing for very flexible searching.
//
// For instance, consider a web server. The top-level span on traces from this web server likely contains interesting
// things such as customer/user id, IPs, HTTP tags, HTTP endpoint, among others. By extracting this top level span from
// each trace, converting it into an APM event and feeding it into trace search, you can potentially search and aggregate
// this information for all requests arriving at your web server. You couldn't do the same thing with traces because these
// capture entire execution trees which are much more expensive to process and store and are therefore heavily sampled.
//
// Of course, if the trace from which APM events were extracted also survives sampling, you can easily see the execution
// tree associated with a particular APM event as this link is kept throughout the entire processing pipeline.
package event
