package main

import (
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/bootstrap"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/common"
	"github.com/Hell0W0rID/edgex-go-clone/internal/core/command"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	serviceInfo := bootstrap.ServiceInfo{
		ServiceName:    common.CoreCommandServiceKey,
		ServiceVersion: common.ServiceVersion,
		Port:          "59882",
	}

	// Create router
	router := mux.NewRouter()

	// Add common EdgeX routes
	bootstrap.AddCommonRoutes(router, serviceInfo.ServiceName, serviceInfo.ServiceVersion)

	// Initialize core command service
	commandService := command.NewCoreCommandService(logger)

	// Create bootstrap handlers
	handlers := []bootstrap.BootstrapHandler{
		commandService,
	}

	// Add service-specific routes
	commandService.AddRoutes(router)

	logger.Infof("Starting %s service", serviceInfo.ServiceName)

	// Bootstrap the service
	bootstrap.Bootstrap(serviceInfo, handlers, router)
}