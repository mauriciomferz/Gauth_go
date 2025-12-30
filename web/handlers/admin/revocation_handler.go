package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mauriciomferz/AgentAuth/pkg/revocations"
)

// RevocationHandler handles revocation transparency endpoints
type RevocationHandler struct {
	repo *revocations.Repository
}

// NewRevocationHandler creates a new revocation handler
func NewRevocationHandler(db *pgxpool.Pool) *RevocationHandler {
	return &RevocationHandler{
		repo: revocations.NewRepository(db),
	}
}

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Hash       string  `json:"hash"`
	Level      int     `json:"level"`
	Position   int     `json:"position"`
	LeftChild  *string `json:"leftChild,omitempty"`
	RightChild *string `json:"rightChild,omitempty"`
	IsLeaf     bool    `json:"isLeaf"`
	Data       *string `json:"data,omitempty"`
}

// MerkleProof represents a Merkle proof for a revoked token
type MerkleProof struct {
	TokenID   string      `json:"tokenId"`
	LeafHash  string      `json:"leafHash"`
	RootHash  string      `json:"rootHash"`
	Path      []ProofStep `json:"path"`
	Verified  bool        `json:"verified"`
	Timestamp time.Time   `json:"timestamp"`
}

// ProofStep represents a single step in a Merkle proof
type ProofStep struct {
	Hash     string `json:"hash"`
	Position string `json:"position"` // "left" or "right"
	Sibling  string `json:"sibling"`
}

// RevocationEntry represents a token revocation entry
type RevocationEntry struct {
	ID          string    `json:"id"`
	TokenID     string    `json:"tokenId"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
	RevokedBy   string    `json:"revokedBy"`
	LeafHash    string    `json:"leafHash"`
	MerkleRoot  string    `json:"merkleRoot"`
	BlockHeight int       `json:"blockHeight"`
	Verified    bool      `json:"verified"`
}

// AppendOnlyLogEntry represents an entry in the append-only log
type AppendOnlyLogEntry struct {
	Index        int       `json:"index"`
	Timestamp    time.Time `json:"timestamp"`
	Operation    string    `json:"operation"` // "append" or "verify"
	Data         string    `json:"data"`
	Hash         string    `json:"hash"`
	PreviousHash string    `json:"previousHash"`
}

// ProofGenerationRequest represents a request to generate a proof
type ProofGenerationRequest struct {
	TokenID string `json:"tokenId" binding:"required"`
}

// ProofVerificationRequest represents a request to verify a proof
type ProofVerificationRequest struct {
	ProofData string `json:"proofData" binding:"required"`
}

// VerificationResult represents the result of proof verification
type VerificationResult struct {
	Valid        bool      `json:"valid"`
	TokenID      string    `json:"tokenId"`
	LeafHash     string    `json:"leafHash"`
	RootHash     string    `json:"rootHash"`
	ComputedRoot string    `json:"computedRoot"`
	PathLength   int       `json:"pathLength"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetMerkleTree retrieves the current Merkle tree structure
func (h *RevocationHandler) GetMerkleTree(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	// Get latest tree version
	treeVersion, err := h.repo.GetLatestTreeVersion(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tree version"})
		return
	}

	// Get tree nodes
	dbNodes, err := h.repo.GetMerkleTree(c.Request.Context(), tenantID, treeVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve Merkle tree"})
		return
	}

	// Convert to response format
	nodes := make([]MerkleNode, len(dbNodes))
	for i, dbNode := range dbNodes {
		var tokenData *string
		if dbNode.TokenID != nil {
			tokenData = dbNode.TokenID
		}

		nodes[i] = MerkleNode{
			Hash:       dbNode.NodeHash,
			Level:      dbNode.Level,
			Position:   dbNode.Position,
			LeftChild:  dbNode.LeftChildHash,
			RightChild: dbNode.RightChildHash,
			IsLeaf:     dbNode.IsLeaf,
			Data:       tokenData,
		}
	}

	// Calculate depth (max level + 1)
	maxLevel := 0
	for _, node := range dbNodes {
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes":        nodes,
		"depth":        maxLevel + 1,
		"tree_version": treeVersion,
		"total_nodes":  len(nodes),
	})
}

// ListProofs lists all generated Merkle proofs
func (h *RevocationHandler) ListProofs(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	// Parse pagination
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get proofs from database
	dbProofs, total, err := h.repo.ListMerkleProofs(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve proofs"})
		return
	}

	// Convert to response format
	proofs := make([]MerkleProof, len(dbProofs))
	for i, dbProof := range dbProofs {
		// Convert JSONB proof path to ProofStep array
		path := make([]ProofStep, 0)
		for _, step := range dbProof.ProofPath {
			proofStep := ProofStep{
				Hash:     step["hash"].(string),
				Position: step["position"].(string),
				Sibling:  step["sibling"].(string),
			}
			path = append(path, proofStep)
		}

		proofs[i] = MerkleProof{
			TokenID:   dbProof.TokenID,
			LeafHash:  dbProof.LeafHash,
			RootHash:  dbProof.RootHash,
			Path:      path,
			Verified:  dbProof.Verified,
			Timestamp: dbProof.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"proofs": proofs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GenerateProof generates a Merkle proof for a specific token
func (h *RevocationHandler) GenerateProof(c *gin.Context) {
	var req ProofGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	// Get latest tree version
	treeVersion, err := h.repo.GetLatestTreeVersion(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tree version"})
		return
	}

	// Get tree nodes
	nodes, err := h.repo.GetMerkleTree(c.Request.Context(), tenantID, treeVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tree"})
		return
	}

	// Find leaf node for token
	var leafNode *revocations.MerkleNode
	for i := range nodes {
		if nodes[i].IsLeaf && nodes[i].TokenID != nil && *nodes[i].TokenID == req.TokenID {
			leafNode = &nodes[i]
			break
		}
	}

	if leafNode == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found in tree"})
		return
	}

	// Build node map for quick lookup
	nodeMap := make(map[string]*revocations.MerkleNode)
	for i := range nodes {
		key := fmt.Sprintf("%d-%d", nodes[i].Level, nodes[i].Position)
		nodeMap[key] = &nodes[i]
	}

	// Traverse from leaf to root, collecting siblings
	proofPath := make([]map[string]interface{}, 0)
	currentLevel := leafNode.Level
	currentPosition := leafNode.Position
	rootHash := ""

	for currentLevel > 0 {
		// Determine sibling position
		siblingPosition := currentPosition ^ 1 // XOR with 1 flips last bit
		siblingKey := fmt.Sprintf("%d-%d", currentLevel, siblingPosition)
		sibling := nodeMap[siblingKey]

		if sibling != nil {
			position := "right"
			if currentPosition%2 == 1 {
				position = "left"
			}

			step := map[string]interface{}{
				"hash":     sibling.NodeHash,
				"position": position,
				"sibling":  sibling.NodeHash,
			}
			proofPath = append(proofPath, step)
		}

		// Move to parent level
		currentLevel--
		currentPosition = currentPosition / 2
	}

	// Get root hash
	rootKey := "0-0"
	if root := nodeMap[rootKey]; root != nil {
		rootHash = root.NodeHash
	}

	// Store proof in database
	dbProof := &revocations.MerkleProof{
		TenantID:    tenantID,
		ProofID:     fmt.Sprintf("proof-%s-%d", req.TokenID, time.Now().Unix()),
		TokenID:     req.TokenID,
		TreeVersion: treeVersion,
		LeafHash:    leafNode.NodeHash,
		RootHash:    rootHash,
		ProofPath:   proofPath,
		Verified:    false,
	}

	err = h.repo.CreateMerkleProof(c.Request.Context(), dbProof)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store proof"})
		return
	}

	// Convert to response format
	path := make([]ProofStep, len(proofPath))
	for i, step := range proofPath {
		path[i] = ProofStep{
			Hash:     step["hash"].(string),
			Position: step["position"].(string),
			Sibling:  step["sibling"].(string),
		}
	}

	proof := MerkleProof{
		TokenID:   req.TokenID,
		LeafHash:  leafNode.NodeHash,
		RootHash:  rootHash,
		Path:      path,
		Verified:  false,
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Proof generated successfully",
		"proof":   proof,
	})
}

// VerifyProof verifies a Merkle proof
func (h *RevocationHandler) VerifyProof(c *gin.Context) {
	var req ProofVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse proof data from request
	var proofData struct {
		TokenID  string                   `json:"tokenId"`
		LeafHash string                   `json:"leafHash"`
		RootHash string                   `json:"rootHash"`
		Path     []map[string]interface{} `json:"path"`
	}

	if err := c.ShouldBindJSON(&proofData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid proof data"})
		return
	}

	// Compute root hash from leaf and path
	currentHash := proofData.LeafHash
	for _, step := range proofData.Path {
		siblingHash := step["sibling"].(string)
		position := step["position"].(string)

		var combined string
		if position == "left" {
			combined = siblingHash + currentHash
		} else {
			combined = currentHash + siblingHash
		}

		// Hash the combined value
		hash := sha256.Sum256([]byte(combined))
		currentHash = hex.EncodeToString(hash[:])
	}

	// Compare computed root with stored root
	valid := currentHash == proofData.RootHash

	// Update verification status in database if valid
	if valid {
		tenantID := c.GetString("tenant_id")
		if tenantID == "" {
			tenantID = defaultTenantID
		}

		treeVersion, _ := h.repo.GetLatestTreeVersion(c.Request.Context(), tenantID)
		dbProof, err := h.repo.GetMerkleProof(c.Request.Context(), tenantID, proofData.TokenID, treeVersion)
		if err == nil && dbProof != nil {
			_ = h.repo.UpdateProofVerification(c.Request.Context(), dbProof.ProofID, true)
		}
	}

	result := VerificationResult{
		Valid:        valid,
		TokenID:      proofData.TokenID,
		LeafHash:     proofData.LeafHash,
		RootHash:     proofData.RootHash,
		ComputedRoot: currentHash,
		PathLength:   len(proofData.Path),
		Timestamp:    time.Now(),
	}

	c.JSON(http.StatusOK, result)
}

// ListRevocations lists all token revocations
func (h *RevocationHandler) ListRevocations(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	// Parse pagination
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get revocations from database
	dbRevocations, total, err := h.repo.ListRevocations(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve revocations"})
		return
	}

	// Convert to response format
	revocations := make([]RevocationEntry, len(dbRevocations))
	for i, dbRev := range dbRevocations {
		revocations[i] = RevocationEntry{
			ID:          dbRev.ID,
			TokenID:     dbRev.TokenID,
			Reason:      dbRev.RevocationReason,
			Timestamp:   dbRev.RevokedAt,
			RevokedBy:   dbRev.RevokedBy,
			LeafHash:    dbRev.LeafHash,
			MerkleRoot:  dbRev.MerkleRoot,
			BlockHeight: dbRev.BlockHeight,
			Verified:    dbRev.Verified,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"revocations": revocations,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetAppendOnlyLog retrieves the append-only audit log
func (h *RevocationHandler) GetAppendOnlyLog(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	// Get revocations to build log from
	revocations, _, err := h.repo.ListRevocations(c.Request.Context(), tenantID, 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve log data"})
		return
	}

	// Build append-only log with hash chain
	entries := make([]AppendOnlyLogEntry, 0)
	previousHash := "0000000000000000000000000000000000000000000000000000000000000000"

	// Genesis block
	if len(revocations) > 0 {
		genesisData := "Genesis block - Revocation system initialized"
		hash := sha256.Sum256([]byte(genesisData + previousHash))
		genesisHash := hex.EncodeToString(hash[:])

		entries = append(entries, AppendOnlyLogEntry{
			Index:        0,
			Timestamp:    revocations[len(revocations)-1].RevokedAt,
			Operation:    "genesis",
			Data:         genesisData,
			Hash:         genesisHash,
			PreviousHash: previousHash,
		})
		previousHash = genesisHash
	}

	// Add revocation entries in chronological order
	for i := len(revocations) - 1; i >= 0; i-- {
		rev := revocations[i]
		data := fmt.Sprintf("Revoked %s - %s", rev.TokenID, rev.RevocationReason)
		hash := sha256.Sum256([]byte(data + previousHash + rev.LeafHash))
		entryHash := hex.EncodeToString(hash[:])

		entries = append(entries, AppendOnlyLogEntry{
			Index:        len(entries),
			Timestamp:    rev.RevokedAt,
			Operation:    "append",
			Data:         data,
			Hash:         entryHash,
			PreviousHash: previousHash,
		})
		previousHash = entryHash
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   len(entries),
	})
}

// GetRevocationMetrics retrieves metrics about the revocation system
func (h *RevocationHandler) GetRevocationMetrics(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		tenantID = defaultTenantID
	}

	// Get stats from repository
	stats, err := h.repo.GetRevocationStats(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metrics"})
		return
	}

	// Get latest tree version for node count
	treeVersion, err := h.repo.GetLatestTreeVersion(c.Request.Context(), tenantID)
	if err != nil {
		treeVersion = 0
	}

	// Get tree nodes count
	treeNodes, err := h.repo.GetMerkleTree(c.Request.Context(), tenantID, treeVersion)
	nodeCount := len(treeNodes)
	if err != nil {
		nodeCount = 0
	}

	// Calculate tree depth
	maxLevel := 0
	for _, node := range treeNodes {
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": gin.H{
			"total_revocations":    stats.TotalRevocations,
			"verified_revocations": stats.VerifiedRevocations,
			"pending_revocations":  stats.TotalRevocations - stats.VerifiedRevocations,
			"merkle_tree_depth":    maxLevel + 1,
			"merkle_tree_nodes":    nodeCount,
			"current_block_height": stats.LatestBlockHeight,
			"latest_tree_version":  stats.LatestTreeVersion,
			"revocations_last_24h": stats.RevocationsLast24h,
			"revocations_last_7d":  stats.RevocationsLast7d,
			"verification_rate":    stats.VerificationRate,
		},
	})
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

// RegisterRoutes registers all revocation transparency routes
func (h *RevocationHandler) RegisterRoutes(router *gin.RouterGroup) {
	revocation := router.Group("/revocation")
	{
		// Merkle tree operations
		revocation.GET("/merkle-tree", h.GetMerkleTree)

		// Proof operations
		revocation.GET("/proofs", h.ListProofs)
		revocation.POST("/generate-proof", h.GenerateProof)
		revocation.POST("/verify", h.VerifyProof)

		// Revocation list
		revocation.GET("/list", h.ListRevocations)

		// Append-only log
		revocation.GET("/log", h.GetAppendOnlyLog)

		// Metrics
		revocation.GET("/metrics", h.GetRevocationMetrics)
	}
}
