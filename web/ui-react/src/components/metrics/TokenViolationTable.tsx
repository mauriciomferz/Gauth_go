import { useEffect, useState } from 'react';
import {
  makeStyles,
  tokens,
  Card,
  Text,
  Title3,
  Badge,
  Button,
  DataGrid,
  DataGridBody,
  DataGridRow,
  DataGridHeader,
  DataGridHeaderCell,
  DataGridCell,
  TableCellLayout,
  TableColumnDefinition,
  createTableColumn,
  Menu,
  MenuTrigger,
  MenuPopover,
  MenuList,
  MenuItem,
} from '@fluentui/react-components';
import {
  Warning24Regular,
  MoreVertical20Regular,
  DocumentSearch24Regular,
  Dismiss24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  title: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  card: {
    padding: '20px',
  },
  dataGridContainer: {
    overflowX: 'auto',
  },
  statusBadge: {
    minWidth: '80px',
  },
});

interface TokenViolation {
  id: string;
  subscriber: string;
  subscriberId: string;
  violationType: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  timestamp: string;
  reason: string;
  tokenId: string;
  resolved: boolean;
}

export default function TokenViolationTable() {
  const classes = useStyles();
  const [violations, setViolations] = useState<TokenViolation[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchViolations();
    const interval = setInterval(fetchViolations, 60000); // Refresh every minute
    return () => clearInterval(interval);
  }, []);

  const fetchViolations = async () => {
    try {
      const response = await fetch('/api/admin/metrics/token-violations', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        setViolations(data.violations || []);
      }
    } catch (error) {
      console.error('Failed to fetch token violations:', error);
    } finally {
      setLoading(false);
    }
  };

  const getSeverityBadge = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <Badge appearance="filled" color="danger">Critical</Badge>;
      case 'high':
        return <Badge appearance="filled" color="warning">High</Badge>;
      case 'medium':
        return <Badge appearance="filled" color="informative">Medium</Badge>;
      case 'low':
        return <Badge appearance="tint" color="subtle">Low</Badge>;
      default:
        return <Badge>{severity}</Badge>;
    }
  };

  const handleResolve = async (violationId: string) => {
    try {
      await fetch(`/api/admin/metrics/token-violations/${violationId}/resolve`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });
      fetchViolations();
    } catch (error) {
      console.error('Failed to resolve violation:', error);
    }
  };

  const handleViewDetails = (violationId: string) => {
    // Navigate to violation details page
    console.log('View details for violation:', violationId);
  };

  const columns: TableColumnDefinition<TokenViolation>[] = [
    createTableColumn<TokenViolation>({
      columnId: 'subscriber',
      compare: (a, b) => a.subscriber.localeCompare(b.subscriber),
      renderHeaderCell: () => 'Subscriber',
      renderCell: (item) => (
        <TableCellLayout>
          <Text weight="semibold">{item.subscriber}</Text>
          <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
            {item.subscriberId}
          </Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<TokenViolation>({
      columnId: 'violationType',
      compare: (a, b) => a.violationType.localeCompare(b.violationType),
      renderHeaderCell: () => 'Violation Type',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{item.violationType}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<TokenViolation>({
      columnId: 'severity',
      compare: (a, b) => {
        const order = { critical: 0, high: 1, medium: 2, low: 3 };
        return order[a.severity] - order[b.severity];
      },
      renderHeaderCell: () => 'Severity',
      renderCell: (item) => (
        <TableCellLayout className={classes.statusBadge}>
          {getSeverityBadge(item.severity)}
        </TableCellLayout>
      ),
    }),
    createTableColumn<TokenViolation>({
      columnId: 'timestamp',
      compare: (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
      renderHeaderCell: () => 'Timestamp',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{new Date(item.timestamp).toLocaleString()}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<TokenViolation>({
      columnId: 'reason',
      renderHeaderCell: () => 'Reason',
      renderCell: (item) => (
        <TableCellLayout truncate title={item.reason}>
          <Text>{item.reason}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<TokenViolation>({
      columnId: 'status',
      renderHeaderCell: () => 'Status',
      renderCell: (item) => (
        <TableCellLayout>
          {item.resolved ? (
            <Badge appearance="tint" color="success">Resolved</Badge>
          ) : (
            <Badge appearance="filled" color="warning">Active</Badge>
          )}
        </TableCellLayout>
      ),
    }),
    createTableColumn<TokenViolation>({
      columnId: 'actions',
      renderHeaderCell: () => 'Actions',
      renderCell: (item) => (
        <TableCellLayout>
          <Menu>
            <MenuTrigger>
              <Button
                appearance="subtle"
                icon={<MoreVertical20Regular />}
                size="small"
              />
            </MenuTrigger>
            <MenuPopover>
              <MenuList>
                <MenuItem
                  icon={<DocumentSearch24Regular />}
                  onClick={() => handleViewDetails(item.id)}
                >
                  View Details
                </MenuItem>
                {!item.resolved && (
                  <MenuItem
                    icon={<Dismiss24Regular />}
                    onClick={() => handleResolve(item.id)}
                  >
                    Mark as Resolved
                  </MenuItem>
                )}
              </MenuList>
            </MenuPopover>
          </Menu>
        </TableCellLayout>
      ),
    }),
  ];

  return (
    <div className={classes.container}>
      <div className={classes.header}>
        <div className={classes.title}>
          <Warning24Regular style={{ fontSize: '24px' }} />
          <Title3>Token Violation Metrics</Title3>
        </div>
        <Button appearance="secondary" onClick={fetchViolations}>
          Refresh
        </Button>
      </div>

      <Card className={classes.card}>
        <div className={classes.dataGridContainer}>
          {loading ? (
            <Text>Loading violations...</Text>
          ) : violations.length === 0 ? (
            <div style={{ padding: '40px', textAlign: 'center' }}>
              <Text size={400}>No token violations detected</Text>
              <Text size={200} style={{ display: 'block', marginTop: '8px', color: tokens.colorNeutralForeground3 }}>
                All tokens are compliant with security policies
              </Text>
            </div>
          ) : (
            <DataGrid
              items={violations}
              columns={columns}
              sortable
              resizableColumns
              size="small"
            >
              <DataGridHeader>
                <DataGridRow>
                  {({ renderHeaderCell }) => (
                    <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>
                  )}
                </DataGridRow>
              </DataGridHeader>
              <DataGridBody<TokenViolation>>
                {({ item, rowId }) => (
                  <DataGridRow<TokenViolation> key={rowId}>
                    {({ renderCell }) => (
                      <DataGridCell>{renderCell(item)}</DataGridCell>
                    )}
                  </DataGridRow>
                )}
              </DataGridBody>
            </DataGrid>
          )}
        </div>
      </Card>
    </div>
  );
}
