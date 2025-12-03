import { useState, useEffect } from 'react'
import {
  makeStyles,
  Text,
  Card,
  Button,
  Badge,
  tokens,
  Spinner,
  TabList,
  Tab,
  Dialog,
  DialogTrigger,
  DialogSurface,
  DialogTitle,
  DialogBody,
  DialogActions,
  DialogContent,
  Input,
  Label,
  Dropdown,
  Option,
  Switch,
  DataGrid,
  DataGridBody,
  DataGridRow,
  DataGridHeader,
  DataGridHeaderCell,
  DataGridCell,
  TableColumnDefinition,
  createTableColumn,
  MessageBar,
  MessageBarBody,
  ProgressBar,
  Textarea,
  Field,
} from '@fluentui/react-components'
import {
  DocumentBulletList24Regular,
  Shield24Regular,
  Link24Regular,
  CheckmarkCircle24Regular,
  Download24Regular,
  Settings24Regular,
  Play24Regular,
  Stop24Regular,
  Filter24Regular,
  ArrowExport24Regular,
  Eye24Regular,
  DocumentSearch24Regular,
} from '@fluentui/react-icons'

const useStyles = makeStyles({
  container: {
    padding: tokens.spacingVerticalXXL,
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalXL,
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  metricsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
    gap: tokens.spacingHorizontalL,
  },
  metricCard: {
    padding: tokens.spacingVerticalL,
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalM,
  },
  metricHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: tokens.spacingHorizontalM,
  },
  metricValue: {
    fontSize: tokens.fontSizeHero900,
    fontWeight: tokens.fontWeightSemibold,
    lineHeight: tokens.lineHeightHero900,
  },
  content: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalL,
  },
  tabContent: {
    marginTop: tokens.spacingVerticalL,
  },
  controlBar: {
    display: 'flex',
    gap: tokens.spacingHorizontalM,
    alignItems: 'center',
    flexWrap: 'wrap',
  },
  filterPanel: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    gap: tokens.spacingHorizontalM,
    padding: tokens.spacingVerticalL,
    backgroundColor: tokens.colorNeutralBackground2,
    borderRadius: tokens.borderRadiusMedium,
  },
  complianceGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
    gap: tokens.spacingHorizontalL,
  },
  complianceCard: {
    padding: tokens.spacingVerticalL,
  },
  complianceHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: tokens.spacingVerticalM,
  },
  complianceMetrics: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalS,
  },
  metricRow: {
    display: 'flex',
    justifyContent: 'space-between',
    padding: tokens.spacingVerticalXS,
  },
  progressSection: {
    marginTop: tokens.spacingVerticalM,
  },
  correlationList: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalM,
  },
  correlationCard: {
    padding: tokens.spacingVerticalL,
  },
  eventsList: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalS,
    marginTop: tokens.spacingVerticalM,
  },
  eventItem: {
    padding: tokens.spacingVerticalS,
    backgroundColor: tokens.colorNeutralBackground2,
    borderRadius: tokens.borderRadiusSmall,
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  verificationPanel: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalL,
  },
  verificationCard: {
    padding: tokens.spacingVerticalL,
  },
  verificationResult: {
    padding: tokens.spacingVerticalM,
    backgroundColor: tokens.colorNeutralBackground2,
    borderRadius: tokens.borderRadiusMedium,
    marginTop: tokens.spacingVerticalM,
  },
  siemList: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalM,
  },
  siemCard: {
    padding: tokens.spacingVerticalL,
  },
  siemHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: tokens.spacingVerticalM,
  },
  siemMetrics: {
    display: 'grid',
    gridTemplateColumns: 'repeat(3, 1fr)',
    gap: tokens.spacingHorizontalM,
    marginTop: tokens.spacingVerticalM,
  },
  formField: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalS,
    marginBottom: tokens.spacingVerticalM,
  },
})

interface AuditEvent {
  id: string
  timestamp: string
  actor: string
  action: string
  resource: string
  result: string
  ip: string
  category: string
  severity: string
  tamperProof: boolean
  metadata: Record<string, any>
}

interface ComplianceReport {
  id: string
  framework: string
  standard: string
  status: 'compliant' | 'non-compliant' | 'partial'
  coverage: number
  violations: number
  lastAudit: string
  requirements: number
  met: number
}

interface EventCorrelation {
  id: string
  pattern: string
  description: string
  severity: string
  events: AuditEvent[]
  firstSeen: string
  lastSeen: string
  occurrences: number
  confidence: number
}

interface TamperVerification {
  eventId: string
  timestamp: string
  status: 'verified' | 'tampered' | 'unknown'
  hash: string
  previousHash: string
  signature: string
  verified: boolean
}

interface SIEMIntegration {
  id: string
  name: string
  type: string
  endpoint: string
  enabled: boolean
  format: string
  eventsSent: number
  lastSync: string
  status: 'active' | 'inactive' | 'error'
}

export default function AuditTrail() {
  const styles = useStyles()
  const [selectedTab, setSelectedTab] = useState<string>('events')
  const [loading, setLoading] = useState(true)
  const [streaming, setStreaming] = useState(false)

  // Audit Events state
  const [auditEvents, setAuditEvents] = useState<AuditEvent[]>([])
  const [filterCategory, setFilterCategory] = useState<string>('')
  const [filterSeverity, setFilterSeverity] = useState<string>('')
  const [filterActor, setFilterActor] = useState<string>('')
  const [filterResult, setFilterResult] = useState<string>('')

  // Compliance state
  const [complianceReports, setComplianceReports] = useState<ComplianceReport[]>([])

  // Correlation state
  const [correlations, setCorrelations] = useState<EventCorrelation[]>([])

  // Verification state
  const [verifications, setVerifications] = useState<TamperVerification[]>([])
  const [verifyEventId, setVerifyEventId] = useState('')
  const [verificationResult, setVerificationResult] = useState<TamperVerification | null>(null)

  // SIEM state
  const [siemIntegrations, setSiemIntegrations] = useState<SIEMIntegration[]>([])
  const [siemDialogOpen, setSiemDialogOpen] = useState(false)
  const [siemName, setSiemName] = useState('')
  const [siemType, setSiemType] = useState('')
  const [siemEndpoint, setSiemEndpoint] = useState('')
  const [siemFormat, setSiemFormat] = useState('json')

  // Export state
  const [exportDialogOpen, setExportDialogOpen] = useState(false)
  const [exportFormat, setExportFormat] = useState('json')
  const [exportDateRange, setExportDateRange] = useState('last-24h')

  useEffect(() => {
    fetchAllData()
  }, [])

  useEffect(() => {
    if (streaming) {
      const interval = setInterval(() => {
        fetchAuditEvents()
      }, 3000)
      return () => clearInterval(interval)
    }
  }, [streaming, filterCategory, filterSeverity, filterActor, filterResult])

  const fetchAllData = async () => {
    setLoading(true)
    try {
      await Promise.all([
        fetchAuditEvents(),
        fetchComplianceReports(),
        fetchCorrelations(),
        fetchSIEMIntegrations(),
      ])
    } finally {
      setLoading(false)
    }
  }

  const fetchAuditEvents = async () => {
    try {
      const params = new URLSearchParams()
      if (filterCategory) params.append('category', filterCategory)
      if (filterSeverity) params.append('severity', filterSeverity)
      if (filterActor) params.append('actor', filterActor)
      if (filterResult) params.append('result', filterResult)

      const response = await fetch(`/api/admin/audit/events?${params}`)
      const data = await response.json()
      setAuditEvents(data.events || [])
    } catch (error) {
      console.error('Failed to fetch audit events:', error)
    }
  }

  const fetchComplianceReports = async () => {
    try {
      const response = await fetch('/api/admin/audit/compliance')
      const data = await response.json()
      setComplianceReports(data.reports || [])
    } catch (error) {
      console.error('Failed to fetch compliance reports:', error)
    }
  }

  const fetchCorrelations = async () => {
    try {
      const response = await fetch('/api/admin/audit/correlations')
      const data = await response.json()
      setCorrelations(data.correlations || [])
    } catch (error) {
      console.error('Failed to fetch correlations:', error)
    }
  }

  const fetchSIEMIntegrations = async () => {
    try {
      const response = await fetch('/api/admin/audit/siem')
      const data = await response.json()
      setSiemIntegrations(data.integrations || [])
    } catch (error) {
      console.error('Failed to fetch SIEM integrations:', error)
    }
  }

  const handleVerifyEvent = async () => {
    if (!verifyEventId) return
    
    try {
      const response = await fetch(`/api/admin/audit/verify/${verifyEventId}`)
      const data = await response.json()
      setVerificationResult(data)
    } catch (error) {
      console.error('Failed to verify event:', error)
    }
  }

  const handleExportAuditTrail = async () => {
    try {
      const response = await fetch('/api/admin/audit/export', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          format: exportFormat,
          dateRange: exportDateRange,
        }),
      })
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `audit-trail-${Date.now()}.${exportFormat}`
      a.click()
      setExportDialogOpen(false)
    } catch (error) {
      console.error('Failed to export audit trail:', error)
    }
  }

  const handleCreateSIEM = async () => {
    try {
      await fetch('/api/admin/audit/siem', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: siemName,
          type: siemType,
          endpoint: siemEndpoint,
          format: siemFormat,
        }),
      })
      setSiemDialogOpen(false)
      resetSiemForm()
      fetchSIEMIntegrations()
    } catch (error) {
      console.error('Failed to create SIEM integration:', error)
    }
  }

  const handleToggleSIEM = async (id: string, enabled: boolean) => {
    try {
      await fetch(`/api/admin/audit/siem/${id}/toggle`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled }),
      })
      fetchSIEMIntegrations()
    } catch (error) {
      console.error('Failed to toggle SIEM:', error)
    }
  }

  const handleDeleteSIEM = async (id: string) => {
    try {
      await fetch(`/api/admin/audit/siem/${id}`, { method: 'DELETE' })
      fetchSIEMIntegrations()
    } catch (error) {
      console.error('Failed to delete SIEM:', error)
    }
  }

  const handleTestSIEM = async (id: string) => {
    try {
      const response = await fetch(`/api/admin/audit/siem/${id}/test`, { method: 'POST' })
      const data = await response.json()
      alert(`SIEM Test Result: ${data.success ? 'Success' : 'Failed'}\n${data.message}`)
    } catch (error) {
      console.error('Failed to test SIEM:', error)
    }
  }

  const resetSiemForm = () => {
    setSiemName('')
    setSiemType('')
    setSiemEndpoint('')
    setSiemFormat('json')
  }

  const getSeverityBadge = (severity: string) => {
    const colorMap: Record<string, 'danger' | 'warning' | 'success' | 'informative' | 'subtle'> = {
      critical: 'danger',
      high: 'danger',
      medium: 'warning',
      low: 'informative',
      info: 'subtle',
    }
    return <Badge appearance="filled" color={colorMap[severity] || 'subtle'}>{severity.toUpperCase()}</Badge>
  }

  const getResultBadge = (result: string) => {
    const colorMap: Record<string, 'success' | 'danger' | 'warning'> = {
      success: 'success',
      failure: 'danger',
      denied: 'warning',
    }
    return <Badge appearance="filled" color={colorMap[result] || 'subtle'}>{result.toUpperCase()}</Badge>
  }

  const getComplianceStatusBadge = (status: string) => {
    const colorMap: Record<string, 'success' | 'danger' | 'warning'> = {
      compliant: 'success',
      'non-compliant': 'danger',
      partial: 'warning',
    }
    return <Badge appearance="filled" color={colorMap[status]}>{status.toUpperCase()}</Badge>
  }

  const getSIEMStatusBadge = (status: string) => {
    const colorMap: Record<string, 'success' | 'danger' | 'warning'> = {
      active: 'success',
      inactive: 'subtle',
      error: 'danger',
    }
    return <Badge appearance="filled" color={colorMap[status]}>{status.toUpperCase()}</Badge>
  }

  // Calculate overview metrics
  const totalEvents = auditEvents.length
  const tamperProofEvents = auditEvents.filter(e => e.tamperProof).length
  const criticalEvents = auditEvents.filter(e => e.severity === 'critical').length
  const failedActions = auditEvents.filter(e => e.result === 'failure' || e.result === 'denied').length

  const eventColumns: TableColumnDefinition<AuditEvent>[] = [
    createTableColumn<AuditEvent>({
      columnId: 'timestamp',
      renderHeaderCell: () => 'Timestamp',
      renderCell: (item) => <Text size={200}>{new Date(item.timestamp).toLocaleString()}</Text>,
    }),
    createTableColumn<AuditEvent>({
      columnId: 'actor',
      renderHeaderCell: () => 'Actor',
      renderCell: (item) => <Text weight="semibold">{item.actor}</Text>,
    }),
    createTableColumn<AuditEvent>({
      columnId: 'action',
      renderHeaderCell: () => 'Action',
      renderCell: (item) => <Text>{item.action}</Text>,
    }),
    createTableColumn<AuditEvent>({
      columnId: 'resource',
      renderHeaderCell: () => 'Resource',
      renderCell: (item) => <Text>{item.resource}</Text>,
    }),
    createTableColumn<AuditEvent>({
      columnId: 'result',
      renderHeaderCell: () => 'Result',
      renderCell: (item) => getResultBadge(item.result),
    }),
    createTableColumn<AuditEvent>({
      columnId: 'severity',
      renderHeaderCell: () => 'Severity',
      renderCell: (item) => getSeverityBadge(item.severity),
    }),
    createTableColumn<AuditEvent>({
      columnId: 'tamperProof',
      renderHeaderCell: () => 'Tamper Proof',
      renderCell: (item) => item.tamperProof ? <CheckmarkCircle24Regular style={{ color: tokens.colorPaletteGreenForeground1 }} /> : <Text>-</Text>,
    }),
    createTableColumn<AuditEvent>({
      columnId: 'actions',
      renderHeaderCell: () => 'Actions',
      renderCell: (item) => (
        <Button
          size="small"
          appearance="subtle"
          icon={<Eye24Regular />}
          onClick={() => {
            setVerifyEventId(item.id)
            handleVerifyEvent()
          }}
        >
          Verify
        </Button>
      ),
    }),
  ]

  if (loading) {
    return (
      <div className={styles.container}>
        <Spinner size="large" label="Loading audit trail..." />
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div>
          <Text size={900} weight="semibold">Audit Trail</Text>
          <Text size={300}>Monitor audit events, compliance, and tamper evidence</Text>
        </div>
        <div style={{ display: 'flex', gap: tokens.spacingHorizontalM }}>
          <Dialog open={exportDialogOpen} onOpenChange={(_, data) => setExportDialogOpen(data.open)}>
            <DialogTrigger>
              <Button appearance="primary" icon={<Download24Regular />}>Export</Button>
            </DialogTrigger>
            <DialogSurface>
              <DialogBody>
                <DialogTitle>Export Audit Trail</DialogTitle>
                <DialogContent>
                  <div className={styles.formField}>
                    <Label>Export Format</Label>
                    <Dropdown
                      value={exportFormat}
                      onOptionSelect={(_, data) => setExportFormat(data.optionValue || 'json')}
                    >
                      <Option value="json">JSON</Option>
                      <Option value="csv">CSV</Option>
                      <Option value="syslog">Syslog</Option>
                      <Option value="cef">CEF (Common Event Format)</Option>
                    </Dropdown>
                  </div>
                  <div className={styles.formField}>
                    <Label>Date Range</Label>
                    <Dropdown
                      value={exportDateRange}
                      onOptionSelect={(_, data) => setExportDateRange(data.optionValue || 'last-24h')}
                    >
                      <Option value="last-1h">Last Hour</Option>
                      <Option value="last-24h">Last 24 Hours</Option>
                      <Option value="last-7d">Last 7 Days</Option>
                      <Option value="last-30d">Last 30 Days</Option>
                      <Option value="all">All Events</Option>
                    </Dropdown>
                  </div>
                </DialogContent>
                <DialogActions>
                  <Button appearance="secondary" onClick={() => setExportDialogOpen(false)}>Cancel</Button>
                  <Button appearance="primary" onClick={handleExportAuditTrail}>Export</Button>
                </DialogActions>
              </DialogBody>
            </DialogSurface>
          </Dialog>
        </div>
      </div>

      {/* Overview Metrics */}
      <div className={styles.metricsGrid}>
        <Card className={styles.metricCard}>
          <div className={styles.metricHeader}>
            <DocumentBulletList24Regular />
            <Text weight="semibold">Total Events</Text>
          </div>
          <Text className={styles.metricValue}>{totalEvents}</Text>
          <Text size={200}>{tamperProofEvents} Tamper-Proof</Text>
        </Card>

        <Card className={styles.metricCard}>
          <div className={styles.metricHeader}>
            <Shield24Regular />
            <Text weight="semibold">Critical Events</Text>
          </div>
          <Text className={styles.metricValue}>{criticalEvents}</Text>
          <Text size={200}>Require Immediate Attention</Text>
        </Card>

        <Card className={styles.metricCard}>
          <div className={styles.metricHeader}>
            <DocumentSearch24Regular />
            <Text weight="semibold">Failed Actions</Text>
          </div>
          <Text className={styles.metricValue}>{failedActions}</Text>
          <Text size={200}>Denied or Failed</Text>
        </Card>

        <Card className={styles.metricCard}>
          <div className={styles.metricHeader}>
            <Link24Regular />
            <Text weight="semibold">Correlations</Text>
          </div>
          <Text className={styles.metricValue}>{correlations.length}</Text>
          <Text size={200}>Active Patterns</Text>
        </Card>
      </div>

      {/* Tabs */}
      <div className={styles.content}>
        <TabList selectedValue={selectedTab} onTabSelect={(_, data) => setSelectedTab(data.value as string)}>
          <Tab value="events">Live Events</Tab>
          <Tab value="compliance">Compliance</Tab>
          <Tab value="correlation">Event Correlation</Tab>
          <Tab value="verification">Tamper Verification</Tab>
          <Tab value="siem">SIEM Integration</Tab>
        </TabList>

        <div className={styles.tabContent}>
          {/* Live Events Tab */}
          {selectedTab === 'events' && (
            <div>
              <div className={styles.controlBar}>
                <Button
                  appearance={streaming ? 'secondary' : 'primary'}
                  icon={streaming ? <Stop24Regular /> : <Play24Regular />}
                  onClick={() => setStreaming(!streaming)}
                >
                  {streaming ? 'Stop Streaming' : 'Start Streaming'}
                </Button>
                <Button icon={<Filter24Regular />}>Filters</Button>
              </div>

              <div className={styles.filterPanel}>
                <div>
                  <Label>Category</Label>
                  <Dropdown
                    placeholder="All Categories"
                    value={filterCategory}
                    onOptionSelect={(_, data) => {
                      setFilterCategory(data.optionValue || '')
                      fetchAuditEvents()
                    }}
                  >
                    <Option value="">All</Option>
                    <Option value="auth">Authentication</Option>
                    <Option value="authz">Authorization</Option>
                    <Option value="token">Token</Option>
                    <Option value="admin">Admin</Option>
                    <Option value="system">System</Option>
                  </Dropdown>
                </div>
                <div>
                  <Label>Severity</Label>
                  <Dropdown
                    placeholder="All Severities"
                    value={filterSeverity}
                    onOptionSelect={(_, data) => {
                      setFilterSeverity(data.optionValue || '')
                      fetchAuditEvents()
                    }}
                  >
                    <Option value="">All</Option>
                    <Option value="critical">Critical</Option>
                    <Option value="high">High</Option>
                    <Option value="medium">Medium</Option>
                    <Option value="low">Low</Option>
                    <Option value="info">Info</Option>
                  </Dropdown>
                </div>
                <div>
                  <Label>Result</Label>
                  <Dropdown
                    placeholder="All Results"
                    value={filterResult}
                    onOptionSelect={(_, data) => {
                      setFilterResult(data.optionValue || '')
                      fetchAuditEvents()
                    }}
                  >
                    <Option value="">All</Option>
                    <Option value="success">Success</Option>
                    <Option value="failure">Failure</Option>
                    <Option value="denied">Denied</Option>
                  </Dropdown>
                </div>
                <div>
                  <Label>Actor</Label>
                  <Input
                    placeholder="Filter by actor..."
                    value={filterActor}
                    onChange={(_, data) => setFilterActor(data.value)}
                  />
                </div>
              </div>

              <DataGrid items={auditEvents} columns={eventColumns} sortable resizableColumns>
                <DataGridHeader>
                  <DataGridRow>
                    {({ renderHeaderCell }) => <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>}
                  </DataGridRow>
                </DataGridHeader>
                <DataGridBody<AuditEvent>>
                  {({ item, rowId }) => (
                    <DataGridRow<AuditEvent> key={rowId}>
                      {({ renderCell }) => <DataGridCell>{renderCell(item)}</DataGridCell>}
                    </DataGridRow>
                  )}
                </DataGridBody>
              </DataGrid>
            </div>
          )}

          {/* Compliance Tab */}
          {selectedTab === 'compliance' && (
            <div>
              <div className={styles.header}>
                <Text size={600} weight="semibold">Compliance Reports</Text>
              </div>

              <div className={styles.complianceGrid}>
                {complianceReports.map((report) => (
                  <Card key={report.id} className={styles.complianceCard}>
                    <div className={styles.complianceHeader}>
                      <div>
                        <Text weight="semibold" size={500}>{report.framework}</Text>
                        <Text size={200}>{report.standard}</Text>
                      </div>
                      {getComplianceStatusBadge(report.status)}
                    </div>

                    <div className={styles.progressSection}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: tokens.spacingVerticalXS }}>
                        <Text size={200}>Coverage</Text>
                        <Text size={200}>{report.coverage}%</Text>
                      </div>
                      <ProgressBar value={report.coverage / 100} max={1} />
                    </div>

                    <div className={styles.complianceMetrics}>
                      <div className={styles.metricRow}>
                        <Text size={200}>Requirements:</Text>
                        <Text weight="semibold">{report.requirements}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Met:</Text>
                        <Text weight="semibold" style={{ color: tokens.colorPaletteGreenForeground1 }}>
                          {report.met}
                        </Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Violations:</Text>
                        <Text weight="semibold" style={{ color: tokens.colorPaletteRedForeground1 }}>
                          {report.violations}
                        </Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Last Audit:</Text>
                        <Text size={200}>{new Date(report.lastAudit).toLocaleDateString()}</Text>
                      </div>
                    </div>

                    <Button appearance="secondary" size="small" style={{ marginTop: tokens.spacingVerticalM }}>
                      View Full Report
                    </Button>
                  </Card>
                ))}
              </div>
            </div>
          )}

          {/* Correlation Tab */}
          {selectedTab === 'correlation' && (
            <div>
              <div className={styles.header}>
                <Text size={600} weight="semibold">Event Correlation Patterns</Text>
              </div>

              <div className={styles.correlationList}>
                {correlations.map((corr) => (
                  <Card key={corr.id} className={styles.correlationCard}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <div style={{ flex: 1 }}>
                        <Text weight="semibold" size={500}>{corr.pattern}</Text>
                        <Text size={200}>{corr.description}</Text>
                        <div style={{ marginTop: tokens.spacingVerticalS }}>
                          {getSeverityBadge(corr.severity)}
                          <Badge appearance="tint" style={{ marginLeft: tokens.spacingHorizontalS }}>
                            {corr.confidence}% confidence
                          </Badge>
                        </div>
                      </div>
                    </div>

                    <div className={styles.complianceMetrics} style={{ marginTop: tokens.spacingVerticalM }}>
                      <div className={styles.metricRow}>
                        <Text size={200}>Occurrences:</Text>
                        <Text weight="semibold">{corr.occurrences}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>First Seen:</Text>
                        <Text size={200}>{new Date(corr.firstSeen).toLocaleString()}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Last Seen:</Text>
                        <Text size={200}>{new Date(corr.lastSeen).toLocaleString()}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Related Events:</Text>
                        <Text weight="semibold">{corr.events.length}</Text>
                      </div>
                    </div>

                    <Text size={400} weight="semibold" style={{ marginTop: tokens.spacingVerticalM }}>
                      Correlated Events:
                    </Text>
                    <div className={styles.eventsList}>
                      {corr.events.slice(0, 3).map((event) => (
                        <div key={event.id} className={styles.eventItem}>
                          <div>
                            <Text size={200} weight="semibold">{event.action}</Text>
                            <Text size={100}> by {event.actor}</Text>
                          </div>
                          <Text size={100}>{new Date(event.timestamp).toLocaleTimeString()}</Text>
                        </div>
                      ))}
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          )}

          {/* Tamper Verification Tab */}
          {selectedTab === 'verification' && (
            <div className={styles.verificationPanel}>
              <Card className={styles.verificationCard}>
                <Text size={600} weight="semibold">Verify Event Integrity</Text>
                <Text size={200}>Check if an audit event has been tampered with</Text>

                <div className={styles.formField} style={{ marginTop: tokens.spacingVerticalL }}>
                  <Label>Event ID</Label>
                  <Input
                    placeholder="Enter event ID..."
                    value={verifyEventId}
                    onChange={(_, data) => setVerifyEventId(data.value)}
                  />
                </div>

                <Button appearance="primary" onClick={handleVerifyEvent} disabled={!verifyEventId}>
                  Verify Event
                </Button>

                {verificationResult && (
                  <div className={styles.verificationResult}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: tokens.spacingHorizontalM, marginBottom: tokens.spacingVerticalM }}>
                      {verificationResult.verified ? (
                        <>
                          <CheckmarkCircle24Regular style={{ color: tokens.colorPaletteGreenForeground1, fontSize: '32px' }} />
                          <div>
                            <Text size={500} weight="semibold">Verified</Text>
                            <Text size={200}>Event integrity confirmed</Text>
                          </div>
                        </>
                      ) : (
                        <>
                          <Shield24Regular style={{ color: tokens.colorPaletteRedForeground1, fontSize: '32px' }} />
                          <div>
                            <Text size={500} weight="semibold">Tampered</Text>
                            <Text size={200}>Event has been modified</Text>
                          </div>
                        </>
                      )}
                    </div>

                    <div className={styles.complianceMetrics}>
                      <div className={styles.metricRow}>
                        <Text size={200}>Event ID:</Text>
                        <Text size={200} style={{ fontFamily: 'monospace' }}>{verificationResult.eventId}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Hash:</Text>
                        <Text size={200} style={{ fontFamily: 'monospace' }}>{verificationResult.hash.substring(0, 32)}...</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Previous Hash:</Text>
                        <Text size={200} style={{ fontFamily: 'monospace' }}>{verificationResult.previousHash.substring(0, 32)}...</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Signature:</Text>
                        <Text size={200} style={{ fontFamily: 'monospace' }}>{verificationResult.signature.substring(0, 32)}...</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Timestamp:</Text>
                        <Text size={200}>{new Date(verificationResult.timestamp).toLocaleString()}</Text>
                      </div>
                    </div>
                  </div>
                )}
              </Card>

              <Card className={styles.verificationCard}>
                <Text size={600} weight="semibold">Recent Verifications</Text>
                <div className={styles.eventsList} style={{ marginTop: tokens.spacingVerticalM }}>
                  {verifications.slice(0, 5).map((ver) => (
                    <div key={ver.eventId} className={styles.eventItem}>
                      <div>
                        <Text size={200} style={{ fontFamily: 'monospace' }}>{ver.eventId}</Text>
                        <Text size={100}> • {new Date(ver.timestamp).toLocaleString()}</Text>
                      </div>
                      {ver.verified ? (
                        <Badge color="success">VERIFIED</Badge>
                      ) : (
                        <Badge color="danger">TAMPERED</Badge>
                      )}
                    </div>
                  ))}
                </div>
              </Card>
            </div>
          )}

          {/* SIEM Integration Tab */}
          {selectedTab === 'siem' && (
            <div>
              <div className={styles.header}>
                <Text size={600} weight="semibold">SIEM Integrations</Text>
                <Dialog open={siemDialogOpen} onOpenChange={(_, data) => setSiemDialogOpen(data.open)}>
                  <DialogTrigger>
                    <Button appearance="primary" icon={<Settings24Regular />}>Add SIEM</Button>
                  </DialogTrigger>
                  <DialogSurface>
                    <DialogBody>
                      <DialogTitle>Create SIEM Integration</DialogTitle>
                      <DialogContent>
                        <div className={styles.formField}>
                          <Label required>Name</Label>
                          <Input value={siemName} onChange={(_, data) => setSiemName(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label required>Type</Label>
                          <Dropdown
                            value={siemType}
                            onOptionSelect={(_, data) => setSiemType(data.optionValue || '')}
                          >
                            <Option value="splunk">Splunk</Option>
                            <Option value="elastic">Elastic SIEM</Option>
                            <Option value="qradar">IBM QRadar</Option>
                            <Option value="sentinel">Azure Sentinel</Option>
                            <Option value="sumologic">Sumo Logic</Option>
                            <Option value="datadog">Datadog</Option>
                          </Dropdown>
                        </div>
                        <div className={styles.formField}>
                          <Label required>Endpoint URL</Label>
                          <Input
                            type="url"
                            placeholder="https://siem.example.com/api/events"
                            value={siemEndpoint}
                            onChange={(_, data) => setSiemEndpoint(data.value)}
                          />
                        </div>
                        <div className={styles.formField}>
                          <Label>Format</Label>
                          <Dropdown
                            value={siemFormat}
                            onOptionSelect={(_, data) => setSiemFormat(data.optionValue || 'json')}
                          >
                            <Option value="json">JSON</Option>
                            <Option value="cef">CEF</Option>
                            <Option value="syslog">Syslog</Option>
                            <Option value="leef">LEEF</Option>
                          </Dropdown>
                        </div>
                      </DialogContent>
                      <DialogActions>
                        <Button appearance="secondary" onClick={() => {
                          setSiemDialogOpen(false)
                          resetSiemForm()
                        }}>Cancel</Button>
                        <Button
                          appearance="primary"
                          onClick={handleCreateSIEM}
                          disabled={!siemName || !siemType || !siemEndpoint}
                        >
                          Create
                        </Button>
                      </DialogActions>
                    </DialogBody>
                  </DialogSurface>
                </Dialog>
              </div>

              <div className={styles.siemList}>
                {siemIntegrations.map((siem) => (
                  <Card key={siem.id} className={styles.siemCard}>
                    <div className={styles.siemHeader}>
                      <div>
                        <Text weight="semibold" size={500}>{siem.name}</Text>
                        <Text size={200}>{siem.type}</Text>
                        <div style={{ marginTop: tokens.spacingVerticalXS }}>
                          {getSIEMStatusBadge(siem.status)}
                          <Badge appearance="tint" style={{ marginLeft: tokens.spacingHorizontalS }}>
                            {siem.format.toUpperCase()}
                          </Badge>
                        </div>
                      </div>
                      <Switch
                        checked={siem.enabled}
                        onChange={(_, data) => handleToggleSIEM(siem.id, data.checked)}
                      />
                    </div>

                    <Text size={200} style={{ fontFamily: 'monospace', marginTop: tokens.spacingVerticalM }}>
                      {siem.endpoint}
                    </Text>

                    <div className={styles.siemMetrics}>
                      <div>
                        <Text size={200}>Events Sent</Text>
                        <Text weight="semibold">{siem.eventsSent.toLocaleString()}</Text>
                      </div>
                      <div>
                        <Text size={200}>Last Sync</Text>
                        <Text size={200}>{new Date(siem.lastSync).toLocaleString()}</Text>
                      </div>
                      <div>
                        <Text size={200}>Status</Text>
                        <Text weight="semibold">{siem.status}</Text>
                      </div>
                    </div>

                    <div style={{ display: 'flex', gap: tokens.spacingHorizontalS, marginTop: tokens.spacingVerticalM }}>
                      <Button appearance="secondary" size="small" onClick={() => handleTestSIEM(siem.id)}>
                        Test Connection
                      </Button>
                      <Button appearance="secondary" size="small" onClick={() => handleDeleteSIEM(siem.id)}>
                        Delete
                      </Button>
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
