import { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  FluentProvider,
  webLightTheme,
  makeStyles,
  tokens,
  Button,
  Avatar,
  Menu,
  MenuTrigger,
  MenuPopover,
  MenuList,
  MenuItem,
  MenuDivider,
  Text,
  Tooltip,
} from '@fluentui/react-components';
import {

  Home24Regular,
  DataArea24Regular,
  PeopleTeam24Regular,
  ShieldTask24Regular,
  DocumentBulletList24Regular,
  Settings24Regular,
  ChevronRight20Regular,
  ChevronLeft20Regular,
  SignOut24Regular,
  PersonAvailable24Regular,
  Gauge24Regular,
  Key24Regular,
  Alert24Regular,
  Database24Regular,
  DocumentCheckmark24Regular,
  Shield24Regular,
  Bot24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
  root: {
    display: 'flex',
    height: '100vh',
    backgroundColor: tokens.colorNeutralBackground3,
  },
  sidebar: {
    display: 'flex',
    flexDirection: 'column',
    backgroundColor: tokens.colorNeutralBackground1,
    borderRight: `1px solid ${tokens.colorNeutralStroke2}`,
    transition: 'width 0.3s ease',
    overflow: 'hidden',
  },
  sidebarExpanded: {
    width: '280px',
  },
  sidebarCollapsed: {
    width: '68px',
  },
  sidebarHeader: {
    padding: '20px 16px',
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    borderBottom: `1px solid ${tokens.colorNeutralStroke2}`,
    minHeight: '72px',
  },
  logo: {
    fontSize: '28px',
    flexShrink: 0,
  },
  sidebarTitle: {
    fontSize: '16px',
    fontWeight: 600,
    whiteSpace: 'nowrap',
  },
  nav: {
    display: 'flex',
    flexDirection: 'column',
    padding: '8px',
    flex: 1,
    overflowY: 'auto',
  },
  navItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    padding: '12px 16px',
    borderRadius: '6px',
    cursor: 'pointer',
    transition: 'background-color 0.2s',
    marginBottom: '4px',
    border: 'none',
    backgroundColor: 'transparent',
    color: tokens.colorNeutralForeground1,
    textAlign: 'left',
    width: '100%',
    fontSize: '14px',
    fontWeight: 500,
    ':hover': {
      backgroundColor: tokens.colorNeutralBackground1Hover,
    },
  },
  navItemActive: {
    backgroundColor: tokens.colorBrandBackground,
    color: tokens.colorNeutralForegroundOnBrand,
    ':hover': {
      backgroundColor: tokens.colorBrandBackgroundHover,
    },
  },
  navIcon: {
    fontSize: '20px',
    flexShrink: 0,
  },
  navText: {
    whiteSpace: 'nowrap',
  },
  toggleButton: {
    margin: '8px',
    justifyContent: 'center',
  },
  main: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden',
  },
  topBar: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '12px 24px',
    backgroundColor: tokens.colorNeutralBackground1,
    borderBottom: `1px solid ${tokens.colorNeutralStroke2}`,
    minHeight: '64px',
  },
  topBarLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: '16px',
  },
  breadcrumb: {
    fontSize: '20px',
    fontWeight: 600,
    color: tokens.colorNeutralForeground1,
  },
  topBarRight: {
    display: 'flex',
    alignItems: 'center',
    gap: '16px',
  },
  userInfo: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  content: {
    flex: 1,
    overflow: 'auto',
    padding: '24px',
  },
});

interface NavItem {
  id: string;
  label: string;
  icon: JSX.Element;
  path: string;
}

const navItems: NavItem[] = [
  {
    id: 'dashboard',
    label: 'Dashboard',
    icon: <Home24Regular />,
    path: '/admin/dashboard',
  },
  {
    id: 'metrics',
    label: 'System Metrics',
    icon: <DataArea24Regular />,
    path: '/admin/metrics',
  },
  {
    id: 'subscribers',
    label: 'Subscribers',
    icon: <PeopleTeam24Regular />,
    path: '/admin/subscribers/list',
  },
  {
    id: 'tokens',
    label: 'Token Management',
    icon: <Key24Regular />,
    path: '/admin/tokens',
  },
  {
    id: 'authorization',
    label: 'Authorization Engine',
    icon: <ShieldTask24Regular />,
    path: '/admin/authorization',
  },
  {
    id: 'poa',
    label: 'Power of Attorney',
    icon: <DocumentCheckmark24Regular />,
    path: '/admin/poa',
  },
  {
    id: 'events',
    label: 'Event System',
    icon: <Alert24Regular />,
    path: '/admin/events',
  },
  {
    id: 'resilience',
    label: 'Resilience Patterns',
    icon: <Shield24Regular />,
    path: '/admin/resilience',
  },
  {
    id: 'audit',
    label: 'Audit Trail',
    icon: <DocumentBulletList24Regular />,
    path: '/admin/audit',
  },
  {
    id: 'gauthplus',
    label: 'GAuth+',
    icon: <Bot24Regular />,
    path: '/admin/gauthplus',
  },
  {
    id: 'revocation',
    label: 'Revocation Transparency',
    icon: <Database24Regular />,
    path: '/admin/revocation',
  },
  {
    id: 'oidc-providers',
    label: 'OIDC Providers',
    icon: <Shield24Regular />,
    path: '/admin/oidc-providers',
  },
  {
    id: 'configuration',
    label: 'Configuration',
    icon: <Settings24Regular />,
    path: '/admin/configuration',
  },
  {
    id: 'performance',
    label: 'Performance',
    icon: <Gauge24Regular />,
    path: '/admin/performance',
  },
];

export default function AdminLayout() {
  const classes = useStyles();
  const navigate = useNavigate();
  const location = useLocation();
  const [sidebarExpanded, setSidebarExpanded] = useState(true);

  const userRole = localStorage.getItem('admin_role') || 'admin';
  const userName = localStorage.getItem('admin_username') || 'Admin User';

  const handleLogout = () => {
    localStorage.removeItem('admin_token');
    localStorage.removeItem('admin_role');
    localStorage.removeItem('admin_username');
    navigate('/admin/login');
  };

  const isActive = (path: string) => location.pathname === path;

  const getCurrentPageTitle = () => {
    const currentItem = navItems.find((item) => item.path === location.pathname);
    return currentItem?.label || 'Admin Portal';
  };

  return (
    <FluentProvider theme={webLightTheme}>
      <div className={classes.root}>
        {/* Sidebar */}
        <div
          className={`${classes.sidebar} ${sidebarExpanded ? classes.sidebarExpanded : classes.sidebarCollapsed
            }`}
        >
          <div className={classes.sidebarHeader}>
            <div className={classes.logo}>🛡️</div>
            {sidebarExpanded && (
              <Text className={classes.sidebarTitle}>GAuth Admin</Text>
            )}
          </div>

          <nav className={classes.nav}>
            {navItems.map((item) => {
              const navButton = (
                <button
                  className={`${classes.navItem} ${isActive(item.path) ? classes.navItemActive : ''
                    }`}
                  onClick={() => navigate(item.path)}
                >
                  <span className={classes.navIcon}>{item.icon}</span>
                  {sidebarExpanded && (
                    <span className={classes.navText}>{item.label}</span>
                  )}
                </button>
              );

              return sidebarExpanded ? (
                <div key={item.id}>{navButton}</div>
              ) : (
                <Tooltip
                  key={item.id}
                  content={item.label}
                  relationship="label"
                  positioning="after"
                  withArrow
                >
                  {navButton}
                </Tooltip>
              );
            })}
          </nav>

          <Button
            appearance="subtle"
            icon={
              sidebarExpanded ? <ChevronLeft20Regular /> : <ChevronRight20Regular />
            }
            onClick={() => setSidebarExpanded(!sidebarExpanded)}
            className={classes.toggleButton}
          />
        </div>

        {/* Main Content */}
        <div className={classes.main}>
          {/* Top Bar */}
          <div className={classes.topBar}>
            <div className={classes.topBarLeft}>
              <Text className={classes.breadcrumb}>{getCurrentPageTitle()}</Text>
            </div>
            <div className={classes.topBarRight}>
              <div className={classes.userInfo}>
                <PersonAvailable24Regular />
                <Text weight="semibold">{userName}</Text>
                <Text size={200}>({userRole})</Text>
              </div>
              <Menu>
                <MenuTrigger>
                  <Button
                    appearance="subtle"
                    icon={<Avatar name={userName} size={32} />}
                  />
                </MenuTrigger>
                <MenuPopover>
                  <MenuList>
                    <MenuItem icon={<PersonAvailable24Regular />}>Profile</MenuItem>
                    <MenuItem icon={<Settings24Regular />}>Settings</MenuItem>
                    <MenuDivider />
                    <MenuItem icon={<SignOut24Regular />} onClick={handleLogout}>
                      Sign Out
                    </MenuItem>
                  </MenuList>
                </MenuPopover>
              </Menu>
            </div>
          </div>

          {/* Page Content */}
          <div className={classes.content}>
            <Outlet />
          </div>
        </div>
      </div>
    </FluentProvider>
  );
}
