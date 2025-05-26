package secrets

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// SecretsClient defines the secrets management interface
type SecretsClient interface {
	GetSecret(path string, keys ...string) (map[string]string, error)
	StoreSecret(path string, secrets map[string]string) error
	DeleteSecret(path string) error
	ListSecrets(path string) ([]string, error)
	SecretExists(path string) (bool, error)
}

// Secret represents a secret with metadata
type Secret struct {
	Path     string            `json:"path"`
	Secrets  map[string]string `json:"secrets"`
	Created  int64             `json:"created"`
	Modified int64             `json:"modified"`
}

// InMemorySecretsClient implements SecretsClient using in-memory storage
type InMemorySecretsClient struct {
	secrets map[string]map[string]string
	logger  *logrus.Logger
	mutex   sync.RWMutex
}

// NewInMemorySecretsClient creates a new in-memory secrets client
func NewInMemorySecretsClient(logger *logrus.Logger) *InMemorySecretsClient {
	return &InMemorySecretsClient{
		secrets: make(map[string]map[string]string),
		logger:  logger,
	}
}

// GetSecret retrieves secrets from the specified path
func (s *InMemorySecretsClient) GetSecret(path string, keys ...string) (map[string]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	pathSecrets, exists := s.secrets[path]
	if !exists {
		return nil, fmt.Errorf("no secrets found at path: %s", path)
	}

	result := make(map[string]string)
	
	if len(keys) == 0 {
		// Return all secrets if no specific keys requested
		for k, v := range pathSecrets {
			result[k] = v
		}
	} else {
		// Return only requested keys
		for _, key := range keys {
			if value, found := pathSecrets[key]; found {
				result[key] = value
			}
		}
	}

	s.logger.Debugf("Retrieved %d secrets from path: %s", len(result), path)
	return result, nil
}

// StoreSecret stores secrets at the specified path
func (s *InMemorySecretsClient) StoreSecret(path string, secrets map[string]string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.secrets[path] == nil {
		s.secrets[path] = make(map[string]string)
	}

	for key, value := range secrets {
		s.secrets[path][key] = value
	}

	s.logger.Infof("Stored %d secrets at path: %s", len(secrets), path)
	return nil
}

// DeleteSecret removes secrets from the specified path
func (s *InMemorySecretsClient) DeleteSecret(path string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.secrets[path]; !exists {
		return fmt.Errorf("no secrets found at path: %s", path)
	}

	delete(s.secrets, path)
	s.logger.Infof("Deleted secrets at path: %s", path)
	return nil
}

// ListSecrets lists all secret paths
func (s *InMemorySecretsClient) ListSecrets(path string) ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var paths []string
	for secretPath := range s.secrets {
		if path == "" || secretPath == path {
			paths = append(paths, secretPath)
		}
	}

	return paths, nil
}

// SecretExists checks if secrets exist at the specified path
func (s *InMemorySecretsClient) SecretExists(path string) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s.secrets[path]
	return exists, nil
}

// SecretProvider provides common secret management functionality
type SecretProvider struct {
	client SecretsClient
	logger *logrus.Logger
}

// NewSecretProvider creates a new secret provider
func NewSecretProvider(client SecretsClient, logger *logrus.Logger) *SecretProvider {
	return &SecretProvider{
		client: client,
		logger: logger,
	}
}

// GetDatabaseCredentials retrieves database credentials
func (sp *SecretProvider) GetDatabaseCredentials(serviceName string) (username, password string, err error) {
	path := fmt.Sprintf("edgex/%s/database", serviceName)
	secrets, err := sp.client.GetSecret(path, "username", "password")
	if err != nil {
		return "", "", err
	}

	username = secrets["username"]
	password = secrets["password"]
	
	if username == "" || password == "" {
		return "", "", fmt.Errorf("incomplete database credentials for service: %s", serviceName)
	}

	return username, password, nil
}

// GetMessagingCredentials retrieves messaging credentials
func (sp *SecretProvider) GetMessagingCredentials(serviceName string) (username, password string, err error) {
	path := fmt.Sprintf("edgex/%s/messaging", serviceName)
	secrets, err := sp.client.GetSecret(path, "username", "password")
	if err != nil {
		return "", "", err
	}

	username = secrets["username"]
	password = secrets["password"]
	
	return username, password, nil
}

// StoreServiceCredentials stores credentials for a service
func (sp *SecretProvider) StoreServiceCredentials(serviceName, credType string, credentials map[string]string) error {
	path := fmt.Sprintf("edgex/%s/%s", serviceName, credType)
	return sp.client.StoreSecret(path, credentials)
}

// Common secret paths
var SecretPaths = struct {
	Database   string
	Messaging  string
	Registry   string
	External   string
}{
	Database:  "edgex/database",
	Messaging: "edgex/messaging", 
	Registry:  "edgex/registry",
	External:  "edgex/external",
}