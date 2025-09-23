import React from 'react';
import { Card, CardContent, Typography, Box, Alert } from '@mui/material';
import { Audit } from '@mui/icons-material';

const AuditTrail: React.FC = () => (
  <Box sx={{ flexGrow: 1 }}>
    <Alert severity="info" sx={{ mb: 4 }}>
      <Typography variant="h5" gutterBottom>
        <Audit sx={{ mr: 1, verticalAlign: 'middle' }} />
        Business Accountability Audit Trail
      </Typography>
    </Alert>
    <Card><CardContent><Typography>Audit Trail Component</Typography></CardContent></Card>
  </Box>
);

export default AuditTrail;