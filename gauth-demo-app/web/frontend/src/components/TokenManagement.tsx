import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
  IconButton,
  Chip,
  Alert,
  Grid,
  CircularProgress,
  Tooltip,
  FormControl,
  InputLabel,
  Select,
  OutlinedInput,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Refresh as RefreshIcon,
  Visibility as ViewIcon,
  Security as SecurityIcon,
  Schedule as ScheduleIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { DataGrid, GridColDef, GridActionsCellItem } from '@mui/x-data-grid';
import { motion } from 'framer-motion';
import { useForm, Controller } from 'react-hook-form';
import { yupResolver } from '@hookform/resolvers/yup';
import * as yup from 'yup';
import toast from 'react-hot-toast';
import { format, parseISO } from 'date-fns';
import apiService from '../services/apiService';
import { useAppStore } from '../store/appStore';

interface Token {
  id: string;
  owner_id: string;
  client_id: string;
  scope: string[];
  claims: Record<string, any>;
  created_at: string;
  expires_at: string;
  valid: boolean;
  status: 'active' | 'expired' | 'revoked';
}

interface TokenFormData {
  owner_id: string;
  client_id: string;
  scope: string[];
  duration: string;
  claims: Record<string, any>;
}

const tokenSchema = yup.object({
  owner_id: yup.string().required('Owner ID is required'),
  client_id: yup.string().required('Client ID is required'),
  scope: yup.array().min(1, 'At least one scope is required'),
  duration: yup.string().required('Duration is required'),
  claims: yup.object().default({}),
});

const scopeOptions = [
  'read', 'write', 'admin', 'user', 'transaction:execute', 
  'audit:read', 'compliance:check', 'token:manage'
];

const durationOptions = [
  { value: '1h', label: '1 Hour' },
  { value: '24h', label: '24 Hours' },
  { value: '7d', label: '7 Days' },
  { value: '30d', label: '30 Days' },
  { value: '90d', label: '90 Days' },
];

export default function TokenManagement() {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [selectedToken, setSelectedToken] = useState<Token | null>(null);
  const [viewDialogOpen, setViewDialogOpen] = useState(false);
  const [customClaims, setCustomClaims] = useState<Array<{ key: string; value: string }>>([]);

  const { isLoading, setLoading: setAppLoading } = useAppStore();

  const { control, handleSubmit, reset, formState: { errors } } = useForm<TokenFormData>({
    resolver: yupResolver(tokenSchema),
    defaultValues: {
      owner_id: '',
      client_id: '',
      scope: [],
      duration: '1h',
      claims: {},
    },
  });

  useEffect(() => {
    loadTokens();
  }, []);

  const loadTokens = async () => {
    try {
      setAppLoading('tokens', true);
      const response = await apiService.getTokens();
      setTokens(response.tokens || []);
    } catch (error) {
      toast.error('Failed to load tokens');
      console.error('Error loading tokens:', error);
    } finally {
      setLoading(false);
      setAppLoading('tokens', false);
    }
  };

  const handleCreateToken = async (data: TokenFormData) => {
    try {
      setAppLoading('createToken', true);
      
      // Validate required fields
      if (!data.owner_id) {
        toast.error('Owner ID is required');
        return;
      }
      
      if (!data.client_id) {
        toast.error('Client ID is required');
        return;
      }
      
      // Convert custom claims to object
      const claims = { ...data.claims };
      customClaims.forEach(claim => {
        if (claim.key && claim.value) {
          claims[claim.key] = claim.value;
        }
      });

      const tokenData = {
        claims: {
          sub: data.owner_id,
          client_id: data.client_id,
          ...claims,
        },
        duration: data.duration || '1h', // Default to 1 hour
        scope: data.scope || [],
      };

      console.log('Creating token with data:', tokenData);
      await apiService.createToken(tokenData);
      toast.success('Token created successfully');
      
      setCreateDialogOpen(false);
      reset();
      setCustomClaims([]);
      loadTokens();
    } catch (error: any) {
      const errorMessage = error?.response?.data?.details || error?.response?.data?.error || error?.message || 'Failed to create token';
      toast.error(`Token creation failed: ${errorMessage}`);
      console.error('Error creating token:', error);
      console.error('Error response:', error?.response?.data);
    } finally {
      setAppLoading('createToken', false);
    }
  };

  const handleRevokeToken = async (tokenId: string) => {
    try {
      await apiService.revokeToken(tokenId);
      toast.success('Token revoked successfully');
      loadTokens();
    } catch (error) {
      toast.error('Failed to revoke token');
      console.error('Error revoking token:', error);
    }
  };

  const handleViewToken = (token: Token) => {
    setSelectedToken(token);
    setViewDialogOpen(true);
  };

  const addCustomClaim = () => {
    setCustomClaims([...customClaims, { key: '', value: '' }]);
  };

  const updateCustomClaim = (index: number, field: 'key' | 'value', value: string) => {
    const newClaims = [...customClaims];
    newClaims[index][field] = value;
    setCustomClaims(newClaims);
  };

  const removeCustomClaim = (index: number) => {
    setCustomClaims(customClaims.filter((_, i) => i !== index));
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'success';
      case 'expired': return 'warning';
      case 'revoked': return 'error';
      default: return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active': return <CheckCircleIcon />;
      case 'expired': return <WarningIcon />;
      case 'revoked': return <ErrorIcon />;
      default: return <SecurityIcon />;
    }
  };

  const columns: GridColDef[] = [
    {
      field: 'status',
      headerName: 'Status',
      width: 120,
      renderCell: (params) => (
        <Chip
          icon={getStatusIcon(params.value)}
          label={params.value}
          color={getStatusColor(params.value) as any}
          size="small"
        />
      ),
    },
    { field: 'owner_id', headerName: 'Owner', width: 150 },
    { field: 'client_id', headerName: 'Client', width: 150 },
    {
      field: 'scope',
      headerName: 'Scope',
      width: 200,
      renderCell: (params) => (
        <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
          {params.value?.slice(0, 2).map((scope: string) => (
            <Chip key={scope} label={scope} size="small" variant="outlined" />
          ))}
          {params.value?.length > 2 && (
            <Chip label={`+${params.value.length - 2}`} size="small" />
          )}
        </Box>
      ),
    },
    {
      field: 'created_at',
      headerName: 'Created',
      width: 150,
      renderCell: (params) => format(parseISO(params.value), 'MMM dd, yyyy'),
    },
    {
      field: 'expires_at',
      headerName: 'Expires',
      width: 150,
      renderCell: (params) => format(parseISO(params.value), 'MMM dd, yyyy'),
    },
    {
      field: 'actions',
      type: 'actions',
      headerName: 'Actions',
      width: 120,
      getActions: (params) => [
        <GridActionsCellItem
          key="view"
          icon={
            <Tooltip title="View Details">
              <ViewIcon />
            </Tooltip>
          }
          label="View"
          onClick={() => handleViewToken(params.row)}
        />,
        <GridActionsCellItem
          key="revoke"
          icon={
            <Tooltip title="Revoke Token">
              <DeleteIcon />
            </Tooltip>
          }
          label="Revoke"
          onClick={() => handleRevokeToken(params.row.id)}
          disabled={params.row.status !== 'active'}
        />,
      ],
    },
  ];

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      <Box sx={{ p: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" component="h1" sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <SecurityIcon fontSize="large" color="primary" />
            Token Management
          </Typography>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={loadTokens}
              disabled={isLoading('tokens')}
            >
              Refresh
            </Button>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={() => setCreateDialogOpen(true)}
            >
              Create Token
            </Button>
          </Box>
        </Box>

        <Card>
          <CardContent>
            <Box sx={{ height: 600, width: '100%' }}>
              <DataGrid
                rows={tokens}
                columns={columns}
                loading={loading}
                pageSizeOptions={[10, 25, 50]}
                initialState={{
                  pagination: {
                    paginationModel: { page: 0, pageSize: 10 },
                  },
                }}
                disableRowSelectionOnClick
                sx={{
                  '& .MuiDataGrid-row:hover': {
                    backgroundColor: 'action.hover',
                  },
                }}
              />
            </Box>
          </CardContent>
        </Card>

        {/* Create Token Dialog */}
        <Dialog 
          open={createDialogOpen} 
          onClose={() => setCreateDialogOpen(false)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>Create New Token</DialogTitle>
          <form onSubmit={handleSubmit(handleCreateToken)}>
            <DialogContent>
              <Grid container spacing={3}>
                <Grid item xs={12} sm={6}>
                  <Controller
                    name="owner_id"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Owner ID"
                        fullWidth
                        error={!!errors.owner_id}
                        helperText={errors.owner_id?.message}
                      />
                    )}
                  />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Controller
                    name="client_id"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        label="Client ID"
                        fullWidth
                        error={!!errors.client_id}
                        helperText={errors.client_id?.message}
                      />
                    )}
                  />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Controller
                    name="scope"
                    control={control}
                    render={({ field }) => (
                      <FormControl fullWidth error={!!errors.scope}>
                        <InputLabel>Scope</InputLabel>
                        <Select
                          {...field}
                          multiple
                          input={<OutlinedInput label="Scope" />}
                          renderValue={(selected) => (
                            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                              {selected.map((value) => (
                                <Chip key={value} label={value} size="small" />
                              ))}
                            </Box>
                          )}
                        >
                          {scopeOptions.map((scope) => (
                            <MenuItem key={scope} value={scope}>
                              {scope}
                            </MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    )}
                  />
                </Grid>
                <Grid item xs={12} sm={6}>
                  <Controller
                    name="duration"
                    control={control}
                    render={({ field }) => (
                      <TextField
                        {...field}
                        select
                        label="Duration"
                        fullWidth
                        error={!!errors.duration}
                        helperText={errors.duration?.message}
                      >
                        {durationOptions.map((option) => (
                          <MenuItem key={option.value} value={option.value}>
                            {option.label}
                          </MenuItem>
                        ))}
                      </TextField>
                    )}
                  />
                </Grid>
                
                {/* Custom Claims */}
                <Grid item xs={12}>
                  <Typography variant="h6" gutterBottom>
                    Custom Claims
                  </Typography>
                  {customClaims.map((claim, index) => (
                    <Box key={index} sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
                      <TextField
                        label="Key"
                        value={claim.key}
                        onChange={(e) => updateCustomClaim(index, 'key', e.target.value)}
                        size="small"
                      />
                      <TextField
                        label="Value"
                        value={claim.value}
                        onChange={(e) => updateCustomClaim(index, 'value', e.target.value)}
                        size="small"
                      />
                      <IconButton onClick={() => removeCustomClaim(index)} color="error">
                        <DeleteIcon />
                      </IconButton>
                    </Box>
                  ))}
                  <Button onClick={addCustomClaim} startIcon={<AddIcon />} variant="outlined">
                    Add Custom Claim
                  </Button>
                </Grid>
              </Grid>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
              <Button 
                type="submit" 
                variant="contained" 
                disabled={isLoading('createToken')}
              >
                {isLoading('createToken') ? <CircularProgress size={20} /> : 'Create Token'}
              </Button>
            </DialogActions>
          </form>
        </Dialog>

        {/* View Token Dialog */}
        <Dialog 
          open={viewDialogOpen} 
          onClose={() => setViewDialogOpen(false)}
          maxWidth="md"
          fullWidth
        >
          <DialogTitle>Token Details</DialogTitle>
          <DialogContent>
            {selectedToken && (
              <Box sx={{ mt: 2 }}>
                <Grid container spacing={2}>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2" color="text.secondary">Status</Typography>
                    <Chip
                      icon={getStatusIcon(selectedToken.status)}
                      label={selectedToken.status}
                      color={getStatusColor(selectedToken.status) as any}
                    />
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2" color="text.secondary">Owner</Typography>
                    <Typography>{selectedToken.owner_id}</Typography>
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2" color="text.secondary">Client ID</Typography>
                    <Typography>{selectedToken.client_id}</Typography>
                  </Grid>
                  <Grid item xs={12} sm={6}>
                    <Typography variant="subtitle2" color="text.secondary">Created</Typography>
                    <Typography>{format(parseISO(selectedToken.created_at), 'PPpp')}</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" color="text.secondary">Scope</Typography>
                    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mt: 1 }}>
                      {selectedToken.scope?.map((scope) => (
                        <Chip key={scope} label={scope} variant="outlined" />
                      ))}
                    </Box>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" color="text.secondary">Claims</Typography>
                    <Box component="pre" sx={{ 
                      mt: 1, 
                      p: 2, 
                      bgcolor: 'grey.100', 
                      borderRadius: 1,
                      fontSize: '0.875rem',
                      overflow: 'auto'
                    }}>
                      {JSON.stringify(selectedToken.claims, null, 2)}
                    </Box>
                  </Grid>
                </Grid>
              </Box>
            )}
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setViewDialogOpen(false)}>Close</Button>
          </DialogActions>
        </Dialog>
      </Box>
    </motion.div>
  );
}
