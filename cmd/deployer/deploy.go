package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mauriciomferz/AgentAuth/pkg/blockchain"
)

func main() {
	log.SetFlags(0)
	log.Println("AgentAuth+ Smart Contract Deployer")
	log.Println("==============================")

	// 1. Load Configuration
	rpcURL := os.Getenv("AGENTAUTH_ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatalf("Error: AGENTAUTH_ETH_RPC_URL environment variable is not set")
	}

	privKeyHex := os.Getenv("AGENTAUTH_ETH_PRIVATE_KEY")
	if privKeyHex == "" {
		log.Fatalf("Error: AGENTAUTH_ETH_PRIVATE_KEY environment variable is not set")
	}

	bytecodeFile := "contracts/build/PoARegistry.bin"
	if len(os.Args) > 1 {
		bytecodeFile = os.Args[1]
	}

	// 2. Read Bytecode
	bytecodeBytes, err := os.ReadFile(bytecodeFile)
	if err != nil {
		log.Fatalf("Error reading bytecode file '%s': %v\n(Did you compile the contract? see Task.md)", bytecodeFile, err)
	}
	bytecode := strings.TrimSpace(string(bytecodeBytes))
	if !strings.HasPrefix(bytecode, "0x") {
		bytecode = "0x" + bytecode
	}

	// 3. Connect to RPC
	log.Printf("Connecting to %s...", rpcURL)
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to ETH client: %v", err)
	}

	// 4. Setup Signer
	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		log.Fatalf("Invalid private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("Error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Failed to retrieve nonce: %v", err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("Failed to suggest gas price: %v", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatalf("Failed to create transactor: %v", err)
	}
	auth.Nonce = big.NewInt(int64(nonce)) // #nosec G115
	auth.Value = big.NewInt(0)            // in wei
	auth.GasLimit = uint64(3000000)       // 3M gas limit
	auth.GasPrice = gasPrice

	log.Printf("Deploying from: %s", fromAddress.Hex())
	log.Printf("Chain ID: %v", chainID)
	log.Printf("Gas Price: %v", gasPrice)

	// 5. Deploy Contract
	parsedABI, err := abi.JSON(strings.NewReader(blockchain.PoARegistryABI))
	if err != nil {
		log.Fatalf("Failed to parse ABI: %v", err)
	}

	log.Println("Sending transaction...")
	address, tx, _, err := bind.DeployContract(auth, parsedABI, common.FromHex(bytecode), client)
	if err != nil {
		log.Fatalf("Failed to deploy contract: %v", err)
	}

	log.Printf("Transaction sent: %s", tx.Hash().Hex())
	log.Println("Waiting for confirmation...")

	// 6. Wait for Confirmation
	addressStr := address.Hex()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Printf("WaitMined failed: %v", err)
		return
	}

	if receipt.Status == types.ReceiptStatusFailed {
		log.Printf("Deployment failed! (Status: 0)")
		return
	}

	log.Println("SUCCESS! Contract deployed.")
	log.Printf("Contract Address: %s", addressStr)
	log.Printf("Gas Used: %d", receipt.GasUsed)

	// Create a verify URL
	etherscanURL := fmt.Sprintf("https://sepolia.etherscan.io/address/%s", addressStr)
	log.Printf("Verify at: %s", etherscanURL)
}
