/**
 * Capability Assessment Panel
 * View AI agent capability assessments
 */

import { useState } from 'react'
import {
  makeStyles,
  Button,
  Card,
  Input,
  Spinner,
  Text,
  Title3,
  Field,
  Badge,
} from '@fluentui/react-components'
import { Search24Regular } from '@fluentui/react-icons'
import { gauthPlusAPI, AICapabilityAssessment } from '../../lib/gauthplus-api'

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '24px',
  },
  card: {
    padding: '24px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '16px',
  },
  searchBar: {
    display: 'flex',
    gap: '8px',
    marginBottom: '16px',
  },
  emptyState: {
    padding: '40px',
    textAlign: 'center',
    color: '#666',
  },
  detailsGrid: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '16px',
    marginTop: '16px',
  },
  detailItem: {
    display: 'flex',
    flexDirection: 'column',
    gap: '4px',
  },
  label: {
    fontWeight: 600,
    fontSize: '12px',
    color: '#666',
  },
  value: {
    fontSize: '14px',
  },
  domainsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))',
    gap: '12px',
    marginTop: '16px',
  },
  domainCard: {
    padding: '12px',
    border: '1px solid #e0e0e0',
    borderRadius: '4px',
  },
})

export default function CapabilityPanel() {
  const styles = useStyles()
  const [loading, setLoading] = useState(false)
  const [agentId, setAgentId] = useState('ai-agent-001')
  const [assessment, setAssessment] = useState<AICapabilityAssessment | null>(null)

  const fetchAssessment = async () => {
    if (!agentId) return
    try {
      setLoading(true)
      const response = await gauthPlusAPI.getLatestAssessment(agentId)
      setAssessment(response.assessment)
    } catch (error) {
      console.error('Failed to fetch assessment:', error)
      setAssessment(null)
    } finally {
      setLoading(false)
    }
  }

  const getLevelBadge = (level: string) => {
    const colors: Record<string, 'success' | 'warning' | 'important' | 'subtle'> = {
      L0: 'subtle',
      L1: 'subtle',
      L2: 'warning',
      L3: 'warning',
      L4: 'important',
      L5: 'success',
    }
    return (
      <Badge appearance="filled" color={colors[level] || 'subtle'} size="large">
        {level}
      </Badge>
    )
  }

  return (
    <div className={styles.container}>
      <Card className={styles.card}>
        <div className={styles.header}>
          <Title3>Capability Assessment</Title3>
        </div>

        <div className={styles.searchBar}>
          <Field label="Agent ID" style={{ flex: 1 }}>
            <Input
              value={agentId}
              onChange={(_, data) => setAgentId(data.value)}
              placeholder="Enter agent ID"
            />
          </Field>
          <Button
            icon={<Search24Regular />}
            appearance="primary"
            onClick={fetchAssessment}
            disabled={loading}
            style={{ alignSelf: 'flex-end' }}
          >
            Search
          </Button>
        </div>

        {loading ? (
          <div style={{ padding: '40px', textAlign: 'center' }}>
            <Spinner size="medium" />
          </div>
        ) : !assessment ? (
          <div className={styles.emptyState}>
            <Text>No assessment found</Text>
          </div>
        ) : (
          <>
            <div className={styles.detailsGrid}>
              <div className={styles.detailItem}>
                <div className={styles.label}>Overall Level</div>
                <div>{getLevelBadge(assessment.overall_level)}</div>
              </div>
              <div className={styles.detailItem}>
                <div className={styles.label}>Certification Status</div>
                <div className={styles.value}>{assessment.certification_status}</div>
              </div>
              <div className={styles.detailItem}>
                <div className={styles.label}>Assessed By</div>
                <div className={styles.value}>{assessment.assessed_by}</div>
              </div>
              <div className={styles.detailItem}>
                <div className={styles.label}>Valid Until</div>
                <div className={styles.value}>
                  {new Date(assessment.valid_until).toLocaleString()}
                </div>
              </div>
            </div>

            <div style={{ marginTop: '24px' }}>
              <Text weight="semibold">Domain Scores</Text>
              <div className={styles.domainsGrid}>
                {Object.entries(assessment.domain_scores || {}).map(([domain, score]) => (
                  <div key={domain} className={styles.domainCard}>
                    <Text size={200} weight="semibold">
                      {domain}
                    </Text>
                    <Text size={400} weight="bold">
                      {(score * 100).toFixed(0)}%
                    </Text>
                  </div>
                ))}
              </div>
            </div>

            {assessment.certifications && assessment.certifications.length > 0 && (
              <div style={{ marginTop: '24px' }}>
                <Text weight="semibold">Certifications</Text>
                <div style={{ marginTop: '8px', display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
                  {assessment.certifications.map((cert) => (
                    <Badge key={cert} appearance="tint">
                      {cert}
                    </Badge>
                  ))}
                </div>
              </div>
            )}

            {assessment.notes && (
              <div style={{ marginTop: '24px' }}>
                <Text weight="semibold">Notes</Text>
                <Text style={{ marginTop: '8px', display: 'block' }}>{assessment.notes}</Text>
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  )
}
