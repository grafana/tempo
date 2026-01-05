package combiner

import (
	"net/http"
	"sort"

	"github.com/go-kit/log/level"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/grafana/tempo/pkg/tempopb"
)

// CombineTagsResults combines SearchTagsResponse results from multiple Tempo instances
func (c *Combiner) CombineTagsResults(results []QueryResult) (*tempopb.SearchTagsResponse, error) {
	tagSet := make(map[string]struct{})
	var combinedMetrics *tempopb.MetadataMetrics

	for _, result := range results {
		if result.Error != nil {
			level.Warn(c.logger).Log("msg", "instance returned error for tags", "instance", result.Instance, "err", result.Error)
			continue
		}

		if result.Response == nil {
			level.Warn(c.logger).Log("msg", "instance returned nil response for tags", "instance", result.Instance)
			continue
		}

		// Skip 404s - instance doesn't have tags
		if result.Response.StatusCode == http.StatusNotFound {
			level.Debug(c.logger).Log("msg", "instance returned 404 for tags", "instance", result.Instance)
			continue
		}

		if result.Response.StatusCode != http.StatusOK {
			level.Warn(c.logger).Log("msg", "instance returned non-OK status for tags", "instance", result.Instance, "status", result.Response.StatusCode)
			continue
		}

		// Body is already read by QueryAllInstances
		if len(result.Body) == 0 {
			level.Debug(c.logger).Log("msg", "instance returned empty body for tags", "instance", result.Instance)
			continue
		}

		// Parse the JSON response
		var tagsResp tempopb.SearchTagsResponse
		if err := jsonpb.UnmarshalString(string(result.Body), &tagsResp); err != nil {
			level.Warn(c.logger).Log("msg", "failed to unmarshal tags response", "instance", result.Instance, "err", err)
			continue
		}

		// Add tags to set (deduplication)
		for _, tag := range tagsResp.TagNames {
			tagSet[tag] = struct{}{}
		}

		// Combine metrics
		if tagsResp.Metrics != nil {
			combinedMetrics = combineMetadataMetrics(combinedMetrics, tagsResp.Metrics)
		}

		level.Debug(c.logger).Log("msg", "combined tags from instance", "instance", result.Instance, "tags", len(tagsResp.TagNames))
	}

	// Convert set to sorted slice
	tagNames := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tagNames = append(tagNames, tag)
	}
	sort.Strings(tagNames)

	return &tempopb.SearchTagsResponse{
		TagNames: tagNames,
		Metrics:  combinedMetrics,
	}, nil
}

// CombineTagsV2Results combines SearchTagsV2Response results from multiple Tempo instances
func (c *Combiner) CombineTagsV2Results(results []QueryResult) (*tempopb.SearchTagsV2Response, error) {
	// Map of scope name to set of tags
	scopeTagsMap := make(map[string]map[string]struct{})
	var combinedMetrics *tempopb.MetadataMetrics

	for _, result := range results {
		if result.Error != nil {
			level.Warn(c.logger).Log("msg", "instance returned error for tags v2", "instance", result.Instance, "err", result.Error)
			continue
		}

		if result.Response == nil {
			level.Warn(c.logger).Log("msg", "instance returned nil response for tags v2", "instance", result.Instance)
			continue
		}

		// Skip 404s - instance doesn't have tags
		if result.Response.StatusCode == http.StatusNotFound {
			level.Debug(c.logger).Log("msg", "instance returned 404 for tags v2", "instance", result.Instance)
			continue
		}

		if result.Response.StatusCode != http.StatusOK {
			level.Warn(c.logger).Log("msg", "instance returned non-OK status for tags v2", "instance", result.Instance, "status", result.Response.StatusCode)
			continue
		}

		// Body is already read by QueryAllInstances
		if len(result.Body) == 0 {
			level.Debug(c.logger).Log("msg", "instance returned empty body for tags v2", "instance", result.Instance)
			continue
		}

		// Parse the JSON response
		var tagsResp tempopb.SearchTagsV2Response
		if err := jsonpb.UnmarshalString(string(result.Body), &tagsResp); err != nil {
			level.Warn(c.logger).Log("msg", "failed to unmarshal tags v2 response", "instance", result.Instance, "err", err)
			continue
		}

		// Merge scopes and tags
		for _, scope := range tagsResp.Scopes {
			if _, exists := scopeTagsMap[scope.Name]; !exists {
				scopeTagsMap[scope.Name] = make(map[string]struct{})
			}
			for _, tag := range scope.Tags {
				scopeTagsMap[scope.Name][tag] = struct{}{}
			}
		}

		// Combine metrics
		if tagsResp.Metrics != nil {
			combinedMetrics = combineMetadataMetrics(combinedMetrics, tagsResp.Metrics)
		}

		level.Debug(c.logger).Log("msg", "combined tags v2 from instance", "instance", result.Instance, "scopes", len(tagsResp.Scopes))
	}

	// Convert map to sorted slices
	scopes := make([]*tempopb.SearchTagsV2Scope, 0, len(scopeTagsMap))
	for scopeName, tagSet := range scopeTagsMap {
		tags := make([]string, 0, len(tagSet))
		for tag := range tagSet {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		scopes = append(scopes, &tempopb.SearchTagsV2Scope{
			Name: scopeName,
			Tags: tags,
		})
	}
	// Sort scopes by name
	sort.Slice(scopes, func(i, j int) bool {
		return scopes[i].Name < scopes[j].Name
	})

	return &tempopb.SearchTagsV2Response{
		Scopes:  scopes,
		Metrics: combinedMetrics,
	}, nil
}

// CombineTagValuesResults combines SearchTagValuesResponse results from multiple Tempo instances
func (c *Combiner) CombineTagValuesResults(results []QueryResult) (*tempopb.SearchTagValuesResponse, error) {
	valueSet := make(map[string]struct{})
	var combinedMetrics *tempopb.MetadataMetrics

	for _, result := range results {
		if result.Error != nil {
			level.Warn(c.logger).Log("msg", "instance returned error for tag values", "instance", result.Instance, "err", result.Error)
			continue
		}

		if result.Response == nil {
			level.Warn(c.logger).Log("msg", "instance returned nil response for tag values", "instance", result.Instance)
			continue
		}

		// Skip 404s - instance doesn't have this tag
		if result.Response.StatusCode == http.StatusNotFound {
			level.Debug(c.logger).Log("msg", "instance returned 404 for tag values", "instance", result.Instance)
			continue
		}

		if result.Response.StatusCode != http.StatusOK {
			level.Warn(c.logger).Log("msg", "instance returned non-OK status for tag values", "instance", result.Instance, "status", result.Response.StatusCode)
			continue
		}

		// Body is already read by QueryAllInstances
		if len(result.Body) == 0 {
			level.Debug(c.logger).Log("msg", "instance returned empty body for tag values", "instance", result.Instance)
			continue
		}

		// Parse the JSON response
		var valuesResp tempopb.SearchTagValuesResponse
		if err := jsonpb.UnmarshalString(string(result.Body), &valuesResp); err != nil {
			level.Warn(c.logger).Log("msg", "failed to unmarshal tag values response", "instance", result.Instance, "err", err)
			continue
		}

		// Add values to set (deduplication)
		for _, value := range valuesResp.TagValues {
			valueSet[value] = struct{}{}
		}

		// Combine metrics
		if valuesResp.Metrics != nil {
			combinedMetrics = combineMetadataMetrics(combinedMetrics, valuesResp.Metrics)
		}

		level.Debug(c.logger).Log("msg", "combined tag values from instance", "instance", result.Instance, "values", len(valuesResp.TagValues))
	}

	// Convert set to sorted slice
	tagValues := make([]string, 0, len(valueSet))
	for value := range valueSet {
		tagValues = append(tagValues, value)
	}
	sort.Strings(tagValues)

	return &tempopb.SearchTagValuesResponse{
		TagValues: tagValues,
		Metrics:   combinedMetrics,
	}, nil
}

// CombineTagValuesV2Results combines SearchTagValuesV2Response results from multiple Tempo instances
func (c *Combiner) CombineTagValuesV2Results(results []QueryResult) (*tempopb.SearchTagValuesV2Response, error) {
	// Map of tag value to TagValue struct (to preserve type information)
	valueMap := make(map[string]*tempopb.TagValue)
	var combinedMetrics *tempopb.MetadataMetrics

	for _, result := range results {
		if result.Error != nil {
			level.Warn(c.logger).Log("msg", "instance returned error for tag values v2", "instance", result.Instance, "err", result.Error)
			continue
		}

		if result.Response == nil {
			level.Warn(c.logger).Log("msg", "instance returned nil response for tag values v2", "instance", result.Instance)
			continue
		}

		// Skip 404s - instance doesn't have this tag
		if result.Response.StatusCode == http.StatusNotFound {
			level.Debug(c.logger).Log("msg", "instance returned 404 for tag values v2", "instance", result.Instance)
			continue
		}

		if result.Response.StatusCode != http.StatusOK {
			level.Warn(c.logger).Log("msg", "instance returned non-OK status for tag values v2", "instance", result.Instance, "status", result.Response.StatusCode)
			continue
		}

		// Body is already read by QueryAllInstances
		if len(result.Body) == 0 {
			level.Debug(c.logger).Log("msg", "instance returned empty body for tag values v2", "instance", result.Instance)
			continue
		}

		// Parse the JSON response
		var valuesResp tempopb.SearchTagValuesV2Response
		if err := jsonpb.UnmarshalString(string(result.Body), &valuesResp); err != nil {
			level.Warn(c.logger).Log("msg", "failed to unmarshal tag values v2 response", "instance", result.Instance, "err", err)
			continue
		}

		// Add values to map (deduplication by value, preserve type)
		for _, tv := range valuesResp.TagValues {
			if tv != nil {
				// If we haven't seen this value, add it
				// If we have, keep the existing one (first one wins)
				if _, exists := valueMap[tv.Value]; !exists {
					valueMap[tv.Value] = tv
				}
			}
		}

		// Combine metrics
		if valuesResp.Metrics != nil {
			combinedMetrics = combineMetadataMetrics(combinedMetrics, valuesResp.Metrics)
		}

		level.Debug(c.logger).Log("msg", "combined tag values v2 from instance", "instance", result.Instance, "values", len(valuesResp.TagValues))
	}

	// Convert map to sorted slice
	tagValues := make([]*tempopb.TagValue, 0, len(valueMap))
	for _, tv := range valueMap {
		tagValues = append(tagValues, tv)
	}
	// Sort by value
	sort.Slice(tagValues, func(i, j int) bool {
		return tagValues[i].Value < tagValues[j].Value
	})

	return &tempopb.SearchTagValuesV2Response{
		TagValues: tagValues,
		Metrics:   combinedMetrics,
	}, nil
}

// combineMetadataMetrics combines MetadataMetrics from multiple responses
func combineMetadataMetrics(existing, incoming *tempopb.MetadataMetrics) *tempopb.MetadataMetrics {
	if existing == nil {
		return incoming
	}
	if incoming == nil {
		return existing
	}

	// Sum up the metrics
	return &tempopb.MetadataMetrics{
		TotalBlocks:     existing.TotalBlocks + incoming.TotalBlocks,
		TotalJobs:       existing.TotalJobs + incoming.TotalJobs,
		CompletedJobs:   existing.CompletedJobs + incoming.CompletedJobs,
		TotalBlockBytes: existing.TotalBlockBytes + incoming.TotalBlockBytes,
	}
}
