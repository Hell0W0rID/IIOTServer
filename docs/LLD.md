# Low-Level Design (LLD) - EdgeX Foundry Complete Implementation

## 📋 Document Overview

This Low-Level Design document provides detailed technical specifications for all components of the complete EdgeX Foundry implementation, including API specifications, database schemas, algorithms, and implementation details.

## 🔧 Core Services Implementation

### 1. Core Data Service (Port 59880)

#### Service Structure
```go
type CoreDataService struct {
    logger   *logrus.Logger
    events   map[string]models.Event    // In-memory event storage
    mutex    sync.RWMutex              // Thread-safe access
}
```

#### Key Methods Implementation

##### Event Ingestion Algorithm
```go
func (s *CoreDataService) addEvent(w http.ResponseWriter, r *http.Request) {
    // 1. Parse and validate JSON payload
    var event models.Event
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        return http.StatusBadRequest
    }
    
    // 2. Generate unique identifiers
    event.Id = models.GenerateUUID()
    event.Created = time.Now().UnixNano() / int64(time.Millisecond)
    event.Modified = event.Created
    
    // 3. Process readings
    for i := range event.Readings {
        if event.Readings[i].Id == "" {
            event.Readings[i].Id = models.GenerateUUID()
        }
        event.Readings[i].Created = event.Created
        event.Readings[i].Modified = event.Modified
    }
    
    // 4. Store event with thread safety
    s.mutex.Lock()
    s.events[event.Id] = event
    s.mutex.Unlock()
    
    // 5. Return response
    return http.StatusCreated, event.Id
}
```

##### Data Retrieval with Pagination
```go
func (s *CoreDataService) getAllEvents(offset, limit int) ([]models.Event, int) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()
    
    events := make([]models.Event, 0, len(s.events))
    for _, event := range s.events {
        events = append(events, event)
    }
    
    totalCount := len(events)
    start := offset
    if start >= len(events) {
        start = len(events)
    }
    
    end := start + limit
    if end > len(events) {
        end = len(events)
    }
    
    return events[start:end], totalCount
}
```

### 2. Core Metadata Service (Port 59881)

#### Service Structure
```go
type CoreMetadataService struct {
    logger         *logrus.Logger
    devices        map[string]models.Device        // Device registry
    deviceProfiles map[string]models.DeviceProfile // Profile definitions
    deviceServices map[string]models.DeviceService // Service configurations
    mutex          sync.RWMutex                    // Thread-safe access
}
```

#### Device Registration Algorithm
```go
func (s *CoreMetadataService) addDevice(device models.Device) (string, error) {
    // 1. Validate device data
    if device.Name == "" {
        return "", errors.New("device name is required")
    }
    
    // 2. Check for name conflicts
    s.mutex.RLock()
    for _, existingDevice := range s.devices {
        if existingDevice.Name == device.Name {
            s.mutex.RUnlock()
            return "", errors.New("device name already exists")
        }
    }
    s.mutex.RUnlock()
    
    // 3. Generate metadata
    device.Id = models.GenerateUUID()
    device.Created = time.Now().UnixNano() / int64(time.Millisecond)
    device.Modified = device.Created
    
    // 4. Set defaults
    if device.AdminState == "" {
        device.AdminState = common.Unlocked
    }
    if device.OperatingState == "" {
        device.OperatingState = common.Up
    }
    
    // 5. Store device
    s.mutex.Lock()
    s.devices[device.Id] = device
    s.mutex.Unlock()
    
    return device.Id, nil
}
```

### 3. Core Command Service (Port 59882)

#### Command Execution Engine
```go
type CoreCommandService struct {
    logger           *logrus.Logger
    commandResponses map[string]CommandResponse
    mutex            sync.RWMutex
}

type CommandResponse struct {
    Id          string            `json:"id"`
    DeviceName  string            `json:"deviceName"`
    CommandName string            `json:"commandName"`
    Parameters  map[string]string `json:"parameters,omitempty"`
    Response    interface{}       `json:"response,omitempty"`
    Timestamp   int64             `json:"timestamp"`
    StatusCode  int               `json:"statusCode"`
}
```

#### Command Processing Algorithm
```go
func (s *CoreCommandService) executeCommand(deviceName, commandName string, parameters map[string]interface{}) (*CommandResponse, error) {
    // 1. Validate command availability
    if !s.isCommandSupported(deviceName, commandName) {
        return nil, errors.New("command not supported")
    }
    
    // 2. Generate command execution context
    responseId := models.GenerateUUID()
    timestamp := time.Now().UnixNano() / int64(time.Millisecond)
    
    // 3. Execute command based on type
    var result interface{}
    switch commandName {
    case "Temperature":
        result = s.executeTemperatureRead(deviceName)
    case "SetPoint":
        result = s.executeSetPoint(deviceName, parameters)
    default:
        return nil, errors.New("unknown command")
    }
    
    // 4. Create response
    response := &CommandResponse{
        Id:          responseId,
        DeviceName:  deviceName,
        CommandName: commandName,
        Response:    result,
        Timestamp:   timestamp,
        StatusCode:  200,
    }
    
    // 5. Store response for audit
    s.mutex.Lock()
    s.commandResponses[responseId] = *response
    s.mutex.Unlock()
    
    return response, nil
}
```

## 🔔 Support Services Implementation

### 1. Support Notifications Service (Port 59860)

#### Notification Processing Engine
```go
type NotificationEngine struct {
    notifications map[string]Notification
    subscriptions map[string]Subscription
    channels      map[string]NotificationChannel
    mutex         sync.RWMutex
}

type NotificationChannel interface {
    Send(notification Notification, recipients []string) error
    GetType() string
    IsAvailable() bool
}
```

#### Subscription Matching Algorithm
```go
func (s *SupportNotificationsService) matchesSubscription(notification Notification, subscription Subscription) bool {
    // 1. Check category matching
    if len(subscription.Categories) > 0 {
        categoryMatch := false
        for _, category := range subscription.Categories {
            if category == notification.Category {
                categoryMatch = true
                break
            }
        }
        if !categoryMatch {
            return false
        }
    }
    
    // 2. Check label matching
    if len(subscription.Labels) > 0 {
        labelMatch := false
        for _, subLabel := range subscription.Labels {
            for _, notifLabel := range notification.Labels {
                if subLabel == notifLabel {
                    labelMatch = true
                    break
                }
            }
            if labelMatch {
                break
            }
        }
        if !labelMatch {
            return false
        }
    }
    
    // 3. Check severity level
    if subscription.MinSeverity != "" {
        if !s.severityMeetsThreshold(notification.Severity, subscription.MinSeverity) {
            return false
        }
    }
    
    return true
}
```

### 2. Support Scheduler Service (Port 59861)

#### Cron Job Execution Engine
```go
type SchedulerEngine struct {
    scheduleEvents  map[string]ScheduleEvent
    runningJobs     map[string]*time.Ticker
    jobQueues       map[string]chan JobExecution
    mutex           sync.RWMutex
}

type JobExecution struct {
    JobId       string
    ExecutionId string
    StartTime   time.Time
    Status      string
    Result      interface{}
    Error       error
}
```

#### Job Scheduling Algorithm
```go
func (s *SupportSchedulerService) startScheduledJob(event ScheduleEvent) error {
    // 1. Parse schedule expression
    interval, err := s.parseScheduleExpression(event.Schedule)
    if err != nil {
        return err
    }
    
    // 2. Create ticker for job execution
    ticker := time.NewTicker(interval)
    
    // 3. Start job execution goroutine
    go func() {
        for {
            select {
            case <-ticker.C:
                s.executeJob(event)
            case <-s.getStopChannel(event.Id):
                ticker.Stop()
                return
            }
        }
    }()
    
    // 4. Register running job
    s.mutex.Lock()
    s.runningJobs[event.Id] = ticker
    s.mutex.Unlock()
    
    return nil
}

func (s *SupportSchedulerService) executeJob(event ScheduleEvent) {
    execution := JobExecution{
        JobId:       event.Id,
        ExecutionId: models.GenerateUUID(),
        StartTime:   time.Now(),
        Status:      "RUNNING",
    }
    
    // Execute job logic based on event configuration
    result, err := s.performJobExecution(event)
    
    execution.Result = result
    execution.Error = err
    execution.Status = "COMPLETED"
    if err != nil {
        execution.Status = "FAILED"
    }
    
    s.logJobExecution(execution)
}
```

## 🚀 Application Services Implementation

### App Service Configurable (Port 59700)

#### Data Processing Pipeline Engine
```go
type PipelineEngine struct {
    pipelines    map[string]Pipeline
    transformers map[string]TransformFunction
    targets      map[string]TargetFunction
    mutex        sync.RWMutex
}

type TransformFunction func(data interface{}, params map[string]interface{}) (interface{}, error)
type TargetFunction func(data interface{}, config Target) error
```

#### Pipeline Execution Algorithm
```go
func (s *ApplicationService) executePipeline(event models.Event, pipeline Pipeline) PipelineResult {
    result := PipelineResult{
        PipelineId: pipeline.Id,
        StartTime:  time.Now(),
        Status:     "PROCESSING",
    }
    
    // 1. Execute transform chain
    processedData := event
    for i, transform := range pipeline.Transforms {
        transformResult, err := s.executeTransform(processedData, transform)
        if err != nil {
            result.Status = "FAILED"
            result.Error = err
            return result
        }
        processedData = transformResult
        result.TransformResults = append(result.TransformResults, 
            fmt.Sprintf("Transform %d: %s completed", i+1, transform.Type))
    }
    
    // 2. Execute target output
    if err := s.executeTarget(processedData, pipeline.Target); err != nil {
        result.Status = "FAILED"
        result.Error = err
        return result
    }
    
    result.Status = "COMPLETED"
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    
    return result
}
```

#### Transform Implementation
```go
func (s *ApplicationService) executeFilterTransform(data interface{}, params map[string]interface{}) (interface{}, error) {
    event, ok := data.(models.Event)
    if !ok {
        return nil, errors.New("invalid data type for filter transform")
    }
    
    condition := params["condition"].(string)
    resourceName := params["resource"].(string)
    
    filteredReadings := []models.Reading{}
    for _, reading := range event.Readings {
        if reading.ResourceName == resourceName {
            if s.evaluateCondition(reading, condition) {
                filteredReadings = append(filteredReadings, reading)
            }
        }
    }
    
    event.Readings = filteredReadings
    return event, nil
}

func (s *ApplicationService) evaluateCondition(reading models.Reading, condition string) bool {
    // Simple condition evaluation (in production, use expression parser)
    if strings.Contains(condition, ">") {
        parts := strings.Split(condition, ">")
        if len(parts) == 2 {
            fieldName := strings.TrimSpace(parts[0])
            threshold := strings.TrimSpace(parts[1])
            
            if fieldName == "temperature" {
                value, _ := strconv.ParseFloat(reading.SimpleReading.Value, 64)
                thresholdValue, _ := strconv.ParseFloat(threshold, 64)
                return value > thresholdValue
            }
        }
    }
    return false
}
```

## 🎮 Device Services Implementation

### Device Virtual Service (Port 59900)

#### Virtual Device Simulation Engine
```go
type VirtualDeviceEngine struct {
    devices      map[string]*VirtualDevice
    generators   map[string]DataGenerator
    publishers   map[string]DataPublisher
    mutex        sync.RWMutex
}

type DataGenerator interface {
    Generate() interface{}
    GetSensorType() string
    Configure(params map[string]interface{}) error
}

type VirtualDevice struct {
    Id            string
    Name          string
    DeviceType    string
    Generator     DataGenerator
    Publisher     DataPublisher
    Interval      time.Duration
    IsRunning     bool
    LastReading   time.Time
    stopChannel   chan bool
}
```

#### Data Generation Algorithm
```go
func (s *DeviceVirtualService) generateDeviceData(device *VirtualDevice) {
    ticker := time.NewTicker(device.Interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // 1. Generate sensor reading
            rawValue := device.Generator.Generate()
            
            // 2. Create EdgeX reading
            reading := s.createReading(device, rawValue)
            
            // 3. Create EdgeX event
            event := models.NewEvent(device.ProfileName, device.Name, device.ServiceName)
            event.AddReading(reading)
            
            // 4. Publish to Core Data (simulated)
            s.publishEvent(event)
            
            // 5. Update device state
            device.LastReading = time.Now()
            
        case <-device.stopChannel:
            s.logger.Infof("Stopping data generation for device: %s", device.Name)
            return
        }
    }
}

func (s *DeviceVirtualService) createReading(device *VirtualDevice, value interface{}) models.Reading {
    var reading models.Reading
    
    switch device.DeviceType {
    case "temperature":
        temp := value.(float64)
        reading = models.NewSimpleReading(
            device.ProfileName,
            device.Name,
            "Temperature",
            common.ValueTypeFloat64,
            fmt.Sprintf("%.2f", temp),
        )
        reading.SimpleReading.Units = "Celsius"
        
    case "humidity":
        humidity := value.(float64)
        reading = models.NewSimpleReading(
            device.ProfileName,
            device.Name,
            "Humidity",
            common.ValueTypeFloat64,
            fmt.Sprintf("%.2f", humidity),
        )
        reading.SimpleReading.Units = "Percent"
        
    case "pressure":
        pressure := value.(float64)
        reading = models.NewSimpleReading(
            device.ProfileName,
            device.Name,
            "Pressure",
            common.ValueTypeFloat64,
            fmt.Sprintf("%.2f", pressure),
        )
        reading.SimpleReading.Units = "hPa"
    }
    
    return reading
}
```

## 🏗️ Infrastructure Implementation

### 1. Bootstrap Service Lifecycle
```go
type BootstrapLifecycle struct {
    serviceInfo ServiceInfo
    handlers    []BootstrapHandler
    dic         *DIContainer
    server      *http.Server
    wg          sync.WaitGroup
    logger      *logrus.Logger
}

func (b *BootstrapLifecycle) Start() error {
    // 1. Initialize dependency injection container
    b.dic = NewDIContainer()
    
    // 2. Initialize all handlers
    ctx := context.Background()
    for _, handler := range b.handlers {
        if !handler.Initialize(ctx, &b.wg, b.dic) {
            return errors.New("handler initialization failed")
        }
    }
    
    // 3. Start HTTP server
    b.server = &http.Server{
        Addr:    ":" + b.serviceInfo.Port,
        Handler: b.router,
    }
    
    go func() {
        if err := b.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            b.logger.Errorf("Server error: %v", err)
        }
    }()
    
    // 4. Setup graceful shutdown
    b.setupGracefulShutdown()
    
    return nil
}

func (b *BootstrapLifecycle) setupGracefulShutdown() {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-c
        b.logger.Info("Shutdown signal received")
        
        // Graceful shutdown with timeout
        ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
        defer cancel()
        
        if err := b.server.Shutdown(ctx); err != nil {
            b.logger.Errorf("Server shutdown error: %v", err)
        }
        
        // Wait for all goroutines to finish
        done := make(chan struct{})
        go func() {
            b.wg.Wait()
            close(done)
        }()
        
        select {
        case <-done:
            b.logger.Info("All goroutines finished")
        case <-time.After(30 * time.Second):
            b.logger.Warn("Timeout waiting for goroutines to finish")
        }
    }()
}
```

### 2. Message Bus Implementation
```go
type RedisMessageBus struct {
    client    *redis.Client
    pubsub    *redis.PubSub
    logger    *logrus.Logger
    topics    map[string][]MessageHandler
    mutex     sync.RWMutex
}

func (r *RedisMessageBus) Publish(topic string, message interface{}) error {
    // 1. Serialize message
    data, err := json.Marshal(message)
    if err != nil {
        return err
    }
    
    // 2. Publish to Redis
    return r.client.Publish(context.Background(), topic, data).Err()
}

func (r *RedisMessageBus) Subscribe(topic string, handler MessageHandler) error {
    // 1. Register handler
    r.mutex.Lock()
    r.topics[topic] = append(r.topics[topic], handler)
    r.mutex.Unlock()
    
    // 2. Subscribe to Redis topic
    r.pubsub = r.client.Subscribe(context.Background(), topic)
    
    // 3. Start message processing
    go func() {
        for msg := range r.pubsub.Channel() {
            r.processMessage(msg.Channel, msg.Payload)
        }
    }()
    
    return nil
}
```

This Low-Level Design provides comprehensive technical implementation details for all components of the complete EdgeX Foundry platform, ensuring maintainable, scalable, and robust code architecture.