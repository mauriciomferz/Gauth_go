import { useEffect, useState } from 'react';
import {
  makeStyles,
  tokens,
  Card,
  Text,
  Title3,
  Title2,
  Badge,
} from '@fluentui/react-components';
import {
  CheckmarkCircle24Regular,
  Warning24Regular,
  People24Regular,
  Shield24Regular,
  Gauge24Regular,
  DocumentBulletList24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '24px',
  },
  welcomeCard: {
    padding: '32px',
    background: `linear-gradient(135deg, ${tokens.colorBrandBackground} 0%, ${tokens.colorBrandBackground2} 100%)`,
  },
  welcomeTitle: {
    color: tokens.colorNeutralForegroundOnBrand,
    marginBottom: '8px',
  },
  welcomeText: {
    color: tokens.colorNeutralForegroundOnBrand,
    fontSize: '16px',
  },
  statsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
    gap: '16px',
  },
  statCard: {
    padding: '24px',
    display: 'flex',
    flexDirection: 'column',
    gap: '12px',
  },
  statHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  statIcon: {
    fontSize: '32px',
    color: tokens.colorBrandForeground1,
  },
  statValue: {
    fontSize: '36px',
    fontWeight: 700,
    color: tokens.colorNeutralForeground1,
  },
  statLabel: {
    fontSize: '14px',
    color: tokens.colorNeutralForeground3,
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
  },
  statusList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '12px',
  },
  statusItem: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '12px 0',
    borderBottom: `1px solid ${tokens.colorNeutralStroke2}`,
  },
  statusInfo: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  quickActions: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    gap: '16px',
  },
  actionCard: {
    padding: '20px',
    cursor: 'pointer',
    transition: 'all 0.2s ease',
    ':hover': {
      transform: 'translateY(-2px)',
      boxShadow: tokens.shadow8,
    },
  },
  actionTitle: {
    fontSize: '16px',
    fontWeight: 600,
    marginTop: '8px',
  },
});

interface DashboardStats {
  activeSubscribers: number;
  totalTokens: number;
  activePolicies: number;
  recentEvents: number;
  systemHealth: string;
  uptime: string;
}

interface SystemStatus {
  name: string;
  status: 'healthy' | 'warning' | 'error';
  message: string;
}

export default function Dashboard() {
  const classes = useStyles();
  const [stats, setStats] = useState<DashboardStats>({
    activeSubscribers: 0,
    totalTokens: 0,
    activePolicies: 0,
    recentEvents: 0,
    systemHealth: 'healthy',
    uptime: '0d 0h',
  });

  const [systemStatus, setSystemStatus] = useState<SystemStatus[]>([
    { name: 'Authorization Service', status: 'healthy', message: 'All systems operational' },
    { name: 'Token Management', status: 'healthy', message: 'Running smoothly' },
    { name: 'Database', status: 'healthy', message: 'Connected and healthy' },
    { name: 'Cache Layer', status: 'healthy', message: '94.3% hit rate' },
  ]);

  useEffect(() => {
    fetchDashboardData();
    const interval = setInterval(fetchDashboardData, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchDashboardData = async () => {
    try {
      const tenantId = 'test-tenant-1'; // TODO: Get from auth context

      // Fetch all data in parallel
      const [metricsResponse, policiesResponse, subscribersResponse] = await Promise.all([
        fetch(`/api/admin/metrics/system?tenant_id=${tenantId}`),
        fetch(`/api/admin/authz/policies?tenant_id=${tenantId}`),
        fetch(`/api/admin/subscribers?tenant_id=${tenantId}`),
      ]);

      if (metricsResponse.ok) {
        const contentType = metricsResponse.headers.get('content-type');
        if (!contentType || !contentType.includes('application/json') {
          console.warn('Dashboard: Metrics endpoint returned non-JSON response');
          return;
        }

        const metricsData = await metricsResponse.json();

        // Get real active policies count from database
        let activePolicies = 0;
        if (policiesResponse.ok) {
          const policyContentType = policiesResponse.headers.get('content-type');
          if (policyContentType && policyContentType.includes('application/json') {
            const policiesData = await policiesResponse.json();
            activePolicies = policiesData.policies?.filter((p: any) => p.status === 'active').length || 0;
          }
        }

        // Get real subscribers count
        let activeSubscribers = 0;
        if (subscribersResponse.ok) {
          const subscriberContentType = subscribersResponse.headers.get('content-type');
          if (subscriberContentType && subscriberContentType.includes('application/json') {
            const subscribersData = await subscribersResponse.json();
            activeSubscribers = subscribersData.subscribers?.filter((s: any) => s.status === 'active').length || 0;
          }
        }

        // Use metrics data for authorization requests count
        const authorizationRequests = metricsData.totalRequests || 0;

        // Calculate system uptime from metrics
        const uptimeSeconds = metricsData.uptime || 0;
        const days = Math.floor(uptimeSeconds / (24 * 60 * 60));
        const hours = Math.floor((uptimeSeconds % (24 * 60 * 60) / (60 * 60));

        setStats({
          activeSubscribers: activeSubscribers,
          totalTokens: authorizationRequests,
          activePolicies: activePolicies,
          recentEvents: metricsData.totalRequests || 0,
          systemHealth: metricsData.componentHealth?.every((c: any) => c.status === 'healthy') ? 'healthy' : 'warning',
          uptime: `${days}d ${hours}h`,
        });

        // Update system status based on real component health
        if (metricsData.componentHealth) {
          const statusMap = metricsData.componentHealth.map((comp: any) => ({
            name: comp.name,
            status: comp.status === 'unhealthy' ? 'error' : comp.status === 'degraded' ? 'warning' : 'healthy',
            message: `Uptime: ${comp.uptime} • ${comp.requests.toLocaleString()} requests`,
          }));
          setSystemStatus(statusMap);
        }
      }
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckmarkCircle24Regular style={{ color: tokens.colorPaletteGreenForeground1 }} />;
      case 'warning':
        return <Warning24Regular style={{ color: tokens.colorPaletteYellowForeground1 }} />;
      default:
        return <CheckmarkCircle24Regular />;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'healthy':
        return <Badge appearance="filled" color="success">Healthy</Badge>;
      case 'warning':
        return <Badge appearance="filled" color="warning">Warning</Badge>;
      case 'error':
        return <Badge appearance="filled" color="danger">Error</Badge>;
      default:
        return <Badge appearance="filled">Unknown</Badge>;
    }
  };

  return (
    <div className={classes.container}>
      {/* Welcome Card */}
      <Card className={classes.welcomeCard}>
        <Title2 className={classes.welcomeTitle}>Admin Dashboard</Title2>
        <Text className={classes.welcomeText}>
          Welcome to the AuthAI Administration Portal. Monitor your authorization system, manage policies, and oversee security operations.
        </Text>
      </Card>

      {/* Key Statistics */}
      <div className={classes.statsGrid}>
        <Card className={classes.statCard}>
          <div className={classes.statHeader}>
            <People24Regular className={classes.statIcon} />
            <Text className={classes.statLabel}>Active Subscribers</Text>
          </div>
          <Text className={classes.statValue}>{stats.activeSubscribers}</Text>
        </Card>

        <Card className={classes.statCard}>
          <div className={classes.statHeader}>
            <Shield24Regular className={classes.statIcon} />
            <Text className={classes.statLabel}>Authorization Requests</Text>
          </div>
          <Text className={classes.statValue}>{stats.totalTokens.toLocaleString()}</Text>
        </Card>

        <Card className={classes.statCard}>
          <div className={classes.statHeader}>
            <DocumentBulletList24Regular className={classes.statIcon} />
            <Text className={classes.statLabel}>Active Policies</Text>
          </div>
          <Text className={classes.statValue}>{stats.activePolicies}</Text>
        </Card>

        <Card className={classes.statCard}>
          <div className={classes.statHeader}>
            <Gauge24Regular className={classes.statIcon} />
            <Text className={classes.statLabel}>System Uptime</Text>
          </div>
          <Text className={classes.statValue}>{stats.uptime}</Text>
        </Card>
      </div>

      {/* System Status */}
      <div className={classes.section}>
        <div className={classes.sectionTitle}>
          <CheckmarkCircle24Regular style={{ fontSize: '24px' }} />
          <Title3>System Status</Title3>
        </div>
        <Card>
          <div className={classes.statusList}>
            {systemStatus.map((status, index) => (
              <div key={index} className={classes.statusItem}>
                <div className={classes.statusInfo}>
                  {getStatusIcon(status.status)}
                  <div>
                    <Text weight="semibold">{status.name}</Text>
                    <Text size={200} style={{ display: 'block', color: tokens.colorNeutralForeground3 }}>
                      {status.message}
                    </Text>
                  </div>
                </div>
                {getStatusBadge(status.status)}
              </div>
            ))}
          </div>
        </Card>
      </div>

      {/* Quick Actions */}
      <div className={classes.section}>
        <div className={classes.sectionTitle}>
          <DocumentBulletList24Regular style={{ fontSize: '24px' }} />
          <Title3>Quick Actions</Title3>
        </div>
        <div className={classes.quickActions}>
          <Card className={classes.actionCard} onClick={() => window.location.href = '/admin/authorization'}>
            <Shield24Regular style={{ fontSize: '32px', color: tokens.colorBrandForeground1 }} />
            <Text className={classes.actionTitle}>Manage Policies</Text>
            <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
              Create and manage authorization policies
            </Text>
          </Card>

          <Card className={classes.actionCard} onClick={() => window.location.href = '/admin/subscribers/list'}>
            <People24Regular style={{ fontSize: '32px', color: tokens.colorBrandForeground1 }} />
            <Text className={classes.actionTitle}>View Subscribers</Text>
            <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
              Manage tenant subscriptions
            </Text>
          </Card>

          <Card className={classes.actionCard} onClick={() => window.location.href = '/admin/tokens'}>
            <DocumentBulletList24Regular style={{ fontSize: '32px', color: tokens.colorBrandForeground1 }} />
            <Text className={classes.actionTitle}>Token Management</Text>
            <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
              View and manage access tokens
            </Text>
          </Card>

          <Card className={classes.actionCard} onClick={() => window.location.href = '/admin/audit'}>
            <Gauge24Regular style={{ fontSize: '32px', color: tokens.colorBrandForeground1 }} />
            <Text className={classes.actionTitle}>Audit Trail</Text>
            <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
              Review system audit logs
            </Text>
          </Card>
        </div>
      </div>
    </div>
  );
}
