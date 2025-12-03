import { useEffect, useState } from 'react';
import {
  makeStyles,
  tokens,
  Card,
  Text,
  Title3,
  Badge,
  ProgressBar,
} from '@fluentui/react-components';
import {
  DocumentCheckmark24Regular,
  ClockAlarm24Regular,
  CheckmarkCircle24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  title: {
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
  cardIcon: {
    fontSize: '20px',
  },
  metricValue: {
    fontSize: '32px',
    fontWeight: 600,
    marginBottom: '8px',
  },
  metricSubtext: {
    fontSize: '13px',
    color: tokens.colorNeutralForeground3,
    marginBottom: '12px',
  },
  detailsGrid: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '12px',
    marginTop: '16px',
  },
  detailItem: {
    display: 'flex',
    flexDirection: 'column',
    gap: '4px',
  },
  detailLabel: {
    fontSize: '11px',
    color: tokens.colorNeutralForeground3,
    textTransform: 'uppercase',
  },
  detailValue: {
    fontSize: '16px',
    fontWeight: 600,
  },
});

interface SemanticCounter {
  capabilityAnchorValidations: number;
  capabilityAnchorResolutions: number;
  avgResolutionTime: number;
  successRate: number;
  activeAnchors: number;
  failedValidations: number;
  cachedAnchors: number;
  cacheHitRate: number;
}

export default function SemanticCounters() {
  const classes = useStyles();
  const [counters, setCounters] = useState<SemanticCounter>({
    capabilityAnchorValidations: 0,
    capabilityAnchorResolutions: 0,
    avgResolutionTime: 0,
    successRate: 0,
    activeAnchors: 0,
    failedValidations: 0,
    cachedAnchors: 0,
    cacheHitRate: 0,
  });

  useEffect(() => {
    fetchCounters();
    const interval = setInterval(fetchCounters, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const fetchCounters = async () => {
    try {
      const response = await fetch('/api/admin/metrics/semantic-counters', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        setCounters(data.counters);
      }
    } catch (error) {
      console.error('Failed to fetch semantic counters:', error);
    }
  };

  return (
    <div className={classes.container}>
      <div className={classes.title}>
        <DocumentCheckmark24Regular style={{ fontSize: '24px' }} />
        <Title3>Semantic Counters & Capability Anchor Status</Title3>
      </div>

      <div className={classes.cardsGrid}>
        {/* Capability Anchor Validations */}
        <Card className={classes.card}>
          <div className={classes.cardHeader}>
            <Text className={classes.cardTitle}>
              <CheckmarkCircle24Regular className={classes.cardIcon} />
              Anchor Validations
            </Text>
            <Badge appearance="tint" color="success">Active</Badge>
          </div>
          <Text className={classes.metricValue}>
            {counters.capabilityAnchorValidations.toLocaleString()}
          </Text>
          <Text className={classes.metricSubtext}>
            Total validations processed
          </Text>
          <div className={classes.detailsGrid}>
            <div className={classes.detailItem}>
              <Text className={classes.detailLabel}>Success Rate</Text>
              <Text className={classes.detailValue} style={{ color: tokens.colorPaletteGreenForeground1 }}>
                {(counters.successRate * 100).toFixed(1)}%
              </Text>
            </div>
            <div className={classes.detailItem}>
              <Text className={classes.detailLabel}>Failed</Text>
              <Text className={classes.detailValue} style={{ color: tokens.colorPaletteRedForeground1 }}>
                {counters.failedValidations.toLocaleString()}
              </Text>
            </div>
          </div>
        </Card>

        {/* Capability Anchor Resolutions */}
        <Card className={classes.card}>
          <div className={classes.cardHeader}>
            <Text className={classes.cardTitle}>
              <DocumentCheckmark24Regular className={classes.cardIcon} />
              Anchor Resolutions
            </Text>
          </div>
          <Text className={classes.metricValue}>
            {counters.capabilityAnchorResolutions.toLocaleString()}
          </Text>
          <Text className={classes.metricSubtext}>
            Capability anchors resolved
          </Text>
          <div className={classes.detailsGrid}>
            <div className={classes.detailItem}>
              <Text className={classes.detailLabel}>Active Anchors</Text>
              <Text className={classes.detailValue}>
                {counters.activeAnchors.toLocaleString()}
              </Text>
            </div>
            <div className={classes.detailItem}>
              <Text className={classes.detailLabel}>Cached</Text>
              <Text className={classes.detailValue}>
                {counters.cachedAnchors.toLocaleString()}
              </Text>
            </div>
          </div>
        </Card>

        {/* Resolution Time */}
        <Card className={classes.card}>
          <div className={classes.cardHeader}>
            <Text className={classes.cardTitle}>
              <ClockAlarm24Regular className={classes.cardIcon} />
              Avg Resolution Time
            </Text>
          </div>
          <Text className={classes.metricValue}>
            {counters.avgResolutionTime.toFixed(1)}ms
          </Text>
          <Text className={classes.metricSubtext}>
            Per capability anchor
          </Text>
          <ProgressBar
            value={Math.min(counters.avgResolutionTime / 200, 1)}
            max={1}
            color={counters.avgResolutionTime < 100 ? 'success' : 'warning'}
          />
          <Text size={200} style={{ marginTop: '8px' }}>
            {counters.avgResolutionTime < 100 ? 'Excellent performance' : 'Within acceptable range'}
          </Text>
        </Card>

        {/* Cache Performance */}
        <Card className={classes.card}>
          <div className={classes.cardHeader}>
            <Text className={classes.cardTitle}>
              <DocumentCheckmark24Regular className={classes.cardIcon} />
              Cache Performance
            </Text>
          </div>
          <Text className={classes.metricValue}>
            {(counters.cacheHitRate * 100).toFixed(1)}%
          </Text>
          <Text className={classes.metricSubtext}>
            Cache hit rate
          </Text>
          <ProgressBar
            value={counters.cacheHitRate}
            max={1}
            color={counters.cacheHitRate > 0.8 ? 'success' : 'warning'}
          />
          <Text size={200} style={{ marginTop: '8px' }}>
            {counters.cachedAnchors.toLocaleString()} anchors cached
          </Text>
        </Card>
      </div>
    </div>
  );
}
