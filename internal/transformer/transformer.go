// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package transformer

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/TrueTickets/api-aggregator/internal/config"
)

// Transformer handles response transformations
type Transformer struct {
	tracer trace.Tracer
}

// Config holds transformer configuration
type Config struct {
	Tracer trace.Tracer
}

// New creates a new transformer instance
func New(cfg Config) *Transformer {
	return &Transformer{
		tracer: cfg.Tracer,
	}
}

// Transform applies all transformations to a response based on backend configuration
func (t *Transformer) Transform(ctx context.Context, data interface{}, backend config.Backend) interface{} {
	_, span := t.tracer.Start(ctx, "transform_response")
	defer span.End()

	if data == nil {
		return nil
	}

	// Apply transformations in order:
	// 1. Target (capturing) - extract nested data
	if backend.Target != "" {
		data = t.ApplyTarget(data, backend.Target)
	}

	// 2. Filtering - apply allow/deny lists
	if len(backend.Allow) > 0 || len(backend.Deny) > 0 {
		data = t.ApplyFiltering(data, backend.Allow, backend.Deny)
	}

	// 3. Mapping - rename fields
	if len(backend.Mapping) > 0 {
		data = t.ApplyMapping(data, backend.Mapping)
	}

	return data
}

// ApplyTarget extracts data from a nested target field
func (t *Transformer) ApplyTarget(data interface{}, target string) interface{} {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	// Navigate through nested path
	current := dataMap
	parts := strings.Split(target, ".")

	for _, part := range parts {
		if next, exists := current[part]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return next
			}
		} else {
			return data // Target not found, return original
		}
	}

	return current
}

// ApplyFiltering applies allow and deny filters to the data
func (t *Transformer) ApplyFiltering(data interface{}, allow, deny []string) interface{} {
	// Handle arrays by applying filtering to each element
	if dataArray, ok := data.([]interface{}); ok {
		result := make([]interface{}, len(dataArray))
		for i, item := range dataArray {
			result[i] = t.ApplyFiltering(item, allow, deny)
		}
		return result
	}

	// Handle maps
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})

	// If allow list is specified, only include those fields (deny is ignored)
	if len(allow) > 0 {
		for _, field := range allow {
			if value := t.GetNestedField(dataMap, field); value != nil {
				t.SetNestedField(result, field, value)
			}
		}
	} else {
		// Copy all fields first
		for key, value := range dataMap {
			result[key] = value
		}

		// Remove denied fields (only if no allow list)
		for _, field := range deny {
			t.DeleteNestedField(result, field)
		}
	}

	return result
}

// ApplyMapping renames fields according to the mapping configuration
func (t *Transformer) ApplyMapping(data interface{}, mapping map[string]string) interface{} {
	// Handle arrays by applying mapping to each element
	if dataArray, ok := data.([]interface{}); ok {
		result := make([]interface{}, len(dataArray))
		for i, item := range dataArray {
			result[i] = t.ApplyMapping(item, mapping)
		}
		return result
	}

	// Handle maps
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})

	// Create a set of fields that will be mapped
	mappedFields := make(map[string]bool)
	for oldField := range mapping {
		mappedFields[oldField] = true
	}

	// Copy fields that are NOT being mapped
	for key, value := range dataMap {
		if !mappedFields[key] {
			result[key] = value
		}
	}

	// Apply mappings - add new field names with values from old fields
	for oldField, newField := range mapping {
		if value := t.GetNestedField(dataMap, oldField); value != nil {
			t.SetNestedField(result, newField, value)
		}
	}

	return result
}

// GetNestedField gets a nested field using dot notation
func (t *Transformer) GetNestedField(data map[string]interface{}, field string) interface{} {
	parts := strings.Split(field, ".")
	current := data

	for i, part := range parts {
		if value, exists := current[part]; exists {
			if i == len(parts)-1 {
				return value
			}
			if nextMap, ok := value.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return nil
}

// SetNestedField sets a nested field using dot notation
func (t *Transformer) SetNestedField(data map[string]interface{}, field string, value interface{}) {
	parts := strings.Split(field, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if _, exists := current[part]; !exists {
				current[part] = make(map[string]interface{})
			}
			if nextMap, ok := current[part].(map[string]interface{}); ok {
				current = nextMap
			} else {
				return
			}
		}
	}
}

// DeleteNestedField deletes a nested field using dot notation
func (t *Transformer) DeleteNestedField(data map[string]interface{}, field string) {
	parts := strings.Split(field, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			delete(current, part)
		} else {
			if nextMap, ok := current[part].(map[string]interface{}); ok {
				current = nextMap
			} else {
				return
			}
		}
	}
}
