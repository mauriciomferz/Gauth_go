import React, { useState } from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  Grid,
  TextField,
  Button,
  Alert,
  Chip,
  Divider,
  FormControlLabel,
  Switch,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Paper,
} from '@mui/material';
import {
  Security,
  Send,
  CheckCircle,
  Business,
  Gavel,
} from '@mui/icons-material';

const RFC111Demo: React.FC = () => {
  const [formData, setFormData] = useState({
    clientId: 'cfo_ai_assistant',
    powerType: 'corporate_financial_authority',
    principalId: 'cfo_jane_smith',
    aiAgentId: 'corporate_ai_assistant_v3',
    jurisdiction: 'US',
    legalBasis: 'corporate_power_of_attorney_act_2024',
    businessOwner: 'cfo_jane_smith',
    department: 'Finance',
    amountLimit: 500000,
    businessJustification: 'AI delegation for operational efficiency while maintaining CFO accountability',
  });

  const [response, setResponse] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const handleInputChange = (field: string, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleSubmit = async () => {
    setLoading(true);
    try {
      const requestBody = {
        client_id: formData.clientId,
        response_type: "code",
        scope: ["financial_power_of_attorney", "corporate_transactions", "regulatory_compliance"],
        redirect_uri: "http://localhost:3000/callback",
        power_type: formData.powerType,
        principal_id: formData.principalId,
        ai_agent_id: formData.aiAgentId,
        jurisdiction: formData.jurisdiction,
        legal_basis: formData.legalBasis,
        business_owner: {
          owner_id: formData.businessOwner,
          role: "Chief Financial Officer",
          department: formData.department,
          delegation_authority: "corporate_financial_powers",
          accountability_level: "executive"
        },
        legal_framework: {
          jurisdiction: formData.jurisdiction,
          entity_type: "corporation",
          capacity_verification: true,
          business_accountability_rules: ["executive_oversight", "board_reporting", "regulatory_compliance"]
        },
        delegated_powers: ["authorize_payments", "sign_contracts", "manage_investments", "regulatory_filings"],
        business_restrictions: {
          amount_limit: formData.amountLimit,
          geo_restrictions: ["US", "EU"],
          business_hours_only: true,
          approval_threshold: 100000,
          executive_oversight_required: true
        },
        accountability_context: {
          business_justification: formData.businessJustification,
          legal_responsibility: "CFO remains legally responsible for all delegated actions",
          compliance_framework: ["SOX", "GAAP", "SEC_regulations"]
        }
      };

      const response = await fetch('http://localhost:8080/api/v1/rfc111/authorize', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      });

      const data = await response.json();
      setResponse(data);
    } catch (error) {
      console.error('Error:', error);
      setResponse({ error: 'Failed to connect to backend' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ flexGrow: 1 }}>
      {/* Header */}
      <Alert severity="info" sx={{ mb: 4 }}>
        <Typography variant="h5" gutterBottom>
          <Security sx={{ mr: 1, verticalAlign: 'middle' }} />
          RFC111: AI Power-of-Attorney Authorization
        </Typography>
        <Typography variant="body1">
          <strong>Business Context:</strong> This demonstrates how a business owner (CFO) delegates 
          specific financial powers to an AI assistant through legal power-of-attorney frameworks, 
          maintaining business accountability rather than IT responsibility.
        </Typography>
      </Alert>

      <Grid container spacing={4}>
        {/* Input Form */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                <Business sx={{ mr: 1, verticalAlign: 'middle' }} />
                Business Power Delegation Request
              </Typography>
              <Divider sx={{ mb: 3 }} />

              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Business Owner (Principal)"
                    value={formData.principalId}
                    onChange={(e) => handleInputChange('principalId', e.target.value)}
                    helperText="Business owner delegating power (NOT IT admin)"
                  />
                </Grid>

                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="AI Agent ID"
                    value={formData.aiAgentId}
                    onChange={(e) => handleInputChange('aiAgentId', e.target.value)}
                    helperText="AI system receiving delegated power"
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <FormControl fullWidth>
                    <InputLabel>Power Type</InputLabel>
                    <Select
                      value={formData.powerType}
                      label="Power Type"
                      onChange={(e) => handleInputChange('powerType', e.target.value)}
                    >
                      <MenuItem value="corporate_financial_authority">Corporate Financial Authority</MenuItem>
                      <MenuItem value="legal_contract_authority">Legal Contract Authority</MenuItem>
                      <MenuItem value="governance_authority">Governance Authority</MenuItem>
                      <MenuItem value="regulatory_compliance_authority">Regulatory Compliance</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>

                <Grid item xs={12} sm={6}>
                  <FormControl fullWidth>
                    <InputLabel>Department</InputLabel>
                    <Select
                      value={formData.department}
                      label="Department"
                      onChange={(e) => handleInputChange('department', e.target.value)}
                    >
                      <MenuItem value="Finance">Finance</MenuItem>
                      <MenuItem value="Legal">Legal</MenuItem>
                      <MenuItem value="Operations">Operations</MenuItem>
                      <MenuItem value="Governance">Governance</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>

                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Amount Limit ($)"
                    type="number"
                    value={formData.amountLimit}
                    onChange={(e) => handleInputChange('amountLimit', parseInt(e.target.value))}
                    helperText="Business-defined financial delegation limit"
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <FormControl fullWidth>
                    <InputLabel>Jurisdiction</InputLabel>
                    <Select
                      value={formData.jurisdiction}
                      label="Jurisdiction"
                      onChange={(e) => handleInputChange('jurisdiction', e.target.value)}
                    >
                      <MenuItem value="US">United States</MenuItem>
                      <MenuItem value="EU">European Union</MenuItem>
                      <MenuItem value="CA">Canada</MenuItem>
                      <MenuItem value="UK">United Kingdom</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>

                <Grid item xs={12} sm={6}>
                  <TextField
                    fullWidth
                    label="Legal Basis"
                    value={formData.legalBasis}
                    onChange={(e) => handleInputChange('legalBasis', e.target.value)}
                    helperText="Legal foundation for power delegation"
                  />
                </Grid>

                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    multiline
                    rows={3}
                    label="Business Justification"
                    value={formData.businessJustification}
                    onChange={(e) => handleInputChange('businessJustification', e.target.value)}
                    helperText="Business case for AI power delegation"
                  />
                </Grid>

                <Grid item xs={12}>
                  <Button
                    fullWidth
                    variant="contained"
                    color="primary"
                    size="large"
                    startIcon={<Send />}
                    onClick={handleSubmit}
                    disabled={loading}
                  >
                    {loading ? 'Delegating Power...' : 'Delegate Power-of-Attorney'}
                  </Button>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        </Grid>

        {/* Response Display */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                <Gavel sx={{ mr: 1, verticalAlign: 'middle' }} />
                RFC111 Authorization Response
              </Typography>
              <Divider sx={{ mb: 3 }} />

              {response ? (
                <Box>
                  {response.error ? (
                    <Alert severity="error">
                      <Typography variant="body2">
                        {response.error}
                      </Typography>
                    </Alert>
                  ) : (
                    <Box>
                      <Alert severity="success" sx={{ mb: 2 }}>
                        <Typography variant="body2">
                          <CheckCircle sx={{ mr: 1, verticalAlign: 'middle', fontSize: 16 }} />
                          Power-of-Attorney Successfully Delegated!
                        </Typography>
                      </Alert>

                      <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
                        <Typography variant="subtitle2" gutterBottom>
                          Authorization Code:
                        </Typography>
                        <Typography variant="body2" sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>
                          {response.code}
                        </Typography>
                      </Paper>

                      {response.legal_validation && (
                        <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
                          <Typography variant="subtitle2" gutterBottom>
                            Legal Framework Validation:
                          </Typography>
                          <Box display="flex" gap={1} flexWrap="wrap">
                            <Chip 
                              label={`Valid: ${response.legal_validation.valid ? 'Yes' : 'No'}`}
                              color={response.legal_validation.valid ? 'success' : 'error'}
                              size="small"
                            />
                            <Chip 
                              label={`Jurisdiction: ${response.legal_validation.jurisdiction_id}`}
                              color="info"
                              size="small"
                            />
                            <Chip 
                              label={`Level: ${response.legal_validation.compliance_level}`}
                              color="primary"
                              size="small"
                            />
                          </Box>
                        </Paper>
                      )}

                      {response.compliance_status && (
                        <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
                          <Typography variant="subtitle2" gutterBottom>
                            Compliance Status:
                          </Typography>
                          <Box display="flex" gap={1} flexWrap="wrap">
                            <Chip 
                              label={`Status: ${response.compliance_status.status}`}
                              color="success"
                              size="small"
                            />
                            <Chip 
                              label={`Level: ${response.compliance_status.compliance_level}`}
                              color="primary"
                              size="small"
                            />
                          </Box>
                          {response.compliance_status.compliance_rules && (
                            <Box sx={{ mt: 1 }}>
                              <Typography variant="caption" display="block">
                                Compliance Rules:
                              </Typography>
                              {response.compliance_status.compliance_rules.map((rule: string, index: number) => (
                                <Chip 
                                  key={index}
                                  label={rule}
                                  size="small"
                                  variant="outlined"
                                  sx={{ mr: 0.5, mb: 0.5 }}
                                />
                              ))}
                            </Box>
                          )}
                        </Paper>
                      )}

                      {response.audit_trail && response.audit_trail.length > 0 && (
                        <Paper variant="outlined" sx={{ p: 2 }}>
                          <Typography variant="subtitle2" gutterBottom>
                            Business Accountability Trail:
                          </Typography>
                          {response.audit_trail.map((event: any, index: number) => (
                            <Box key={index} sx={{ mb: 1 }}>
                              <Typography variant="body2">
                                <strong>{event.action}</strong> by {event.actor_id}
                              </Typography>
                              <Typography variant="caption" color="text.secondary">
                                {new Date(event.timestamp).toLocaleString()}
                              </Typography>
                            </Box>
                          ))}
                        </Paper>
                      )}
                    </Box>
                  )}
                </Box>
              ) : (
                <Alert severity="info">
                  <Typography variant="body2">
                    Submit a power delegation request to see the RFC111 authorization response. 
                    This demonstrates business owner accountability rather than IT responsibility.
                  </Typography>
                </Alert>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default RFC111Demo;