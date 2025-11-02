package capabilities

import (
	"net/http"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/capability"
	"github.com/gin-gonic/gin"
)

// BasicDeps abstracts minimal server state required for capability listing & negotiation.
// It is implemented by the core server (BetaServer) without creating a package import cycle.
// Future extensions can grow a richer interface or split into read/write facets.
type BasicDeps interface {
	LifecycleStrict() bool
}

// localErrorResponse mirrors web.ErrorResponse to avoid import cycle.
type localErrorResponse struct {
	Success bool        `json:"success"`
	Code    string      `json:"code"`
	Error   string      `json:"error"`
	Message string      `json:"message,omitempty"`
	RFCRef  string      `json:"rfc_ref,omitempty"`
	Detail  interface{} `json:"detail,omitempty"`
}

// respondError duplicates core helper (no trace IDs) to preserve taxonomy unchanged.
func respondError(c *gin.Context, status int, code, errStr, msg, rfcRef string, detail interface{}) {
	c.JSON(status, localErrorResponse{Success: false, Code: code, Error: errStr, Message: msg, RFCRef: rfcRef, Detail: detail})
}

// RegisterBasic mounts capability listing and negotiation endpoints on an existing router group.
// Paths preserved exactly as previous in monolith: /capabilities (GET), /capabilities/negotiate (POST).
func RegisterBasic(group *gin.RouterGroup, deps BasicDeps) {
	group.GET("/capabilities", listHandler())
	group.POST("/capabilities/negotiate", negotiateHandler(deps))
}

// listHandler returns all registered capabilities from the global registry.
func listHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		list := capability.DefaultRegistry().List()
		c.JSON(http.StatusOK, gin.H{"success": true, "capabilities": list})
	}
}

// negotiateHandler performs multi-version negotiation identical to legacy apiCapabilitiesNegotiate implementation.
// Request: {"client_versions": {"cap.id": ["1.0", "1.1"], ...}}
// Response: {success, agreed: {cap.id: version}, unsupported: {cap.id: [client_versions]}, lifecycle_strict: bool}
func negotiateHandler(deps BasicDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ClientVersions map[string][]string `json:"client_versions"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ClientVersions) == 0 {
			respondError(c, http.StatusBadRequest, "capabilities_negotiate_invalid_payload", "invalid_payload", "client_versions required", "rfc111:capabilities_negotiate", nil)
			return
		}
		caps := capability.DefaultRegistry().List()
		regMap := make(map[string]capability.Capability, len(caps))
		for _, cap := range caps {
			regMap[cap.ID] = cap
		}
		agreed := make(map[string]string)
		unsupported := make(map[string][]string)
		for cid, clientVers := range req.ClientVersions {
			regCap, ok := regMap[cid]
			if !ok {
				unsupported[cid] = clientVers
				continue
			}
			serverVers := make(map[string]struct{})
			if regCap.Version != "" {
				serverVers[regCap.Version] = struct{}{}
			}
			for _, v := range regCap.Versions {
				serverVers[v] = struct{}{}
			}
			// Strict lifecycle: exclude all versions if deprecated_after passed.
			if deps.LifecycleStrict() && regCap.DeprecatedAfter != "" {
				if t, err := time.Parse(time.RFC3339, regCap.DeprecatedAfter); err == nil {
					if time.Now().After(t) { // treat capability deprecated => no versions negotiable
						serverVers = map[string]struct{}{}
					}
				}
			}
			negotiated := ""
			for _, cv := range clientVers {
				if _, ok := serverVers[cv]; ok {
					negotiated = cv
					break
				}
			}
			if negotiated == "" {
				unsupported[cid] = clientVers
			} else {
				agreed[cid] = negotiated
			}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "agreed": agreed, "unsupported": unsupported, "lifecycle_strict": deps.LifecycleStrict()})
	}
}
