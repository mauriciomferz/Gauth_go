// Package testutil contains shared JSON fixtures for capabilities and policies used by web tests.
package testutil

// Canonical capability registry JSON fixtures used across capability anchor & persistence tests.
// Centralizing reduces duplication and makes future schema evolutions simpler.

const (
	// CapTransferV1 is a single capability (transfer) registry fixture.
	CapTransferV1 = `{"schema_version":1,"capabilities":[{"id":"cap.transfer","version":"1.0","stable":true}],"action_mappings":{"transaction:execute":["cap.transfer"]}}`

	// CapTransferIssueV1 extends transfer with issue capability.
	CapTransferIssueV1 = `{"schema_version":1,"capabilities":[{"id":"cap.transfer","version":"1.0","stable":true},{"id":"cap.issue","version":"1.0","stable":true}],"action_mappings":{"transaction:execute":["cap.transfer"],"transaction:issue":["cap.issue"]}}`

	// CapTransferIssueDelegationCreateV1 adds delegation create capability.
	CapTransferIssueDelegationCreateV1 = `{"schema_version":1,"capabilities":[{"id":"cap.transfer","version":"1.0","stable":true},{"id":"cap.issue","version":"1.0","stable":true},{"id":"cap.delegation.create","version":"1.0","stable":true}],"action_mappings":{"transaction:execute":["cap.transfer"],"transaction:issue":["cap.issue"],"delegation:create":["cap.delegation.create"]}}`

	// CapAlphaV1 minimal alpha capability registry.
	CapAlphaV1 = `{"schema_version":1,"capabilities":[{"id":"cap.alpha","version":"1.0","stable":true}],"action_mappings":{"transaction:execute":["cap.alpha"]}}`

	// CapAlphaUnknownMapping invalid mapping referencing unknown capability.
	CapAlphaUnknownMapping = `{"schema_version":1,"capabilities":[{"id":"cap.alpha","version":"1.0","stable":true}],"action_mappings":{"delegation:create":["cap.unknown"]}}`

	// CapAlphaDuplicateIDs invalid duplicate id entries.
	CapAlphaDuplicateIDs = `{"schema_version":1,"capabilities":[{"id":"cap.alpha","version":"1.0","stable":true},{"id":"cap.alpha","version":"1.1","stable":true}],"action_mappings":{"transaction:execute":["cap.alpha"]}}`

	// CapAlphaMissingSchemaVersion missing schema_version field.
	CapAlphaMissingSchemaVersion = `{"capabilities":[{"id":"cap.alpha","version":"1.0","stable":true}],"action_mappings":{"transaction:execute":["cap.alpha"]}}`

	// PolicyBundleB1V1 single policy bundle used in persistence tests.
	PolicyBundleB1V1 = `{"id":"b1","policies":[{"id":"p1","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]}]}`

	// PolicyBundleB2V1 second policy bundle appended after B1.
	PolicyBundleB2V1 = `{"id":"b2","policies":[{"id":"p2","subjects":["a"],"rules":[{"actions":["y"],"resources":["r"],"effect":"allow"}]}]}`

	// PolicyBundleMultiPerm1V1 multi-policy bundle (order variant 1) used for canonical permutation tests.
	PolicyBundleMultiPerm1V1 = `{"id":"multi","policies":[{"id":"p1","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]},{"id":"p2","subjects":["a"],"rules":[{"actions":["y"],"resources":["r"],"effect":"allow"}]}]}`

	// PolicyBundleMultiPerm2V1 multi-policy bundle (order variant 2) same semantics different ordering.
	PolicyBundleMultiPerm2V1 = `{"id":"multi","policies":[{"id":"p2","subjects":["a"],"rules":[{"actions":["y"],"resources":["r"],"effect":"allow"}]},{"id":"p1","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]}]}`

	// PolicyBundleMultiPlusP3V1 semantic change adds third policy p3.
	PolicyBundleMultiPlusP3V1 = `{"id":"multi","policies":[{"id":"p1","subjects":["a"],"rules":[{"actions":["x"],"resources":["r"],"effect":"allow"}]},{"id":"p2","subjects":["a"],"rules":[{"actions":["y"],"resources":["r"],"effect":"allow"}]},{"id":"p3","subjects":["a"],"rules":[{"actions":["z"],"resources":["r"],"effect":"deny"}]}]}`

	// CapTransferAuditV1 adds audit capability to transfer for hash change tests.
	CapTransferAuditV1 = `{"schema_version":1,"capabilities":[{"id":"cap.transfer","version":"1.0","stable":true},{"id":"cap.audit","version":"1.0","stable":true}],"action_mappings":{"transaction:execute":["cap.transfer"],"audit:read":["cap.audit"]}}`

	// CapAlphaBetaIssueV1 two capabilities alpha & beta with issue action mapping to beta.
	CapAlphaBetaIssueV1 = `{"schema_version":1,"capabilities":[{"id":"cap.alpha","version":"1.0","stable":true},{"id":"cap.beta","version":"1.0","stable":true}],"action_mappings":{"transaction:issue":["cap.beta"]}}`

	// CapAlphaBetaGammaDelegationIssueV1 alpha, beta (stable), gamma (unstable) with issue & delegation mappings.
	CapAlphaBetaGammaDelegationIssueV1 = `{"schema_version":1,"capabilities":[{"id":"cap.alpha","version":"1.0","stable":true},{"id":"cap.beta","version":"1.0","stable":true},{"id":"cap.gamma","version":"1.0","stable":false}],"action_mappings":{"transaction:issue":["cap.beta"],"delegation:create":["cap.gamma"]}}`

	// CapABDelegationIssuePerm1V1 permutation 1 used in hash determinism tests.
	CapABDelegationIssuePerm1V1 = `{"schema_version":1,"capabilities":[{"id":"cap.a","version":"1.0"},{"id":"cap.b","version":"1.0"}],"action_mappings":{"delegation:create":["cap.a","cap.b"],"token:issue":["cap.b"]}}`

	// CapABDelegationIssuePerm2V1 permutation 2 used in hash determinism tests.
	CapABDelegationIssuePerm2V1 = `{"schema_version":1,"capabilities":[{"id":"cap.b","version":"1.0"},{"id":"cap.a","version":"1.0"}],"action_mappings":{"token:issue":["cap.b"],"delegation:create":["cap.b","cap.a"]}}`

	// CapABCDelegationIssueV1 semantic change with added cap.c referenced in token:issue mapping.
	CapABCDelegationIssueV1 = `{"schema_version":1,"capabilities":[{"id":"cap.a","version":"1.0"},{"id":"cap.b","version":"1.0"},{"id":"cap.c","version":"1.0"}],"action_mappings":{"delegation:create":["cap.a","cap.b"],"token:issue":["cap.b","cap.c"]}}`
)
