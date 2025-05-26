package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/common"
)

// ServiceInfo contains service identification information
type ServiceInfo struct {
	ServiceName    string
	ServiceVersion string
	Port           string
}

// BootstrapHandler interface for service initialization
type BootstrapHandler interface {
	Initialize(ctx context.Context, wg *sync.WaitGroup, dic *DIContainer) bool
}

// DIContainer provides dependency injection
type DIContainer struct {
	services map[string]interface{}
	mutex    sync.RWMutex
}

// NewDIContainer creates a new dependency injection container
func NewDIContainer() *DIContainer {
	return &DIContainer{
		services: make(map[string]interface{}),
	}
}

// Add adds a service to the container
func (c *DIContainer) Add(name string, service interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.services[name] = service
}

// Get retrieves a service from the container
func (c *DIContainer) Get(name string) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.services[name]
}

// Bootstrap starts the EdgeX service with proper lifecycle management
func Bootstrap(
	serviceInfo ServiceInfo,
	handlers []BootstrapHandler,
	router *mux.Router,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	dic := NewDIContainer()
	dic.Add(common.LoggingClientName, logger)

	var wg sync.WaitGroup

	// Initialize all bootstrap handlers
	for _, handler := range handlers {
		if !handler.Initialize(ctx, &wg, dic) {
			logger.Error("Failed to initialize bootstrap handler")
			os.Exit(1)
		}
	}

	// Setup HTTP server
	server := &http.Server{
		Addr:    ":" + serviceInfo.Port,
		Handler: router,
	}

	// Start HTTP server in goroutine
	go func() {
		logger.Infof("Starting %s service on port %s", serviceInfo.ServiceName, serviceInfo.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP server error: %v", err)
			cancel()
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("Shutdown signal received")
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("All goroutines finished")
	case <-time.After(30 * time.Second):
		logger.Warn("Timeout waiting for goroutines to finish")
	}

	logger.Infof("%s service stopped", serviceInfo.ServiceName)
}

// AddCommonRoutes adds standard EdgeX routes to the router
func AddCommonRoutes(router *mux.Router, serviceName string, serviceVersion string) {
	router.HandleFunc(common.ApiPingRoute, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"apiVersion": "%s",
			"timestamp": "%s",
			"serviceName": "%s"
		}`, serviceVersion, time.Now().Format(time.RFC3339), serviceName)
	}).Methods("GET")

	router.HandleFunc(common.ApiVersionRoute, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"version": "%s",
			"serviceName": "%s"
		}`, serviceVersion, serviceName)
	}).Methods("GET")

	router.HandleFunc(common.ApiConfigRoute, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"config": "Configuration endpoint for %s"
		}`, serviceName)
	}).Methods("GET")
}