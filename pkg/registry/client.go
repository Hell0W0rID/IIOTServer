package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

// RegistryClient defines the service registry interface
type RegistryClient interface {
	Register(service ServiceRegistration) error
	Deregister(serviceID string) error
	GetService(serviceName string) ([]ServiceEndpoint, error)
	GetAllServices() (map[string][]ServiceEndpoint, error)
	IsServiceAvailable(serviceName string) bool
	WatchService(serviceName string, callback ServiceChangeCallback) error
}

// ServiceRegistration represents service registration information
type ServiceRegistration struct {
	ServiceID   string
	ServiceName string
	Host        string
	Port        int
	Tags        []string
	Check       HealthCheck
}

// ServiceEndpoint represents a service endpoint
type ServiceEndpoint struct {
	ServiceID   string
	ServiceName string
	Address     string
	Port        int
	Tags        []string
	Status      string
}

// HealthCheck represents service health check configuration
type HealthCheck struct {
	HTTP                           string
	Interval                       string
	Timeout                        string
	DeregisterCriticalServiceAfter string
}

// ServiceChangeCallback defines callback for service changes
type ServiceChangeCallback func(serviceName string, endpoints []ServiceEndpoint)

// ConsulRegistryClient implements RegistryClient using Consul
type ConsulRegistryClient struct {
	client   *api.Client
	logger   *logrus.Logger
	watchers map[string]ServiceChangeCallback
	mutex    sync.RWMutex
}

// NewConsulRegistryClient creates a new Consul registry client
func NewConsulRegistryClient(address string, logger *logrus.Logger) (*ConsulRegistryClient, error) {
	config := api.DefaultConfig()
	config.Address = address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	return &ConsulRegistryClient{
		client:   client,
		logger:   logger,
		watchers: make(map[string]ServiceChangeCallback),
	}, nil
}

// Register registers a service with the registry
func (c *ConsulRegistryClient) Register(service ServiceRegistration) error {
	registration := &api.AgentServiceRegistration{
		ID:      service.ServiceID,
		Name:    service.ServiceName,
		Address: service.Host,
		Port:    service.Port,
		Tags:    service.Tags,
		Check: &api.AgentServiceCheck{
			HTTP:                           service.Check.HTTP,
			Interval:                       service.Check.Interval,
			Timeout:                        service.Check.Timeout,
			DeregisterCriticalServiceAfter: service.Check.DeregisterCriticalServiceAfter,
		},
	}

	err := c.client.Agent().ServiceRegister(registration)
	if err != nil {
		c.logger.Errorf("Failed to register service %s: %v", service.ServiceName, err)
		return err
	}

	c.logger.Infof("Successfully registered service: %s", service.ServiceName)
	return nil
}

// Deregister removes a service from the registry
func (c *ConsulRegistryClient) Deregister(serviceID string) error {
	err := c.client.Agent().ServiceDeregister(serviceID)
	if err != nil {
		c.logger.Errorf("Failed to deregister service %s: %v", serviceID, err)
		return err
	}

	c.logger.Infof("Successfully deregistered service: %s", serviceID)
	return nil
}

// GetService retrieves all instances of a service
func (c *ConsulRegistryClient) GetService(serviceName string) ([]ServiceEndpoint, error) {
	services, _, err := c.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s: %w", serviceName, err)
	}

	var endpoints []ServiceEndpoint
	for _, service := range services {
		endpoint := ServiceEndpoint{
			ServiceID:   service.Service.ID,
			ServiceName: service.Service.Service,
			Address:     service.Service.Address,
			Port:        service.Service.Port,
			Tags:        service.Service.Tags,
			Status:      service.Checks.AggregatedStatus(),
		}
		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

// GetAllServices retrieves all registered services
func (c *ConsulRegistryClient) GetAllServices() (map[string][]ServiceEndpoint, error) {
	services, _, err := c.client.Catalog().Services(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get all services: %w", err)
	}

	result := make(map[string][]ServiceEndpoint)
	for serviceName := range services {
		endpoints, err := c.GetService(serviceName)
		if err != nil {
			c.logger.Warnf("Failed to get endpoints for service %s: %v", serviceName, err)
			continue
		}
		result[serviceName] = endpoints
	}

	return result, nil
}

// IsServiceAvailable checks if a service is available
func (c *ConsulRegistryClient) IsServiceAvailable(serviceName string) bool {
	endpoints, err := c.GetService(serviceName)
	if err != nil {
		return false
	}

	for _, endpoint := range endpoints {
		if endpoint.Status == "passing" {
			return true
		}
	}

	return false
}

// WatchService watches for changes in a service
func (c *ConsulRegistryClient) WatchService(serviceName string, callback ServiceChangeCallback) error {
	c.mutex.Lock()
	c.watchers[serviceName] = callback
	c.mutex.Unlock()

	go c.watchServiceChanges(serviceName)
	
	c.logger.Infof("Started watching service: %s", serviceName)
	return nil
}

// watchServiceChanges monitors service changes
func (c *ConsulRegistryClient) watchServiceChanges(serviceName string) {
	var lastIndex uint64
	
	for {
		services, meta, err := c.client.Health().Service(serviceName, "", true, &api.QueryOptions{
			WaitIndex: lastIndex,
			WaitTime:  time.Minute,
		})
		
		if err != nil {
			c.logger.Errorf("Error watching service %s: %v", serviceName, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if meta.LastIndex != lastIndex {
			lastIndex = meta.LastIndex
			
			var endpoints []ServiceEndpoint
			for _, service := range services {
				endpoint := ServiceEndpoint{
					ServiceID:   service.Service.ID,
					ServiceName: service.Service.Service,
					Address:     service.Service.Address,
					Port:        service.Service.Port,
					Tags:        service.Service.Tags,
					Status:      service.Checks.AggregatedStatus(),
				}
				endpoints = append(endpoints, endpoint)
			}

			c.mutex.RLock()
			if callback, exists := c.watchers[serviceName]; exists {
				callback(serviceName, endpoints)
			}
			c.mutex.RUnlock()
		}
	}
}

// CreateServiceRegistration creates a service registration
func CreateServiceRegistration(serviceID, serviceName, host string, port int, healthCheckURL string) ServiceRegistration {
	return ServiceRegistration{
		ServiceID:   serviceID,
		ServiceName: serviceName,
		Host:        host,
		Port:        port,
		Tags:        []string{"edgex", "microservice"},
		Check: HealthCheck{
			HTTP:                           healthCheckURL,
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}
}