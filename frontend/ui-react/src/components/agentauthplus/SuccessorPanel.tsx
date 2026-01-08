/**
 * Successor Management Panel
 * Manage AI successor activations and history
 */

import { useState, useEffect, useCallback } from 'react'
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
  Dialog,
  DialogSurface,
  DialogTitle,
  DialogBody,
  DialogActions,
  DialogContent,
  Field,
  Badge,
} from '@fluentui/react-components'
import {
  Add24Regular,
  ArrowSync24Regular,
  History24Regular,
} from '@fluentui/react-icons'
import { agentAuthPlusAPI, SuccessorActivation } from '../../lib/agentauthplus-api'

type ErrorWithResponseStatus = { response?: { status?: number } }

function getResponseStatus(error: unknown): number | undefined {
  const status = (error as ErrorWithResponseStatus).response?.status
  return typeof status === 'number' ? status : undefined
}

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
  card: {
    padding: '24px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '16px',
  },
  formGrid: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '16px',
  },
  statusBadge: {
    marginLeft: '8px',
  },
  emptyState: {
    padding: '40px',
    textAlign: 'center',
    color: '#666',
  },
})

export default function SuccessorPanel() {
  const styles = useStyles()
  const [loading, setLoading] = useState(false)
  const [activeSuccessor, setActiveSuccessor] = useState<SuccessorActivation | null>(null)
  const [history, setHistory] = useState<SuccessorActivation[]>([])
  const [activateDialogOpen, setActivateDialogOpen] = useState(false)
  const [poaId, setPoaId] = useState('00000000-0000-0000-0000-000000000001')
  const [formData, setFormData] = useState({
    primary_agent_id: '',
    successor_agent_id: '',
    reason: 'unavailable',
    activated_by: 'admin',
  })

  const fetchActiveSuccessor = useCallback(async () => {
    try {
      setLoading(true)
      const response = await agentAuthPlusAPI.getActiveSuccessor(poaId)
      setActiveSuccessor(response.active_successor)
    } catch (error: unknown) {
      if (getResponseStatus(error) === 404) {
        // Endpoint not available (dev mode without database) - silently handle
        setActiveSuccessor(null)
      } else {
        console.error('Failed to fetch active successor:', error)
      }
    } finally {
      setLoading(false)
    }
  }, [poaId])

  const fetchHistory = useCallback(async () => {
    try {
      const response = await agentAuthPlusAPI.listSuccessorHistory(poaId)
      setHistory(response.history || [])
    } catch (error: unknown) {
      if (getResponseStatus(error) === 404) {
        // Endpoint not available (dev mode without database) - silently handle
        setHistory([])
      } else {
        console.error('Failed to fetch history:', error)
      }
    }
  }, [poaId])

  useEffect(() => {
    if (poaId) {
      fetchActiveSuccessor()
      fetchHistory()
    }
  }, [poaId, fetchActiveSuccessor, fetchHistory])

  const handleActivate = async () => {
    try {
      setLoading(true)
      await agentAuthPlusAPI.activateSuccessor({
        poa_id: poaId,
        ...formData,
      })
      setActivateDialogOpen(false)
      setFormData({
        primary_agent_id: '',
        successor_agent_id: '',
        reason: 'unavailable',
        activated_by: 'admin',
      })
      await fetchActiveSuccessor()
      await fetchHistory()
    } catch (error) {
      console.error('Failed to activate successor:', error)
      alert('Failed to activate successor. Check console for details.')
    } finally {
      setLoading(false)
    }
  }

  const handleDeactivate = async () => {
    if (!activeSuccessor) return
    if (!confirm('Are you sure you want to deactivate the current successor?')) return

    try {
      setLoading(true)
      await agentAuthPlusAPI.deactivateSuccessor({
        activation_id: activeSuccessor.id,
        deactivated_by: 'admin',
      })
      await fetchActiveSuccessor()
      await fetchHistory()
    } catch (error) {
      console.error('Failed to deactivate successor:', error)
      alert('Failed to deactivate successor. Check console for details.')
    } finally {
      setLoading(false)
    }
  }

  const getStatusBadge = (status: string) => {
    const colors: Record<string, 'success' | 'warning' | 'subtle'> = {
      active: 'success',
      deactivated: 'subtle',
      superseded: 'warning',
    }
    return (
      <Badge appearance="filled" color={colors[status] || 'subtle'}>
        {status.toUpperCase()}
      </Badge>
    )
  }

  return (
    <div className={styles.container}>
      {/* Active Successor Section */}
      <Card className={styles.card}>
        <div className={styles.header}>
          <Title3>Active Successor</Title3>
          <div style={{ display: 'flex', gap: '8px' }}>
            <Button
              icon={<Add24Regular />}
              appearance="primary"
              onClick={() => setActivateDialogOpen(true)}
              disabled={loading || !!activeSuccessor}
            >
              Activate Successor
            </Button>
            {activeSuccessor && (
              <Button
                icon={<ArrowSync24Regular />}
                onClick={handleDeactivate}
                disabled={loading}
              >
                Deactivate
              </Button>
            )}
          </div>
        </div>

        <Field label="Proof of Authorization ID">
          <Input
            value={poaId}
            onChange={(_, data) => setPoaId(data.value)}
            placeholder="Enter PoA ID"
          />
        </Field>

        {loading ? (
          <div style={{ padding: '40px', textAlign: 'center' }}>
            <Spinner size="medium" />
          </div>
        ) : activeSuccessor ? (
          <div style={{ marginTop: '16px' }}>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHeaderCell>Primary Agent</TableHeaderCell>
                  <TableHeaderCell>Successor Agent</TableHeaderCell>
                  <TableHeaderCell>Reason</TableHeaderCell>
                  <TableHeaderCell>Activated</TableHeaderCell>
                  <TableHeaderCell>Status</TableHeaderCell>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell>{activeSuccessor.primary_agent_id}</TableCell>
                  <TableCell>{activeSuccessor.successor_agent_id}</TableCell>
                  <TableCell>{activeSuccessor.activation_reason}</TableCell>
                  <TableCell>{new Date(activeSuccessor.activated_at).toLocaleString()}</TableCell>
                  <TableCell>{getStatusBadge(activeSuccessor.status)}</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        ) : (
          <div className={styles.emptyState}>
            <Text>No active successor</Text>
          </div>
        )}
      </Card>

      {/* History Section */}
      <Card className={styles.card}>
        <div className={styles.header}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <History24Regular />
            <Title3>Activation History</Title3>
          </div>
        </div>

        {history.length === 0 ? (
          <div className={styles.emptyState}>
            <Text>No activation history</Text>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHeaderCell>Primary Agent</TableHeaderCell>
                <TableHeaderCell>Successor Agent</TableHeaderCell>
                <TableHeaderCell>Reason</TableHeaderCell>
                <TableHeaderCell>Activated</TableHeaderCell>
                <TableHeaderCell>Deactivated</TableHeaderCell>
                <TableHeaderCell>Status</TableHeaderCell>
              </TableRow>
            </TableHeader>
            <TableBody>
              {history.map((activation) => (
                <TableRow key={activation.id}>
                  <TableCell>{activation.primary_agent_id}</TableCell>
                  <TableCell>{activation.successor_agent_id}</TableCell>
                  <TableCell>{activation.activation_reason}</TableCell>
                  <TableCell>{new Date(activation.activated_at).toLocaleString()}</TableCell>
                  <TableCell>
                    {activation.deactivated_at
                      ? new Date(activation.deactivated_at).toLocaleString()
                      : '-'}
                  </TableCell>
                  <TableCell>{getStatusBadge(activation.status)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {/* Activate Dialog */}
      <Dialog open={activateDialogOpen} onOpenChange={(_, data) => setActivateDialogOpen(data.open)}>
        <DialogSurface>
          <DialogBody>
            <DialogTitle>Activate Successor</DialogTitle>
            <DialogContent>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                <Field label="Primary Agent ID" required>
                  <Input
                    value={formData.primary_agent_id}
                    onChange={(_, data) =>
                      setFormData({ ...formData, primary_agent_id: data.value })
                    }
                    placeholder="e.g., ai-agent-001"
                  />
                </Field>
                <Field label="Successor Agent ID" required>
                  <Input
                    value={formData.successor_agent_id}
                    onChange={(_, data) =>
                      setFormData({ ...formData, successor_agent_id: data.value })
                    }
                    placeholder="e.g., ai-agent-backup"
                  />
                </Field>
                <Field label="Reason" required>
                  <Input
                    value={formData.reason}
                    onChange={(_, data) => setFormData({ ...formData, reason: data.value })}
                    placeholder="unavailable, failure, manual, timeout"
                  />
                </Field>
                <Field label="Activated By">
                  <Input
                    value={formData.activated_by}
                    onChange={(_, data) => setFormData({ ...formData, activated_by: data.value })}
                  />
                </Field>
              </div>
            </DialogContent>
            <DialogActions>
              <Button appearance="secondary" onClick={() => setActivateDialogOpen(false)}>
                Cancel
              </Button>
              <Button
                appearance="primary"
                onClick={handleActivate}
                disabled={
                  !formData.primary_agent_id ||
                  !formData.successor_agent_id ||
                  !formData.reason ||
                  loading
                }
              >
                {loading ? <Spinner size="tiny" /> : 'Activate'}
              </Button>
            </DialogActions>
          </DialogBody>
        </DialogSurface>
      </Dialog>
    </div>
  )
}
