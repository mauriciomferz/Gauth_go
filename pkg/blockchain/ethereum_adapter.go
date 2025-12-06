package blockchain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Prometheus metrics for blockchain operations
	blockchainOpsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "blockchain_operations_total",
			Help: "Total number of blockchain operations",
		},
		[]string{"operation", "status"},
	)

	blockchainOpsDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "blockchain_operations_duration_seconds",
			Help:    "Duration of blockchain operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	blockchainGasUsed = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "blockchain_gas_used",
			Help:    "Gas used by blockchain operations",
			Buckets: []float64{50000, 100000, 200000, 500000, 1000000},
		},
		[]string{"operation"},
	)
)

// EthereumAdapter implements BlockchainRegistry using Ethereum/Polygon
type EthereumAdapter struct {
	client          *ethclient.Client
	contractAddress common.Address
	contract        *PoARegistryContract
	privateKey      *ecdsa.PrivateKey
	publicAddress   common.Address
	chainID         *big.Int
	config          *EthereumConfig
}

// EthereumConfig holds Ethereum adapter configuration
type EthereumConfig struct {
	RPCURL          string        `json:"rpc_url"`
	PrivateKey      string        `json:"private_key"`
	ContractAddress string        `json:"contract_address"`
	ChainID         int64         `json:"chain_id"`
	GasLimit        uint64        `json:"gas_limit"`
	GasPrice        *big.Int      `json:"gas_price"`
	MaxGasPrice     *big.Int      `json:"max_gas_price"`
	ConfirmBlocks   int           `json:"confirm_blocks"`
	TxTimeout       time.Duration `json:"tx_timeout"`
	NetworkName     string        `json:"network_name"` // "ethereum", "polygon", "sepolia"
}

// PoARegistryContract wraps the smart contract ABI
type PoARegistryContract struct {
	abi     abi.ABI
	address common.Address
	client  *ethclient.Client
}

// TransactionStatus represents blockchain transaction status
type TransactionStatus struct {
	TxHash        string
	BlockNumber   uint64
	Confirmations int
	Status        string // "pending", "confirmed", "failed"
	GasUsed       uint64
	Error         string
}

// NewEthereumAdapter creates a new Ethereum blockchain adapter
func NewEthereumAdapter(config *EthereumConfig) (*EthereumAdapter, error) {
	// Connect to Ethereum client
	client, err := ethclient.Dial(config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Derive public address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}
	publicAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Parse contract address
	contractAddress := common.HexToAddress(config.ContractAddress)

	// Load contract ABI
	contractABI, err := abi.JSON(strings.NewReader(PoARegistryABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	contract := &PoARegistryContract{
		abi:     contractABI,
		address: contractAddress,
		client:  client,
	}

	chainID := big.NewInt(config.ChainID)

	adapter := &EthereumAdapter{
		client:          client,
		contractAddress: contractAddress,
		contract:        contract,
		privateKey:      privateKey,
		publicAddress:   publicAddress,
		chainID:         chainID,
		config:          config,
	}

	return adapter, nil
}

// RegisterPoA registers a PoA on the blockchain
func (e *EthereumAdapter) RegisterPoA(ctx context.Context, record *PoARecord) (string, error) {
	startTime := time.Now()
	defer func() {
		blockchainOpsDuration.WithLabelValues("register_poa").Observe(time.Since(startTime).Seconds())
	}()

	// Prepare transaction data
	data, err := e.contract.abi.Pack(
		"registerPoA",
		record.ID,
		e.hashString(record.IssuerID),
		e.hashString(record.GranteeID),
		record.ScopeHash,
		record.AttestationHash,
		record.MetadataHash,
		record.MetadataURI,
		big.NewInt(record.ValidFrom.Unix()),
		big.NewInt(record.ValidUntil.Unix()),
	)
	if err != nil {
		blockchainOpsTotal.WithLabelValues("register_poa", "error").Inc()
		return "", fmt.Errorf("failed to pack transaction data: %w", err)
	}

	// Send transaction
	txHash, gasUsed, err := e.sendTransaction(ctx, data)
	if err != nil {
		blockchainOpsTotal.WithLabelValues("register_poa", "failure").Inc()
		return "", err
	}

	blockchainGasUsed.WithLabelValues("register_poa").Observe(float64(gasUsed))
	blockchainOpsTotal.WithLabelValues("register_poa", "success").Inc()

	return txHash, nil
}

// RevokePoA revokes a PoA on the blockchain
func (e *EthereumAdapter) RevokePoA(ctx context.Context, poaID string, revokedBy string, reason string) (string, error) {
	startTime := time.Now()
	defer func() {
		blockchainOpsDuration.WithLabelValues("revoke_poa").Observe(time.Since(startTime).Seconds())
	}()

	data, err := e.contract.abi.Pack(
		"revokePoA",
		poaID,
		e.hashString(revokedBy),
		reason,
	)
	if err != nil {
		blockchainOpsTotal.WithLabelValues("revoke_poa", "error").Inc()
		return "", fmt.Errorf("failed to pack transaction data: %w", err)
	}

	txHash, gasUsed, err := e.sendTransaction(ctx, data)
	if err != nil {
		blockchainOpsTotal.WithLabelValues("revoke_poa", "failure").Inc()
		return "", err
	}

	blockchainGasUsed.WithLabelValues("revoke_poa").Observe(float64(gasUsed))
	blockchainOpsTotal.WithLabelValues("revoke_poa", "success").Inc()

	return txHash, nil
}

// UpdatePoAStatus updates the status of a PoA on blockchain
func (e *EthereumAdapter) UpdatePoAStatus(ctx context.Context, poaID string, status string) (string, error) {
	startTime := time.Now()
	defer func() {
		blockchainOpsDuration.WithLabelValues("update_status").Observe(time.Since(startTime).Seconds())
	}()

	data, err := e.contract.abi.Pack(
		"updatePoAStatus",
		poaID,
		status,
	)
	if err != nil {
		blockchainOpsTotal.WithLabelValues("update_status", "error").Inc()
		return "", fmt.Errorf("failed to pack transaction data: %w", err)
	}

	txHash, gasUsed, err := e.sendTransaction(ctx, data)
	if err != nil {
		blockchainOpsTotal.WithLabelValues("update_status", "failure").Inc()
		return "", err
	}

	blockchainGasUsed.WithLabelValues("update_status").Observe(float64(gasUsed))
	blockchainOpsTotal.WithLabelValues("update_status", "success").Inc()

	return txHash, nil
}

// VerifyPoAOnChain verifies a PoA exists on the blockchain
func (e *EthereumAdapter) VerifyPoAOnChain(ctx context.Context, poaID string) (*BlockchainPoARecord, error) {
	startTime := time.Now()
	defer func() {
		blockchainOpsDuration.WithLabelValues("verify_poa").Observe(time.Since(startTime).Seconds())
	}()

	// Call verifyPoA function
	data, err := e.contract.abi.Pack("verifyPoA", poaID)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &e.contractAddress,
		Data: data,
	}

	result, err := e.client.CallContract(ctx, msg, nil)
	if err != nil {
		blockchainOpsTotal.WithLabelValues("verify_poa", "failure").Inc()
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	// Unpack result: (bool exists, bool active, bool revoked)
	var out struct {
		Exists  bool
		Active  bool
		Revoked bool
	}
	err = e.contract.abi.UnpackIntoInterface(&out, "verifyPoA", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	if !out.Exists {
		return nil, fmt.Errorf("PoA not found on blockchain")
	}

	// Get full PoA details
	record, err := e.getPoADetails(ctx, poaID)
	if err != nil {
		return nil, err
	}

	blockchainOpsTotal.WithLabelValues("verify_poa", "success").Inc()

	return record, nil
}

// getPoADetails retrieves full PoA details from blockchain
func (e *EthereumAdapter) getPoADetails(ctx context.Context, poaID string) (*BlockchainPoARecord, error) {
	data, err := e.contract.abi.Pack("getPoA", poaID)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &e.contractAddress,
		Data: data,
	}

	result, err := e.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	// Unpack result
	var out struct {
		IssuerIdHash     [32]byte
		GranteeIdHash    [32]byte
		ScopeHash        string
		AttestationHash  string
		MetadataHash     string
		MetadataURI      string
		ValidFrom        *big.Int
		ValidUntil       *big.Int
		Status           string
		RegisteredAt     *big.Int
		Revoked          bool
		RevokedAt        *big.Int
		RevokedByHash    [32]byte
		RevocationReason string
	}
	err = e.contract.abi.UnpackIntoInterface(&out, "getPoA", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	record := &BlockchainPoARecord{
		ID:              poaID,
		IssuerIDHash:    fmt.Sprintf("0x%x", out.IssuerIdHash),
		GranteeIDHash:   fmt.Sprintf("0x%x", out.GranteeIdHash),
		ScopeHash:       out.ScopeHash,
		AttestationHash: out.AttestationHash,
		MetadataHash:    out.MetadataHash,
		MetadataURI:     out.MetadataURI,
		ValidFrom:       time.Unix(out.ValidFrom.Int64(), 0),
		ValidUntil:      time.Unix(out.ValidUntil.Int64(), 0),
		Status:          out.Status,
		RegisteredAt:    time.Unix(out.RegisteredAt.Int64(), 0),
		Revoked:         out.Revoked,
		TxHash:          "", // Would need to track separately
		BlockNumber:     0,  // Would need to track separately
	}

	if out.Revoked {
		record.RevokedAt = time.Unix(out.RevokedAt.Int64(), 0)
		record.RevokedByHash = fmt.Sprintf("0x%x", out.RevokedByHash)
		record.RevocationReason = out.RevocationReason
	}

	return record, nil
}

// GetPublicVerificationURL returns public verification URL
func (e *EthereumAdapter) GetPublicVerificationURL(poaID string) string {
	// Return blockchain explorer URL based on network
	switch e.config.NetworkName {
	case "ethereum":
		return fmt.Sprintf("https://etherscan.io/address/%s#readContract", e.contractAddress.Hex())
	case "polygon":
		return fmt.Sprintf("https://polygonscan.com/address/%s#readContract", e.contractAddress.Hex())
	case "sepolia":
		return fmt.Sprintf("https://sepolia.etherscan.io/address/%s#readContract", e.contractAddress.Hex())
	default:
		return fmt.Sprintf("blockchain://%s/poa/%s", e.contractAddress.Hex(), poaID)
	}
}

// ListPoAsByIssuer lists all PoAs issued by a principal
func (e *EthereumAdapter) ListPoAsByIssuer(ctx context.Context, issuerID string) ([]*BlockchainPoARecord, error) {
	issuerHash := e.hashString(issuerID)

	data, err := e.contract.abi.Pack("getPoAsByIssuer", issuerHash)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &e.contractAddress,
		Data: data,
	}

	result, err := e.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	var poaIDs []string
	err = e.contract.abi.UnpackIntoInterface(&poaIDs, "getPoAsByIssuer", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	records := make([]*BlockchainPoARecord, 0, len(poaIDs))
	for _, poaID := range poaIDs {
		record, err := e.getPoADetails(ctx, poaID)
		if err != nil {
			continue // Skip errors
		}
		records = append(records, record)
	}

	return records, nil
}

// ListPoAsByGrantee lists all PoAs granted to a representative
func (e *EthereumAdapter) ListPoAsByGrantee(ctx context.Context, granteeID string) ([]*BlockchainPoARecord, error) {
	granteeHash := e.hashString(granteeID)

	data, err := e.contract.abi.Pack("getPoAsByGrantee", granteeHash)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &e.contractAddress,
		Data: data,
	}

	result, err := e.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	var poaIDs []string
	err = e.contract.abi.UnpackIntoInterface(&poaIDs, "getPoAsByGrantee", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	records := make([]*BlockchainPoARecord, 0, len(poaIDs))
	for _, poaID := range poaIDs {
		record, err := e.getPoADetails(ctx, poaID)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// RegisterAIAgent registers an AI agent on the blockchain
func (e *EthereumAdapter) RegisterAIAgent(ctx context.Context, registration *AIAgentRegistration) (string, string, error) {
	startTime := time.Now()
	defer func() {
		blockchainOpsDuration.WithLabelValues("register_ai_agent").Observe(time.Since(startTime).Seconds())
	}()

	data, err := e.contract.abi.Pack(
		"registerAIAgent",
		registration.AgentID,
		e.hashString(registration.OwnerID),
		registration.AgentName,
		registration.AgentType,
	)
	if err != nil {
		blockchainOpsTotal.WithLabelValues("register_ai_agent", "error").Inc()
		return "", "", fmt.Errorf("failed to pack transaction data: %w", err)
	}

	txHash, gasUsed, err := e.sendTransaction(ctx, data)
	if err != nil {
		blockchainOpsTotal.WithLabelValues("register_ai_agent", "failure").Inc()
		return "", "", err
	}

	blockchainGasUsed.WithLabelValues("register_ai_agent").Observe(float64(gasUsed))
	blockchainOpsTotal.WithLabelValues("register_ai_agent", "success").Inc()

	// Return registration ID (in this case, same as agent ID) and tx hash
	return registration.AgentID, txHash, nil
}

// GetBlockchainHeight returns current blockchain height
func (e *EthereumAdapter) GetBlockchainHeight(ctx context.Context) (int64, error) {
	header, err := e.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get block header: %w", err)
	}
	return header.Number.Int64(), nil
}

// GetTransactionStatus returns the status of a transaction
func (e *EthereumAdapter) GetTransactionStatus(ctx context.Context, txHash string) (*TransactionStatus, error) {
	hash := common.HexToHash(txHash)

	receipt, err := e.client.TransactionReceipt(ctx, hash)
	if err != nil {
		// Transaction might be pending
		return &TransactionStatus{
			TxHash: txHash,
			Status: "pending",
		}, nil
	}

	currentBlock, err := e.client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %w", err)
	}

	confirmations := int(currentBlock - receipt.BlockNumber.Uint64())

	status := &TransactionStatus{
		TxHash:        txHash,
		BlockNumber:   receipt.BlockNumber.Uint64(),
		Confirmations: confirmations,
		GasUsed:       receipt.GasUsed,
	}

	if receipt.Status == types.ReceiptStatusSuccessful {
		status.Status = "confirmed"
	} else {
		status.Status = "failed"
	}

	return status, nil
}

// sendTransaction sends a transaction to the blockchain
func (e *EthereumAdapter) sendTransaction(ctx context.Context, data []byte) (string, uint64, error) {
	// Get nonce
	nonce, err := e.client.PendingNonceAt(ctx, e.publicAddress)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice := e.config.GasPrice
	if gasPrice == nil {
		gasPrice, err = e.client.SuggestGasPrice(ctx)
		if err != nil {
			return "", 0, fmt.Errorf("failed to suggest gas price: %w", err)
		}
	}

	// Cap gas price if max is set
	if e.config.MaxGasPrice != nil && gasPrice.Cmp(e.config.MaxGasPrice) > 0 {
		gasPrice = e.config.MaxGasPrice
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		e.contractAddress,
		big.NewInt(0), // No ETH transfer
		e.config.GasLimit,
		gasPrice,
		data,
	)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(e.chainID), e.privateKey)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = e.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send transaction: %w", err)
	}

	txHash := signedTx.Hash().Hex()

	// Wait for receipt with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.TxTimeout)
	defer cancel()

	receipt, err := bind.WaitMined(timeoutCtx, e.client, signedTx)
	if err != nil {
		// Transaction sent but receipt not received - it's still pending
		return txHash, 0, nil
	}

	if receipt.Status == types.ReceiptStatusFailed {
		return txHash, receipt.GasUsed, fmt.Errorf("transaction failed")
	}

	return txHash, receipt.GasUsed, nil
}

// hashString creates a hash of a string (for privacy on blockchain)
func (e *EthereumAdapter) hashString(s string) [32]byte {
	return crypto.Keccak256Hash([]byte(s))
}

// Close closes the Ethereum client connection
func (e *EthereumAdapter) Close() {
	e.client.Close()
}

// HealthCheck checks if the Ethereum client is connected
func (e *EthereumAdapter) HealthCheck(ctx context.Context) error {
	_, err := e.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("ethereum client health check failed: %w", err)
	}
	return nil
}

// PoARegistryABI is the ABI for the PoA Registry smart contract
const PoARegistryABI = `[
	{
		"inputs": [
			{"name": "poaId", "type": "string"},
			{"name": "issuerIdHash", "type": "bytes32"},
			{"name": "granteeIdHash", "type": "bytes32"},
			{"name": "scopeHash", "type": "string"},
			{"name": "attestationHash", "type": "string"},
			{"name": "metadataHash", "type": "string"},
			{"name": "metadataURI", "type": "string"},
			{"name": "validFrom", "type": "uint256"},
			{"name": "validUntil", "type": "uint256"}
		],
		"name": "registerPoA",
		"outputs": [{"name": "", "type": "bool"}],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "poaId", "type": "string"},
			{"name": "revokedBy", "type": "bytes32"},
			{"name": "reason", "type": "string"}
		],
		"name": "revokePoA",
		"outputs": [{"name": "", "type": "bool"}],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "poaId", "type": "string"},
			{"name": "status", "type": "string"}
		],
		"name": "updatePoAStatus",
		"outputs": [{"name": "", "type": "bool"}],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [{"name": "poaId", "type": "string"}],
		"name": "verifyPoA",
		"outputs": [
			{"name": "exists", "type": "bool"},
			{"name": "active", "type": "bool"},
			{"name": "revoked", "type": "bool"}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [{"name": "poaId", "type": "string"}],
		"name": "getPoA",
		"outputs": [
			{"name": "issuerIdHash", "type": "bytes32"},
			{"name": "granteeIdHash", "type": "bytes32"},
			{"name": "scopeHash", "type": "string"},
			{"name": "attestationHash", "type": "string"},
			{"name": "metadataHash", "type": "string"},
			{"name": "metadataURI", "type": "string"},
			{"name": "validFrom", "type": "uint256"},
			{"name": "validUntil", "type": "uint256"},
			{"name": "status", "type": "string"},
			{"name": "registeredAt", "type": "uint256"},
			{"name": "revoked", "type": "bool"},
			{"name": "revokedAt", "type": "uint256"},
			{"name": "revokedByHash", "type": "bytes32"},
			{"name": "revocationReason", "type": "string"}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [{"name": "issuerIdHash", "type": "bytes32"}],
		"name": "getPoAsByIssuer",
		"outputs": [{"name": "", "type": "string[]"}],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [{"name": "granteeIdHash", "type": "bytes32"}],
		"name": "getPoAsByGrantee",
		"outputs": [{"name": "", "type": "string[]"}],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "agentId", "type": "string"},
			{"name": "ownerIdHash", "type": "bytes32"},
			{"name": "agentName", "type": "string"},
			{"name": "agentType", "type": "string"}
		],
		"name": "registerAIAgent",
		"outputs": [{"name": "", "type": "bool"}],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`
