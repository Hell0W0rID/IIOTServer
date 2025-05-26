package common

// Service Names
const (
        CoreDataServiceKey        = "core-data"
        CoreMetaDataServiceKey    = "core-metadata"
        CoreCommandServiceKey     = "core-command"
        SupportNotificationsServiceKey = "support-notifications"
        SupportSchedulerServiceKey     = "support-scheduler"
        AppServiceConfigurableKey      = "app-service-configurable"
        DeviceVirtualServiceKey        = "device-virtual"
)

// API Routes
const (
        ApiBase          = "/api/v3"
        ApiPingRoute     = ApiBase + "/ping"
        ApiVersionRoute  = ApiBase + "/version"
        ApiConfigRoute   = ApiBase + "/config"
        
        // Core Data Routes
        ApiEventRoute               = ApiBase + "/event"
        ApiEventByIdRoute          = ApiBase + "/event/id/{id}"
        ApiEventByDeviceNameRoute  = ApiBase + "/event/device/name/{name}"
        ApiReadingRoute            = ApiBase + "/reading"
        ApiReadingByIdRoute        = ApiBase + "/reading/id/{id}"
        ApiReadingByDeviceNameRoute = ApiBase + "/reading/device/name/{name}"
        
        // Core Metadata Routes
        ApiDeviceRoute             = ApiBase + "/device"
        ApiDeviceByIdRoute         = ApiBase + "/device/id/{id}"
        ApiDeviceByNameRoute       = ApiBase + "/device/name/{name}"
        ApiDeviceProfileRoute      = ApiBase + "/deviceprofile"
        ApiDeviceProfileByIdRoute  = ApiBase + "/deviceprofile/id/{id}"
        ApiDeviceProfileByNameRoute = ApiBase + "/deviceprofile/name/{name}"
        ApiDeviceServiceRoute      = ApiBase + "/deviceservice"
        ApiDeviceServiceByIdRoute  = ApiBase + "/deviceservice/id/{id}"
        ApiDeviceServiceByNameRoute = ApiBase + "/deviceservice/name/{name}"
        
        // Core Command Routes
        ApiDeviceByNameCommandRoute = ApiBase + "/device/name/{name}/command"
        ApiCommandRoute           = ApiBase + "/device/name/{name}/{command}"
        ApiCommandAllRoute        = ApiBase + "/device/all"
)

// HTTP Headers
const (
        ContentType     = "Content-Type"
        ContentTypeJSON = "application/json"
        CorrelationHeader = "X-Correlation-ID"
)

// Common Parameters
const (
        Id       = "id"
        Name     = "name"
        Command  = "command"
        Offset   = "offset"
        Limit    = "limit"
)

// Default Values
const (
        DefaultOffset = 0
        DefaultLimit  = 20
        MaxLimit      = 1000
)

// Device Admin States
const (
        Locked   = "LOCKED"
        Unlocked = "UNLOCKED"
)

// Device Operating States  
const (
        Up      = "UP"
        Down    = "DOWN"
        Unknown = "UNKNOWN"
)

// Value Types
const (
        ValueTypeBool    = "Bool"
        ValueTypeString  = "String"
        ValueTypeUint8   = "Uint8"
        ValueTypeUint16  = "Uint16"
        ValueTypeUint32  = "Uint32"
        ValueTypeUint64  = "Uint64"
        ValueTypeInt8    = "Int8"
        ValueTypeInt16   = "Int16"
        ValueTypeInt32   = "Int32"
        ValueTypeInt64   = "Int64"
        ValueTypeFloat32 = "Float32"
        ValueTypeFloat64 = "Float64"
        ValueTypeBinary  = "Binary"
)

// DI Container Keys
const (
        LoggingClientName = "LoggingClient"
        DatabaseName      = "Database"
        MessagingClientName = "MessagingClient"
        RegistryClientName  = "RegistryClient"
        ConfigurationName   = "Configuration"
)

// Service Version
const ServiceVersion = "3.1.0"