package authz

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	pkagentauthz "github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/policy"
)

// PolicyEvaluator abstracts the policy engine.
type PolicyEvaluator interface {
	Evaluate(ctx context.Context, req policy.EvalRequest) (policy.EvalDecision, error)
}

// Authorizer abstracts the legacy memory authorizer.
type Authorizer interface {
	GetMetricsSnapshot() pkagentauthz.MetricsSnapshot
	Authorize(ctx context.Context, req pkagentauthz.Request) (pkagentauthz.Decision, error)
}

// MetricsProvider is replaced by direct usage of metrics.Metrics to allow optional SnapshotEx
type API struct {
	Authorizer Authorizer
	Policy     PolicyEvaluator
	Metrics    metrics.Metrics
}

func NewAPI(authz Authorizer, policy PolicyEvaluator, m metrics.Metrics) *API {
	return &API{
		Authorizer: authz,
		Policy:     policy,
		Metrics:    m,
	}
}

func (a *API) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/v1/beta/authz")
	{
		group.POST("/evaluate", a.Evaluate)
		group.GET("/metrics", a.MetricsHandler)
		group.GET("/metrics/prometheus", a.PrometheusHandler)
		group.GET("/decisions", a.DecisionMetrics)
	}
	// Legacy alias: /api/v1/beta/metrics/decisions
	router.GET("/api/v1/beta/metrics/decisions", a.DecisionMetrics)
}

// MetricsHandler exposes MemoryAuthorizer metrics snapshot.
func (a *API) MetricsHandler(c *gin.Context) {
	if a.Authorizer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "authorizer not initialized"})
		return
	}
	snap := a.Authorizer.GetMetricsSnapshot()
	c.JSON(http.StatusOK, gin.H{"success": true, "metrics": snap, "timestamp": time.Now().Format(time.RFC3339)})
}

// DecisionMetrics exposes detailed decision/reason counts from server metrics.
func (a *API) DecisionMetrics(c *gin.Context) {
	if a.Metrics == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "decisions": gin.H{"available": false}})
		return
	}

	// Check if metrics supports SnapshotEx
	type memoryLike interface{ SnapshotEx() metrics.SnapshotStruct }
	ml, ok := a.Metrics.(memoryLike)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"decisions": gin.H{
				"available": false,
				"reason":    "metrics_backend_does_not_support_snapshot",
			},
		})
		return
	}
	snap := ml.SnapshotEx()

	// CSV export support
	wantsCSV := func() bool {
		if strings.EqualFold(c.Query("format"), "csv") {
			return true
		}
		accept := c.GetHeader("Accept")
		if accept == "" {
			return false
		}
		for _, part := range strings.Split(accept, ",") {
			p := strings.ToLower(strings.TrimSpace(strings.Split(part, ";")[0]))
			if p == "text/csv" || p == "application/csv" {
				return true
			}
		}
		return false
	}()

	if wantsCSV {
		var b strings.Builder
		b.WriteString("action,resource,outcome,reason,count\n")
		// Flatten decisions
		for k, v := range snap.DecisionBreakdown {
			// k format: action|resource|outcome
			parts := strings.Split(k, "|")
			if len(parts) >= 3 {
				b.WriteString(fmtCSV(parts[0], parts[1], parts[2], "", v))
			}
		}
		// Flatten reasons
		for k, v := range snap.DecisionReasonBreakdown {
			// k format: action|resource|outcome|reason
			parts := strings.Split(k, "|")
			if len(parts) >= 4 {
				b.WriteString(fmtCSV(parts[0], parts[1], parts[2], parts[3], v))
			}
		}
		c.Data(http.StatusOK, "text/csv", []byte(b.String()))
		return
	}

	// JSON response matching legacy format
	counts := make([]gin.H, 0, len(snap.DecisionBreakdown))
	for k, v := range snap.DecisionBreakdown {
		parts := strings.Split(k, "|")
		if len(parts) >= 3 {
			counts = append(counts, gin.H{"action": parts[0], "resource": parts[1], "outcome": parts[2], "count": v})
		}
	}

	reasons := make([]gin.H, 0, len(snap.DecisionReasonBreakdown))
	for k, v := range snap.DecisionReasonBreakdown {
		parts := strings.Split(k, "|")
		if len(parts) >= 4 {
			reasons = append(reasons, gin.H{"action": parts[0], "resource": parts[1], "outcome": parts[2], "reason": parts[3], "count": v})
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "decisions": gin.H{"counts": counts, "reasons": reasons}})
}

func fmtCSV(act, res, out, rsn string, cnt uint64) string {
	// Simple CSV escaping: if contains comma, quote.
	quote := func(s string) string {
		if strings.Contains(s, ",") {
			return `"` + s + `"`
		}
		return s
	}
	return fmt.Sprintf("%s,%s,%s,%s,%d\n", quote(act), quote(res), quote(out), quote(rsn), cnt)
}

// Evaluate performs a demo evaluation.
func (a *API) Evaluate(c *gin.Context) {
	if a.Authorizer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "authorizer not initialized"})
		return
	}
	var req struct {
		Subject  string            `json:"subject"`
		Resource string            `json:"resource"`
		Action   string            `json:"action"`
		Context  map[string]string `json:"context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Subject == "" || req.Resource == "" || req.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid payload"})
		return
	}
	// Defensive nil context handling
	if req.Context == nil {
		req.Context = make(map[string]string)
	}

	// Prefer Policy Handler
	if a.Policy != nil {
		pReq := policy.EvalRequest{
			Subject:  req.Subject,
			Action:   req.Action,
			Resource: req.Resource,
			Attrs:    req.Context,
			Now:      time.Now().UTC(),
		}
		dec, err := a.Policy.Evaluate(c.Request.Context(), pReq)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"success": true, "decision": dec})
			return
		}
	}

	// Legacy evaluation
	d, err := a.Authorizer.Authorize(c.Request.Context(), pkagentauthz.Request{
		Subject:  req.Subject,
		Resource: req.Resource,
		Action:   req.Action,
		Context:  req.Context,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "decision": d})
}

// PrometheusHandler wraps the authorizer's prometheus exposition.
func (a *API) PrometheusHandler(c *gin.Context) {
	if a.Authorizer == nil {
		c.String(http.StatusServiceUnavailable, "authorizer not initialized")
		return
	}
	// Type assert to MemoryAuthorizer for Prometheus handler
	mem, ok := a.Authorizer.(*pkagentauthz.MemoryAuthorizer)
	if !ok {
		c.String(http.StatusInternalServerError, "authorizer type not supported for prometheus")
		return
	}
	handler := pkagentauthz.PrometheusHandler(mem)
	handler.ServeHTTP(c.Writer, c.Request)
}
