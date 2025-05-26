package metadata

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

// CoreMetadataService handles device, profile, and service management
type CoreMetadataService struct {
	logger         *logrus.Logger
	devices        map[string]models.Device
	deviceProfiles map[string]models.DeviceProfile
	deviceServices map[string]models.DeviceService
	mutex          sync.RWMutex
}

// NewCoreMetadataService creates a new core metadata service
func NewCoreMetadataService(logger *logrus.Logger) *CoreMetadataService {
	return &CoreMetadataService{
		logger:         logger,
		devices:        make(map[string]models.Device),
		deviceProfiles: make(map[string]models.DeviceProfile),
		deviceServices: make(map[string]models.DeviceService),
	}
}

// Initialize implements the BootstrapHandler interface
func (s *CoreMetadataService) Initialize(ctx context.Context, wg *sync.WaitGroup, dic *bootstrap.DIContainer) bool {
	s.logger.Info("Initializing Core Metadata Service")
	
	// Add service to DI container
	dic.Add("CoreMetadataService", s)
	
	s.logger.Info("Core Metadata Service initialization completed")
	return true
}

// AddRoutes adds core metadata specific routes
func (s *CoreMetadataService) AddRoutes(router *mux.Router) {
	// Device routes
	router.HandleFunc(common.ApiDeviceRoute, s.addDevice).Methods("POST")
	router.HandleFunc(common.ApiDeviceRoute+"/all", s.getAllDevices).Methods("GET")
	router.HandleFunc(common.ApiDeviceByIdRoute, s.getDeviceById).Methods("GET")
	router.HandleFunc(common.ApiDeviceByNameRoute, s.getDeviceByName).Methods("GET")
	router.HandleFunc(common.ApiDeviceByIdRoute, s.updateDevice).Methods("PUT")
	router.HandleFunc(common.ApiDeviceByIdRoute, s.deleteDevice).Methods("DELETE")

	// Device Profile routes
	router.HandleFunc(common.ApiDeviceProfileRoute, s.addDeviceProfile).Methods("POST")
	router.HandleFunc(common.ApiDeviceProfileRoute+"/all", s.getAllDeviceProfiles).Methods("GET")
	router.HandleFunc(common.ApiDeviceProfileByIdRoute, s.getDeviceProfileById).Methods("GET")
	router.HandleFunc(common.ApiDeviceProfileByNameRoute, s.getDeviceProfileByName).Methods("GET")

	// Device Service routes
	router.HandleFunc(common.ApiDeviceServiceRoute, s.addDeviceService).Methods("POST")
	router.HandleFunc(common.ApiDeviceServiceRoute+"/all", s.getAllDeviceServices).Methods("GET")
	router.HandleFunc(common.ApiDeviceServiceByIdRoute, s.getDeviceServiceById).Methods("GET")
	router.HandleFunc(common.ApiDeviceServiceByNameRoute, s.getDeviceServiceByName).Methods("GET")

	s.logger.Info("Core Metadata routes registered")
}

// Device handlers
func (s *CoreMetadataService) addDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		s.logger.Errorf("Failed to decode device: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Generate ID and timestamps
	device.Id = models.GenerateUUID()
	device.Created = time.Now().UnixNano() / int64(time.Millisecond)
	device.Modified = device.Created
	
	// Set defaults
	if device.AdminState == "" {
		device.AdminState = common.Unlocked
	}
	if device.OperatingState == "" {
		device.OperatingState = common.Up
	}
	
	s.mutex.Lock()
	s.devices[device.Id] = device
	s.mutex.Unlock()
	
	s.logger.Infof("Device created: %s", device.Name)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusCreated,
		"id":         device.Id,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) getAllDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	s.mutex.RLock()
	devices := make([]models.Device, 0, len(s.devices))
	for _, device := range s.devices {
		devices = append(devices, device)
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":  common.ServiceVersion,
		"statusCode":  http.StatusOK,
		"totalCount":  len(devices),
		"devices":     devices,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) getDeviceById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.RLock()
	device, exists := s.devices[id]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"device":     device,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) getDeviceByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	name := vars["name"]
	
	s.mutex.RLock()
	var foundDevice *models.Device
	for _, device := range s.devices {
		if device.Name == name {
			foundDevice = &device
			break
		}
	}
	s.mutex.RUnlock()
	
	if foundDevice == nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"device":     *foundDevice,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) updateDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	var updatedDevice models.Device
	if err := json.NewDecoder(r.Body).Decode(&updatedDevice); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	s.mutex.Lock()
	existingDevice, exists := s.devices[id]
	if exists {
		updatedDevice.Id = id
		updatedDevice.Created = existingDevice.Created
		updatedDevice.Modified = time.Now().UnixNano() / int64(time.Millisecond)
		s.devices[id] = updatedDevice
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Device updated successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) deleteDevice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.Lock()
	_, exists := s.devices[id]
	if exists {
		delete(s.devices, id)
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Device deleted successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// Device Profile handlers
func (s *CoreMetadataService) addDeviceProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var profile models.DeviceProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	profile.Id = models.GenerateUUID()
	profile.Created = time.Now().UnixNano() / int64(time.Millisecond)
	profile.Modified = profile.Created
	
	s.mutex.Lock()
	s.deviceProfiles[profile.Id] = profile
	s.mutex.Unlock()
	
	s.logger.Infof("Device profile created: %s", profile.Name)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusCreated,
		"id":         profile.Id,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) getAllDeviceProfiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	s.mutex.RLock()
	profiles := make([]models.DeviceProfile, 0, len(s.deviceProfiles))
	for _, profile := range s.deviceProfiles {
		profiles = append(profiles, profile)
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":     common.ServiceVersion,
		"statusCode":     http.StatusOK,
		"totalCount":     len(profiles),
		"deviceProfiles": profiles,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) getDeviceProfileById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.RLock()
	profile, exists := s.deviceProfiles[id]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Device profile not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"deviceProfile": profile,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) getDeviceProfileByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	name := vars["name"]
	
	s.mutex.RLock()
	var foundProfile *models.DeviceProfile
	for _, profile := range s.deviceProfiles {
		if profile.Name == name {
			foundProfile = &profile
			break
		}
	}
	s.mutex.RUnlock()
	
	if foundProfile == nil {
		http.Error(w, "Device profile not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"deviceProfile": *foundProfile,
	}
	
	json.NewEncoder(w).Encode(response)
}

// Device Service handlers
func (s *CoreMetadataService) addDeviceService(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var deviceService models.DeviceService
	if err := json.NewDecoder(r.Body).Decode(&deviceService); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	deviceService.Id = models.GenerateUUID()
	deviceService.Created = time.Now().UnixNano() / int64(time.Millisecond)
	deviceService.Modified = deviceService.Created
	
	if deviceService.AdminState == "" {
		deviceService.AdminState = common.Unlocked
	}
	if deviceService.OperatingState == "" {
		deviceService.OperatingState = common.Up
	}
	
	s.mutex.Lock()
	s.deviceServices[deviceService.Id] = deviceService
	s.mutex.Unlock()
	
	s.logger.Infof("Device service created: %s", deviceService.Name)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusCreated,
		"id":         deviceService.Id,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) getAllDeviceServices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	s.mutex.RLock()
	services := make([]models.DeviceService, 0, len(s.deviceServices))
	for _, service := range s.deviceServices {
		services = append(services, service)
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":     common.ServiceVersion,
		"statusCode":     http.StatusOK,
		"totalCount":     len(services),
		"deviceServices": services,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) getDeviceServiceById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.RLock()
	service, exists := s.deviceServices[id]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Device service not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"deviceService": service,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (s *CoreMetadataService) getDeviceServiceByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	name := vars["name"]
	
	s.mutex.RLock()
	var foundService *models.DeviceService
	for _, service := range s.deviceServices {
		if service.Name == name {
			foundService = &service
			break
		}
	}
	s.mutex.RUnlock()
	
	if foundService == nil {
		http.Error(w, "Device service not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"deviceService": *foundService,
	}
	
	json.NewEncoder(w).Encode(response)
}