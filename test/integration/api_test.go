package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/models"
)

// EdgeXAPITestSuite provides integration tests for EdgeX APIs
type EdgeXAPITestSuite struct {
	suite.Suite
	baseURL    string
	httpClient *http.Client
}

// SetupSuite runs before all tests in the suite
func (suite *EdgeXAPITestSuite) SetupSuite() {
	suite.baseURL = "http://localhost" // Will be configured per service
	suite.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
}

// TestCoreDataAPIFlow tests the complete Core Data API workflow
func (suite *EdgeXAPITestSuite) TestCoreDataAPIFlow() {
	baseURL := suite.baseURL + ":59880"
	
	// Test ping endpoint
	resp, err := suite.httpClient.Get(baseURL + "/api/v3/ping")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	// Create test event
	event := models.Event{
		DeviceName:  "Integration-Test-Device",
		ProfileName: "Integration-Test-Profile",
		SourceName:  "integration-test",
		Readings: []models.Reading{
			{
				DeviceName:   "Integration-Test-Device",
				ResourceName: "Temperature",
				ProfileName:  "Integration-Test-Profile",
				ValueType:    "Float64",
				SimpleReading: models.SimpleReading{
					Value: "25.5",
					Units: "Celsius",
				},
			},
		},
	}
	
	// Test POST /api/v3/event
	eventJSON, err := json.Marshal(event)
	require.NoError(suite.T(), err)
	
	resp, err = suite.httpClient.Post(baseURL+"/api/v3/event", "application/json", bytes.NewBuffer(eventJSON))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	
	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	require.NoError(suite.T(), err)
	
	eventId := createResponse["id"].(string)
	assert.NotEmpty(suite.T(), eventId)
	
	// Test GET /api/v3/event/all
	resp, err = suite.httpClient.Get(baseURL + "/api/v3/event/all")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var getAllResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&getAllResponse)
	require.NoError(suite.T(), err)
	
	events := getAllResponse["events"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(events), 1)
	
	// Test GET /api/v3/event/id/{id}
	resp, err = suite.httpClient.Get(baseURL + "/api/v3/event/id/" + eventId)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	var getByIdResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&getByIdResponse)
	require.NoError(suite.T(), err)
	
	retrievedEvent := getByIdResponse["event"].(map[string]interface{})
	assert.Equal(suite.T(), event.DeviceName, retrievedEvent["deviceName"])
	
	// Test DELETE /api/v3/event/id/{id}
	req, err := http.NewRequest("DELETE", baseURL+"/api/v3/event/id/"+eventId, nil)
	require.NoError(suite.T(), err)
	
	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// TestCoreMetadataAPIFlow tests the complete Core Metadata API workflow
func (suite *EdgeXAPITestSuite) TestCoreMetadataAPIFlow() {
	baseURL := suite.baseURL + ":59881"
	
	// Test ping endpoint
	resp, err := suite.httpClient.Get(baseURL + "/api/v3/ping")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	
	// Create test device
	device := models.Device{
		Name:        "Integration-Test-Device",
		Description: "Device for integration testing",
		ProfileName: "Integration-Test-Profile",
		ServiceName: "integration-test-service",
		Protocols: map[string]models.ProtocolProperties{
			"modbus": {
				"Address": "192.168.1.100",
				"Port":    "502",
				"UnitID":  "1",
			},
		},
		Labels: []string{"test", "integration"},
	}
	
	// Test POST /api/v3/device
	deviceJSON, err := json.Marshal(device)
	require.NoError(suite.T(), err)
	
	resp, err = suite.httpClient.Post(baseURL+"/api/v3/device", "application/json", bytes.NewBuffer(deviceJSON))
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
	
	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	require.NoError(suite.T(), err)
	
	deviceId := createResponse["id"].(string)
	assert.NotEmpty(suite.T(), deviceId)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestEdgeXAPITestSuite(t *testing.T) {
	suite.Run(t, new(EdgeXAPITestSuite))
}