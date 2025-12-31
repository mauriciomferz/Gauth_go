/**
 * Fiduciary Duty Violation Panel
 * Track and resolve fiduciary duty breaches
 */

import { useState, useEffect } from 'react'
import {
  makeStyles,
  Button,
  Card,
  Input,
  Spinner,
  Text,
  Title3,
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableHeaderCell,
  TableRow,
  Field,
  Badge,
  Dropdown,
  Option,
} from '@fluentui/react-components'
import { Filter24Regular } from '@fluentui/react-icons'
import { agentAuthPlusAPI, FiduciaryDutyViolation } from '../../lib/agentauthplus-api'

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
  filterBar: {
    display: 'flex',
    gap: '8px',
    marginBottom: '16px',
  },
  emptyState: {
    padding: '40px',
    textAlign: 'center',
    color: '#666',
  },
})

export default function FiduciaryPanel() {
  const styles = useStyles()
  const [loading, setLoading] = useState(false)
  const [violations, setViolations] = useState<FiduciaryDutyViolation[]>([])
  const [poaId, setPoaId] = useState('')
  const [severityFilter, setSeverityFilter] = useState<string>('')

  useEffect(() => {
    fetchViolations()
  }, [])

  const fetchViolations = async () => {
    try {
      setLoading(true)
      let response
      if (severityFilter) {
        response = await agentAuthPlusAPI.getViolationsBySeverity(severityFilter)
      } else {
        response = await agentAuthPlusAPI.getViolations(poaId || undefined)
      }
      setViolations(response.violations || [])
    } catch (error) {
      console.error('Failed to fetch violations:', error)
    } finally {
      setLoading(false)
    }
  }

  const getSeverityBadge = (severity: string) => {
    const colors: Record<string, 'success' | 'warning' | 'important' | 'danger'> = {
      minor: 'success',
      moderate: 'warning',
      major: 'important',
      critical: 'danger',
    }
    return (
      <Badge appearance="filled" color={colors[severity] || 'subtle'}>
        {severity.toUpperCase()}
      </Badge>
    )
  }

  const getStatusBadge = (status: string) => {
    const colors: Record<string, 'success' | 'warning' | 'important' | 'subtle'> = {
      open: 'important',
      investigating: 'warning',
      resolved: 'success',
      dismissed: 'subtle',
    }
    return (
      <Badge appearance="filled" color={colors[status] || 'subtle'}>
        {status.toUpperCase()}
      </Badge>
    )
  }

  return (
    <div className={styles.container}>
      <Card className={styles.card}>
        <div className={styles.header}>
          <Title3>Fiduciary Duty Violations</Title3>
          <Button icon={<Filter24Regular />} onClick={fetchViolations} disabled={loading}>
            Refresh
          </Button>
        </div>

        <div className={styles.filterBar}>
          <Field label="Proof of Authorization ID" style={{ flex: 1 }}>
            <Input
              value={poaId}
              onChange={(_, data) => setPoaId(data.value)}
              placeholder="Filter by PoA ID (optional)"
            />
          </Field>
          <Field label="Min Severity">
            <Dropdown
              placeholder="All severities"
              value={severityFilter}
              onOptionSelect={(_, data) => setSeverityFilter(data.optionValue || '')}
              style={{ minWidth: '150px' }}
            >
              <Option value="">All</Option>
              <Option value="minor">Minor</Option>
              <Option value="moderate">Moderate</Option>
              <Option value="major">Major</Option>
              <Option value="critical">Critical</Option>
            </Dropdown>
          </Field>
          <Button
            appearance="primary"
            onClick={fetchViolations}
            disabled={loading}
            style={{ alignSelf: 'flex-end' }}
          >
            Apply
          </Button>
        </div>

        {loading ? (
          <div style={{ padding: '40px', textAlign: 'center' }}>
            <Spinner size="medium" />
          </div>
        ) : violations.length === 0 ? (
          <div className={styles.emptyState}>
            <Text>No violations found</Text>
          </div>
        ) : (
          <>
            <Text style={{ marginBottom: '16px' }}>
              Found {violations.length} violation{violations.length !== 1 ? 's' : ''}
            </Text>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHeaderCell>Agent ID</TableHeaderCell>
                  <TableHeaderCell>Duty Type</TableHeaderCell>
                  <TableHeaderCell>Description</TableHeaderCell>
                  <TableHeaderCell>Severity</TableHeaderCell>
                  <TableHeaderCell>Detected</TableHeaderCell>
                  <TableHeaderCell>Status</TableHeaderCell>
                </TableRow>
              </TableHeader>
              <TableBody>
                {violations.map((violation) => (
                  <TableRow key={violation.id}>
                    <TableCell>{violation.agent_id}</TableCell>
                    <TableCell>{violation.duty_type}</TableCell>
                    <TableCell>{violation.violation_description}</TableCell>
                    <TableCell>{getSeverityBadge(violation.severity)}</TableCell>
                    <TableCell>{new Date(violation.detected_at).toLocaleString()}</TableCell>
                    <TableCell>{getStatusBadge(violation.resolution_status)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </>
        )}
      </Card>
    </div>
  )
}
