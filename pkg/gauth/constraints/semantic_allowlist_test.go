package constraints

import (
	"math/big"
	"strings"
	"testing"
)

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestSemanticAllowList_Validate(t *testing.T) {
	tests := []struct {
		name    string
		list    *SemanticAllowList
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid allow-list with single contract",
			list: &SemanticAllowList{
				AllowedContracts: []ContractPermission{
					{
						Address:          "0xE592427A0AEce92De3Edee1F18E0157C05861564",
						AllowedFunctions: []string{"exactInputSingle((address,address,uint24,address,uint256,uint256,uint256,uint160))"},
						Description:      "Uniswap V3 Router",
					},
				},
				HardLimits: HardLimits{
					MaxTransactionValue: big.NewInt(10000),
				},
			},
			wantErr: false,
		},
		{
			name: "Empty allow-list should error",
			list: &SemanticAllowList{
				AllowedContracts: []ContractPermission{},
			},
			wantErr: true,
			errMsg:  "allow-list must specify at least one contract",
		},
		{
			name: "Invalid contract address format",
			list: &SemanticAllowList{
				AllowedContracts: []ContractPermission{
					{
						Address:          "invalid-address",
						AllowedFunctions: []string{"swap()"},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid contract address format",
		},
		{
			name: "Contract without functions should error",
			list: &SemanticAllowList{
				AllowedContracts: []ContractPermission{
					{
						Address:          "0xE592427A0AEce92De3Edee1F18E0157C05861564",
						AllowedFunctions: []string{},
					},
				},
			},
			wantErr: true,
			errMsg:  "must specify at least one allowed function",
		},
		{
			name: "Invalid function signature format",
			list: &SemanticAllowList{
				AllowedContracts: []ContractPermission{
					{
						Address:          "0xE592427A0AEce92De3Edee1F18E0157C05861564",
						AllowedFunctions: []string{"swapWithoutParentheses"},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid function signature format",
		},
		{
			name: "Daily limit exceeds weekly limit should error",
			list: &SemanticAllowList{
				AllowedContracts: []ContractPermission{
					{
						Address:          "0xE592427A0AEce92De3Edee1F18E0157C05861564",
						AllowedFunctions: []string{"swap()"},
					},
				},
				HardLimits: HardLimits{
					MaxDailyValue:  big.NewInt(100000),
					MaxWeeklyValue: big.NewInt(50000), // Less than daily!
				},
			},
			wantErr: true,
			errMsg:  "max_daily_value",
		},
		{
			name: "Multisig required but not configured should error",
			list: &SemanticAllowList{
				AllowedContracts: []ContractPermission{
					{
						Address:          "0xE592427A0AEce92De3Edee1F18E0157C05861564",
						AllowedFunctions: []string{"swap()"},
					},
				},
				HardLimits: HardLimits{
					RequireMultisig:   true,
					MultisigThreshold: nil, // Missing config!
				},
			},
			wantErr: true,
			errMsg:  "multisig_threshold is not specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.list.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, should contain %q", err, tt.errMsg)
			}
		})
	}
}

func TestMultisigConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *MultisigConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid 2-of-3 multisig",
			config: &MultisigConfig{
				RequiredApprovals: 2,
				TotalSigners:      3,
				AuthorizedSigners: []string{"signer1", "signer2", "signer3"},
			},
			wantErr: false,
		},
		{
			name: "Required approvals exceeds total signers",
			config: &MultisigConfig{
				RequiredApprovals: 4,
				TotalSigners:      3,
				AuthorizedSigners: []string{"signer1", "signer2", "signer3"},
			},
			wantErr: true,
			errMsg:  "cannot exceed total_signers",
		},
		{
			name: "Zero required approvals should error",
			config: &MultisigConfig{
				RequiredApprovals: 0,
				TotalSigners:      3,
				AuthorizedSigners: []string{"signer1", "signer2", "signer3"},
			},
			wantErr: true,
			errMsg:  "required_approvals must be positive",
		},
		{
			name: "Mismatch between total signers and authorized signers list",
			config: &MultisigConfig{
				RequiredApprovals: 2,
				TotalSigners:      3,
				AuthorizedSigners: []string{"signer1", "signer2"}, // Only 2, not 3
			},
			wantErr: true,
			errMsg:  "does not match total_signers",
		},
		{
			name: "Duplicate signers should error",
			config: &MultisigConfig{
				RequiredApprovals: 2,
				TotalSigners:      3,
				AuthorizedSigners: []string{"signer1", "signer1", "signer2"}, // Duplicate!
			},
			wantErr: true,
			errMsg:  "duplicate signer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, should contain %q", err, tt.errMsg)
			}
		})
	}
}

func TestSemanticAllowList_IsContractAllowed(t *testing.T) {
	allowList := &SemanticAllowList{
		AllowedContracts: []ContractPermission{
			{
				Address:          "0xE592427A0AEce92De3Edee1F18E0157C05861564",
				AllowedFunctions: []string{"swap()"},
			},
			{
				Address:          "0x1234567890123456789012345678901234567890",
				AllowedFunctions: []string{"transfer()"},
			},
		},
	}

	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{
			name:    "Allowed contract (lowercase)",
			address: "0xe592427a0aece92de3edee1f18e0157c05861564",
			want:    true,
		},
		{
			name:    "Allowed contract (uppercase)",
			address: "0xE592427A0AECE92DE3EDEE1F18E0157C05861564",
			want:    true,
		},
		{
			name:    "Allowed contract (mixed case)",
			address: "0xE592427A0AEce92De3Edee1F18E0157C05861564",
			want:    true,
		},
		{
			name:    "Not allowed contract",
			address: "0xABCDEF0123456789ABCDEF0123456789ABCDEF01",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allowList.IsContractAllowed(tt.address); got != tt.want {
				t.Errorf("IsContractAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSemanticAllowList_IsFunctionAllowed(t *testing.T) {
	allowList := &SemanticAllowList{
		AllowedContracts: []ContractPermission{
			{
				Address: "0xE592427A0AEce92De3Edee1F18E0157C05861564",
				AllowedFunctions: []string{
					"exactInputSingle((address,address,uint24,address,uint256,uint256,uint256,uint160))",
					"swap(uint256,uint256,uint256,uint256)",
				},
			},
		},
	}

	tests := []struct {
		name              string
		contractAddress   string
		functionSignature string
		want              bool
	}{
		{
			name:              "Allowed function on allowed contract",
			contractAddress:   "0xE592427A0AEce92De3Edee1F18E0157C05861564",
			functionSignature: "swap(uint256,uint256,uint256,uint256)",
			want:              true,
		},
		{
			name:              "Not allowed function on allowed contract",
			contractAddress:   "0xE592427A0AEce92De3Edee1F18E0157C05861564",
			functionSignature: "transfer(address,uint256)",
			want:              false,
		},
		{
			name:              "Allowed function on not allowed contract",
			contractAddress:   "0xABCDEF0123456789ABCDEF0123456789ABCDEF01",
			functionSignature: "swap(uint256,uint256,uint256,uint256)",
			want:              false,
		},
		{
			name:              "Case insensitive contract address",
			contractAddress:   "0xe592427a0aece92de3edee1f18e0157c05861564",
			functionSignature: "swap(uint256,uint256,uint256,uint256)",
			want:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allowList.IsFunctionAllowed(tt.contractAddress, tt.functionSignature); got != tt.want {
				t.Errorf("IsFunctionAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHardLimits_CheckTransactionValue(t *testing.T) {
	tests := []struct {
		name    string
		limits  *HardLimits
		value   *big.Int
		wantOk  bool
		wantErr bool
	}{
		{
			name: "Value within limit",
			limits: &HardLimits{
				MaxTransactionValue: big.NewInt(10000),
			},
			value:   big.NewInt(5000),
			wantOk:  true,
			wantErr: false,
		},
		{
			name: "Value equals limit (boundary)",
			limits: &HardLimits{
				MaxTransactionValue: big.NewInt(10000),
			},
			value:   big.NewInt(10000),
			wantOk:  true,
			wantErr: false,
		},
		{
			name: "Value exceeds limit",
			limits: &HardLimits{
				MaxTransactionValue: big.NewInt(10000),
			},
			value:   big.NewInt(10001),
			wantOk:  false,
			wantErr: true,
		},
		{
			name:    "No limit set (always allowed)",
			limits:  &HardLimits{},
			value:   big.NewInt(999999999),
			wantOk:  true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, err := tt.limits.CheckTransactionValue(tt.value)
			if gotOk != tt.wantOk {
				t.Errorf("CheckTransactionValue() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTransactionValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContractPermission_Validate(t *testing.T) {
	tests := []struct {
		name       string
		permission *ContractPermission
		wantErr    bool
		errMsg     string
	}{
		{
			name: "Valid permission",
			permission: &ContractPermission{
				Address:          "0xE592427A0AEce92De3Edee1F18E0157C05861564",
				AllowedFunctions: []string{"swap(uint256,uint256,uint256,uint256)"},
			},
			wantErr: false,
		},
		{
			name: "Empty address",
			permission: &ContractPermission{
				Address:          "",
				AllowedFunctions: []string{"swap()"},
			},
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name: "Address without 0x prefix",
			permission: &ContractPermission{
				Address:          "E592427A0AEce92De3Edee1F18E0157C05861564",
				AllowedFunctions: []string{"swap()"},
			},
			wantErr: true,
			errMsg:  "invalid contract address format",
		},
		{
			name: "Address too short",
			permission: &ContractPermission{
				Address:          "0x123",
				AllowedFunctions: []string{"swap()"},
			},
			wantErr: true,
			errMsg:  "invalid contract address format",
		},
		{
			name: "No allowed functions",
			permission: &ContractPermission{
				Address:          "0xE592427A0AEce92De3Edee1F18E0157C05861564",
				AllowedFunctions: []string{},
			},
			wantErr: true,
			errMsg:  "must specify at least one allowed function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.permission.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, should contain %q", err, tt.errMsg)
			}
		})
	}
}

// Benchmark for contract allow-list checking
func BenchmarkIsContractAllowed(b *testing.B) {
	allowList := &SemanticAllowList{
		AllowedContracts: []ContractPermission{
			{Address: "0xE592427A0AEce92De3Edee1F18E0157C05861564", AllowedFunctions: []string{"swap()"}},
			{Address: "0x1234567890123456789012345678901234567890", AllowedFunctions: []string{"transfer()"}},
			{Address: "0xABCDEF0123456789ABCDEF0123456789ABCDEF01", AllowedFunctions: []string{"approve()"}},
		},
	}

	address := "0xE592427A0AEce92De3Edee1F18E0157C05861564"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = allowList.IsContractAllowed(address)
	}
}

// Benchmark for function allow-list checking
func BenchmarkIsFunctionAllowed(b *testing.B) {
	allowList := &SemanticAllowList{
		AllowedContracts: []ContractPermission{
			{
				Address: "0xE592427A0AEce92De3Edee1F18E0157C05861564",
				AllowedFunctions: []string{
					"exactInputSingle((address,address,uint24,address,uint256,uint256,uint256,uint160))",
					"swap(uint256,uint256,uint256,uint256)",
					"transfer(address,uint256)",
				},
			},
		},
	}

	address := "0xE592427A0AEce92De3Edee1F18E0157C05861564"
	funcSig := "swap(uint256,uint256,uint256,uint256)"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = allowList.IsFunctionAllowed(address, funcSig)
	}
}
