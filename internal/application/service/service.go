package service

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

// Pipeline represents a data processing pipeline
type Pipeline struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Transforms  []Transform `json:"transforms"`
	Target      Target      `json:"target"`
	AdminState  string      `json:"adminState"`
	Created     int64       `json:"created"`
	Modified    int64       `json:"modified"`
}

// Transform represents a data transformation step
type Transform struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// Target represents the output destination
type Target struct {
	Type       string                 `json:"type"`
	Host       string                 `json:"host,omitempty"`
	Port       int                    `json:"port,omitempty"`
	Topic      string                 `json:"topic,omitempty"`
	Format     string                 `json:"format,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ApplicationService handles data processing pipelines
type ApplicationService struct {
	logger    *logrus.Logger
	pipelines map[string]Pipeline
	mutex     sync.RWMutex
}

// NewApplicationService creates a new application service
func NewApplicationService(logger *logrus.Logger) *ApplicationService {
	service := &ApplicationService{
		logger:    logger,
		pipelines: make(map[string]Pipeline),
	}
	
	// Initialize with default pipelines
	service.initializeDefaultPipelines()
	
	return service
}

// Initialize implements the BootstrapHandler interface
func (s *ApplicationService) Initialize(ctx context.Context, wg *sync.WaitGroup, dic *bootstrap.DIContainer) bool {
	s.logger.Info("Initializing Application Service")
	
	// Add service to DI container
	dic.Add("ApplicationService", s)
	
	s.logger.Info("Application Service initialization completed")
	return true
}

// AddRoutes adds application service specific routes
func (s *ApplicationService) AddRoutes(router *mux.Router) {
	// Pipeline management routes
	router.HandleFunc("/api/v3/pipeline", s.addPipeline).Methods("POST")
	router.HandleFunc("/api/v3/pipeline/all", s.getAllPipelines).Methods("GET")
	router.HandleFunc("/api/v3/pipeline/id/{id}", s.getPipelineById).Methods("GET")
	router.HandleFunc("/api/v3/pipeline/id/{id}", s.updatePipeline).Methods("PUT")
	router.HandleFunc("/api/v3/pipeline/id/{id}", s.deletePipeline).Methods("DELETE")
	router.HandleFunc("/api/v3/pipeline/name/{name}", s.getPipelineByName).Methods("GET")
	router.HandleFunc("/api/v3/pipeline/id/{id}/start", s.startPipeline).Methods("POST")
	router.HandleFunc("/api/v3/pipeline/id/{id}/stop", s.stopPipeline).Methods("POST")
	
	// Data processing routes
	router.HandleFunc("/api/v3/process", s.processData).Methods("POST")
	router.HandleFunc("/api/v3/trigger/{pipelineId}", s.triggerPipeline).Methods("POST")
	
	s.logger.Info("Application Service routes registered")
}

// initializeDefaultPipelines creates sample data processing pipelines
func (s *ApplicationService) initializeDefaultPipelines() {
	pipelines := []Pipeline{
		{
			Id:          models.GenerateUUID(),
			Name:        "DefaultFilterPipeline",
			Description: "Default pipeline that filters temperature readings above 30°C",
			Transforms: []Transform{
				{
					Type: "Filter",
					Parameters: map[string]interface{}{
						"condition": "temperature > 30",
						"resource":  "Temperature",
					},
				},
				{
					Type: "Convert",
					Parameters: map[string]interface{}{
						"format": "json",
					},
				},
			},
			Target: Target{
				Type:   "HTTP",
				Host:   "localhost",
				Port:   8080,
				Format: "json",
			},
			AdminState: common.Unlocked,
			Created:    time.Now().UnixNano() / int64(time.Millisecond),
			Modified:   time.Now().UnixNano() / int64(time.Millisecond),
		},
		{
			Id:          models.GenerateUUID(),
			Name:        "DataExportPipeline",
			Description: "Pipeline that exports all sensor data to external system",
			Transforms: []Transform{
				{
					Type: "Batch",
					Parameters: map[string]interface{}{
						"batchSize": 10,
						"timeout":   "30s",
					},
				},
				{
					Type: "Compress",
					Parameters: map[string]interface{}{
						"algorithm": "gzip",
					},
				},
			},
			Target: Target{
				Type:  "MQTT",
				Host:  "mqtt-broker",
				Port:  1883,
				Topic: "edgex/export",
				Format: "json",
			},
			AdminState: common.Unlocked,
			Created:    time.Now().UnixNano() / int64(time.Millisecond),
			Modified:   time.Now().UnixNano() / int64(time.Millisecond),
		},
	}
	
	for _, pipeline := range pipelines {
		s.pipelines[pipeline.Id] = pipeline
	}
	
	s.logger.Infof("Initialized %d default pipelines", len(pipelines))
}

// Pipeline handlers

// addPipeline handles POST /api/v3/pipeline
func (s *ApplicationService) addPipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var pipeline Pipeline
	if err := json.NewDecoder(r.Body).Decode(&pipeline); err != nil {
		s.logger.Errorf("Failed to decode pipeline: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Generate ID and timestamps
	pipeline.Id = models.GenerateUUID()
	pipeline.Created = time.Now().UnixNano() / int64(time.Millisecond)
	pipeline.Modified = pipeline.Created
	
	// Set defaults
	if pipeline.AdminState == "" {
		pipeline.AdminState = common.Unlocked
	}
	
	s.mutex.Lock()
	s.pipelines[pipeline.Id] = pipeline
	s.mutex.Unlock()
	
	s.logger.Infof("Pipeline created: %s", pipeline.Name)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusCreated,
		"id":         pipeline.Id,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getAllPipelines handles GET /api/v3/pipeline/all
func (s *ApplicationService) getAllPipelines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	s.mutex.RLock()
	pipelines := make([]Pipeline, 0, len(s.pipelines))
	for _, pipeline := range s.pipelines {
		pipelines = append(pipelines, pipeline)
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":  common.ServiceVersion,
		"statusCode":  http.StatusOK,
		"totalCount":  len(pipelines),
		"pipelines":   pipelines,
	}
	
	json.NewEncoder(w).Encode(response)
}

// getPipelineById handles GET /api/v3/pipeline/id/{id}
func (s *ApplicationService) getPipelineById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.RLock()
	pipeline, exists := s.pipelines[id]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"pipeline":   pipeline,
	}
	
	json.NewEncoder(w).Encode(response)
}

// processData handles POST /api/v3/process
func (s *ApplicationService) processData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var event models.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		s.logger.Errorf("Failed to decode event: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Process through all active pipelines
	results := s.processEventThroughPipelines(event)
	
	response := map[string]interface{}{
		"apiVersion":       common.ServiceVersion,
		"statusCode":       http.StatusOK,
		"processedEvent":   event,
		"pipelineResults":  results,
		"totalPipelines":   len(results),
	}
	
	json.NewEncoder(w).Encode(response)
}

// processEventThroughPipelines processes an event through all active pipelines
func (s *ApplicationService) processEventThroughPipelines(event models.Event) []map[string]interface{} {
	var results []map[string]interface{}
	
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	for _, pipeline := range s.pipelines {
		if pipeline.AdminState == common.Unlocked {
			result := s.executePipeline(event, pipeline)
			results = append(results, result)
		}
	}
	
	return results
}

// executePipeline executes a single pipeline on an event
func (s *ApplicationService) executePipeline(event models.Event, pipeline Pipeline) map[string]interface{} {
	s.logger.Debugf("Executing pipeline: %s for event: %s", pipeline.Name, event.Id)
	
	processedEvent := event
	transformResults := []string{}
	
	// Execute transforms
	for _, transform := range pipeline.Transforms {
		result := s.executeTransform(processedEvent, transform)
		transformResults = append(transformResults, result)
	}
	
	// Execute target (output)
	targetResult := s.executeTarget(processedEvent, pipeline.Target)
	
	return map[string]interface{}{
		"pipelineId":       pipeline.Id,
		"pipelineName":     pipeline.Name,
		"transformResults": transformResults,
		"targetResult":     targetResult,
		"status":           "success",
		"timestamp":        time.Now().UnixNano() / int64(time.Millisecond),
	}
}

// executeTransform executes a single transform
func (s *ApplicationService) executeTransform(event models.Event, transform Transform) string {
	switch transform.Type {
	case "Filter":
		return s.executeFilterTransform(event, transform)
	case "Convert":
		return s.executeConvertTransform(event, transform)
	case "Batch":
		return s.executeBatchTransform(event, transform)
	case "Compress":
		return s.executeCompressTransform(event, transform)
	default:
		return "Unknown transform type"
	}
}

// executeFilterTransform simulates filtering data
func (s *ApplicationService) executeFilterTransform(event models.Event, transform Transform) string {
	// Simulate filter logic
	condition := transform.Parameters["condition"]
	s.logger.Debugf("Applying filter: %v", condition)
	return "Filter applied successfully"
}

// executeConvertTransform simulates data conversion
func (s *ApplicationService) executeConvertTransform(event models.Event, transform Transform) string {
	format := transform.Parameters["format"]
	s.logger.Debugf("Converting to format: %v", format)
	return "Data converted successfully"
}

// executeBatchTransform simulates batching data
func (s *ApplicationService) executeBatchTransform(event models.Event, transform Transform) string {
	batchSize := transform.Parameters["batchSize"]
	s.logger.Debugf("Batching with size: %v", batchSize)
	return "Data batched successfully"
}

// executeCompressTransform simulates data compression
func (s *ApplicationService) executeCompressTransform(event models.Event, transform Transform) string {
	algorithm := transform.Parameters["algorithm"]
	s.logger.Debugf("Compressing with algorithm: %v", algorithm)
	return "Data compressed successfully"
}

// executeTarget simulates sending data to target
func (s *ApplicationService) executeTarget(event models.Event, target Target) string {
	switch target.Type {
	case "HTTP":
		s.logger.Debugf("Sending to HTTP endpoint: %s:%d", target.Host, target.Port)
		return "Sent to HTTP endpoint"
	case "MQTT":
		s.logger.Debugf("Publishing to MQTT topic: %s", target.Topic)
		return "Published to MQTT"
	case "FILE":
		s.logger.Debugf("Writing to file")
		return "Written to file"
	default:
		return "Unknown target type"
	}
}

// Additional handlers

// updatePipeline handles PUT /api/v3/pipeline/id/{id}
func (s *ApplicationService) updatePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	var updatedPipeline Pipeline
	if err := json.NewDecoder(r.Body).Decode(&updatedPipeline); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	s.mutex.Lock()
	existingPipeline, exists := s.pipelines[id]
	if exists {
		updatedPipeline.Id = id
		updatedPipeline.Created = existingPipeline.Created
		updatedPipeline.Modified = time.Now().UnixNano() / int64(time.Millisecond)
		s.pipelines[id] = updatedPipeline
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Pipeline updated successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// deletePipeline handles DELETE /api/v3/pipeline/id/{id}
func (s *ApplicationService) deletePipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.Lock()
	_, exists := s.pipelines[id]
	if exists {
		delete(s.pipelines, id)
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Pipeline deleted successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// getPipelineByName handles GET /api/v3/pipeline/name/{name}
func (s *ApplicationService) getPipelineByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	name := vars["name"]
	
	s.mutex.RLock()
	var foundPipeline *Pipeline
	for _, pipeline := range s.pipelines {
		if pipeline.Name == name {
			foundPipeline = &pipeline
			break
		}
	}
	s.mutex.RUnlock()
	
	if foundPipeline == nil {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"pipeline":   *foundPipeline,
	}
	
	json.NewEncoder(w).Encode(response)
}

// startPipeline handles POST /api/v3/pipeline/id/{id}/start
func (s *ApplicationService) startPipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.Lock()
	pipeline, exists := s.pipelines[id]
	if exists {
		pipeline.AdminState = common.Unlocked
		pipeline.Modified = time.Now().UnixNano() / int64(time.Millisecond)
		s.pipelines[id] = pipeline
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	
	s.logger.Infof("Started pipeline: %s", pipeline.Name)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Pipeline started successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// stopPipeline handles POST /api/v3/pipeline/id/{id}/stop
func (s *ApplicationService) stopPipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.Lock()
	pipeline, exists := s.pipelines[id]
	if exists {
		pipeline.AdminState = common.Locked
		pipeline.Modified = time.Now().UnixNano() / int64(time.Millisecond)
		s.pipelines[id] = pipeline
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	
	s.logger.Infof("Stopped pipeline: %s", pipeline.Name)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Pipeline stopped successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// triggerPipeline handles POST /api/v3/trigger/{pipelineId}
func (s *ApplicationService) triggerPipeline(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	pipelineId := vars["pipelineId"]
	
	var event models.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		s.logger.Errorf("Failed to decode event: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	s.mutex.RLock()
	pipeline, exists := s.pipelines[pipelineId]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Pipeline not found", http.StatusNotFound)
		return
	}
	
	if pipeline.AdminState != common.Unlocked {
		http.Error(w, "Pipeline is not active", http.StatusConflict)
		return
	}
	
	result := s.executePipeline(event, pipeline)
	
	response := map[string]interface{}{
		"apiVersion":     common.ServiceVersion,
		"statusCode":     http.StatusOK,
		"pipelineResult": result,
	}
	
	json.NewEncoder(w).Encode(response)
}