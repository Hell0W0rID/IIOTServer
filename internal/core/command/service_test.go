package command

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/bootstrap"
)

func TestNewCoreCommandService(t *testing.T) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	
	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.commandResponses)
	assert.Equal(t, 0, len(service.commandResponses))
}

func TestCoreCommandService_Initialize(t *testing.T) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	dic := bootstrap.NewDIContainer()
	var wg sync.WaitGroup
	
	result := service.Initialize(context.Background(), &wg, dic)
	
	assert.True(t, result)
	assert.NotNil(t, dic.Get("CoreCommandService"))
}

func TestCoreCommandService_GetDeviceCommands(t *testing.T) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	
	tests := []struct {
		name         string
		deviceName   string
		expectedCode int
	}{
		{
			name:         "Valid device name",
			deviceName:   "TestDevice",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Another device name",
			deviceName:   "AnotherDevice",
			expectedCode: http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v3/device/name/"+tt.deviceName+"/command", nil)
			require.NoError(t, err)
			
			rr := httptest.NewRecorder()
			
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/device/name/{name}/command", service.getDeviceCommands).Methods("GET")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, "3.1.0", response["apiVersion"])
			assert.Equal(t, tt.deviceName, response["deviceName"])
			assert.NotNil(t, response["commands"])
			
			commands := response["commands"].([]interface{})
			assert.Greater(t, len(commands), 0)
			
			// Verify command structure
			for _, cmdInterface := range commands {
				cmd := cmdInterface.(map[string]interface{})
				assert.NotEmpty(t, cmd["name"])
				assert.NotNil(t, cmd["get"])
				assert.NotNil(t, cmd["set"])
				assert.NotEmpty(t, cmd["path"])
				assert.NotNil(t, cmd["parameters"])
			}
		})
	}
}

func TestCoreCommandService_IssueGetCommand(t *testing.T) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	
	tests := []struct {
		name         string
		deviceName   string
		commandName  string
		expectedCode int
	}{
		{
			name:         "Get Temperature command",
			deviceName:   "TestDevice",
			commandName:  "Temperature",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Get Humidity command",
			deviceName:   "TestDevice",
			commandName:  "Humidity",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Get SetPoint command",
			deviceName:   "TestDevice",
			commandName:  "SetPoint",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Unknown command",
			deviceName:   "TestDevice",
			commandName:  "UnknownCommand",
			expectedCode: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v3/device/name/"+tt.deviceName+"/command/"+tt.commandName, nil)
			require.NoError(t, err)
			
			rr := httptest.NewRecorder()
			
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/device/name/{name}/command/{command}", service.issueGetCommand).Methods("GET")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			if tt.expectedCode == http.StatusOK {
				var response map[string]interface{}
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Equal(t, "3.1.0", response["apiVersion"])
				assert.NotNil(t, response["event"])
				
				event := response["event"].(map[string]interface{})
				assert.Equal(t, tt.deviceName, event["deviceName"])
				assert.NotEmpty(t, event["id"])
				assert.NotNil(t, event["readings"])
				
				readings := event["readings"].([]interface{})
				assert.Equal(t, 1, len(readings))
				
				reading := readings[0].(map[string]interface{})
				assert.Equal(t, tt.commandName, reading["resourceName"])
				assert.NotNil(t, reading["value"])
			}
		})
	}
}

func TestCoreCommandService_IssueSetCommand(t *testing.T) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	
	tests := []struct {
		name         string
		deviceName   string
		commandName  string
		parameters   map[string]interface{}
		expectedCode int
	}{
		{
			name:        "Set valid SetPoint command",
			deviceName:  "TestDevice",
			commandName: "SetPoint",
			parameters: map[string]interface{}{
				"value": "25.0",
				"units": "Celsius",
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Set Temperature command (not supported)",
			deviceName:   "TestDevice",
			commandName:  "Temperature",
			parameters:   map[string]interface{}{},
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			name:         "Set Humidity command (not supported)",
			deviceName:   "TestDevice",
			commandName:  "Humidity",
			parameters:   map[string]interface{}{},
			expectedCode: http.StatusMethodNotAllowed,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.parameters)
			require.NoError(t, err)
			
			req, err := http.NewRequest("PUT", "/api/v3/device/name/"+tt.deviceName+"/command/"+tt.commandName, bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/device/name/{name}/command/{command}", service.issueSetCommand).Methods("PUT")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			if tt.expectedCode == http.StatusOK {
				var response map[string]interface{}
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Equal(t, "3.1.0", response["apiVersion"])
				assert.NotEmpty(t, response["commandId"])
				assert.Contains(t, response["message"], "successfully")
			}
		})
	}
}

func TestCoreCommandService_InvalidJSON(t *testing.T) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	
	req, err := http.NewRequest("PUT", "/api/v3/device/name/TestDevice/command/SetPoint", bytes.NewBuffer([]byte("invalid json")))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	
	router := mux.NewRouter()
	router.HandleFunc("/api/v3/device/name/{name}/command/{command}", service.issueSetCommand).Methods("PUT")
	
	router.ServeHTTP(rr, req)
	
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// Benchmark tests
func BenchmarkCoreCommandService_IssueGetCommand(b *testing.B) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	
	router := mux.NewRouter()
	router.HandleFunc("/api/v3/device/name/{name}/command/{command}", service.issueGetCommand).Methods("GET")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v3/device/name/TestDevice/command/Temperature", nil)
		rr := httptest.NewRecorder()
		
		router.ServeHTTP(rr, req)
	}
}

func BenchmarkCoreCommandService_IssueSetCommand(b *testing.B) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	
	router := mux.NewRouter()
	router.HandleFunc("/api/v3/device/name/{name}/command/{command}", service.issueSetCommand).Methods("PUT")
	
	parameters := map[string]interface{}{
		"value": "25.0",
		"units": "Celsius",
	}
	body, _ := json.Marshal(parameters)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("PUT", "/api/v3/device/name/TestDevice/command/SetPoint", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		
		router.ServeHTTP(rr, req)
	}
}

// Thread safety tests
func TestCoreCommandService_ConcurrentCommandExecution(t *testing.T) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	
	var wg sync.WaitGroup
	numGoroutines := 100
	wg.Add(numGoroutines)
	
	// Test concurrent GET commands
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			req, _ := http.NewRequest("GET", "/api/v3/device/name/TestDevice/command/Temperature", nil)
			rr := httptest.NewRecorder()
			
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/device/name/{name}/command/{command}", service.issueGetCommand).Methods("GET")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, http.StatusOK, rr.Code)
		}(i)
	}
	
	wg.Wait()
	
	// Verify command responses were stored
	assert.Equal(t, numGoroutines, len(service.commandResponses))
}

func TestCoreCommandService_ConcurrentSetCommands(t *testing.T) {
	logger := logrus.New()
	service := NewCoreCommandService(logger)
	
	var wg sync.WaitGroup
	numGoroutines := 50
	wg.Add(numGoroutines)
	
	// Test concurrent SET commands
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			parameters := map[string]interface{}{
				"value": "25.0",
				"units": "Celsius",
			}
			body, _ := json.Marshal(parameters)
			
			req, _ := http.NewRequest("PUT", "/api/v3/device/name/TestDevice/command/SetPoint", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/device/name/{name}/command/{command}", service.issueSetCommand).Methods("PUT")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, http.StatusOK, rr.Code)
		}(i)
	}
	
	wg.Wait()
	
	// Verify command responses were stored
	assert.Equal(t, numGoroutines, len(service.commandResponses))
}