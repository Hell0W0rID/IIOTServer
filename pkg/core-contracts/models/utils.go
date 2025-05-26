package models

import (
	"crypto/rand"
	"fmt"
	"io"
)

// GenerateUUID generates a new UUID v4
func GenerateUUID() string {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		panic(err)
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

// BaseModel represents common fields for all models
type BaseModel struct {
	Created  int64 `json:"created"`
	Modified int64 `json:"modified"`
}

// BaseWithIdModel represents common fields with ID
type BaseWithIdModel struct {
	Id string `json:"id"`
	BaseModel
}

// Versionable represents models that have versioning
type Versionable interface {
	GetVersion() string
	SetVersion(version string)
}

// Timestampable represents models that have timestamps
type Timestampable interface {
	GetCreated() int64
	GetModified() int64
	SetCreated(timestamp int64)
	SetModified(timestamp int64)
}