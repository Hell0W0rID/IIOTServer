# EdgeX Go Clone - Complete Implementation

🚀 **Complete EdgeX Foundry Clone with All Modules**

A comprehensive clone of the entire EdgeX Foundry ecosystem with all modules recreated in one unified repository. This project includes all the functionality of go-mod-bootstrap, go-mod-core-contracts, go-mod-messaging, go-mod-registry, and go-mod-secrets.

## 🏗️ Directory Structure

```
edgex-go-clone/
├── pkg/                          # Core EdgeX modules (recreated)
│   ├── bootstrap/               # go-mod-bootstrap equivalent
│   ├── core-contracts/          # go-mod-core-contracts equivalent
│   ├── messaging/               # go-mod-messaging equivalent
│   ├── registry/                # go-mod-registry equivalent
│   ├── secrets/                 # go-mod-secrets equivalent
│   └── configuration/           # go-mod-configuration equivalent
├── cmd/                         # Service entry points
│   ├── core-data/
│   ├── core-metadata/
│   ├── core-command/
│   ├── support-notifications/
│   ├── support-scheduler/
│   ├── app-service-configurable/
│   └── device-virtual/
├── internal/                    # Service implementations
│   ├── core/
│   ├── support/
│   ├── application/
│   └── device/
├── configs/                     # Configuration files
├── scripts/                     # Build and deployment scripts
└── docs/                       # Documentation
```

## 🔧 Recreated EdgeX Modules

### pkg/bootstrap
- Service lifecycle management
- Configuration loading
- Dependency injection
- HTTP server setup
- Graceful shutdown

### pkg/core-contracts
- Data models (Event, Reading, Device, etc.)
- DTOs and request/response structures
- Common constants and enums
- API route definitions

### pkg/messaging
- Message bus abstraction
- Redis Streams implementation
- MQTT client support
- Message publishing/subscribing

### pkg/registry
- Service discovery
- Consul integration
- Health checks
- Service registration

### pkg/secrets
- Secret management
- Vault integration
- Token handling
- Secure configuration

## 🚀 Services Included

### Core Services
- **Core Data** - Event and reading management
- **Core Metadata** - Device and profile registry
- **Core Command** - Device command execution

### Support Services
- **Support Notifications** - Alert management
- **Support Scheduler** - Job scheduling

### Application Services
- **App Service Configurable** - Data processing pipelines

### Device Services
- **Device Virtual** - Virtual device simulation

## 🎯 Quick Start

```bash
# Install dependencies
go mod tidy

# Start Core Data Service
go run cmd/core-data/main.go

# Start Core Metadata Service
go run cmd/core-metadata/main.go

# Start all services with Docker
docker-compose up -d
```

## 🔄 API Endpoints

### Core Data (Port 59880)
- `POST /api/v3/event` - Create event
- `GET /api/v3/event/all` - Get all events
- `GET /api/v3/event/device/name/{name}` - Get events by device

### Core Metadata (Port 59881)
- `POST /api/v3/device` - Register device
- `POST /api/v3/deviceprofile` - Create device profile
- `GET /api/v3/device/all` - List devices

### Core Command (Port 59882)
- `GET /api/v3/device/all` - List controllable devices
- `PUT /api/v3/device/name/{name}/{command}` - Execute command

## 🛠️ Development

This repository contains everything needed for EdgeX development without external dependencies on EdgeX modules. All functionality is implemented from scratch following EdgeX patterns.

## 📝 License

Apache 2.0 License (same as EdgeX Foundry)

---

Complete EdgeX implementation with all modules recreated in one unified repository.