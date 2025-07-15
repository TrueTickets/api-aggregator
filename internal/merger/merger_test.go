// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package merger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/TrueTickets/api-aggregator/internal/config"
	"github.com/TrueTickets/api-aggregator/internal/types"
)

func TestMerger_Merge(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	merger := New(Config{Tracer: tracer})

	tests := []struct {
		name      string
		responses []types.BackendResponse
		expected  map[string]interface{}
		completed bool
	}{
		{
			name: "simple merge",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"id":   1,
						"name": "John",
					},
					Error: nil,
				},
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"email": "john@example.com",
						"age":   30,
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"id":    1,
				"name":  "John",
				"email": "john@example.com",
				"age":   30,
			},
			completed: true,
		},
		{
			name: "merge with group",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{
						Group: "user",
					},
					Data: map[string]interface{}{
						"id":   1,
						"name": "John",
					},
					Error: nil,
				},
				{
					Backend: config.Backend{
						Group: "profile",
					},
					Data: map[string]interface{}{
						"email": "john@example.com",
						"age":   30,
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"id":   1,
					"name": "John",
				},
				"profile": map[string]interface{}{
					"email": "john@example.com",
					"age":   30,
				},
			},
			completed: true,
		},
		{
			name: "merge with error",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"id":   1,
						"name": "John",
					},
					Error: nil,
				},
				{
					Backend: config.Backend{},
					Data:    nil,
					Error:   assert.AnError,
				},
			},
			expected: map[string]interface{}{
				"id":   1,
				"name": "John",
			},
			completed: false,
		},
		{
			name: "deep merge arrays",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"users": []interface{}{
							map[string]interface{}{"id": 1, "name": "John"},
							map[string]interface{}{"id": 2, "name": "Jane"},
						},
						"tags": []interface{}{"tag1", "tag2"},
					},
					Error: nil,
				},
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"users": []interface{}{
							map[string]interface{}{"id": 3, "name": "Bob"},
						},
						"tags": []interface{}{"tag3", "tag4"},
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"id": 1, "name": "John"},
					map[string]interface{}{"id": 2, "name": "Jane"},
					map[string]interface{}{"id": 3, "name": "Bob"},
				},
				"tags": []interface{}{"tag1", "tag2", "tag3", "tag4"},
			},
			completed: true,
		},
		{
			name: "deep merge nested objects",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"user": map[string]interface{}{
							"id":   1,
							"name": "John",
							"profile": map[string]interface{}{
								"email": "john@example.com",
							},
						},
					},
					Error: nil,
				},
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"user": map[string]interface{}{
							"age": 30,
							"profile": map[string]interface{}{
								"phone": "123-456-7890",
							},
						},
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"id":   1,
					"name": "John",
					"age":  30,
					"profile": map[string]interface{}{
						"email": "john@example.com",
						"phone": "123-456-7890",
					},
				},
			},
			completed: true,
		},
		{
			name: "deep merge mixed types - replace on conflict",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"value": []interface{}{"a", "b"},
					},
					Error: nil,
				},
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"value": "string_value",
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"value": "string_value",
			},
			completed: true,
		},
		{
			name: "concat functionality - multiple responses to same array",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{
						Concat: "items",
					},
					Data: map[string]interface{}{
						"id":   1,
						"name": "John",
					},
					Error: nil,
				},
				{
					Backend: config.Backend{
						Concat: "items",
					},
					Data: map[string]interface{}{
						"id":   2,
						"name": "Jane",
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{
						"id":   1,
						"name": "John",
					},
					map[string]interface{}{
						"id":   2,
						"name": "Jane",
					},
				},
			},
			completed: true,
		},
		{
			name: "concat functionality - different concat keys",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{
						Concat: "users",
					},
					Data: map[string]interface{}{
						"id":   1,
						"name": "John",
					},
					Error: nil,
				},
				{
					Backend: config.Backend{
						Concat: "posts",
					},
					Data: map[string]interface{}{
						"id":    101,
						"title": "First Post",
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"id":   1,
						"name": "John",
					},
				},
				"posts": []interface{}{
					map[string]interface{}{
						"id":    101,
						"title": "First Post",
					},
				},
			},
			completed: true,
		},
		{
			name: "concat with existing non-array value - converts to array",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"items": "single_item",
					},
					Error: nil,
				},
				{
					Backend: config.Backend{
						Concat: "items",
					},
					Data: map[string]interface{}{
						"id":   1,
						"name": "John",
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"items": []interface{}{
					"single_item",
					map[string]interface{}{
						"id":   1,
						"name": "John",
					},
				},
			},
			completed: true,
		},
		{
			name: "concat mixed with group and regular merge",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{
						Concat: "items",
					},
					Data: map[string]interface{}{
						"id":   1,
						"name": "Item 1",
					},
					Error: nil,
				},
				{
					Backend: config.Backend{
						Group: "metadata",
					},
					Data: map[string]interface{}{
						"version": "1.0",
						"author":  "system",
					},
					Error: nil,
				},
				{
					Backend: config.Backend{},
					Data: map[string]interface{}{
						"timestamp": "2024-01-01T00:00:00Z",
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{
						"id":   1,
						"name": "Item 1",
					},
				},
				"metadata": map[string]interface{}{
					"version": "1.0",
					"author":  "system",
				},
				"timestamp": "2024-01-01T00:00:00Z",
			},
			completed: true,
		},
		{
			name: "concat functionality - array flattening",
			responses: []types.BackendResponse{
				{
					Backend: config.Backend{
						Concat: "items",
					},
					Data: []interface{}{
						map[string]interface{}{
							"id":   1,
							"name": "John",
						},
						map[string]interface{}{
							"id":   2,
							"name": "Jane",
						},
					},
					Error: nil,
				},
				{
					Backend: config.Backend{
						Concat: "items",
					},
					Data: []interface{}{
						map[string]interface{}{
							"id":   3,
							"name": "Bob",
						},
					},
					Error: nil,
				},
			},
			expected: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{
						"id":   1,
						"name": "John",
					},
					map[string]interface{}{
						"id":   2,
						"name": "Jane",
					},
					map[string]interface{}{
						"id":   3,
						"name": "Bob",
					},
				},
			},
			completed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, completed := merger.Merge(tt.responses)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.completed, completed)
		})
	}
}
