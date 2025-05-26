package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/bootstrap"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/common"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/models"
)

func TestNewCoreMetadataService(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.devices)
	assert.NotNil(t, service.deviceProfiles)
	assert.NotNil(t, service.deviceServices)
	assert.Equal(t, 0, len(service.devices))
}

func TestCoreMetadataService_Initialize(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	dic := bootstrap.NewDIContainer()
	var wg sync.WaitGroup
	
	result := service.Initialize(context.Background(), &wg, dic)
	
	assert.True(t, result)
	assert.NotNil(t, dic.Get("CoreMetadataService"))
}

func TestCoreMetadataService_AddDevice(t *testing.T) {
	tests := []struct {
		name         string
		device       models.Device
		expectedCode int
		expectError  bool
	}{
		{
			name: "Valid device",
			device: models.Device{
				Name:        "TestDevice",
				Description: "Test device description",
				ProfileName: "TestProfile",
				ServiceName: "TestService",
				Protocols: map[string]models.ProtocolProperties{
					"modbus": {
						"Address": "192.168.1.100",
						"Port":    "502",
					},
				},
			},
			expectedCode: http.StatusCreated,
			expectError:  false,
		},
		{
			name:         "Invalid JSON",
			device:       models.Device{},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			service := NewCoreMetadataService(logger)
			
			var body []byte
			var err error
			
			if tt.name == "Invalid JSON" {
				body = []byte("invalid json")
			} else {
				body, err = json.Marshal(tt.device)
				require.NoError(t, err)
			}
			
			req, err := http.NewRequest("POST", "/api/v3/device", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(service.addDevice)
			
			handler.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			if !tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Equal(t, "3.1.0", response["apiVersion"])
				assert.NotEmpty(t, response["id"])
				
				// Verify device was stored
				assert.Equal(t, 1, len(service.devices))
				
				// Verify defaults were set
				for _, device := range service.devices {
					assert.Equal(t, common.Unlocked, device.AdminState)
					assert.Equal(t, common.Up, device.OperatingState)
					assert.NotEmpty(t, device.Id)
					assert.NotZero(t, device.Created)
				}
			}
		})
	}
}

func TestCoreMetadataService_GetAllDevices(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	// Add test devices
	testDevices := []models.Device{
		{
			Id:          "device-1",
			Name:        "Device1",
			Description: "Test device 1",
			ProfileName: "Profile1",
			ServiceName: "Service1",
			AdminState:  common.Unlocked,
			Created:     time.Now().UnixNano() / int64(time.Millisecond),
		},
		{
			Id:          "device-2",
			Name:        "Device2",
			Description: "Test device 2",
			ProfileName: "Profile2",
			ServiceName: "Service2",
			AdminState:  common.Unlocked,
			Created:     time.Now().UnixNano() / int64(time.Millisecond),
		},
	}
	
	for _, device := range testDevices {
		service.devices[device.Id] = device
	}
	
	req, err := http.NewRequest("GET", "/api/v3/device/all", nil)
	require.NoError(t, err)
	
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(service.getAllDevices)
	
	handler.ServeHTTP(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "3.1.0", response["apiVersion"])
	assert.Equal(t, float64(2), response["totalCount"])
	
	devices := response["devices"].([]interface{})
	assert.Equal(t, 2, len(devices))
}

func TestCoreMetadataService_GetDeviceById(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	testDevice := models.Device{
		Id:          "test-device-id",
		Name:        "TestDevice",
		Description: "Test device",
		ProfileName: "TestProfile",
		ServiceName: "TestService",
		AdminState:  common.Unlocked,
		Created:     time.Now().UnixNano() / int64(time.Millisecond),
	}
	service.devices[testDevice.Id] = testDevice
	
	tests := []struct {
		name         string
		deviceId     string
		expectedCode int
	}{
		{
			name:         "Get existing device",
			deviceId:     "test-device-id",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Get non-existing device",
			deviceId:     "non-existing-id",
			expectedCode: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v3/device/id/"+tt.deviceId, nil)
			require.NoError(t, err)
			
			rr := httptest.NewRecorder()
			
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/device/id/{id}", service.getDeviceById).Methods("GET")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			if tt.expectedCode == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Equal(t, "3.1.0", response["apiVersion"])
				assert.NotNil(t, response["device"])
				
				device := response["device"].(map[string]interface{})
				assert.Equal(t, testDevice.Id, device["id"])
				assert.Equal(t, testDevice.Name, device["name"])
			}
		})
	}
}

func TestCoreMetadataService_GetDeviceByName(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	testDevice := models.Device{
		Id:          "test-device-id",
		Name:        "TestDevice",
		Description: "Test device",
		ProfileName: "TestProfile",
		ServiceName: "TestService",
		AdminState:  common.Unlocked,
		Created:     time.Now().UnixNano() / int64(time.Millisecond),
	}
	service.devices[testDevice.Id] = testDevice
	
	tests := []struct {
		name         string
		deviceName   string
		expectedCode int
	}{
		{
			name:         "Get existing device by name",
			deviceName:   "TestDevice",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Get non-existing device by name",
			deviceName:   "NonExistingDevice",
			expectedCode: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v3/device/name/"+tt.deviceName, nil)
			require.NoError(t, err)
			
			rr := httptest.NewRecorder()
			
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/device/name/{name}", service.getDeviceByName).Methods("GET")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			if tt.expectedCode == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Equal(t, "3.1.0", response["apiVersion"])
				assert.NotNil(t, response["device"])
				
				device := response["device"].(map[string]interface{})
				assert.Equal(t, testDevice.Name, device["name"])
			}
		})
	}
}

func TestCoreMetadataService_UpdateDevice(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	// Create initial device
	originalDevice := models.Device{
		Id:          "test-device-id",
		Name:        "OriginalDevice",
		Description: "Original description",
		ProfileName: "OriginalProfile",
		ServiceName: "OriginalService",
		AdminState:  common.Unlocked,
		Created:     time.Now().UnixNano() / int64(time.Millisecond),
	}
	service.devices[originalDevice.Id] = originalDevice
	
	updatedDevice := models.Device{
		Name:        "UpdatedDevice",
		Description: "Updated description",
		ProfileName: "UpdatedProfile",
		ServiceName: "UpdatedService",
		AdminState:  common.Locked,
	}
	
	body, err := json.Marshal(updatedDevice)
	require.NoError(t, err)
	
	req, err := http.NewRequest("PUT", "/api/v3/device/id/test-device-id", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	
	router := mux.NewRouter()
	router.HandleFunc("/api/v3/device/id/{id}", service.updateDevice).Methods("PUT")
	
	router.ServeHTTP(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	
	// Verify device was updated
	device := service.devices["test-device-id"]
	assert.Equal(t, "UpdatedDevice", device.Name)
	assert.Equal(t, "Updated description", device.Description)
	assert.Equal(t, originalDevice.Created, device.Created) // Created should remain unchanged
	assert.NotEqual(t, originalDevice.Modified, device.Modified) // Modified should be updated
}

func TestCoreMetadataService_DeleteDevice(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	testDevice := models.Device{
		Id:          "test-device-id",
		Name:        "TestDevice",
		Description: "Test device",
		ProfileName: "TestProfile",
		ServiceName: "TestService",
		AdminState:  common.Unlocked,
		Created:     time.Now().UnixNano() / int64(time.Millisecond),
	}
	service.devices[testDevice.Id] = testDevice
	
	tests := []struct {
		name         string
		deviceId     string
		expectedCode int
	}{
		{
			name:         "Delete existing device",
			deviceId:     "test-device-id",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Delete non-existing device",
			deviceId:     "non-existing-id",
			expectedCode: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/v3/device/id/"+tt.deviceId, nil)
			require.NoError(t, err)
			
			rr := httptest.NewRecorder()
			
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/device/id/{id}", service.deleteDevice).Methods("DELETE")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			if tt.expectedCode == http.StatusOK && tt.deviceId == "test-device-id" {
				// Verify device was deleted
				_, exists := service.devices[tt.deviceId]
				assert.False(t, exists)
			}
		})
	}
}

func TestCoreMetadataService_AddDeviceProfile(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	deviceProfile := models.DeviceProfile{
		Name:         "TestProfile",
		Description:  "Test device profile",
		Manufacturer: "TestManufacturer",
		Model:        "TestModel",
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
	
	body, err := json.Marshal(deviceProfile)
	require.NoError(t, err)
	
	req, err := http.NewRequest("POST", "/api/v3/deviceprofile", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(service.addDeviceProfile)
	
	handler.ServeHTTP(rr, req)
	
	assert.Equal(t, http.StatusCreated, rr.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "3.1.0", response["apiVersion"])
	assert.NotEmpty(t, response["id"])
	
	// Verify device profile was stored
	assert.Equal(t, 1, len(service.deviceProfiles))
}

func TestCoreMetadataService_AddDeviceService(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	deviceService := models.DeviceService{
		Name:        "TestService",
		Description: "Test device service",
		BaseAddress: "http://localhost:59999",
		Labels:      []string{"test", "service"},
	}
	
	body, err := json.Marshal(deviceService)
	require.NoError(t, err)
	
	req, err := http.NewRequest("POST", "/api/v3/deviceservice", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(service.addDeviceService)
	
	handler.ServeHTTP(rr, req)
	
	assert.Equal(t, http.StatusCreated, rr.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "3.1.0", response["apiVersion"])
	assert.NotEmpty(t, response["id"])
	
	// Verify device service was stored
	assert.Equal(t, 1, len(service.deviceServices))
	
	// Verify defaults were set
	for _, ds := range service.deviceServices {
		assert.Equal(t, common.Unlocked, ds.AdminState)
		assert.Equal(t, common.Up, ds.OperatingState)
	}
}

// Benchmark tests
func BenchmarkCoreMetadataService_AddDevice(b *testing.B) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	device := models.Device{
		Name:        "BenchmarkDevice",
		Description: "Benchmark device",
		ProfileName: "BenchmarkProfile",
		ServiceName: "BenchmarkService",
		Protocols: map[string]models.ProtocolProperties{
			"modbus": {
				"Address": "192.168.1.100",
				"Port":    "502",
			},
		},
	}
	
	body, _ := json.Marshal(device)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v3/device", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.addDevice)
		
		handler.ServeHTTP(rr, req)
	}
}

// Thread safety tests
func TestCoreMetadataService_ConcurrentDeviceOperations(t *testing.T) {
	logger := logrus.New()
	service := NewCoreMetadataService(logger)
	
	var wg sync.WaitGroup
	numGoroutines := 50
	
	// Concurrent device additions
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			device := models.Device{
				Name:        "ConcurrentDevice",
				Description: "Concurrent test device",
				ProfileName: "ConcurrentProfile",
				ServiceName: "ConcurrentService",
			}
			
			body, _ := json.Marshal(device)
			req, _ := http.NewRequest("POST", "/api/v3/device", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(service.addDevice)
			
			handler.ServeHTTP(rr, req)
		}(i)
	}
	
	wg.Wait()
	
	// Verify all devices were added
	assert.Equal(t, numGoroutines, len(service.devices))
}