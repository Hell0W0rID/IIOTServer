package virtual

import (
        "context"
        "encoding/json"
        "fmt"
        "math/rand"
        "net/http"
        "sync"
        "time"

        "github.com/gorilla/mux"
        "github.com/sirupsen/logrus"

        "github.com/Hell0W0rID/edgex-go-clone/pkg/bootstrap"
        "github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/common"
        "github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/models"
)

// VirtualDevice represents a simulated IoT device
type VirtualDevice struct {
        Id            string            `json:"id"`
        Name          string            `json:"name"`
        Description   string            `json:"description"`
        ProfileName   string            `json:"profileName"`
        ServiceName   string            `json:"serviceName"`
        AdminState    string            `json:"adminState"`
        OperatingState string           `json:"operatingState"`
        Protocols     map[string]string `json:"protocols"`
        LastReading   time.Time         `json:"lastReading"`
        IsRunning     bool              `json:"isRunning"`
}

// DeviceVirtualService handles virtual device simulation
type DeviceVirtualService struct {
        logger         *logrus.Logger
        virtualDevices map[string]*VirtualDevice
        mutex          sync.RWMutex
        stopChannels   map[string]chan bool
}

// NewDeviceVirtualService creates a new device virtual service
func NewDeviceVirtualService(logger *logrus.Logger) *DeviceVirtualService {
        service := &DeviceVirtualService{
                logger:         logger,
                virtualDevices: make(map[string]*VirtualDevice),
                stopChannels:   make(map[string]chan bool),
        }
        
        // Initialize with some default virtual devices
        service.initializeDefaultDevices()
        
        return service
}

// Initialize implements the BootstrapHandler interface
func (s *DeviceVirtualService) Initialize(ctx context.Context, wg *sync.WaitGroup, dic *bootstrap.DIContainer) bool {
        s.logger.Info("Initializing Device Virtual Service")
        
        // Add service to DI container
        dic.Add("DeviceVirtualService", s)
        
        // Start virtual device data generation
        s.startDataGeneration()
        
        s.logger.Info("Device Virtual Service initialization completed")
        return true
}

// AddRoutes adds device virtual specific routes
func (s *DeviceVirtualService) AddRoutes(router *mux.Router) {
        // Virtual device management routes
        router.HandleFunc("/api/v3/device/virtual", s.getAllVirtualDevices).Methods("GET")
        router.HandleFunc("/api/v3/device/virtual", s.createVirtualDevice).Methods("POST")
        router.HandleFunc("/api/v3/device/virtual/{id}", s.getVirtualDevice).Methods("GET")
        router.HandleFunc("/api/v3/device/virtual/{id}", s.updateVirtualDevice).Methods("PUT")
        router.HandleFunc("/api/v3/device/virtual/{id}", s.deleteVirtualDevice).Methods("DELETE")
        router.HandleFunc("/api/v3/device/virtual/{id}/start", s.startDevice).Methods("POST")
        router.HandleFunc("/api/v3/device/virtual/{id}/stop", s.stopDevice).Methods("POST")
        
        s.logger.Info("Device Virtual routes registered")
}

// initializeDefaultDevices creates sample virtual devices
func (s *DeviceVirtualService) initializeDefaultDevices() {
        devices := []*VirtualDevice{
                {
                        Id:             models.GenerateUUID(),
                        Name:           "Virtual-Temperature-Sensor-01",
                        Description:    "Virtual temperature sensor for testing",
                        ProfileName:    "TemperatureSensorProfile",
                        ServiceName:    common.DeviceVirtualServiceKey,
                        AdminState:     common.Unlocked,
                        OperatingState: common.Up,
                        Protocols: map[string]string{
                                "virtual": "true",
                                "type":    "temperature",
                        },
                        IsRunning: false,
                },
                {
                        Id:             models.GenerateUUID(),
                        Name:           "Virtual-Humidity-Sensor-01",
                        Description:    "Virtual humidity sensor for testing",
                        ProfileName:    "HumiditySensorProfile",
                        ServiceName:    common.DeviceVirtualServiceKey,
                        AdminState:     common.Unlocked,
                        OperatingState: common.Up,
                        Protocols: map[string]string{
                                "virtual": "true",
                                "type":    "humidity",
                        },
                        IsRunning: false,
                },
                {
                        Id:             models.GenerateUUID(),
                        Name:           "Virtual-Pressure-Sensor-01",
                        Description:    "Virtual pressure sensor for testing",
                        ProfileName:    "PressureSensorProfile",
                        ServiceName:    common.DeviceVirtualServiceKey,
                        AdminState:     common.Unlocked,
                        OperatingState: common.Up,
                        Protocols: map[string]string{
                                "virtual": "true",
                                "type":    "pressure",
                        },
                        IsRunning: false,
                },
        }
        
        for _, device := range devices {
                s.virtualDevices[device.Id] = device
        }
        
        s.logger.Infof("Initialized %d default virtual devices", len(devices))
}

// startDataGeneration begins generating simulated sensor data
func (s *DeviceVirtualService) startDataGeneration() {
        s.mutex.RLock()
        for _, device := range s.virtualDevices {
                if !device.IsRunning {
                        device.IsRunning = true
                        s.stopChannels[device.Id] = make(chan bool)
                        go s.generateDeviceData(device)
                }
        }
        s.mutex.RUnlock()
}

// generateDeviceData simulates sensor readings for a virtual device
func (s *DeviceVirtualService) generateDeviceData(device *VirtualDevice) {
        ticker := time.NewTicker(5 * time.Second) // Generate data every 5 seconds
        defer ticker.Stop()
        
        for {
                select {
                case <-ticker.C:
                        s.publishSensorReading(device)
                case <-s.stopChannels[device.Id]:
                        s.logger.Infof("Stopping data generation for device: %s", device.Name)
                        return
                }
        }
}

// publishSensorReading creates and publishes a sensor reading event
func (s *DeviceVirtualService) publishSensorReading(device *VirtualDevice) {
        reading := s.generateReading(device)
        
        // In a real implementation, this would publish to Core Data service
        s.logger.Debugf("Generated reading for device %s: %v", device.Name, reading.SimpleReading.Value)
        
        device.LastReading = time.Now()
}

// generateReading creates a simulated sensor reading based on device type
func (s *DeviceVirtualService) generateReading(device *VirtualDevice) models.Reading {
        var value string
        var units string
        var resourceName string
        var valueType string
        
        deviceType := device.Protocols["type"]
        
        switch deviceType {
        case "temperature":
                temp := 20.0 + rand.Float64()*15.0 // 20-35°C
                value = fmt.Sprintf("%.2f", temp)
                units = "Celsius"
                resourceName = "Temperature"
                valueType = common.ValueTypeFloat64
        case "humidity":
                humidity := 30.0 + rand.Float64()*40.0 // 30-70%
                value = fmt.Sprintf("%.2f", humidity)
                units = "Percent"
                resourceName = "Humidity"
                valueType = common.ValueTypeFloat64
        case "pressure":
                pressure := 1013.0 + rand.Float64()*20.0 // 1013-1033 hPa
                value = fmt.Sprintf("%.2f", pressure)
                units = "hPa"
                resourceName = "Pressure"
                valueType = common.ValueTypeFloat64
        default:
                genericValue := rand.Float64() * 100.0
                value = fmt.Sprintf("%.2f", genericValue)
                units = "Units"
                resourceName = "GenericSensor"
                valueType = common.ValueTypeFloat64
        }
        
        reading := models.NewSimpleReading(device.ProfileName, device.Name, resourceName, valueType, value)
        reading.SimpleReading.Units = units
        return reading
}

// HTTP Handlers

// getAllVirtualDevices handles GET /api/v3/device/virtual
func (s *DeviceVirtualService) getAllVirtualDevices(w http.ResponseWriter, r *http.Request) {
        w.Header().Set(common.ContentType, common.ContentTypeJSON)
        
        s.mutex.RLock()
        devices := make([]*VirtualDevice, 0, len(s.virtualDevices))
        for _, device := range s.virtualDevices {
                devices = append(devices, device)
        }
        s.mutex.RUnlock()
        
        response := map[string]interface{}{
                "apiVersion":     common.ServiceVersion,
                "statusCode":     http.StatusOK,
                "totalCount":     len(devices),
                "virtualDevices": devices,
        }
        
        json.NewEncoder(w).Encode(response)
}

// createVirtualDevice handles POST /api/v3/device/virtual
func (s *DeviceVirtualService) createVirtualDevice(w http.ResponseWriter, r *http.Request) {
        w.Header().Set(common.ContentType, common.ContentTypeJSON)
        
        var device VirtualDevice
        if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
                s.logger.Errorf("Failed to decode virtual device: %v", err)
                http.Error(w, "Invalid JSON", http.StatusBadRequest)
                return
        }
        
        // Generate ID and set defaults
        device.Id = models.GenerateUUID()
        device.ServiceName = common.DeviceVirtualServiceKey
        device.IsRunning = false
        
        if device.AdminState == "" {
                device.AdminState = common.Unlocked
        }
        if device.OperatingState == "" {
                device.OperatingState = common.Up
        }
        
        s.mutex.Lock()
        s.virtualDevices[device.Id] = &device
        s.mutex.Unlock()
        
        s.logger.Infof("Virtual device created: %s", device.Name)
        
        response := map[string]interface{}{
                "apiVersion": common.ServiceVersion,
                "statusCode": http.StatusCreated,
                "id":         device.Id,
        }
        
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(response)
}

// getVirtualDevice handles GET /api/v3/device/virtual/{id}
func (s *DeviceVirtualService) getVirtualDevice(w http.ResponseWriter, r *http.Request) {
        w.Header().Set(common.ContentType, common.ContentTypeJSON)
        
        vars := mux.Vars(r)
        id := vars["id"]
        
        s.mutex.RLock()
        device, exists := s.virtualDevices[id]
        s.mutex.RUnlock()
        
        if !exists {
                http.Error(w, "Virtual device not found", http.StatusNotFound)
                return
        }
        
        response := map[string]interface{}{
                "apiVersion":    common.ServiceVersion,
                "statusCode":    http.StatusOK,
                "virtualDevice": device,
        }
        
        json.NewEncoder(w).Encode(response)
}

// updateVirtualDevice handles PUT /api/v3/device/virtual/{id}
func (s *DeviceVirtualService) updateVirtualDevice(w http.ResponseWriter, r *http.Request) {
        w.Header().Set(common.ContentType, common.ContentTypeJSON)
        
        vars := mux.Vars(r)
        id := vars["id"]
        
        var updatedDevice VirtualDevice
        if err := json.NewDecoder(r.Body).Decode(&updatedDevice); err != nil {
                http.Error(w, "Invalid JSON", http.StatusBadRequest)
                return
        }
        
        s.mutex.Lock()
        existingDevice, exists := s.virtualDevices[id]
        if exists {
                updatedDevice.Id = id
                updatedDevice.IsRunning = existingDevice.IsRunning
                s.virtualDevices[id] = &updatedDevice
        }
        s.mutex.Unlock()
        
        if !exists {
                http.Error(w, "Virtual device not found", http.StatusNotFound)
                return
        }
        
        response := map[string]interface{}{
                "apiVersion": common.ServiceVersion,
                "statusCode": http.StatusOK,
                "message":    "Virtual device updated successfully",
        }
        
        json.NewEncoder(w).Encode(response)
}

// deleteVirtualDevice handles DELETE /api/v3/device/virtual/{id}
func (s *DeviceVirtualService) deleteVirtualDevice(w http.ResponseWriter, r *http.Request) {
        w.Header().Set(common.ContentType, common.ContentTypeJSON)
        
        vars := mux.Vars(r)
        id := vars["id"]
        
        s.mutex.Lock()
        device, exists := s.virtualDevices[id]
        if exists {
                // Stop data generation if running
                if device.IsRunning {
                        close(s.stopChannels[id])
                        delete(s.stopChannels, id)
                }
                delete(s.virtualDevices, id)
        }
        s.mutex.Unlock()
        
        if !exists {
                http.Error(w, "Virtual device not found", http.StatusNotFound)
                return
        }
        
        response := map[string]interface{}{
                "apiVersion": common.ServiceVersion,
                "statusCode": http.StatusOK,
                "message":    "Virtual device deleted successfully",
        }
        
        json.NewEncoder(w).Encode(response)
}

// startDevice handles POST /api/v3/device/virtual/{id}/start
func (s *DeviceVirtualService) startDevice(w http.ResponseWriter, r *http.Request) {
        w.Header().Set(common.ContentType, common.ContentTypeJSON)
        
        vars := mux.Vars(r)
        id := vars["id"]
        
        s.mutex.Lock()
        device, exists := s.virtualDevices[id]
        if exists && !device.IsRunning {
                device.IsRunning = true
                s.stopChannels[id] = make(chan bool)
                go s.generateDeviceData(device)
        }
        s.mutex.Unlock()
        
        if !exists {
                http.Error(w, "Virtual device not found", http.StatusNotFound)
                return
        }
        
        s.logger.Infof("Started virtual device: %s", device.Name)
        
        response := map[string]interface{}{
                "apiVersion": common.ServiceVersion,
                "statusCode": http.StatusOK,
                "message":    "Virtual device started successfully",
        }
        
        json.NewEncoder(w).Encode(response)
}

// stopDevice handles POST /api/v3/device/virtual/{id}/stop
func (s *DeviceVirtualService) stopDevice(w http.ResponseWriter, r *http.Request) {
        w.Header().Set(common.ContentType, common.ContentTypeJSON)
        
        vars := mux.Vars(r)
        id := vars["id"]
        
        s.mutex.Lock()
        device, exists := s.virtualDevices[id]
        if exists && device.IsRunning {
                device.IsRunning = false
                close(s.stopChannels[id])
                delete(s.stopChannels, id)
        }
        s.mutex.Unlock()
        
        if !exists {
                http.Error(w, "Virtual device not found", http.StatusNotFound)
                return
        }
        
        s.logger.Infof("Stopped virtual device: %s", device.Name)
        
        response := map[string]interface{}{
                "apiVersion": common.ServiceVersion,
                "statusCode": http.StatusOK,
                "message":    "Virtual device stopped successfully",
        }
        
        json.NewEncoder(w).Encode(response)
}