package models

import (
	"time"
)

// Device represents an IoT device in the EdgeX ecosystem
type Device struct {
	Id             string                        `json:"id"`
	Name           string                        `json:"name"`
	Description    string                        `json:"description,omitempty"`
	AdminState     string                        `json:"adminState"`
	OperatingState string                        `json:"operatingState"`
	LastConnected  int64                         `json:"lastConnected,omitempty"`
	LastReported   int64                         `json:"lastReported,omitempty"`
	Labels         []string                      `json:"labels,omitempty"`
	Location       map[string]string             `json:"location,omitempty"`
	ServiceName    string                        `json:"serviceName"`
	ProfileName    string                        `json:"profileName"`
	Protocols      map[string]ProtocolProperties `json:"protocols"`
	AutoEvents     []AutoEvent                   `json:"autoEvents,omitempty"`
	Notify         bool                          `json:"notify,omitempty"`
	Created        int64                         `json:"created"`
	Modified       int64                         `json:"modified"`
}

// DeviceProfile defines device capabilities and commands
type DeviceProfile struct {
	Id              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	Manufacturer    string          `json:"manufacturer,omitempty"`
	Model           string          `json:"model,omitempty"`
	Labels          []string        `json:"labels,omitempty"`
	DeviceResources []DeviceResource `json:"deviceResources"`
	DeviceCommands  []DeviceCommand  `json:"deviceCommands,omitempty"`
	CoreCommands    []Command        `json:"coreCommands,omitempty"`
	Created         int64           `json:"created"`
	Modified        int64           `json:"modified"`
}

// DeviceService manages a group of devices
type DeviceService struct {
	Id             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	BaseAddress    string   `json:"baseAddress"`
	AdminState     string   `json:"adminState"`
	OperatingState string   `json:"operatingState"`
	Labels         []string `json:"labels,omitempty"`
	Created        int64    `json:"created"`
	Modified       int64    `json:"modified"`
}

// DeviceResource defines a device capability
type DeviceResource struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	IsHidden    bool                   `json:"isHidden,omitempty"`
	Properties  ResourceProperties     `json:"properties"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Tags        map[string]string      `json:"tags,omitempty"`
}

// DeviceCommand defines a device command
type DeviceCommand struct {
	Name               string              `json:"name"`
	IsHidden           bool                `json:"isHidden,omitempty"`
	ReadWrite          string              `json:"readWrite"`
	ResourceOperations []ResourceOperation `json:"resourceOperations"`
	Tags               map[string]string   `json:"tags,omitempty"`
}

// Command represents a core command
type Command struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Get        bool   `json:"get"`
	Put        bool   `json:"put"`
	Path       string `json:"path"`
	Url        string `json:"url"`
	Parameters []CommandParameter `json:"parameters,omitempty"`
	Response   []CommandResponse  `json:"response,omitempty"`
	Created    int64  `json:"created"`
	Modified   int64  `json:"modified"`
}

// CommandParameter defines command parameters
type CommandParameter struct {
	ResourceName string `json:"resourceName"`
	ValueType    string `json:"valueType"`
}

// CommandResponse defines command response
type CommandResponse struct {
	Code        string   `json:"code"`
	Description string   `json:"description"`
	ExpectedValues []string `json:"expectedValues,omitempty"`
}

// ResourceProperties defines resource properties
type ResourceProperties struct {
	ValueType    string `json:"valueType"`
	ReadWrite    string `json:"readWrite"`
	Minimum      string `json:"minimum,omitempty"`
	Maximum      string `json:"maximum,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Units        string `json:"units,omitempty"`
	Assertion    string `json:"assertion,omitempty"`
	Precision    string `json:"precision,omitempty"`
	FloatEncoding string `json:"floatEncoding,omitempty"`
	MediaType    string `json:"mediaType,omitempty"`
}

// ResourceOperation defines a resource operation
type ResourceOperation struct {
	DeviceResource string            `json:"deviceResource"`
	DefaultValue   string            `json:"defaultValue,omitempty"`
	Mappings       map[string]string `json:"mappings,omitempty"`
}

// ProtocolProperties defines protocol-specific properties
type ProtocolProperties struct {
	Address  string                 `json:"address,omitempty"`
	Port     string                 `json:"port,omitempty"`
	Protocol string                 `json:"protocol,omitempty"`
	Other    map[string]interface{} `json:"other,omitempty"`
}

// AutoEvent defines automatic event generation
type AutoEvent struct {
	Interval   string `json:"interval"`
	OnChange   bool   `json:"onChange,omitempty"`
	SourceName string `json:"sourceName"`
}

// NewDevice creates a new Device with generated ID and timestamps
func NewDevice(name, description, serviceName, profileName string) Device {
	return Device{
		Id:             GenerateUUID(),
		Name:           name,
		Description:    description,
		AdminState:     "UNLOCKED",
		OperatingState: "UP",
		ServiceName:    serviceName,
		ProfileName:    profileName,
		Protocols:      make(map[string]ProtocolProperties),
		Labels:         []string{},
		Location:       make(map[string]string),
		AutoEvents:     []AutoEvent{},
		Created:        time.Now().UnixNano() / int64(time.Millisecond),
		Modified:       time.Now().UnixNano() / int64(time.Millisecond),
	}
}

// NewDeviceProfile creates a new DeviceProfile with generated ID and timestamps
func NewDeviceProfile(name, description, manufacturer, model string) DeviceProfile {
	return DeviceProfile{
		Id:              GenerateUUID(),
		Name:            name,
		Description:     description,
		Manufacturer:    manufacturer,
		Model:           model,
		Labels:          []string{},
		DeviceResources: []DeviceResource{},
		DeviceCommands:  []DeviceCommand{},
		CoreCommands:    []Command{},
		Created:         time.Now().UnixNano() / int64(time.Millisecond),
		Modified:        time.Now().UnixNano() / int64(time.Millisecond),
	}
}

// NewDeviceService creates a new DeviceService with generated ID and timestamps
func NewDeviceService(name, description, baseAddress string) DeviceService {
	return DeviceService{
		Id:             GenerateUUID(),
		Name:           name,
		Description:    description,
		BaseAddress:    baseAddress,
		AdminState:     "UNLOCKED",
		OperatingState: "UP",
		Labels:         []string{},
		Created:        time.Now().UnixNano() / int64(time.Millisecond),
		Modified:       time.Now().UnixNano() / int64(time.Millisecond),
	}
}