import React from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  Grid,
  Alert,
  Chip,
  Button,
  LinearProgress,
} from '@mui/material';
import {
  Gavel,
  Business,
  Security,
  CompareArrows,
  CheckCircle,
  Cancel,
} from '@mui/icons-material';

const Dashboard: React.FC = () => {
  return (
    <Box sx={{ flexGrow: 1 }}>
      {/* Header Section - P*P Paradigm Explanation */}
      <Alert severity="info" sx={{ mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          🚀 GAuth Power-of-Attorney Protocol (P*P) Dashboard
        </Typography>
        <Typography variant="body2">
          <strong>PARADIGM SHIFT:</strong> This system implements POWER-based authorization, NOT policy-based! 
          Business owners delegate specific powers through legal frameworks, maintaining accountability for their decisions.
        </Typography>
      </Alert>

      {/* Key Metrics Row */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: 'primary.main', color: 'white' }}>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Gavel sx={{ mr: 2, fontSize: 40 }} />
                <Box>
                  <Typography variant="h4" component="div">
                    23
                  </Typography>
                  <Typography variant="body2">
                    Active Power Delegations
                  </Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: 'secondary.main', color: 'white' }}>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Business sx={{ mr: 2, fontSize: 40 }} />
                <Box>
                  <Typography variant="h4" component="div">
                    8
                  </Typography>
                  <Typography variant="body2">
                    Business Owners
                  </Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: 'success.main', color: 'white' }}>
            <CardContent>
              <Box display="flex" alignItems="center">
                <Security sx={{ mr: 2, fontSize: 40 }} />
                <Box>
                  <Typography variant="h4" component="div">
                    156
                  </Typography>
                  <Typography variant="body2">
                    RFC111/115 Authorizations
                  </Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card sx={{ bgcolor: 'warning.main', color: 'white' }}>
            <CardContent>
              <Box display="flex" alignItems="center">
                <CompareArrows sx={{ mr: 2, fontSize: 40 }} />
                <Box>
                  <Typography variant="h4" component="div">
                    100%
                  </Typography>
                  <Typography variant="body2">
                    Compliance Rate
                  </Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* P*P vs Traditional Comparison */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%', bgcolor: 'error.50' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom color="error">
                <Cancel sx={{ mr: 1, verticalAlign: 'middle' }} />
                Traditional IT Model (Replaced)
              </Typography>
              <Typography variant="body2" sx={{ mb: 2 }}>
                Policy-based Permission System:
              </Typography>
              <Box sx={{ pl: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  • IT creates and manages policies
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  • Technical rules govern access
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  • IT is RESPONSIBLE for decisions
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  • Administrative control by tech teams
                </Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%', bgcolor: 'success.50' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom color="success.main">
                <CheckCircle sx={{ mr: 1, verticalAlign: 'middle' }} />
                GAuth P*P Model (Revolutionary)
              </Typography>
              <Typography variant="body2" sx={{ mb: 2 }}>
                Power-of-Attorney Protocol:
              </Typography>
              <Box sx={{ pl: 2 }}>
                <Typography variant="body2" color="text.secondary">
                  • Business owners DELEGATE powers
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  • Legal frameworks govern authorization
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  • Business owners are ACCOUNTABLE
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  • Functional control by business teams
                </Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Recent Power Delegations */}
      <Grid container spacing={3}>
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Recent Power Delegations
              </Typography>
              <Box sx={{ mb: 2 }}>
                <Box display="flex" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                  <Typography variant="body2">
                    CFO → AI Financial Assistant (RFC111)
                  </Typography>
                  <Chip label="Active" color="success" size="small" />
                </Box>
                <LinearProgress variant="determinate" value={100} color="success" />
              </Box>
              
              <Box sx={{ mb: 2 }}>
                <Box display="flex" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                  <Typography variant="body2">
                    Board Chair → Governance AI (RFC115)
                  </Typography>
                  <Chip label="Attestation Required" color="warning" size="small" />
                </Box>
                <LinearProgress variant="determinate" value={75} color="warning" />
              </Box>

              <Box sx={{ mb: 2 }}>
                <Box display="flex" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                  <Typography variant="body2">
                    Legal Counsel → Contract AI (RFC111)
                  </Typography>
                  <Chip label="Pending Validation" color="info" size="small" />
                </Box>
                <LinearProgress variant="determinate" value={50} color="info" />
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Quick Actions
              </Typography>
              <Box display="flex" flexDirection="column" gap={2}>
                <Button 
                  variant="contained" 
                  color="primary" 
                  startIcon={<Gavel />}
                  href="/power-delegation"
                >
                  Create Power Delegation
                </Button>
                <Button 
                  variant="outlined" 
                  color="secondary"
                  startIcon={<Security />}
                  href="/rfc111"
                >
                  RFC111 AI Authorization
                </Button>
                <Button 
                  variant="outlined" 
                  color="info"
                  startIcon={<CompareArrows />}
                  href="/paradigm"
                >
                  View P*P Comparison
                </Button>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default Dashboard;