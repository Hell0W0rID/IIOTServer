package models

import (
	"time"
)

// Event represents a collection of readings from a device
type Event struct {
	Id          string    `json:"id"`
	DeviceName  string    `json:"deviceName"`
	ProfileName string    `json:"profileName"`
	SourceName  string    `json:"sourceName"`
	Origin      int64     `json:"origin"`
	Tags        map[string]interface{} `json:"tags,omitempty"`
	Readings    []Reading `json:"readings"`
	Created     int64     `json:"created"`
	Modified    int64     `json:"modified"`
}

// Reading represents a single sensor reading
type Reading struct {
	Id           string                 `json:"id"`
	Origin       int64                  `json:"origin"`
	DeviceName   string                 `json:"deviceName"`
	ResourceName string                 `json:"resourceName"`
	ProfileName  string                 `json:"profileName"`
	ValueType    string                 `json:"valueType"`
	BinaryReading BinaryReading         `json:"binaryReading,omitempty"`
	SimpleReading SimpleReading         `json:"simpleReading,omitempty"`
	ObjectReading ObjectReading         `json:"objectReading,omitempty"`
	Tags         map[string]interface{} `json:"tags,omitempty"`
	Created      int64                  `json:"created"`
	Modified     int64                  `json:"modified"`
}

// SimpleReading contains value for simple data types
type SimpleReading struct {
	Value string `json:"value"`
	Units string `json:"units,omitempty"`
}

// BinaryReading contains binary data
type BinaryReading struct {
	BinaryValue []byte `json:"binaryValue"`
	MediaType   string `json:"mediaType"`
}

// ObjectReading contains structured object data
type ObjectReading struct {
	ObjectValue interface{} `json:"objectValue"`
}

// NewEvent creates a new Event with generated ID and timestamps
func NewEvent(profileName, deviceName, sourceName string) Event {
	return Event{
		Id:          GenerateUUID(),
		DeviceName:  deviceName,
		ProfileName: profileName,
		SourceName:  sourceName,
		Origin:      time.Now().UnixNano() / int64(time.Millisecond),
		Created:     time.Now().UnixNano() / int64(time.Millisecond),
		Modified:    time.Now().UnixNano() / int64(time.Millisecond),
		Readings:    []Reading{},
		Tags:        make(map[string]interface{}),
	}
}

// NewSimpleReading creates a new simple Reading
func NewSimpleReading(profileName, deviceName, resourceName, valueType, value string) Reading {
	return Reading{
		Id:           GenerateUUID(),
		Origin:       time.Now().UnixNano() / int64(time.Millisecond),
		DeviceName:   deviceName,
		ResourceName: resourceName,
		ProfileName:  profileName,
		ValueType:    valueType,
		SimpleReading: SimpleReading{
			Value: value,
		},
		Created:  time.Now().UnixNano() / int64(time.Millisecond),
		Modified: time.Now().UnixNano() / int64(time.Millisecond),
		Tags:     make(map[string]interface{}),
	}
}

// NewBinaryReading creates a new binary Reading
func NewBinaryReading(profileName, deviceName, resourceName string, binaryValue []byte, mediaType string) Reading {
	return Reading{
		Id:           GenerateUUID(),
		Origin:       time.Now().UnixNano() / int64(time.Millisecond),
		DeviceName:   deviceName,
		ResourceName: resourceName,
		ProfileName:  profileName,
		ValueType:    "Binary",
		BinaryReading: BinaryReading{
			BinaryValue: binaryValue,
			MediaType:   mediaType,
		},
		Created:  time.Now().UnixNano() / int64(time.Millisecond),
		Modified: time.Now().UnixNano() / int64(time.Millisecond),
		Tags:     make(map[string]interface{}),
	}
}

// AddReading adds a reading to the event
func (e *Event) AddReading(reading Reading) {
	e.Readings = append(e.Readings, reading)
	e.Modified = time.Now().UnixNano() / int64(time.Millisecond)
}

// AddSimpleReading adds a simple reading to the event
func (e *Event) AddSimpleReading(resourceName, valueType, value, units string) {
	reading := NewSimpleReading(e.ProfileName, e.DeviceName, resourceName, valueType, value)
	reading.SimpleReading.Units = units
	e.AddReading(reading)
}

// AddBinaryReading adds a binary reading to the event
func (e *Event) AddBinaryReading(resourceName string, binaryValue []byte, mediaType string) {
	reading := NewBinaryReading(e.ProfileName, e.DeviceName, resourceName, binaryValue, mediaType)
	e.AddReading(reading)
}