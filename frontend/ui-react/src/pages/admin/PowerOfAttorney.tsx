import { useState, useEffect } from 'react';
import {
  makeStyles,
  tokens,
  Card,
  Text,
  Title3,
  Button,
  Input,
  TabList,
  Tab,
  Badge,
  Field,
  Dropdown,
  Option,
  Textarea,
  DataGrid,
  DataGridBody,
  DataGridRow,
  DataGridHeader,
  DataGridHeaderCell,
  DataGridCell,
  TableCellLayout,
  TableColumnDefinition,
  createTableColumn,
  Dialog,
  DialogTrigger,
  DialogSurface,
  DialogTitle,
  DialogBody,
  DialogActions,
  DialogContent,
  Checkbox,

  ProgressBar,
} from '@fluentui/react-components';
import {
  PersonAccounts24Regular,
  Add24Regular,
  Eye24Regular,
  Edit24Regular,

  CheckmarkCircle24Regular,
  DismissCircle24Regular,
  Clock24Regular,

  Shield24Regular,
} from '@fluentui/react-icons';

// Import admin API hooks and types
import { usePowerOfAttorneyList, usePoAMutations } from '../../hooks/useAdminApi';
import type { PowerOfAttorney } from '../../types/admin';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '24px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  headerLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  cardsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
    gap: '16px',
  },
  card: {
    padding: '20px',
  },
  metricValue: {
    fontSize: '32px',
    fontWeight: 600,
    marginBottom: '8px',
  },
  metricLabel: {
    fontSize: '14px',
    color: tokens.colorNeutralForeground3,
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  twoColumn: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '16px',
  },
  wizardContainer: {
    marginTop: '24px',
  },
  stepIndicator: {
    display: 'flex',
    justifyContent: 'space-between',
    marginBottom: '32px',
    position: 'relative',
  },
  stepItem: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    flex: 1,
    position: 'relative',
    zIndex: 1,
  },
  stepCircle: {
    width: '40px',
    height: '40px',
    borderRadius: '50%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: '8px',
    fontWeight: 600,
    border: `2px solid ${tokens.colorNeutralStroke1}`,
    backgroundColor: tokens.colorNeutralBackground1,
  },
  stepCircleActive: {
    backgroundColor: tokens.colorBrandBackground,
    borderTopColor: tokens.colorBrandBackground,
    borderRightColor: tokens.colorBrandBackground,
    borderBottomColor: tokens.colorBrandBackground,
    borderLeftColor: tokens.colorBrandBackground,
    color: tokens.colorNeutralForegroundOnBrand,
  },
  stepCircleCompleted: {
    backgroundColor: tokens.colorPaletteGreenBackground2,
    borderTopColor: tokens.colorPaletteGreenBorder2,
    borderRightColor: tokens.colorPaletteGreenBorder2,
    borderBottomColor: tokens.colorPaletteGreenBorder2,
    borderLeftColor: tokens.colorPaletteGreenBorder2,
    color: tokens.colorNeutralForegroundOnBrand,
  },
  stepLabel: {
    fontSize: '12px',
    textAlign: 'center',
    color: tokens.colorNeutralForeground3,
  },
  stepLabelActive: {
    color: tokens.colorBrandForeground1,
    fontWeight: 600,
  },
  actionScopeGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(3, 1fr)',
    gap: '12px',
    marginTop: '12px',
  },
  checkboxCard: {
    padding: '12px',
    border: `1px solid ${tokens.colorNeutralStroke1}`,
    borderRadius: '4px',
    cursor: 'pointer',
    transition: 'all 0.2s',
    '&:hover': {
      backgroundColor: tokens.colorNeutralBackground1Hover,
    },
  },
  geoRestrictionMap: {
    padding: '16px',
    backgroundColor: tokens.colorNeutralBackground3,
    borderRadius: '4px',
    minHeight: '200px',
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  approvalFlow: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  approvalStep: {
    padding: '16px',
    border: `1px solid ${tokens.colorNeutralStroke1}`,
    borderRadius: '4px',
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  reviewSection: {
    padding: '16px',
    backgroundColor: tokens.colorNeutralBackground3,
    borderRadius: '4px',
    marginBottom: '16px',
  },
  reviewItem: {
    display: 'flex',
    justifyContent: 'space-between',
    padding: '8px 0',
    borderBottom: `1px solid ${tokens.colorNeutralStroke2}`,
  },
});

interface PoA {
  id: string;
  principalId: string;
  principalName: string;
  representativeId: string;
  representativeName: string;
  representativeType: string;
  status: 'active' | 'pending' | 'expired' | 'revoked';
  validFrom: string;
  validUntil: string;
  actions: string[];
  resources: string[];
  geoRestrictions: string[];
  approvalStatus: string;
  createdAt: string;
}

interface PoAFormData {
  principalId: string;
  principalName: string;
  representativeId: string;
  representativeName: string;
  representativeType: string;
  validFrom: string;
  validUntil: string;
  selectedActions: string[];
  selectedResources: string[];
  geoRestrictions: string[];
  requiresApproval: boolean;
  notificationEmail: string;
  reason: string;
}

export default function PowerOfAttorney() {
  const classes = useStyles();
  const [selectedTab, setSelectedTab] = useState<string>('overview');
  const [poaList, setPoaList] = useState<PoA[]>([]);
  const [loading, setLoading] = useState(false);
  const [builderDialogOpen, setBuilderDialogOpen] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);

  // Form state
  const [formData, setFormData] = useState<PoAFormData>({
    principalId: '',
    principalName: '',
    representativeId: '',
    representativeName: '',
    representativeType: 'individual',
    validFrom: '',
    validUntil: '',
    selectedActions: [],
    selectedResources: [],
    geoRestrictions: [],
    requiresApproval: false,
    notificationEmail: '',
    reason: '',
  });

  const steps = [
    { id: 0, label: 'Principal & Representative', icon: '1' },
    { id: 1, label: 'Action Scope', icon: '2' },
    { id: 2, label: 'Resources', icon: '3' },
    { id: 3, label: 'Geographic Restrictions', icon: '4' },
    { id: 4, label: 'Time Validity', icon: '5' },
    { id: 5, label: 'Approval Workflow', icon: '6' },
    { id: 6, label: 'Review & Submit', icon: '7' },
  ];

  const availableActions = [
    'read', 'write', 'delete', 'create', 'update',
    'approve', 'reject', 'revoke', 'delegate', 'audit',
    'configure', 'execute'
  ];

  const availableResources = [
    'tokens', 'policies', 'users', 'documents', 'configurations',
    'reports', 'audit_logs', 'api_keys', 'webhooks', 'notifications'
  ];

  const availableRegions = [
    'North America', 'South America', 'Europe', 'Asia Pacific',
    'Middle East', 'Africa', 'Global'
  ];

  // Use the new hooks
  const { data: poaData, refetch } = usePowerOfAttorneyList();
  const { createPoA, deletePoA } = usePoAMutations();

  // Update local state when data changes
  useEffect(() => {
    if (poaData?.powerOfAttorneys) {
      setPoaList(poaData.powerOfAttorneys);
    }
  }, [poaData]);

  const handleCreatePoA = async () => {
    setLoading(true);
    try {
      await createPoA(formData);
      refetch();
      setBuilderDialogOpen(false);
      resetForm();
    } catch (error) {
      console.error('Failed to create PoA:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleRevokePoA = async (poaId: string) => {
    try {
      await deletePoA(poaId);
      refetch();
    } catch (error) {
      console.error('Failed to revoke PoA:', error);
    }
  };

  const handleViewPoA = (poaId: string) => {
    const poa = poaList.find(p => p.id === poaId);
    if (poa) {
      alert(`PoA Details:\n\nID: ${poa.id}\nPrincipal: ${poa.principalName}\nRepresentative: ${poa.representativeName}\nStatus: ${poa.status}\nActions: ${poa.actions.join(', ')}\nResources: ${poa.resources.join(', ')}`);
      // TODO: Replace with proper detail view modal
    }
  };

  const handleEditPoA = (poaId: string) => {
    const poa = poaList.find(p => p.id === poaId);
    if (poa) {
      // Populate form with existing PoA data
      setFormData({
        principalId: poa.principalId,
        principalName: poa.principalName,
        representativeId: poa.representativeId,
        representativeName: poa.representativeName,
        representativeType: poa.representativeType,
        validFrom: poa.validFrom,
        validUntil: poa.validUntil,
        selectedActions: poa.actions,
        selectedResources: poa.resources,
        geoRestrictions: poa.geoRestrictions,
        requiresApproval: poa.approvalStatus === 'pending',
        notificationEmail: '',
        reason: '',
      });
      setBuilderDialogOpen(true);
    }
  };

  const resetForm = () => {
    setFormData({
      principalId: '',
      principalName: '',
      representativeId: '',
      representativeName: '',
      representativeType: 'individual',
      validFrom: '',
      validUntil: '',
      selectedActions: [],
      selectedResources: [],
      geoRestrictions: [],
      requiresApproval: false,
      notificationEmail: '',
      reason: '',
    });
    setCurrentStep(0);
  };

  const handleActionToggle = (action: string) => {
    setFormData(prev => ({
      ...prev,
      selectedActions: prev.selectedActions.includes(action)
        ? prev.selectedActions.filter(a => a !== action)
        : [...prev.selectedActions, action]
    }));
  };

  const handleResourceToggle = (resource: string) => {
    setFormData(prev => ({
      ...prev,
      selectedResources: prev.selectedResources.includes(resource)
        ? prev.selectedResources.filter(r => r !== resource)
        : [...prev.selectedResources, resource]
    }));
  };

  const handleGeoToggle = (region: string) => {
    setFormData(prev => ({
      ...prev,
      geoRestrictions: prev.geoRestrictions.includes(region)
        ? prev.geoRestrictions.filter(g => g !== region)
        : [...prev.geoRestrictions, region]
    }));
  };

  const canProceed = () => {
    switch (currentStep) {
      case 0:
        return formData.principalId && formData.representativeId;
      case 1:
        return formData.selectedActions.length > 0;
      case 2:
        return formData.selectedResources.length > 0;
      case 3:
        return formData.geoRestrictions.length > 0;
      case 4:
        return formData.validFrom && formData.validUntil;
      case 5:
        return true;
      case 6:
        return true;
      default:
        return false;
    }
  };

  const renderStep = () => {
    switch (currentStep) {
      case 0:
        return (
          <div className={classes.form}>
            <Text weight="semibold" size={400}>Principal Information</Text>
            <Field label="Principal ID" required>
              <Input
                value={formData.principalId}
                onChange={(e) => setFormData({ ...formData, principalId: e.target.value })}
                placeholder="user@example.com"
              />
            </Field>
            <Field label="Principal Name">
              <Input
                value={formData.principalName}
                onChange={(e) => setFormData({ ...formData, principalName: e.target.value })}
                placeholder="John Doe"
              />
            </Field>

            <Text weight="semibold" size={400} style={{ marginTop: '24px' }}>Representative Information</Text>
            <Field label="Representative Type">
              <Dropdown
                value={formData.representativeType}
                onOptionSelect={(_, data) => setFormData({ ...formData, representativeType: data.optionValue as string })}
              >
                <Option value="individual">Individual</Option>
                <Option value="organization">Organization</Option>
                <Option value="service_account">Service Account</Option>
                <Option value="automated_system">Automated System</Option>
              </Dropdown>
            </Field>
            <Field label="Representative ID" required>
              <Input
                value={formData.representativeId}
                onChange={(e) => setFormData({ ...formData, representativeId: e.target.value })}
                placeholder="representative@example.com"
              />
            </Field>
            <Field label="Representative Name">
              <Input
                value={formData.representativeName}
                onChange={(e) => setFormData({ ...formData, representativeName: e.target.value })}
                placeholder="Jane Smith"
              />
            </Field>
          </div>
        );

      case 1:
        return (
          <div className={classes.form}>
            <Text weight="semibold" size={400}>Select Actions</Text>
            <Text size={300}>Choose which actions the representative can perform</Text>
            <div className={classes.actionScopeGrid}>
              {availableActions.map((action) => (
                <div key={action} className={classes.checkboxCard}>
                  <Checkbox
                    checked={formData.selectedActions.includes(action)}
                    onChange={() => handleActionToggle(action)}
                    label={action.charAt(0).toUpperCase() + action.slice(1)}
                  />
                </div>
              ))}
            </div>
            <Text size={200} style={{ marginTop: '8px', color: tokens.colorNeutralForeground3 }}>
              Selected: {formData.selectedActions.length} action(s)
            </Text>
          </div>
        );

      case 2:
        return (
          <div className={classes.form}>
            <Text weight="semibold" size={400}>Select Resources</Text>
            <Text size={300}>Choose which resources the representative can access</Text>
            <div className={classes.actionScopeGrid}>
              {availableResources.map((resource) => (
                <div key={resource} className={classes.checkboxCard}>
                  <Checkbox
                    checked={formData.selectedResources.includes(resource)}
                    onChange={() => handleResourceToggle(resource)}
                    label={resource.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')}
                  />
                </div>
              ))}
            </div>
            <Text size={200} style={{ marginTop: '8px', color: tokens.colorNeutralForeground3 }}>
              Selected: {formData.selectedResources.length} resource(s)
            </Text>
          </div>
        );

      case 3:
        return (
          <div className={classes.form}>
            <Text weight="semibold" size={400}>Geographic Restrictions</Text>
            <Text size={300}>Select regions where the PoA is valid</Text>
            <div className={classes.geoRestrictionMap}>
              {availableRegions.map((region) => (
                <Checkbox
                  key={region}
                  checked={formData.geoRestrictions.includes(region)}
                  onChange={() => handleGeoToggle(region)}
                  label={region}
                />
              ))}
            </div>
            <Text size={200} style={{ marginTop: '8px', color: tokens.colorNeutralForeground3 }}>
              Selected: {formData.geoRestrictions.length} region(s)
            </Text>
          </div>
        );

      case 4:
        return (
          <div className={classes.form}>
            <Text weight="semibold" size={400}>Time Validity</Text>
            <Text size={300}>Set the validity period for this Proof of Authorization</Text>
            <div className={classes.twoColumn}>
              <Field label="Valid From" required>
                <Input
                  type="datetime-local"
                  value={formData.validFrom}
                  onChange={(e) => setFormData({ ...formData, validFrom: e.target.value })}
                />
              </Field>
              <Field label="Valid Until" required>
                <Input
                  type="datetime-local"
                  value={formData.validUntil}
                  onChange={(e) => setFormData({ ...formData, validUntil: e.target.value })}
                />
              </Field>
            </div>
          </div>
        );

      case 5:
        return (
          <div className={classes.form}>
            <Text weight="semibold" size={400}>Approval Workflow</Text>
            <Text size={300}>Configure approval requirements for this PoA</Text>

            <Checkbox
              checked={formData.requiresApproval}
              onChange={(_, data) => setFormData({ ...formData, requiresApproval: data.checked as boolean })}
              label="Require approval before activation"
            />

            {formData.requiresApproval && (
              <div className={classes.approvalFlow}>
                <div className={classes.approvalStep}>
                  <Shield24Regular style={{ fontSize: '24px', color: tokens.colorBrandForeground1 }} />
                  <div>
                    <Text weight="semibold">Step 1: Manager Approval</Text>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                      Principal's manager must approve
                    </Text>
                  </div>
                </div>
                <div className={classes.approvalStep}>
                  <Shield24Regular style={{ fontSize: '24px', color: tokens.colorBrandForeground1 }} />
                  <div>
                    <Text weight="semibold">Step 2: Security Review</Text>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                      Security team reviews permissions
                    </Text>
                  </div>
                </div>
                <div className={classes.approvalStep}>
                  <Shield24Regular style={{ fontSize: '24px', color: tokens.colorBrandForeground1 }} />
                  <div>
                    <Text weight="semibold">Step 3: Compliance Check</Text>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                      Automated compliance validation
                    </Text>
                  </div>
                </div>
              </div>
            )}

            <Field label="Notification Email">
              <Input
                type="email"
                value={formData.notificationEmail}
                onChange={(e) => setFormData({ ...formData, notificationEmail: e.target.value })}
                placeholder="notifications@example.com"
              />
            </Field>

            <Field label="Reason for PoA">
              <Textarea
                value={formData.reason}
                onChange={(e) => setFormData({ ...formData, reason: e.target.value })}
                placeholder="Explain why this Proof of Authorization is needed..."
                rows={4}
              />
            </Field>
          </div>
        );

      case 6:
        return (
          <div className={classes.form}>
            <Text weight="semibold" size={500}>Review Proof of Authorization</Text>
            <Text size={300}>Please review all details before submission</Text>

            <div className={classes.reviewSection}>
              <Text weight="semibold" size={400} style={{ marginBottom: '12px' }}>Principal & Representative</Text>
              <div className={classes.reviewItem}>
                <Text>Principal</Text>
                <Text weight="semibold">{formData.principalId}</Text>
              </div>
              <div className={classes.reviewItem}>
                <Text>Representative</Text>
                <Text weight="semibold">{formData.representativeId}</Text>
              </div>
              <div className={classes.reviewItem}>
                <Text>Type</Text>
                <Badge>{formData.representativeType}</Badge>
              </div>
            </div>

            <div className={classes.reviewSection}>
              <Text weight="semibold" size={400} style={{ marginBottom: '12px' }}>Permissions</Text>
              <div className={classes.reviewItem}>
                <Text>Actions</Text>
                <Text weight="semibold">{formData.selectedActions.length} selected</Text>
              </div>
              <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                {formData.selectedActions.join(', ')}
              </Text>
              <div className={classes.reviewItem} style={{ marginTop: '8px' }}>
                <Text>Resources</Text>
                <Text weight="semibold">{formData.selectedResources.length} selected</Text>
              </div>
              <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                {formData.selectedResources.join(', ')}
              </Text>
            </div>

            <div className={classes.reviewSection}>
              <Text weight="semibold" size={400} style={{ marginBottom: '12px' }}>Restrictions</Text>
              <div className={classes.reviewItem}>
                <Text>Geographic</Text>
                <Text weight="semibold">{formData.geoRestrictions.join(', ')}</Text>
              </div>
              <div className={classes.reviewItem}>
                <Text>Valid From</Text>
                <Text weight="semibold">{formData.validFrom}</Text>
              </div>
              <div className={classes.reviewItem}>
                <Text>Valid Until</Text>
                <Text weight="semibold">{formData.validUntil}</Text>
              </div>
            </div>

            <div className={classes.reviewSection}>
              <Text weight="semibold" size={400} style={{ marginBottom: '12px' }}>Approval</Text>
              <div className={classes.reviewItem}>
                <Text>Requires Approval</Text>
                <Badge color={formData.requiresApproval ? 'warning' : 'success'}>
                  {formData.requiresApproval ? 'Yes' : 'No'}
                </Badge>
              </div>
              {formData.notificationEmail && (
                <div className={classes.reviewItem}>
                  <Text>Notification Email</Text>
                  <Text weight="semibold">{formData.notificationEmail}</Text>
                </div>
              )}
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return <Badge appearance="filled" color="success">Active</Badge>;
      case 'pending':
        return <Badge appearance="filled" color="warning">Pending</Badge>;
      case 'expired':
        return <Badge appearance="filled" color="danger">Expired</Badge>;
      case 'revoked':
        return <Badge appearance="filled" color="danger">Revoked</Badge>;
      default:
        return <Badge>{status}</Badge>;
    }
  };

  const columns: TableColumnDefinition<PoA>[] = [
    createTableColumn<PoA>({
      columnId: 'principal',
      renderHeaderCell: () => 'Principal',
      renderCell: (item) => (
        <TableCellLayout>
          <Text weight="semibold">{item.principalName}</Text>
          <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
            {item.principalId}
          </Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<PoA>({
      columnId: 'representative',
      renderHeaderCell: () => 'Representative',
      renderCell: (item) => (
        <TableCellLayout>
          <Text weight="semibold">{item.representativeName}</Text>
          <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
            {item.representativeType}
          </Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<PoA>({
      columnId: 'status',
      renderHeaderCell: () => 'Status',
      renderCell: (item) => (
        <TableCellLayout>
          {getStatusBadge(item.status)}
        </TableCellLayout>
      ),
    }),
    createTableColumn<PoA>({
      columnId: 'validity',
      renderHeaderCell: () => 'Validity Period',
      renderCell: (item) => (
        <TableCellLayout>
          <Text size={200}>From: {new Date(item.validFrom).toLocaleDateString()}</Text>
          <Text size={200}>Until: {new Date(item.validUntil).toLocaleDateString()}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<PoA>({
      columnId: 'permissions',
      renderHeaderCell: () => 'Permissions',
      renderCell: (item) => (
        <TableCellLayout>
          <Text size={200}>{item.actions.length} actions</Text>
          <Text size={200}>{item.resources.length} resources</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<PoA>({
      columnId: 'actions',
      renderHeaderCell: () => 'Actions',
      renderCell: (item) => (
        <TableCellLayout>
          <div style={{ display: 'flex', gap: '8px' }}>
            <Button
              size="small"
              icon={<Eye24Regular />}
              onClick={() => handleViewPoA(item.id)}
              title="View Details"
            />
            {(item.status === 'active' || item.status === 'pending') && (
              <Button
                size="small"
                icon={<Edit24Regular />}
                onClick={() => handleEditPoA(item.id)}
                title="Edit"
              />
            )}
            {item.status === 'active' && (
              <Button
                size="small"
                appearance="subtle"
                icon={<DismissCircle24Regular />}
                onClick={() => handleRevokePoA(item.id)}
                title="Revoke"
              />
            )}
          </div>
        </TableCellLayout>
      ),
    }),
  ];

  const activePoAs = poaList.filter(p => p.status === 'active').length;
  const pendingPoAs = poaList.filter(p => p.status === 'pending').length;
  const expiredPoAs = poaList.filter(p => p.status === 'expired').length;

  return (
    <div className={classes.container}>
      <div className={classes.header}>
        <div className={classes.headerLeft}>
          <PersonAccounts24Regular style={{ fontSize: '24px' }} />
          <Title3>Proof of Authorization Management</Title3>
        </div>
        <Dialog open={builderDialogOpen} onOpenChange={(_, data) => {
          setBuilderDialogOpen(data.open);
          if (!data.open) resetForm();
        }}>
          <DialogTrigger>
            <Button appearance="primary" icon={<Add24Regular />}>
              Create Proof of Authorization
            </Button>
          </DialogTrigger>
          <DialogSurface style={{ maxWidth: '900px', maxHeight: '90vh' }}>
            <DialogBody>
              <DialogTitle>Create New Proof of Authorization</DialogTitle>
              <DialogContent>
                <div className={classes.wizardContainer}>
                  {/* Step Indicator */}
                  <div className={classes.stepIndicator}>
                    {steps.map((step, index) => (
                      <div key={step.id} className={classes.stepItem}>
                        <div
                          className={`${classes.stepCircle} ${index < currentStep ? classes.stepCircleCompleted :
                            index === currentStep ? classes.stepCircleActive : ''
                            }`}
                        >
                          {index < currentStep ? <CheckmarkCircle24Regular /> : step.icon}
                        </div>
                        <Text
                          className={`${classes.stepLabel} ${index === currentStep ? classes.stepLabelActive : ''
                            }`}
                        >
                          {step.label}
                        </Text>
                      </div>
                    ))}
                  </div>

                  <ProgressBar value={(currentStep + 1) / steps.length} max={1} />

                  {/* Step Content */}
                  <div style={{ marginTop: '24px' }}>
                    {renderStep()}
                  </div>
                </div>
              </DialogContent>
              <DialogActions>
                <Button
                  appearance="secondary"
                  onClick={() => {
                    if (currentStep > 0) {
                      setCurrentStep(currentStep - 1);
                    } else {
                      setBuilderDialogOpen(false);
                      resetForm();
                    }
                  }}
                >
                  {currentStep > 0 ? 'Previous' : 'Cancel'}
                </Button>
                {currentStep < steps.length - 1 ? (
                  <Button
                    appearance="primary"
                    onClick={() => setCurrentStep(currentStep + 1)}
                    disabled={!canProceed()}
                  >
                    Next
                  </Button>
                ) : (
                  <Button
                    appearance="primary"
                    onClick={handleCreatePoA}
                    disabled={loading || !canProceed()}
                  >
                    {loading ? 'Creating...' : 'Create PoA'}
                  </Button>
                )}
              </DialogActions>
            </DialogBody>
          </DialogSurface>
        </Dialog>
      </div>

      {/* Overview Cards */}
      <div className={classes.cardsGrid}>
        <Card className={classes.card}>
          <Text className={classes.metricValue} style={{ color: tokens.colorPaletteGreenForeground1 }}>
            {activePoAs}
          </Text>
          <Text className={classes.metricLabel}>Active PoAs</Text>
        </Card>

        <Card className={classes.card}>
          <Text className={classes.metricValue} style={{ color: tokens.colorPaletteYellowForeground1 }}>
            {pendingPoAs}
          </Text>
          <Text className={classes.metricLabel}>Pending Approval</Text>
        </Card>

        <Card className={classes.card}>
          <Text className={classes.metricValue} style={{ color: tokens.colorPaletteRedForeground1 }}>
            {expiredPoAs}
          </Text>
          <Text className={classes.metricLabel}>Expired</Text>
        </Card>

        <Card className={classes.card}>
          <Text className={classes.metricValue}>
            {poaList.length}
          </Text>
          <Text className={classes.metricLabel}>Total PoAs</Text>
        </Card>
      </div>

      {/* Main Content */}
      <Card className={classes.card}>
        <TabList selectedValue={selectedTab} onTabSelect={(_, data) => setSelectedTab(data.value as string)}>
          <Tab value="overview" icon={<PersonAccounts24Regular />}>All PoAs</Tab>
          <Tab value="active" icon={<CheckmarkCircle24Regular />}>Active</Tab>
          <Tab value="pending" icon={<Clock24Regular />}>Pending</Tab>
        </TabList>

        <div style={{ marginTop: '24px' }}>
          <DataGrid
            items={selectedTab === 'overview' ? poaList :
              selectedTab === 'active' ? poaList.filter(p => p.status === 'active') :
                poaList.filter(p => p.status === 'pending')}
            columns={columns}
            sortable
            resizableColumns
          >
            <DataGridHeader>
              <DataGridRow>
                {({ renderHeaderCell }) => (
                  <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>
                )}
              </DataGridRow>
            </DataGridHeader>
            <DataGridBody<PoA>>
              {({ item, rowId }) => (
                <DataGridRow<PoA> key={rowId}>
                  {({ renderCell }) => (
                    <DataGridCell>{renderCell(item)}</DataGridCell>
                  )}
                </DataGridRow>
              )}
            </DataGridBody>
          </DataGrid>
        </div>
      </Card>
    </div>
  );
}
