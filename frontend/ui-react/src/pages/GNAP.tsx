import React, { useState, useEffect } from 'react';
import {
    Box,
    Typography,
    Card,
    CardContent,
    Button,
    TextField,
    Grid,
    Chip,
    Alert,
    CircularProgress,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Paper,
    IconButton,
    Tooltip,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import DeleteIcon from '@mui/icons-material/Delete';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';

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
    continue?: {
        uri: string;
        access_token?: { value: string };
    };
    access_token?: GNAPToken;
}

const GNAP: React.FC = () => {
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

    const revokeToken = async (tokenValue: string) => {
        // TODO: Implement token revocation
        setGrants(prev => prev.filter(g => g.access_token?.value !== tokenValue));
    };

    return (
        <Box sx={{ p: 3 }}>
            <Typography variant="h4" gutterBottom>
                GNAP Grant Management
            </Typography>
            <Typography variant="subtitle1" color="text.secondary" gutterBottom>
                RFC 9635 - Grant Negotiation and Authorization Protocol
            </Typography>

            {/* Discovery Info */}
            <Card sx={{ mb: 3 }}>
                <CardContent>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                        <Typography variant="h6">Authorization Server Discovery</Typography>
                        <IconButton onClick={fetchDiscovery} size="small">
                            <RefreshIcon />
                        </IconButton>
                    </Box>
                    {discovery ? (
                        <Grid container spacing={2}>
                            <Grid item xs={12}>
                                <Typography variant="body2" color="text.secondary">Grant Endpoint</Typography>
                                <Typography variant="body1">{discovery.grant_request_endpoint}</Typography>
                            </Grid>
                            <Grid item xs={12} sm={6}>
                                <Typography variant="body2" color="text.secondary">Interaction Modes</Typography>
                                <Box sx={{ mt: 1 }}>
                                    {discovery.interaction_start_modes_supported?.map(mode => (
                                        <Chip key={mode} label={mode} size="small" sx={{ mr: 0.5, mb: 0.5 }} />
                                    ))}
                                </Box>
                            </Grid>
                            <Grid item xs={12} sm={6}>
                                <Typography variant="body2" color="text.secondary">Key Proofs</Typography>
                                <Box sx={{ mt: 1 }}>
                                    {discovery.key_proofs_supported?.map(proof => (
                                        <Chip key={proof} label={proof} size="small" color="primary" sx={{ mr: 0.5 }} />
                                    ))}
                                </Box>
                            </Grid>
                        </Grid>
                    ) : (
                        <Typography color="text.secondary">Loading discovery...</Typography>
                    )}
                </CardContent>
            </Card>

            {/* Request Grant Form */}
            <Card sx={{ mb: 3 }}>
                <CardContent>
                    <Typography variant="h6" gutterBottom>Request New Grant</Typography>
                    <Grid container spacing={2} alignItems="center">
                        <Grid item xs={12} sm={3}>
                            <TextField
                                fullWidth
                                label="Access Type"
                                value={accessType}
                                onChange={(e) => setAccessType(e.target.value)}
                                size="small"
                            />
                        </Grid>
                        <Grid item xs={12} sm={6}>
                            <TextField
                                fullWidth
                                label="Actions (comma-separated)"
                                value={accessActions}
                                onChange={(e) => setAccessActions(e.target.value)}
                                size="small"
                            />
                        </Grid>
                        <Grid item xs={12} sm={3}>
                            <Button
                                variant="contained"
                                fullWidth
                                onClick={requestGrant}
                                disabled={loading}
                            >
                                {loading ? <CircularProgress size={20} /> : 'Request Grant'}
                            </Button>
                        </Grid>
                    </Grid>
                    {error && <Alert severity="error" sx={{ mt: 2 }}>{error}</Alert>}
                </CardContent>
            </Card>

            {/* Active Tokens */}
            <Card>
                <CardContent>
                    <Typography variant="h6" gutterBottom>Issued Tokens</Typography>
                    {grants.length === 0 ? (
                        <Typography color="text.secondary">No tokens issued yet</Typography>
                    ) : (
                        <TableContainer component={Paper} variant="outlined">
                            <Table size="small">
                                <TableHead>
                                    <TableRow>
                                        <TableCell>Token</TableCell>
                                        <TableCell>Expires In</TableCell>
                                        <TableCell>Issued At</TableCell>
                                        <TableCell align="right">Actions</TableCell>
                                    </TableRow>
                                </TableHead>
                                <TableBody>
                                    {grants.map((grant, idx) => grant.access_token && (
                                        <TableRow key={idx}>
                                            <TableCell>
                                                <code style={{ fontSize: '0.8rem' }}>
                                                    {grant.access_token.value.substring(0, 20)}...
                                                </code>
                                                <Tooltip title="Copy token">
                                                    <IconButton
                                                        size="small"
                                                        onClick={() => copyToClipboard(grant.access_token!.value)}
                                                    >
                                                        <ContentCopyIcon fontSize="small" />
                                                    </IconButton>
                                                </Tooltip>
                                            </TableCell>
                                            <TableCell>{grant.access_token.expires_in}s</TableCell>
                                            <TableCell>
                                                {new Date(grant.access_token.issued_at).toLocaleTimeString()}
                                            </TableCell>
                                            <TableCell align="right">
                                                <Tooltip title="Revoke token">
                                                    <IconButton
                                                        size="small"
                                                        color="error"
                                                        onClick={() => revokeToken(grant.access_token!.value)}
                                                    >
                                                        <DeleteIcon fontSize="small" />
                                                    </IconButton>
                                                </Tooltip>
                                            </TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        </TableContainer>
                    )}
                </CardContent>
            </Card>
        </Box>
    );
};

export default GNAP;
