
import React, { useState, useEffect } from 'react';
import {
  makeStyles,
  shorthands,
  tokens,
  Card,
  Button,
  Text,
  Input,

  Tab,
  TabList,
  Spinner,
  Title3,
  Badge,
  Textarea,
} from '@fluentui/react-components';
import {
  CheckmarkCircle24Regular,
  DismissCircle24Regular,
  DocumentText24Regular,
  Search24Regular,
  Shield24Regular,
  Organization24Regular,
  ArrowDownload24Regular,
  Info24Regular,
  LockClosed24Regular,
  Certificate24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
  root: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('16px'),
    ...shorthands.padding('24px'),
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  tabsContainer: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('16px'),
  },
  overviewCards: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
    ...shorthands.gap('16px'),
    marginBottom: '24px',
  },
  card: {
    ...shorthands.padding('16px'),
  },
  cardHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  cardTitle: {
    fontSize: tokens.fontSizeBase300,
    fontWeight: tokens.fontWeightSemibold,
  },
  cardValue: {
    fontSize: tokens.fontSizeBase600,
    fontWeight: tokens.fontWeightBold,
    color: tokens.colorBrandForeground1,
    marginTop: '8px',
  },
  treeContainer: {
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke1),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    ...shorthands.padding('24px'),
    backgroundColor: tokens.colorNeutralBackground1,
    minHeight: '500px',
    position: 'relative',
    overflowX: 'auto',
  },
  treeNode: {
    display: 'inline-flex',
    flexDirection: 'column',
    alignItems: 'center',
    ...shorthands.padding('8px'),
    minWidth: '120px',
  },
  nodeBox: {
    ...shorthands.padding('12px', '16px'),
    ...shorthands.border('2px', 'solid', tokens.colorBrandStroke1),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    backgroundColor: tokens.colorBrandBackground2,
    cursor: 'pointer',
    transition: 'all 0.2s',
    '&:hover': {
      backgroundColor: tokens.colorBrandBackground,
      transform: 'scale(1.05)',
    },
  },
  nodeHash: {
    fontFamily: tokens.fontFamilyMonospace,
    fontSize: tokens.fontSizeBase200,
    fontWeight: tokens.fontWeightSemibold,
    color: tokens.colorBrandForeground1,
    marginBottom: '4px',
  },
  nodeLevel: {
    fontSize: tokens.fontSizeBase100,
    color: tokens.colorNeutralForeground3,
  },
  treeLevel: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'flex-start',
    ...shorthands.gap('24px'),
    marginBottom: '48px',
    position: 'relative',
  },
  treeLine: {
    position: 'absolute',
    height: '40px',
    width: '2px',
    backgroundColor: tokens.colorNeutralStroke1,
    top: '-40px',
    left: '50%',
    transform: 'translateX(-50%)',
  },
  proofCard: {
    ...shorthands.padding('16px'),
    marginBottom: '12px',
  },
  proofHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  proofInfo: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('8px'),
  },
  proofHash: {
    fontFamily: tokens.fontFamilyMonospace,
    fontSize: tokens.fontSizeBase200,
    wordBreak: 'break-all',
  },
  proofSteps: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('8px'),
    marginTop: '12px',
    ...shorthands.padding('12px'),
    backgroundColor: tokens.colorNeutralBackground2,
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
  },
  proofStep: {
    display: 'flex',
    alignItems: 'center',
    ...shorthands.gap('12px'),
    ...shorthands.padding('8px'),
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke2),
    ...shorthands.borderRadius(tokens.borderRadiusSmall),
    backgroundColor: tokens.colorNeutralBackground1,
  },
  stepNumber: {
    width: '32px',
    height: '32px',
    ...shorthands.borderRadius('50%'),
    backgroundColor: tokens.colorBrandBackground,
    color: tokens.colorBrandForeground1,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontWeight: tokens.fontWeightSemibold,
    flexShrink: 0,
  },
  stepContent: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('4px'),
    flex: 1,
  },
  verificationPanel: {
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke1),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    ...shorthands.padding('24px'),
    backgroundColor: tokens.colorNeutralBackground1,
  },
  verificationInput: {
    display: 'flex',
    ...shorthands.gap('12px'),
    marginBottom: '24px',
  },
  verificationResult: {
    ...shorthands.padding('16px'),
    ...shorthands.border('2px', 'solid'),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    marginTop: '16px',
  },
  verificationSuccess: {
    borderTopColor: tokens.colorPaletteGreenBorder1,
    borderRightColor: tokens.colorPaletteGreenBorder1,
    borderBottomColor: tokens.colorPaletteGreenBorder1,
    borderLeftColor: tokens.colorPaletteGreenBorder1,
    backgroundColor: tokens.colorPaletteGreenBackground1,
  },
  verificationFailure: {
    borderTopColor: tokens.colorPaletteRedBorder1,
    borderRightColor: tokens.colorPaletteRedBorder1,
    borderBottomColor: tokens.colorPaletteRedBorder1,
    borderLeftColor: tokens.colorPaletteRedBorder1,
    backgroundColor: tokens.colorPaletteRedBackground1,
  },
  verificationDetails: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    ...shorthands.gap('12px'),
    marginTop: '12px',
  },
  detailItem: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('4px'),
  },
  detailLabel: {
    fontSize: tokens.fontSizeBase200,
    color: tokens.colorNeutralForeground3,
  },
  detailValue: {
    fontSize: tokens.fontSizeBase300,
    fontWeight: tokens.fontWeightSemibold,
    fontFamily: tokens.fontFamilyMonospace,
  },
  revocationCard: {
    ...shorthands.padding('16px'),
    marginBottom: '12px',
  },
  revocationHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  revocationInfo: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('4px'),
  },
  revocationTokenId: {
    fontWeight: tokens.fontWeightSemibold,
    fontFamily: tokens.fontFamilyMonospace,
  },
  revocationMetrics: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
    ...shorthands.gap('12px'),
    marginTop: '12px',
  },
  metric: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('4px'),
  },
  metricLabel: {
    fontSize: tokens.fontSizeBase200,
    color: tokens.colorNeutralForeground3,
  },
  metricValue: {
    fontSize: tokens.fontSizeBase400,
    fontWeight: tokens.fontWeightSemibold,
  },
  logContainer: {
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke1),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    ...shorthands.padding('16px'),
    backgroundColor: tokens.colorNeutralBackground2,
    maxHeight: '400px',
    overflowY: 'auto',
  },
  logEntry: {
    ...shorthands.padding('8px'),
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke2),
    ...shorthands.borderRadius(tokens.borderRadiusSmall),
    marginBottom: '8px',
    backgroundColor: tokens.colorNeutralBackground1,
    fontFamily: tokens.fontFamilyMonospace,
    fontSize: tokens.fontSizeBase200,
  },
  logTimestamp: {
    color: tokens.colorNeutralForeground3,
    marginRight: '12px',
  },
  logHash: {
    color: tokens.colorBrandForeground1,
    fontWeight: tokens.fontWeightSemibold,
  },
  formField: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('8px'),
    marginBottom: '16px',
  },
});

interface MerkleNode {
  hash: string;
  level: number;
  position: number;
  leftChild?: string;
  rightChild?: string;
  isLeaf: boolean;
  data?: string;
}

interface MerkleProof {
  tokenId: string;
  leafHash: string;
  rootHash: string;
  path: ProofStep[];
  verified: boolean;
  timestamp: string;
}

interface ProofStep {
  hash: string;
  position: 'left' | 'right';
  sibling: string;
}

interface RevocationEntry {
  id: string;
  tokenId: string;
  reason: string;
  timestamp: string;
  revokedBy: string;
  leafHash: string;
  merkleRoot: string;
  blockHeight: number;
  verified: boolean;
}

interface AppendOnlyLogEntry {
  index: number;
  timestamp: string;
  operation: 'append' | 'verify';
  data: string;
  hash: string;
  previousHash: string;
}

const RevocationTransparency: React.FC = () => {
  const styles = useStyles();
  const [selectedTab, setSelectedTab] = useState<string>('tree');
  const [loading, setLoading] = useState(false);

  // Merkle tree tab state
  const [merkleTree, setMerkleTree] = useState<MerkleNode[]>([]);
  const [selectedNode, setSelectedNode] = useState<MerkleNode | null>(null);
  const [treeDepth, setTreeDepth] = useState(3);

  // Proof generation tab state
  const [proofs, setProofs] = useState<MerkleProof[]>([]);
  const [proofTokenId, setProofTokenId] = useState('');

  // Verification tab state
  const [verificationInput, setVerificationInput] = useState('');
  const [verificationResult, setVerificationResult] = useState<any>(null);

  // Revocation list tab state
  const [revocations, setRevocations] = useState<RevocationEntry[]>([]);

  // Append-only log tab state
  const [logEntries, setLogEntries] = useState<AppendOnlyLogEntry[]>([]);

  useEffect(() => {
    fetchMerkleTree();
    fetchProofs();
    fetchRevocations();
    fetchLogEntries();
  }, []);

  const fetchMerkleTree = async () => {
    try {
      const response = await fetch('/api/admin/revocation/merkle-tree');
      const data = await response.json();
      setMerkleTree(data.nodes || []);
      setTreeDepth(data.depth || 3);
    } catch (error) {
      console.error('Failed to fetch Merkle tree:', error);
    }
  };

  const fetchProofs = async () => {
    try {
      const response = await fetch('/api/admin/revocation/proofs');
      const data = await response.json();
      setProofs(data.proofs || []);
    } catch (error) {
      console.error('Failed to fetch proofs:', error);
    }
  };

  const fetchRevocations = async () => {
    try {
      const response = await fetch('/api/admin/revocation/list');
      const data = await response.json();
      setRevocations(data.revocations || []);
    } catch (error) {
      console.error('Failed to fetch revocations:', error);
    }
  };

  const fetchLogEntries = async () => {
    try {
      const response = await fetch('/api/admin/revocation/log');
      const data = await response.json();
      setLogEntries(data.entries || []);
    } catch (error) {
      console.error('Failed to fetch log entries:', error);
    }
  };

  const handleGenerateProof = async () => {
    if (!proofTokenId.trim()) return;

    try {
      setLoading(true);
      const response = await fetch('/api/admin/revocation/generate-proof', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tokenId: proofTokenId }),
      });
      if (response.ok) {
        setProofTokenId('');
        fetchProofs();
      }
    } catch (error) {
      console.error('Failed to generate proof:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyProof = async () => {
    if (!verificationInput.trim()) return;

    try {
      setLoading(true);
      const response = await fetch('/api/admin/revocation/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ proofData: verificationInput }),
      });
      const data = await response.json();
      setVerificationResult(data);
    } catch (error) {
      console.error('Failed to verify proof:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDownloadProof = (proof: MerkleProof) => {
    const dataStr = JSON.stringify(proof, null, 2);
    const dataUri = 'data:application/json;charset=utf-8,' + encodeURIComponent(dataStr);
    const exportFileDefaultName = `merkle - proof - ${proof.tokenId}.json`;

    const linkElement = document.createElement('a');
    linkElement.setAttribute('href', dataUri);
    linkElement.setAttribute('download', exportFileDefaultName);
    linkElement.click();
  };

  const renderMerkleTree = () => {
    const nodesByLevel: MerkleNode[][] = [];
    for (let i = 0; i <= treeDepth; i++) {
      nodesByLevel[i] = merkleTree.filter(node => node.level === i);
    }

    return (
      <div className={styles.treeContainer}>
        {nodesByLevel.map((levelNodes, levelIndex) => (
          <div key={levelIndex} className={styles.treeLevel}>
            {levelNodes.map((node, _nodeIndex) => (
              <div key={`${node.level} -${node.position} `} className={styles.treeNode}>
                <div
                  className={styles.nodeBox}
                  onClick={() => setSelectedNode(node)}
                  style={{
                    backgroundColor: node.isLeaf
                      ? tokens.colorPaletteGreenBackground2
                      : node.level === 0
                        ? tokens.colorBrandStroke2
                        : tokens.colorBrandBackground2,
                  }}
                >
                  <Text className={styles.nodeHash}>
                    {node.hash.substring(0, 8)}...
                  </Text>
                  <Text className={styles.nodeLevel}>
                    {node.isLeaf ? 'Leaf' : node.level === 0 ? 'Root' : `Level ${node.level} `}
                  </Text>
                </div>
              </div>
            ))}
          </div>
        ))}

        {selectedNode && (
          <Card style={{ position: 'absolute', top: '16px', right: '16px', width: '300px' }}>
            <div style={{ padding: '16px' }}>
              <Text weight="semibold" size={400} style={{ marginBottom: '12px', display: 'block' }}>
                Node Details
              </Text>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <div>
                  <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Hash</Text>
                  <Text size={200} style={{ fontFamily: tokens.fontFamilyMonospace, wordBreak: 'break-all' }}>
                    {selectedNode.hash}
                  </Text>
                </div>
                <div>
                  <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Level</Text>
                  <Text size={300}>{selectedNode.level}</Text>
                </div>
                <div>
                  <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Position</Text>
                  <Text size={300}>{selectedNode.position}</Text>
                </div>
                {selectedNode.isLeaf && selectedNode.data && (
                  <div>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Data</Text>
                    <Text size={200} style={{ fontFamily: tokens.fontFamilyMonospace }}>
                      {selectedNode.data}
                    </Text>
                  </div>
                )}
                {selectedNode.leftChild && (
                  <div>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Left Child</Text>
                    <Text size={200} style={{ fontFamily: tokens.fontFamilyMonospace, wordBreak: 'break-all' }}>
                      {selectedNode.leftChild}
                    </Text>
                  </div>
                )}
                {selectedNode.rightChild && (
                  <div>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Right Child</Text>
                    <Text size={200} style={{ fontFamily: tokens.fontFamilyMonospace, wordBreak: 'break-all' }}>
                      {selectedNode.rightChild}
                    </Text>
                  </div>
                )}
              </div>
              <Button
                appearance="secondary"
                size="small"
                onClick={() => setSelectedNode(null)}
                style={{ marginTop: '12px', width: '100%' }}
              >
                Close
              </Button>
            </div>
          </Card>
        )}
      </div>
    );
  };

  // Calculate overview metrics
  const totalRevocations = revocations.length;
  const verifiedRevocations = revocations.filter(r => r.verified).length;
  const merkleRootHash = merkleTree.find(n => n.level === 0)?.hash || 'N/A';
  const logSize = logEntries.length;

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <Title3>Revocation Transparency</Title3>
      </div>

      {/* Overview Cards */}
      <div className={styles.overviewCards}>
        <Card className={styles.card}>
          <div className={styles.cardHeader}>
            <Organization24Regular />
            <Text className={styles.cardTitle}>Merkle Tree Depth</Text>
          </div>
          <Text className={styles.cardValue}>{treeDepth}</Text>
          <Text size={200}>{merkleTree.length} nodes</Text>
        </Card>
        <Card className={styles.card}>
          <div className={styles.cardHeader}>
            <Shield24Regular />
            <Text className={styles.cardTitle}>Root Hash</Text>
          </div>
          <Text style={{ fontSize: tokens.fontSizeBase300, fontFamily: tokens.fontFamilyMonospace, marginTop: '8px' }}>
            {merkleRootHash.substring(0, 16)}...
          </Text>
          <Text size={200}>Current root</Text>
        </Card>
        <Card className={styles.card}>
          <div className={styles.cardHeader}>
            <DismissCircle24Regular />
            <Text className={styles.cardTitle}>Total Revocations</Text>
          </div>
          <Text className={styles.cardValue}>{totalRevocations}</Text>
          <Text size={200}>{verifiedRevocations} verified</Text>
        </Card>
        <Card className={styles.card}>
          <div className={styles.cardHeader}>
            <DocumentText24Regular />
            <Text className={styles.cardTitle}>Append-Only Log</Text>
          </div>
          <Text className={styles.cardValue}>{logSize}</Text>
          <Text size={200}>Total entries</Text>
        </Card>
      </div>

      {/* Tabs */}
      <div className={styles.tabsContainer}>
        <TabList selectedValue={selectedTab} onTabSelect={(_, data) => setSelectedTab(data.value as string)}>
          <Tab value="tree">Merkle Tree Visualization</Tab>
          <Tab value="proofs">Proof Generation</Tab>
          <Tab value="verification">Proof Verification</Tab>
          <Tab value="revocations">Revocation List</Tab>
          <Tab value="log">Append-Only Log</Tab>
        </TabList>

        {/* Merkle Tree Visualization Tab */}
        {selectedTab === 'tree' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '16px' }}>
              <Text weight="semibold">Merkle Tree Structure</Text>
              <Button
                appearance="secondary"
                icon={<Organization24Regular />}
                onClick={fetchMerkleTree}
              >
                Refresh Tree
              </Button>
            </div>
            <Card style={{ padding: '0' }}>
              {renderMerkleTree()}
            </Card>
          </div>
        )}

        {/* Proof Generation Tab */}
        {selectedTab === 'proofs' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '16px' }}>
              <Text weight="semibold">Merkle Proofs</Text>
            </div>

            <Card style={{ padding: '16px', marginBottom: '24px' }}>
              <Text weight="semibold" size={300} style={{ marginBottom: '12px', display: 'block' }}>
                Generate New Proof
              </Text>
              <div style={{ display: 'flex', gap: '12px' }}>
                <Input
                  placeholder="Enter Token ID"
                  value={proofTokenId}
                  onChange={(e) => setProofTokenId(e.target.value)}
                  style={{ flex: 1 }}
                />
                <Button
                  appearance="primary"
                  icon={<Certificate24Regular />}
                  onClick={handleGenerateProof}
                  disabled={loading || !proofTokenId.trim()}
                >
                  {loading ? <Spinner size="tiny" /> : 'Generate Proof'}
                </Button>
              </div>
            </Card>

            {proofs.map((proof) => (
              <Card key={proof.tokenId} className={styles.proofCard}>
                <div className={styles.proofHeader}>
                  <div className={styles.proofInfo}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <Text weight="semibold" size={400}>Token: {proof.tokenId}</Text>
                      {proof.verified ? (
                        <Badge color="success" icon={<CheckmarkCircle24Regular />}>VERIFIED</Badge>
                      ) : (
                        <Badge color="danger" icon={<DismissCircle24Regular />}>UNVERIFIED</Badge>
                      )}
                    </div>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                      Generated: {proof.timestamp}
                    </Text>
                  </div>
                  <Button
                    icon={<ArrowDownload24Regular />}
                    size="small"
                    onClick={() => handleDownloadProof(proof)}
                  >
                    Download
                  </Button>
                </div>

                <div style={{ marginTop: '12px' }}>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginBottom: '12px' }}>
                    <div>
                      <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Leaf Hash</Text>
                      <Text className={styles.proofHash}>{proof.leafHash}</Text>
                    </div>
                    <div>
                      <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Root Hash</Text>
                      <Text className={styles.proofHash}>{proof.rootHash}</Text>
                    </div>
                  </div>

                  <Text weight="semibold" size={200} style={{ marginBottom: '8px', display: 'block' }}>
                    Proof Path ({proof.path.length} steps)
                  </Text>
                  <div className={styles.proofSteps}>
                    {proof.path.map((step, index) => (
                      <div key={index} className={styles.proofStep}>
                        <div className={styles.stepNumber}>{index + 1}</div>
                        <div className={styles.stepContent}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <Badge color={step.position === 'left' ? 'informative' : 'warning'}>
                              {step.position.toUpperCase()}
                            </Badge>
                            <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>sibling</Text>
                          </div>
                          <Text size={200} style={{ fontFamily: tokens.fontFamilyMonospace, wordBreak: 'break-all' }}>
                            {step.sibling}
                          </Text>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}

        {/* Proof Verification Tab */}
        {selectedTab === 'verification' && (
          <div className={styles.verificationPanel}>
            <Text weight="semibold" size={400} style={{ marginBottom: '16px', display: 'block' }}>
              Verify Merkle Proof
            </Text>

            <div className={styles.verificationInput}>
              <Textarea
                placeholder="Paste Merkle proof JSON here..."
                value={verificationInput}
                onChange={(_, data) => setVerificationInput(data.value)}
                style={{ flex: 1, minHeight: '200px' }}
              />
            </div>

            <Button
              appearance="primary"
              icon={<Search24Regular />}
              onClick={handleVerifyProof}
              disabled={loading || !verificationInput.trim()}
              style={{ width: '100%' }}
            >
              {loading ? <Spinner size="tiny" /> : 'Verify Proof'}
            </Button>

            {verificationResult && (
              <div
                className={`${styles.verificationResult} ${verificationResult.valid ? styles.verificationSuccess : styles.verificationFailure
                  } `}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '12px' }}>
                  {verificationResult.valid ? (
                    <>
                      <CheckmarkCircle24Regular style={{ fontSize: '32px', color: tokens.colorPaletteGreenForeground1 }} />
                      <div>
                        <Text weight="bold" size={500}>Proof Valid</Text>
                        <Text size={300} style={{ display: 'block' }}>
                          The Merkle proof is cryptographically valid
                        </Text>
                      </div>
                    </>
                  ) : (
                    <>
                      <DismissCircle24Regular style={{ fontSize: '32px', color: tokens.colorPaletteRedForeground1 }} />
                      <div>
                        <Text weight="bold" size={500}>Proof Invalid</Text>
                        <Text size={300} style={{ display: 'block' }}>
                          The Merkle proof could not be verified
                        </Text>
                      </div>
                    </>
                  )}
                </div>

                <div className={styles.verificationDetails}>
                  <div className={styles.detailItem}>
                    <Text className={styles.detailLabel}>Token ID</Text>
                    <Text className={styles.detailValue}>{verificationResult.tokenId}</Text>
                  </div>
                  <div className={styles.detailItem}>
                    <Text className={styles.detailLabel}>Leaf Hash</Text>
                    <Text className={styles.detailValue}>
                      {verificationResult.leafHash?.substring(0, 16)}...
                    </Text>
                  </div>
                  <div className={styles.detailItem}>
                    <Text className={styles.detailLabel}>Root Hash</Text>
                    <Text className={styles.detailValue}>
                      {verificationResult.rootHash?.substring(0, 16)}...
                    </Text>
                  </div>
                  <div className={styles.detailItem}>
                    <Text className={styles.detailLabel}>Computed Root</Text>
                    <Text className={styles.detailValue}>
                      {verificationResult.computedRoot?.substring(0, 16)}...
                    </Text>
                  </div>
                  <div className={styles.detailItem}>
                    <Text className={styles.detailLabel}>Path Length</Text>
                    <Text className={styles.detailValue}>{verificationResult.pathLength}</Text>
                  </div>
                  <div className={styles.detailItem}>
                    <Text className={styles.detailLabel}>Verified At</Text>
                    <Text className={styles.detailValue}>{verificationResult.timestamp}</Text>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Revocation List Tab */}
        {selectedTab === 'revocations' && (
          <div>
            <Text weight="semibold" style={{ marginBottom: '16px', display: 'block' }}>
              Revocation Registry
            </Text>

            {revocations.map((revocation) => (
              <Card key={revocation.id} className={styles.revocationCard}>
                <div className={styles.revocationHeader}>
                  <div className={styles.revocationInfo}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <Text className={styles.revocationTokenId}>{revocation.tokenId}</Text>
                      {revocation.verified ? (
                        <Badge color="success" icon={<CheckmarkCircle24Regular />}>VERIFIED</Badge>
                      ) : (
                        <Badge color="warning" icon={<Info24Regular />}>PENDING</Badge>
                      )}
                    </div>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                      Revoked by {revocation.revokedBy} on {revocation.timestamp}
                    </Text>
                    <Text size={300} style={{ marginTop: '4px' }}>
                      Reason: {revocation.reason}
                    </Text>
                  </div>
                </div>

                <div className={styles.revocationMetrics}>
                  <div className={styles.metric}>
                    <Text className={styles.metricLabel}>Leaf Hash</Text>
                    <Text className={styles.metricValue} style={{ fontSize: tokens.fontSizeBase200 }}>
                      {revocation.leafHash.substring(0, 16)}...
                    </Text>
                  </div>
                  <div className={styles.metric}>
                    <Text className={styles.metricLabel}>Merkle Root</Text>
                    <Text className={styles.metricValue} style={{ fontSize: tokens.fontSizeBase200 }}>
                      {revocation.merkleRoot.substring(0, 16)}...
                    </Text>
                  </div>
                  <div className={styles.metric}>
                    <Text className={styles.metricLabel}>Block Height</Text>
                    <Text className={styles.metricValue}>{revocation.blockHeight.toLocaleString()}</Text>
                  </div>
                  <div className={styles.metric}>
                    <Text className={styles.metricLabel}>Status</Text>
                    <Text className={styles.metricValue}>
                      {revocation.verified ? 'Verified' : 'Pending'}
                    </Text>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}

        {/* Append-Only Log Tab */}
        {selectedTab === 'log' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '16px' }}>
              <Text weight="semibold">Append-Only Audit Log</Text>
              <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                {logSize} entries (cryptographically linked)
              </Text>
            </div>

            <div className={styles.logContainer}>
              {logEntries.map((entry) => (
                <div key={entry.index} className={styles.logEntry}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
                    <div style={{ display: 'flex', gap: '12px' }}>
                      <Text className={styles.logTimestamp}>[{entry.index}]</Text>
                      <Text className={styles.logTimestamp}>{entry.timestamp}</Text>
                      <Badge color={entry.operation === 'append' ? 'success' : 'informative'}>
                        {entry.operation.toUpperCase()}
                      </Badge>
                    </div>
                    <LockClosed24Regular style={{ fontSize: '16px', color: tokens.colorPaletteGreenForeground1 }} />
                  </div>
                  <div style={{ marginBottom: '4px' }}>
                    <Text size={200}>Data: {entry.data}</Text>
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                    <Text size={100} style={{ color: tokens.colorNeutralForeground3 }}>
                      Hash: <span className={styles.logHash}>{entry.hash}</span>
                    </Text>
                    <Text size={100} style={{ color: tokens.colorNeutralForeground3 }}>
                      Prev: <span style={{ color: tokens.colorNeutralForeground2 }}>{entry.previousHash}</span>
                    </Text>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default RevocationTransparency;
