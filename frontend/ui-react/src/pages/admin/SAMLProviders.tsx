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
    DialogTrigger,
    Input,
    Field,
    Textarea,
    Switch,
    Text,
    Title3,
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
    Edit24Regular,
    MoreHorizontal24Regular,
    Dismiss24Regular,
    CloudCheckmark24Regular,
    Shield24Regular
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
        maxWidth: '800px',
        width: '90vw',
    },
    form: {
        display: 'flex',
        flexDirection: 'column',
        ...shorthands.gap('16px'),
    },
    formRow: {
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        ...shorthands.gap('16px'),
    },
    badge: {
        textTransform: 'capitalize',
    },
    loading: {
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '200px',
    },
    actions: {
        display: 'flex',
        ...shorthands.gap('8px'),
    },
});

interface SAMLProvider {
    id: string;
    tenantId: string;
    providerName: string;
    displayName: string;
    entityId: string;
    ssoUrl: string;
    certificate: string;
    attributeMapping: string; // JSON string
    status: string;
    createdAt?: string;
}

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export default function SAMLProviders() {
    const classes = useStyles();
    const [providers, setProviders] = useState<SAMLProvider[]>([]);
    const [loading, setLoading] = useState(true);
    const [dialogOpen, setDialogOpen] = useState(false);
    const [editingProvider, setEditingProvider] = useState<SAMLProvider | null>(null);
    const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

    // Form State
    const [formData, setFormData] = useState<Partial<SAMLProvider>>({
        providerName: '',
        displayName: '',
        entityId: '',
        ssoUrl: '',
        certificate: '',
        attributeMapping: '{}',
        status: 'active',
    });

    const fetchProviders = async () => {
        setLoading(true);
        try {
            // Stubbing the tenant ID for now or getting from context
            const response = await fetch(`${apiBaseUrl}/saml/providers`, {
                headers: {
                    'X-Tenant-ID': 'default', // Replace with actual tenant context
                    'Authorization': `Bearer ${localStorage.getItem('admin_token')}`
                }
            });
            if (response.ok) {
                const data = await response.json();
                setProviders(data || []);
            } else {
                // Fallback or error handling
                console.error("Failed to fetch SAML providers");
                setProviders([]);
            }
        } catch (err) {
            console.error(err);
            setMessage({ type: 'error', text: 'Failed to load SAML providers' });
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchProviders();
    }, []);

    const handleSave = async () => {
        try {
            const method = editingProvider ? 'PUT' : 'POST';
            const url = editingProvider
                ? `${apiBaseUrl}/saml/providers/${editingProvider.id}`
                : `${apiBaseUrl}/saml/providers`;

            const body = {
                ...formData,
                tenantId: 'default', // Ensure tenant ID is set
            };

            const response = await fetch(url, {
                method,
                headers: {
                    'Content-Type': 'application/json',
                    'X-Tenant-ID': 'default',
                    'Authorization': `Bearer ${localStorage.getItem('admin_token')}`
                },
                body: JSON.stringify(body),
            });

            if (response.ok) {
                setMessage({ type: 'success', text: `Provider ${editingProvider ? 'updated' : 'created'} successfully` });
                setDialogOpen(false);
                fetchProviders();
            } else {
                const error = await response.json();
                setMessage({ type: 'error', text: error.error || 'Operation failed' });
            }
        } catch (err) {
            setMessage({ type: 'error', text: 'Network error occurred' });
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure you want to delete this provider?')) return;
        try {
            const response = await fetch(`${apiBaseUrl}/saml/providers/${id}`, {
                method: 'DELETE',
                headers: {
                    'X-Tenant-ID': 'default',
                    'Authorization': `Bearer ${localStorage.getItem('admin_token')}`
                }
            });
            if (response.ok) {
                setMessage({ type: 'success', text: 'Provider deleted successfully' });
                fetchProviders();
            } else {
                setMessage({ type: 'error', text: 'Failed to delete provider' });
            }
        } catch (err) {
            setMessage({ type: 'error', text: 'Network error' });
        }
    };

    const openEdit = (provider: SAMLProvider) => {
        setEditingProvider(provider);
        setFormData(provider);
        setDialogOpen(true);
    };

    const openCreate = () => {
        setEditingProvider(null);
        setFormData({
            providerName: '',
            displayName: '',
            entityId: '',
            ssoUrl: '',
            certificate: '',
            attributeMapping: '{\n  "email": "email",\n  "firstName": "givenName",\n  "lastName": "sn"\n}',
            status: 'active',
        });
        setDialogOpen(true);
    };

    const columns: TableColumnDefinition<SAMLProvider>[] = [
        createTableColumn({
            columnId: 'displayName',
            compare: (a, b) => a.displayName.localeCompare(b.displayName),
            renderHeaderCell: () => 'Display Name',
            renderCell: (item) => (
                <TableCellLayout media={<Shield24Regular />}>
                    {item.displayName}
                    <div style={{ fontSize: '12px', color: tokens.colorNeutralForeground2 }}>{item.providerName}</div>
                </TableCellLayout>
            ),
        }),
        createTableColumn({
            columnId: 'entityId',
            renderHeaderCell: () => 'Entity ID (Issuer)',
            renderCell: (item) => item.entityId,
        }),
        createTableColumn({
            columnId: 'status',
            renderHeaderCell: () => 'Status',
            renderCell: (item) => (
                <Badge
                    appearance="tint"
                    color={item.status === 'active' ? 'success' : 'warning'}
                    className={classes.badge}
                >
                    {item.status}
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
                            <MenuItem icon={<Edit24Regular />} onClick={() => openEdit(item)}>Edit</MenuItem>
                            <MenuItem icon={<CloudCheckmark24Regular />} onClick={() => window.open(`${apiBaseUrl}/saml/metadata/${item.providerName}`, '_blank')}>View Metadata</MenuItem>
                            <MenuItem icon={<Delete24Regular />} onClick={() => handleDelete(item.id)}>Delete</MenuItem>
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
                    <Title3>SAML Identity Providers</Title3>
                    <Text block style={{ color: tokens.colorNeutralForeground2 }}>
                        Configure external SAML 2.0 Identity Providers (IdP) for single sign-on.
                    </Text>
                </div>
                <Button appearance="primary" icon={<Add24Regular />} onClick={openCreate}>
                    New Provider
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

            {loading ? (
                <div className={classes.loading}><Spinner label="Loading providers..." /></div>
            ) : (
                <div className={classes.grid}>
                    <DataGrid
                        items={providers}
                        columns={columns}
                        getRowId={(item) => item.id}
                    >
                        <DataGridHeader>
                            <DataGridRow>
                                {({ renderHeaderCell }) => <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>}
                            </DataGridRow>
                        </DataGridHeader>
                        <DataGridBody<SAMLProvider>>
                            {({ item, rowId }) => (
                                <DataGridRow<SAMLProvider> key={rowId}>
                                    {({ renderCell }) => <DataGridCell>{renderCell(item)}</DataGridCell>}
                                </DataGridRow>
                            )}
                        </DataGridBody>
                    </DataGrid>
                </div>
            )}

            <Dialog open={dialogOpen} onOpenChange={(_, data) => setDialogOpen(data.open)}>
                <DialogSurface className={classes.dialog}>
                    <DialogBody>
                        <DialogTitle>{editingProvider ? 'Edit SAML Provider' : 'New SAML Provider'}</DialogTitle>
                        <DialogContent className={classes.form}>
                            <div className={classes.formRow}>
                                <Field label="Provider Name (ID)" required hint="Unique identifier (e.g. okta-corporate)">
                                    <Input value={formData.providerName} onChange={(_e, d) => setFormData({ ...formData, providerName: d.value })} disabled={!!editingProvider} />
                                </Field>
                                <Field label="Display Name" required>
                                    <Input value={formData.displayName} onChange={(_e, d) => setFormData({ ...formData, displayName: d.value })} />
                                </Field>
                            </div>

                            <Field label="Entity ID (Issuer URL)" required hint="The Entity ID from your IdP metadata">
                                <Input value={formData.entityId} onChange={(_e, d) => setFormData({ ...formData, entityId: d.value })} />
                            </Field>

                            <Field label="SSO Service URL" required hint="The HTTP-POST Single Sign-On URL">
                                <Input value={formData.ssoUrl} onChange={(_e, d) => setFormData({ ...formData, ssoUrl: d.value })} />
                            </Field>

                            <Field label="X.509 Certificate" required hint="PEM encoded certificate from IdP">
                                <Textarea value={formData.certificate} onChange={(_e, d) => setFormData({ ...formData, certificate: d.value })} rows={5} resize="vertical" style={{ fontFamily: 'monospace' }} />
                            </Field>

                            <Field label="Attribute Mapping (JSON)" hint="Map SAML attributes to user fields">
                                <Textarea value={formData.attributeMapping} onChange={(_e, d) => setFormData({ ...formData, attributeMapping: d.value })} rows={5} resize="vertical" style={{ fontFamily: 'monospace' }} />
                            </Field>

                            <Field label="Status">
                                <Switch checked={formData.status === 'active'} onChange={(_e, d) => setFormData({ ...formData, status: d.checked ? 'active' : 'inactive' })} label={formData.status === 'active' ? 'Active' : 'Inactive'} />
                            </Field>

                        </DialogContent>
                        <DialogActions>
                            <DialogTrigger disableButtonEnhancement>
                                <Button appearance="secondary" onClick={() => setDialogOpen(false)}>Cancel</Button>
                            </DialogTrigger>
                            <Button appearance="primary" onClick={handleSave}>Save</Button>
                        </DialogActions>
                    </DialogBody>
                </DialogSurface>
            </Dialog>
        </div>
    );
}
