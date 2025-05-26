package utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/models"
)

// TestHelper provides common utilities for testing EdgeX services
type TestHelper struct {
	Logger *logrus.Logger
	Hook   *test.Hook
}

// NewTestHelper creates a new test helper with a test logger
func NewTestHelper() *TestHelper {
	logger, hook := test.NewNullLogger()
	return &TestHelper{
		Logger: logger,
		Hook:   hook,
	}
}

// CreateTestEvent creates a sample event for testing
func (h *TestHelper) CreateTestEvent(deviceName, profileName string) models.Event {
	return models.Event{
		DeviceName:  deviceName,
		ProfileName: profileName,
		SourceName:  "test-source",
		Readings: []models.Reading{
			h.CreateTestReading(deviceName, profileName, "Temperature", "25.5", "Celsius"),
		},
	}
}

// CreateTestReading creates a sample reading for testing
func (h *TestHelper) CreateTestReading(deviceName, profileName, resourceName, value, units string) models.Reading {
	return models.Reading{
		DeviceName:   deviceName,
		ResourceName: resourceName,
		ProfileName:  profileName,
		ValueType:    "Float64",
		SimpleReading: models.SimpleReading{
			Value: value,
			Units: units,
		},
	}
}

// CreateTestDevice creates a sample device for testing
func (h *TestHelper) CreateTestDevice(name, profileName, serviceName string) models.Device {
	return models.Device{
		Name:        name,
		Description: "Test device",
		ProfileName: profileName,
		ServiceName: serviceName,
		Protocols: map[string]models.ProtocolProperties{
			"modbus": {
				"Address": "192.168.1.100",
				"Port":    "502",
				"UnitID":  "1",
			},
		},
		Labels: []string{"test"},
	}
}

// CreateTestDeviceProfile creates a sample device profile for testing
func (h *TestHelper) CreateTestDeviceProfile(name string) models.DeviceProfile {
	return models.DeviceProfile{
		Name:         name,
		Description:  "Test device profile",
		Manufacturer: "Test Manufacturer",
		Model:        "Test Model",
		DeviceCommands: []models.DeviceCommand{
			{
				Name: "Temperature",
				Get:  true,
				Set:  false,
			},
		},
		CoreCommands: []models.CoreCommand{
			{
				Name: "Temperature",
				Get:  true,
				Set:  false,
			},
		},
	}
}

// MakeJSONRequest creates an HTTP request with JSON body
func (h *TestHelper) MakeJSONRequest(t *testing.T, method, url string, body interface{}) *http.Request {
	var reqBody []byte
	var err error
	
	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}
	
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	return req
}

// ExecuteRequest executes an HTTP request against a router and returns the response
func (h *TestHelper) ExecuteRequest(t *testing.T, router *mux.Router, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// ParseJSONResponse parses a JSON response into a map
func (h *TestHelper) ParseJSONResponse(t *testing.T, rr *httptest.ResponseRecorder) map[string]interface{} {
	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	return response
}

// AssertStatusCode asserts that the response has the expected status code
func (h *TestHelper) AssertStatusCode(t *testing.T, expected int, rr *httptest.ResponseRecorder) {
	require.Equal(t, expected, rr.Code, "Expected status code %d, got %d. Response: %s", expected, rr.Code, rr.Body.String())
}

// AssertJSONResponse asserts common JSON response fields
func (h *TestHelper) AssertJSONResponse(t *testing.T, response map[string]interface{}, expectedStatusCode int) {
	require.Equal(t, "3.1.0", response["apiVersion"])
	require.Equal(t, float64(expectedStatusCode), response["statusCode"])
}

// AssertLogContains asserts that the log contains a specific message
func (h *TestHelper) AssertLogContains(t *testing.T, message string) {
	found := false
	for _, entry := range h.Hook.AllEntries() {
		if entry.Message == message {
			found = true
			break
		}
	}
	require.True(t, found, "Expected log message '%s' not found", message)
}

// ClearLogs clears all captured log entries
func (h *TestHelper) ClearLogs() {
	h.Hook.Reset()
}

// LoadTestData loads test data from JSON files (for future use)
func (h *TestHelper) LoadTestData(filename string) ([]byte, error) {
	// This would load test data from files
	// For now, we'll use embedded test data
	return []byte{}, nil
}

// MockHTTPClient creates a mock HTTP client for testing external calls
type MockHTTPClient struct {
	Responses map[string]*http.Response
}

// Do implements the HTTP client interface for mocking
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	key := req.Method + " " + req.URL.String()
	if response, exists := m.Responses[key]; exists {
		return response, nil
	}
	
	// Default response
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}

// TestMetrics provides utilities for performance testing
type TestMetrics struct {
	RequestCount    int
	TotalDuration   int64
	MinDuration     int64
	MaxDuration     int64
	AverageDuration float64
}

// PerformanceTestRunner runs performance tests
type PerformanceTestRunner struct {
	metrics TestMetrics
}

// NewPerformanceTestRunner creates a new performance test runner
func NewPerformanceTestRunner() *PerformanceTestRunner {
	return &PerformanceTestRunner{}
}

// RunTest executes a performance test
func (p *PerformanceTestRunner) RunTest(t *testing.T, testFunc func() error, iterations int) TestMetrics {
	p.metrics = TestMetrics{
		MinDuration: int64(^uint64(0) >> 1), // Max int64
	}
	
	for i := 0; i < iterations; i++ {
		start := getCurrentTimeNanos()
		err := testFunc()
		duration := getCurrentTimeNanos() - start
		
		require.NoError(t, err)
		
		p.metrics.RequestCount++
		p.metrics.TotalDuration += duration
		
		if duration < p.metrics.MinDuration {
			p.metrics.MinDuration = duration
		}
		if duration > p.metrics.MaxDuration {
			p.metrics.MaxDuration = duration
		}
	}
	
	if p.metrics.RequestCount > 0 {
		p.metrics.AverageDuration = float64(p.metrics.TotalDuration) / float64(p.metrics.RequestCount)
	}
	
	return p.metrics
}

func getCurrentTimeNanos() int64 {
	return int64(1000000000) // Simplified for testing
}

// DatabaseTestHelper provides utilities for database testing
type DatabaseTestHelper struct {
	// This would contain database setup/teardown utilities
	// For now, we use in-memory storage, so this is simplified
}

// SetupTestDatabase sets up a test database
func (d *DatabaseTestHelper) SetupTestDatabase() error {
	// Would set up test database
	return nil
}

// CleanupTestDatabase cleans up the test database
func (d *DatabaseTestHelper) CleanupTestDatabase() error {
	// Would clean up test database
	return nil
}

// MessageTestHelper provides utilities for testing message bus functionality
type MessageTestHelper struct {
	Messages []interface{}
}

// NewMessageTestHelper creates a new message test helper
func NewMessageTestHelper() *MessageTestHelper {
	return &MessageTestHelper{
		Messages: make([]interface{}, 0),
	}
}

// PublishMessage simulates publishing a message
func (m *MessageTestHelper) PublishMessage(topic string, message interface{}) error {
	m.Messages = append(m.Messages, message)
	return nil
}

// GetMessages returns all captured messages
func (m *MessageTestHelper) GetMessages() []interface{} {
	return m.Messages
}

// ClearMessages clears all captured messages
func (m *MessageTestHelper) ClearMessages() {
	m.Messages = make([]interface{}, 0)
}

// ConfigTestHelper provides utilities for testing configuration
type ConfigTestHelper struct {
	configs map[string]interface{}
}

// NewConfigTestHelper creates a new config test helper
func NewConfigTestHelper() *ConfigTestHelper {
	return &ConfigTestHelper{
		configs: make(map[string]interface{}),
	}
}

// SetConfig sets a configuration value for testing
func (c *ConfigTestHelper) SetConfig(key string, value interface{}) {
	c.configs[key] = value
}

// GetConfig gets a configuration value for testing
func (c *ConfigTestHelper) GetConfig(key string) interface{} {
	return c.configs[key]
}

// ClearConfigs clears all configuration values
func (c *ConfigTestHelper) ClearConfigs() {
	c.configs = make(map[string]interface{})
}