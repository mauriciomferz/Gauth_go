/**
 * Delegation Management Panel
 * View and manage AI-to-AI delegation chains
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
  Field,
  Badge,
} from '@fluentui/react-components'
import { Search24Regular } from '@fluentui/react-icons'
import { gauthPlusAPI, AIDelegation } from '../../lib/gauthplus-api'

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
})

export default function DelegationPanel() {
  const styles = useStyles()
  const [loading, setLoading] = useState(false)
  const [agentId, setAgentId] = useState('ai-agent-001')
  const [chain, setChain] = useState<AIDelegation[]>([])

  const fetchChain = useCallback(async () => {
    if (!agentId) return
    try {
      setLoading(true)
      const response = await gauthPlusAPI.getDelegationChain(agentId)
      setChain(response.chain || [])
    } catch (error) {
      console.error('Failed to fetch delegation chain:', error)
    } finally {
      setLoading(false)
    }
  }, [agentId])

  useEffect(() => {
    if (agentId) {
      fetchChain()
    }
  }, [agentId, fetchChain])

  const getStatusBadge = (status: string) => {
    const colors: Record<string, 'success' | 'warning' | 'subtle'> = {
      active: 'success',
      revoked: 'warning',
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
          <Title3>Delegation Chain Viewer</Title3>
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
            onClick={fetchChain}
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
        ) : chain.length === 0 ? (
          <div className={styles.emptyState}>
            <Text>No delegation chain found</Text>
          </div>
        ) : (
          <>
            <Text style={{ marginBottom: '16px' }}>
              Found {chain.length} delegation{chain.length !== 1 ? 's' : ''} (Depth: {chain.length})
            </Text>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHeaderCell>Source Agent</TableHeaderCell>
                  <TableHeaderCell>Target Agent</TableHeaderCell>
                  <TableHeaderCell>Scope</TableHeaderCell>
                  <TableHeaderCell>Depth</TableHeaderCell>
                  <TableHeaderCell>Valid Until</TableHeaderCell>
                  <TableHeaderCell>Status</TableHeaderCell>
                </TableRow>
              </TableHeader>
              <TableBody>
                {chain.map((delegation) => (
                  <TableRow key={delegation.id}>
                    <TableCell>{delegation.source_agent_id}</TableCell>
                    <TableCell>{delegation.target_agent_id}</TableCell>
                    <TableCell>{delegation.delegated_scope.join(', ')}</TableCell>
                    <TableCell>
                      {delegation.delegation_depth}/{delegation.max_allowed_depth}
                    </TableCell>
                    <TableCell>{new Date(delegation.valid_until).toLocaleString()}</TableCell>
                    <TableCell>{getStatusBadge(delegation.status)}</TableCell>
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
