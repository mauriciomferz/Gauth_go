import { useState, useEffect } from 'react';
import {
    makeStyles,
    shorthands,
    tokens,
    Button,
    Dialog,
    DialogSurface,
    DialogTitle,
    DialogBody,
    DialogActions,
    DialogContent,
    Input,
    Field,
    Title3,
    Text,
    DataGrid,
    DataGridHeader,
    DataGridRow,
    DataGridHeaderCell,
    DataGridBody,
    DataGridCell,
    createTableColumn,
    TableCellLayout,
    TableColumnDefinition,
    Badge,
    Menu,
    MenuTrigger,
    MenuPopover,
    MenuList,
    MenuItem,
    MessageBar,
    MessageBarBody,
    MessageBarActions,
    Spinner,
} from '@fluentui/react-components';
import {
    Add24Regular,
    Delete24Regular,
    MoreHorizontal24Regular,
    Copy24Regular,
    PeopleTeam24Regular,
    Dismiss24Regular
} from '@fluentui/react-icons';

const useStyles = makeStyles({
    container: {
        display: 'flex',
        flexDirection: 'column',
        ...shorthands.gap('20px'),
        ...shorthands.padding('24px'),
    },
    header: {
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
    },
    grid: {
        width: '100%',
    },
    dialog: {
        maxWidth: '500px',
        width: '90vw',
    },
    form: {
        display: 'flex',
        flexDirection: 'column',
        ...shorthands.gap('16px'),
    },
    loading: {
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '200px',
    },
    codeBlock: {
        backgroundColor: tokens.colorNeutralBackground2,
        padding: '12px',
        borderRadius: '4px',
        fontFamily: 'monospace',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
    }
});

interface SCIMClient {
    id: string;
    clientName: string;
    tokenId: string; // ID of the token
    isActive: boolean;
    createdAt: string;
}

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export default function SCIMSettings() {
    const classes = useStyles();
    const [clients, setClients] = useState<SCIMClient[]>([]);
    const [loading, setLoading] = useState(true);
    const [dialogOpen, setDialogOpen] = useState(false);
    const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

    const [formData, setFormData] = useState({
        clientName: '',
    });

    const fetchClients = async () => {
        setLoading(true);
        try {
            const response = await fetch(`${apiBaseUrl}/admin/scim/clients`, {
                headers: {
                    'X-Tenant-ID': 'default',
                    'Authorization': `Bearer ${localStorage.getItem('admin_token')}`
                }
            });
            if (response.ok) {
                const data = await response.json();
                setClients(data || []);
            } else {
                setClients([]);
            }
        } catch (err) {
            console.error(err);
            setClients([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchClients();
    }, []);

    const handleCreate = async () => {
        try {
            const response = await fetch(`${apiBaseUrl}/admin/scim/clients`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Tenant-ID': 'default',
                    'Authorization': `Bearer ${localStorage.getItem('admin_token')}`
                },
                body: JSON.stringify({ clientName: formData.clientName }),
            });

            if (response.ok) {
                await response.json();
                setMessage({ type: 'success', text: 'SCIM Client created successfully' });
                setDialogOpen(false);
                fetchClients(); // Reload list
                // Optionally show the token to the user here. For now it is in the list or response.
            } else {
                const err = await response.json();
                setMessage({ type: 'error', text: err.error || 'Failed to create client' });
            }
        } catch (_err) {
            setMessage({ type: 'error', text: 'Network error' });
        }
    };

    const handleRevoke = async (id: string) => {
        if (!confirm('Are you sure you want to revoke this client?')) return;
        try {
            const response = await fetch(`${apiBaseUrl}/admin/scim/clients/${id}`, {
                method: 'DELETE',
                headers: {
                    'X-Tenant-ID': 'default',
                    'Authorization': `Bearer ${localStorage.getItem('admin_token')}`
                }
            });
            if (response.ok) {
                setMessage({ type: 'success', text: 'Client revoked' });
                fetchClients();
            } else {
                setMessage({ type: 'error', text: 'Failed to revoke client' });
            }
        } catch (_err) {
            setMessage({ type: 'error', text: 'Network error' });
        }
    };

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        setMessage({ type: 'success', text: 'Copied to clipboard' });
    };

    const columns: TableColumnDefinition<SCIMClient>[] = [
        createTableColumn({
            columnId: 'clientName',
            compare: (a, b) => a.clientName.localeCompare(b.clientName),
            renderHeaderCell: () => 'Client Name',
            renderCell: (item) => (
                <TableCellLayout media={<PeopleTeam24Regular />}>
                    {item.clientName}
                </TableCellLayout>
            ),
        }),
        createTableColumn({
            columnId: 'status',
            renderHeaderCell: () => 'Status',
            renderCell: (item) => (
                <Badge appearance="tint" color={item.isActive ? 'success' : 'important'}>
                    {item.isActive ? 'Active' : 'Inactive'}
                </Badge>
            ),
        }),
        createTableColumn({
            columnId: 'actions',
            renderHeaderCell: () => 'Actions',
            renderCell: (item) => (
                <Menu>
                    <MenuTrigger disableButtonEnhancement>
                        <Button appearance="transparent" icon={<MoreHorizontal24Regular />} />
                    </MenuTrigger>
                    <MenuPopover>
                        <MenuList>
                            <MenuItem icon={<Delete24Regular />} onClick={() => handleRevoke(item.id)}>Revoke (Delete)</MenuItem>
                        </MenuList>
                    </MenuPopover>
                </Menu>
            ),
        }),
    ];

    return (
        <div className={classes.container}>
            <div className={classes.header}>
                <div>
                    <Title3>SCIM User Provisioning</Title3>
                    <Text block style={{ color: tokens.colorNeutralForeground2 }}>
                        Manage SCIM clients and view connection details.
                    </Text>
                </div>
                <Button appearance="primary" icon={<Add24Regular />} onClick={() => setDialogOpen(true)}>
                    New Client
                </Button>
            </div>

            {message && (
                <MessageBar intent={message.type}>
                    <MessageBarBody>
                        <MessageBarActions containerAction={<Button appearance="transparent" icon={<Dismiss24Regular />} onClick={() => setMessage(null)} />} />
                        {message.text}
                    </MessageBarBody>
                </MessageBar>
            )}

            <div className={classes.codeBlock}>
                <div>
                    <Text weight="semibold">SCIM Base URL:</Text><br />
                    <Text style={{ fontSize: '12px' }}>{window.location.origin}/api/scim/v2</Text>
                </div>
                <Button icon={<Copy24Regular />} appearance="subtle" onClick={() => copyToClipboard(`${window.location.origin}/api/scim/v2`)} />
            </div>

            {loading ? (
                <div className={classes.loading}><Spinner label="Loading clients..." /></div>
            ) : (
                <div className={classes.grid}>
                    {clients.length === 0 ? (
                        <div style={{ padding: '40px', textAlign: 'center', color: tokens.colorNeutralForeground2 }}>
                            No SCIM clients configured. Create one to enable provisioning.
                        </div>
                    ) : (
                        <DataGrid items={clients} columns={columns} getRowId={(item) => item.id}>
                            <DataGridHeader>
                                <DataGridRow>{({ renderHeaderCell }) => <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>}</DataGridRow>
                            </DataGridHeader>
                            <DataGridBody<SCIMClient>>
                                {({ item, rowId }) => <DataGridRow<SCIMClient> key={rowId}>{({ renderCell }) => <DataGridCell>{renderCell(item)}</DataGridCell>}</DataGridRow>}
                            </DataGridBody>
                        </DataGrid>
                    )}
                </div>
            )}

            <Dialog open={dialogOpen} onOpenChange={(_, data) => setDialogOpen(data.open)}>
                <DialogSurface className={classes.dialog}>
                    <DialogBody>
                        <DialogTitle>New SCIM Client</DialogTitle>
                        <DialogContent className={classes.form}>
                            <Field label="Client Name" required hint="Identifier for the provisioning system (e.g. Azure AD)">
                                <Input value={formData.clientName} onChange={(_e, d) => setFormData({ ...formData, clientName: d.value })} />
                            </Field>
                            <MessageBar intent="info">
                                <MessageBarBody>
                                    Generating a client will create a persistent token.
                                </MessageBarBody>
                            </MessageBar>
                        </DialogContent>
                        <DialogActions>
                            <Button appearance="secondary" onClick={() => setDialogOpen(false)}>Cancel</Button>
                            <Button appearance="primary" onClick={handleCreate}>Generate</Button>
                        </DialogActions>
                    </DialogBody>
                </DialogSurface>
            </Dialog>
        </div>
    );
}
