package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Hell0W0rID/edgex-go-clone/pkg/bootstrap"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/common"
	"github.com/Hell0W0rID/edgex-go-clone/pkg/core-contracts/models"
)

// Notification represents a system notification
type Notification struct {
	Id          string                 `json:"id"`
	Category    string                 `json:"category"`
	Content     string                 `json:"content"`
	ContentType string                 `json:"contentType"`
	Description string                 `json:"description"`
	Labels      []string               `json:"labels"`
	Sender      string                 `json:"sender"`
	Severity    string                 `json:"severity"`
	Status      string                 `json:"status"`
	Created     int64                  `json:"created"`
	Modified    int64                  `json:"modified"`
}

// Subscription represents a notification subscription
type Subscription struct {
	Id           string            `json:"id"`
	Name         string            `json:"name"`
	Channels     []Channel         `json:"channels"`
	Categories   []string          `json:"categories"`
	Labels       []string          `json:"labels"`
	Receiver     string            `json:"receiver"`
	Description  string            `json:"description"`
	ResendLimit  int               `json:"resendLimit"`
	ResendInterval string          `json:"resendInterval"`
	Created      int64             `json:"created"`
	Modified     int64             `json:"modified"`
}

// Channel represents a notification channel (email, SMS, etc.)
type Channel struct {
	Type       string            `json:"type"`
	Host       string            `json:"host,omitempty"`
	Port       int               `json:"port,omitempty"`
	Recipients []string          `json:"recipients"`
	Properties map[string]string `json:"properties,omitempty"`
}

// SupportNotificationsService handles notifications and subscriptions
type SupportNotificationsService struct {
	logger        *logrus.Logger
	notifications map[string]Notification
	subscriptions map[string]Subscription
	mutex         sync.RWMutex
}

// NewSupportNotificationsService creates a new support notifications service
func NewSupportNotificationsService(logger *logrus.Logger) *SupportNotificationsService {
	return &SupportNotificationsService{
		logger:        logger,
		notifications: make(map[string]Notification),
		subscriptions: make(map[string]Subscription),
	}
}

// Initialize implements the BootstrapHandler interface
func (s *SupportNotificationsService) Initialize(ctx context.Context, wg *sync.WaitGroup, dic *bootstrap.DIContainer) bool {
	s.logger.Info("Initializing Support Notifications Service")
	
	// Add service to DI container
	dic.Add("SupportNotificationsService", s)
	
	s.logger.Info("Support Notifications Service initialization completed")
	return true
}

// AddRoutes adds support notifications specific routes
func (s *SupportNotificationsService) AddRoutes(router *mux.Router) {
	// Notification routes
	router.HandleFunc("/api/v3/notification", s.addNotification).Methods("POST")
	router.HandleFunc("/api/v3/notification/all", s.getAllNotifications).Methods("GET")
	router.HandleFunc("/api/v3/notification/id/{id}", s.getNotificationById).Methods("GET")
	router.HandleFunc("/api/v3/notification/id/{id}", s.deleteNotification).Methods("DELETE")
	router.HandleFunc("/api/v3/notification/category/{category}", s.getNotificationsByCategory).Methods("GET")
	router.HandleFunc("/api/v3/notification/label/{label}", s.getNotificationsByLabel).Methods("GET")
	router.HandleFunc("/api/v3/notification/status/{status}", s.getNotificationsByStatus).Methods("GET")
	
	// Subscription routes
	router.HandleFunc("/api/v3/subscription", s.addSubscription).Methods("POST")
	router.HandleFunc("/api/v3/subscription/all", s.getAllSubscriptions).Methods("GET")
	router.HandleFunc("/api/v3/subscription/id/{id}", s.getSubscriptionById).Methods("GET")
	router.HandleFunc("/api/v3/subscription/id/{id}", s.updateSubscription).Methods("PUT")
	router.HandleFunc("/api/v3/subscription/id/{id}", s.deleteSubscription).Methods("DELETE")
	router.HandleFunc("/api/v3/subscription/name/{name}", s.getSubscriptionByName).Methods("GET")
	
	s.logger.Info("Support Notifications routes registered")
}

// Notification handlers

// addNotification handles POST /api/v3/notification
func (s *SupportNotificationsService) addNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var notification Notification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		s.logger.Errorf("Failed to decode notification: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Generate ID and timestamps
	notification.Id = models.GenerateUUID()
	notification.Created = time.Now().UnixNano() / int64(time.Millisecond)
	notification.Modified = notification.Created
	
	// Set defaults
	if notification.Status == "" {
		notification.Status = "NEW"
	}
	if notification.ContentType == "" {
		notification.ContentType = "text/plain"
	}
	if notification.Severity == "" {
		notification.Severity = "NORMAL"
	}
	
	s.mutex.Lock()
	s.notifications[notification.Id] = notification
	s.mutex.Unlock()
	
	// Process notification (send to subscribers)
	go s.processNotification(notification)
	
	s.logger.Infof("Notification created: %s", notification.Id)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusCreated,
		"id":         notification.Id,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getAllNotifications handles GET /api/v3/notification/all
func (s *SupportNotificationsService) getAllNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	s.mutex.RLock()
	notifications := make([]Notification, 0, len(s.notifications))
	for _, notification := range s.notifications {
		notifications = append(notifications, notification)
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"totalCount":    len(notifications),
		"notifications": notifications,
	}
	
	json.NewEncoder(w).Encode(response)
}

// getNotificationById handles GET /api/v3/notification/id/{id}
func (s *SupportNotificationsService) getNotificationById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.RLock()
	notification, exists := s.notifications[id]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":   common.ServiceVersion,
		"statusCode":   http.StatusOK,
		"notification": notification,
	}
	
	json.NewEncoder(w).Encode(response)
}

// processNotification sends notification to all matching subscribers
func (s *SupportNotificationsService) processNotification(notification Notification) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	for _, subscription := range s.subscriptions {
		if s.matchesSubscription(notification, subscription) {
			s.sendNotification(notification, subscription)
		}
	}
	
	// Update notification status
	notification.Status = "PROCESSED"
	notification.Modified = time.Now().UnixNano() / int64(time.Millisecond)
	s.notifications[notification.Id] = notification
}

// matchesSubscription checks if notification matches subscription criteria
func (s *SupportNotificationsService) matchesSubscription(notification Notification, subscription Subscription) bool {
	// Check categories
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
	
	// Check labels
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
	
	return true
}

// sendNotification sends notification through subscription channels
func (s *SupportNotificationsService) sendNotification(notification Notification, subscription Subscription) {
	for _, channel := range subscription.Channels {
		switch channel.Type {
		case "EMAIL":
			s.sendEmailNotification(notification, channel)
		case "SMS":
			s.sendSMSNotification(notification, channel)
		case "WEBHOOK":
			s.sendWebhookNotification(notification, channel)
		default:
			s.logger.Warnf("Unknown channel type: %s", channel.Type)
		}
	}
}

// sendEmailNotification simulates sending email notification
func (s *SupportNotificationsService) sendEmailNotification(notification Notification, channel Channel) {
	s.logger.Infof("Sending email notification: %s to %v", notification.Content, channel.Recipients)
	// In a real implementation, this would integrate with an email service
}

// sendSMSNotification simulates sending SMS notification
func (s *SupportNotificationsService) sendSMSNotification(notification Notification, channel Channel) {
	s.logger.Infof("Sending SMS notification: %s to %v", notification.Content, channel.Recipients)
	// In a real implementation, this would integrate with an SMS service
}

// sendWebhookNotification simulates sending webhook notification
func (s *SupportNotificationsService) sendWebhookNotification(notification Notification, channel Channel) {
	s.logger.Infof("Sending webhook notification: %s to %s", notification.Content, channel.Host)
	// In a real implementation, this would make HTTP requests to webhook URLs
}

// Subscription handlers

// addSubscription handles POST /api/v3/subscription
func (s *SupportNotificationsService) addSubscription(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	var subscription Subscription
	if err := json.NewDecoder(r.Body).Decode(&subscription); err != nil {
		s.logger.Errorf("Failed to decode subscription: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Generate ID and timestamps
	subscription.Id = models.GenerateUUID()
	subscription.Created = time.Now().UnixNano() / int64(time.Millisecond)
	subscription.Modified = subscription.Created
	
	// Set defaults
	if subscription.ResendLimit == 0 {
		subscription.ResendLimit = 3
	}
	if subscription.ResendInterval == "" {
		subscription.ResendInterval = "5m"
	}
	
	s.mutex.Lock()
	s.subscriptions[subscription.Id] = subscription
	s.mutex.Unlock()
	
	s.logger.Infof("Subscription created: %s", subscription.Name)
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusCreated,
		"id":         subscription.Id,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getAllSubscriptions handles GET /api/v3/subscription/all
func (s *SupportNotificationsService) getAllSubscriptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	s.mutex.RLock()
	subscriptions := make([]Subscription, 0, len(s.subscriptions))
	for _, subscription := range s.subscriptions {
		subscriptions = append(subscriptions, subscription)
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"totalCount":    len(subscriptions),
		"subscriptions": subscriptions,
	}
	
	json.NewEncoder(w).Encode(response)
}

// getNotificationsByCategory handles GET /api/v3/notification/category/{category}
func (s *SupportNotificationsService) getNotificationsByCategory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	category := vars["category"]
	
	s.mutex.RLock()
	var categoryNotifications []Notification
	for _, notification := range s.notifications {
		if notification.Category == category {
			categoryNotifications = append(categoryNotifications, notification)
		}
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"totalCount":    len(categoryNotifications),
		"notifications": categoryNotifications,
	}
	
	json.NewEncoder(w).Encode(response)
}

// Additional handlers for other endpoints would follow the same pattern...

// getNotificationsByLabel handles GET /api/v3/notification/label/{label}
func (s *SupportNotificationsService) getNotificationsByLabel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	label := vars["label"]
	
	s.mutex.RLock()
	var labelNotifications []Notification
	for _, notification := range s.notifications {
		for _, notifLabel := range notification.Labels {
			if notifLabel == label {
				labelNotifications = append(labelNotifications, notification)
				break
			}
		}
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"totalCount":    len(labelNotifications),
		"notifications": labelNotifications,
	}
	
	json.NewEncoder(w).Encode(response)
}

// getNotificationsByStatus handles GET /api/v3/notification/status/{status}
func (s *SupportNotificationsService) getNotificationsByStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	status := vars["status"]
	
	s.mutex.RLock()
	var statusNotifications []Notification
	for _, notification := range s.notifications {
		if notification.Status == status {
			statusNotifications = append(statusNotifications, notification)
		}
	}
	s.mutex.RUnlock()
	
	response := map[string]interface{}{
		"apiVersion":    common.ServiceVersion,
		"statusCode":    http.StatusOK,
		"totalCount":    len(statusNotifications),
		"notifications": statusNotifications,
	}
	
	json.NewEncoder(w).Encode(response)
}

// getSubscriptionById handles GET /api/v3/subscription/id/{id}
func (s *SupportNotificationsService) getSubscriptionById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.RLock()
	subscription, exists := s.subscriptions[id]
	s.mutex.RUnlock()
	
	if !exists {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":   common.ServiceVersion,
		"statusCode":   http.StatusOK,
		"subscription": subscription,
	}
	
	json.NewEncoder(w).Encode(response)
}

// updateSubscription handles PUT /api/v3/subscription/id/{id}
func (s *SupportNotificationsService) updateSubscription(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	var updatedSubscription Subscription
	if err := json.NewDecoder(r.Body).Decode(&updatedSubscription); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	s.mutex.Lock()
	existingSubscription, exists := s.subscriptions[id]
	if exists {
		updatedSubscription.Id = id
		updatedSubscription.Created = existingSubscription.Created
		updatedSubscription.Modified = time.Now().UnixNano() / int64(time.Millisecond)
		s.subscriptions[id] = updatedSubscription
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Subscription updated successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// deleteSubscription handles DELETE /api/v3/subscription/id/{id}
func (s *SupportNotificationsService) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.Lock()
	_, exists := s.subscriptions[id]
	if exists {
		delete(s.subscriptions, id)
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Subscription deleted successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}

// getSubscriptionByName handles GET /api/v3/subscription/name/{name}
func (s *SupportNotificationsService) getSubscriptionByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	name := vars["name"]
	
	s.mutex.RLock()
	var foundSubscription *Subscription
	for _, subscription := range s.subscriptions {
		if subscription.Name == name {
			foundSubscription = &subscription
			break
		}
	}
	s.mutex.RUnlock()
	
	if foundSubscription == nil {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion":   common.ServiceVersion,
		"statusCode":   http.StatusOK,
		"subscription": *foundSubscription,
	}
	
	json.NewEncoder(w).Encode(response)
}

// deleteNotification handles DELETE /api/v3/notification/id/{id}
func (s *SupportNotificationsService) deleteNotification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(common.ContentType, common.ContentTypeJSON)
	
	vars := mux.Vars(r)
	id := vars["id"]
	
	s.mutex.Lock()
	_, exists := s.notifications[id]
	if exists {
		delete(s.notifications, id)
	}
	s.mutex.Unlock()
	
	if !exists {
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}
	
	response := map[string]interface{}{
		"apiVersion": common.ServiceVersion,
		"statusCode": http.StatusOK,
		"message":    "Notification deleted successfully",
	}
	
	json.NewEncoder(w).Encode(response)
}