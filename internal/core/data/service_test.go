package data

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
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/models"
)

func TestNewCoreDataService(t *testing.T) {
	logger := logrus.New()
	service := NewCoreDataService(logger)
	
	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.events)
	assert.Equal(t, 0, len(service.events))
}

func TestCoreDataService_Initialize(t *testing.T) {
	logger := logrus.New()
	service := NewCoreDataService(logger)
	dic := bootstrap.NewDIContainer()
	var wg sync.WaitGroup
	
	result := service.Initialize(context.Background(), &wg, dic)
	
	assert.True(t, result)
	assert.NotNil(t, dic.Get("CoreDataService"))
}

func TestCoreDataService_AddEvent(t *testing.T) {
	tests := []struct {
		name         string
		event        models.Event
		expectedCode int
		expectError  bool
	}{
		{
			name: "Valid event",
			event: models.Event{
				DeviceName:  "TestDevice",
				ProfileName: "TestProfile",
				SourceName:  "TestSource",
				Readings: []models.Reading{
					{
						DeviceName:   "TestDevice",
						ResourceName: "Temperature",
						ProfileName:  "TestProfile",
						ValueType:    "Float64",
						SimpleReading: models.SimpleReading{
							Value: "22.5",
							Units: "Celsius",
						},
					},
				},
			},
			expectedCode: http.StatusCreated,
			expectError:  false,
		},
		{
			name:         "Invalid JSON",
			event:        models.Event{}, // Will be sent as invalid JSON
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			service := NewCoreDataService(logger)
			
			var body []byte
			var err error
			
			if tt.name == "Invalid JSON" {
				body = []byte("invalid json")
			} else {
				body, err = json.Marshal(tt.event)
				require.NoError(t, err)
			}
			
			req, err := http.NewRequest("POST", "/api/v3/event", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(service.addEvent)
			
			handler.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			if !tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Equal(t, "3.1.0", response["apiVersion"])
				assert.NotEmpty(t, response["id"])
				
				// Verify event was stored
				assert.Equal(t, 1, len(service.events))
			}
		})
	}
}

func TestCoreDataService_GetAllEvents(t *testing.T) {
	logger := logrus.New()
	service := NewCoreDataService(logger)
	
	// Add test events
	testEvents := []models.Event{
		{
			Id:          "event-1",
			DeviceName:  "Device1",
			ProfileName: "Profile1",
			SourceName:  "Source1",
			Created:     time.Now().UnixNano() / int64(time.Millisecond),
		},
		{
			Id:          "event-2",
			DeviceName:  "Device2",
			ProfileName: "Profile2",
			SourceName:  "Source2",
			Created:     time.Now().UnixNano() / int64(time.Millisecond),
		},
	}
	
	for _, event := range testEvents {
		service.events[event.Id] = event
	}
	
	tests := []struct {
		name           string
		offset         string
		limit          string
		expectedCount  int
		expectedTotal  int
		expectedCode   int
	}{
		{
			name:          "Get all events",
			offset:        "",
			limit:         "",
			expectedCount: 2,
			expectedTotal: 2,
			expectedCode:  http.StatusOK,
		},
		{
			name:          "Get events with limit",
			offset:        "0",
			limit:         "1",
			expectedCount: 1,
			expectedTotal: 2,
			expectedCode:  http.StatusOK,
		},
		{
			name:          "Get events with offset",
			offset:        "1",
			limit:         "10",
			expectedCount: 1,
			expectedTotal: 2,
			expectedCode:  http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v3/event/all"
			if tt.offset != "" || tt.limit != "" {
				url += "?"
				if tt.offset != "" {
					url += "offset=" + tt.offset
				}
				if tt.limit != "" {
					if tt.offset != "" {
						url += "&"
					}
					url += "limit=" + tt.limit
				}
			}
			
			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)
			
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(service.getAllEvents)
			
			handler.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, "3.1.0", response["apiVersion"])
			assert.Equal(t, float64(tt.expectedTotal), response["totalCount"])
			
			events := response["events"].([]interface{})
			assert.Equal(t, tt.expectedCount, len(events))
		})
	}
}

func TestCoreDataService_GetEventById(t *testing.T) {
	logger := logrus.New()
	service := NewCoreDataService(logger)
	
	testEvent := models.Event{
		Id:          "test-event-id",
		DeviceName:  "TestDevice",
		ProfileName: "TestProfile",
		SourceName:  "TestSource",
		Created:     time.Now().UnixNano() / int64(time.Millisecond),
	}
	service.events[testEvent.Id] = testEvent
	
	tests := []struct {
		name         string
		eventId      string
		expectedCode int
	}{
		{
			name:         "Get existing event",
			eventId:      "test-event-id",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Get non-existing event",
			eventId:      "non-existing-id",
			expectedCode: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v3/event/id/"+tt.eventId, nil)
			require.NoError(t, err)
			
			rr := httptest.NewRecorder()
			
			// Setup mux router to handle path parameters
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/event/id/{id}", service.getEventById).Methods("GET")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			if tt.expectedCode == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Equal(t, "3.1.0", response["apiVersion"])
				assert.NotNil(t, response["event"])
				
				event := response["event"].(map[string]interface{})
				assert.Equal(t, testEvent.Id, event["id"])
				assert.Equal(t, testEvent.DeviceName, event["deviceName"])
			}
		})
	}
}

func TestCoreDataService_DeleteEventById(t *testing.T) {
	logger := logrus.New()
	service := NewCoreDataService(logger)
	
	testEvent := models.Event{
		Id:          "test-event-id",
		DeviceName:  "TestDevice",
		ProfileName: "TestProfile",
		SourceName:  "TestSource",
		Created:     time.Now().UnixNano() / int64(time.Millisecond),
	}
	service.events[testEvent.Id] = testEvent
	
	tests := []struct {
		name         string
		eventId      string
		expectedCode int
	}{
		{
			name:         "Delete existing event",
			eventId:      "test-event-id",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Delete non-existing event",
			eventId:      "non-existing-id",
			expectedCode: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/v3/event/id/"+tt.eventId, nil)
			require.NoError(t, err)
			
			rr := httptest.NewRecorder()
			
			// Setup mux router to handle path parameters
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/event/id/{id}", service.deleteEventById).Methods("DELETE")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			if tt.expectedCode == http.StatusOK {
				// Verify event was deleted
				_, exists := service.events[tt.eventId]
				assert.False(t, exists)
			}
		})
	}
}

func TestCoreDataService_GetEventsByDeviceName(t *testing.T) {
	logger := logrus.New()
	service := NewCoreDataService(logger)
	
	// Add test events for different devices
	testEvents := []models.Event{
		{
			Id:          "event-1",
			DeviceName:  "Device1",
			ProfileName: "Profile1",
			SourceName:  "Source1",
			Created:     time.Now().UnixNano() / int64(time.Millisecond),
		},
		{
			Id:          "event-2",
			DeviceName:  "Device1",
			ProfileName: "Profile1",
			SourceName:  "Source1",
			Created:     time.Now().UnixNano() / int64(time.Millisecond),
		},
		{
			Id:          "event-3",
			DeviceName:  "Device2",
			ProfileName: "Profile2",
			SourceName:  "Source2",
			Created:     time.Now().UnixNano() / int64(time.Millisecond),
		},
	}
	
	for _, event := range testEvents {
		service.events[event.Id] = event
	}
	
	tests := []struct {
		name          string
		deviceName    string
		expectedCount int
		expectedCode  int
	}{
		{
			name:          "Get events for Device1",
			deviceName:    "Device1",
			expectedCount: 2,
			expectedCode:  http.StatusOK,
		},
		{
			name:          "Get events for Device2",
			deviceName:    "Device2",
			expectedCount: 1,
			expectedCode:  http.StatusOK,
		},
		{
			name:          "Get events for non-existing device",
			deviceName:    "NonExisting",
			expectedCount: 0,
			expectedCode:  http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v3/event/device/name/"+tt.deviceName, nil)
			require.NoError(t, err)
			
			rr := httptest.NewRecorder()
			
			// Setup mux router to handle path parameters
			router := mux.NewRouter()
			router.HandleFunc("/api/v3/event/device/name/{name}", service.getEventsByDeviceName).Methods("GET")
			
			router.ServeHTTP(rr, req)
			
			assert.Equal(t, tt.expectedCode, rr.Code)
			
			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)
			
			assert.Equal(t, "3.1.0", response["apiVersion"])
			assert.Equal(t, float64(tt.expectedCount), response["totalCount"])
			
			events := response["events"].([]interface{})
			assert.Equal(t, tt.expectedCount, len(events))
			
			// Verify all events belong to the correct device
			for _, eventInterface := range events {
				event := eventInterface.(map[string]interface{})
				assert.Equal(t, tt.deviceName, event["deviceName"])
			}
		})
	}
}

// Benchmark tests
func BenchmarkCoreDataService_AddEvent(b *testing.B) {
	logger := logrus.New()
	service := NewCoreDataService(logger)
	
	event := models.Event{
		DeviceName:  "BenchmarkDevice",
		ProfileName: "BenchmarkProfile",
		SourceName:  "BenchmarkSource",
		Readings: []models.Reading{
			{
				DeviceName:   "BenchmarkDevice",
				ResourceName: "Temperature",
				ProfileName:  "BenchmarkProfile",
				ValueType:    "Float64",
				SimpleReading: models.SimpleReading{
					Value: "22.5",
					Units: "Celsius",
				},
			},
		},
	}
	
	body, _ := json.Marshal(event)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v3/event", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.addEvent)
		
		handler.ServeHTTP(rr, req)
	}
}

func BenchmarkCoreDataService_GetAllEvents(b *testing.B) {
	logger := logrus.New()
	service := NewCoreDataService(logger)
	
	// Add some test data
	for i := 0; i < 1000; i++ {
		event := models.Event{
			Id:          models.GenerateUUID(),
			DeviceName:  "BenchmarkDevice",
			ProfileName: "BenchmarkProfile",
			SourceName:  "BenchmarkSource",
			Created:     time.Now().UnixNano() / int64(time.Millisecond),
		}
		service.events[event.Id] = event
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/api/v3/event/all", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.getAllEvents)
		
		handler.ServeHTTP(rr, req)
	}
}

// Thread safety tests
func TestCoreDataService_ConcurrentAccess(t *testing.T) {
	logger := logrus.New()
	service := NewCoreDataService(logger)
	
	// Test concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 100
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			event := models.Event{
				DeviceName:  "ConcurrentDevice",
				ProfileName: "ConcurrentProfile",
				SourceName:  "ConcurrentSource",
				Readings: []models.Reading{
					{
						DeviceName:   "ConcurrentDevice",
						ResourceName: "Temperature",
						ProfileName:  "ConcurrentProfile",
						ValueType:    "Float64",
						SimpleReading: models.SimpleReading{
							Value: "22.5",
							Units: "Celsius",
						},
					},
				},
			}
			
			body, _ := json.Marshal(event)
			req, _ := http.NewRequest("POST", "/api/v3/event", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(service.addEvent)
			
			handler.ServeHTTP(rr, req)
		}(i)
	}
	
	wg.Wait()
	
	// Verify all events were added
	assert.Equal(t, numGoroutines, len(service.events))
}