package web

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ProtocolFlowStep represents a step in the GAuth protocol flow
type ProtocolFlowStep struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Icon        string                 `json:"icon"`
	Status      string                 `json:"status"` // pending, in-progress, completed, error
	Substeps    []ProtocolFlowSubstep  `json:"substeps"`
	StartTime   *time.Time             `json:"start_time,omitempty"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ProtocolFlowSubstep represents a substep within a protocol step
type ProtocolFlowSubstep struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// ProtocolFlowState tracks the complete state of a protocol flow session
type ProtocolFlowState struct {
	SessionID      string                       `json:"session_id"`
	CurrentStep    string                       `json:"current_step,omitempty"`
	CurrentSubstep string                       `json:"current_substep,omitempty"`
	Steps          map[string]*ProtocolFlowStep `json:"steps"`
	History        []ProtocolFlowHistoryEntry   `json:"history"`
	Progress       int                          `json:"progress"` // 0-100
	CreatedAt      time.Time                    `json:"created_at"`
	UpdatedAt      time.Time                    `json:"updated_at"`
	Metadata       map[string]interface{}       `json:"metadata,omitempty"`
}

// ProtocolFlowHistoryEntry records navigation history
type ProtocolFlowHistoryEntry struct {
	Step      string    `json:"step"`
	Substep   string    `json:"substep,omitempty"`
	Action    string    `json:"action"` // navigate, complete, error
	Timestamp time.Time `json:"timestamp"`
}

// ProtocolFlowManager manages protocol flow sessions
type ProtocolFlowManager struct {
	sessions map[string]*ProtocolFlowState
	mu       sync.RWMutex
}

// NewProtocolFlowManager creates a new flow manager
func NewProtocolFlowManager() *ProtocolFlowManager {
	return &ProtocolFlowManager{
		sessions: make(map[string]*ProtocolFlowState),
	}
}

// CreateSession creates a new protocol flow session
func (m *ProtocolFlowManager) CreateSession(sessionID string) *ProtocolFlowState {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	state := &ProtocolFlowState{
		SessionID: sessionID,
		Steps:     m.initializeSteps(),
		History:   []ProtocolFlowHistoryEntry{},
		Progress:  0,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}

	m.sessions[sessionID] = state
	return state
}

// GetSession retrieves a session by ID
func (m *ProtocolFlowManager) GetSession(sessionID string) *ProtocolFlowState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionID]
}

// UpdateStepStatus updates the status of a step
func (m *ProtocolFlowManager) UpdateStepStatus(sessionID, stepID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	step, exists := state.Steps[stepID]
	if !exists {
		return ErrStepNotFound
	}

	now := time.Now()
	step.Status = status

	if status == "in-progress" && step.StartTime == nil {
		step.StartTime = &now
	}

	if status == "completed" && step.EndTime == nil {
		step.EndTime = &now
	}

	state.UpdatedAt = now
	state.History = append(state.History, ProtocolFlowHistoryEntry{
		Step:      stepID,
		Action:    status,
		Timestamp: now,
	})

	m.calculateProgress(state)
	return nil
}

// NavigateToStep updates current step/substep
func (m *ProtocolFlowManager) NavigateToStep(sessionID, stepID, substepID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	state.CurrentStep = stepID
	state.CurrentSubstep = substepID
	state.UpdatedAt = time.Now()

	state.History = append(state.History, ProtocolFlowHistoryEntry{
		Step:      stepID,
		Substep:   substepID,
		Action:    "navigate",
		Timestamp: time.Now(),
	})

	return nil
}

// CompleteSubstep marks a substep as completed
func (m *ProtocolFlowManager) CompleteSubstep(sessionID, stepID, substepID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	step, exists := state.Steps[stepID]
	if !exists {
		return ErrStepNotFound
	}

	// Find and update substep
	for i := range step.Substeps {
		if step.Substeps[i].ID == substepID {
			now := time.Now()
			step.Substeps[i].Status = "completed"
			step.Substeps[i].EndTime = &now
			break
		}
	}

	// Check if all substeps are completed
	allCompleted := true
	for _, substep := range step.Substeps {
		if substep.Status != "completed" {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		m.UpdateStepStatus(sessionID, stepID, "completed")
	}

	state.UpdatedAt = time.Now()
	return nil
}

// initializeSteps creates the default protocol steps
func (m *ProtocolFlowManager) initializeSteps() map[string]*ProtocolFlowStep {
	return map[string]*ProtocolFlowStep{
		"subscription": {
			ID:          "subscription",
			Name:        "Subscription",
			Description: "Authorization server registration and client setup",
			Icon:        "📝",
			Status:      "pending",
			Substeps: []ProtocolFlowSubstep{
				{ID: "register", Name: "Register Client", Status: "pending"},
				{ID: "configure", Name: "Configure Scopes", Status: "pending"},
				{ID: "credentials", Name: "Obtain Credentials", Status: "pending"},
			},
		},
		"matching": {
			ID:          "matching",
			Name:        "Matching",
			Description: "PoA Definition validation and capability matching",
			Icon:        "🔍",
			Status:      "pending",
			Substeps: []ProtocolFlowSubstep{
				{ID: "validate_poa", Name: "Validate PoA Definition", Status: "pending"},
				{ID: "check_capabilities", Name: "Check AI Capabilities", Status: "pending"},
				{ID: "verify_jurisdiction", Name: "Verify Jurisdiction", Status: "pending"},
				{ID: "match_policies", Name: "Match Policies", Status: "pending"},
			},
		},
		"subset_request": {
			ID:          "subset_request",
			Name:        "Subset/Request",
			Description: "Authorization request with scope subset selection",
			Icon:        "🎯",
			Status:      "pending",
			Substeps: []ProtocolFlowSubstep{
				{ID: "create_request", Name: "Create Auth Request", Status: "pending"},
				{ID: "select_scope", Name: "Select Scope Subset", Status: "pending"},
				{ID: "pdp_decision", Name: "PDP Decision", Status: "pending"},
				{ID: "generate_token", Name: "Generate Token", Status: "pending"},
			},
		},
		"enforcement": {
			ID:          "enforcement",
			Name:        "Enforcement",
			Description: "PEP enforcement (supply-side and demand-side)",
			Icon:        "🛡️",
			Status:      "pending",
			Substeps: []ProtocolFlowSubstep{
				{ID: "supply_pep", Name: "Supply-Side PEP", Status: "pending"},
				{ID: "demand_pep", Name: "Demand-Side PEP", Status: "pending"},
				{ID: "disclosure", Name: "Disclosure Requirements", Status: "pending"},
				{ID: "audit", Name: "Audit Logging", Status: "pending"},
			},
		},
		"verification": {
			ID:          "verification",
			Name:        "Verification",
			Description: "Token verification and PVP identity validation",
			Icon:        "✓",
			Status:      "pending",
			Substeps: []ProtocolFlowSubstep{
				{ID: "validate_token", Name: "Validate Token", Status: "pending"},
				{ID: "verify_signature", Name: "Verify Signature", Status: "pending"},
				{ID: "check_revocation", Name: "Check Revocation", Status: "pending"},
				{ID: "pvp_check", Name: "PVP Identity Check", Status: "pending"},
			},
		},
		"audit": {
			ID:          "audit",
			Name:        "Audit & Compliance",
			Description: "Audit trail and compliance reporting",
			Icon:        "📊",
			Status:      "pending",
			Substeps: []ProtocolFlowSubstep{
				{ID: "log_event", Name: "Log Audit Event", Status: "pending"},
				{ID: "compliance_check", Name: "Compliance Check", Status: "pending"},
				{ID: "report_generation", Name: "Generate Reports", Status: "pending"},
			},
		},
	}
}

// calculateProgress calculates completion percentage
func (m *ProtocolFlowManager) calculateProgress(state *ProtocolFlowState) {
	total := len(state.Steps)
	completed := 0

	for _, step := range state.Steps {
		if step.Status == "completed" {
			completed++
		}
	}

	state.Progress = (completed * 100) / total
}

// API Handlers

func (s *BetaServer) handleProtocolFlowPage(c *gin.Context) {
	c.HTML(http.StatusOK, "protocol-flow.html", nil)
}

func (s *BetaServer) apiProtocolFlowCreateSession(c *gin.Context) {
	var req struct {
		SessionID string                 `json:"session_id"`
		Metadata  map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.SessionID == "" {
		req.SessionID = generateSessionID()
	}

	state := s.protocolFlowManager.CreateSession(req.SessionID)
	if req.Metadata != nil {
		state.Metadata = req.Metadata
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"session": state,
	})
}

func (s *BetaServer) apiProtocolFlowGetSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	state := s.protocolFlowManager.GetSession(sessionID)
	if state == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"session": state,
	})
}

func (s *BetaServer) apiProtocolFlowNavigate(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req struct {
		Step    string `json:"step"`
		Substep string `json:"substep"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := s.protocolFlowManager.NavigateToStep(sessionID, req.Step, req.Substep); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "navigation updated",
	})
}

func (s *BetaServer) apiProtocolFlowUpdateStep(c *gin.Context) {
	sessionID := c.Param("session_id")
	stepID := c.Param("step_id")

	var req struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := s.protocolFlowManager.UpdateStepStatus(sessionID, stepID, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "step status updated",
	})
}

func (s *BetaServer) apiProtocolFlowCompleteSubstep(c *gin.Context) {
	sessionID := c.Param("session_id")

	var req struct {
		Step    string `json:"step"`
		Substep string `json:"substep"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := s.protocolFlowManager.CompleteSubstep(sessionID, req.Step, req.Substep); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "substep completed",
	})
}

// Utility functions

func generateSessionID() string {
	return time.Now().Format("20060102150405")
}

// Errors
var (
	ErrSessionNotFound = &ProtocolFlowError{Code: "session_not_found", Message: "Protocol flow session not found"}
	ErrStepNotFound    = &ProtocolFlowError{Code: "step_not_found", Message: "Protocol step not found"}
)

type ProtocolFlowError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ProtocolFlowError) Error() string {
	return e.Message
}

func (e *ProtocolFlowError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{
		Code:    e.Code,
		Message: e.Message,
	})
}
