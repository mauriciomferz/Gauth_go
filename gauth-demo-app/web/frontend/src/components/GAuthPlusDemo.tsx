import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Button,
  TextField,
  Alert,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  CircularProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Divider,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
} from '@mui/material';
import {
  ExpandMore,
  Security,
  Gavel,
  AccountBalance,
  Verified,
  Error,
  CheckCircle,
  Info,
  Business,
  Person,
  Computer,
} from '@mui/icons-material';
import { motion } from 'framer-motion';
import toast from 'react-hot-toast';

interface AuthorizingParty {
  id: string;
  name: string;
  type: 'individual' | 'corporation' | 'government';
  authority_level: string;
  verification_status: string;
}

interface AIAuthorizationRecord {
  ai_system_id: string;
  authorizing_party: AuthorizingParty;
  authorized_decisions: string[];
  permitted_transactions: string[];
  allowed_actions: string[];
  blockchain_hash?: string;
  created_at?: string;
}

interface CommercialRegisterEntry {
  id: string;
  ai_system_id: string;
  registration_status: string;
  blockchain_hash: string;
  verification_level: string;
  created_at: string;
  last_updated: string;
}

const GAuthPlusDemo: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [aiSystemId, setAiSystemId] = useState('');
  const [authorizationRecord, setAuthorizationRecord] = useState<AIAuthorizationRecord | null>(null);
  const [validationResult, setValidationResult] = useState<any>(null);
  const [commercialRegisterEntries, setCommercialRegisterEntries] = useState<CommercialRegisterEntry[]>([]);
  const [activeTab, setActiveTab] = useState<'register' | 'validate' | 'query'>('register');

  // Register AI Authorization
  const handleRegisterAuthorization = async () => {
    if (!aiSystemId.trim()) {
      toast.error('Please enter an AI System ID');
      return;
    }

    setLoading(true);
    try {
      const authRecord: AIAuthorizationRecord = {
        ai_system_id: aiSystemId,
        authorizing_party: {
          id: `party_${Date.now()}`,
          name: 'Acme Corporation Legal Department',
          type: 'corporation',
          authority_level: 'primary',
          verification_status: 'verified',
        },
        authorized_decisions: [
          'contract_approval_up_to_500k',
          'vendor_payment_authorization',
          'legal_document_signing',
        ],
        permitted_transactions: [
          'financial_transactions_business_hours',
          'contract_execution_corporate_policy',
          'audit_trail_generation',
        ],
        allowed_actions: [
          'sign_contracts_with_dual_control',
          'approve_payments_under_threshold',
          'generate_compliance_reports',
        ],
      };

      const response = await fetch('/api/v1/gauth-plus/authorize', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(authRecord),
      });

      if (response.ok) {
        const result = await response.json();
        setAuthorizationRecord(result.authorization_record);
        toast.success('AI Authorization registered successfully on blockchain!');
        
        // Auto-query the commercial register
        await queryCommercialRegister();
      } else {
        const error = await response.json();
        toast.error(`Registration failed: ${error.message || 'Unknown error'}`);
      }
    } catch (error) {
      console.error('Registration error:', error);
      toast.error('Network error during registration');
    } finally {
      setLoading(false);
    }
  };

  // Validate AI Authority
  const handleValidateAuthority = async () => {
    if (!aiSystemId.trim()) {
      toast.error('Please enter an AI System ID');
      return;
    }

    setLoading(true);
    try {
      const response = await fetch('/api/v1/gauth-plus/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ai_system_id: aiSystemId,
          requested_action: 'contract_signing',
          transaction_type: 'financial_transaction',
          amount: 250000,
        }),
      });

      if (response.ok) {
        const result = await response.json();
        setValidationResult(result);
        toast.success('Authority validation completed!');
      } else {
        const error = await response.json();
        toast.error(`Validation failed: ${error.message || 'Unknown error'}`);
      }
    } catch (error) {
      console.error('Validation error:', error);
      toast.error('Network error during validation');
    } finally {
      setLoading(false);
    }
  };

  // Query Commercial Register
  const queryCommercialRegister = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/gauth-plus/commercial-register', {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
      });

      if (response.ok) {
        const result = await response.json();
        setCommercialRegisterEntries(result.entries || []);
      } else {
        console.error('Failed to query commercial register');
      }
    } catch (error) {
      console.error('Query error:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    queryCommercialRegister();
  }, []);

  const renderAuthorizationForm = () => (
    <Card elevation={3}>
      <CardContent>
        <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Security color="primary" />
          Register AI Authorization
        </Typography>
        <Typography variant="body2" color="text.secondary" paragraph>
          Register comprehensive AI power-of-attorney on the blockchain commercial register.
        </Typography>
        
        <TextField
          fullWidth
          label="AI System ID"
          value={aiSystemId}
          onChange={(e) => setAiSystemId(e.target.value)}
          placeholder="Enter AI system identifier (e.g., ai-legal-assistant-v2)"
          margin="normal"
        />
        
        <Button
          variant="contained"
          color="primary"
          onClick={handleRegisterAuthorization}
          disabled={loading}
          sx={{ mt: 2 }}
          startIcon={loading ? <CircularProgress size={20} /> : <Gavel />}
        >
          {loading ? 'Registering...' : 'Register on Blockchain'}
        </Button>

        {authorizationRecord && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            <Alert severity="success" sx={{ mt: 2 }}>
              <Typography variant="subtitle2">Registration Successful!</Typography>
              <Typography variant="body2">
                Blockchain Hash: <code>{authorizationRecord.blockchain_hash}</code>
              </Typography>
            </Alert>
            
            <Box sx={{ mt: 2 }}>
              <Typography variant="subtitle2" gutterBottom>Authorized Powers:</Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {authorizationRecord.authorized_decisions.map((decision, index) => (
                  <Chip key={index} label={decision} size="small" color="primary" />
                ))}
              </Box>
            </Box>
          </motion.div>
        )}
      </CardContent>
    </Card>
  );

  const renderValidationForm = () => (
    <Card elevation={3}>
      <CardContent>
        <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Verified color="success" />
          Validate AI Authority
        </Typography>
        <Typography variant="body2" color="text.secondary" paragraph>
          Validate an AI system's authority to perform specific actions against the blockchain registry.
        </Typography>
        
        <TextField
          fullWidth
          label="AI System ID"
          value={aiSystemId}
          onChange={(e) => setAiSystemId(e.target.value)}
          placeholder="Enter AI system identifier to validate"
          margin="normal"
        />
        
        <Button
          variant="contained"
          color="success"
          onClick={handleValidateAuthority}
          disabled={loading}
          sx={{ mt: 2 }}
          startIcon={loading ? <CircularProgress size={20} /> : <CheckCircle />}
        >
          {loading ? 'Validating...' : 'Validate Authority'}
        </Button>

        {validationResult && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            <Alert 
              severity={validationResult.authorized ? "success" : "error"} 
              sx={{ mt: 2 }}
            >
              <Typography variant="subtitle2">
                Authority Status: {validationResult.authorized ? 'AUTHORIZED' : 'NOT AUTHORIZED'}
              </Typography>
              <Typography variant="body2">
                Validation Level: {validationResult.validation_level || 'N/A'}
              </Typography>
            </Alert>
            
            {validationResult.authority_details && (
              <Accordion sx={{ mt: 2 }}>
                <AccordionSummary expandIcon={<ExpandMore />}>
                  <Typography>Authority Details</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <pre style={{ fontSize: '12px', overflow: 'auto' }}>
                    {JSON.stringify(validationResult.authority_details, null, 2)}
                  </pre>
                </AccordionDetails>
              </Accordion>
            )}
          </motion.div>
        )}
      </CardContent>
    </Card>
  );

  const renderCommercialRegister = () => (
    <Card elevation={3}>
      <CardContent>
        <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <AccountBalance color="info" />
          Commercial Register Query
        </Typography>
        <Typography variant="body2" color="text.secondary" paragraph>
          View all registered AI systems in the blockchain commercial register.
        </Typography>
        
        <Button
          variant="outlined"
          onClick={queryCommercialRegister}
          disabled={loading}
          sx={{ mb: 2 }}
          startIcon={loading ? <CircularProgress size={20} /> : <Info />}
        >
          {loading ? 'Querying...' : 'Refresh Registry'}
        </Button>

        {commercialRegisterEntries.length > 0 ? (
          <TableContainer component={Paper} elevation={1}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>AI System ID</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Verification</TableCell>
                  <TableCell>Blockchain Hash</TableCell>
                  <TableCell>Registered</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {commercialRegisterEntries.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell>{entry.ai_system_id}</TableCell>
                    <TableCell>
                      <Chip
                        label={entry.registration_status}
                        color={entry.registration_status === 'active' ? 'success' : 'default'}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={entry.verification_level}
                        color="primary"
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <code style={{ fontSize: '10px' }}>
                        {entry.blockchain_hash.substring(0, 16)}...
                      </code>
                    </TableCell>
                    <TableCell>{new Date(entry.created_at).toLocaleDateString()}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        ) : (
          <Alert severity="info">
            No entries found in the commercial register. Register an AI system first.
          </Alert>
        )}
      </CardContent>
    </Card>
  );

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom sx={{ mb: 3, display: 'flex', alignItems: 'center', gap: 2 }}>
        <Business color="primary" />
        GAuth+ Commercial Register for AI Systems
      </Typography>

      <Typography variant="body1" color="text.secondary" paragraph>
        Comprehensive blockchain-based commercial register implementing power-of-attorney framework for AI systems.
        This system answers the four fundamental questions: WHO, WHAT, TRANSACTIONS, and ACTIONS.
      </Typography>

      <Box sx={{ mb: 3 }}>
        <Button
          variant={activeTab === 'register' ? 'contained' : 'outlined'}
          onClick={() => setActiveTab('register')}
          sx={{ mr: 1 }}
        >
          Register AI
        </Button>
        <Button
          variant={activeTab === 'validate' ? 'contained' : 'outlined'}
          onClick={() => setActiveTab('validate')}
          sx={{ mr: 1 }}
        >
          Validate Authority
        </Button>
        <Button
          variant={activeTab === 'query' ? 'contained' : 'outlined'}
          onClick={() => setActiveTab('query')}
        >
          Query Register
        </Button>
      </Box>

      <Grid container spacing={3}>
        <Grid item xs={12}>
          {activeTab === 'register' && renderAuthorizationForm()}
          {activeTab === 'validate' && renderValidationForm()}
          {activeTab === 'query' && renderCommercialRegister()}
        </Grid>
      </Grid>

      <Box sx={{ mt: 4 }}>
        <Alert severity="info">
          <Typography variant="subtitle2">GAuth+ Features:</Typography>
          <List dense>
            <ListItem>
              <ListItemIcon><Computer fontSize="small" /></ListItemIcon>
              <ListItemText primary="Blockchain-based AI authorization registry" />
            </ListItem>
            <ListItem>
              <ListItemIcon><Gavel fontSize="small" /></ListItemIcon>
              <ListItemText primary="Comprehensive power-of-attorney framework" />
            </ListItem>
            <ListItem>
              <ListItemIcon><Security fontSize="small" /></ListItemIcon>
              <ListItemText primary="Dual control principle with human accountability" />
            </ListItem>
            <ListItem>
              <ListItemIcon><AccountBalance fontSize="small" /></ListItemIcon>
              <ListItemText primary="Global commercial register with cryptographic verification" />
            </ListItem>
          </List>
        </Alert>
      </Box>
    </Box>
  );
};

export default GAuthPlusDemo;