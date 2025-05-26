package command

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/bootstrap"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/common"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/models"
)

// CommandResponse represents a device command response
type CommandResponse struct {
	Id          string            `json:"id"`
	DeviceName  string            `json:"deviceName"`
	ProfileName string            `json:"profileName"`
	CommandName string            `json:"commandName"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Response    interface{}       `json:"response,omitempty"`
	Timestamp   int64             `json:"timestamp"`
	StatusCode  int               `json:"statusCode"`
}

// CoreCommandService handles device command execution
type CoreCommandService struct {
	logger           *logrus.Logger
	commandResponses map[string]CommandResponse
	mutex            sync.RWMutex
}

// NewCoreCommandService creates a new core command service
func NewCoreCommandService(logger *logrus.Logger) *CoreCommandService {
	return &CoreCommandService{
		logger:           logger,
		commandResponses: make(map[string]CommandResponse),
	}
}

// Initialize implements the BootstrapHandler interface
func (s *CoreCommandService) Initialize(ctx context.Context, wg *sync.WaitGroup, dic *bootstrap.DIContainer) bool {
	s.logger.Info("Initializing Core Command Service")
	
	// Add service to DI container
	dic.Add("CoreCommandService", s)
	
	s.logger.Info("Core Command Service initialization completed")
	return true
}

// AddRoutes adds core command specific routes
func (s *CoreCommandService) AddRoutes(router *mux.Router) {
	// Device command routes
	router.HandleFunc(common.ApiDeviceByNameCommandRoute, s.getDeviceCommands).Methods("GET")
	router.HandleFunc(common.ApiDeviceByNameCommandRoute+"/{command}", s.issueGetCommand).Methods("GET")
	router.HandleFunc(common.ApiDeviceByNameCommandRoute+"/{command}", s.issueSetCommand).Methods("PUT")
	
	s.logger.Info("Core Command routes registered")
}

// getDeviceCommands handles GET /api/v3/device/name/{name}/command
func (s *CoreCommandService) getDeviceCommands(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	deviceName := vars["name"]
	
	// In a real implementation, this would query metadata service for device profile
	// For now, return a sample set of available commands
	commands := []map[string]interface{}{
		{
			"name":       "Temperature",
			"get":        true,
			"set":        false,
			"path":       fmt.Sprintf("/api/v3/device/name/%s/command/Temperature", deviceName),
			"parameters": []string{},
		},
		{
			"name":       "Humidity",
			"get":        true,
			"set":        false,
			"path":       fmt.Sprintf("/api/v3/device/name/%s/command/Humidity", deviceName),
			"parameters": []string{},
		},
		{
			"name":       "SetPoint",
			"get":        true,
			"set":        true,
			"path":       fmt.Sprintf("/api/v3/device/name/%s/command/SetPoint", deviceName),
			"parameters": []string{"value"},
		},
	}
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"deviceName":    deviceName,
		"commands":      commands,
	}
	
	s.logger.Infof("Retrieved commands for device: %s", deviceName)
	json.NewEncoder(w).Encode(response)
}

// issueGetCommand handles GET /api/v3/device/name/{name}/command/{command}
func (s *CoreCommandService) issueGetCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	deviceName := vars["name"]
	commandName := vars["command"]
	
	// Simulate command execution
	responseId := models.GenerateUUID()
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	
	var commandResult interface{}
	
	// Simulate different command responses based on command name
	switch commandName {
	case "Temperature":
		commandResult = map[string]interface{}{
			"value": 22.5,
			"units": "Celsius",
		}
	case "Humidity":
		commandResult = map[string]interface{}{
			"value": 65.2,
			"units": "Percent",
		}
	case "SetPoint":
		commandResult = map[string]interface{}{
			"value": 20.0,
			"units": "Celsius",
		}
	default:
		http.Error(w, "Command not found", http.StatusNotFound)
		return
	}
	
	cmdResponse := CommandResponse{
		Id:          responseId,
		DeviceName:  deviceName,
		CommandName: commandName,
		Response:    commandResult,
		Timestamp:   timestamp,
		StatusCode:  http.StatusOK,
	}
	
	// Store command response
	s.mutex.Lock()
	s.commandResponses[responseId] = cmdResponse
	s.mutex.Unlock()
	
	s.logger.Infof("Executed GET command %s on device %s", commandName, deviceName)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"event": map[string]interface{}{
			"id":         models.GenerateUUID(),
			"deviceName": deviceName,
			"profileName": "DefaultProfile",
			"sourceName": commandName,
			"origin":     timestamp,
			"readings": []map[string]interface{}{
				{
					"id":           models.GenerateUUID(),
					"origin":       timestamp,
					"deviceName":   deviceName,
					"resourceName": commandName,
					"profileName":  "DefaultProfile",
					"valueType":    "Object",
					"value":        commandResult,
				},
			},
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

// issueSetCommand handles PUT /api/v3/device/name/{name}/command/{command}
func (s *CoreCommandService) issueSetCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	deviceName := vars["name"]
	commandName := vars["command"]
	
	// Parse command parameters from request body
	var commandRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&commandRequest); err != nil {
		s.logger.Errorf("Failed to decode command request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Validate command exists and supports SET
	if commandName != "SetPoint" {
		http.Error(w, "Command does not support SET operation", http.StatusMethodNotAllowed)
		return
	}
	
	// Simulate command execution
	responseId := models.GenerateUUID()
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	
	cmdResponse := CommandResponse{
		Id:          responseId,
		DeviceName:  deviceName,
		CommandName: commandName,
		Parameters:  make(map[string]string),
		Response:    "Command executed successfully",
		Timestamp:   timestamp,
		StatusCode:  http.StatusOK,
	}
	
	// Convert parameters to string map
	for key, value := range commandRequest {
		cmdResponse.Parameters[key] = fmt.Sprintf("%v", value)
	}
	
	// Store command response
	s.mutex.Lock()
	s.commandResponses[responseId] = cmdResponse
	s.mutex.Unlock()
	
	s.logger.Infof("Executed SET command %s on device %s with parameters: %v", commandName, deviceName, commandRequest)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Command executed successfully",
		"commandId":  responseId,
	}
	
	json.NewEncoder(w).Encode(response)
}