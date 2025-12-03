import { useEffect, useState } from 'react';
import {
  makeStyles,
  tokens,
  Card,
  Text,
  Title3,
  Badge,
  Spinner,
  ProgressBar,
} from '@fluentui/react-components';
import {
  CheckmarkCircle24Regular,
  ErrorCircle24Regular,
  Warning24Regular,
  Gauge24Regular,
  Clock24Regular,
  DocumentDatabase24Regular,
} from '@fluentui/react-icons';
import { Line, LineChart, ResponsiveContainer, Tooltip as RechartsTooltip, XAxis, YAxis } from 'recharts';
import TokenViolationTable from '../../components/metrics/TokenViolationTable';
import SemanticCounters from '../../components/metrics/SemanticCounters';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '24px',
  },
  section: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  sectionTitle: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    marginBottom: '8px',
  },
  cardsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
    gap: '16px',
  },
  card: {
    padding: '20px',
  },
  cardHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: '16px',
  },
  cardTitle: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '14px',
    fontWeight: 600,
    color: tokens.colorNeutralForeground2,
  },
  cardValue: {
    fontSize: '32px',
    fontWeight: 600,
    marginBottom: '8px',
  },
  cardSubtext: {
    fontSize: '13px',
    color: tokens.colorNeutralForeground3,
  },
  statusGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
    gap: '16px',
  },
  statusCard: {
    padding: '16px',
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  statusIcon: {
    fontSize: '32px',
  },
  statusContent: {
    flex: 1,
  },
  metricsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    gap: '12px',
  },
  metricItem: {
    display: 'flex',
    flexDirection: 'column',
    gap: '4px',
  },
  metricLabel: {
    fontSize: '12px',
    color: tokens.colorNeutralForeground3,
  },
  metricValue: {
    fontSize: '20px',
    fontWeight: 600,
  },
  chartContainer: {
    height: '200px',
    marginTop: '16px',
  },
  twoColumnGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
    gap: '16px',
  },
});

interface ComponentHealth {
  name: string;
  status: 'healthy' | 'degraded' | 'down';
  uptime: string;
  requests: number;
}

interface PerformanceMetric {
  timestamp: string;
  requests: number;
  latency: number;
  errors: number;
}

interface SystemMetrics {
  totalRequests: number;
  avgLatency: number;
  p95Latency: number;
  p99Latency: number;
  errorCount: number;
  errorRate: number;
  cacheHitRate: number;
  cacheSize: number;
  memoryUsage: number;
  cacheEvictions: number;
  avgTTL: number;
  compressionRatio: number;
}

export default function SystemMetrics() {
  const classes = useStyles();
  const [loading, setLoading] = useState(true);
  const [componentHealth, setComponentHealth] = useState<ComponentHealth[]>([]);
  const [performanceData, setPerformanceData] = useState<PerformanceMetric[]>([]);
  const [systemMetrics, setSystemMetrics] = useState<SystemMetrics>({
    totalRequests: 0,
    avgLatency: 0,
    p95Latency: 0,
    p99Latency: 0,
    errorCount: 0,
    errorRate: 0,
    cacheHitRate: 0,
    cacheSize: 0,
    memoryUsage: 0,
    cacheEvictions: 0,
    avgTTL: 0,
    compressionRatio: 0,
  });

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchMetrics = async () => {
    try {
      // Fetch system metrics from backend
      const response = await fetch('/api/admin/metrics/system?tenant_id=test-tenant-1', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        // Backend returns flattened structure with metrics at top level
        setSystemMetrics({
          totalRequests: data.totalRequests || 0,
          avgLatency: data.avgLatency || 0,
          p95Latency: data.p95Latency || 0,
          p99Latency: data.p99Latency || 0,
          errorCount: data.errorCount || 0,
          errorRate: data.errorRate || 0,
          cacheHitRate: data.cacheHitRate || 0,
          cacheSize: data.cacheSize || 0,
          memoryUsage: data.memoryUsage || 0,
          cacheEvictions: data.cacheEvictions || 0,
          avgTTL: data.avgTTL || 0,
          compressionRatio: data.compressionRatio || 0,
        });
        setComponentHealth(data.componentHealth || []);
        setPerformanceData(data.performanceHistory || []);
      }
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckmarkCircle24Regular style={{ color: tokens.colorPaletteGreenForeground1 }} />;
      case 'degraded':
        return <Warning24Regular style={{ color: tokens.colorPaletteYellowForeground1 }} />;
      case 'down':
        return <ErrorCircle24Regular style={{ color: tokens.colorPaletteRedForeground1 }} />;
      default:
        return <CheckmarkCircle24Regular />;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'healthy':
        return <Badge appearance="filled" color="success">Healthy</Badge>;
      case 'degraded':
        return <Badge appearance="filled" color="warning">Degraded</Badge>;
      case 'down':
        return <Badge appearance="filled" color="danger">Down</Badge>;
      default:
        return <Badge appearance="filled">Unknown</Badge>;
    }
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '40px' }}>
        <Spinner size="large" label="Loading metrics..." />
      </div>
    );
  }

  return (
    <div className={classes.container}>
      {/* System Health */}
      <div className={classes.section}>
        <div className={classes.sectionTitle}>
          <Gauge24Regular style={{ fontSize: '24px' }} />
          <Title3>System Health</Title3>
        </div>
        <div className={classes.statusGrid}>
          {componentHealth.map((component) => (
            <Card key={component.name} className={classes.statusCard}>
              <div className={classes.statusIcon}>
                {getStatusIcon(component.status)}
              </div>
              <div className={classes.statusContent}>
                <Text weight="semibold">{component.name}</Text>
                <div style={{ marginTop: '4px' }}>
                  {getStatusBadge(component.status)}
                </div>
                <Text size={200} style={{ marginTop: '8px', display: 'block' }}>
                  Uptime: {component.uptime} • {component.requests.toLocaleString()} req
                </Text>
              </div>
            </Card>
          ))}
        </div>
      </div>

      {/* Performance Overview */}
      <div className={classes.section}>
        <div className={classes.sectionTitle}>
          <Clock24Regular style={{ fontSize: '24px' }} />
          <Title3>Performance Overview</Title3>
        </div>
        <Card className={classes.card}>
          <div className={classes.metricsGrid}>
            <div className={classes.metricItem}>
              <Text className={classes.metricLabel}>Total Requests</Text>
              <Text className={classes.metricValue}>
                {systemMetrics.totalRequests.toLocaleString()}
              </Text>
            </div>
            <div className={classes.metricItem}>
              <Text className={classes.metricLabel}>Avg Latency</Text>
              <Text className={classes.metricValue}>
                {systemMetrics.avgLatency.toFixed(2)}ms
              </Text>
            </div>
            <div className={classes.metricItem}>
              <Text className={classes.metricLabel}>P95 Latency</Text>
              <Text className={classes.metricValue}>
                {systemMetrics.p95Latency.toFixed(2)}ms
              </Text>
            </div>
            <div className={classes.metricItem}>
              <Text className={classes.metricLabel}>P99 Latency</Text>
              <Text className={classes.metricValue}>
                {systemMetrics.p99Latency.toFixed(2)}ms
              </Text>
            </div>
            <div className={classes.metricItem}>
              <Text className={classes.metricLabel}>Error Count</Text>
              <Text className={classes.metricValue} style={{ color: tokens.colorPaletteRedForeground1 }}>
                {systemMetrics.errorCount.toLocaleString()}
              </Text>
            </div>
            <div className={classes.metricItem}>
              <Text className={classes.metricLabel}>Error Rate</Text>
              <Text className={classes.metricValue}>
                {(systemMetrics.errorRate * 100).toFixed(2)}%
              </Text>
            </div>
          </div>
          
          {performanceData.length > 0 && (
            <div className={classes.chartContainer}>
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={performanceData}>
                  <XAxis dataKey="timestamp" stroke={tokens.colorNeutralForeground3} />
                  <YAxis stroke={tokens.colorNeutralForeground3} />
                  <RechartsTooltip />
                  <Line
                    type="monotone"
                    dataKey="latency"
                    stroke={tokens.colorBrandForeground1}
                    strokeWidth={2}
                    dot={false}
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </Card>
      </div>

      {/* Cache Statistics */}
      <div className={classes.section}>
        <div className={classes.sectionTitle}>
          <DocumentDatabase24Regular style={{ fontSize: '24px' }} />
          <Title3>Cache Statistics</Title3>
        </div>
        <div className={classes.cardsGrid}>
          <Card className={classes.card}>
            <div className={classes.cardHeader}>
              <Text className={classes.cardTitle}>Hit Rate</Text>
            </div>
            <Text className={classes.cardValue}>
              {(systemMetrics.cacheHitRate * 100).toFixed(1)}%
            </Text>
            <ProgressBar value={systemMetrics.cacheHitRate} max={1} />
            <Text className={classes.cardSubtext}>
              Higher is better
            </Text>
          </Card>
          
          <Card className={classes.card}>
            <div className={classes.cardHeader}>
              <Text className={classes.cardTitle}>Cache Size</Text>
            </div>
            <Text className={classes.cardValue}>
              {(systemMetrics.cacheSize / 1024 / 1024).toFixed(1)} MB
            </Text>
            <Text className={classes.cardSubtext}>
              Memory: {(systemMetrics.memoryUsage / 1024 / 1024).toFixed(1)} MB
            </Text>
          </Card>
          
          <Card className={classes.card}>
            <div className={classes.cardHeader}>
              <Text className={classes.cardTitle}>Evictions</Text>
            </div>
            <Text className={classes.cardValue}>
              {systemMetrics.cacheEvictions.toLocaleString()}
            </Text>
            <Text className={classes.cardSubtext}>
              Avg TTL: {systemMetrics.avgTTL.toFixed(0)}s
            </Text>
          </Card>
          
          <Card className={classes.card}>
            <div className={classes.cardHeader}>
              <Text className={classes.cardTitle}>Compression</Text>
            </div>
            <Text className={classes.cardValue}>
              {systemMetrics.compressionRatio.toFixed(2)}x
            </Text>
            <Text className={classes.cardSubtext}>
              Space saved
            </Text>
          </Card>
        </div>
      </div>

      {/* Component Benchmarks */}
      <div className={classes.section}>
        <div className={classes.sectionTitle}>
          <Title3>Component Benchmarks</Title3>
        </div>
        <div className={classes.twoColumnGrid}>
          <Card className={classes.card}>
            <Text weight="semibold" style={{ marginBottom: '12px' }}>
              PVP Identity Chain
            </Text>
            <div className={classes.metricsGrid}>
              <div className={classes.metricItem}>
                <Text className={classes.metricLabel}>Avg Time</Text>
                <Text className={classes.metricValue}>125ms</Text>
              </div>
              <div className={classes.metricItem}>
                <Text className={classes.metricLabel}>Success Rate</Text>
                <Text className={classes.metricValue}>99.8%</Text>
              </div>
            </div>
          </Card>
          
          <Card className={classes.card}>
            <Text weight="semibold" style={{ marginBottom: '12px' }}>
              Registry Verification
            </Text>
            <div className={classes.metricsGrid}>
              <div className={classes.metricItem}>
                <Text className={classes.metricLabel}>Avg Time</Text>
                <Text className={classes.metricValue}>85ms</Text>
              </div>
              <div className={classes.metricItem}>
                <Text className={classes.metricLabel}>Success Rate</Text>
                <Text className={classes.metricValue}>99.9%</Text>
              </div>
            </div>
          </Card>
          
          <Card className={classes.card}>
            <Text weight="semibold" style={{ marginBottom: '12px' }}>
              PIP Authorization
            </Text>
            <div className={classes.metricsGrid}>
              <div className={classes.metricItem}>
                <Text className={classes.metricLabel}>Avg Time</Text>
                <Text className={classes.metricValue}>45ms</Text>
              </div>
              <div className={classes.metricItem}>
                <Text className={classes.metricLabel}>Success Rate</Text>
                <Text className={classes.metricValue}>99.7%</Text>
              </div>
            </div>
          </Card>
          
          <Card className={classes.card}>
            <Text weight="semibold" style={{ marginBottom: '12px' }}>
              PoA Validation
            </Text>
            <div className={classes.metricsGrid}>
              <div className={classes.metricItem}>
                <Text className={classes.metricLabel}>Avg Time</Text>
                <Text className={classes.metricValue}>95ms</Text>
              </div>
              <div className={classes.metricItem}>
                <Text className={classes.metricLabel}>Success Rate</Text>
                <Text className={classes.metricValue}>99.6%</Text>
              </div>
            </div>
          </Card>
        </div>
      </div>

      {/* Token Violation Metrics */}
      <div className={classes.section}>
        <TokenViolationTable />
      </div>

      {/* Semantic Counters */}
      <div className={classes.section}>
        <SemanticCounters />
      </div>
    </div>
  );
}
