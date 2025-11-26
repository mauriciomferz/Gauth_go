package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/webhook"
	"github.com/gorilla/mux"
)

// WebhookHandler handles webhook-related HTTP requests
type WebhookHandler struct {
	manager    *webhook.Manager
	dispatcher *webhook.Dispatcher
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(manager *webhook.Manager, dispatcher *webhook.Dispatcher) *WebhookHandler {
	return &WebhookHandler{
		manager:    manager,
		dispatcher: dispatcher,
	}
}

// RegisterRoutes registers webhook routes
func (h *WebhookHandler) RegisterRoutes(r *mux.Router) {
	// Webhook management routes (require authentication)
	r.HandleFunc("/api/v1/webhooks", h.CreateWebhook).Methods("POST")
	r.HandleFunc("/api/v1/webhooks", h.ListWebhooks).Methods("GET")
	r.HandleFunc("/api/v1/webhooks/{id}", h.GetWebhook).Methods("GET")
	r.HandleFunc("/api/v1/webhooks/{id}", h.UpdateWebhook).Methods("PUT")
	r.HandleFunc("/api/v1/webhooks/{id}", h.DeleteWebhook).Methods("DELETE")
	r.HandleFunc("/api/v1/webhooks/{id}/regenerate", h.RegenerateSecret).Methods("POST")
	r.HandleFunc("/api/v1/webhooks/{id}/test", h.TestWebhook).Methods("POST")
	
	// Delivery tracking routes
	r.HandleFunc("/api/v1/webhooks/{id}/deliveries", h.ListDeliveries).Methods("GET")
	r.HandleFunc("/api/v1/webhooks/deliveries/{deliveryId}", h.GetDelivery).Methods("GET")
	
	// Statistics routes
	r.HandleFunc("/api/v1/webhooks/{id}/stats", h.GetWebhookStats).Methods("GET")
}

// CreateWebhook handles POST /api/v1/webhooks
func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}

	var req webhook.CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	wh, secret, err := h.manager.CreateWebhook(r.Context(), userID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Include secret only in creation response
	response := webhook.WebhookResponse{
		Webhook: *wh,
		Secret:  secret,
	}

	respondJSON(w, http.StatusCreated, response)
}

// ListWebhooks handles GET /api/v1/webhooks
func (h *WebhookHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	query := &webhook.ListWebhooksQuery{
		UserID: userID,
	}

	// Parse query parameters
	if enabled := r.URL.Query().Get("enabled"); enabled != "" {
		val := enabled == "true"
		query.Enabled = &val
	}

	webhooks, err := h.manager.ListWebhooks(r.Context(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"webhooks": webhooks,
		"count":    len(webhooks),
	})
}

// GetWebhook handles GET /api/v1/webhooks/{id}
func (h *WebhookHandler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	wh, err := h.manager.GetWebhook(r.Context(), id)
	if err != nil {
		if err == webhook.ErrWebhookNotFound {
			respondError(w, http.StatusNotFound, "not_found", "Webhook not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}

	// Don't include secret in response
	respondJSON(w, http.StatusOK, wh)
}

// UpdateWebhook handles PUT /api/v1/webhooks/{id}
func (h *WebhookHandler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req webhook.UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	wh, err := h.manager.UpdateWebhook(r.Context(), id, &req)
	if err != nil {
		if err == webhook.ErrWebhookNotFound {
			respondError(w, http.StatusNotFound, "Webhook not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, wh)
}

// DeleteWebhook handles DELETE /api/v1/webhooks/{id}
func (h *WebhookHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := h.manager.DeleteWebhook(r.Context(), id)
	if err != nil {
		if err == webhook.ErrWebhookNotFound {
			respondError(w, http.StatusNotFound, "Webhook not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Webhook deleted successfully",
	})
}

// RegenerateSecret handles POST /api/v1/webhooks/{id}/regenerate
func (h *WebhookHandler) RegenerateSecret(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	secret, err := h.manager.RegenerateSecret(r.Context(), id)
	if err != nil {
		if err == webhook.ErrWebhookNotFound {
			respondError(w, http.StatusNotFound, "Webhook not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"secret":  secret,
		"message": "Secret regenerated successfully. Store this value securely, it will not be shown again.",
	})
}

// TestWebhook handles POST /api/v1/webhooks/{id}/test
func (h *WebhookHandler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get webhook
	wh, err := h.manager.GetWebhook(r.Context(), id)
	if err != nil {
		if err == webhook.ErrWebhookNotFound {
			respondError(w, http.StatusNotFound, "Webhook not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create test event
	testEvent := &webhook.Event{
		ID:        "test-" + id,
		Type:      webhook.EventPoACreated,
		Timestamp: time.Now(),
		UserID:    userID,
		Data: map[string]interface{}{
			"test":    true,
			"message": "This is a test webhook delivery",
			"webhook": map[string]interface{}{
				"id":   wh.ID,
				"name": wh.Name,
			},
		},
	}

	// Dispatch test event
	err = h.dispatcher.Dispatch(r.Context(), testEvent)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Test webhook dispatched successfully",
		"event":   testEvent,
	})
}

// ListDeliveries handles GET /api/v1/webhooks/{id}/deliveries
func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	query := &webhook.ListDeliveriesQuery{
		WebhookID: webhookID,
		Limit:     50,
	}

	deliveries, err := h.dispatcher.ListDeliveries(r.Context(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"deliveries": deliveries,
		"count":      len(deliveries),
	})
}

// GetDelivery handles GET /api/v1/webhooks/deliveries/{deliveryId}
func (h *WebhookHandler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deliveryID := vars["deliveryId"]

	delivery, err := h.dispatcher.GetDelivery(r.Context(), deliveryID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Delivery not found")
		return
	}

	respondJSON(w, http.StatusOK, delivery)
}

// GetWebhookStats handles GET /api/v1/webhooks/{id}/stats
func (h *WebhookHandler) GetWebhookStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	stats, err := h.manager.GetWebhookStats(r.Context(), id)
	if err != nil {
		if err == webhook.ErrWebhookNotFound {
			respondError(w, http.StatusNotFound, "Webhook not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// Helper functions (reuse from existing handlers or define here)

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}
