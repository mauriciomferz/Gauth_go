/**
 * Dual Control Approval Panel
 * Manage multi-approver workflows
 */

import { useState, useEffect } from 'react'
import {
  makeStyles,
  Button,
  Card,
  Spinner,
  Text,
  Title3,
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableHeaderCell,
  TableRow,
  Badge,
} from '@fluentui/react-components'
import { Checkmark24Regular, Dismiss24Regular } from '@fluentui/react-icons'
import { agentAuthPlusAPI, DualControlApproval } from '../../lib/agentauthplus-api'

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
  emptyState: {
    padding: '40px',
    textAlign: 'center',
    color: '#666',
  },
  actionButtons: {
    display: 'flex',
    gap: '8px',
  },
})

export default function DualControlPanel() {
  const styles = useStyles()
  const [loading, setLoading] = useState(false)
  const [approvals, setApprovals] = useState<DualControlApproval[]>([])

  useEffect(() => {
    fetchPendingApprovals()
  }, [])

  const fetchPendingApprovals = async () => {
    try {
      setLoading(true)
      const response = await agentAuthPlusAPI.getPendingApprovals()
      setApprovals(response.approvals || [])
    } catch (error) {
      console.error('Failed to fetch pending approvals:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleApprove = async (approvalId: string) => {
    try {
      await agentAuthPlusAPI.approveAction(approvalId, 'admin', 'Approved from UI')
      await fetchPendingApprovals()
    } catch (error) {
      console.error('Failed to approve:', error)
      alert('Failed to approve. Check console for details.')
    }
  }

  const handleReject = async (approvalId: string) => {
    try {
      await agentAuthPlusAPI.rejectAction(approvalId, 'admin', 'Rejected from UI')
      await fetchPendingApprovals()
    } catch (error) {
      console.error('Failed to reject:', error)
      alert('Failed to reject. Check console for details.')
    }
  }

  const getStatusBadge = (status: string) => {
    const colors: Record<string, 'success' | 'warning' | 'danger' | 'subtle'> = {
      pending: 'warning',
      approved: 'success',
      rejected: 'danger',
      expired: 'subtle',
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
          <Title3>Pending Approvals</Title3>
          <Button onClick={fetchPendingApprovals} disabled={loading}>
            Refresh
          </Button>
        </div>

        {loading ? (
          <div style={{ padding: '40px', textAlign: 'center' }}>
            <Spinner size="medium" />
          </div>
        ) : approvals.length === 0 ? (
          <div className={styles.emptyState}>
            <Text>No pending approvals</Text>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHeaderCell>Action</TableHeaderCell>
                <TableHeaderCell>Description</TableHeaderCell>
                <TableHeaderCell>Requested By</TableHeaderCell>
                <TableHeaderCell>Required Approvers</TableHeaderCell>
                <TableHeaderCell>Status</TableHeaderCell>
                <TableHeaderCell>Actions</TableHeaderCell>
              </TableRow>
            </TableHeader>
            <TableBody>
              {approvals.map((approval) => (
                <TableRow key={approval.id}>
                  <TableCell>{approval.action_type}</TableCell>
                  <TableCell>{approval.action_description}</TableCell>
                  <TableCell>{approval.requested_by}</TableCell>
                  <TableCell>
                    {(approval.approved_by || []).length}/{approval.required_approvers}
                  </TableCell>
                  <TableCell>{getStatusBadge(approval.status)}</TableCell>
                  <TableCell>
                    {approval.status === 'pending' && (
                      <div className={styles.actionButtons}>
                        <Button
                          size="small"
                          icon={<Checkmark24Regular />}
                          appearance="primary"
                          onClick={() => handleApprove(approval.id)}
                        >
                          Approve
                        </Button>
                        <Button
                          size="small"
                          icon={<Dismiss24Regular />}
                          onClick={() => handleReject(approval.id)}
                        >
                          Reject
                        </Button>
                      </div>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  )
}
