import React, { useState, useEffect } from 'react';
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import {
  AppBar,
  Toolbar,
  Typography,
  Container,
  Box,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  ListItemButton,
  Divider,
  Alert,
  Snackbar,
  Chip,
  IconButton,
  Badge,
  Menu,
  MenuItem,
  Avatar,
  Switch,
  FormControlLabel,
  CssBaseline,
  Tooltip,
} from '@mui/material';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import {
  Gavel,
  BusinessCenter,
  CompareArrows,
  Security,
  AccountBalance,
  Assessment,
  Audit,
  SupervisorAccount,
  Policy,
  Menu as MenuIcon,
  Notifications,
  Settings,
  Logout,
  Token,
  Dashboard as DashboardIcon,
  DarkMode,
  LightMode,
} from '@mui/icons-material';
import { Toaster } from 'react-hot-toast';
import { motion, AnimatePresence } from 'framer-motion';

import Dashboard from './components/Dashboard';
import ParadigmComparison from './components/ParadigmComparison';
import PowerDelegation from './components/PowerDelegation';
import RFC111Demo from './components/RFC111Demo';
import RFC115Demo from './components/RFC115Demo';
import BusinessOwners from './components/BusinessOwners';
import ComplianceMonitor from './components/ComplianceMonitor';
import AuditTrail from './components/AuditTrail';
import TokenManagement from './components/TokenManagement';
import WebSocketService from './services/WebSocketService';
import { useAppStore } from './store/appStore';
import { useAuthStore } from './store/authStore';

const DRAWER_WIDTH = 280;

interface NavigationItem {
  text: string;
  icon: React.ReactElement;
  path: string;
  description: string;
  paradigm: 'power' | 'policy' | 'comparison';
}

const navigationItems: NavigationItem[] = [
  { 
    text: 'Dashboard', 
    icon: <Assessment />, 
    path: '/', 
    description: 'P*P Overview',
    paradigm: 'power'
  },
  { 
    text: 'P*P Paradigm', 
    icon: <CompareArrows />, 
    path: '/paradigm', 
    description: 'Power vs Policy',
    paradigm: 'comparison'
  },
  { 
    text: 'Power Delegation', 
    icon: <Gavel />, 
    path: '/power-delegation', 
    description: 'Business Authority',
    paradigm: 'power'
  },
  { 
    text: 'RFC111 (AI PoA)', 
    icon: <Security />, 
    path: '/rfc111', 
    description: 'AI Power-of-Attorney',
    paradigm: 'power'
  },
  { 
    text: 'RFC115 (Enhanced)', 
    icon: <AccountBalance />, 
    path: '/rfc115', 
    description: 'Advanced Delegation',
    paradigm: 'power'
  },
  { 
    text: 'Business Owners', 
    icon: <SupervisorAccount />, 
    path: '/business-owners', 
    description: 'Functional Authority',
    paradigm: 'power'
  },
  { 
    text: 'Compliance Monitor', 
    icon: <Policy />, 
    path: '/compliance', 
    description: 'Legal Framework',
    paradigm: 'power'
  },
  { 
    text: 'Audit Trail', 
    icon: <Audit />, 
    path: '/audit', 
    description: 'Accountability',
    paradigm: 'power'
  },
];

const App: React.FC = () => {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [notifications, setNotifications] = useState<Array<{ 
    id: number; 
    message: string; 
    severity: 'success' | 'error' | 'warning' | 'info';
    paradigm?: 'power' | 'policy';
  }>>([]);
  const navigate = useNavigate();

  useEffect(() => {
    // Initialize WebSocket connection for real-time Power-of-Attorney events
    const wsService = WebSocketService.getInstance();
    
    wsService.connect('ws://localhost:8080/ws/events');
    
    wsService.onMessage((event) => {
      // Handle different Power-of-Attorney event types
      let message = '';
      let severity: 'success' | 'error' | 'warning' | 'info' = 'info';
      let paradigm: 'power' | 'policy' = 'power';
      
      switch (event.type) {
        case 'power_delegation_request':
          message = `🎯 Business owner ${event.data.business_owner} delegating ${event.data.power_type}`;
          severity = 'info';
          paradigm = 'power';
          break;
        case 'rfc111_authorization':
          message = `⚖️ RFC111 AI Power-of-Attorney authorized for ${event.data.ai_agent_id}`;
          severity = 'success';
          paradigm = 'power';
          break;
        case 'rfc115_delegation':
          message = `🔐 RFC115 Enhanced delegation with attestation level: ${event.data.attestation_level}`;
          severity = 'success';
          paradigm = 'power';
          break;
        case 'business_accountability_event':
          message = `💼 Business accountability: ${event.data.business_actor} - ${event.data.action}`;
          severity = 'warning';
          paradigm = 'power';
          break;
        case 'legal_framework_validation':
          message = `⚖️ Legal framework validated for jurisdiction: ${event.data.jurisdiction}`;
          severity = 'success';
          paradigm = 'power';
          break;
        case 'compliance_assessment':
          message = `📋 Compliance assessment: ${event.data.compliance_level}`;
          severity = 'info';
          paradigm = 'power';
          break;
        case 'welcome':
          message = `🚀 GAuth Power-of-Attorney Protocol (P*P) Active`;
          severity = 'success';
          paradigm = 'power';
          break;
        default:
          message = `P*P Event: ${event.type}`;
          severity = 'info';
          paradigm = 'power';
      }
      
      if (message) {
        const notification = {
          id: Date.now(),
          message,
          severity,
          paradigm,
        };
        
        setNotifications(prev => [...prev, notification]);
        
        // Auto-remove notification after 5 seconds
        setTimeout(() => {
          setNotifications(prev => prev.filter(n => n.id !== notification.id));
        }, 5000);
      }
    });

    return () => {
      wsService.disconnect();
    };
  }, []);

  const handleDrawerToggle = () => {
    setDrawerOpen(!drawerOpen);
  };

  const handleNavigation = (path: string) => {
    navigate(path);
    setDrawerOpen(false);
  };

  const handleNotificationClose = (notificationId: number) => {
    setNotifications(prev => prev.filter(n => n.id !== notificationId));
  };

  const drawer = (
    <Box>
      <Toolbar>
        <Typography variant="h6" noWrap component="div" sx={{ fontWeight: 'bold' }}>
          🚀 GAuth P*P
        </Typography>
      </Toolbar>
      <Divider />
      <Alert severity="warning" sx={{ m: 1, fontSize: '0.75rem' }}>
        <Typography variant="caption" sx={{ fontWeight: 'bold' }}>
          P*P = POWER-OF-ATTORNEY
        </Typography>
        <br />
        <Typography variant="caption">
          Not policies! Business power delegation.
        </Typography>
      </Alert>
      <List>
        {navigationItems.map((item) => (
          <ListItem key={item.text} disablePadding>
            <ListItemButton onClick={() => handleNavigation(item.path)}>
              <ListItemIcon 
                sx={{ 
                  color: item.paradigm === 'power' ? 'success.main' : 
                         item.paradigm === 'comparison' ? 'warning.main' : 'text.secondary' 
                }}
              >
                {item.icon}
              </ListItemIcon>
              <Box sx={{ flexGrow: 1 }}>
                <ListItemText 
                  primary={item.text}
                  secondary={item.description}
                  primaryTypographyProps={{ fontSize: '0.875rem' }}
                  secondaryTypographyProps={{ fontSize: '0.75rem' }}
                />
                <Chip 
                  label={item.paradigm === 'power' ? 'P*P' : item.paradigm === 'comparison' ? 'Compare' : 'Info'}
                  size="small"
                  variant="outlined"
                  color={item.paradigm === 'power' ? 'success' : item.paradigm === 'comparison' ? 'warning' : 'default'}
                  sx={{ mt: 0.5, fontSize: '0.6rem', height: '16px' }}
                />
              </Box>
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex' }}>
      <AppBar
        position="fixed"
        sx={{
          width: { sm: `calc(100% - ${DRAWER_WIDTH}px)` },
          ml: { sm: `${DRAWER_WIDTH}px` },
        }}
      >
        <Toolbar>
          <Typography variant="h6" noWrap component="div" sx={{ flexGrow: 1 }}>
            🚀 GAuth Power-of-Attorney Protocol (P*P) - Enterprise Demo
          </Typography>
          <Chip 
            label="POWER vs POLICY"
            color="warning"
            variant="outlined"
            size="small"
            sx={{ mr: 1 }}
          />
          <Chip 
            label="Business Accountability"
            color="success"
            variant="outlined"
            size="small"
          />
        </Toolbar>
      </AppBar>

      <Box
        component="nav"
        sx={{ width: { sm: DRAWER_WIDTH }, flexShrink: { sm: 0 } }}
      >
        <Drawer
          variant="temporary"
          open={drawerOpen}
          onClose={handleDrawerToggle}
          ModalProps={{
            keepMounted: true,
          }}
          sx={{
            display: { xs: 'block', sm: 'none' },
            '& .MuiDrawer-paper': { boxSizing: 'border-box', width: DRAWER_WIDTH },
          }}
        >
          {drawer}
        </Drawer>
        <Drawer
          variant="permanent"
          sx={{
            display: { xs: 'none', sm: 'block' },
            '& .MuiDrawer-paper': { boxSizing: 'border-box', width: DRAWER_WIDTH },
          }}
          open
        >
          {drawer}
        </Drawer>
      </Box>

      <Box
        component="main"
        sx={{
          flexGrow: 1,
          p: 3,
          width: { sm: `calc(100% - ${DRAWER_WIDTH}px)` },
        }}
      >
        <Toolbar />
        <Container maxWidth="xl">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/paradigm" element={<ParadigmComparison />} />
            <Route path="/power-delegation" element={<PowerDelegation />} />
            <Route path="/rfc111" element={<RFC111Demo />} />
            <Route path="/rfc115" element={<RFC115Demo />} />
            <Route path="/business-owners" element={<BusinessOwners />} />
            <Route path="/compliance" element={<ComplianceMonitor />} />
            <Route path="/audit" element={<AuditTrail />} />
          </Routes>
        </Container>
      </Box>

      {/* Real-time notifications */}
      {notifications.map((notification) => (
        <Snackbar
          key={notification.id}
          open={true}
          autoHideDuration={5000}
          onClose={() => handleNotificationClose(notification.id)}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        >
          <Alert
            onClose={() => handleNotificationClose(notification.id)}
            severity={notification.severity}
            sx={{ width: '100%' }}
          >
            {notification.message}
          </Alert>
        </Snackbar>
      ))}
    </Box>
  );
};

export default App;