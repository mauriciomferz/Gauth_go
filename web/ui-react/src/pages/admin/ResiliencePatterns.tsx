import { useState, useEffect } from 'react'
import { addTenantParam } from '../../utils/tenant'
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
  Slider,
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
} from '@fluentui/react-components'
import {
  ShieldCheckmark24Regular,
  Timer24Regular,
  ArrowRepeatAll24Regular,
  Grid24Regular,
  Add24Regular,
  Edit24Regular,
  Delete24Regular,
  CircleHalfFill24Regular,
  CheckmarkCircle24Filled,
  DismissCircle24Filled,
  Warning24Filled,
  Play24Regular,
  Stop24Regular,
} from '@fluentui/react-icons'

// Import admin API hooks
import { useCircuitBreakers, useRateLimiters, useRetryPolicies, useResilienceMutations } from '../../hooks/useAdminApi'

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
  circuitBreakerGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
    gap: tokens.spacingHorizontalL,
  },
  circuitCard: {
    padding: tokens.spacingVerticalL,
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalM,
  },
  circuitHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
  },
  circuitInfo: {
    flex: 1,
  },
  circuitActions: {
    display: 'flex',
    gap: tokens.spacingHorizontalS,
  },
  stateIndicator: {
    display: 'flex',
    alignItems: 'center',
    gap: tokens.spacingHorizontalS,
  },
  metrics: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: tokens.spacingVerticalS,
  },
  metricRow: {
    display: 'flex',
    justifyContent: 'space-between',
    padding: tokens.spacingVerticalXS,
  },
  formField: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalS,
    marginBottom: tokens.spacingVerticalM,
  },
  rateLimiterList: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalM,
  },
  rateLimiterCard: {
    padding: tokens.spacingVerticalL,
  },
  rateLimiterHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: tokens.spacingVerticalM,
  },
  rateLimiterMetrics: {
    display: 'grid',
    gridTemplateColumns: 'repeat(3, 1fr)',
    gap: tokens.spacingHorizontalL,
    marginTop: tokens.spacingVerticalM,
  },
  retryPolicyGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
    gap: tokens.spacingHorizontalL,
  },
  policyCard: {
    padding: tokens.spacingVerticalL,
  },
  bulkheadGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
    gap: tokens.spacingHorizontalL,
  },
  bulkheadCard: {
    padding: tokens.spacingVerticalL,
  },
  progressSection: {
    marginTop: tokens.spacingVerticalS,
  },
  compositeList: {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.spacingVerticalM,
  },
  compositeCard: {
    padding: tokens.spacingVerticalL,
  },
  patternsList: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: tokens.spacingHorizontalS,
    marginTop: tokens.spacingVerticalS,
  },
})

interface CircuitBreaker {
  id: string
  name: string
  service: string
  state: 'closed' | 'open' | 'half-open'
  failureThreshold: number
  successThreshold: number
  timeout: number
  failures: number
  successes: number
  lastStateChange: string
  totalRequests: number
  failureRate: number
}

interface RateLimiter {
  id: string
  name: string
  resource: string
  algorithm: 'token-bucket' | 'leaky-bucket' | 'fixed-window' | 'sliding-window'
  limit: number
  window: number
  burst?: number
  current: number
  throttled: number
  totalRequests: number
}

interface RetryPolicy {
  id: string
  name: string
  operation: string
  strategy: 'fixed' | 'exponential' | 'fibonacci' | 'linear'
  maxAttempts: number
  baseDelay: number
  maxDelay: number
  jitter: boolean
  totalRetries: number
  successfulRetries: number
  failedRetries: number
}

interface Bulkhead {
  id: string
  name: string
  service: string
  maxConcurrency: number
  maxQueueSize: number
  currentConcurrency: number
  queuedRequests: number
  rejectedRequests: number
  completedRequests: number
  timeout: number
}

interface CompositePattern {
  id: string
  name: string
  description: string
  patterns: string[]
  services: string[]
  enabled: boolean
  appliedCount: number
}

export default function ResiliencePatterns() {
  const styles = useStyles()
  const [selectedTab, setSelectedTab] = useState<string>('circuit-breakers')
  const [loading, setLoading] = useState(true)

  // Circuit Breaker state
  const [circuitBreakers, setCircuitBreakers] = useState<CircuitBreaker[]>([])
  const [cbDialogOpen, setCbDialogOpen] = useState(false)
  const [editingCb, setEditingCb] = useState<CircuitBreaker | null>(null)

  // Rate Limiter state
  const [rateLimiters, setRateLimiters] = useState<RateLimiter[]>([])
  const [rlDialogOpen, setRlDialogOpen] = useState(false)

  // Retry Policy state
  const [retryPolicies, setRetryPolicies] = useState<RetryPolicy[]>([])
  const [rpDialogOpen, setRpDialogOpen] = useState(false)

  // Bulkhead state
  const [bulkheads, setBulkheads] = useState<Bulkhead[]>([])
  const [bhDialogOpen, setBhDialogOpen] = useState(false)

  // Composite Pattern state
  const [compositePatterns, setCompositePatterns] = useState<CompositePattern[]>([])
  const [cpDialogOpen, setCpDialogOpen] = useState(false)

  // Form state for Circuit Breaker
  const [cbName, setCbName] = useState('')
  const [cbService, setCbService] = useState('')
  const [cbFailureThreshold, setCbFailureThreshold] = useState(5)
  const [cbSuccessThreshold, setCbSuccessThreshold] = useState(2)
  const [cbTimeout, setCbTimeout] = useState(60000)

  // Form state for Rate Limiter
  const [rlName, setRlName] = useState('')
  const [rlResource, setRlResource] = useState('')
  const [rlAlgorithm, setRlAlgorithm] = useState<string>('token-bucket')
  const [rlLimit, setRlLimit] = useState(100)
  const [rlWindow, setRlWindow] = useState(60)
  const [rlBurst, setRlBurst] = useState(120)

  // Form state for Retry Policy
  const [rpName, setRpName] = useState('')
  const [rpOperation, setRpOperation] = useState('')
  const [rpStrategy, setRpStrategy] = useState<string>('exponential')
  const [rpMaxAttempts, setRpMaxAttempts] = useState(3)
  const [rpBaseDelay, setRpBaseDelay] = useState(1000)
  const [rpMaxDelay, setRpMaxDelay] = useState(30000)
  const [rpJitter, setRpJitter] = useState(true)

  // Form state for Bulkhead
  const [bhName, setBhName] = useState('')
  const [bhService, setBhService] = useState('')
  const [bhMaxConcurrency, setBhMaxConcurrency] = useState(10)
  const [bhMaxQueueSize, setBhMaxQueueSize] = useState(20)
  const [bhTimeout, setBhTimeout] = useState(5000)

  // Form state for Composite Pattern
  const [cpName, setCpName] = useState('')
  const [cpDescription, setCpDescription] = useState('')
  const [cpPatterns, setCpPatterns] = useState<string[]>([])
  const [cpServices, setCpServices] = useState<string[]>([])

  // Use hooks for data fetching
  const { data: cbData, loading: cbLoading, refetch: refetchCbs } = useCircuitBreakers()
  const { data: rlData, loading: rlLoading, refetch: refetchRls } = useRateLimiters()
  const { data: rpData, loading: rpLoading, refetch: refetchRps } = useRetryPolicies()
  const { createCircuitBreaker, createRateLimiter, createRetryPolicy } = useResilienceMutations()

  // Update local state when data changes
  useEffect(() => {
    if (cbData?.circuitBreakers) setCircuitBreakers(cbData.circuitBreakers)
  }, [cbData])

  useEffect(() => {
    if (rlData?.rateLimiters) setRateLimiters(rlData.rateLimiters)
  }, [rlData])

  useEffect(() => {
    if (rpData?.retryPolicies) setRetryPolicies(rpData.retryPolicies)
  }, [rpData])

  // Set loading state based on hooks
  useEffect(() => {
    setLoading(cbLoading || rlLoading || rpLoading)
  }, [cbLoading, rlLoading, rpLoading])

  const fetchAllData = async () => {
    setLoading(true)
    try {
      await Promise.all([refetchCbs(), refetchRls(), refetchRps()])
    } finally {
      setLoading(false)
    }
  }

  const fetchBulkheads = async () => {
    try {
      const response = await fetch(addTenantParam('/api/admin/resilience/bulkheads'))
      const data = await response.json()
      setBulkheads(data.bulkheads || [])
    } catch (error) {
      console.error('Failed to fetch bulkheads:', error)
    }
  }

  const fetchCompositePatterns = async () => {
    try {
      const response = await fetch(addTenantParam('/api/admin/resilience/composite-patterns'))
      const data = await response.json()
      setCompositePatterns(data.patterns || [])
    } catch (error) {
      console.error('Failed to fetch composite patterns:', error)
    }
  }

  const handleCreateCircuitBreaker = async () => {
    try {
      await createCircuitBreaker({
        name: cbName,
        service: cbService,
        failureThreshold: cbFailureThreshold,
        successThreshold: cbSuccessThreshold,
        timeout: cbTimeout,
      })
      setCbDialogOpen(false)
      resetCbForm()
      refetchCbs()
    } catch (error) {
      console.error('Failed to create circuit breaker:', error)
    }
  }

  const handleUpdateCircuitBreaker = async () => {
    if (!editingCb) return
    try {
      await fetch(addTenantParam(`/api/admin/resilience/circuit-breakers/${editingCb.id}`), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: cbName,
          service: cbService,
          failureThreshold: cbFailureThreshold,
          successThreshold: cbSuccessThreshold,
          timeout: cbTimeout,
        }),
      })
      setCbDialogOpen(false)
      setEditingCb(null)
      resetCbForm()
      refetchCbs()
    } catch (error) {
      console.error('Failed to update circuit breaker:', error)
    }
  }

  const handleResetCircuitBreaker = async (id: string) => {
    try {
      await fetch(addTenantParam(`/api/admin/resilience/circuit-breakers/${id}/reset`), {
        method: 'POST',
      })
      refetchCbs()
    } catch (error) {
      console.error('Failed to reset circuit breaker:', error)
    }
  }

  const handleDeleteCircuitBreaker = async (id: string) => {
    try {
      await fetch(addTenantParam(`/api/admin/resilience/circuit-breakers/${id}`), {
        method: 'DELETE',
      })
      refetchCbs()
    } catch (error) {
      console.error('Failed to delete circuit breaker:', error)
    }
  }

  const handleCreateRateLimiter = async () => {
    try {
      await createRateLimiter({
        name: rlName,
        resource: rlResource,
        algorithm: rlAlgorithm,
        limit: rlLimit,
        window: rlWindow,
        burst: rlBurst,
      })
      setRlDialogOpen(false)
      resetRlForm()
      refetchRls()
    } catch (error) {
      console.error('Failed to create rate limiter:', error)
    }
  }

  const handleDeleteRateLimiter = async (id: string) => {
    try {
      await fetch(addTenantParam(`/api/admin/resilience/rate-limiters/${id}`), {
        method: 'DELETE',
      })
      refetchRls()
    } catch (error) {
      console.error('Failed to delete rate limiter:', error)
    }
  }

  const handleCreateRetryPolicy = async () => {
    try {
      await createRetryPolicy({
        name: rpName,
        operation: rpOperation,
        strategy: rpStrategy,
        maxAttempts: rpMaxAttempts,
        baseDelay: rpBaseDelay,
        maxDelay: rpMaxDelay,
        jitter: rpJitter,
      })
      setRpDialogOpen(false)
      resetRpForm()
      refetchRps()
    } catch (error) {
      console.error('Failed to create retry policy:', error)
    }
  }

  const handleDeleteRetryPolicy = async (id: string) => {
    try {
      await fetch(addTenantParam(`/api/admin/resilience/retry-policies/${id}`), {
        method: 'DELETE',
      })
      refetchRps()
    } catch (error) {
      console.error('Failed to delete retry policy:', error)
    }
  }

  const handleCreateBulkhead = async () => {
    try {
      await fetch(addTenantParam('/api/admin/resilience/bulkheads'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: bhName,
          service: bhService,
          maxConcurrency: bhMaxConcurrency,
          maxQueueSize: bhMaxQueueSize,
          timeout: bhTimeout,
        }),
      })
      setBhDialogOpen(false)
      resetBhForm()
      fetchBulkheads()
    } catch (error) {
      console.error('Failed to create bulkhead:', error)
    }
  }

  const handleDeleteBulkhead = async (id: string) => {
    try {
      await fetch(addTenantParam(`/api/admin/resilience/bulkheads/${id}`), {
        method: 'DELETE',
      })
      fetchBulkheads()
    } catch (error) {
      console.error('Failed to delete bulkhead:', error)
    }
  }

  const handleCreateCompositePattern = async () => {
    try {
      await fetch(addTenantParam('/api/admin/resilience/composite-patterns'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: cpName,
          description: cpDescription,
          patterns: cpPatterns,
          services: cpServices,
        }),
      })
      setCpDialogOpen(false)
      resetCpForm()
      fetchCompositePatterns()
    } catch (error) {
      console.error('Failed to create composite pattern:', error)
    }
  }

  const handleToggleCompositePattern = async (id: string, enabled: boolean) => {
    try {
      await fetch(addTenantParam(`/api/admin/resilience/composite-patterns/${id}/toggle`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled }),
      })
      fetchCompositePatterns()
    } catch (error) {
      console.error('Failed to toggle composite pattern:', error)
    }
  }

  const handleDeleteCompositePattern = async (id: string) => {
    try {
      await fetch(addTenantParam(`/api/admin/resilience/composite-patterns/${id}`), {
        method: 'DELETE',
      })
      fetchCompositePatterns()
    } catch (error) {
      console.error('Failed to delete composite pattern:', error)
    }
  }

  const resetCbForm = () => {
    setCbName('')
    setCbService('')
    setCbFailureThreshold(5)
    setCbSuccessThreshold(2)
    setCbTimeout(60000)
  }

  const resetRlForm = () => {
    setRlName('')
    setRlResource('')
    setRlAlgorithm('token-bucket')
    setRlLimit(100)
    setRlWindow(60)
    setRlBurst(120)
  }

  const resetRpForm = () => {
    setRpName('')
    setRpOperation('')
    setRpStrategy('exponential')
    setRpMaxAttempts(3)
    setRpBaseDelay(1000)
    setRpMaxDelay(30000)
    setRpJitter(true)
  }

  const resetBhForm = () => {
    setBhName('')
    setBhService('')
    setBhMaxConcurrency(10)
    setBhMaxQueueSize(20)
    setBhTimeout(5000)
  }

  const resetCpForm = () => {
    setCpName('')
    setCpDescription('')
    setCpPatterns([])
    setCpServices([])
  }

  const getStateIcon = (state: string) => {
    switch (state) {
      case 'closed':
        return <CheckmarkCircle24Filled style={{ color: tokens.colorPaletteGreenForeground1 }} />
      case 'open':
        return <DismissCircle24Filled style={{ color: tokens.colorPaletteRedForeground1 }} />
      case 'half-open':
        return <Warning24Filled style={{ color: tokens.colorPaletteYellowForeground1 }} />
      default:
        return <CircleHalfFill24Regular />
    }
  }

  const getStateBadge = (state: string) => {
    const colorMap: Record<string, 'success' | 'danger' | 'warning'> = {
      closed: 'success',
      open: 'danger',
      'half-open': 'warning',
    }
    return <Badge appearance="filled" color={colorMap[state]}>{state.toUpperCase()}</Badge>
  }

  const getAlgorithmBadge = (algorithm: string) => {
    return <Badge appearance="tint">{algorithm}</Badge>
  }

  const getStrategyBadge = (strategy: string) => {
    return <Badge appearance="tint">{strategy}</Badge>
  }

  // Calculate overview metrics
  const totalCircuitBreakers = circuitBreakers.length
  const openCircuits = circuitBreakers.filter(cb => cb.state === 'open').length
  const avgFailureRate = circuitBreakers.length > 0
    ? circuitBreakers.reduce((sum, cb) => sum + cb.failureRate, 0) / circuitBreakers.length
    : 0

  const totalRateLimiters = rateLimiters.length
  const activeRateLimiters = rateLimiters.filter(rl => rl.current > 0).length
  const totalThrottled = rateLimiters.reduce((sum, rl) => sum + rl.throttled, 0)

  const totalRetryPolicies = retryPolicies.length
  const totalRetries = retryPolicies.reduce((sum, rp) => sum + rp.totalRetries, 0)
  const retrySuccessRate = totalRetries > 0
    ? (retryPolicies.reduce((sum, rp) => sum + rp.successfulRetries, 0) / totalRetries) * 100
    : 0

  const totalBulkheads = bulkheads.length
  const avgConcurrency = bulkheads.length > 0
    ? bulkheads.reduce((sum, bh) => sum + (bh.currentConcurrency / bh.maxConcurrency) * 100, 0) / bulkheads.length
    : 0

  if (loading) {
    return (
      <div className={styles.container}>
        <Spinner size="large" label="Loading resilience patterns..." />
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div>
          <Text size={900} weight="semibold">Resilience Patterns</Text>
          <Text size={300}>Configure circuit breakers, rate limiters, retry policies, and bulkheads</Text>
        </div>
      </div>

      {/* Overview Metrics */}
      <div className={styles.metricsGrid}>
        <Card className={styles.metricCard}>
          <div className={styles.metricHeader}>
            <ShieldCheckmark24Regular />
            <Text weight="semibold">Circuit Breakers</Text>
          </div>
          <Text className={styles.metricValue}>{totalCircuitBreakers}</Text>
          <Text size={200}>{openCircuits} Open • {avgFailureRate.toFixed(1)}% Avg Failure Rate</Text>
        </Card>

        <Card className={styles.metricCard}>
          <div className={styles.metricHeader}>
            <Timer24Regular />
            <Text weight="semibold">Rate Limiters</Text>
          </div>
          <Text className={styles.metricValue}>{totalRateLimiters}</Text>
          <Text size={200}>{activeRateLimiters} Active • {totalThrottled} Throttled</Text>
        </Card>

        <Card className={styles.metricCard}>
          <div className={styles.metricHeader}>
            <ArrowRepeatAll24Regular />
            <Text weight="semibold">Retry Policies</Text>
          </div>
          <Text className={styles.metricValue}>{totalRetryPolicies}</Text>
          <Text size={200}>{totalRetries} Total Retries • {retrySuccessRate.toFixed(1)}% Success</Text>
        </Card>

        <Card className={styles.metricCard}>
          <div className={styles.metricHeader}>
            <Grid24Regular />
            <Text weight="semibold">Bulkheads</Text>
          </div>
          <Text className={styles.metricValue}>{totalBulkheads}</Text>
          <Text size={200}>{avgConcurrency.toFixed(1)}% Avg Concurrency</Text>
        </Card>
      </div>

      {/* Tabs */}
      <div className={styles.content}>
        <TabList selectedValue={selectedTab} onTabSelect={(_, data) => setSelectedTab(data.value as string)}>
          <Tab value="circuit-breakers">Circuit Breakers</Tab>
          <Tab value="rate-limiters">Rate Limiters</Tab>
          <Tab value="retry-policies">Retry Policies</Tab>
          <Tab value="bulkheads">Bulkheads</Tab>
          <Tab value="composite">Composite Patterns</Tab>
        </TabList>

        <div className={styles.tabContent}>
          {/* Circuit Breakers Tab */}
          {selectedTab === 'circuit-breakers' && (
            <div>
              <div className={styles.header}>
                <Text size={600} weight="semibold">Circuit Breakers</Text>
                <Dialog open={cbDialogOpen} onOpenChange={(_, data) => setCbDialogOpen(data.open)}>
                  <DialogTrigger>
                    <Button appearance="primary" icon={<Add24Regular />}>Add Circuit Breaker</Button>
                  </DialogTrigger>
                  <DialogSurface>
                    <DialogBody>
                      <DialogTitle>{editingCb ? 'Edit' : 'Create'} Circuit Breaker</DialogTitle>
                      <DialogContent>
                        <div className={styles.formField}>
                          <Label required>Name</Label>
                          <Input value={cbName} onChange={(_, data) => setCbName(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label required>Service</Label>
                          <Input value={cbService} onChange={(_, data) => setCbService(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label>Failure Threshold</Label>
                          <Slider
                            min={1}
                            max={20}
                            value={cbFailureThreshold}
                            onChange={(_, data) => setCbFailureThreshold(data.value)}
                          />
                          <Text size={200}>{cbFailureThreshold} failures</Text>
                        </div>
                        <div className={styles.formField}>
                          <Label>Success Threshold</Label>
                          <Slider
                            min={1}
                            max={10}
                            value={cbSuccessThreshold}
                            onChange={(_, data) => setCbSuccessThreshold(data.value)}
                          />
                          <Text size={200}>{cbSuccessThreshold} successes</Text>
                        </div>
                        <div className={styles.formField}>
                          <Label>Timeout (ms)</Label>
                          <Input
                            type="number"
                            value={cbTimeout.toString()}
                            onChange={(_, data) => setCbTimeout(parseInt(data.value) || 60000)}
                          />
                        </div>
                      </DialogContent>
                      <DialogActions>
                        <Button appearance="secondary" onClick={() => {
                          setCbDialogOpen(false)
                          setEditingCb(null)
                          resetCbForm()
                        }}>Cancel</Button>
                        <Button
                          appearance="primary"
                          onClick={editingCb ? handleUpdateCircuitBreaker : handleCreateCircuitBreaker}
                          disabled={!cbName || !cbService}
                        >
                          {editingCb ? 'Update' : 'Create'}
                        </Button>
                      </DialogActions>
                    </DialogBody>
                  </DialogSurface>
                </Dialog>
              </div>

              <div className={styles.circuitBreakerGrid}>
                {circuitBreakers.map((cb) => (
                  <Card key={cb.id} className={styles.circuitCard}>
                    <div className={styles.circuitHeader}>
                      <div className={styles.circuitInfo}>
                        <Text weight="semibold" size={500}>{cb.name}</Text>
                        <Text size={200}>{cb.service}</Text>
                        <div className={styles.stateIndicator} style={{ marginTop: tokens.spacingVerticalS }}>
                          {getStateIcon(cb.state)}
                          {getStateBadge(cb.state)}
                        </div>
                      </div>
                      <div className={styles.circuitActions}>
                        <Button
                          size="small"
                          icon={<Edit24Regular />}
                          onClick={() => {
                            setEditingCb(cb)
                            setCbName(cb.name)
                            setCbService(cb.service)
                            setCbFailureThreshold(cb.failureThreshold)
                            setCbSuccessThreshold(cb.successThreshold)
                            setCbTimeout(cb.timeout)
                            setCbDialogOpen(true)
                          }}
                        />
                        <Button
                          size="small"
                          icon={<Delete24Regular />}
                          onClick={() => handleDeleteCircuitBreaker(cb.id)}
                        />
                      </div>
                    </div>

                    {cb.state === 'open' && (
                      <MessageBar intent="error">
                        <MessageBarBody>
                          Circuit is open! Requests are failing fast.
                        </MessageBarBody>
                      </MessageBar>
                    )}

                    <div className={styles.metrics}>
                      <div className={styles.metricRow}>
                        <Text size={200}>Failures:</Text>
                        <Text weight="semibold">{cb.failures}/{cb.failureThreshold}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Successes:</Text>
                        <Text weight="semibold">{cb.successes}/{cb.successThreshold}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Total Requests:</Text>
                        <Text weight="semibold">{cb.totalRequests}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Failure Rate:</Text>
                        <Text weight="semibold">{cb.failureRate.toFixed(1)}%</Text>
                      </div>
                    </div>

                    <Button
                      appearance="secondary"
                      size="small"
                      style={{ marginTop: tokens.spacingVerticalM }}
                      onClick={() => handleResetCircuitBreaker(cb.id)}
                    >
                      Reset Circuit
                    </Button>
                  </Card>
                ))}
              </div>
            </div>
          )}

          {/* Rate Limiters Tab */}
          {selectedTab === 'rate-limiters' && (
            <div>
              <div className={styles.header}>
                <Text size={600} weight="semibold">Rate Limiters</Text>
                <Dialog open={rlDialogOpen} onOpenChange={(_, data) => setRlDialogOpen(data.open)}>
                  <DialogTrigger>
                    <Button appearance="primary" icon={<Add24Regular />}>Add Rate Limiter</Button>
                  </DialogTrigger>
                  <DialogSurface>
                    <DialogBody>
                      <DialogTitle>Create Rate Limiter</DialogTitle>
                      <DialogContent>
                        <div className={styles.formField}>
                          <Label required>Name</Label>
                          <Input value={rlName} onChange={(_, data) => setRlName(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label required>Resource</Label>
                          <Input value={rlResource} onChange={(_, data) => setRlResource(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label>Algorithm</Label>
                          <Dropdown
                            value={rlAlgorithm}
                            onOptionSelect={(_, data) => setRlAlgorithm(data.optionValue || 'token-bucket')}
                          >
                            <Option value="token-bucket">Token Bucket</Option>
                            <Option value="leaky-bucket">Leaky Bucket</Option>
                            <Option value="fixed-window">Fixed Window</Option>
                            <Option value="sliding-window">Sliding Window</Option>
                          </Dropdown>
                        </div>
                        <div className={styles.formField}>
                          <Label>Limit (requests)</Label>
                          <Input
                            type="number"
                            value={rlLimit.toString()}
                            onChange={(_, data) => setRlLimit(parseInt(data.value) || 100)}
                          />
                        </div>
                        <div className={styles.formField}>
                          <Label>Window (seconds)</Label>
                          <Input
                            type="number"
                            value={rlWindow.toString()}
                            onChange={(_, data) => setRlWindow(parseInt(data.value) || 60)}
                          />
                        </div>
                        <div className={styles.formField}>
                          <Label>Burst Capacity</Label>
                          <Input
                            type="number"
                            value={rlBurst.toString()}
                            onChange={(_, data) => setRlBurst(parseInt(data.value) || 120)}
                          />
                        </div>
                      </DialogContent>
                      <DialogActions>
                        <Button appearance="secondary" onClick={() => {
                          setRlDialogOpen(false)
                          resetRlForm()
                        }}>Cancel</Button>
                        <Button
                          appearance="primary"
                          onClick={handleCreateRateLimiter}
                          disabled={!rlName || !rlResource}
                        >
                          Create
                        </Button>
                      </DialogActions>
                    </DialogBody>
                  </DialogSurface>
                </Dialog>
              </div>

              <div className={styles.rateLimiterList}>
                {rateLimiters.map((rl) => (
                  <Card key={rl.id} className={styles.rateLimiterCard}>
                    <div className={styles.rateLimiterHeader}>
                      <div>
                        <Text weight="semibold" size={500}>{rl.name}</Text>
                        <Text size={200}>{rl.resource}</Text>
                        <div style={{ marginTop: tokens.spacingVerticalXS }}>
                          {getAlgorithmBadge(rl.algorithm)}
                        </div>
                      </div>
                      <Button
                        size="small"
                        icon={<Delete24Regular />}
                        onClick={() => handleDeleteRateLimiter(rl.id)}
                      />
                    </div>

                    <div className={styles.progressSection}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: tokens.spacingVerticalXS }}>
                        <Text size={200}>Current Usage</Text>
                        <Text size={200}>{rl.current}/{rl.limit} requests</Text>
                      </div>
                      <ProgressBar value={rl.current / rl.limit} max={1} />
                    </div>

                    <div className={styles.rateLimiterMetrics}>
                      <div>
                        <Text size={200}>Window</Text>
                        <Text weight="semibold">{rl.window}s</Text>
                      </div>
                      <div>
                        <Text size={200}>Throttled</Text>
                        <Text weight="semibold">{rl.throttled}</Text>
                      </div>
                      <div>
                        <Text size={200}>Total Requests</Text>
                        <Text weight="semibold">{rl.totalRequests}</Text>
                      </div>
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          )}

          {/* Retry Policies Tab */}
          {selectedTab === 'retry-policies' && (
            <div>
              <div className={styles.header}>
                <Text size={600} weight="semibold">Retry Policies</Text>
                <Dialog open={rpDialogOpen} onOpenChange={(_, data) => setRpDialogOpen(data.open)}>
                  <DialogTrigger>
                    <Button appearance="primary" icon={<Add24Regular />}>Add Retry Policy</Button>
                  </DialogTrigger>
                  <DialogSurface>
                    <DialogBody>
                      <DialogTitle>Create Retry Policy</DialogTitle>
                      <DialogContent>
                        <div className={styles.formField}>
                          <Label required>Name</Label>
                          <Input value={rpName} onChange={(_, data) => setRpName(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label required>Operation</Label>
                          <Input value={rpOperation} onChange={(_, data) => setRpOperation(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label>Strategy</Label>
                          <Dropdown
                            value={rpStrategy}
                            onOptionSelect={(_, data) => setRpStrategy(data.optionValue || 'exponential')}
                          >
                            <Option value="fixed">Fixed</Option>
                            <Option value="exponential">Exponential</Option>
                            <Option value="fibonacci">Fibonacci</Option>
                            <Option value="linear">Linear</Option>
                          </Dropdown>
                        </div>
                        <div className={styles.formField}>
                          <Label>Max Attempts</Label>
                          <Slider
                            min={1}
                            max={10}
                            value={rpMaxAttempts}
                            onChange={(_, data) => setRpMaxAttempts(data.value)}
                          />
                          <Text size={200}>{rpMaxAttempts} attempts</Text>
                        </div>
                        <div className={styles.formField}>
                          <Label>Base Delay (ms)</Label>
                          <Input
                            type="number"
                            value={rpBaseDelay.toString()}
                            onChange={(_, data) => setRpBaseDelay(parseInt(data.value) || 1000)}
                          />
                        </div>
                        <div className={styles.formField}>
                          <Label>Max Delay (ms)</Label>
                          <Input
                            type="number"
                            value={rpMaxDelay.toString()}
                            onChange={(_, data) => setRpMaxDelay(parseInt(data.value) || 30000)}
                          />
                        </div>
                        <div className={styles.formField}>
                          <Label>Enable Jitter</Label>
                          <Switch checked={rpJitter} onChange={(_, data) => setRpJitter(data.checked)} />
                        </div>
                      </DialogContent>
                      <DialogActions>
                        <Button appearance="secondary" onClick={() => {
                          setRpDialogOpen(false)
                          resetRpForm()
                        }}>Cancel</Button>
                        <Button
                          appearance="primary"
                          onClick={handleCreateRetryPolicy}
                          disabled={!rpName || !rpOperation}
                        >
                          Create
                        </Button>
                      </DialogActions>
                    </DialogBody>
                  </DialogSurface>
                </Dialog>
              </div>

              <div className={styles.retryPolicyGrid}>
                {retryPolicies.map((rp) => (
                  <Card key={rp.id} className={styles.policyCard}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: tokens.spacingVerticalM }}>
                      <div>
                        <Text weight="semibold" size={500}>{rp.name}</Text>
                        <Text size={200}>{rp.operation}</Text>
                        <div style={{ marginTop: tokens.spacingVerticalXS }}>
                          {getStrategyBadge(rp.strategy)}
                        </div>
                      </div>
                      <Button
                        size="small"
                        icon={<Delete24Regular />}
                        onClick={() => handleDeleteRetryPolicy(rp.id)}
                      />
                    </div>

                    <div className={styles.metrics}>
                      <div className={styles.metricRow}>
                        <Text size={200}>Max Attempts:</Text>
                        <Text weight="semibold">{rp.maxAttempts}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Base Delay:</Text>
                        <Text weight="semibold">{rp.baseDelay}ms</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Total Retries:</Text>
                        <Text weight="semibold">{rp.totalRetries}</Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Successful:</Text>
                        <Text weight="semibold" style={{ color: tokens.colorPaletteGreenForeground1 }}>
                          {rp.successfulRetries}
                        </Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Failed:</Text>
                        <Text weight="semibold" style={{ color: tokens.colorPaletteRedForeground1 }}>
                          {rp.failedRetries}
                        </Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Jitter:</Text>
                        <Text weight="semibold">{rp.jitter ? 'Enabled' : 'Disabled'}</Text>
                      </div>
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          )}

          {/* Bulkheads Tab */}
          {selectedTab === 'bulkheads' && (
            <div>
              <div className={styles.header}>
                <Text size={600} weight="semibold">Bulkheads</Text>
                <Dialog open={bhDialogOpen} onOpenChange={(_, data) => setBhDialogOpen(data.open)}>
                  <DialogTrigger>
                    <Button appearance="primary" icon={<Add24Regular />}>Add Bulkhead</Button>
                  </DialogTrigger>
                  <DialogSurface>
                    <DialogBody>
                      <DialogTitle>Create Bulkhead</DialogTitle>
                      <DialogContent>
                        <div className={styles.formField}>
                          <Label required>Name</Label>
                          <Input value={bhName} onChange={(_, data) => setBhName(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label required>Service</Label>
                          <Input value={bhService} onChange={(_, data) => setBhService(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label>Max Concurrency</Label>
                          <Slider
                            min={1}
                            max={100}
                            value={bhMaxConcurrency}
                            onChange={(_, data) => setBhMaxConcurrency(data.value)}
                          />
                          <Text size={200}>{bhMaxConcurrency} concurrent requests</Text>
                        </div>
                        <div className={styles.formField}>
                          <Label>Max Queue Size</Label>
                          <Slider
                            min={0}
                            max={200}
                            value={bhMaxQueueSize}
                            onChange={(_, data) => setBhMaxQueueSize(data.value)}
                          />
                          <Text size={200}>{bhMaxQueueSize} queued requests</Text>
                        </div>
                        <div className={styles.formField}>
                          <Label>Timeout (ms)</Label>
                          <Input
                            type="number"
                            value={bhTimeout.toString()}
                            onChange={(_, data) => setBhTimeout(parseInt(data.value) || 5000)}
                          />
                        </div>
                      </DialogContent>
                      <DialogActions>
                        <Button appearance="secondary" onClick={() => {
                          setBhDialogOpen(false)
                          resetBhForm()
                        }}>Cancel</Button>
                        <Button
                          appearance="primary"
                          onClick={handleCreateBulkhead}
                          disabled={!bhName || !bhService}
                        >
                          Create
                        </Button>
                      </DialogActions>
                    </DialogBody>
                  </DialogSurface>
                </Dialog>
              </div>

              <div className={styles.bulkheadGrid}>
                {bulkheads.map((bh) => (
                  <Card key={bh.id} className={styles.bulkheadCard}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: tokens.spacingVerticalM }}>
                      <div>
                        <Text weight="semibold" size={500}>{bh.name}</Text>
                        <Text size={200}>{bh.service}</Text>
                      </div>
                      <Button
                        size="small"
                        icon={<Delete24Regular />}
                        onClick={() => handleDeleteBulkhead(bh.id)}
                      />
                    </div>

                    <div className={styles.progressSection}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: tokens.spacingVerticalXS }}>
                        <Text size={200}>Concurrency</Text>
                        <Text size={200}>{bh.currentConcurrency}/{bh.maxConcurrency}</Text>
                      </div>
                      <ProgressBar value={bh.currentConcurrency / bh.maxConcurrency} max={1} />
                    </div>

                    <div className={styles.progressSection}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: tokens.spacingVerticalXS }}>
                        <Text size={200}>Queue</Text>
                        <Text size={200}>{bh.queuedRequests}/{bh.maxQueueSize}</Text>
                      </div>
                      <ProgressBar value={bh.queuedRequests / bh.maxQueueSize} max={1} />
                    </div>

                    <div className={styles.metrics}>
                      <div className={styles.metricRow}>
                        <Text size={200}>Rejected:</Text>
                        <Text weight="semibold" style={{ color: tokens.colorPaletteRedForeground1 }}>
                          {bh.rejectedRequests}
                        </Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Completed:</Text>
                        <Text weight="semibold" style={{ color: tokens.colorPaletteGreenForeground1 }}>
                          {bh.completedRequests}
                        </Text>
                      </div>
                      <div className={styles.metricRow}>
                        <Text size={200}>Timeout:</Text>
                        <Text weight="semibold">{bh.timeout}ms</Text>
                      </div>
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          )}

          {/* Composite Patterns Tab */}
          {selectedTab === 'composite' && (
            <div>
              <div className={styles.header}>
                <Text size={600} weight="semibold">Composite Patterns</Text>
                <Dialog open={cpDialogOpen} onOpenChange={(_, data) => setCpDialogOpen(data.open)}>
                  <DialogTrigger>
                    <Button appearance="primary" icon={<Add24Regular />}>Create Composite Pattern</Button>
                  </DialogTrigger>
                  <DialogSurface>
                    <DialogBody>
                      <DialogTitle>Create Composite Pattern</DialogTitle>
                      <DialogContent>
                        <div className={styles.formField}>
                          <Label required>Name</Label>
                          <Input value={cpName} onChange={(_, data) => setCpName(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label>Description</Label>
                          <Input value={cpDescription} onChange={(_, data) => setCpDescription(data.value)} />
                        </div>
                        <div className={styles.formField}>
                          <Label>Patterns (comma-separated)</Label>
                          <Input
                            placeholder="circuit-breaker, rate-limiter, retry"
                            value={cpPatterns.join(', ')}
                            onChange={(_, data) => setCpPatterns(data.value.split(',').map(s => s.trim()).filter(Boolean))}
                          />
                        </div>
                        <div className={styles.formField}>
                          <Label>Services (comma-separated)</Label>
                          <Input
                            placeholder="auth-service, token-service"
                            value={cpServices.join(', ')}
                            onChange={(_, data) => setCpServices(data.value.split(',').map(s => s.trim()).filter(Boolean))}
                          />
                        </div>
                      </DialogContent>
                      <DialogActions>
                        <Button appearance="secondary" onClick={() => {
                          setCpDialogOpen(false)
                          resetCpForm()
                        }}>Cancel</Button>
                        <Button
                          appearance="primary"
                          onClick={handleCreateCompositePattern}
                          disabled={!cpName || cpPatterns.length === 0}
                        >
                          Create
                        </Button>
                      </DialogActions>
                    </DialogBody>
                  </DialogSurface>
                </Dialog>
              </div>

              <div className={styles.compositeList}>
                {compositePatterns.map((cp) => (
                  <Card key={cp.id} className={styles.compositeCard}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <div style={{ flex: 1 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: tokens.spacingHorizontalM, marginBottom: tokens.spacingVerticalS }}>
                          <Text weight="semibold" size={500}>{cp.name}</Text>
                          <Switch
                            checked={cp.enabled}
                            onChange={(_, data) => handleToggleCompositePattern(cp.id, data.checked)}
                          />
                        </div>
                        <Text size={200}>{cp.description}</Text>
                        
                        <div style={{ marginTop: tokens.spacingVerticalM }}>
                          <Text size={200} weight="semibold">Patterns:</Text>
                          <div className={styles.patternsList}>
                            {cp.patterns.map((pattern, idx) => (
                              <Badge key={idx} appearance="tint">{pattern}</Badge>
                            ))}
                          </div>
                        </div>

                        <div style={{ marginTop: tokens.spacingVerticalM }}>
                          <Text size={200} weight="semibold">Services:</Text>
                          <div className={styles.patternsList}>
                            {cp.services.map((service, idx) => (
                              <Badge key={idx} appearance="outline">{service}</Badge>
                            ))}
                          </div>
                        </div>

                        <div style={{ marginTop: tokens.spacingVerticalM }}>
                          <Text size={200}>Applied {cp.appliedCount} times</Text>
                        </div>
                      </div>
                      <Button
                        size="small"
                        icon={<Delete24Regular />}
                        onClick={() => handleDeleteCompositePattern(cp.id)}
                      />
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
