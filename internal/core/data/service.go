package data

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/bootstrap"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/common"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/models"
)

// CoreDataService handles event and reading management
type CoreDataService struct {
	logger *logrus.Logger
	events map[string]models.Event
	mutex  sync.RWMutex
}

// NewCoreDataService creates a new core data service
func NewCoreDataService(logger *logrus.Logger) *CoreDataService {
	return &CoreDataService{
		logger: logger,
		events: make(map[string]models.Event),
	}
}

// Initialize implements the BootstrapHandler interface
func (s *CoreDataService) Initialize(ctx context.Context, wg *sync.WaitGroup, dic *bootstrap.DIContainer) bool {
	s.logger.Info("Initializing Core Data Service")
	
	// Add service to DI container
	dic.Add("CoreDataService", s)
	
	s.logger.Info("Core Data Service initialization completed")
	return true
}

// AddRoutes adds core data specific routes
func (s *CoreDataService) AddRoutes(router *mux.Router) {
	// Event routes
	router.HandleFunc(common.ApiEventRoute, s.addEvent).Methods("POST")
	router.HandleFunc(common.ApiEventRoute+"/all", s.getAllEvents).Methods("GET")
	router.HandleFunc(common.ApiEventByIdRoute, s.getEventById).Methods("GET")
	router.HandleFunc(common.ApiEventByIdRoute, s.deleteEventById).Methods("DELETE")
	router.HandleFunc(common.ApiEventByDeviceNameRoute, s.getEventsByDeviceName).Methods("GET")
	
	s.logger.Info("Core Data routes registered")
}

// addEvent handles POST /api/v3/event
func (s *CoreDataService) addEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var event models.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		s.logger.Errorf("Failed to decode event: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Generate ID and timestamps if not provided
	if event.Id == "" {
		event.Id = models.GenerateUUID()
	}
	if event.Created == 0 {
		event.Created = time.Now().UnixNano() / int64(time.Millisecond)
	}
	event.Modified = time.Now().UnixNano() / int64(time.Millisecond)
	
	// Generate IDs for readings
	for i := range event.Readings {
		if event.Readings[i].Id == "" {
			event.Readings[i].Id = models.GenerateUUID()
		}
		if event.Readings[i].Created == 0 {
			event.Readings[i].Created = event.Created
		}
		event.Readings[i].Modified = event.Modified
	}
	
	// Store event
	s.mutex.Lock()
	s.events[event.Id] = event
	s.mutex.Unlock()
	
	s.logger.Infof("Event created with ID: %s", event.Id)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusCreated,
		"id":         event.Id,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getAllEvents handles GET /api/v3/event/all
func (s *CoreDataService) getAllEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	// Parse query parameters
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")
	
	offset := 0
	limit := 20
	
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l <= 1000 {
			limit = l
		}
	}
	
	s.mutex.RLock()
	events := make([]models.Event, 0, len(s.events))
	for _, event := range s.events {
		events = append(events, event)
	}
	s.mutex.RUnlock()
	
	totalCount := len(events)
	
	// Apply pagination
	start := offset
	if start >= len(events) {
		start = len(events)
	}
	
	end := start + limit
	if end > len(events) {
		end = len(events)
	}
	
	paginatedEvents := events[start:end]
	
	response := map[string]interface{}{
		"apiVersion":  common.ServiceVersion,
		"statusCode":  http.StatusOK,
		"totalCount":  totalCount,
		"events":      paginatedEvents,
	}
	
	json.NewEncoder(w).Encode(response)
}

// getEventById handles GET /api/v3/event/id/{id}
func (s *CoreDataService) getEventById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.RLock()
	event, exists := s.events[id]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"event":      event,
	}
	
	json.NewEncoder(w).Encode(response)
}

// deleteEventById handles DELETE /api/v3/event/id/{id}
func (s *CoreDataService) deleteEventById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.Lock()
	_, exists := s.events[id]
	if exists {
		delete(s.events, id)
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}
	
	s.logger.Infof("Event deleted with ID: %s", id)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Event deleted successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// getEventsByDeviceName handles GET /api/v3/event/device/name/{name}
func (s *CoreDataService) getEventsByDeviceName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	deviceName := vars["name"]
	
	s.mutex.RLock()
	var deviceEvents []models.Event
	for _, event := range s.events {
		if event.DeviceName == deviceName {
			deviceEvents = append(deviceEvents, event)
		}
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":  common.ServiceVersion,
		"statusCode":  http.StatusOK,
		"totalCount":  len(deviceEvents),
		"events":      deviceEvents,
	}
	
	json.NewEncoder(w).Encode(response)
}