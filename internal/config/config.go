// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the entire service configuration
type Config struct {
	// Global timeout for all endpoints (can be overridden per endpoint)
	Timeout time.Duration `yaml:"timeout"`

	// Service configuration
	Port            string        `yaml:"port"`
	LogLevel        string        `yaml:"log_level"`
	LogFormat       string        `yaml:"log_format"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`

	// OpenTelemetry configuration
	TracingEnabled  bool   `yaml:"tracing_enabled"`
	TracingEndpoint string `yaml:"tracing_endpoint"`
	MetricsEnabled  bool   `yaml:"metrics_enabled"`
	ServiceName     string `yaml:"service_name"`

	// Endpoints configuration
	Endpoints []Endpoint `yaml:"endpoints"`
}

// Endpoint represents a single API endpoint configuration
type Endpoint struct {
	// Endpoint path (can include path parameters like {user})
	Endpoint string `yaml:"endpoint"`

	// HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string `yaml:"method"`

	// Timeout for this specific endpoint (overrides global timeout)
	Timeout time.Duration `yaml:"timeout"`

	// Default encoding for backends (json, xml, yaml)
	Encoding string `yaml:"encoding"`

	// Backend services to aggregate
	Backends []Backend `yaml:"backends"`
}

// Backend represents a backend service configuration
type Backend struct {
	// URL pattern to call (can include path parameters) - optional, defaults to endpoint path
	URLPattern string `yaml:"url_pattern,omitempty"`

	// Encoding for this specific backend (overrides endpoint encoding)
	Encoding string `yaml:"encoding"`

	// Host for this backend
	Host string `yaml:"host"`

	// Headers to remove before making the request to this backend
	RemoveHeaders []string `yaml:"remove_headers,omitempty"`

	// Response transformations
	Group   string            `yaml:"group,omitempty"`
	Target  string            `yaml:"target,omitempty"`
	Allow   []string          `yaml:"allow,omitempty"`
	Deny    []string          `yaml:"deny,omitempty"`
	Mapping map[string]string `yaml:"mapping,omitempty"`
	Concat  string            `yaml:"concat,omitempty"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	cfg.setDefaults()

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (*Config, error) {
	configPath := os.Getenv("API_AGGREGATOR_CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Override with environment variables if present
	if port := os.Getenv("API_AGGREGATOR_PORT"); port != "" {
		cfg.Port = port
	}
	if logLevel := os.Getenv("API_AGGREGATOR_LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if logFormat := os.Getenv("API_AGGREGATOR_LOG_FORMAT"); logFormat != "" {
		cfg.LogFormat = logFormat
	}
	if tracingEnabled := os.Getenv("API_AGGREGATOR_TRACING_ENABLED"); tracingEnabled != "" {
		cfg.TracingEnabled, err = strconv.ParseBool(tracingEnabled)
		if err != nil {
			return nil, fmt.Errorf("failed to parse API_AGGREGATOR_TRACING_ENABLED as boolean: %w", err)
		}
	}
	if tracingEndpoint := os.Getenv("API_AGGREGATOR_TRACING_ENDPOINT"); tracingEndpoint != "" {
		cfg.TracingEndpoint = tracingEndpoint
	}
	if metricsEnabled := os.Getenv("API_AGGREGATOR_METRICS_ENABLED"); metricsEnabled != "" {
		cfg.MetricsEnabled, err = strconv.ParseBool(metricsEnabled)
		if err != nil {
			return nil, fmt.Errorf("failed to parse API_AGGREGATOR_METRICS_ENABLED as boolean: %w", err)
		}
	}
	if serviceName := os.Getenv("API_AGGREGATOR_SERVICE_NAME"); serviceName != "" {
		cfg.ServiceName = serviceName
	}

	return cfg, nil
}

const (
	defaultPort            = "8080"
	defaultLogLevel        = "info"
	defaultLogFormat       = "json"
	defaultShutdownTimeout = 15 * time.Second
	defaultServiceName     = "api-aggregator"
	defaultTimeout         = 10 * time.Second
	defaultMethod          = "GET"
	defaultEncoding        = "json"
)

// setDefaults sets default values for configuration
func (c *Config) setDefaults() {
	c.setServiceDefaults()
	c.setTimeoutDefaults()
	c.setEndpointDefaults()
}

func (c *Config) setServiceDefaults() {
	if c.Port == "" {
		c.Port = defaultPort
	}
	if c.LogLevel == "" {
		c.LogLevel = defaultLogLevel
	}
	if c.LogFormat == "" {
		c.LogFormat = defaultLogFormat
	}
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = defaultShutdownTimeout
	}
	if c.ServiceName == "" {
		c.ServiceName = defaultServiceName
	}
}

func (c *Config) setTimeoutDefaults() {
	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
}

func (c *Config) setEndpointDefaults() {
	for i := range c.Endpoints {
		endpoint := &c.Endpoints[i]
		c.setEndpointTimeout(endpoint)
		c.setEndpointMethod(endpoint)
		c.setEndpointEncoding(endpoint)
		c.setBackendDefaults(endpoint)
	}
}

func (c *Config) setEndpointTimeout(endpoint *Endpoint) {
	if endpoint.Timeout == 0 {
		endpoint.Timeout = c.Timeout
	}
}

func (c *Config) setEndpointMethod(endpoint *Endpoint) {
	if endpoint.Method == "" {
		endpoint.Method = defaultMethod
	}
}

func (c *Config) setEndpointEncoding(endpoint *Endpoint) {
	if endpoint.Encoding == "" {
		endpoint.Encoding = defaultEncoding
	}
}

func (c *Config) setBackendDefaults(endpoint *Endpoint) {
	for j := range endpoint.Backends {
		backend := &endpoint.Backends[j]
		if backend.Encoding == "" {
			backend.Encoding = endpoint.Encoding
		}
		// Set URL pattern to endpoint path if not specified
		if backend.URLPattern == "" {
			backend.URLPattern = endpoint.Endpoint
		}
	}
}

// validate validates the configuration
func (c *Config) validate() error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}

	validEncodings := c.getValidEncodings()

	for i, endpoint := range c.Endpoints {
		if err := c.validateEndpoint(i, endpoint, validEncodings); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) getValidEncodings() map[string]bool {
	return map[string]bool{"json": true, "xml": true, "yaml": true}
}

func (c *Config) validateEndpoint(i int, endpoint Endpoint, validEncodings map[string]bool) error {
	if endpoint.Endpoint == "" {
		return fmt.Errorf("endpoint %d: endpoint path is required", i)
	}

	if len(endpoint.Backends) == 0 {
		return fmt.Errorf("endpoint %s: at least one backend is required", endpoint.Endpoint)
	}

	if !validEncodings[endpoint.Encoding] {
		return fmt.Errorf("endpoint %s: invalid encoding %s", endpoint.Endpoint, endpoint.Encoding)
	}

	return c.validateBackends(endpoint, validEncodings)
}

func (c *Config) validateBackends(endpoint Endpoint, validEncodings map[string]bool) error {
	for j, backend := range endpoint.Backends {
		if err := c.validateBackend(endpoint.Endpoint, j, backend, validEncodings); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) validateBackend(endpointName string, j int, backend Backend, validEncodings map[string]bool) error {
	// Note: URLPattern is now optional and defaults are set in setBackendDefaults

	if backend.Host == "" {
		return fmt.Errorf("endpoint %s, backend %d: host is required", endpointName, j)
	}

	if !validEncodings[backend.Encoding] {
		return fmt.Errorf("endpoint %s, backend %d: invalid encoding %s",
			endpointName, j, backend.Encoding)
	}

	return nil
}
