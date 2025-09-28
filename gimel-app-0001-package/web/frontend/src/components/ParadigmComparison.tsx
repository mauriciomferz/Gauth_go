import React from 'react';
import {
  Card,
  CardContent,
  Typography,
  Box,
  Grid,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Divider,
} from '@mui/material';
import {
  Cancel,
  CheckCircle,
  CompareArrows,
  Policy,
  Gavel,
  Business,
  Engineering,
} from '@mui/icons-material';

const ParadigmComparison: React.FC = () => {
  const comparisonData = [
    {
      aspect: 'Authorization Basis',
      traditional: 'IT Policies & Technical Rules',
      gauth: 'Legal Power-of-Attorney Delegation',
      paradigm: 'Fundamental'
    },
    {
      aspect: 'Decision Authority',
      traditional: 'IT Department',
      gauth: 'Business Owners',
      paradigm: 'Fundamental'
    },
    {
      aspect: 'Responsibility Model',
      traditional: 'IT Teams RESPONSIBLE',
      gauth: 'Business Owners ACCOUNTABLE',
      paradigm: 'Fundamental'
    },
    {
      aspect: 'Legal Framework',
      traditional: 'Technical Compliance',
      gauth: 'Power-of-Attorney Law',
      paradigm: 'Revolutionary'
    },
    {
      aspect: 'AI Delegation',
      traditional: 'Policy-Based AI Access',
      gauth: 'Legal AI Power-of-Attorney',
      paradigm: 'Revolutionary'
    },
    {
      aspect: 'Scalability Model',
      traditional: 'Administrative Bottlenecks',
      gauth: 'Distributed Business Delegation',
      paradigm: 'Significant'
    },
    {
      aspect: 'Compliance Approach',
      traditional: 'Technical Audit Trails',
      gauth: 'Legal Accountability Trails',
      paradigm: 'Significant'
    },
    {
      aspect: 'Access Control',
      traditional: 'Permission Matrices',
      gauth: 'Delegated Authority Scope',
      paradigm: 'Significant'
    }
  ];

  const getChipColor = (paradigm: string) => {
    switch (paradigm) {
      case 'Fundamental': return 'error';
      case 'Revolutionary': return 'success';
      case 'Significant': return 'warning';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ flexGrow: 1 }}>
      {/* Header */}
      <Alert severity="warning" sx={{ mb: 4 }}>
        <Typography variant="h5" gutterBottom>
          <CompareArrows sx={{ mr: 1, verticalAlign: 'middle' }} />
          P*P Paradigm Revolution: Power vs Policy
        </Typography>
        <Typography variant="body1">
          <strong>Critical Understanding:</strong> GAuth represents a fundamental shift from IT-managed policies 
          to business-managed power delegation. The first "P" in P*P stands for <strong>POWER-OF-ATTORNEY</strong>, not policies!
        </Typography>
      </Alert>

      {/* Visual Comparison Cards */}
      <Grid container spacing={4} sx={{ mb: 4 }}>
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%', border: '2px solid', borderColor: 'error.main' }}>
            <CardContent>
              <Box display="flex" alignItems="center" sx={{ mb: 2 }}>
                <Cancel color="error" sx={{ mr: 2, fontSize: 32 }} />
                <Typography variant="h5" color="error">
                  Traditional IT Model
                </Typography>
              </Box>
              
              <Typography variant="h6" gutterBottom>
                Policy-based Permission (P*P)
              </Typography>

              <Box sx={{ bgcolor: 'error.50', p: 2, borderRadius: 1, mb: 2 }}>
                <Typography variant="body2" sx={{ fontWeight: 'bold', mb: 1 }}>
                  <Engineering sx={{ mr: 1, verticalAlign: 'middle', fontSize: 16 }} />
                  IT-Centric Authorization Flow:
                </Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>
                  User Request → IT Policy Engine → Technical Evaluation → Access Granted/Denied
                </Typography>
              </Box>

              <Typography variant="subtitle2" gutterBottom color="error">
                Key Characteristics:
              </Typography>
              <Box component="ul" sx={{ pl: 2, m: 0 }}>
                <Typography component="li" variant="body2">
                  IT creates and manages <strong>policies</strong>
                </Typography>
                <Typography component="li" variant="body2">
                  Technical rules govern access decisions
                </Typography>
                <Typography component="li" variant="body2">
                  IT department is <strong>responsible</strong>
                </Typography>
                <Typography component="li" variant="body2">
                  Administrative control by technical teams
                </Typography>
                <Typography component="li" variant="body2">
                  Policy enforcement through technical mechanisms
                </Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%', border: '2px solid', borderColor: 'success.main' }}>
            <CardContent>
              <Box display="flex" alignItems="center" sx={{ mb: 2 }}>
                <CheckCircle color="success" sx={{ mr: 2, fontSize: 32 }} />
                <Typography variant="h5" color="success.main">
                  GAuth P*P Model
                </Typography>
              </Box>
              
              <Typography variant="h6" gutterBottom>
                Power-of-Attorney Protocol (P*P)
              </Typography>

              <Box sx={{ bgcolor: 'success.50', p: 2, borderRadius: 1, mb: 2 }}>
                <Typography variant="body2" sx={{ fontWeight: 'bold', mb: 1 }}>
                  <Business sx={{ mr: 1, verticalAlign: 'middle', fontSize: 16 }} />
                  Business-Centric Authorization Flow:
                </Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>
                  Business Owner → Power Delegation → Legal Validation → Authority Exercised
                </Typography>
              </Box>

              <Typography variant="subtitle2" gutterBottom color="success.main">
                Key Characteristics:
              </Typography>
              <Box component="ul" sx={{ pl: 2, m: 0 }}>
                <Typography component="li" variant="body2">
                  Business owners delegate specific <strong>powers</strong>
                </Typography>
                <Typography component="li" variant="body2">
                  Legal frameworks govern authorization decisions
                </Typography>
                <Typography component="li" variant="body2">
                  Business owners are <strong>accountable</strong>
                </Typography>
                <Typography component="li" variant="body2">
                  Functional control by business teams
                </Typography>
                <Typography component="li" variant="body2">
                  Power delegation through legal mechanisms
                </Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Detailed Comparison Table */}
      <Card sx={{ mb: 4 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            <Gavel sx={{ mr: 1, verticalAlign: 'middle' }} />
            Detailed Paradigm Comparison
          </Typography>
          <Divider sx={{ mb: 2 }} />
          
          <TableContainer component={Paper} variant="outlined">
            <Table>
              <TableHead>
                <TableRow sx={{ bgcolor: 'grey.100' }}>
                  <TableCell><strong>Aspect</strong></TableCell>
                  <TableCell><strong>Traditional IT Model</strong></TableCell>
                  <TableCell><strong>GAuth P*P Model</strong></TableCell>
                  <TableCell><strong>Impact Level</strong></TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {comparisonData.map((row, index) => (
                  <TableRow key={index} hover>
                    <TableCell sx={{ fontWeight: 'bold' }}>
                      {row.aspect}
                    </TableCell>
                    <TableCell sx={{ color: 'error.main' }}>
                      <Cancel sx={{ mr: 1, verticalAlign: 'middle', fontSize: 16 }} />
                      {row.traditional}
                    </TableCell>
                    <TableCell sx={{ color: 'success.main' }}>
                      <CheckCircle sx={{ mr: 1, verticalAlign: 'middle', fontSize: 16 }} />
                      {row.gauth}
                    </TableCell>
                    <TableCell>
                      <Chip 
                        label={row.paradigm} 
                        color={getChipColor(row.paradigm) as any}
                        size="small"
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      {/* RFC111 & RFC115 Integration */}
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Card sx={{ bgcolor: 'primary.50' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom color="primary">
                RFC111: AI Power-of-Attorney
              </Typography>
              <Typography variant="body2" paragraph>
                RFC111 implements the P*P paradigm for AI delegation scenarios, where business owners 
                can legally delegate specific powers to AI agents through proper legal frameworks.
              </Typography>
              <Box component="ul" sx={{ pl: 2, m: 0 }}>
                <Typography component="li" variant="body2">
                  Business-driven AI authorization
                </Typography>
                <Typography component="li" variant="body2">
                  Legal framework compliance
                </Typography>
                <Typography component="li" variant="body2">
                  Power-specific delegation scope
                </Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card sx={{ bgcolor: 'secondary.50' }}>
            <CardContent>
              <Typography variant="h6" gutterBottom color="secondary">
                RFC115: Enhanced Delegation
              </Typography>
              <Typography variant="body2" paragraph>
                RFC115 extends the P*P paradigm with advanced attestation and multi-signature 
                requirements for high-stakes business power delegations.
              </Typography>
              <Box component="ul" sx={{ pl: 2, m: 0 }}>
                <Typography component="li" variant="body2">
                  Multi-signature attestation
                </Typography>
                <Typography component="li" variant="body2">
                  Time-bound validity controls
                </Typography>
                <Typography component="li" variant="body2">
                  Enhanced accountability trails
                </Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default ParadigmComparison;