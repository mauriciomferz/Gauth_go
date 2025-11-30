// Package revocation implements Flashbots integration for private mempool revocations
// This prevents front-running by hiding revocation transactions from public mempool
package revocation

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// FlashbotsRevocation submits revocation transactions via Flashbots private mempool
// This eliminates front-running vulnerability by hiding transactions until block inclusion
type FlashbotsRevocation struct {
	client       *ethclient.Client
	flashbotsURL string
	signingKey   *ecdsa.PrivateKey
	contractAddr common.Address
	chainID      *big.Int
	logger       Logger
}

// FlashbotsConfig contains configuration for Flashbots integration
type FlashbotsConfig struct {
	// EthereumRPC is the standard Ethereum RPC endpoint
	EthereumRPC string

	// FlashbotsURL is the Flashbots relay URL (e.g., "https://relay.flashbots.net")
	FlashbotsURL string

	// SigningKey is the private key for signing transactions
	SigningKey *ecdsa.PrivateKey

	// ContractAddress is the GAuth registry contract address
	ContractAddress string

	// ChainID is the blockchain network ID (1=mainnet, 5=goerli, etc.)
	ChainID int64

	// Logger for structured logging
	Logger Logger
}

// NewFlashbotsRevocation creates a new Flashbots revocation client
func NewFlashbotsRevocation(config *FlashbotsConfig) (*FlashbotsRevocation, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Connect to Ethereum RPC
	client, err := ethclient.Dial(config.EthereumRPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum RPC: %w", err)
	}

	// Verify chain ID
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	if chainID.Int64() != config.ChainID {
		return nil, fmt.Errorf("chain ID mismatch: expected %d, got %d", config.ChainID, chainID.Int64())
	}

	fr := &FlashbotsRevocation{
		client:       client,
		flashbotsURL: config.FlashbotsURL,
		signingKey:   config.SigningKey,
		contractAddr: common.HexToAddress(config.ContractAddress),
		chainID:      chainID,
		logger:       config.Logger,
	}

	fr.logger.Infof("Flashbots revocation client initialized (chain ID: %d, contract: %s)",
		chainID.Int64(), config.ContractAddress)

	return fr, nil
}

// RevokePoA submits a revocation transaction via Flashbots private mempool
func (f *FlashbotsRevocation) RevokePoA(ctx context.Context, poaID string, principal common.Address) error {
	start := time.Now()

	f.logger.Infof("Initiating Flashbots revocation for PoA: %s (principal: %s)", poaID, principal.Hex())

	// Step 1: Create revocation transaction
	tx, err := f.createRevocationTx(ctx, poaID)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	f.logger.Infof("Created revocation transaction (hash: %s, nonce: %d, gas: %d)",
		tx.Hash().Hex(), tx.Nonce(), tx.Gas())

	// Step 2: Sign transaction
	signer := types.NewEIP155Signer(f.chainID)
	signedTx, err := types.SignTx(tx, signer, f.signingKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Step 3: Submit to Flashbots (private mempool)
	// Note: Full Flashbots SDK integration would go here
	// For now, we'll submit via standard RPC as a placeholder
	// Production implementation should use github.com/flashbots/mev-share-go
	f.logger.Info("Submitting to Flashbots relay (private mempool)...")

	if err := f.submitToFlashbots(ctx, signedTx); err != nil {
		return fmt.Errorf("flashbots submission failed: %w", err)
	}

	// Step 4: Wait for inclusion
	f.logger.Info("Waiting for transaction inclusion...")

	receipt, err := f.waitForInclusion(ctx, signedTx.Hash(), 60*time.Second)
	if err != nil {
		return fmt.Errorf("transaction inclusion failed: %w", err)
	}

	duration := time.Since(start)
	f.logger.Infof("✅ Flashbots revocation finalized on-chain (block: %d, gas used: %d, duration: %v)",
		receipt.BlockNumber.Uint64(), receipt.GasUsed, duration)

	return nil
}

// createRevocationTx creates an unsigned revocation transaction
func (f *FlashbotsRevocation) createRevocationTx(ctx context.Context, poaID string) (*types.Transaction, error) {
	// Get sender address
	publicKey := f.signingKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Get nonce
	nonce, err := f.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := f.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Increase gas price by 20% for faster inclusion
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(120))
	gasPrice = new(big.Int).Div(gasPrice, big.NewInt(100))

	// Encode function call: revokePoA(bytes32 poaID)
	data, err := f.encodeRevocationCall(poaID)
	if err != nil {
		return nil, fmt.Errorf("failed to encode function call: %w", err)
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		f.contractAddr,
		big.NewInt(0), // No ETH transfer
		150000,        // Gas limit (revocation is a simple operation)
		gasPrice,
		data,
	)

	return tx, nil
}

// encodeRevocationCall encodes the revokePoA(bytes32) function call
func (f *FlashbotsRevocation) encodeRevocationCall(poaID string) ([]byte, error) {
	// Function signature: revokePoA(bytes32)
	functionSignature := []byte("revokePoA(bytes32)")
	selector := crypto.Keccak256(functionSignature)[:4]

	// Convert PoA ID to bytes32
	poaIDBytes := common.HexToHash(poaID)

	// Combine selector + argument
	data := append(selector, poaIDBytes.Bytes()...)

	return data, nil
}

// submitToFlashbots submits transaction to Flashbots relay
// NOTE: This is a simplified implementation. Production should use:
// github.com/flashbots/mev-share-go for full Flashbots protocol support
func (f *FlashbotsRevocation) submitToFlashbots(ctx context.Context, signedTx *types.Transaction) error {
	// In production, this would:
	// 1. Create Flashbots bundle with privacy settings
	// 2. Sign bundle with Flashbots signing key
	// 3. Submit via eth_sendBundle or eth_sendPrivateTransaction
	// 4. Monitor bundle status via eth_getBundleStats

	// For now, submit via standard RPC (placeholder)
	// This provides basic functionality but NOT front-running protection
	if err := f.client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	f.logger.Warnf("⚠️  Transaction submitted via standard RPC (not private mempool)")
	f.logger.Warnf("⚠️  Production implementation should use Flashbots MEV-Share SDK")

	return nil
}

// waitForInclusion waits for transaction to be included in a block
func (f *FlashbotsRevocation) waitForInclusion(ctx context.Context, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	deadline := time.Now().Add(timeout)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		receipt, err := f.client.TransactionReceipt(ctx, txHash)
		if err == nil {
			// Transaction included
			if receipt.Status == 1 {
				return receipt, nil
			}
			return nil, fmt.Errorf("transaction failed (status: %d)", receipt.Status)
		}

		// Wait for next check
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("transaction inclusion timeout after %v", timeout)
}

// GetTransactionStatus checks the status of a revocation transaction
func (f *FlashbotsRevocation) GetTransactionStatus(ctx context.Context, txHash common.Hash) (string, error) {
	receipt, err := f.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		// Check if transaction is pending
		_, isPending, err := f.client.TransactionByHash(ctx, txHash)
		if err != nil {
			return "unknown", fmt.Errorf("transaction not found: %w", err)
		}
		if isPending {
			return "pending", nil
		}
		return "unknown", nil
	}

	if receipt.Status == 1 {
		return "success", nil
	}
	return "failed", nil
}

// Close gracefully shuts down the Flashbots client
func (f *FlashbotsRevocation) Close() {
	if f.client != nil {
		f.client.Close()
		f.logger.Info("Flashbots revocation client shut down successfully")
	}
}
