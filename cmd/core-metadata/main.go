package main

import (
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/bootstrap"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/common"
	"github.com/Hell0W0rID/edgex-go-clone/internal/core/metadata"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	serviceInfo := bootstrap.ServiceInfo{
		ServiceName:    common.CoreMetaDataServiceKey,
		ServiceVersion: common.ServiceVersion,
		Port:          "59881",
	}

	// Create router
	router := mux.NewRouter()

	// Add common EdgeX routes
	bootstrap.AddCommonRoutes(router, serviceInfo.ServiceName, serviceInfo.ServiceVersion)

	// Initialize core metadata service
	metadataService := metadata.NewCoreMetadataService(logger)

	// Create bootstrap handlers
	handlers := []bootstrap.BootstrapHandler{
		metadataService,
	}

	// Add service-specific routes
	metadataService.AddRoutes(router)

	logger.Infof("Starting %s service", serviceInfo.ServiceName)

	// Bootstrap the service
	bootstrap.Bootstrap(serviceInfo, handlers, router)
}