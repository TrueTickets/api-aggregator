// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package transformer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/TrueTickets/api-aggregator/internal/config"
)

func TestTransformer_ApplyTarget(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	transformer := New(Config{Tracer: tracer})

	tests := []struct {
		name     string
		data     interface{}
		target   string
		expected interface{}
	}{
		{
			name: "simple target",
			data: map[string]interface{}{
				"data": map[string]interface{}{
					"id":   1,
					"name": "John",
				},
			},
			target: "data",
			expected: map[string]interface{}{
				"id":   1,
				"name": "John",
			},
		},
		{
			name: "nested target",
			data: map[string]interface{}{
				"response": map[string]interface{}{
					"data": map[string]interface{}{
						"user": map[string]interface{}{
							"id":   1,
							"name": "John",
						},
					},
				},
			},
			target: "response.data.user",
			expected: map[string]interface{}{
				"id":   1,
				"name": "John",
			},
		},
		{
			name: "target not found",
			data: map[string]interface{}{
				"id":   1,
				"name": "John",
			},
			target: "nonexistent",
			expected: map[string]interface{}{
				"id":   1,
				"name": "John",
			},
		},
		{
			name:     "non-map data",
			data:     "string data",
			target:   "data",
			expected: "string data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.ApplyTarget(tt.data, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformer_ApplyFiltering(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	transformer := New(Config{Tracer: tracer})

	data := map[string]interface{}{
		"id":     1,
		"name":   "John",
		"email":  "john@example.com",
		"age":    30,
		"secret": "hidden",
	}

	tests := []struct {
		name     string
		allow    []string
		deny     []string
		expected map[string]interface{}
	}{
		{
			name:  "allow list",
			allow: []string{"id", "name"},
			deny:  nil,
			expected: map[string]interface{}{
				"id":   1,
				"name": "John",
			},
		},
		{
			name:  "deny list",
			allow: nil,
			deny:  []string{"secret", "age"},
			expected: map[string]interface{}{
				"id":    1,
				"name":  "John",
				"email": "john@example.com",
			},
		},
		{
			name:  "allow takes precedence",
			allow: []string{"id", "name"},
			deny:  []string{"name"},
			expected: map[string]interface{}{
				"id":   1,
				"name": "John",
			},
		},
		{
			name:  "no filters",
			allow: nil,
			deny:  nil,
			expected: map[string]interface{}{
				"id":     1,
				"name":   "John",
				"email":  "john@example.com",
				"age":    30,
				"secret": "hidden",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.ApplyFiltering(data, tt.allow, tt.deny)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformer_ApplyMapping(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	transformer := New(Config{Tracer: tracer})

	data := map[string]interface{}{
		"id":       1,
		"fullName": "John Doe",
		"email":    "john@example.com",
	}

	mapping := map[string]string{
		"fullName": "name",
		"email":    "emailAddress",
	}

	expected := map[string]interface{}{
		"id":           1,
		"name":         "John Doe",
		"emailAddress": "john@example.com",
	}

	result := transformer.ApplyMapping(data, mapping)
	assert.Equal(t, expected, result)
}

func TestTransformer_Transform(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	transformer := New(Config{Tracer: tracer})
	ctx := context.Background()

	data := map[string]interface{}{
		"response": map[string]interface{}{
			"data": map[string]interface{}{
				"id":       1,
				"fullName": "John Doe",
				"email":    "john@example.com",
				"age":      30,
				"secret":   "hidden",
			},
		},
	}

	backend := config.Backend{
		Target:  "response.data",
		Allow:   []string{"id", "fullName", "email"},
		Mapping: map[string]string{"fullName": "name"},
	}

	expected := map[string]interface{}{
		"id":    1,
		"name":  "John Doe",
		"email": "john@example.com",
	}

	result := transformer.Transform(ctx, data, backend)
	assert.Equal(t, expected, result)
}

func TestTransformer_NestedFieldOperations(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	transformer := New(Config{Tracer: tracer})

	data := map[string]interface{}{
		"user": map[string]interface{}{
			"personal": map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			"contact": map[string]interface{}{
				"email": "john@example.com",
			},
		},
	}

	t.Run("get nested field", func(t *testing.T) {
		result := transformer.GetNestedField(data, "user.personal.name")
		assert.Equal(t, "John", result)

		result = transformer.GetNestedField(data, "user.contact.email")
		assert.Equal(t, "john@example.com", result)

		result = transformer.GetNestedField(data, "user.nonexistent")
		assert.Nil(t, result)
	})

	t.Run("set nested field", func(t *testing.T) {
		testData := make(map[string]interface{})
		transformer.SetNestedField(testData, "user.personal.name", "Jane")
		transformer.SetNestedField(testData, "user.contact.phone", "123-456-7890")

		expected := map[string]interface{}{
			"user": map[string]interface{}{
				"personal": map[string]interface{}{
					"name": "Jane",
				},
				"contact": map[string]interface{}{
					"phone": "123-456-7890",
				},
			},
		}

		assert.Equal(t, expected, testData)
	})

	t.Run("delete nested field", func(t *testing.T) {
		testData := map[string]interface{}{
			"user": map[string]interface{}{
				"personal": map[string]interface{}{
					"name": "John",
					"age":  30,
				},
				"contact": map[string]interface{}{
					"email": "john@example.com",
				},
			},
		}

		transformer.DeleteNestedField(testData, "user.personal.age")
		transformer.DeleteNestedField(testData, "user.contact")

		expected := map[string]interface{}{
			"user": map[string]interface{}{
				"personal": map[string]interface{}{
					"name": "John",
				},
			},
		}

		assert.Equal(t, expected, testData)
	})
}

func TestTransformer_EdgeCases(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	transformer := New(Config{Tracer: tracer})
	ctx := context.Background()

	tests := []struct {
		name     string
		data     interface{}
		backend  config.Backend
		expected interface{}
	}{
		{
			name:     "nil data",
			data:     nil,
			backend:  config.Backend{},
			expected: nil,
		},
		{
			name:     "non-map data with transformations",
			data:     "string data",
			backend:  config.Backend{Target: "data", Allow: []string{"field"}},
			expected: "string data",
		},
		{
			name: "empty transformations",
			data: map[string]interface{}{
				"id":   1,
				"name": "John",
			},
			backend: config.Backend{},
			expected: map[string]interface{}{
				"id":   1,
				"name": "John",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.Transform(ctx, tt.data, tt.backend)
			assert.Equal(t, tt.expected, result)
		})
	}
}
