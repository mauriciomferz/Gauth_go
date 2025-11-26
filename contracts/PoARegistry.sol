// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title PoARegistry
 * @dev Smart contract for registering and managing AI Power of Attorney records
 * This contract serves as a "Commercial Register for AI Systems"
 * Provides immutable, publicly verifiable records of AI authorization
 */
contract PoARegistry {
    
    // Events
    event PoARegistered(
        string indexed poaId,
        string issuerIdHash,
        string granteeIdHash,
        uint256 validFrom,
        uint256 validUntil,
        string scopeHash,
        string metadataURI
    );
    
    event PoARevoked(
        string indexed poaId,
        string revokedBy,
        string reason,
        uint256 revokedAt
    );
    
    event PoAStatusUpdated(
        string indexed poaId,
        string newStatus,
        uint256 updatedAt
    );
    
    event AIAgentRegistered(
        string indexed agentId,
        string ownerIdHash,
        string agentType,
        uint256 registeredAt
    );
    
    // Structs
    struct PoARecord {
        string poaId;
        string issuerIdHash;      // Hashed for privacy
        string granteeIdHash;     // Hashed for privacy
        string scopeHash;
        string attestationHash;
        string metadataHash;
        string metadataURI;       // IPFS CID or HTTPS URL
        uint256 validFrom;
        uint256 validUntil;
        string status;            // "active", "revoked", "expired"
        bool exists;
        uint256 registeredAt;
        uint256 lastUpdated;
    }
    
    struct RevocationRecord {
        bool revoked;
        string revokedBy;
        string reason;
        uint256 revokedAt;
        uint256 blockNumber;
    }
    
    struct AIAgentRecord {
        string agentId;
        string ownerIdHash;
        string agentName;
        string agentType;
        string[] poaIds;
        uint256 registeredAt;
        bool exists;
    }
    
    // State variables
    mapping(string => PoARecord) private poaRecords;
    mapping(string => RevocationRecord) private revocations;
    mapping(string => AIAgentRecord) private aiAgents;
    mapping(string => string[]) private issuerPoAs;  // issuerIdHash => poaIds[]
    mapping(string => string[]) private granteePoAs; // granteeIdHash => poaIds[]
    
    string[] private allPoAIds;
    string[] private allAgentIds;
    
    address public owner;
    uint256 public totalPoAs;
    uint256 public totalRevocations;
    uint256 public totalAIAgents;
    
    // Modifiers
    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner can call this function");
        _;
    }
    
    modifier poaExists(string memory poaId) {
        require(poaRecords[poaId].exists, "PoA does not exist");
        _;
    }
    
    modifier poaNotRevoked(string memory poaId) {
        require(!revocations[poaId].revoked, "PoA is revoked");
        _;
    }
    
    // Constructor
    constructor() {
        owner = msg.sender;
        totalPoAs = 0;
        totalRevocations = 0;
        totalAIAgents = 0;
    }
    
    /**
     * @dev Register a new Power of Attorney on the blockchain
     * @param poaId Unique identifier for the PoA
     * @param issuerIdHash Hashed issuer ID (for privacy)
     * @param granteeIdHash Hashed grantee ID (for privacy)
     * @param scopeHash Hash of the scope
     * @param attestationHash Hash of attestations
     * @param metadataHash Hash of complete metadata
     * @param metadataURI IPFS CID or URL to full metadata
     * @param validFrom Unix timestamp when PoA becomes valid
     * @param validUntil Unix timestamp when PoA expires
     */
    function registerPoA(
        string memory poaId,
        string memory issuerIdHash,
        string memory granteeIdHash,
        string memory scopeHash,
        string memory attestationHash,
        string memory metadataHash,
        string memory metadataURI,
        uint256 validFrom,
        uint256 validUntil
    ) public returns (bool) {
        require(!poaRecords[poaId].exists, "PoA already registered");
        require(validUntil > validFrom, "Invalid validity period");
        require(bytes(poaId).length > 0, "PoA ID cannot be empty");
        
        poaRecords[poaId] = PoARecord({
            poaId: poaId,
            issuerIdHash: issuerIdHash,
            granteeIdHash: granteeIdHash,
            scopeHash: scopeHash,
            attestationHash: attestationHash,
            metadataHash: metadataHash,
            metadataURI: metadataURI,
            validFrom: validFrom,
            validUntil: validUntil,
            status: "active",
            exists: true,
            registeredAt: block.timestamp,
            lastUpdated: block.timestamp
        });
        
        // Add to indices
        allPoAIds.push(poaId);
        issuerPoAs[issuerIdHash].push(poaId);
        granteePoAs[granteeIdHash].push(poaId);
        
        totalPoAs++;
        
        emit PoARegistered(
            poaId,
            issuerIdHash,
            granteeIdHash,
            validFrom,
            validUntil,
            scopeHash,
            metadataURI
        );
        
        return true;
    }
    
    /**
     * @dev Revoke a Power of Attorney
     * @param poaId ID of the PoA to revoke
     * @param revokedBy Entity performing the revocation
     * @param reason Reason for revocation
     */
    function revokePoA(
        string memory poaId,
        string memory revokedBy,
        string memory reason
    ) public poaExists(poaId) poaNotRevoked(poaId) returns (bool) {
        
        poaRecords[poaId].status = "revoked";
        poaRecords[poaId].lastUpdated = block.timestamp;
        
        revocations[poaId] = RevocationRecord({
            revoked: true,
            revokedBy: revokedBy,
            reason: reason,
            revokedAt: block.timestamp,
            blockNumber: block.number
        });
        
        totalRevocations++;
        
        emit PoARevoked(poaId, revokedBy, reason, block.timestamp);
        
        return true;
    }
    
    /**
     * @dev Update PoA status
     * @param poaId ID of the PoA
     * @param newStatus New status value
     */
    function updatePoAStatus(
        string memory poaId,
        string memory newStatus
    ) public poaExists(poaId) returns (bool) {
        poaRecords[poaId].status = newStatus;
        poaRecords[poaId].lastUpdated = block.timestamp;
        
        emit PoAStatusUpdated(poaId, newStatus, block.timestamp);
        
        return true;
    }
    
    /**
     * @dev Register an AI agent in the commercial register
     * @param agentId Unique identifier for the AI agent
     * @param ownerIdHash Hashed owner ID
     * @param agentName Name of the AI agent
     * @param agentType Type of AI agent
     */
    function registerAIAgent(
        string memory agentId,
        string memory ownerIdHash,
        string memory agentName,
        string memory agentType
    ) public returns (bool) {
        require(!aiAgents[agentId].exists, "AI Agent already registered");
        require(bytes(agentId).length > 0, "Agent ID cannot be empty");
        
        string[] memory emptyPoAs;
        aiAgents[agentId] = AIAgentRecord({
            agentId: agentId,
            ownerIdHash: ownerIdHash,
            agentName: agentName,
            agentType: agentType,
            poaIds: emptyPoAs,
            registeredAt: block.timestamp,
            exists: true
        });
        
        allAgentIds.push(agentId);
        totalAIAgents++;
        
        emit AIAgentRegistered(agentId, ownerIdHash, agentType, block.timestamp);
        
        return true;
    }
    
    /**
     * @dev Link a PoA to an AI agent
     * @param agentId AI agent ID
     * @param poaId PoA ID
     */
    function linkPoAToAgent(
        string memory agentId,
        string memory poaId
    ) public returns (bool) {
        require(aiAgents[agentId].exists, "AI Agent does not exist");
        require(poaRecords[poaId].exists, "PoA does not exist");
        
        aiAgents[agentId].poaIds.push(poaId);
        
        return true;
    }
    
    /**
     * @dev Verify if a PoA exists and is active
     * @param poaId ID of the PoA to verify
     * @return exists Whether the PoA exists
     * @return active Whether the PoA is currently active
     * @return revoked Whether the PoA has been revoked
     */
    function verifyPoA(string memory poaId) public view returns (
        bool exists,
        bool active,
        bool revoked
    ) {
        if (!poaRecords[poaId].exists) {
            return (false, false, false);
        }
        
        PoARecord memory poa = poaRecords[poaId];
        bool isActive = (
            block.timestamp >= poa.validFrom &&
            block.timestamp <= poa.validUntil &&
            !revocations[poaId].revoked &&
            keccak256(bytes(poa.status)) == keccak256(bytes("active"))
        );
        
        return (true, isActive, revocations[poaId].revoked);
    }
    
    /**
     * @dev Get PoA record details
     * @param poaId ID of the PoA
     */
    function getPoA(string memory poaId) public view poaExists(poaId) returns (
        string memory issuerIdHash,
        string memory granteeIdHash,
        string memory scopeHash,
        string memory metadataURI,
        uint256 validFrom,
        uint256 validUntil,
        string memory status,
        uint256 registeredAt
    ) {
        PoARecord memory poa = poaRecords[poaId];
        return (
            poa.issuerIdHash,
            poa.granteeIdHash,
            poa.scopeHash,
            poa.metadataURI,
            poa.validFrom,
            poa.validUntil,
            poa.status,
            poa.registeredAt
        );
    }
    
    /**
     * @dev Get revocation details
     * @param poaId ID of the PoA
     */
    function getRevocation(string memory poaId) public view returns (
        bool revoked,
        string memory revokedBy,
        string memory reason,
        uint256 revokedAt
    ) {
        RevocationRecord memory rev = revocations[poaId];
        return (rev.revoked, rev.revokedBy, rev.reason, rev.revokedAt);
    }
    
    /**
     * @dev Get all PoAs issued by an issuer
     * @param issuerIdHash Hashed issuer ID
     */
    function getPoAsByIssuer(string memory issuerIdHash) public view returns (string[] memory) {
        return issuerPoAs[issuerIdHash];
    }
    
    /**
     * @dev Get all PoAs granted to a grantee
     * @param granteeIdHash Hashed grantee ID
     */
    function getPoAsByGrantee(string memory granteeIdHash) public view returns (string[] memory) {
        return granteePoAs[granteeIdHash];
    }
    
    /**
     * @dev Get AI agent details
     * @param agentId AI agent ID
     */
    function getAIAgent(string memory agentId) public view returns (
        string memory ownerIdHash,
        string memory agentName,
        string memory agentType,
        uint256 registeredAt,
        uint256 poaCount
    ) {
        require(aiAgents[agentId].exists, "AI Agent does not exist");
        AIAgentRecord memory agent = aiAgents[agentId];
        return (
            agent.ownerIdHash,
            agent.agentName,
            agent.agentType,
            agent.registeredAt,
            agent.poaIds.length
        );
    }
    
    /**
     * @dev Get all PoAs for an AI agent
     * @param agentId AI agent ID
     */
    function getAIAgentPoAs(string memory agentId) public view returns (string[] memory) {
        require(aiAgents[agentId].exists, "AI Agent does not exist");
        return aiAgents[agentId].poaIds;
    }
    
    /**
     * @dev Get total number of registered PoAs
     */
    function getTotalPoAs() public view returns (uint256) {
        return totalPoAs;
    }
    
    /**
     * @dev Get total number of revocations
     */
    function getTotalRevocations() public view returns (uint256) {
        return totalRevocations;
    }
    
    /**
     * @dev Get total number of registered AI agents
     */
    function getTotalAIAgents() public view returns (uint256) {
        return totalAIAgents;
    }
    
    /**
     * @dev Get all PoA IDs (paginated)
     * @param offset Starting index
     * @param limit Number of results
     */
    function getAllPoAIds(uint256 offset, uint256 limit) public view returns (string[] memory) {
        require(offset < allPoAIds.length, "Offset out of bounds");
        
        uint256 end = offset + limit;
        if (end > allPoAIds.length) {
            end = allPoAIds.length;
        }
        
        string[] memory result = new string[](end - offset);
        for (uint256 i = offset; i < end; i++) {
            result[i - offset] = allPoAIds[i];
        }
        
        return result;
    }
    
    /**
     * @dev Get all AI agent IDs (paginated)
     * @param offset Starting index
     * @param limit Number of results
     */
    function getAllAIAgentIds(uint256 offset, uint256 limit) public view returns (string[] memory) {
        require(offset < allAgentIds.length, "Offset out of bounds");
        
        uint256 end = offset + limit;
        if (end > allAgentIds.length) {
            end = allAgentIds.length;
        }
        
        string[] memory result = new string[](end - offset);
        for (uint256 i = offset; i < end; i++) {
            result[i - offset] = allAgentIds[i];
        }
        
        return result;
    }
}
