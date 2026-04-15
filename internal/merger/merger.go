// Copyright (c) 2025-2026 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package merger

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/TrueTickets/api-aggregator/internal/transformer"
	"github.com/TrueTickets/api-aggregator/internal/types"
)

// MergeResult holds the result of merging backend responses
type MergeResult struct {
	Data         interface{}
	AllCompleted bool
	StatusCode   int
}

// Merger handles merging responses from multiple backends
type Merger struct {
	tracer      trace.Tracer
	transformer *transformer.Transformer
}

// Config holds merger configuration
type Config struct {
	Tracer trace.Tracer
}

// New creates a new merger instance
func New(cfg Config) *Merger {
	return &Merger{
		tracer: cfg.Tracer,
		transformer: transformer.New(transformer.Config{
			Tracer: cfg.Tracer,
		}),
	}
}

// Merge merges multiple backend responses into a single response
func (m *Merger) Merge(responses []types.BackendResponse) MergeResult {
	ctx := context.Background()
	_, span := m.tracer.Start(ctx, "merge_responses")
	defer span.End()

	result := make(map[string]interface{})
	allCompleted := true
	successfulResponses := 0

	// Track status codes: use the lowest status code among responses
	// that returned data. If all responses had no data (e.g. all 204),
	// use the status code from the no-data responses.
	dataStatusCode := 0
	noDataStatusCode := 0

	for _, resp := range responses {
		if resp.Error != nil {
			allCompleted = false
			continue
		}

		if resp.Data == nil {
			// Track status code from responses with no data (e.g. 204)
			if resp.StatusCode > 0 && (noDataStatusCode == 0 || resp.StatusCode < noDataStatusCode) {
				noDataStatusCode = resp.StatusCode
			}
			continue
		}

		successfulResponses++

		// Track status code from responses with data
		if resp.StatusCode > 0 && (dataStatusCode == 0 || resp.StatusCode < dataStatusCode) {
			dataStatusCode = resp.StatusCode
		}

		// Process the response data through transformations
		processedData := m.transformer.Transform(ctx, resp.Data, resp.Backend)

		// Merge the processed data based on backend configuration
		switch {
		case resp.Backend.Concat != "":
			// If concat is specified, append the data to an array under the specified key
			m.appendToArray(result, resp.Backend.Concat, processedData)
		case resp.Backend.Group != "":
			// If group is specified, wrap the data in a group
			result[resp.Backend.Group] = processedData
		default:
			// If we have only one successful response and no grouping,
			// return the processed data directly (could be array, object, etc.)
			if successfulResponses == 1 && len(responses) == 1 {
				return MergeResult{
					Data:         processedData,
					AllCompleted: allCompleted,
					StatusCode:   m.resolveStatusCode(dataStatusCode, noDataStatusCode),
				}
			}
			// Otherwise, merge into result map
			m.mergeIntoResult(result, processedData)
		}
	}

	return MergeResult{
		Data:         result,
		AllCompleted: allCompleted,
		StatusCode:   m.resolveStatusCode(dataStatusCode, noDataStatusCode),
	}
}

// resolveStatusCode determines the final status code from backend responses.
// Priority: status code from responses with data > status code from responses
// without data > default 200 OK.
func (m *Merger) resolveStatusCode(dataStatusCode, noDataStatusCode int) int {
	if dataStatusCode > 0 {
		return dataStatusCode
	}
	if noDataStatusCode > 0 {
		return noDataStatusCode
	}
	return http.StatusOK
}

// appendToArray appends data to an array under the specified key in the result map
// If data is an array, its elements are spread into the target array (flattened)
func (m *Merger) appendToArray(result map[string]interface{}, key string, data interface{}) {
	existing, exists := result[key]

	// Handle the case where data is an array - spread its elements
	if dataArray, ok := data.([]interface{}); ok {
		if !exists {
			// Create new array with the flattened data
			targetArray := make([]interface{}, len(dataArray))
			copy(targetArray, dataArray)
			result[key] = targetArray
			return
		}

		// If existing value is an array, append all elements from data array
		if existingArray, ok := existing.([]interface{}); ok {
			result[key] = append(existingArray, dataArray...)
		} else {
			// If existing value is not an array, convert it to an array and append all elements
			newArray := []interface{}{existing}
			result[key] = append(newArray, dataArray...)
		}
		return
	}

	// Handle the case where data is not an array - add as single element
	if !exists {
		// Create new array with the data
		result[key] = []interface{}{data}
		return
	}

	// If existing value is an array, append to it
	if existingArray, ok := existing.([]interface{}); ok {
		result[key] = append(existingArray, data)
	} else {
		// If existing value is not an array, convert it to an array and append
		result[key] = []interface{}{existing, data}
	}
}

// mergeIntoResult merges data into the result map using deep merge
func (m *Merger) mergeIntoResult(result map[string]interface{}, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	for key, value := range dataMap {
		m.deepMerge(result, key, value)
	}
}

// deepMerge performs a deep merge of the value into the result map at the given key
func (m *Merger) deepMerge(result map[string]interface{}, key string, value interface{}) {
	existing, exists := result[key]
	if !exists {
		result[key] = value
		return
	}

	// Handle different merge scenarios based on types
	switch existingVal := existing.(type) {
	case map[string]interface{}:
		if valueMap, ok := value.(map[string]interface{}); ok {
			// Merge two maps recursively
			for k, v := range valueMap {
				m.deepMerge(existingVal, k, v)
			}
		} else {
			// Replace if types don't match
			result[key] = value
		}
	case []interface{}:
		if valueSlice, ok := value.([]interface{}); ok {
			// Combine arrays
			result[key] = append(existingVal, valueSlice...)
		} else {
			// Replace if types don't match
			result[key] = value
		}
	default:
		// For primitive types or mismatched types, replace
		result[key] = value
	}
}
