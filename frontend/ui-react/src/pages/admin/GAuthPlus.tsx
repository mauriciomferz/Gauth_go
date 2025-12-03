/**
 * GAuth+ Management Dashboard
 * Comprehensive UI for managing AI-specific authorization features
 */

import { useState } from 'react'
import {
  makeStyles,
  Tab,
  TabList,
  Spinner,
  Text,
  Title3,
  Card,
} from '@fluentui/react-components'
import {
  ShieldCheckmark24Regular,
  PersonSwap24Regular,
  BuildingMultiple24Regular,
  Certificate24Regular,
  Shield24Regular,
} from '@fluentui/react-icons'
import SuccessorPanel from '../../components/gauthplus/SuccessorPanel'
import DelegationPanel from '../../components/gauthplus/DelegationPanel'
import DualControlPanel from '../../components/gauthplus/DualControlPanel'
import CapabilityPanel from '../../components/gauthplus/CapabilityPanel'
import FiduciaryPanel from '../../components/gauthplus/FiduciaryPanel'

const useStyles = makeStyles({
  container: {
    padding: '24px',
    maxWidth: '1600px',
    margin: '0 auto',
  },
  header: {
    marginBottom: '24px',
  },
  subtitle: {
    color: '#666',
    marginTop: '8px',
  },
  tabsContainer: {
    marginBottom: '24px',
  },
  tabContent: {
    padding: '24px 0',
  },
  loadingContainer: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    minHeight: '400px',
  },
  statsCard: {
    marginBottom: '24px',
    padding: '16px',
  },
  statsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    gap: '16px',
  },
  statItem: {
    display: 'flex',
    flexDirection: 'column',
    gap: '4px',
  },
  statValue: {
    fontSize: '24px',
    fontWeight: 600,
  },
  statLabel: {
    fontSize: '12px',
    color: '#666',
  },
})

type TabValue = 'successor' | 'delegation' | 'dual-control' | 'capability' | 'fiduciary'

export default function GAuthPlus() {
  const styles = useStyles()
  const [selectedTab, setSelectedTab] = useState<TabValue>('successor')
  const [loading] = useState(false)

  const renderTabContent = () => {
    if (loading) {
      return (
        <div className={styles.loadingContainer}>
          <Spinner size="large" label="Loading..." />
        </div>
      )
    }

    switch (selectedTab) {
      case 'successor':
        return <SuccessorPanel />
      case 'delegation':
        return <DelegationPanel />
      case 'dual-control':
        return <DualControlPanel />
      case 'capability':
        return <CapabilityPanel />
      case 'fiduciary':
        return <FiduciaryPanel />
      default:
        return null
    }
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <Title3>GAuth+ Management</Title3>
        <Text className={styles.subtitle}>
          Advanced AI authorization features - Successor management, delegation chains, dual
          control approvals, capability assessments, and fiduciary duty tracking
        </Text>
      </div>

      <Card className={styles.statsCard}>
        <div className={styles.statsGrid}>
          <div className={styles.statItem}>
            <div className={styles.statValue}>27</div>
            <div className={styles.statLabel}>REST Endpoints</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statValue}>5</div>
            <div className={styles.statLabel}>Core Features</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statValue}>100%</div>
            <div className={styles.statLabel}>Test Coverage</div>
          </div>
          <div className={styles.statItem}>
            <div className={styles.statValue}>Production</div>
            <div className={styles.statLabel}>Status</div>
          </div>
        </div>
      </Card>

      <div className={styles.tabsContainer}>
        <TabList
          selectedValue={selectedTab}
          onTabSelect={(_, data) => setSelectedTab(data.value as TabValue)}
        >
          <Tab
            icon={<PersonSwap24Regular />}
            value="successor"
          >
            Successor Management
          </Tab>
          <Tab
            icon={<BuildingMultiple24Regular />}
            value="delegation"
          >
            Delegation Chains
          </Tab>
          <Tab
            icon={<ShieldCheckmark24Regular />}
            value="dual-control"
          >
            Dual Control
          </Tab>
          <Tab
            icon={<Certificate24Regular />}
            value="capability"
          >
            Capability Assessment
          </Tab>
          <Tab
            icon={<Shield24Regular />}
            value="fiduciary"
          >
            Fiduciary Duties
          </Tab>
        </TabList>
      </div>

      <div className={styles.tabContent}>{renderTabContent()}</div>
    </div>
  )
}
