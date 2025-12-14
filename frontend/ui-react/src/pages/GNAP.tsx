import React, { useState, useEffect } from 'react';
import {
    makeStyles,
    tokens,
    Card,
    Button,
    Input,
    Text,
    Badge,
    Spinner,
    Table,
    TableBody,
    TableCell,
    TableHeader,
    TableHeaderCell,
    TableRow,
    Tooltip,
} from '@fluentui/react-components';
import {
    ArrowSync24Regular,
    Delete24Regular,
    Copy24Regular,
    Key24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
    container: {
        display: 'flex',
        flexDirection: 'column',
        gap: '24px',
        padding: '24px',
    },
    card: {
        padding: '20px',
    },
    header: {
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: '16px',
    },
    grid: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
        gap: '16px',
    },
    formRow: {
        display: 'flex',
        gap: '12px',
        alignItems: 'flex-end',
        flexWrap: 'wrap',
    },
    inputGroup: {
        display: 'flex',
        flexDirection: 'column',
        gap: '4px',
        flex: 1,
        minWidth: '150px',
    },
    badges: {
        display: 'flex',
        gap: '8px',
        flexWrap: 'wrap',
    },
    tokenCode: {
        fontFamily: 'monospace',
        fontSize: '12px',
        backgroundColor: tokens.colorNeutralBackground3,
        padding: '4px 8px',
        borderRadius: '4px',
    },
    actions: {
        display: 'flex',
        gap: '4px',
    },
    error: {
        color: tokens.colorPaletteRedForeground1,
        padding: '12px',
        backgroundColor: tokens.colorPaletteRedBackground1,
        borderRadius: '4px',
    },
});

interface GNAPDiscovery {
    grant_request_endpoint: string;
    interaction_start_modes_supported: string[];
    key_proofs_supported: string[];
}

interface GNAPToken {
    value: string;
    expires_in: number;
    issued_at: string;
}

interface GNAPGrant {
    access_token?: GNAPToken;
}

const GNAP: React.FC = () => {
    const classes = useStyles();
    const [discovery, setDiscovery] = useState<GNAPDiscovery | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [grants, setGrants] = useState<GNAPGrant[]>([]);
    const [accessType, setAccessType] = useState('api');
    const [accessActions, setAccessActions] = useState('read,write');

    const fetchDiscovery = async () => {
        try {
            const response = await fetch('/.well-known/gnap-as-rs');
            if (response.ok) {
                const data = await response.json();
                setDiscovery(data);
            }
        } catch (err) {
            console.error('Failed to fetch discovery:', err);
        }
    };

    useEffect(() => {
        fetchDiscovery();
    }, []);

    const requestGrant = async () => {
        setLoading(true);
        setError(null);
        try {
            const response = await fetch('/gnap/tx', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    access_token: {
                        access: [{
                            type: accessType,
                            actions: accessActions.split(',').map(a => a.trim())
                        }]
                    }
                })
            });
            if (response.ok) {
                const grant = await response.json();
                setGrants(prev => [grant, ...prev]);
            } else {
                setError('Failed to request grant');
            }
        } catch (err) {
            setError('Network error');
        } finally {
            setLoading(false);
        }
    };

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
    };

    const revokeToken = (tokenValue: string) => {
        setGrants(prev => prev.filter(g => g.access_token?.value !== tokenValue));
    };

    return (
        <div className={classes.container}>
            <div>
                <Text size={700} weight="bold">GNAP Grant Management</Text>
                <Text size={300} style={{ display: 'block', color: tokens.colorNeutralForeground3 }}>
                    RFC 9635 - Grant Negotiation and Authorization Protocol
                </Text>
            </div>

            {/* Discovery Card */}
            <Card className={classes.card}>
                <div className={classes.header}>
                    <Text size={500} weight="semibold">Authorization Server Discovery</Text>
                    <Button icon={<ArrowSync24Regular />} appearance="subtle" onClick={fetchDiscovery} />
                </div>
                {discovery ? (
                    <div className={classes.grid}>
                        <div>
                            <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Grant Endpoint</Text>
                            <Text block>{discovery.grant_request_endpoint}</Text>
                        </div>
                        <div>
                            <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Interaction Modes</Text>
                            <div className={classes.badges}>
                                {discovery.interaction_start_modes_supported?.map(mode => (
                                    <Badge key={mode} appearance="outline">{mode}</Badge>
                                ))}
                            </div>
                        </div>
                        <div>
                            <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Key Proofs</Text>
                            <div className={classes.badges}>
                                {discovery.key_proofs_supported?.map(proof => (
                                    <Badge key={proof} color="brand">{proof}</Badge>
                                ))}
                            </div>
                        </div>
                    </div>
                ) : (
                    <Spinner size="small" label="Loading discovery..." />
                )}
            </Card>

            {/* Request Grant Card */}
            <Card className={classes.card}>
                <Text size={500} weight="semibold" block style={{ marginBottom: '16px' }}>
                    Request New Grant
                </Text>
                <div className={classes.formRow}>
                    <div className={classes.inputGroup}>
                        <Text size={200}>Access Type</Text>
                        <Input
                            value={accessType}
                            onChange={(_, data) => setAccessType(data.value)}
                        />
                    </div>
                    <div className={classes.inputGroup} style={{ flex: 2 }}>
                        <Text size={200}>Actions (comma-separated)</Text>
                        <Input
                            value={accessActions}
                            onChange={(_, data) => setAccessActions(data.value)}
                        />
                    </div>
                    <Button
                        appearance="primary"
                        icon={<Key24Regular />}
                        onClick={requestGrant}
                        disabled={loading}
                    >
                        {loading ? <Spinner size="tiny" /> : 'Request Grant'}
                    </Button>
                </div>
                {error && <div className={classes.error} style={{ marginTop: '12px' }}>{error}</div>}
            </Card>

            {/* Tokens Card */}
            <Card className={classes.card}>
                <Text size={500} weight="semibold" block style={{ marginBottom: '16px' }}>
                    Issued Tokens
                </Text>
                {grants.length === 0 ? (
                    <Text style={{ color: tokens.colorNeutralForeground3 }}>No tokens issued yet</Text>
                ) : (
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHeaderCell>Token</TableHeaderCell>
                                <TableHeaderCell>Expires In</TableHeaderCell>
                                <TableHeaderCell>Issued At</TableHeaderCell>
                                <TableHeaderCell>Actions</TableHeaderCell>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {grants.map((grant, idx) => grant.access_token && (
                                <TableRow key={idx}>
                                    <TableCell>
                                        <span className={classes.tokenCode}>
                                            {grant.access_token.value.substring(0, 20)}...
                                        </span>
                                    </TableCell>
                                    <TableCell>{grant.access_token.expires_in}s</TableCell>
                                    <TableCell>
                                        {new Date(grant.access_token.issued_at).toLocaleTimeString()}
                                    </TableCell>
                                    <TableCell>
                                        <div className={classes.actions}>
                                            <Tooltip content="Copy token" relationship="label">
                                                <Button
                                                    icon={<Copy24Regular />}
                                                    appearance="subtle"
                                                    size="small"
                                                    onClick={() => copyToClipboard(grant.access_token!.value)}
                                                />
                                            </Tooltip>
                                            <Tooltip content="Revoke token" relationship="label">
                                                <Button
                                                    icon={<Delete24Regular />}
                                                    appearance="subtle"
                                                    size="small"
                                                    onClick={() => revokeToken(grant.access_token!.value)}
                                                />
                                            </Tooltip>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </Card>
        </div>
    );
};

export default GNAP;
