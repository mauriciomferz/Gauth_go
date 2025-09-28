import React from 'react';
import { Card, CardContent, Typography, Box, Alert } from '@mui/material';
import { SupervisorAccount } from '@mui/icons-material';

const BusinessOwners: React.FC = () => (
  <Box sx={{ flexGrow: 1 }}>
    <Alert severity="info" sx={{ mb: 4 }}>
      <Typography variant="h5" gutterBottom>
        <SupervisorAccount sx={{ mr: 1, verticalAlign: 'middle' }} />
        Business Owners & Functional Authority
      </Typography>
    </Alert>
    <Card><CardContent><Typography>Business Owners Management Component</Typography></CardContent></Card>
  </Box>
);

export default BusinessOwners;