// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

const (
	statusCodeBadRequest = 400
	// Encoding constants
	encodingJSON = "json"
	encodingXML  = "xml"
	encodingYAML = "yaml"
)

// Client handles HTTP requests to backend services
type Client struct {
	httpClient *http.Client
	tracer     trace.Tracer
	logger     zerolog.Logger
}

// Config holds client configuration
type Config struct {
	HTTPClient *http.Client
	Tracer     trace.Tracer
	Logger     zerolog.Logger
}

// RequestConfig holds configuration for a single request
type RequestConfig struct {
	Method   string
	URL      string
	Encoding string
	Headers  map[string]string
	Body     io.Reader
}

// New creates a new client instance
func New(cfg Config) *Client {
	return &Client{
		httpClient: cfg.HTTPClient,
		tracer:     cfg.Tracer,
		logger:     cfg.Logger,
	}
}

// Request makes an HTTP request and returns the parsed response
func (c *Client) Request(ctx context.Context, cfg RequestConfig) (interface{}, error) {
	ctx, span := c.tracer.Start(ctx, fmt.Sprintf("backend_request_%s", cfg.URL))
	defer span.End()

	// Read and log request body
	bodyBytes, err := c.readAndLogRequestBody(&cfg)
	if err != nil {
		return nil, err
	}

	// Reset body for the actual request if we read it
	if len(bodyBytes) > 0 {
		cfg.Body = bytes.NewReader(bodyBytes)
	}

	// Create and configure HTTP request
	req, err := c.createHTTPRequest(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Make request and handle response
	return c.makeRequestAndHandleResponse(req, cfg)
}

// readAndLogRequestBody reads the request body and logs the outgoing request
func (c *Client) readAndLogRequestBody(cfg *RequestConfig) ([]byte, error) {
	var bodyBytes []byte
	if cfg.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(cfg.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	// Debug logging for outgoing requests
	if c.logger.GetLevel() <= zerolog.DebugLevel {
		c.logOutgoingRequest(*cfg, bodyBytes)
	}

	return bodyBytes, nil
}

// logOutgoingRequest logs debug information for outgoing requests
func (c *Client) logOutgoingRequest(cfg RequestConfig, bodyBytes []byte) {
	logEvent := c.logger.Debug().
		Str("method", cfg.Method).
		Str("url", cfg.URL).
		Str("encoding", cfg.Encoding)

	if len(cfg.Headers) > 0 {
		logEvent = logEvent.Interface("headers", cfg.Headers)
	}

	if len(bodyBytes) > 0 {
		logEvent = logEvent.RawJSON("body", bodyBytes)
	}

	logEvent.Msg("outgoing backend request")
}

// createHTTPRequest creates and configures the HTTP request
func (c *Client) createHTTPRequest(ctx context.Context, cfg RequestConfig) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, cfg.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set custom headers
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	// Set Accept header based on encoding
	c.setAcceptHeader(req, cfg.Encoding)

	// Set Content-Type if we have a body and it's not already set
	if cfg.Body != nil && req.Header.Get("Content-Type") == "" {
		c.setContentTypeHeader(req, cfg.Encoding)
	}

	return req, nil
}

// setAcceptHeader sets the Accept header based on encoding
func (c *Client) setAcceptHeader(req *http.Request, encoding string) {
	switch encoding {
	case encodingJSON:
		req.Header.Set("Accept", "application/json")
	case encodingXML:
		req.Header.Set("Accept", "application/xml")
	case encodingYAML:
		req.Header.Set("Accept", "application/yaml")
	}
}

// setContentTypeHeader sets the Content-Type header based on encoding
func (c *Client) setContentTypeHeader(req *http.Request, encoding string) {
	switch encoding {
	case encodingJSON:
		req.Header.Set("Content-Type", "application/json")
	case encodingXML:
		req.Header.Set("Content-Type", "application/xml")
	case encodingYAML:
		req.Header.Set("Content-Type", "application/yaml")
	}
}

// makeRequestAndHandleResponse executes the HTTP request and processes the response
func (c *Client) makeRequestAndHandleResponse(req *http.Request, cfg RequestConfig) (interface{}, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn().Err(closeErr).Msg("Failed to close response body")
		}
	}()

	// Read response body (Go HTTP client handles decompression automatically)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response at trace level
	if c.logger.GetLevel() <= zerolog.TraceLevel {
		c.logBackendResponse(cfg, resp, body)
	}

	// Check for error status codes
	if err := c.checkStatusCode(resp.StatusCode, body); err != nil {
		return nil, err
	}

	// Parse and return response
	return c.parseResponse(body, cfg.Encoding)
}

// logBackendResponse logs trace information for backend responses
func (c *Client) logBackendResponse(cfg RequestConfig, resp *http.Response, body []byte) {
	logEvent := c.logger.Trace().
		Str("method", cfg.Method).
		Str("url", cfg.URL).
		Int("status_code", resp.StatusCode).
		Interface("response_headers", resp.Header)

	if len(body) > 0 {
		logEvent = logEvent.RawJSON("response_body", body)
	}

	logEvent.Msg("backend response received")
}

// checkStatusCode validates the HTTP status code and returns an error if needed
func (c *Client) checkStatusCode(statusCode int, body []byte) error {
	if statusCode >= statusCodeBadRequest {
		if len(body) == 0 {
			return fmt.Errorf("backend returned status %d with empty body", statusCode)
		}
		return fmt.Errorf("backend returned status %d: %s", statusCode, string(body))
	}
	return nil
}

// parseResponse parses the response body based on the encoding
func (c *Client) parseResponse(body []byte, encoding string) (interface{}, error) {
	if len(body) == 0 {
		return nil, nil
	}

	var data interface{}
	var err error

	switch encoding {
	case encodingJSON:
		err = json.Unmarshal(body, &data)
	case encodingXML:
		err = xml.Unmarshal(body, &data)
	case encodingYAML:
		err = yaml.Unmarshal(body, &data)
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encoding)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse %s response: %w", encoding, err)
	}

	return data, nil
}

// Get makes a GET request
func (c *Client) Get(ctx context.Context, url, encoding string, headers map[string]string) (interface{}, error) {
	return c.Request(ctx, RequestConfig{
		Method:   "GET",
		URL:      url,
		Encoding: encoding,
		Headers:  headers,
	})
}

// Post makes a POST request
func (c *Client) Post(ctx context.Context, url, encoding string, headers map[string]string,
	body io.Reader) (interface{}, error) {
	return c.Request(ctx, RequestConfig{
		Method:   "POST",
		URL:      url,
		Encoding: encoding,
		Headers:  headers,
		Body:     body,
	})
}

// Put makes a PUT request
func (c *Client) Put(ctx context.Context, url, encoding string, headers map[string]string,
	body io.Reader) (interface{}, error) {
	return c.Request(ctx, RequestConfig{
		Method:   "PUT",
		URL:      url,
		Encoding: encoding,
		Headers:  headers,
		Body:     body,
	})
}

// Delete makes a DELETE request
func (c *Client) Delete(ctx context.Context, url, encoding string, headers map[string]string) (interface{}, error) {
	return c.Request(ctx, RequestConfig{
		Method:   "DELETE",
		URL:      url,
		Encoding: encoding,
		Headers:  headers,
	})
}
