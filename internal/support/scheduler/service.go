package scheduler

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/bootstrap"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/common"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/models"
)

// ScheduleEvent represents a scheduled job
type ScheduleEvent struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Schedule    string `json:"schedule"`    // Cron expression
	Addressable string `json:"addressable"` // Target endpoint
	Parameters  string `json:"parameters"`
	Service     string `json:"service"`
	AdminState  string `json:"adminState"`
	Created     int64  `json:"created"`
	Modified    int64  `json:"modified"`
}

// ScheduleAction represents a scheduled action
type ScheduleAction struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Schedule    string `json:"schedule"`
	Target      string `json:"target"`
	Protocol    string `json:"protocol"`
	HTTPMethod  string `json:"httpMethod"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	Path        string `json:"path"`
	Parameters  string `json:"parameters"`
	User        string `json:"user"`
	Password    string `json:"password"`
	AdminState  string `json:"adminState"`
	Created     int64  `json:"created"`
	Modified    int64  `json:"modified"`
}

// SupportSchedulerService handles scheduled jobs and actions
type SupportSchedulerService struct {
	logger          *logrus.Logger
	scheduleEvents  map[string]ScheduleEvent
	scheduleActions map[string]ScheduleAction
	runningJobs     map[string]*time.Ticker
	mutex           sync.RWMutex
}

// NewSupportSchedulerService creates a new support scheduler service
func NewSupportSchedulerService(logger *logrus.Logger) *SupportSchedulerService {
	return &SupportSchedulerService{
		logger:          logger,
		scheduleEvents:  make(map[string]ScheduleEvent),
		scheduleActions: make(map[string]ScheduleAction),
		runningJobs:     make(map[string]*time.Ticker),
	}
}

// Initialize implements the BootstrapHandler interface
func (s *SupportSchedulerService) Initialize(ctx context.Context, wg *sync.WaitGroup, dic *bootstrap.DIContainer) bool {
	s.logger.Info("Initializing Support Scheduler Service")
	
	// Add service to DI container
	dic.Add("SupportSchedulerService", s)
	
	s.logger.Info("Support Scheduler Service initialization completed")
	return true
}

// AddRoutes adds support scheduler specific routes
func (s *SupportSchedulerService) AddRoutes(router *mux.Router) {
	// Schedule Event routes
	router.HandleFunc("/api/v3/scheduleevent", s.addScheduleEvent).Methods("POST")
	router.HandleFunc("/api/v3/scheduleevent/all", s.getAllScheduleEvents).Methods("GET")
	router.HandleFunc("/api/v3/scheduleevent/id/{id}", s.getScheduleEventById).Methods("GET")
	router.HandleFunc("/api/v3/scheduleevent/id/{id}", s.updateScheduleEvent).Methods("PUT")
	router.HandleFunc("/api/v3/scheduleevent/id/{id}", s.deleteScheduleEvent).Methods("DELETE")
	router.HandleFunc("/api/v3/scheduleevent/name/{name}", s.getScheduleEventByName).Methods("GET")
	
	// Schedule Action routes
	router.HandleFunc("/api/v3/scheduleaction", s.addScheduleAction).Methods("POST")
	router.HandleFunc("/api/v3/scheduleaction/all", s.getAllScheduleActions).Methods("GET")
	router.HandleFunc("/api/v3/scheduleaction/id/{id}", s.getScheduleActionById).Methods("GET")
	router.HandleFunc("/api/v3/scheduleaction/id/{id}", s.updateScheduleAction).Methods("PUT")
	router.HandleFunc("/api/v3/scheduleaction/id/{id}", s.deleteScheduleAction).Methods("DELETE")
	router.HandleFunc("/api/v3/scheduleaction/name/{name}", s.getScheduleActionByName).Methods("GET")
	
	s.logger.Info("Support Scheduler routes registered")
}

// Schedule Event handlers

// addScheduleEvent handles POST /api/v3/scheduleevent
func (s *SupportSchedulerService) addScheduleEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var event ScheduleEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		s.logger.Errorf("Failed to decode schedule event: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Generate ID and timestamps
	event.Id = models.GenerateUUID()
	event.Created = time.Now().UnixNano() / int64(time.Millisecond)
	event.Modified = event.Created
	
	// Set defaults
	if event.AdminState == "" {
		event.AdminState = common.Unlocked
	}
	
	s.mutex.Lock()
	s.scheduleEvents[event.Id] = event
	s.mutex.Unlock()
	
	// Start the scheduled job if it's enabled
	if event.AdminState == common.Unlocked {
		s.startScheduledJob(event)
	}
	
	s.logger.Infof("Schedule event created: %s", event.Name)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusCreated,
		"id":         event.Id,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getAllScheduleEvents handles GET /api/v3/scheduleevent/all
func (s *SupportSchedulerService) getAllScheduleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	s.mutex.RLock()
	events := make([]ScheduleEvent, 0, len(s.scheduleEvents))
	for _, event := range s.scheduleEvents {
		events = append(events, event)
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":     common.ServiceVersion,
		"statusCode":     http.StatusOK,
		"totalCount":     len(events),
		"scheduleEvents": events,
	}
	
	json.NewEncoder(w).Encode(response)
}

// getScheduleEventById handles GET /api/v3/scheduleevent/id/{id}
func (s *SupportSchedulerService) getScheduleEventById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.RLock()
	event, exists := s.scheduleEvents[id]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Schedule event not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"scheduleEvent": event,
	}
	
	json.NewEncoder(w).Encode(response)
}

// startScheduledJob creates and starts a scheduled job
func (s *SupportSchedulerService) startScheduledJob(event ScheduleEvent) {
	// For simplicity, we'll use a fixed interval instead of parsing cron expressions
	// In a real implementation, you'd use a cron library like github.com/robfig/cron
	
	var interval time.Duration
	switch event.Schedule {
	case "@every 1m":
		interval = time.Minute
	case "@every 5m":
		interval = 5 * time.Minute
	case "@every 10m":
		interval = 10 * time.Minute
	case "@every 1h":
		interval = time.Hour
	default:
		interval = 5 * time.Minute // Default to 5 minutes
	}
	
	ticker := time.NewTicker(interval)
	s.mutex.Lock()
	s.runningJobs[event.Id] = ticker
	s.mutex.Unlock()
	
	go func() {
		for range ticker.C {
			s.executeScheduledJob(event)
		}
	}()
	
	s.logger.Infof("Started scheduled job: %s with interval: %v", event.Name, interval)
}

// executeScheduledJob executes a scheduled job
func (s *SupportSchedulerService) executeScheduledJob(event ScheduleEvent) {
	s.logger.Infof("Executing scheduled job: %s", event.Name)
	
	// In a real implementation, this would make HTTP requests to the addressable endpoint
	// For now, we'll just log the execution
	s.logger.Infof("Job %s executed successfully at %v", event.Name, time.Now())
}

// stopScheduledJob stops a running scheduled job
func (s *SupportSchedulerService) stopScheduledJob(eventId string) {
	s.mutex.Lock()
	if ticker, exists := s.runningJobs[eventId]; exists {
		ticker.Stop()
		delete(s.runningJobs, eventId)
	}
	s.mutex.Unlock()
}

// Schedule Action handlers

// addScheduleAction handles POST /api/v3/scheduleaction
func (s *SupportSchedulerService) addScheduleAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var action ScheduleAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		s.logger.Errorf("Failed to decode schedule action: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Generate ID and timestamps
	action.Id = models.GenerateUUID()
	action.Created = time.Now().UnixNano() / int64(time.Millisecond)
	action.Modified = action.Created
	
	// Set defaults
	if action.AdminState == "" {
		action.AdminState = common.Unlocked
	}
	if action.HTTPMethod == "" {
		action.HTTPMethod = "GET"
	}
	if action.Protocol == "" {
		action.Protocol = "HTTP"
	}
	
	s.mutex.Lock()
	s.scheduleActions[action.Id] = action
	s.mutex.Unlock()
	
	s.logger.Infof("Schedule action created: %s", action.Name)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusCreated,
		"id":         action.Id,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getAllScheduleActions handles GET /api/v3/scheduleaction/all
func (s *SupportSchedulerService) getAllScheduleActions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	s.mutex.RLock()
	actions := make([]ScheduleAction, 0, len(s.scheduleActions))
	for _, action := range s.scheduleActions {
		actions = append(actions, action)
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":      common.ServiceVersion,
		"statusCode":      http.StatusOK,
		"totalCount":      len(actions),
		"scheduleActions": actions,
	}
	
	json.NewEncoder(w).Encode(response)
}

// updateScheduleEvent handles PUT /api/v3/scheduleevent/id/{id}
func (s *SupportSchedulerService) updateScheduleEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	var updatedEvent ScheduleEvent
	if err := json.NewDecoder(r.Body).Decode(&updatedEvent); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	s.mutex.Lock()
	existingEvent, exists := s.scheduleEvents[id]
	if exists {
		// Stop existing job
		if ticker, running := s.runningJobs[id]; running {
			ticker.Stop()
			delete(s.runningJobs, id)
		}
		
		updatedEvent.Id = id
		updatedEvent.Created = existingEvent.Created
		updatedEvent.Modified = time.Now().UnixNano() / int64(time.Millisecond)
		s.scheduleEvents[id] = updatedEvent
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Schedule event not found", http.StatusNotFound)
		return
	}
	
	// Start new job if enabled
	if updatedEvent.AdminState == common.Unlocked {
		s.startScheduledJob(updatedEvent)
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Schedule event updated successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// deleteScheduleEvent handles DELETE /api/v3/scheduleevent/id/{id}
func (s *SupportSchedulerService) deleteScheduleEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.Lock()
	_, exists := s.scheduleEvents[id]
	if exists {
		// Stop the job
		s.stopScheduledJob(id)
		delete(s.scheduleEvents, id)
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Schedule event not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Schedule event deleted successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// Additional handlers following the same pattern...

// getScheduleEventByName handles GET /api/v3/scheduleevent/name/{name}
func (s *SupportSchedulerService) getScheduleEventByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	name := vars["name"]
	
	s.mutex.RLock()
	var foundEvent *ScheduleEvent
	for _, event := range s.scheduleEvents {
		if event.Name == name {
			foundEvent = &event
			break
		}
	}
	s.mutex.RUnlock()
	
	if foundEvent == nil {
		http.Error(w, "Schedule event not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"scheduleEvent": *foundEvent,
	}
	
	json.NewEncoder(w).Encode(response)
}

// getScheduleActionById handles GET /api/v3/scheduleaction/id/{id}
func (s *SupportSchedulerService) getScheduleActionById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.RLock()
	action, exists := s.scheduleActions[id]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Schedule action not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":     common.ServiceVersion,
		"statusCode":     http.StatusOK,
		"scheduleAction": action,
	}
	
	json.NewEncoder(w).Encode(response)
}

// updateScheduleAction handles PUT /api/v3/scheduleaction/id/{id}
func (s *SupportSchedulerService) updateScheduleAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	var updatedAction ScheduleAction
	if err := json.NewDecoder(r.Body).Decode(&updatedAction); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	s.mutex.Lock()
	existingAction, exists := s.scheduleActions[id]
	if exists {
		updatedAction.Id = id
		updatedAction.Created = existingAction.Created
		updatedAction.Modified = time.Now().UnixNano() / int64(time.Millisecond)
		s.scheduleActions[id] = updatedAction
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Schedule action not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Schedule action updated successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// deleteScheduleAction handles DELETE /api/v3/scheduleaction/id/{id}
func (s *SupportSchedulerService) deleteScheduleAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.Lock()
	_, exists := s.scheduleActions[id]
	if exists {
		delete(s.scheduleActions, id)
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Schedule action not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Schedule action deleted successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// getScheduleActionByName handles GET /api/v3/scheduleaction/name/{name}
func (s *SupportSchedulerService) getScheduleActionByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	name := vars["name"]
	
	s.mutex.RLock()
	var foundAction *ScheduleAction
	for _, action := range s.scheduleActions {
		if action.Name == name {
			foundAction = &action
			break
		}
	}
	s.mutex.RUnlock()
	
	if foundAction == nil {
		http.Error(w, "Schedule action not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":     common.ServiceVersion,
		"statusCode":     http.StatusOK,
		"scheduleAction": *foundAction,
	}
	
	json.NewEncoder(w).Encode(response)
}